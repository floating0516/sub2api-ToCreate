package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterLiheOAuthRoutes exposes the confidential-client endpoints at the
// OAuth-standard root paths and the API-user management endpoints under v1.
func RegisterLiheOAuthRoutes(
	r *gin.Engine,
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
) {
	r.POST("/oauth/token", h.APIKey.ExchangeLiheOAuthToken)
	r.POST("/oauth/revoke", h.APIKey.RevokeLiheOAuthToken)

	lihe := v1.Group("/oauth/lihe")
	lihe.Use(gin.HandlerFunc(jwtAuth))
	lihe.Use(middleware.BackendModeUserGuard(settingService))
	lihe.Use(gin.HandlerFunc(auditLog))
	{
		lihe.GET("", h.APIKey.GetLiheIntegration)
		lihe.GET("/authorize", h.APIKey.AuthorizeLiheOAuth)
		lihe.GET("/tokens", h.APIKey.ListLiheAccessTokens)
		lihe.DELETE("/tokens/:id", h.APIKey.RevokeMyLiheAccessToken)
	}

	admin := v1.Group("/admin/oauth/lihe")
	admin.Use(gin.HandlerFunc(adminAuth))
	admin.Use(gin.HandlerFunc(auditLog))
	{
		admin.GET("/tokens", h.APIKey.ListLiheAccessTokensAsAdmin)
		admin.DELETE("/tokens/:id", h.APIKey.RevokeLiheAccessTokenAsAdmin)
	}
}
