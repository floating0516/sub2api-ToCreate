//go:build unit

package routes

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterSupport_ExplicitPortalRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterSupportRoutes(r.Group("/api/v1"), &handler.SupportHandler{}, middleware.JWTAuthMiddleware(func(c *gin.Context) { c.Next() }), nil, middleware.NewPanelRateLimiter(nil,nil))
	routes := map[string]bool{}
	for _, route := range r.Routes() { routes[route.Method+" "+route.Path] = true }
	for _, prefix := range []string{"/api/v1/support","/api/v1/admin/support"} {
		for _, path := range []string{"GET /tickets","GET /tickets/stats","GET /tickets/:id","POST /tickets/:id/replies","POST /tickets/:id/resolve","POST /tickets/:id/close","POST /tickets/:id/reopen","POST /uploads","GET /uploads/:id"} {
			method, suffix, _ := strings.Cut(path, " ")
			require.True(t, routes[method+" "+prefix+suffix], path)
		}
	}
	require.True(t,routes["POST /api/v1/support/tickets"])
	for key := range routes { require.NotContains(t,key,"*path") }
}
