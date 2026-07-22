package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIQuotaResetSourcesMigrationAddsClassificationAndCreditEvidence(t *testing.T) {
	content, err := FS.ReadFile("186_openai_quota_reset_sources.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "detected_reset_source VARCHAR(16) NOT NULL DEFAULT 'unknown'")
	require.Contains(t, sql, "reset_source_override VARCHAR(16)")
	require.Contains(t, sql, "reset_credit_snapshot_at TIMESTAMPTZ")
	require.Contains(t, sql, "reset_credit_expirations JSONB")
	require.Contains(t, sql, "detection_reason = 'manual_reset' THEN 'manual'")
	require.Contains(t, sql, "detection_reason = 'window_elapsed' THEN 'provider'")
	require.Contains(t, sql, "reset_source_override IS NULL OR reset_source_override IN ('manual', 'provider')")
}
