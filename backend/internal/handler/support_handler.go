package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	supportMaxRequestBytes    int64 = 64 * 1024
	defaultSupportTimeout           = 60 * time.Second
	defaultSupportMaxResponse       = 1 << 20
)

type SupportHandler struct {
	settingService *service.SettingService
	cfg            *config.Config
	client         *http.Client
	inflightMu     sync.Mutex
	inflight       map[int64]int
	validateOrder  func(context.Context, int64, int64) error
	uploadsDir     string
}

func NewSupportHandler(cfg *config.Config, settingService *service.SettingService) *SupportHandler {
	timeout := defaultSupportTimeout
	if cfg != nil && cfg.SupportAgent.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.SupportAgent.TimeoutSeconds) * time.Second
	}
	return &SupportHandler{
		settingService: settingService,
		cfg:            cfg,
		client:         &http.Client{Timeout: timeout},
		inflight:       make(map[int64]int),
	}
}

type supportChatRequest struct {
	Message         string     `json:"message"`
	ClientMessageID uuid.UUID  `json:"client_message_id"`
	ThreadID        *uuid.UUID `json:"thread_id,omitempty"`
}

type supportConfirmRequest struct {
	ThreadID uuid.UUID `json:"thread_id"`
	Confirm  *bool     `json:"confirm"`
}

func (h *SupportHandler) Chat(c *gin.Context) { h.proxyJSON(c, http.MethodPost, "/chat", true) }
func (h *SupportHandler) ConfirmTicket(c *gin.Context) {
	h.proxyJSON(c, http.MethodPost, "/tickets/confirm", true)
}
func (h *SupportHandler) GetThread(c *gin.Context) {
	if !h.supportReady(c, "/threads") { return }
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.BadRequest(c, "invalid thread id")
		return
	}
	h.proxyJSON(c, http.MethodGet, "/threads/"+c.Param("id"), false)
}
func (h *SupportHandler) GetTicket(c *gin.Context) {
	if !h.supportReady(c, "/tickets") { return }
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.BadRequest(c, "invalid ticket id")
		return
	}
	h.proxyJSON(c, http.MethodGet, "/tickets/"+c.Param("id"), false)
}

func (h *SupportHandler) proxyJSON(c *gin.Context, method, path string, hasBody bool) {
	if !h.supportReady(c, path) { return }
	var body []byte
	if hasBody {
		var err error
		body, err = h.readPortalRequest(c, path)
		if err != nil {
			if !errors.Is(err, errSupportResponseWritten) { response.BadRequest(c, err.Error()) }
			return
		}
	}
	if payload, ok := h.requestJSON(c, method, path, body); ok { response.Success(c, payload) }
}

func (h *SupportHandler) supportReady(c *gin.Context, path string) bool {
	if h == nil || h.settingService == nil || !h.settingService.IsSupportAgentEnabled(c.Request.Context()) {
		response.NotFound(c, "support is not enabled")
		return false
	}
	if h.cfg == nil || strings.TrimSpace(h.cfg.SupportAgent.BaseURL) == "" || strings.TrimSpace(h.cfg.SupportAgent.JWTSecret) == "" {
		response.Error(c, http.StatusServiceUnavailable, "support service is not configured")
		return false
	}
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "UNAUTHORIZED")
		return false
	}
	if strings.HasPrefix(path, "/admin/") {
		role, hasRole := servermiddleware.GetUserRoleFromContext(c)
		if !hasRole || role != service.RoleAdmin {
			response.Forbidden(c, "admin access required")
			return false
		}
	}
	return true
}

