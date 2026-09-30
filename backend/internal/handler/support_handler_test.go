//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type supportHandlerRepoStub struct {
	enabled bool
}

func (s *supportHandlerRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, errors.New("not implemented")
}

func (s *supportHandlerRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key == service.SettingKeySupportAgentEnabled {
		if s.enabled {
			return "true", nil
		}
		return "false", nil
	}
	return "", errors.New("setting not found")
}

func (s *supportHandlerRepoStub) Set(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (s *supportHandlerRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (s *supportHandlerRepoStub) SetMultiple(context.Context, map[string]string) error {
	return errors.New("not implemented")
}

func (s *supportHandlerRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, errors.New("not implemented")
}

func (s *supportHandlerRepoStub) Delete(context.Context, string) error {
	return errors.New("not implemented")
}

func newSupportHandlerForTest(baseURL string, enabled bool) *SupportHandler {
	cfg := &config.Config{
		SupportAgent: config.SupportAgentConfig{
			BaseURL:         baseURL,
			JWTSecret:       "support-handler-test-secret-32bytes",
			Issuer:          "support-agent",
			Audience:        "support-agent",
			TokenTTLSeconds: 120,
		},
	}
	settings := service.NewSettingService(&supportHandlerRepoStub{enabled: enabled}, cfg)
	return NewSupportHandler(cfg, settings)
}

func newSupportContext(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 123})
	return c, recorder
}

func TestSupportHandler_ReplacesBrowserCredentialsAndIssuesScopedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	type capturedRequest struct {
		authorization string
		cookie        string
	}
	captured := make(chan capturedRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured <- capturedRequest{
			authorization: r.Header.Get("Authorization"),
			cookie:        r.Header.Get("Cookie"),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"thread_id":"ba4a51ed-1d20-4ef5-878b-37c70e9ef45a","answer":"ok","citations":[]}`))
	}))
	defer upstream.Close()

	h := newSupportHandlerForTest(upstream.URL, true)
	c, recorder := newSupportContext(
		http.MethodPost,
		"/api/v1/support/chat",
		`{"message":"Codex 401","client_message_id":"87b2580f-6a03-44d5-9e0c-c96741a20ce7"}`,
	)
	c.Request.Header.Set("Authorization", "Bearer console-jwt")
	c.Request.Header.Set("Cookie", "session=browser-cookie")

	h.Chat(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	request := <-captured
	require.Empty(t, request.cookie)
	require.NotEqual(t, "Bearer console-jwt", request.authorization)
	require.True(t, strings.HasPrefix(request.authorization, "Bearer "))

	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		strings.TrimPrefix(request.authorization, "Bearer "),
		claims,
		func(token *jwt.Token) (any, error) {
			return []byte("support-handler-test-secret-32bytes"), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("support-agent"),
		jwt.WithAudience("support-agent"),
	)
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, "123", claims.Subject)
	require.Equal(t, jwt.SigningMethodHS256.Alg(), token.Method.Alg())

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Answer string `json:"answer"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Zero(t, envelope.Code)
	require.Equal(t, "ok", envelope.Data.Answer)
}

func TestSupportHandler_RejectsUnknownChatFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newSupportHandlerForTest("http://127.0.0.1:1", true)
	c, recorder := newSupportContext(
		http.MethodPost,
		"/api/v1/support/chat",
		`{"message":"help","client_message_id":"87b2580f-6a03-44d5-9e0c-c96741a20ce7","product":"forged"}`,
	)

	h.Chat(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestSupportHandler_DisabledReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newSupportHandlerForTest("http://127.0.0.1:1", false)
	c, recorder := newSupportContext(http.MethodGet, "/api/v1/support/threads/ba4a51ed-1d20-4ef5-878b-37c70e9ef45a", "")
	c.Params = gin.Params{{Key: "id", Value: "ba4a51ed-1d20-4ef5-878b-37c70e9ef45a"}}

	h.GetThread(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestSupportHandler_MapsUpstreamUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	h := newSupportHandlerForTest(upstream.URL, true)
	c, recorder := newSupportContext(
		http.MethodPost,
		"/api/v1/support/chat",
		`{"message":"help","client_message_id":"87b2580f-6a03-44d5-9e0c-c96741a20ce7"}`,
	)

	h.Chat(c)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	var envelope struct {
		Reason string `json:"reason"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, "SUPPORT_UPSTREAM_AUTH_FAILED", envelope.Reason)
}

func TestSupportHandler_MapsTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"answer":"too late"}`))
	}))
	defer upstream.Close()

	h := newSupportHandlerForTest(upstream.URL, true)
	h.client.Timeout = 20 * time.Millisecond
	c, recorder := newSupportContext(
		http.MethodPost,
		"/api/v1/support/chat",
		`{"message":"help","client_message_id":"87b2580f-6a03-44d5-9e0c-c96741a20ce7"}`,
	)

	h.Chat(c)

	require.Equal(t, http.StatusGatewayTimeout, recorder.Code)
}

func TestSupportHandler_RejectsInvalidUUIDAndMissingConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newSupportHandlerForTest("http://127.0.0.1:1", true)

	invalid, invalidRecorder := newSupportContext(http.MethodGet, "/api/v1/support/threads/not-a-uuid", "")
	invalid.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	h.GetThread(invalid)
	require.Equal(t, http.StatusBadRequest, invalidRecorder.Code)

	missing, missingRecorder := newSupportContext(
		http.MethodPost,
		"/api/v1/support/tickets/confirm",
		`{"thread_id":"ba4a51ed-1d20-4ef5-878b-37c70e9ef45a"}`,
	)
	h.ConfirmTicket(missing)
	require.Equal(t, http.StatusBadRequest, missingRecorder.Code)
}

func TestMapSupportStatus_PreservesExpectedClientErrors(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusUnprocessableEntity,
		http.StatusTooManyRequests,
	} {
		require.Equal(t, status, mapSupportStatus(status))
	}
	require.Equal(t, http.StatusBadGateway, mapSupportStatus(http.StatusUnauthorized))
}
