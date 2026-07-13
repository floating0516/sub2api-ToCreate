package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type subscriptionAddonRepoStub struct {
	usable  *SubscriptionAddonPack
	getErr  error
	created *SubscriptionAddonPack
}

func (s *subscriptionAddonRepoStub) Create(_ context.Context, pack *SubscriptionAddonPack) error {
	copy := *pack
	copy.ID = 99
	s.created = &copy
	pack.ID = copy.ID
	return nil
}

func (s *subscriptionAddonRepoStub) GetByID(context.Context, int64) (*SubscriptionAddonPack, error) {
	return nil, ErrSubscriptionAddonNotFound
}

func (s *subscriptionAddonRepoStub) GetUsableForSubscription(context.Context, int64, time.Time) (*SubscriptionAddonPack, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.usable == nil {
		return nil, ErrSubscriptionAddonNotFound
	}
	copy := *s.usable
	return &copy, nil
}

func (s *subscriptionAddonRepoStub) ListBySubscriptionID(context.Context, int64) ([]SubscriptionAddonPack, error) {
	return nil, nil
}

func (s *subscriptionAddonRepoStub) GetActiveSummaries(context.Context, []int64, time.Time) (map[int64]SubscriptionAddonSummary, error) {
	return map[int64]SubscriptionAddonSummary{}, nil
}

func (s *subscriptionAddonRepoStub) GetCurrentTermQuotaTotals(context.Context, []int64, time.Time) (map[int64]SubscriptionAddonQuotaTotal, error) {
	return map[int64]SubscriptionAddonQuotaTotal{}, nil
}

func (s *subscriptionAddonRepoStub) GetGrantedQuotaForTerm(context.Context, int64, time.Time, time.Time) (float64, error) {
	return 0, nil
}

func (s *subscriptionAddonRepoStub) Revoke(context.Context, int64, time.Time) error { return nil }

type addonGrantSubscriptionRepo struct {
	userSubRepoNoop
	sub *UserSubscription
}

func (r addonGrantSubscriptionRepo) GetByID(context.Context, int64) (*UserSubscription, error) {
	copy := *r.sub
	return &copy, nil
}

func TestResolveUsageAccessFallsBackToAddon(t *testing.T) {
	now := time.Now()
	limit := 10.0
	repo := &subscriptionAddonRepoStub{usable: &SubscriptionAddonPack{
		ID:        7,
		QuotaUSD:  20,
		UsedUSD:   3,
		StartsAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(time.Hour),
		Status:    SubscriptionAddonStatusActive,
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, nil, nil, nil)
	svc.addonRepo = repo
	sub := &UserSubscription{
		ID:               1,
		Status:           SubscriptionStatusActive,
		ExpiresAt:        now.Add(time.Hour),
		DailyWindowStart: &now,
		DailyUsageUSD:    limit,
	}

	resolved, err := svc.ResolveUsageAccess(context.Background(), sub, &Group{DailyLimitUSD: &limit})
	require.NoError(t, err)
	require.NotNil(t, resolved.ActiveAddon)
	require.Equal(t, int64(7), resolved.ActiveAddon.ID)
}

func TestResolveUsageAccessKeepsQuotaErrorWithoutAddon(t *testing.T) {
	now := time.Now()
	limit := 10.0
	svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, nil, nil, nil)
	svc.addonRepo = &subscriptionAddonRepoStub{}
	sub := &UserSubscription{
		ID:               1,
		Status:           SubscriptionStatusActive,
		ExpiresAt:        now.Add(time.Hour),
		DailyWindowStart: &now,
		DailyUsageUSD:    limit,
	}

	_, err := svc.ResolveUsageAccess(context.Background(), sub, &Group{DailyLimitUSD: &limit})
	require.ErrorIs(t, err, ErrDailyLimitExceeded)
}

func TestBillingCacheEligibilitySelectsAddonWhenCachedQuotaIsExhausted(t *testing.T) {
	now := time.Now()
	limit := 10.0
	repo := &subscriptionAddonRepoStub{usable: &SubscriptionAddonPack{
		ID:        8,
		QuotaUSD:  20,
		StartsAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(time.Hour),
		Status:    SubscriptionAddonStatusActive,
	}}
	sub := &UserSubscription{ID: 3, ExpiresAt: now.Add(time.Hour)}
	svc := &BillingCacheService{
		cache: &subscriptionQuotaCacheStub{data: &SubscriptionCacheData{
			Status:     SubscriptionStatusActive,
			ExpiresAt:  sub.ExpiresAt,
			DailyUsage: limit,
		}},
		addonRepo: repo,
	}

	err := svc.checkSubscriptionEligibility(context.Background(), 2, &Group{ID: 4, DailyLimitUSD: &limit}, sub)
	require.NoError(t, err)
	require.NotNil(t, sub.ActiveAddon)
	require.Equal(t, int64(8), sub.ActiveAddon.ID)
}

func TestGrantAddonDefaultsToSubscriptionExpiry(t *testing.T) {
	now := time.Now()
	sub := &UserSubscription{
		ID:        12,
		UserID:    34,
		GroupID:   56,
		Status:    SubscriptionStatusActive,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	addonRepo := &subscriptionAddonRepoStub{}
	svc := NewSubscriptionService(groupRepoNoop{}, addonGrantSubscriptionRepo{sub: sub}, nil, nil, nil)
	svc.addonRepo = addonRepo

	pack, err := svc.GrantAddon(context.Background(), &GrantSubscriptionAddonInput{
		SubscriptionID: sub.ID,
		QuotaUSD:       20,
		AssignedBy:     9,
		Notes:          "manual grant",
	})
	require.NoError(t, err)
	require.Equal(t, int64(99), pack.ID)
	require.Equal(t, sub.ExpiresAt, pack.ExpiresAt)
	require.Equal(t, int64(9), *pack.AssignedBy)
	require.Equal(t, "manual grant", addonRepo.created.Notes)
}
