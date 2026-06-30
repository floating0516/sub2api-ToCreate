package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	leaderboardService *service.LeaderboardService
}

func NewLeaderboardHandler(leaderboardService *service.LeaderboardService) *LeaderboardHandler {
	return &LeaderboardHandler{leaderboardService: leaderboardService}
}

func (h *LeaderboardHandler) Get(c *gin.Context) {
	limit := parseLeaderboardLimit(c.DefaultQuery("limit", "50"))
	snapshot, err := h.leaderboardService.GetAdminSnapshot(
		c.Request.Context(),
		c.DefaultQuery("period", service.LeaderboardPeriodWeek),
		limit,
		timezone.Now(),
		timezone.Location(),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, snapshot)
}

func (h *LeaderboardHandler) GetSettings(c *gin.Context) {
	settings, err := h.leaderboardService.GetRewardSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *LeaderboardHandler) UpdateSettings(c *gin.Context) {
	var req service.LeaderboardRewardSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	settings, err := h.leaderboardService.UpdateRewardSettings(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *LeaderboardHandler) GenerateReward(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		Period string `json:"period"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	result, err := h.leaderboardService.GenerateReward(c.Request.Context(), service.GenerateLeaderboardRewardInput{
		Period:    req.Period,
		CreatedBy: subject.UserID,
	}, timezone.Now(), timezone.Location())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseLeaderboardLimit(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 50
	}
	if n <= 0 {
		return 50
	}
	if n > 100 {
		return 100
	}
	return n
}
