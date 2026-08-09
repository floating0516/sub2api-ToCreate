package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestLoadAccountContributionAdminOverviewReturnsSafeReadOnlyProjection(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT key, value.*FROM settings`).
		WillReturnRows(sqlmock.NewRows([]string{"key", "value"}).
			AddRow(SettingKeyAccountContributionEnabled, "false").
			AddRow(SettingKeyAccountContributionSubmissionEnabled, "true").
			AddRow(SettingKeyAccountContributionPayoutEnabled, "true"))

	mock.ExpectQuery(`(?s)SELECT.*FROM contributor_profiles`).
		WillReturnRows(sqlmock.NewRows([]string{
			"contributors_total",
			"contributors_pending",
			"contributions_total",
			"contributions_active",
			"earning_entries_total",
			"total_earnings_cny_fen",
			"available_earnings_cny_fen",
			"payout_requests_total",
			"payout_requests_pending",
			"pending_payout_cny_fen",
		}).AddRow(3, 1, 2, 1, 4, 12345, 10000, 1, 1, 5000))

	mock.ExpectQuery(`(?s)SELECT.*cp\.id.*COUNT\(ac\.id\).*FROM contributor_profiles cp`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "email", "username", "status", "contributions", "created_at",
		}).AddRow(11, 7, "contributor@example.com", "contributor", "active", 2, now))

	mock.ExpectQuery(`(?s)SELECT.*ac\.id.*FROM account_contributions ac`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "contributor_id", "contributor", "account_id", "account_name", "platform", "status", "settlement_mode", "share_rate_bps", "created_at",
		}).AddRow(21, 11, "contributor@example.com", 31, "Codex Pool 1", "openai", "active", "cost_share", 3500, now))

	mock.ExpectQuery(`(?s)SELECT.*l\.id.*FROM contributor_earning_ledger l`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "contributor_id", "contributor", "contribution_id", "account_name", "entry_type", "amount_cny_fen", "available_at", "created_at",
		}).AddRow(41, 11, "contributor@example.com", 21, "Codex Pool 1", "accrual", 12345, now.Add(24*time.Hour), now))

	mock.ExpectQuery(`(?s)SELECT.*pr\.id.*FROM contributor_payout_requests pr`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "contributor_id", "contributor", "amount_cny_fen", "status", "method_type", "masked_destination", "requested_at",
		}).AddRow(51, 11, "contributor@example.com", 5000, "requested", "alipay", "138****0000", now))

	overview, err := loadAccountContributionAdminOverview(context.Background(), db)
	require.NoError(t, err)
	require.False(t, overview.Features.Enabled)
	require.True(t, overview.Features.SubmissionConfigured)
	require.True(t, overview.Features.PayoutConfigured)
	require.False(t, overview.Features.SubmissionEnabled)
	require.False(t, overview.Features.PayoutEnabled)
	require.Equal(t, int64(3), overview.Stats.ContributorsTotal)
	require.Len(t, overview.Contributors, 1)
	require.Equal(t, int64(7), *overview.Contributors[0].UserID)
	require.Len(t, overview.Contributions, 1)
	require.Equal(t, int64(31), *overview.Contributions[0].AccountID)
	require.Len(t, overview.Earnings, 1)
	require.Len(t, overview.Payouts, 1)
	require.Equal(t, "138****0000", overview.Payouts[0].MaskedDestination)

	payload, err := json.Marshal(overview)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "encrypted_payload")
	require.NotContains(t, string(payload), "upstream_identity_hash")
	require.NotContains(t, string(payload), "metadata")
	require.NoError(t, mock.ExpectationsWereMet())
}
