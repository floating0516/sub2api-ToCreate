package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserFromServiceAdmin_MapsActivityTimestamps(t *testing.T) {
	t.Parallel()

	lastLoginAt := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	lastActiveAt := lastLoginAt.Add(15 * time.Minute)
	lastUsedAt := lastLoginAt.Add(45 * time.Minute)
	lastUsageAt := lastLoginAt.Add(50 * time.Minute)

	out := UserFromServiceAdmin(&service.User{
		ID:           42,
		Email:        "admin@example.com",
		Username:     "admin",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		LastActiveAt: &lastActiveAt,
		LastUsedAt:   &lastUsedAt,
		UsageSummary: &service.UserUsageSummary{
			TotalRequests:   100,
			TodayRequests:   5,
			Requests7D:      30,
			Requests30D:     80,
			ActiveDays30D:   12,
			TotalTokens:     9000,
			Tokens30D:       7000,
			TotalCost:       3.5,
			Cost30D:         2.8,
			TotalActualCost: 2.4,
			ActualCost30D:   1.9,
			LastUsageAt:     &lastUsageAt,
		},
		ModelPreferences: []service.UserModelPreference{
			{
				Model:      "claude-sonnet-4-5",
				Requests:   24,
				Tokens:     6000,
				Cost:       1.8,
				ActualCost: 1.2,
				Share:      0.75,
			},
		},
	})

	require.NotNil(t, out)
	require.NotNil(t, out.LastActiveAt)
	require.NotNil(t, out.LastUsedAt)
	require.WithinDuration(t, lastActiveAt, *out.LastActiveAt, time.Second)
	require.WithinDuration(t, lastUsedAt, *out.LastUsedAt, time.Second)
	require.NotNil(t, out.UsageSummary)
	require.Equal(t, int64(100), out.UsageSummary.TotalRequests)
	require.Equal(t, int64(5), out.UsageSummary.TodayRequests)
	require.Equal(t, int64(30), out.UsageSummary.Requests7D)
	require.Equal(t, int64(80), out.UsageSummary.Requests30D)
	require.Equal(t, int64(12), out.UsageSummary.ActiveDays30D)
	require.Equal(t, int64(9000), out.UsageSummary.TotalTokens)
	require.Equal(t, int64(7000), out.UsageSummary.Tokens30D)
	require.Equal(t, 3.5, out.UsageSummary.TotalCost)
	require.Equal(t, 2.8, out.UsageSummary.Cost30D)
	require.Equal(t, 2.4, out.UsageSummary.TotalActualCost)
	require.Equal(t, 1.9, out.UsageSummary.ActualCost30D)
	require.NotNil(t, out.UsageSummary.LastUsageAt)
	require.WithinDuration(t, lastUsageAt, *out.UsageSummary.LastUsageAt, time.Second)
	require.Len(t, out.ModelPreferences, 1)
	require.Equal(t, "claude-sonnet-4-5", out.ModelPreferences[0].Model)
	require.Equal(t, int64(24), out.ModelPreferences[0].Requests)
	require.Equal(t, int64(6000), out.ModelPreferences[0].Tokens)
	require.Equal(t, 1.8, out.ModelPreferences[0].Cost)
	require.Equal(t, 1.2, out.ModelPreferences[0].ActualCost)
	require.Equal(t, 0.75, out.ModelPreferences[0].Share)
}
