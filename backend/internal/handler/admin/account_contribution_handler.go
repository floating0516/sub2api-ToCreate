package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetContributionOverview returns the read-only account contribution dashboard.
// GET /api/v1/admin/account-contributions/overview
func (h *AccountHandler) GetContributionOverview(c *gin.Context) {
	overview, err := h.adminService.GetAccountContributionOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overview)
}
