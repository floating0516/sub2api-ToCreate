package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/oidcprovider"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	liheOIDCBrowserCookie = "__Host-lihe_oidc_browser"
	liheOIDCCookieTTL     = 10 * time.Minute
)

type LiheOIDCHandler struct {
	provider *oidcprovider.Provider
}

func NewLiheOIDCHandler(provider *oidcprovider.Provider) *LiheOIDCHandler {
	return &LiheOIDCHandler{provider: provider}
}

type prepareLiheOIDCRequest struct {
	Params map[string]string `json:"params" binding:"required"`
}

type authorizeLiheOIDCRequest struct {
	RequestID string `json:"request_id" binding:"required"`
}

func (h *LiheOIDCHandler) Discovery(c *gin.Context) {
	document, err := h.provider.Discovery()
	if err != nil {
		h.writeUnavailable(c)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, document)
}

func (h *LiheOIDCHandler) JWKS(c *gin.Context) {
	keys, err := h.provider.JWKS()
	if err != nil {
		h.writeUnavailable(c)
		return
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.JSON(http.StatusOK, keys)
}

func (h *LiheOIDCHandler) PrepareAuthorization(c *gin.Context) {
	setOIDCProtocolHeaders(c)
	if h.provider == nil || !h.provider.Enabled() {
		h.writeUnavailable(c)
		return
	}
	var request prepareLiheOIDCRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, "Invalid OIDC authorization request", "OIDC_INVALID_REQUEST", nil)
		return
	}
	browserBinding, err := ensureOIDCBrowserBinding(c)
	if err != nil {
		response.InternalError(c, "Unable to initialize OIDC authorization")
		return
	}
	params := make(url.Values, len(request.Params))
	for key, value := range request.Params {
		params.Set(key, value)
	}
	prepared, err := h.provider.PrepareAuthorization(c.Request.Context(), params, browserBinding)
	if err != nil {
		writeOIDCPreparationError(c, err)
		return
	}
	response.Success(c, prepared)
}

func (h *LiheOIDCHandler) Authorize(c *gin.Context) {
	setOIDCProtocolHeaders(c)
	servermiddleware.SetAuditAction(c, "oidc.authorize")
	if h.provider == nil || !h.provider.Enabled() {
		h.writeUnavailable(c)
		return
	}
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var request authorizeLiheOIDCRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, "Invalid OIDC authorization request", "OIDC_INVALID_REQUEST", nil)
		return
	}
	browserBinding, err := readOIDCBrowserBinding(c)
	if err != nil {
		response.ErrorWithDetails(c, http.StatusBadRequest, "OIDC authorization request expired", "OIDC_REQUEST_EXPIRED", nil)
		return
	}
	result, err := h.provider.Authorize(
		c.Request.Context(),
		strings.TrimSpace(request.RequestID),
		browserBinding,
		subject.UserID,
		subject.AuthenticatedAt,
	)
	if err != nil {
		if errors.Is(err, oidcprovider.ErrPendingRequestNotFound) {
			response.ErrorWithDetails(c, http.StatusBadRequest, "OIDC authorization request expired", "OIDC_REQUEST_EXPIRED", nil)
			return
		}
		if errors.Is(err, oidcprovider.ErrUserInactive) {
			response.Forbidden(c, "User account is not active")
			return
		}
		slog.ErrorContext(c.Request.Context(), "OIDC authorization failed", "event", "oidc_authorize_failed")
		response.InternalError(c, "OIDC authorization failed")
		return
	}
	auditResult := "issued"
	if result.Reauthenticate {
		auditResult = "reauthentication_required"
	} else if strings.Contains(result.RedirectTo, "error=login_required") {
		auditResult = "login_required"
	}
	servermiddleware.SetAuditExtra(c, map[string]any{"result": auditResult})
	response.Success(c, result)
}

func (h *LiheOIDCHandler) Token(c *gin.Context) {
	setOIDCProtocolHeaders(c)
	if h.provider == nil || !h.provider.Enabled() {
		h.writeUnavailable(c)
		return
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		writeOIDCProtocolError(c, http.StatusBadRequest, "invalid_request", "application/x-www-form-urlencoded is required")
		return
	}
	profile, err := h.provider.HandleToken(c.Request.Context(), c.Writer, c.Request)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "OIDC token request rejected",
			"event", "oidc_token_rejected",
			"error_code", oidcprovider.ProtocolErrorCode(err),
		)
		return
	}
	slog.InfoContext(c.Request.Context(), "OIDC token issued",
		"event", "oidc_token_issued",
		"user_id", profile.UserID,
	)
}

