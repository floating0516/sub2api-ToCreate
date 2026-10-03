//go:build unit

package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestSupportOrder_ReadOnlyOwnership(t *testing.T) {
	for _, tc := range []struct {
		name    string
		owner   int64
		missing bool
		want    error
	}{
		{"own", 123, false, nil}, {"other", 999, false, ErrSupportOrderForbidden}, {"missing", 0, true, ErrManagedRechargeOrderMissing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			query := mock.ExpectQuery("SELECT user_id FROM managed_recharge_orders WHERE id = ").WithArgs(int64(42))
			if tc.missing {
				query.WillReturnError(sql.ErrNoRows)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(tc.owner))
			}
			s := &ManagedRechargeService{db: db}
			err = s.ValidateSupportOrder(context.Background(), 123, 42)
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.want)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
