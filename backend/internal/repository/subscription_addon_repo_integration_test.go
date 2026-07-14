//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionAddonCreatePurchasedRollsBackWithEntTransaction(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	suffix := time.Now().UnixNano()

	user, err := tx.User.Create().
		SetEmail(fmt.Sprintf("addon-tx-%d@example.com", suffix)).
		SetUsername("addon-tx-user").
		SetPasswordHash("test-password-hash").
		SetStatus(payment.EntityStatusActive).
		Save(txCtx)
	require.NoError(t, err)

	group, err := tx.Group.Create().
		SetName(fmt.Sprintf("addon-tx-group-%d", suffix)).
		SetPlatform("openai").
		SetSubscriptionType("subscription").
		Save(txCtx)
	require.NoError(t, err)

	expiresAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond)
	subscription, err := tx.UserSubscription.Create().
		SetUserID(user.ID).
		SetGroupID(group.ID).
		SetStartsAt(time.Now().Add(-time.Hour)).
		SetExpiresAt(expiresAt).
		SetStatus(service.SubscriptionStatusActive).
		Save(txCtx)
	require.NoError(t, err)

	order, err := tx.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(1.99).
		SetPayAmount(1.99).
		SetRechargeCode("BAL-ADD-TEST").
		SetOutTradeNo(fmt.Sprintf("addon-tx-%d", suffix)).
		SetPaymentType(service.PaymentTypeBalanceWallet).
		SetPaymentTradeNo(fmt.Sprintf("balance:addon:%d", suffix)).
		SetOrderType(payment.OrderTypeAddon).
		SetStatus(service.OrderStatusCompleted).
		SetExpiresAt(time.Now()).
		SetClientIP("127.0.0.1").
		SetSrcHost("integration.test").
		Save(txCtx)
	require.NoError(t, err)

	repo := NewSubscriptionAddonRepository(integrationDB)
	pack, err := repo.CreatePurchased(txCtx, service.CreatePurchasedSubscriptionAddonInput{
		OrderID:        order.ID,
		SubscriptionID: subscription.ID,
		UserID:         user.ID,
		GroupID:        group.ID,
		QuotaUSD:       10,
		ExpiresAt:      expiresAt,
		Notes:          "transaction rollback test",
	})
	require.NoError(t, err)
	require.NotZero(t, pack.ID)

	visibleInTransaction, err := repo.GetByPurchaseOrderID(txCtx, order.ID)
	require.NoError(t, err)
	require.Equal(t, pack.ID, visibleInTransaction.ID)

	require.NoError(t, tx.Rollback())

	var count int
	err = integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM subscription_addon_packs WHERE id = $1`, pack.ID).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSubscriptionAddonUpdateProductRollsBackWithEntTransaction(t *testing.T) {
	ctx := context.Background()
	var original service.SubscriptionAddonProduct
	var originalPrice *float64
	err := integrationDB.QueryRowContext(ctx, `
		SELECT id, sku, name, quota_usd, price, original_price, for_sale, sort_order, created_at, updated_at
		FROM subscription_addon_products
		ORDER BY id ASC
		LIMIT 1
	`).Scan(
		&original.ID,
		&original.SKU,
		&original.Name,
		&original.QuotaUSD,
		&original.Price,
		&originalPrice,
		&original.ForSale,
		&original.SortOrder,
		&original.CreatedAt,
		&original.UpdatedAt,
	)
	require.NoError(t, err)
	original.OriginalPrice = originalPrice

	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	repo := NewSubscriptionAddonRepository(integrationDB)
	updatedOriginalPrice := 9.99
	updated, err := repo.UpdateProduct(txCtx, original.ID, service.UpdateSubscriptionAddonProductInput{
		Name:          "Transaction-only add-on",
		QuotaUSD:      42,
		Price:         7.25,
		OriginalPrice: &updatedOriginalPrice,
		ForSale:       !original.ForSale,
		SortOrder:     original.SortOrder + 1,
	})
	require.NoError(t, err)
	require.Equal(t, original.SKU, updated.SKU)
	require.Equal(t, "Transaction-only add-on", updated.Name)

	visibleInTransaction, err := repo.GetProductByID(txCtx, original.ID)
	require.NoError(t, err)
	require.Equal(t, updated.Name, visibleInTransaction.Name)
	require.NoError(t, tx.Rollback())

	var name string
	var quotaUSD, price float64
	var forSale bool
	var sortOrder int
	err = integrationDB.QueryRowContext(ctx, `
		SELECT name, quota_usd, price, for_sale, sort_order
		FROM subscription_addon_products
		WHERE id = $1
	`, original.ID).Scan(&name, &quotaUSD, &price, &forSale, &sortOrder)
	require.NoError(t, err)
	require.Equal(t, original.Name, name)
	require.InDelta(t, original.QuotaUSD, quotaUSD, 0.000001)
	require.InDelta(t, original.Price, price, 0.000001)
	require.Equal(t, original.ForSale, forSale)
	require.Equal(t, original.SortOrder, sortOrder)
}
