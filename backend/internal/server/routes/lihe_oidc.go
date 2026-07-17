package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterLiheOIDCRoutes exposes the login-only OIDC Provider. It is isolated
// from /oauth/*, which remains the long-lived Lihe API Key OAuth protocol.
func RegisterLiheOIDCRoutes(
	r *gin.Engine,
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	auditLog servermiddleware.AuditLogMiddleware,
	settingService *service.SettingService,
	redisClient *redis.Client,
) {
	rateLimiter := appmiddleware.NewRateLimiter(redisClient)
	failClose := appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailClose}

	r.GET(
		"/.well-known/openid-configuration",
		rateLimiter.LimitWithOptions("lihe-oidc-discovery", 300, time.Minute, failClose),
		h.LiheOIDC.Discovery,
	)
	r.GET(
		"/oidc/jwks",
		rateLimiter.LimitWithOptions("lihe-oidc-jwks", 300, time.Minute, failClose),
		h.LiheOIDC.JWKS,
	)
	v1.POST(
		"/oidc/prepare",
		servermiddleware.RequestBodyLimit(32*1024),
		rateLimiter.LimitWithOptions("lihe-oidc-prepare", 30, time.Minute, failClose),
		h.LiheOIDC.PrepareAuthorization,
	)
	r.POST(
		"/oidc/token",
		servermiddleware.RequestBodyLimit(32*1024),
		// Token and UserInfo calls originate from the shared LibreChat backend
		// address, so their IP bucket must accommodate aggregate login traffic.
		rateLimiter.LimitWithOptions("lihe-oidc-token", 600, time.Minute, failClose),
		h.LiheOIDC.Token,
	)
	r.GET(
		"/oidc/userinfo",
		rateLimiter.LimitWithOptions("lihe-oidc-userinfo", 600, time.Minute, failClose),
		h.LiheOIDC.UserInfo,
	)

	authorize := v1.Group("/oidc")
	authorize.Use(gin.HandlerFunc(jwtAuth))
	authorize.Use(servermiddleware.BackendModeUserGuard(settingService))
	authorize.Use(gin.HandlerFunc(auditLog))
	{
		authorize.POST(
			"/authorize",
			servermiddleware.RequestBodyLimit(8*1024),
			rateLimiter.LimitWithOptions("lihe-oidc-authorize", 30, time.Minute, failClose),
			h.LiheOIDC.Authorize,
		)
	}
}
