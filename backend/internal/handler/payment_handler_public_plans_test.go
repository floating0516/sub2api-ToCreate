package handler

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestPublicCatalogPlansKeepsMarketingFields(t *testing.T) {
	original := 48.16
	plans := []*dbent.SubscriptionPlan{
		{
			ID:            22,
			GroupID:       99,
			Name:          "GPT Pro 标准周卡",
			Description:   "推荐档，适合一周连续开发。",
			Price:         39.90,
			OriginalPrice: &original,
			Currency:      "",
			ValidityDays:  7,
			ValidityUnit:  "days",
			Features:      "7 天有效\n\n支持 GPT Pro\n支付宝按人民币结算",
			ProductName:   "internal",
			ForSale:       true,
			SortOrder:     30,
		},
		nil,
	}

	got := publicCatalogPlans(plans)
	require.Len(t, got, 1)
	require.Equal(t, int64(22), got[0].ID)
	require.Equal(t, "GPT Pro 标准周卡", got[0].Name)
	require.Equal(t, 39.90, got[0].Price)
	require.Equal(t, &original, got[0].OriginalPrice)
	require.Equal(t, 7, got[0].ValidityDays)
	require.Equal(t, []string{"7 天有效", "支持 GPT Pro", "支付宝按人民币结算"}, got[0].Features)
	require.Equal(t, 30, got[0].SortOrder)
}
