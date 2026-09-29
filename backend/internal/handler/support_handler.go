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
	supportMaxRequestBytes int64 = 64 * 1024
	defaultSupportTimeout        = 60 * time.Second
	defaultSupportMaxResponse   = 1 << 20
)

type SupportHandler struct {
	settingService *service.SettingService
	cfg            *config.Config
	client         *http.Client
	inflightMu     sync.Mutex
	inflight       map[int64]int
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
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.BadRequest(c, "invalid thread id")
		return
	}
	h.proxyJSON(c, http.MethodGet, "/threads/"+c.Param("id"), false)
}
func (h *SupportHandler) GetTicket(c *gin.Context) {
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		response.BadRequest(c, "invalid ticket id")
		return
	}
	h.proxyJSON(c, http.MethodGet, "/tickets/"+c.Param("id"), false)
}

func (h *SupportHandler) proxyJSON(c *gin.Context, method, path string, hasBody bool) {
	if h == nil || h.settingService == nil || !h.settingService.IsSupportAgentEnabled(c.Request.Context()) {
		response.NotFound(c, "support is not enabled")
		return
	}
	if h.cfg == nil || strings.TrimSpace(h.cfg.SupportAgent.BaseURL) == "" || strings.TrimSpace(h.cfg.SupportAgent.JWTSecret) == "" {
		response.Error(c, http.StatusServiceUnavailable, "support service is not configured")
		return
	}
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "UNAUTHORIZED")
		return
	}
	if !h.acquire(subject.UserID) {
		response.Error(c, http.StatusTooManyRequests, "support request already in progress")
		return
	}
	defer h.release(subject.UserID)

	var body []byte
	if hasBody {
		var err error
		body, err = readSupportRequest(c)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	token, err := h.issueSupportToken(subject.UserID)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "support service is not configured")
		return
	}
	base := strings.TrimRight(strings.TrimSpace(h.cfg.SupportAgent.BaseURL), "/")
	req, err := http.NewRequestWithContext(c.Request.Context(), method, base+path, bytes.NewReader(body))
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "support service is unavailable")
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
			response.Error(c, http.StatusGatewayTimeout, "support service timed out")
			return
		}
		response.Error(c, http.StatusServiceUnavailable, "support service is unavailable")
		return
	}
	defer resp.Body.Close()
	maxBytes := int64(defaultSupportMaxResponse)
	if h.cfg.SupportAgent.MaxResponseBytes > 0 {
		maxBytes = h.cfg.SupportAgent.MaxResponseBytes
	}
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if readErr != nil || int64(len(raw)) > maxBytes {
		response.Error(c, http.StatusBadGateway, "invalid support service response")
		return
	}
	if resp.StatusCode == http.StatusUnauthorized {
		response.ErrorWithDetails(c, http.StatusBadGateway, "support service authentication failed", "SUPPORT_UPSTREAM_AUTH_FAILED", nil)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.Error(c, mapSupportStatus(resp.StatusCode), "support service request failed")
		return
	}
	var payload any
	if len(raw) == 0 {
		payload = map[string]any{}
	} else if err := json.Unmarshal(raw, &payload); err != nil {
		response.Error(c, http.StatusBadGateway, "invalid support service response")
		return
	}
	response.Success(c, payload)
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
	if strings.Contains(c.Request.URL.Path, "/chat") {
		req := payload.(*supportChatRequest)
		if strings.TrimSpace(req.Message) == "" || len(req.Message) > 8000 || req.ClientMessageID == uuid.Nil {
			return nil, errors.New("message and client_message_id are required")
		}
	} else {
		req := payload.(*supportConfirmRequest)
		if req.ThreadID == uuid.Nil || req.Confirm == nil {
			return nil, errors.New("thread_id and confirm are required")
		}
	}
	return raw, nil
}

func (h *SupportHandler) issueSupportToken(userID int64) (string, error) {
	secret := strings.TrimSpace(h.cfg.SupportAgent.JWTSecret)
	if secret == "" {
		return "", errors.New("support jwt secret is not configured")
	}
	ttl := h.cfg.SupportAgent.TokenTTLSeconds
	if ttl <= 0 {
		ttl = 300
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    firstSupportString(h.cfg.SupportAgent.Issuer, "support-agent"),
		Audience:  jwt.ClaimStrings{firstSupportString(h.cfg.SupportAgent.Audience, "support-agent")},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl) * time.Second)),
	}
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
