package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBenefitGrantCreateCampaignRollsBackBeforeConflictLookup(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO benefit_grant_campaigns").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()
	mock.ExpectQuery("SELECT .* FROM benefit_grant_campaigns c").
		WillReturnError(errors.New("lookup after rollback"))

	repo := NewBenefitGrantRepository(db)
	campaign, created, err := repo.CreateCampaign(context.Background(), &service.BenefitGrantCampaign{
		OperationKey: "benefit-grant-operation-1",
		CreatedBy:    9,
	}, nil, nil)

	require.ErrorContains(t, err, "lookup after rollback")
	require.False(t, created)
	require.Nil(t, campaign)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBenefitGrantListSubscriptionStatesExcludesSoftDeletedRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(
		"FROM user_subscriptions WHERE user_id = ANY\\(\\$1\\) AND group_id = \\$2 AND deleted_at IS NULL",
	).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "status", "expires_at", "notes"}))

	repo := NewBenefitGrantRepository(db)
	states, err := repo.ListSubscriptionStates(context.Background(), []int64{7}, 17)

	require.NoError(t, err)
	require.Empty(t, states)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBenefitGrantApplyBalanceCommitsBalanceResultAndInvalidationTogether(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE benefit_grant_recipients").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE users").
		WillReturnRows(
			sqlmock.NewRows([]string{"balance_before", "balance_after"}).
				AddRow(5.0, 7.5),
		)
	mock.ExpectExec("INSERT INTO auth_cache_invalidation_outbox").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE benefit_grant_recipients").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewBenefitGrantRepository(db)
	result, err := repo.ApplyBalanceRecipient(
		context.Background(),
		11,
		7,
		2.5,
		false,
		time.Time{},
	)

	require.NoError(t, err)
	require.True(t, result.Claimed)
	require.True(t, result.Granted)
	require.Equal(t, 5.0, result.BalanceBefore)
	require.Equal(t, 7.5, result.BalanceAfter)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBenefitGrantApplyBalanceRollsBackWhenInvalidationCannotBeQueued(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE benefit_grant_recipients").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("UPDATE users").
		WillReturnRows(
			sqlmock.NewRows([]string{"balance_before", "balance_after"}).
				AddRow(5.0, 7.5),
		)
	mock.ExpectExec("INSERT INTO auth_cache_invalidation_outbox").
		WillReturnError(errors.New("outbox unavailable"))
	mock.ExpectRollback()

	repo := NewBenefitGrantRepository(db)
	result, err := repo.ApplyBalanceRecipient(
		context.Background(),
		11,
		7,
		2.5,
		false,
		time.Time{},
	)

	require.ErrorContains(t, err, "outbox unavailable")
	require.Nil(t, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTruncateBenefitGrantErrorKeepsUTF8Valid(t *testing.T) {
	require.Equal(
		t,
		"\u6743\u76ca\u53d1\u653e",
		truncateBenefitGrantError("\u6743\u76ca\u53d1\u653e\u5931\u8d25", 4),
	)
}
