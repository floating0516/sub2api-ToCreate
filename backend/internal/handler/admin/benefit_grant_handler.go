package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BenefitGrantHandler struct {
	service *service.BenefitGrantService
}

func NewBenefitGrantHandler(benefitGrantService *service.BenefitGrantService) *BenefitGrantHandler {
	return &BenefitGrantHandler{service: benefitGrantService}
}

type BenefitGrantRequest struct {
	OperationKey   string  `json:"operation_key" binding:"required,min=8,max=128"`
	AudienceType   string  `json:"audience_type" binding:"required,oneof=today_active recent_active recent_registered"`
	AudienceDate   string  `json:"audience_date" binding:"omitempty,max=10"`
	AudienceDays   int     `json:"audience_days" binding:"required,min=1,max=365"`
	Timezone       string  `json:"timezone" binding:"omitempty,max=64"`
	BenefitType    string  `json:"benefit_type" binding:"required,oneof=subscription balance"`
	ConflictPolicy string  `json:"conflict_policy" binding:"omitempty,oneof=skip_active extend_active none"`
	GroupID        int64   `json:"group_id" binding:"omitempty,gte=0"`
	ValidityDays   int     `json:"validity_days" binding:"omitempty,gte=0,lte=36500"`
	BalanceAmount  float64 `json:"balance_amount" binding:"omitempty,gte=0,lte=1000000"`
	Notes          string  `json:"notes" binding:"max=1800"`
}

type ExecuteBenefitGrantRequest struct {
	BenefitGrantRequest
	ExpectedMatchedCount  *int   `json:"expected_matched_count" binding:"required,min=0"`
	ExpectedEligibleCount *int   `json:"expected_eligible_count" binding:"required,min=0"`
	ExpectedSnapshot      string `json:"expected_snapshot" binding:"required,len=64"`
}

// Preview resolves the exact audience and benefit eligibility without writing.
// POST /api/v1/admin/benefit-grants/preview
func (h *BenefitGrantHandler) Preview(c *gin.Context) {
	middleware2.SkipAudit(c)

	var req BenefitGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	preview, err := h.service.Preview(c.Request.Context(), benefitGrantInputFromRequest(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, preview)
}

// Execute snapshots the previewed audience and executes a durable grant batch.
// POST /api/v1/admin/benefit-grants/execute
func (h *BenefitGrantHandler) Execute(c *gin.Context) {
	middleware2.SetAuditAction(c, "admin.benefit_grants.execute")

	var req ExecuteBenefitGrantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if headerKey := strings.TrimSpace(c.GetHeader("Idempotency-Key")); headerKey != "" &&
		headerKey != strings.TrimSpace(req.OperationKey) {
		response.BadRequest(c, "Idempotency-Key must match operation_key")
		return
	}

	adminID := getAdminIDFromContext(c)
	executeAdminIdempotentJSON(
		c,
		"admin.benefit_grants.execute",
		req,
		24*time.Hour,
		func(ctx context.Context) (any, error) {
			input := benefitGrantInputFromRequest(req.BenefitGrantRequest)
			return h.service.Execute(ctx, &service.ExecuteBenefitGrantInput{
				BenefitGrantInput:     *input,
				ExpectedMatchedCount:  *req.ExpectedMatchedCount,
				ExpectedEligibleCount: *req.ExpectedEligibleCount,
				ExpectedSnapshot:      req.ExpectedSnapshot,
				AssignedBy:            adminID,
			})
		},
	)
}

// List returns grant campaign history.
// GET /api/v1/admin/benefit-grants
func (h *BenefitGrantHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	campaigns, total, err := h.service.ListCampaigns(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, campaigns, total, page, pageSize)
}

// Get returns one grant campaign.
// GET /api/v1/admin/benefit-grants/:id
func (h *BenefitGrantHandler) Get(c *gin.Context) {
	campaignID, ok := parseBenefitGrantCampaignID(c)
	if !ok {
		return
	}
	campaign, err := h.service.GetCampaign(c.Request.Context(), campaignID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, campaign)
}

// ListRecipients returns the immutable audience snapshot and per-user results.
// GET /api/v1/admin/benefit-grants/:id/recipients
func (h *BenefitGrantHandler) ListRecipients(c *gin.Context) {
	campaignID, ok := parseBenefitGrantCampaignID(c)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	switch status {
	case "",
		service.BenefitGrantRecipientPending,
		service.BenefitGrantRecipientProcessing,
		service.BenefitGrantRecipientGranted,
		service.BenefitGrantRecipientSkipped,
		service.BenefitGrantRecipientFailed:
	default:
		response.BadRequest(c, "Invalid recipient status")
		return
	}

	page, pageSize := response.ParsePagination(c)
	recipients, total, err := h.service.ListRecipients(
		c.Request.Context(),
		campaignID,
		page,
		pageSize,
		status,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, recipients, total, page, pageSize)
}

// Retry replays only failed, pending, or stale in-progress recipients from the
// original snapshot. It never resolves a new audience.
// POST /api/v1/admin/benefit-grants/:id/retry
func (h *BenefitGrantHandler) Retry(c *gin.Context) {
	campaignID, ok := parseBenefitGrantCampaignID(c)
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.benefit_grants.retry")

	payload := struct {
		CampaignID int64 `json:"campaign_id"`
	}{CampaignID: campaignID}
	executeAdminIdempotentJSON(
		c,
		"admin.benefit_grants.retry",
		payload,
		24*time.Hour,
		func(ctx context.Context) (any, error) {
			return h.service.Retry(ctx, campaignID)
		},
	)
}

func benefitGrantInputFromRequest(req BenefitGrantRequest) *service.BenefitGrantInput {
	return &service.BenefitGrantInput{
		OperationKey:   req.OperationKey,
		AudienceType:   req.AudienceType,
		AudienceDate:   req.AudienceDate,
		AudienceDays:   req.AudienceDays,
		Timezone:       req.Timezone,
		BenefitType:    req.BenefitType,
		ConflictPolicy: req.ConflictPolicy,
		GroupID:        req.GroupID,
		ValidityDays:   req.ValidityDays,
		BalanceAmount:  req.BalanceAmount,
		Notes:          req.Notes,
	}
}

func parseBenefitGrantCampaignID(c *gin.Context) (int64, bool) {
	campaignID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || campaignID <= 0 {
		response.BadRequest(c, "Invalid benefit grant campaign ID")
		return 0, false
	}
	return campaignID, true
}