func (h *LiheOIDCHandler) UserInfo(c *gin.Context) {
	setOIDCProtocolHeaders(c)
	if h.provider == nil || !h.provider.Enabled() {
		h.writeUnavailable(c)
		return
	}
	authorization := strings.TrimSpace(c.GetHeader("Authorization"))
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeOIDCUserInfoUnauthorized(c)
		return
	}
	claims, profile, err := h.provider.UserInfo(c.Request.Context(), strings.TrimSpace(parts[1]))
	if err != nil {
		slog.WarnContext(c.Request.Context(), "OIDC UserInfo request rejected",
			"event", "oidc_userinfo_rejected",
			"error_code", oidcprovider.ProtocolErrorCode(err),
		)
		writeOIDCUserInfoUnauthorized(c)
		return
	}
	slog.InfoContext(c.Request.Context(), "OIDC UserInfo request accepted",
		"event", "oidc_userinfo_accepted",
		"user_id", profile.UserID,
	)
	c.JSON(http.StatusOK, claims)
}

func (h *LiheOIDCHandler) writeUnavailable(c *gin.Context) {
	response.ErrorWithDetails(c, http.StatusNotFound, "OIDC Provider is not enabled", "OIDC_PROVIDER_DISABLED", nil)
}

func setOIDCProtocolHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
}

func ensureOIDCBrowserBinding(c *gin.Context) (string, error) {
	if existing, err := readOIDCBrowserBinding(c); err == nil {
		return existing, nil
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	binding := base64.RawURLEncoding.EncodeToString(raw)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     liheOIDCBrowserCookie,
		Value:    binding,
		Path:     "/",
		MaxAge:   int(liheOIDCCookieTTL / time.Second),
		Expires:  time.Now().UTC().Add(liheOIDCCookieTTL),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return binding, nil
}

func readOIDCBrowserBinding(c *gin.Context) (string, error) {
	cookie, err := c.Request.Cookie(liheOIDCBrowserCookie)
	if err != nil || cookie == nil {
		return "", oidcprovider.ErrInvalidBrowserBinding
	}
	value := strings.TrimSpace(cookie.Value)
	if len(value) != 43 {
		return "", oidcprovider.ErrInvalidBrowserBinding
	}
	if _, err := base64.RawURLEncoding.DecodeString(value); err != nil {
		return "", oidcprovider.ErrInvalidBrowserBinding
	}
	return value, nil
}

func writeOIDCPreparationError(c *gin.Context, err error) {
	slog.WarnContext(c.Request.Context(), "OIDC authorization request rejected",
		"event", "oidc_prepare_rejected",
		"error_code", oidcprovider.ProtocolErrorCode(err),
	)
	switch {
	case errors.Is(err, oidcprovider.ErrProviderDisabled):
		response.ErrorWithDetails(c, http.StatusNotFound, "OIDC Provider is not enabled", "OIDC_PROVIDER_DISABLED", nil)
	case errors.Is(err, oidcprovider.ErrInvalidRequest):
		response.ErrorWithDetails(c, http.StatusBadRequest, "Invalid OIDC authorization request", "OIDC_INVALID_REQUEST", nil)
	default:
		code := oidcprovider.ProtocolErrorCode(err)
		if code == "invalid_scope" || code == "invalid_request" || code == "invalid_state" || code == "insufficient_entropy" {
			response.ErrorWithDetails(c, http.StatusBadRequest, "Invalid OIDC authorization request", "OIDC_INVALID_REQUEST", nil)
			return
		}
		slog.ErrorContext(c.Request.Context(), "OIDC request preparation failed", "event", "oidc_prepare_failed")
		response.InternalError(c, "Unable to prepare OIDC authorization")
	}
}

func writeOIDCProtocolError(c *gin.Context, status int, code, description string) {
	c.JSON(status, gin.H{"error": code, "error_description": description})
}

func writeOIDCUserInfoUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer realm="lihe-oidc", error="invalid_token"`)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
}
