package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

type subscriptionAddonRepoStub struct {
	usable           *SubscriptionAddonPack
	getErr           error
	created          *SubscriptionAddonPack
	products         map[int64]*SubscriptionAddonProduct
	productErr       error
	listForSaleOnly  *bool
	updatedProduct   *UpdateSubscriptionAddonProductInput
	purchasedByOrder map[int64]*SubscriptionAddonPack
	purchaseCalls    int
	purchaseInTx     bool
	purchaseErr      error
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

func (s *subscriptionAddonRepoStub) ListProducts(_ context.Context, forSaleOnly bool) ([]SubscriptionAddonProduct, error) {
	s.listForSaleOnly = &forSaleOnly
	result := make([]SubscriptionAddonProduct, 0, len(s.products))
	for _, product := range s.products {
		if forSaleOnly && !product.ForSale {
			continue
		}
		copy := *product
		result = append(result, copy)
	}
	return result, s.productErr
}

func (s *subscriptionAddonRepoStub) GetProductByID(_ context.Context, id int64) (*SubscriptionAddonProduct, error) {
	if s.productErr != nil {
		return nil, s.productErr
	}
	product, ok := s.products[id]
	if !ok {
		return nil, ErrSubscriptionAddonProductNotFound
	}
	copy := *product
	return &copy, nil
}

func (s *subscriptionAddonRepoStub) UpdateProduct(_ context.Context, id int64, input UpdateSubscriptionAddonProductInput) (*SubscriptionAddonProduct, error) {
	if s.productErr != nil {
		return nil, s.productErr
	}
	product, ok := s.products[id]
	if !ok {
		return nil, ErrSubscriptionAddonProductNotFound
	}
	s.updatedProduct = &input
	product.Name = input.Name
	product.QuotaUSD = input.QuotaUSD
	product.Price = input.Price
	product.OriginalPrice = input.OriginalPrice
	product.ForSale = input.ForSale
	product.SortOrder = input.SortOrder
	copy := *product
	return &copy, nil
}

func (s *subscriptionAddonRepoStub) CreatePurchased(ctx context.Context, input CreatePurchasedSubscriptionAddonInput) (*SubscriptionAddonPack, error) {
	s.purchaseCalls++
	s.purchaseInTx = dbent.TxFromContext(ctx) != nil
	if s.purchaseErr != nil {
		return nil, s.purchaseErr
	}
	if s.purchasedByOrder == nil {
		s.purchasedByOrder = make(map[int64]*SubscriptionAddonPack)
	}
	if existing, ok := s.purchasedByOrder[input.OrderID]; ok {
		if input.ExpiresAt.Before(existing.ExpiresAt) {
			existing.ExpiresAt = input.ExpiresAt
		}
		copy := *existing
		return &copy, nil
	}
	pack := &SubscriptionAddonPack{
		ID:             int64(len(s.purchasedByOrder) + 1),
		SubscriptionID: input.SubscriptionID,
		UserID:         input.UserID,
		GroupID:        input.GroupID,
		QuotaUSD:       input.QuotaUSD,
		StartsAt:       time.Now(),
		ExpiresAt:      input.ExpiresAt,
		Status:         SubscriptionAddonStatusActive,
		Notes:          input.Notes,
	}
	s.purchasedByOrder[input.OrderID] = pack
	copy := *pack
	return &copy, nil
}

func (s *subscriptionAddonRepoStub) GetByPurchaseOrderID(_ context.Context, orderID int64) (*SubscriptionAddonPack, error) {
	pack, ok := s.purchasedByOrder[orderID]
	if !ok {
		return nil, ErrSubscriptionAddonNotFound
	}
	copy := *pack
	return &copy, nil
}

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
		DailyUsageUSD:    limit + 1,
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
		DailyUsageUSD:    limit + 1,
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
