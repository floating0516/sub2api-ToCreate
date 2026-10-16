//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const portalTestID = "ba4a51ed-1d20-4ef5-878b-37c70e9ef45a"
const portalCreateJSON = `{"type":"technical","title":"充值未到账","description":"订单完成付款后一直没有到账","priority":"normal","product":"tocreate","client_request_id":"87b2580f-6a03-44d5-9e0c-c96741a20ce7"}`
const portalReplyJSON = `{"content":"还是没有到账","client_message_id":"87b2580f-6a03-44d5-9e0c-c96741a20ce7"}`

func TestSupportPortal_ExplicitForwardingAndEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, method, path, body string
		handler                  func(*SupportHandler) gin.HandlerFunc
	}{
		{"list", "GET", "/tickets?q=a%26b&type=order&status=pending_agent&page=2&page_size=20", "", func(h *SupportHandler) gin.HandlerFunc { return h.ListTickets }},
		{"stats", "GET", "/tickets/stats", "", func(h *SupportHandler) gin.HandlerFunc { return h.TicketStats }},
		{"create", "POST", "/tickets", portalCreateJSON, func(h *SupportHandler) gin.HandlerFunc { return h.CreateTicket }},
		{"detail", "GET", "/tickets/" + portalTestID, "", func(h *SupportHandler) gin.HandlerFunc { return h.GetTicket }},
		{"reply", "POST", "/tickets/" + portalTestID + "/replies", portalReplyJSON, func(h *SupportHandler) gin.HandlerFunc { return h.ReplyTicket }},
		{"resolve", "POST", "/tickets/" + portalTestID + "/resolve", "{}", func(h *SupportHandler) gin.HandlerFunc { return h.ResolveTicket }},
		{"close", "POST", "/tickets/" + portalTestID + "/close", "{}", func(h *SupportHandler) gin.HandlerFunc { return h.CloseTicket }},
		{"reopen", "POST", "/tickets/" + portalTestID + "/reopen", "{}", func(h *SupportHandler) gin.HandlerFunc { return h.ReopenTicket }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(chan string, 1)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims := &supportClaims{}
				_, err := jwt.ParseWithClaims(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), claims,
					func(_ *jwt.Token) (any, error) { return []byte("support-handler-test-secret-32bytes"), nil }, jwt.WithIssuer("support-agent"), jwt.WithAudience("support-agent"))
				if err != nil || claims.Subject != "123" || claims.Role != "user" || r.Header.Get("Cookie") != "" {
					w.WriteHeader(401)
					return
				}
				seen <- r.Method + " " + r.URL.RequestURI()
				_, _ = io.WriteString(w, `{"items":[],"all":12,"reused":true}`)
			}))
			defer upstream.Close()
			h := newSupportHandlerForTest(upstream.URL, true)
			path := "/api/v1/support" + tc.path
			if tc.name == "list" {
				path += "&user_id=999"
			}
			c, recorder := newSupportContext(tc.method, path, tc.body)
			c.Params = gin.Params{{Key: "id", Value: portalTestID}}
			c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
			tc.handler(h)(c)
			require.Equal(t, 200, recorder.Code, recorder.Body.String())
			request := <-seen
			if tc.name == "list" {
				require.Contains(t, request, "q=a%26b")
				require.NotContains(t, request, "user_id")
			} else {
				require.Equal(t, tc.method+" "+tc.path, request)
			}
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
			require.Equal(t, float64(0), envelope["code"])
			require.Equal(t, float64(12), envelope["data"].(map[string]any)["all"])
		})
	}
}

func TestSupportPortal_OrderOwnershipBeforeForwarding(t *testing.T) {
	var forwarded atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, exists := body["user_id"]; exists {
			w.WriteHeader(400)
			return
		}
		_, _ = io.WriteString(w, `{"id":"`+portalTestID+`"}`)
	}))
	defer upstream.Close()
	h := newSupportHandlerForTest(upstream.URL, true)
	h.BindOrderValidator(func(_ context.Context, userID, orderID int64) error {
		require.Equal(t, int64(123), userID)
		if orderID == 2 {
			return service.ErrSupportOrderForbidden
		}
		if orderID == 3 {
			return service.ErrManagedRechargeOrderMissing
		}
		return nil
	})
	for _, tc := range []struct {
		order  string
		status int
	}{{"1", 200}, {"2", 403}, {"3", 404}} {
		body := strings.Replace(portalCreateJSON, `"type":"technical"`, `"type":"order","order_id":"`+tc.order+`","user_id":"999"`, 1)
		c, recorder := newSupportContext("POST", "/api/v1/support/tickets", body)
		h.CreateTicket(c)
		require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
	}
	require.Equal(t, int32(1), forwarded.Load())
	missing, rec := newSupportContext("POST", "/api/v1/support/tickets", strings.Replace(portalCreateJSON, "technical", "order", 1))
	h.CreateTicket(missing)
	require.Equal(t, 400, rec.Code)
}

