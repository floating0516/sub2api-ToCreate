package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterInstallTokenRoutes(
	v1 *gin.RouterGroup,
	h *handler.InstallTokenHandler,
	jwtAuth middleware.JWTAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	public := v1.Group("/install-token")
	public.Use(panelRateLimiter.PublicIP())
	{
		public.POST("/peek", h.Peek)
		public.POST("/redeem", h.Redeem)
		public.POST("/confirm", h.Confirm)
	}

	authenticated := v1.Group("/install-token")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	authenticated.Use(panelRateLimiter.Global())
	authenticated.Use(gin.HandlerFunc(auditLog))
	{
		authenticated.POST("", h.Issue)
		authenticated.POST("/revoke", h.Revoke)
	}
}
