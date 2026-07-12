package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionTermQuotaCapacityCombinesDailyWeeklyAndMonthlyLimits(t *testing.T) {
	daily, weekly, monthly := 50.0, 350.0, 1500.0
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	group := &Group{DailyLimitUSD: &daily, WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &monthly}

	tests := []struct {
		name      string
		duration  time.Duration
		wantLimit float64
	}{
		{name: "daily card", duration: 24 * time.Hour, wantLimit: 50},
		{name: "weekly card", duration: 7 * 24 * time.Hour, wantLimit: 350},
		{name: "eight day standard card", duration: 8 * 24 * time.Hour, wantLimit: 400},
		{name: "monthly card", duration: 30 * 24 * time.Hour, wantLimit: 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &UserSubscription{StartsAt: start, ExpiresAt: start.Add(tt.duration), Group: group}
			limit, ok := subscriptionTermQuotaCapacity(sub)
			require.True(t, ok)
			require.Equal(t, tt.wantLimit, limit)
		})
	}
}

func TestQuotaCapacityForEightDayLightCard(t *testing.T) {
	daily, weekly, monthly := 50.0, 150.0, 600.0
	limit, ok := quotaCapacityForDays(8, &daily, &weekly, &monthly)
	require.True(t, ok)
	require.Equal(t, 200.0, limit)
}

func TestQuotaCapacityForThirtyDayLightCardUsesMonthlyCap(t *testing.T) {
	daily, weekly, monthly := 50.0, 150.0, 600.0
	limit, ok := quotaCapacityForDays(30, &daily, &weekly, &monthly)
	require.True(t, ok)
	require.Equal(t, 600.0, limit)
}

func TestCurrentSubscriptionUsageFloorUsesLargestLiveWindow(t *testing.T) {
	sub := &UserSubscription{DailyUsageUSD: 12, WeeklyUsageUSD: 75, MonthlyUsageUSD: 63}
	require.Equal(t, 75.0, currentSubscriptionUsageFloor(sub))
}

func TestCalculateSubscriptionQuotaOutcomePreservesOverage(t *testing.T) {
	used, unused, overage := calculateSubscriptionQuotaOutcome(150, 240)
	require.Equal(t, 240.0, used)
	require.Equal(t, 0.0, unused)
	require.Equal(t, 90.0, overage)
}
