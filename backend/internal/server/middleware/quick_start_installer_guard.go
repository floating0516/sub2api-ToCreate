package middleware

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var errQuickStartInstallerDisabled = infraerrors.Forbidden(
	"quick_start_disabled",
	"quick start installer is not enabled",
)

// QuickStartInstallerGuard allows administrators to preview the installer
// while requiring the public feature switch for ordinary users.
func QuickStartInstallerGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := GetUserRoleFromContext(c)
		if role == service.RoleAdmin {
			c.Next()
			return
		}
		if settingService != nil && settingService.IsQuickStartInstallerEnabled(c.Request.Context()) {
			c.Next()
			return
		}

		response.ErrorFrom(c, errQuickStartInstallerDisabled)
		c.Abort()
	}
}
