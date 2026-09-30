package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterSupportRoutes exposes the authenticated same-origin BFF. The browser
// never receives the agent JWT; SupportHandler mints it per request.
func RegisterSupportRoutes(
	v1 *gin.RouterGroup,
	h *handler.SupportHandler,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	authenticated := v1.Group("/support")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	authenticated.Use(panelRateLimiter.Global())
	authenticated.Use(func(c *gin.Context) {
		if settingService == nil || !settingService.IsSupportAgentEnabled(c.Request.Context()) {
			response.NotFound(c, "support is not enabled")
			c.Abort()
			return
		}
		c.Next()
	})
	{
		authenticated.POST("/chat", h.Chat)
		authenticated.POST("/tickets/confirm", h.ConfirmTicket)
		authenticated.GET("/threads/:id", h.GetThread)
		authenticated.GET("/tickets", h.ListTickets)
		authenticated.GET("/tickets/stats", h.TicketStats)
		authenticated.POST("/tickets", h.CreateTicket)
		authenticated.GET("/tickets/:id", h.GetTicket)
		authenticated.POST("/tickets/:id/replies", h.ReplyTicket)
		authenticated.POST("/tickets/:id/resolve", h.ResolveTicket)
		authenticated.POST("/tickets/:id/close", h.CloseTicket)
		authenticated.POST("/tickets/:id/reopen", h.ReopenTicket)
		authenticated.POST("/uploads", h.Upload)
		authenticated.GET("/uploads/:id", h.DownloadUpload)
	}
	admin := v1.Group("/admin/support")
	admin.Use(gin.HandlerFunc(jwtAuth), middleware.AdminOnly(), panelRateLimiter.Global())
	admin.Use(func(c *gin.Context) {
		if settingService == nil || !settingService.IsSupportAgentEnabled(c.Request.Context()) {
			response.NotFound(c, "support is not enabled"); c.Abort(); return
		}
		c.Next()
	})
	admin.GET("/tickets", h.AdminListTickets)
	admin.GET("/tickets/stats", h.AdminTicketStats)
	admin.GET("/tickets/:id", h.AdminGetTicket)
	admin.POST("/tickets/:id/replies", h.AdminReplyTicket)
	admin.POST("/tickets/:id/resolve", h.AdminResolveTicket)
	admin.POST("/tickets/:id/close", h.AdminCloseTicket)
	admin.POST("/tickets/:id/reopen", h.AdminReopenTicket)
	admin.POST("/uploads", h.AdminUpload)
	admin.GET("/uploads/:id", h.AdminDownloadUpload)
}