func TestSupportPortal_PreservesOwnershipAndClosedErrors(t *testing.T) {
	for _, status := range []int{403, 404, 409} {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"detail":"resolved or closed tickets cannot receive replies"}`)
		}))
		h := newSupportHandlerForTest(upstream.URL, true)
		for _, fn := range []gin.HandlerFunc{h.GetTicket, h.ReplyTicket, h.CloseTicket} {
			c, rec := newSupportContext("POST", "/api/v1/support/tickets/"+portalTestID+"/replies", portalReplyJSON)
			c.Params = gin.Params{{Key: "id", Value: portalTestID}}
			fn(c)
			require.Equal(t, status, rec.Code)
			require.Contains(t, rec.Body.String(), "cannot receive replies")
		}
		upstream.Close()
	}
}

func TestSupportPortal_AdminRoleAndUserRouteIsolation(t *testing.T) {
	roles := make(chan string, 2)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := jwt.MapClaims{}
		_, _ = jwt.ParseWithClaims(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), claims, func(_ *jwt.Token) (any, error) { return []byte("support-handler-test-secret-32bytes"), nil })
		roles <- claims["role"].(string)
		_, _ = io.WriteString(w, `{"items":[{"attachments":[{"url":"/api/v1/support/uploads/`+portalTestID+`"}]}]}`)
	}))
	defer upstream.Close()
	h := newSupportHandlerForTest(upstream.URL, true)
	c, rec := newSupportContext("GET", "/api/v1/admin/support/tickets", "")
	h.AdminListTickets(c)
	require.Equal(t, 403, rec.Code)
	admin, adminRec := newSupportContext("GET", "/api/v1/admin/support/tickets", "")
	admin.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
	h.AdminListTickets(admin)
	require.Equal(t, 200, adminRec.Code)
	require.Equal(t, "agent", <-roles)
	require.Contains(t, adminRec.Body.String(), "/api/v1/admin/support/uploads/")
	user, userRec := newSupportContext("GET", "/api/v1/support/tickets", "")
	user.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
	h.ListTickets(user)
	require.Equal(t, 200, userRec.Code)
	require.Equal(t, "user", <-roles)
	require.Contains(t, userRec.Body.String(), "/api/v1/support/uploads/")
}

func TestSupportPortal_FlagDisablesEveryHandler(t *testing.T) {
	h := newSupportHandlerForTest("http://127.0.0.1:1", false)
	for _, fn := range []gin.HandlerFunc{h.Chat, h.ConfirmTicket, h.GetThread, h.GetTicket, h.ListTickets, h.TicketStats, h.CreateTicket, h.ReplyTicket, h.ResolveTicket, h.CloseTicket, h.ReopenTicket, h.Upload, h.DownloadUpload, h.AdminListTickets, h.AdminGetTicket, h.AdminTicketStats, h.AdminReplyTicket, h.AdminResolveTicket, h.AdminCloseTicket, h.AdminReopenTicket, h.AdminUpload, h.AdminDownloadUpload} {
		c, rec := newSupportContext("POST", "/api/v1/support/tickets", "{}")
		fn(c)
		require.Equal(t, 404, rec.Code)
	}
}

func multipartSupportRequest(t *testing.T, filename string, data []byte) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	file, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = file.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	c, rec := newSupportContext("POST", "/api/v1/support/uploads", "")
	c.Request = httptest.NewRequest("POST", "/api/v1/support/uploads", &buffer)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c, rec
}

func TestSupportPortal_PrivateUploadDownloadAndValidation(t *testing.T) {
	var metadata map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&metadata)
		} else {
			metadata["storage_key"] = metadata["id"]
		}
		_ = json.NewEncoder(w).Encode(metadata)
	}))
	defer upstream.Close()
	h := newSupportHandlerForTest(upstream.URL, true)
	h.SetUploadsDir(t.TempDir())
	c, rec := multipartSupportRequest(t, "notes.txt", []byte("Support attachment"))
	h.Upload(c)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	id := metadata["id"].(string)
	_, err := uuid.Parse(id)
	require.NoError(t, err)
	stat, err := os.Stat(filepath.Join(h.uploadsDir, id))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), stat.Mode().Perm())
	download, downloadRec := newSupportContext("GET", "/api/v1/support/uploads/"+id, "")
	download.Params = gin.Params{{Key: "id", Value: id}}
	h.DownloadUpload(download)
	require.Equal(t, 200, downloadRec.Code, downloadRec.Body.String())
	require.Equal(t, "Support attachment", downloadRec.Body.String())
	require.Contains(t, downloadRec.Header().Get("Content-Disposition"), "attachment")
	require.Equal(t, "private, no-store", downloadRec.Header().Get("Cache-Control"))
	for _, tc := range []struct {
		name string
		data []byte
	}{{"auth.json", []byte("{}")}, {"fake.png", []byte("plain text")}, {"oversized.txt", bytes.Repeat([]byte("x"), int(supportMaxUploadBytes)+1)}} {
		invalid, invalidRec := multipartSupportRequest(t, tc.name, tc.data)
		h.Upload(invalid)
		require.Equal(t, 400, invalidRec.Code)
	}
}

func TestSupportPortal_UploadForbiddenOrClosedLeavesNoFile(t *testing.T) {
	for _, status := range []int{403, 409} {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"detail":"attachment rejected"}`)
		}))
		h := newSupportHandlerForTest(upstream.URL, true)
		h.SetUploadsDir(t.TempDir())
		c, rec := multipartSupportRequest(t, "notes.txt", []byte("Support attachment"))
		h.Upload(c)
		require.Equal(t, status, rec.Code)
		files, err := os.ReadDir(h.uploadsDir)
		require.NoError(t, err)
		require.Empty(t, files)
		upstream.Close()
	}
}
