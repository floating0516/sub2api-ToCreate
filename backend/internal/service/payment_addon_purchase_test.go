package service

import (
	"context"
	"errors"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type addonPurchaseSubscriptionRepo struct {
	userSubRepoNoop
	sub *UserSubscription
	err error
}

func (r *addonPurchaseSubscriptionRepo) GetByID(context.Context, int64) (*UserSubscription, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	copy := *r.sub
	return &copy, nil
}

func newAddonPurchaseService(sub *UserSubscription, addonRepo SubscriptionAddonRepository) *PaymentService {
	subscriptionSvc := NewSubscriptionService(groupRepoNoop{}, &addonPurchaseSubscriptionRepo{sub: sub}, nil, nil, nil)
	subscriptionSvc.addonRepo = addonRepo
	return &PaymentService{subscriptionSvc: subscriptionSvc}
}

func TestValidateAddonOrderUsesServerCatalogAndOwnedSubscription(t *testing.T) {
	now := time.Now()
	product := &SubscriptionAddonProduct{
		ID:       5,
		SKU:      "addon-usd-30",
		Name:     "30 USD add-on",
		QuotaUSD: 30,
		Price:    7.99,
		ForSale:  true,
	}
	subscription := &UserSubscription{
		ID:        19,
		UserID:    42,
		GroupID:   7,
		Status:    SubscriptionStatusActive,
		StartsAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}
	repo := &subscriptionAddonRepoStub{products: map[int64]*SubscriptionAddonProduct{product.ID: product}}
	svc := newAddonPurchaseService(subscription, repo)

	selection, err := svc.validateAddonOrder(context.Background(), CreateOrderRequest{
		UserID:          subscription.UserID,
		Amount:          0.01,
		OrderType:       payment.OrderTypeAddon,
		AddonProductID:  product.ID,
		SubscriptionID:  subscription.ID,
	}, &PaymentConfig{AddonPurchaseEnabled: true, MinAmount: 0.01, MaxAmount: 500})
	require.NoError(t, err)
	require.Equal(t, product.Price, selection.product.Price)
	require.Equal(t, product.QuotaUSD, selection.product.QuotaUSD)

	snapshot := attachAddonOrderSnapshot(nil, selection)
	parsed, err := parseAddonOrderSnapshot(&dbent.PaymentOrder{
		Amount:           product.Price,
		ProviderSnapshot: snapshot,
	})
	require.NoError(t, err)
	require.Equal(t, product.ID, parsed.ProductID)
	require.Equal(t, subscription.ID, parsed.SubscriptionID)
	require.Equal(t, subscription.GroupID, parsed.GroupID)
	require.Equal(t, product.Price, parsed.Price)
	require.Equal(t, product.QuotaUSD, parsed.QuotaUSD)
	require.True(t, parsed.ExpiresAt.Equal(subscription.ExpiresAt))
}

func TestValidateAddonOrderRejectsAnotherUsersSubscription(t *testing.T) {
	product := &SubscriptionAddonProduct{ID: 5, SKU: "addon-usd-10", QuotaUSD: 10, Price: 2.99, ForSale: true}
	subscription := &UserSubscription{
		ID:        19,
		UserID:    99,
		GroupID:   7,
		Status:    SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	svc := newAddonPurchaseService(subscription, &subscriptionAddonRepoStub{
		products: map[int64]*SubscriptionAddonProduct{product.ID: product},
	})

	_, err := svc.validateAddonOrder(context.Background(), CreateOrderRequest{
		UserID:          42,
		AddonProductID:  product.ID,
		SubscriptionID:  subscription.ID,
	}, &PaymentConfig{AddonPurchaseEnabled: true})
	require.Error(t, err)
	require.Equal(t, "FORBIDDEN", infraerrors.Reason(err))
}

func TestValidateAddonOrderKeepsRepositoryFailuresDistinctFromMissingProducts(t *testing.T) {
	repoErr := errors.New("database unavailable")
	svc := newAddonPurchaseService(&UserSubscription{}, &subscriptionAddonRepoStub{productErr: repoErr})

	_, err := svc.validateAddonOrder(context.Background(), CreateOrderRequest{
		UserID:         42,
		AddonProductID: 5,
		SubscriptionID: 19,
	}, &PaymentConfig{AddonPurchaseEnabled: true})
	require.ErrorIs(t, err, repoErr)
	require.NotEqual(t, "ADDON_PRODUCT_NOT_AVAILABLE", infraerrors.Reason(err))
}

func TestExecuteAddonFulfillmentGrantsOnceAndHonorsSnapshotExpiry(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)

	user, err := client.User.Create().
		SetEmail("addon-purchase@example.com").
		SetPasswordHash("hash").
		SetUsername("addon-purchase-user").
		Save(ctx)
	require.NoError(t, err)

	snapshotExpiry := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)
	product := &SubscriptionAddonProduct{
		ID:       5,
		SKU:      "addon-usd-30",
		Name:     "30 USD add-on",
		QuotaUSD: 30,
		Price:    7.99,
		ForSale:  true,
	}
	subscription := &UserSubscription{
		ID:        19,
		UserID:    user.ID,
		GroupID:   7,
		Status:    SubscriptionStatusActive,
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: snapshotExpiry,
	}
	selection := &addonOrderSelection{product: product, subscription: subscription}
	providerSnapshot := attachAddonOrderSnapshot(nil, selection)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(product.Price).
		SetPayAmount(product.Price).
		SetFeeRate(0).
		SetRechargeCode("PAY-ADDON-1").
		SetOutTradeNo("sub2_addon_fulfillment_1").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-addon-1").
		SetOrderType(payment.OrderTypeAddon).
		SetStatus(OrderStatusPaid).
		SetPaidAt(time.Now()).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		SetProviderSnapshot(providerSnapshot).
		Save(ctx)
	require.NoError(t, err)

	// Extending the subscription after checkout must not silently extend the
	// add-on beyond the date shown to the buyer.
	subscription.ExpiresAt = snapshotExpiry.Add(48 * time.Hour)
	addonRepo := &subscriptionAddonRepoStub{}
	svc := newAddonPurchaseService(subscription, addonRepo)
	svc.entClient = client

	require.NoError(t, svc.ExecuteAddonFulfillment(ctx, order.ID))
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status)
	require.Equal(t, 1, addonRepo.purchaseCalls)
	require.Len(t, addonRepo.purchasedByOrder, 1)
	pack := addonRepo.purchasedByOrder[order.ID]
	require.NotNil(t, pack)
	require.Equal(t, subscription.ID, pack.SubscriptionID)
	require.Equal(t, user.ID, pack.UserID)
	require.Equal(t, product.QuotaUSD, pack.QuotaUSD)
	require.True(t, pack.ExpiresAt.Equal(snapshotExpiry))

	require.NoError(t, svc.ExecuteAddonFulfillment(ctx, order.ID))
	require.Equal(t, 1, addonRepo.purchaseCalls)
	require.Len(t, addonRepo.purchasedByOrder, 1)

	// Simulate a failure after quota creation but before the order was durably
	// completed. Recovery must use purchase_order_id even after subscription expiry.
	_, err = client.PaymentOrder.UpdateOneID(order.ID).
		SetStatus(OrderStatusFailed).
		Save(ctx)
	require.NoError(t, err)
	subscription.Status = SubscriptionStatusExpired
	subscription.ExpiresAt = time.Now().Add(-time.Minute)
	require.NoError(t, svc.ExecuteAddonFulfillment(ctx, order.ID))
	require.Equal(t, 1, addonRepo.purchaseCalls)
	require.Len(t, addonRepo.purchasedByOrder, 1)
}
