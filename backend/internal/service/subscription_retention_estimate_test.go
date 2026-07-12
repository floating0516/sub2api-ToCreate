package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetentionQuotaWindowSelectsQuotaBySubscriptionDuration(t *testing.T) {
	daily, weekly, monthly := 50.0, 350.0, 1500.0
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	group := &Group{DailyLimitUSD: &daily, WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &monthly}

	tests := []struct {
		name      string
		duration  time.Duration
		wantLimit float64
		wantUsage float64
	}{
		{name: "daily card", duration: 24 * time.Hour, wantLimit: daily, wantUsage: 12},
		{name: "weekly card", duration: 7 * 24 * time.Hour, wantLimit: weekly, wantUsage: 123},
		{name: "monthly card", duration: 30 * 24 * time.Hour, wantLimit: monthly, wantUsage: 456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &UserSubscription{
				StartsAt: start, ExpiresAt: start.Add(tt.duration), Group: group,
				DailyUsageUSD: 12, WeeklyUsageUSD: 123, MonthlyUsageUSD: 456,
			}
			limit, used, ok := retentionQuotaWindow(sub)
			require.True(t, ok)
			require.Equal(t, tt.wantLimit, limit)
			require.Equal(t, tt.wantUsage, used)
		})
	}
}

func TestRetentionQuotaWindowFallsBackToAvailableLimit(t *testing.T) {
	weekly := 200.0
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	sub := &UserSubscription{
		StartsAt: start, ExpiresAt: start.Add(30 * 24 * time.Hour),
		WeeklyUsageUSD: 75,
		Group:          &Group{WeeklyLimitUSD: &weekly},
	}

	limit, used, ok := retentionQuotaWindow(sub)
	require.True(t, ok)
	require.Equal(t, weekly, limit)
	require.Equal(t, 75.0, used)
}
