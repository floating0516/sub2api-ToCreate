package handler

import (
	"errors"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RedeemHandler handles redeem code-related requests
type RedeemHandler struct {
	redeemService *service.RedeemService
	promoService  *service.PromoService
}

// NewRedeemHandler creates a new RedeemHandler
func NewRedeemHandler(redeemService *service.RedeemService, promoService *service.PromoService) *RedeemHandler {
	return &RedeemHandler{
		redeemService: redeemService,
		promoService:  promoService,
	}
}

// RedeemRequest represents the redeem code request payload
type RedeemRequest struct {
	Code string `json:"code" binding:"required"`
}

// RedeemResponse represents the redeem response
type RedeemResponse struct {
	Message        string   `json:"message"`
	Type           string   `json:"type"`
	Value          float64  `json:"value"`
	NewBalance     *float64 `json:"new_balance,omitempty"`
	NewConcurrency *int     `json:"new_concurrency,omitempty"`
}

// Redeem handles redeeming a code
// POST /api/v1/redeem
func (h *RedeemHandler) Redeem(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req RedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	_, lookupErr := h.redeemService.GetByCode(ctx, req.Code)
	if lookupErr == nil {
		result, err := h.redeemService.Redeem(ctx, subject.UserID, req.Code)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, dto.RedeemCodeFromService(result))
		return
	}
	if !errors.Is(lookupErr, service.ErrRedeemCodeNotFound) {
		response.ErrorFrom(c, lookupErr)
		return
	}

	if h.promoService != nil {
		usage, err := h.promoService.Redeem(ctx, subject.UserID, req.Code)
		if err == nil {
			response.Success(c, dto.RedeemCodeFromService(redeemCodeFromPromoUsage(usage)))
			return
		}
		if !errors.Is(err, service.ErrPromoCodeNotFound) {
			response.ErrorFrom(c, err)
			return
		}
	}

	// Preserve the existing unknown-code rate limit and error contract.
	result, err := h.redeemService.Redeem(ctx, subject.UserID, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.RedeemCodeFromService(result))
}

// GetHistory returns the user's redemption history
// GET /api/v1/redeem/history
func (h *RedeemHandler) GetHistory(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Default limit is 25
	limit := 25

	ctx := c.Request.Context()
	codes, err := h.redeemService.GetUserHistory(ctx, subject.UserID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.RedeemCode, 0, len(codes)+limit)
	for i := range codes {
		out = append(out, *dto.RedeemCodeFromService(&codes[i]))
	}
	if h.promoService != nil {
		usages, usageErr := h.promoService.ListUserUsages(ctx, subject.UserID, limit)
		if usageErr != nil {
			response.ErrorFrom(c, usageErr)
			return
		}
		for i := range usages {
			out = append(out, *dto.RedeemCodeFromService(redeemCodeFromPromoUsage(&usages[i])))
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UsedAt == nil {
			return false
		}
		if out[j].UsedAt == nil {
			return true
		}
		return out[i].UsedAt.After(*out[j].UsedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	response.Success(c, out)
}

func redeemCodeFromPromoUsage(usage *service.PromoCodeUsage) *service.RedeemCode {
	if usage == nil {
		return nil
	}

	usedBy := usage.UserID
	usedAt := usage.UsedAt
	code := ""
	var expiresAt *time.Time
	if usage.PromoCode != nil {
		code = usage.PromoCode.Code
		expiresAt = usage.PromoCode.ExpiresAt
	}

	return &service.RedeemCode{
		ID:        -usage.ID,
		Code:      code,
		Type:      service.RedeemTypeBalance,
		Value:     usage.BonusAmount,
		Status:    service.StatusUsed,
		UsedBy:    &usedBy,
		UsedAt:    &usedAt,
		CreatedAt: usage.UsedAt,
		ExpiresAt: expiresAt,
	}
}
