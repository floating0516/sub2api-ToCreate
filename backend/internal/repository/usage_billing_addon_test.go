package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestIncrementUsageBillingSubscriptionAddon(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE subscription_addon_packs")).
		WithArgs(3.0, int64(7), int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"quota_usd", "used_usd"}).AddRow(10.0, 10.5))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO subscription_addon_usage")).
		WithArgs(int64(7), int64(8), int64(9), int64(10), "req-1", 3.0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	addonID := int64(7)
	subscriptionID := int64(8)
	exhausted, err := incrementUsageBillingSubscriptionAddon(context.Background(), tx, &service.UsageBillingCommand{
		RequestID:      "req-1",
		APIKeyID:       10,
		UserID:         9,
		SubscriptionID: &subscriptionID,
		AddonPackID:    &addonID,
		AddonCost:       3,
	})
	require.NoError(t, err)
	require.True(t, exhausted)
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
