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
