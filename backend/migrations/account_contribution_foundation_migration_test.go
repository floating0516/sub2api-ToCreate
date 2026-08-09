package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountContributionFoundationIsDisabledAndPreservesFinancialRecords(t *testing.T) {
	content, err := FS.ReadFile("221_account_contribution_foundation.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "('account_contribution_enabled', 'false'")
	require.Contains(t, sql, "('account_contribution_submission_enabled', 'false'")
	require.Contains(t, sql, "('account_contribution_payout_enabled', 'false'")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS contributor_profiles")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS account_contributions")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS contributor_earning_ledger")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS contributor_payout_requests")
	require.Contains(t, sql, "idempotency_key VARCHAR(160) NOT NULL UNIQUE")
	require.Contains(t, sql, "source_usage_log_id BIGINT")
	require.NotContains(t, sql, "source_usage_log_id BIGINT REFERENCES usage_logs")
	require.Contains(t, sql, "REFERENCES account_contributions(id) ON DELETE RESTRICT")
	require.Contains(t, sql, "REFERENCES contributor_earning_ledger(id) ON DELETE RESTRICT")
	require.Contains(t, sql, "BEFORE UPDATE OR DELETE ON contributor_earning_ledger")
	require.Contains(t, sql, "add a reversal entry instead")
	require.False(t, strings.Contains(strings.ToUpper(sql), "DROP TABLE"))
}
