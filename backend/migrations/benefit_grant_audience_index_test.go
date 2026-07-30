package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenefitGrantAudienceIndexMigration(t *testing.T) {
	content, err := FS.ReadFile("192_users_today_active_index_notx.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_today_active_audience")
	require.Contains(t, sql, "ON users (last_active_at, id)")
	require.Contains(t, sql, "deleted_at IS NULL")
	require.Contains(t, sql, "status = 'active'")
	require.Contains(t, sql, "role = 'user'")
	require.False(t, strings.Contains(strings.ToUpper(sql), "BEGIN;"))
}
