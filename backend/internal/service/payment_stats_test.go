//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func TestPaymentDashboardStatsExcludeBalanceWalletOrders(t *testing.T) {
	ctx := context.Background()
	client := newPaymentStatsTestClient(t)
	now := time.Now()

	user, err := client.User.Create().
		SetEmail("payment-stats@example.com").
		SetPasswordHash("hash").
		SetUsername("payment-stats-user").
		Save(ctx)
	require.NoError(t, err)

	createPaymentStatsOrder(t, ctx, client, user.ID, user.Email, user.Username, payment.TypeAlipay, 20, now.Add(-time.Hour))
	createPaymentStatsOrder(t, ctx, client, user.ID, user.Email, user.Username, PaymentTypeBalanceWallet, 8, now.Add(-30*time.Minute))

	svc := &PaymentService{entClient: client}
	got, err := svc.GetDashboardStats(ctx, 1)
	require.NoError(t, err)

	require.Equal(t, 20.0, got.TotalAmount)
	require.Equal(t, 20.0, got.TodayAmount)
	require.Equal(t, 1, got.TotalCount)
	require.Equal(t, 1, got.TodayCount)
	require.Equal(t, 20.0, got.AvgAmount)
	require.Len(t, got.PaymentMethods, 1)
	require.Equal(t, payment.TypeAlipay, got.PaymentMethods[0].Type)
	require.Equal(t, 20.0, got.PaymentMethods[0].Amount)
	require.Len(t, got.TopUsers, 1)
	require.Equal(t, 20.0, got.TopUsers[0].Amount)
	require.NotEmpty(t, got.DailySeries)
	require.Equal(t, 20.0, got.DailySeries[len(got.DailySeries)-1].Amount)
}

func createPaymentStatsOrder(t *testing.T, ctx context.Context, client *dbent.Client, userID int64, email, username, paymentType string, amount float64, paidAt time.Time) {
	t.Helper()

	_, err := client.PaymentOrder.Create().
		SetUserID(userID).
		SetUserEmail(email).
		SetUserName(username).
		SetAmount(amount).
		SetPayAmount(amount).
		SetFeeRate(0).
		SetRechargeCode("STAT").
		SetOutTradeNo("").
		SetPaymentType(paymentType).
		SetPaymentTradeNo(paymentType + "-trade").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(paidAt.Add(time.Hour)).
		SetPaidAt(paidAt).
		SetCompletedAt(paidAt).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.com").
		Save(ctx)
	require.NoError(t, err)
}

func newPaymentStatsTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:payment_stats?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}
