package handler

import (
	"math"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SubscriptionSummaryItem represents a subscription item in summary
type SubscriptionSummaryItem struct {
	ID              int64   `json:"id"`
	GroupID         int64   `json:"group_id"`
	GroupName       string  `json:"group_name"`
	Status          string  `json:"status"`
	DailyUsedUSD    float64 `json:"daily_used_usd,omitempty"`
	DailyLimitUSD   float64 `json:"daily_limit_usd,omitempty"`
	WeeklyUsedUSD   float64 `json:"weekly_used_usd,omitempty"`
	WeeklyLimitUSD  float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyUsedUSD  float64 `json:"monthly_used_usd,omitempty"`
	MonthlyLimitUSD float64 `json:"monthly_limit_usd,omitempty"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
}

// SubscriptionProgressInfo represents subscription with progress info
type SubscriptionProgressInfo struct {
	Subscription *dto.UserSubscription         `json:"subscription"`
	Progress     *service.SubscriptionProgress `json:"progress"`
}

// SubscriptionHandler handles user subscription operations
type SubscriptionHandler struct {
	subscriptionService *service.SubscriptionService
	usageService        *service.UsageService
}

// NewSubscriptionHandler creates a new user subscription handler
func NewSubscriptionHandler(subscriptionService *service.SubscriptionService, usageService ...*service.UsageService) *SubscriptionHandler {
	var usageSvc *service.UsageService
	if len(usageService) > 0 {
		usageSvc = usageService[0]
	}
	return &SubscriptionHandler{
		subscriptionService: subscriptionService,
		usageService:        usageSvc,
	}
}

// List handles listing current user's subscriptions
// GET /api/v1/subscriptions
func (h *SubscriptionHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	subscriptions, err := h.subscriptionService.ListUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserSubscription, 0, len(subscriptions))
	for i := range subscriptions {
		out = append(out, *dto.UserSubscriptionFromService(&subscriptions[i]))
	}
	response.Success(c, out)
}

// GetActive handles getting current user's active subscriptions
// GET /api/v1/subscriptions/active
func (h *SubscriptionHandler) GetActive(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UserSubscription, 0, len(subscriptions))
	for i := range subscriptions {
		out = append(out, *dto.UserSubscriptionFromService(&subscriptions[i]))
	}
	response.Success(c, out)
}

// GetProgress handles getting subscription progress for current user
// GET /api/v1/subscriptions/progress
func (h *SubscriptionHandler) GetProgress(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	// Get all active subscriptions with progress
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	result := make([]SubscriptionProgressInfo, 0, len(subscriptions))
	for i := range subscriptions {
		sub := &subscriptions[i]
		progress, err := h.subscriptionService.GetSubscriptionProgress(c.Request.Context(), sub.ID)
		if err != nil {
			// Skip subscriptions with errors
			continue
		}
		result = append(result, SubscriptionProgressInfo{
			Subscription: dto.UserSubscriptionFromService(sub),
			Progress:     progress,
		})
	}

	response.Success(c, result)
}

// GetSummary handles getting a summary of current user's subscription status
// GET /api/v1/subscriptions/summary
func (h *SubscriptionHandler) GetSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	// Get all active subscriptions
	subscriptions, err := h.subscriptionService.ListActiveUserSubscriptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var totalUsed float64
	items := make([]SubscriptionSummaryItem, 0, len(subscriptions))

	for _, sub := range subscriptions {
		item := SubscriptionSummaryItem{
			ID:             sub.ID,
			GroupID:        sub.GroupID,
			Status:         sub.Status,
			DailyUsedUSD:   sub.DailyUsageUSD,
			WeeklyUsedUSD:  sub.WeeklyUsageUSD,
			MonthlyUsedUSD: sub.MonthlyUsageUSD,
		}

		// Add group info if preloaded
		if sub.Group != nil {
			item.GroupName = sub.Group.Name
			if sub.Group.DailyLimitUSD != nil {
				item.DailyLimitUSD = *sub.Group.DailyLimitUSD
			}
			if sub.Group.WeeklyLimitUSD != nil {
				item.WeeklyLimitUSD = *sub.Group.WeeklyLimitUSD
			}
			if sub.Group.MonthlyLimitUSD != nil {
				item.MonthlyLimitUSD = *sub.Group.MonthlyLimitUSD
			}
		}

		// Format expiration time
		if !sub.ExpiresAt.IsZero() {
			formatted := sub.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
			item.ExpiresAt = &formatted
		}

		// Track total usage (use monthly as the most comprehensive)
		totalUsed += sub.MonthlyUsageUSD

		items = append(items, item)
	}

	summary := struct {
		ActiveCount   int                       `json:"active_count"`
		TotalUsedUSD  float64                   `json:"total_used_usd"`
		Subscriptions []SubscriptionSummaryItem `json:"subscriptions"`
	}{
		ActiveCount:   len(subscriptions),
		TotalUsedUSD:  totalUsed,
		Subscriptions: items,
	}

	response.Success(c, summary)
}

// GetUsageTimeline returns a fixed-bucket usage timeline for one owned subscription.
// GET /api/v1/subscriptions/:id/usage-timeline?window=daily|weekly|monthly
func (h *SubscriptionHandler) GetUsageTimeline(c *gin.Context) {
	if h.usageService == nil {
		response.InternalError(c, "Usage service not available")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	subscriptionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid subscription ID")
		return
	}

	sub, err := h.subscriptionService.GetByID(c.Request.Context(), subscriptionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if sub.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to access this subscription")
		return
	}

	window := c.DefaultQuery("window", "daily")
	start, end, bucketSeconds, bucketCount, ok := subscriptionTimelineWindow(sub, window, time.Now())
	if !ok {
		response.BadRequest(c, "Invalid or unavailable subscription usage window")
		return
	}

	timeline, err := h.usageService.GetSubscriptionUsageTimeline(c.Request.Context(), subscriptionID, start, end, bucketSeconds, bucketCount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	timeline.Window = window
	response.Success(c, timeline)
}

func subscriptionTimelineWindow(sub *service.UserSubscription, window string, now time.Time) (time.Time, time.Time, int64, int, bool) {
	if sub == nil || sub.Group == nil || sub.StartsAt.IsZero() {
		return time.Time{}, time.Time{}, 0, 0, false
	}

	switch window {
	case "daily":
		if !sub.Group.HasDailyLimit() {
			return time.Time{}, time.Time{}, 0, 0, false
		}
		start := rollingTimelineWindowStart(sub.StartsAt, now, 24*time.Hour)
		if sub.DailyWindowStart != nil && !sub.NeedsDailyResetAt(now) {
			start = *sub.DailyWindowStart
		}
		end := start.Add(24 * time.Hour)
		if sub.HasOneTimeDailyQuota() {
			start = sub.StartsAt
			end = sub.ExpiresAt
		}
		return normalizeTimelineRange(start, end, time.Hour, 24)
	case "weekly":
		if !sub.Group.HasWeeklyLimit() {
			return time.Time{}, time.Time{}, 0, 0, false
		}
		start := rollingTimelineWindowStart(sub.StartsAt, now, 7*24*time.Hour)
		if sub.WeeklyWindowStart != nil && !sub.NeedsWeeklyResetAt(now) {
			start = *sub.WeeklyWindowStart
		}
		return normalizeTimelineRange(start, start.Add(7*24*time.Hour), 24*time.Hour, 7)
	case "monthly":
		if !sub.Group.HasMonthlyLimit() {
			return time.Time{}, time.Time{}, 0, 0, false
		}
		start := rollingTimelineWindowStart(sub.StartsAt, now, 30*24*time.Hour)
		if sub.MonthlyWindowStart != nil && !sub.NeedsMonthlyResetAt(now) {
			start = *sub.MonthlyWindowStart
		}
		return normalizeTimelineRange(start, start.Add(30*24*time.Hour), 24*time.Hour, 30)
	default:
		return time.Time{}, time.Time{}, 0, 0, false
	}
}

func rollingTimelineWindowStart(anchor, now time.Time, period time.Duration) time.Time {
	if anchor.IsZero() || period <= 0 || now.Before(anchor) {
		return anchor
	}
	elapsed := now.Sub(anchor) / period
	return anchor.Add(elapsed * period)
}

func normalizeTimelineRange(start, end time.Time, bucketSize time.Duration, maxBuckets int) (time.Time, time.Time, int64, int, bool) {
	if start.IsZero() || !end.After(start) || bucketSize <= 0 || maxBuckets <= 0 {
		return time.Time{}, time.Time{}, 0, 0, false
	}
	bucketCount := int(math.Ceil(float64(end.Sub(start)) / float64(bucketSize)))
	if bucketCount < 1 {
		bucketCount = 1
	}
	if bucketCount > maxBuckets {
		bucketCount = maxBuckets
	}
	return start, end, int64(bucketSize / time.Second), bucketCount, true
}
