package handler

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRedeemCodeFromPromoUsage(t *testing.T) {
	usedAt := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	expiresAt := usedAt.Add(24 * time.Hour)

	result := redeemCodeFromPromoUsage(&service.PromoCodeUsage{
		ID:          7,
		PromoCodeID: 3,
		UserID:      42,
		BonusAmount: 8.5,
		UsedAt:      usedAt,
		PromoCode: &service.PromoCode{
			ID:        3,
			Code:      "SUMMER2026",
			ExpiresAt: &expiresAt,
		},
	})

	require.Equal(t, int64(-7), result.ID)
	require.Equal(t, "SUMMER2026", result.Code)
	require.Equal(t, service.RedeemTypeBalance, result.Type)
	require.Equal(t, service.StatusUsed, result.Status)
	require.Equal(t, 8.5, result.Value)
	require.Equal(t, int64(42), *result.UsedBy)
	require.Equal(t, usedAt, *result.UsedAt)
	require.Equal(t, expiresAt, *result.ExpiresAt)
}