// requestJSON is called only by explicit handlers with fixed upstream paths.
func (h *SupportHandler) requestJSON(c *gin.Context, method, path string, body []byte) (any, bool) {
	if !h.supportReady(c, path) { return nil, false }
	subject, _ := servermiddleware.GetAuthSubjectFromContext(c)
	if !h.acquire(subject.UserID) {
		response.Error(c, http.StatusTooManyRequests, "support request already in progress")
		return nil, false
	}
	defer h.release(subject.UserID)

	role := "user"
	if strings.HasPrefix(path, "/admin/") { role = "agent" }
	token, err := h.issueSupportToken(subject.UserID, role)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "support service is not configured")
		return nil, false
	}
	base := strings.TrimRight(strings.TrimSpace(h.cfg.SupportAgent.BaseURL), "/")
	req, err := http.NewRequestWithContext(c.Request.Context(), method, base+path, bytes.NewReader(body))
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "support service is unavailable")
		return nil, false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			response.Error(c, http.StatusGatewayTimeout, "support service timed out")
			return nil, false
		}
		response.Error(c, http.StatusServiceUnavailable, "support service is unavailable")
		return nil, false
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	maxBytes := int64(defaultSupportMaxResponse)
	if h.cfg.SupportAgent.MaxResponseBytes > 0 {
		maxBytes = h.cfg.SupportAgent.MaxResponseBytes
	}
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if readErr != nil || int64(len(raw)) > maxBytes {
		response.Error(c, http.StatusBadGateway, "invalid support service response")
		return nil, false
	}
	if resp.StatusCode == http.StatusUnauthorized {
		response.ErrorWithDetails(c, http.StatusBadGateway, "support service authentication failed", "SUPPORT_UPSTREAM_AUTH_FAILED", nil)
		return nil, false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := "support service request failed"
		var failure struct { Detail string `json:"detail"` }
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && json.Unmarshal(raw, &failure) == nil && failure.Detail != "" && len(failure.Detail) <= 1000 { message = failure.Detail }
		response.Error(c, mapSupportStatus(resp.StatusCode), message)
		return nil, false
	}
	var payload any
	if len(raw) == 0 {
		payload = map[string]any{}
	} else if err := json.Unmarshal(raw, &payload); err != nil {
		response.Error(c, http.StatusBadGateway, "invalid support service response")
		return nil, false
	}
	return payload, true
}

func (h *SupportHandler) acquire(userID int64) bool {
	h.inflightMu.Lock()
	defer h.inflightMu.Unlock()
	if h.inflight[userID] >= 2 {
		return false
	}
	h.inflight[userID]++
	return true
}

func (h *SupportHandler) release(userID int64) {
	h.inflightMu.Lock()
	defer h.inflightMu.Unlock()
	if h.inflight[userID] <= 1 {
		delete(h.inflight, userID)
		return
	}
	h.inflight[userID]--
}

func readSupportRequest(c *gin.Context) ([]byte, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, supportMaxRequestBytes)
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	var payload any
	if strings.Contains(c.Request.URL.Path, "/chat") {
		payload = &supportChatRequest{}
	} else {
		payload = &supportConfirmRequest{}
	}
	if err := dec.Decode(payload); err != nil {
		return nil, err
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return nil, errors.New("request body must contain one JSON object")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("invalid JSON body")
	}
	switch req := payload.(type) {
	case *supportChatRequest:
		if strings.TrimSpace(req.Message) == "" || len(req.Message) > 8000 || req.ClientMessageID == uuid.Nil {
			return nil, errors.New("message and client_message_id are required")
		}
	case *supportConfirmRequest:
		if req.ThreadID == uuid.Nil || req.Confirm == nil {
			return nil, errors.New("thread_id and confirm are required")
		}
	default:
		return nil, errors.New("unsupported support request")
	}
	return raw, nil
}

type supportClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

func (h *SupportHandler) issueSupportToken(userID int64, roles ...string) (string, error) {
	secret := strings.TrimSpace(h.cfg.SupportAgent.JWTSecret)
	if secret == "" {
		return "", errors.New("support jwt secret is not configured")
	}
	ttl := h.cfg.SupportAgent.TokenTTLSeconds
	if ttl <= 0 {
		ttl = 300
	}
	now := time.Now()
	role := "user"
	if len(roles) > 0 && roles[0] == "agent" { role = "agent" }
	claims := supportClaims{Role: role, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    "support-agent",
		Audience:  jwt.ClaimStrings{"support-agent"},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl) * time.Second)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func firstSupportString(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func mapSupportStatus(status int) int {
	switch status {
	case http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusUnprocessableEntity,
		http.StatusTooManyRequests:
		return status
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return http.StatusGatewayTimeout
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return status
	default:
		return http.StatusBadGateway
	}
}
