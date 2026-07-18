package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIQuotaSamplesMigrationBoundsAndIndexesTimelineData(t *testing.T) {
	content, err := FS.ReadFile("185_openai_quota_samples.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS openai_quota_samples")
	require.Contains(t, sql, "cycle_id BIGINT NOT NULL REFERENCES openai_quota_cycles(id) ON DELETE CASCADE")
	require.Contains(t, sql, "CHECK (used_percent >= 0 AND used_percent <= 100)")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_openai_quota_samples_cycle_bucket")
	require.Contains(t, sql, "ON openai_quota_samples (cycle_id, bucket_started_at)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_openai_quota_samples_account_observed")
	require.Contains(t, sql, "ON openai_quota_samples (account_id, observed_at DESC)")
}
