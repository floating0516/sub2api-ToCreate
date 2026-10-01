//go:build unit

package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type resumeRulesRepo struct{ *subscriptionUserSubRepoStub }

func (r *resumeRulesRepo) ExtendExpiry(ctx context.Context, id int64, expiry time.Time) error {
	sub, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	sub.ExpiresAt = expiry
	return r.Update(ctx, sub)
}

func TestResumeQuotaRules(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	defer func() { _ = timezone.Init("UTC") }()
	loc := timezone.Location()
	start := time.Date(2026, 1, 31, 12, 0, 0, 0, loc)
	// Calendar dates below are independent expected results, not production helpers.
	type boundary struct {
		name        string
		start, edge time.Time
		kind        string
		one         bool
	}
	cases := []boundary{
		{"daily", start, time.Date(2026, 2, 1, 0, 0, 0, 0, loc), "daily", false},
		{"one_day_card", start, time.Date(2026, 2, 1, 0, 0, 0, 0, loc), "daily", true},
		{"weekly", start, time.Date(2026, 2, 7, 12, 0, 0, 0, loc), "weekly", false},
		{"monthly_jan31", start, time.Date(2026, 3, 2, 12, 0, 0, 0, loc), "monthly", false},
		{"monthly_feb28", time.Date(2026, 2, 28, 12, 0, 0, 0, loc), time.Date(2026, 3, 30, 12, 0, 0, 0, loc), "monthly", false},
		{"monthly_leap_feb29", time.Date(2028, 2, 29, 12, 0, 0, 0, loc), time.Date(2028, 3, 30, 12, 0, 0, 0, loc), "monthly", false},
	}
	for _, tc := range cases {
		for _, offset := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, offset), func(t *testing.T) {
				anchor := tc.start
				expiry := tc.start.Add(100 * 24 * time.Hour)
				if tc.one {
					expiry = tc.start.Add(24 * time.Hour)
				}
				sub := &UserSubscription{StartsAt: tc.start, ExpiresAt: expiry, DailyWindowStart: &anchor, WeeklyWindowStart: &anchor, MonthlyWindowStart: &anchor}
				now := tc.edge.Add(offset)
				var got bool
				switch tc.kind {
				case "daily":
					got = sub.NeedsDailyResetAt(now)
				case "weekly":
					got = sub.canAutomaticallyResetWeeklyAt(now)
				case "monthly":
					got = sub.canAutomaticallyResetMonthlyAt(now)
				}
				require.Equal(t, offset >= 0 && !tc.one, got)
			})
		}
	}
	t.Run("delayed_first_use", func(t *testing.T) {
		repo := &activateWindowUserSubRepo{}
		svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
		svc.now = func() time.Time { return time.Date(2026, 2, 3, 19, 0, 0, 0, loc) }
		sub := &UserSubscription{ID: 1, StartsAt: start, ExpiresAt: start.Add(90 * 24 * time.Hour)}
		require.NoError(t, svc.CheckAndActivateWindow(context.Background(), sub))
		require.Equal(t, start, repo.periodicStart)
	})
	t.Run("skip_multiple_periods", func(t *testing.T) {
		sub := &UserSubscription{StartsAt: start, ExpiresAt: start.Add(180 * 24 * time.Hour)}
		got, ok := sub.automaticWindowStartAt(&start, 30*24*time.Hour, time.Date(2026, 4, 7, 12, 0, 0, 0, loc))
		require.True(t, ok)
		require.Equal(t, time.Date(2026, 4, 1, 12, 0, 0, 0, loc), got)
	})
	for _, offset := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
		t.Run(fmt.Sprintf("expiry/%d", offset), func(t *testing.T) {
			expiry := time.Date(2026, 3, 2, 12, 0, 0, 0, loc)
			sub := &UserSubscription{Status: SubscriptionStatusActive, StartsAt: start, ExpiresAt: expiry, MonthlyWindowStart: &start}
			svc := NewSubscriptionService(groupRepoNoop{}, &monthlyResetUserSubRepo{}, nil, nil, nil)
			svc.now = func() time.Time { return expiry.Add(offset) }
			_, err := svc.ValidateAndCheckLimits(sub, &Group{})
			if offset < 0 {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrSubscriptionExpired)
			}
		})
	}
	for _, offset := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
		t.Run(fmt.Sprintf("renewal/%d", offset), func(t *testing.T) {
			expiry := time.Date(2026, 3, 2, 12, 0, 0, 0, loc)
			now := expiry.Add(offset)
			anchor := start
			repo := &resumeRulesRepo{newSubscriptionUserSubRepoStub()}
			repo.seed(&UserSubscription{ID: 99, UserID: 7, GroupID: 1, Status: SubscriptionStatusActive, StartsAt: start, ExpiresAt: expiry, DailyWindowStart: &anchor, WeeklyWindowStart: &anchor, MonthlyWindowStart: &anchor, DailyUsageUSD: 3, WeeklyUsageUSD: 4, MonthlyUsageUSD: 5})
			svc := NewSubscriptionService(&subscriptionGroupRepoStub{group: &Group{ID: 1, SubscriptionType: SubscriptionTypeSubscription}}, repo, nil, nil, nil)
			svc.now = func() time.Time { return now }
			got, _, err := svc.AssignOrExtendSubscription(context.Background(), &AssignSubscriptionInput{UserID: 7, GroupID: 1, ValidityDays: 7})
			require.NoError(t, err)
			if offset < 0 {
				require.Equal(t, start, got.StartsAt)
				require.Equal(t, expiry.Add(7*24*time.Hour), got.ExpiresAt)
				require.Equal(t, 5.0, got.MonthlyUsageUSD)
				require.Equal(t, start, *got.MonthlyWindowStart)
			} else {
				require.Equal(t, now, got.StartsAt)
				require.Equal(t, now.Add(7*24*time.Hour), got.ExpiresAt)
				require.Zero(t, got.MonthlyUsageUSD)
				require.Equal(t, now, *got.MonthlyWindowStart)
			}
		})
	}
	for _, offset := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
		t.Run(fmt.Sprintf("addon_expiry/%d", offset), func(t *testing.T) {
			expiry := start.Add(24 * time.Hour)
			pack := &SubscriptionAddonPack{Status: SubscriptionAddonStatusActive, StartsAt: start, ExpiresAt: expiry, QuotaUSD: 10, UsedUSD: 3}
			require.Equal(t, offset < 0, pack.IsUsableAt(expiry.Add(offset)))
		})
	}
}
