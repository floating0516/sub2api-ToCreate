package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type balanceAddonUserRepoStub struct {
	UserRepository
	user *User
}

func (s *balanceAddonUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	copy := *s.user
	return &copy, nil
}

type balanceAddonPurchaseFixture struct {
	service   *PaymentService
	client    *dbent.Client
	addonRepo *subscriptionAddonRepoStub
	product   *SubscriptionAddonProduct
	sub       *UserSubscription
	user      *User
}

func newBalanceAddonPurchaseFixture(t *testing.T, balance float64) *balanceAddonPurchaseFixture {
	t.Helper()
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	suffix := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())

	userEntity, err := client.User.Create().
		SetEmail("balance-addon-" + suffix + "@example.com").
		SetUsername("balance-addon-user").
		SetPasswordHash("hash").
		SetBalance(balance).
		SetStatus(payment.EntityStatusActive).
		Save(ctx)
	require.NoError(t, err)
	groupEntity, err := client.Group.Create().
		SetName("balance-addon-group-" + suffix).
		SetPlatform("openai").
		SetSubscriptionType("subscription").
		Save(ctx)
	require.NoError(t, err)
	expiresAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)
	subEntity, err := client.UserSubscription.Create().
		SetUserID(userEntity.ID).
		SetGroupID(groupEntity.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(expiresAt).
		SetStatus(SubscriptionStatusActive).
		Save(ctx)
	require.NoError(t, err)

	user := &User{
		ID:           userEntity.ID,
		Email:        userEntity.Email,
		Username:     userEntity.Username,
		PasswordHash: userEntity.PasswordHash,
		Balance:      balance,
		Status:       payment.EntityStatusActive,
	}
	sub := &UserSubscription{
		ID:        subEntity.ID,
		UserID:    userEntity.ID,
		GroupID:   groupEntity.ID,
		StartsAt:  subEntity.StartsAt,
		ExpiresAt: expiresAt,
		Status:    SubscriptionStatusActive,
	}
	product := &SubscriptionAddonProduct{
		ID:       1,
		SKU:      "addon-usd-10",
		Name:     "10 USD add-on",
		QuotaUSD: 10,
		Price:    1.99,
		ForSale:  true,
	}
	addonRepo := &subscriptionAddonRepoStub{
		products: map[int64]*SubscriptionAddonProduct{product.ID: product},
	}
	subscriptionSvc := NewSubscriptionService(
		groupRepoNoop{},
		&addonPurchaseSubscriptionRepo{sub: sub},
		nil,
		nil,
		nil,
	)
	subscriptionSvc.addonRepo = addonRepo
	configService := NewPaymentConfigService(nil, &paymentConfigSettingRepoStub{values: map[string]string{
		SettingAddonPurchaseEnabled: "true",
		SettingMinRechargeAmount:    "0.01",
		SettingMaxRechargeAmount:    "500",
	}}, nil)

	return &balanceAddonPurchaseFixture{
		service: &PaymentService{
			entClient:       client,
			userRepo:        &balanceAddonUserRepoStub{user: user},
			subscriptionSvc: subscriptionSvc,
			configService:   configService,
		},
		client:    client,
		addonRepo: addonRepo,
		product:   product,
		sub:       sub,
		user:      user,
	}
}

func TestPurchaseAddonWithBalanceCommitsDebitOrderAndGrantTogether(t *testing.T) {
	ctx := context.Background()
	fixture := newBalanceAddonPurchaseFixture(t, 10)

	result, err := fixture.service.PurchaseAddonWithBalance(ctx, BalanceAddonPurchaseRequest{
		UserID:         fixture.user.ID,
		AddonProductID: fixture.product.ID,
		SubscriptionID: fixture.sub.ID,
		ClientIP:       "127.0.0.1",
		SrcHost:        "example.test",
		SrcURL:         "https://example.test/purchase",
	})
	require.NoError(t, err)
	require.Equal(t, PaymentTypeBalanceWallet, result.PaymentType)
	require.Equal(t, fixture.product.ID, result.AddonProductID)
	require.Equal(t, fixture.sub.ID, result.SubscriptionID)
	require.InDelta(t, 10, result.BalanceBefore, 0.0001)
	require.InDelta(t, 8.01, result.BalanceAfter, 0.0001)
	require.True(t, fixture.addonRepo.purchaseInTx)
	require.Equal(t, 1, fixture.addonRepo.purchaseCalls)

	storedUser, err := fixture.client.User.Get(ctx, fixture.user.ID)
	require.NoError(t, err)
	require.InDelta(t, 8.01, storedUser.Balance, 0.0001)
	order, err := fixture.client.PaymentOrder.Query().
		Where(paymentorder.IDEQ(result.OrderID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, payment.OrderTypeAddon, order.OrderType)
	require.Equal(t, PaymentTypeBalanceWallet, order.PaymentType)
	require.Equal(t, OrderStatusCompleted, order.Status)
	require.InDelta(t, fixture.product.Price, order.Amount, 0.0001)
	snapshot, err := parseAddonOrderSnapshot(order)
	require.NoError(t, err)
	require.Equal(t, fixture.product.ID, snapshot.ProductID)
	require.Equal(t, fixture.sub.ID, snapshot.SubscriptionID)
	require.Equal(t, fixture.product.QuotaUSD, snapshot.QuotaUSD)
	require.True(t, snapshot.ExpiresAt.Equal(fixture.sub.ExpiresAt))

	audits, err := fixture.service.GetOrderAuditLogs(ctx, order.ID)
	require.NoError(t, err)
	require.Len(t, audits, 4)
	require.ElementsMatch(t, []string{"ORDER_CREATED", "BALANCE_DEDUCTED", "ADDON_GRANTED", "ADDON_SUCCESS"}, []string{
		audits[0].Action,
		audits[1].Action,
		audits[2].Action,
		audits[3].Action,
	})
}

func TestPurchaseAddonWithBalanceRollsBackWhenGrantFails(t *testing.T) {
	ctx := context.Background()
	fixture := newBalanceAddonPurchaseFixture(t, 10)
	fixture.addonRepo.purchaseErr = errors.New("grant failed")

	_, err := fixture.service.PurchaseAddonWithBalance(ctx, BalanceAddonPurchaseRequest{
		UserID:         fixture.user.ID,
		AddonProductID: fixture.product.ID,
		SubscriptionID: fixture.sub.ID,
		ClientIP:       "127.0.0.1",
		SrcHost:        "example.test",
	})
	require.ErrorContains(t, err, "grant balance add-on")
	require.True(t, fixture.addonRepo.purchaseInTx)

	storedUser, err := fixture.client.User.Get(ctx, fixture.user.ID)
	require.NoError(t, err)
	require.InDelta(t, 10, storedUser.Balance, 0.0001)
	orderCount, err := fixture.client.PaymentOrder.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, orderCount)
	auditCount, err := fixture.client.PaymentAuditLog.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, auditCount)
}
