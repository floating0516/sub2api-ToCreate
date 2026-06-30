package handler

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
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	limit := parseLeaderboardLimit(c.DefaultQuery("limit", "20"))
	snapshot, err := h.leaderboardService.GetUserSnapshot(
		c.Request.Context(),
		subject.UserID,
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

func (h *LeaderboardHandler) UpdatePrivacy(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		Anonymous bool `json:"anonymous"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	pref, err := h.leaderboardService.SetPreference(c.Request.Context(), subject.UserID, req.Anonymous)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pref)
}

func parseLeaderboardLimit(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 20
	}
	if n <= 0 {
		return 20
	}
	if n > 100 {
		return 100
	}
	return n
}
