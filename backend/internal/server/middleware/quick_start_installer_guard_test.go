//go:build unit

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestQuickStartInstallerGuard(t *testing.T) {
	tests := []struct {
		name       string
		nilService bool
		setting    *string
		role       string
		wantStatus int
		wantReason string
	}{
		{
			name:       "missing setting blocks ordinary user",
			role:       service.RoleUser,
			wantStatus: http.StatusForbidden,
			wantReason: "quick_start_disabled",
		},
		{
			name:       "disabled setting blocks ordinary user",
			setting:    quickStartStringPtr("false"),
			role:       service.RoleUser,
			wantStatus: http.StatusForbidden,
			wantReason: "quick_start_disabled",
		},
		{
			name:       "enabled setting allows ordinary user",
			setting:    quickStartStringPtr("true"),
			role:       service.RoleUser,
			wantStatus: http.StatusOK,
		},
		{
			name:       "administrator bypasses disabled setting",
			setting:    quickStartStringPtr("false"),
			role:       service.RoleAdmin,
			wantStatus: http.StatusOK,
		},
		{
			name:       "administrator bypasses missing service",
			nilService: true,
			role:       service.RoleAdmin,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing service fails closed for ordinary user",
			nilService: true,
			role:       service.RoleUser,
			wantStatus: http.StatusForbidden,
			wantReason: "quick_start_disabled",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			var settingService *service.SettingService
			if !tc.nilService {
				values := map[string]string{}
				if tc.setting != nil {
					values[service.SettingKeyQuickStartInstallerEnabled] = *tc.setting
				}
				settingService = service.NewSettingService(&bmSettingRepo{values: values}, &config.Config{})
			}

			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(string(ContextKeyUserRole), tc.role)
				c.Next()
			})
			router.Use(QuickStartInstallerGuard(settingService))
			router.POST("/install-token", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/install-token", nil)
			router.ServeHTTP(recorder, request)

			require.Equal(t, tc.wantStatus, recorder.Code)
			if tc.wantReason == "" {
				return
			}
			var body struct {
				Reason string `json:"reason"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, tc.wantReason, body.Reason)
		})
	}
}

func quickStartStringPtr(value string) *string {
	return &value
}
