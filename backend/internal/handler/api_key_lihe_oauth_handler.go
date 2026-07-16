package handler

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type liheOAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func setLiheOAuthNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	// Long-lived credentials must never be included in referrers.
	c.Header("Referrer-Policy", "no-referrer")
}

func writeLiheOAuthError(c *gin.Context, status int, code, description string) {
	setLiheOAuthNoStoreHeaders(c)
	c.JSON(status, liheOAuthErrorResponse{Error: code, ErrorDescription: description})
}

func requireLiheOAuthForm(c *gin.Context) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		writeLiheOAuthError(c, http.StatusBadRequest, "invalid_request", "application/x-www-form-urlencoded is required")
		return false
	}
	if err := c.Request.ParseForm(); err != nil {
		writeLiheOAuthError(c, http.StatusBadRequest, "invalid_request", "request form is invalid")
		return false
	}
	return true
}

func (h *APIKeyHandler) authenticateLiheOAuthClient(c *gin.Context) bool {
	clientID, clientSecret, ok := c.Request.BasicAuth()
	if !ok || !h.apiKeyService.AuthenticateLiheOAuthClient(clientID, clientSecret) {
		c.Header("WWW-Authenticate", `Basic realm="lihe-oauth"`)
		writeLiheOAuthError(c, http.StatusUnauthorized, "invalid_client", "client authentication failed")
		return false
	}
	return true
}

// AuthorizeLiheOAuth validates the API user and returns a redirect to the one
// configured LibreChat callback. The browser receives only a single-use code.
func (h *APIKeyHandler) AuthorizeLiheOAuth(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	result, err := h.apiKeyService.CreateLiheAuthorizationCode(
		c.Request.Context(),
		subject.UserID,
		service.LiheAuthorizeRequest{
			ResponseType:        c.Query("response_type"),
			ClientID:            c.Query("client_id"),
			RedirectURI:         c.Query("redirect_uri"),
			Scope:               c.Query("scope"),
			State:               c.Query("state"),
			CodeChallenge:       c.Query("code_challenge"),
			CodeChallengeMethod: c.Query("code_challenge_method"),
		},
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	setLiheOAuthNoStoreHeaders(c)
	response.Success(c, result)
}

// ExchangeLiheOAuthToken consumes a 60-second authorization code and returns
// the plaintext long-lived token exactly once.
func (h *APIKeyHandler) ExchangeLiheOAuthToken(c *gin.Context) {
	setLiheOAuthNoStoreHeaders(c)
	if !requireLiheOAuthForm(c) || !h.authenticateLiheOAuthClient(c) {
		return
	}
	if c.PostForm("grant_type") != "authorization_code" {
		writeLiheOAuthError(c, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code")
		return
	}
	result, err := h.apiKeyService.ExchangeLiheAuthorizationCode(
		c.Request.Context(),
		c.PostForm("code"),
		c.PostForm("redirect_uri"),
		c.PostForm("code_verifier"),
	)
	if err != nil {
		h.writeLiheTokenExchangeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *APIKeyHandler) writeLiheTokenExchangeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrLiheInvalidGrant):
		writeLiheOAuthError(c, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
	case errors.Is(err, service.ErrLiheNoProviders):
		writeLiheOAuthError(c, http.StatusForbidden, "access_denied", "this account has no available model providers")
	case infraerrors.Reason(err) == "LIHE_USER_INACTIVE":
		writeLiheOAuthError(c, http.StatusForbidden, "access_denied", "user account is not active")
	case errors.Is(err, service.ErrLiheOAuthDisabled):
		writeLiheOAuthError(c, http.StatusServiceUnavailable, "temporarily_unavailable", "Lihe Chat integration is not enabled")
	case errors.Is(err, service.ErrLiheOAuthRepositoryUnavailable):
		writeLiheOAuthError(c, http.StatusServiceUnavailable, "temporarily_unavailable", "Lihe Chat integration is temporarily unavailable")
	default:
		writeLiheOAuthError(c, http.StatusInternalServerError, "server_error", "token exchange failed")
	}
}

// RevokeLiheOAuthToken is intentionally idempotent per RFC 7009.
func (h *APIKeyHandler) RevokeLiheOAuthToken(c *gin.Context) {
	setLiheOAuthNoStoreHeaders(c)
	if !requireLiheOAuthForm(c) || !h.authenticateLiheOAuthClient(c) {
		return
	}
	if err := h.apiKeyService.RevokeLiheAccessToken(c.Request.Context(), c.PostForm("token")); err != nil {
		writeLiheOAuthError(c, http.StatusServiceUnavailable, "temporarily_unavailable", "token revocation failed")
		return
	}
	c.Status(http.StatusOK)
}

func (h *APIKeyHandler) GetLiheIntegration(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	enabled, connectURL := h.apiKeyService.LiheOAuthPublicConfig()
	tokens := make([]service.LiheAccessToken, 0)
	if enabled {
		var err error
		tokens, err = h.apiKeyService.ListLiheAccessTokens(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	response.Success(c, gin.H{
		"enabled":     enabled,
		"connect_url": connectURL,
		"tokens":      tokens,
	})
}

func (h *APIKeyHandler) ListLiheAccessTokens(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	tokens, err := h.apiKeyService.ListLiheAccessTokens(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tokens)
}

func (h *APIKeyHandler) RevokeMyLiheAccessToken(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	tokenID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || tokenID <= 0 {
		response.BadRequest(c, "Invalid connection ID")
		return
	}
	if err := h.apiKeyService.RevokeLiheAccessTokenByID(c.Request.Context(), tokenID, subject.UserID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"revoked": true})
}

func (h *APIKeyHandler) ListLiheAccessTokensAsAdmin(c *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(c.Query("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Valid user_id is required")
		return
	}
	tokens, err := h.apiKeyService.ListLiheAccessTokens(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tokens)
}

func (h *APIKeyHandler) RevokeLiheAccessTokenAsAdmin(c *gin.Context) {
	tokenID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || tokenID <= 0 {
		response.BadRequest(c, "Invalid connection ID")
		return
	}
	if err := h.apiKeyService.RevokeLiheAccessTokenByIDAsAdmin(c.Request.Context(), tokenID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"revoked": true})
}
