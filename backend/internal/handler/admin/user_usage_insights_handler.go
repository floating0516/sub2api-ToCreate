package admin

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetUsageInsights loads the expensive list columns separately from user data.
// Registered under the existing admin authentication middleware.
func (h *UserHandler) GetUsageInsights(c *gin.Context) {
	var req struct {
		UserIDs []int64 `json:"user_ids" binding:"required,min=1,max=100,dive,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "user_ids must contain between 1 and 100 positive IDs")
		return
	}
	reader, ok := h.adminService.(service.AdminUserUsageInsightsReader)
	if !ok {
		response.Error(c, http.StatusServiceUnavailable, "User usage insights are unavailable")
		return
	}
	insights, err := reader.GetUserUsageInsights(c.Request.Context(), req.UserIDs)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Failed to load user usage insights")
		return
	}
	type insightDTO struct {
		Summary          *dto.AdminUserUsageSummary     `json:"usage_summary"`
		ModelPreferences []dto.AdminUserModelPreference `json:"model_preferences"`
	}
	out := make(map[int64]insightDTO, len(insights))
	for id, insight := range insights {
		out[id] = insightDTO{
			Summary:          dto.AdminUserUsageSummaryFromService(insight.Summary),
			ModelPreferences: dto.AdminUserModelPreferencesFromService(insight.ModelPreferences),
		}
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, gin.H{"insights": out})
}
