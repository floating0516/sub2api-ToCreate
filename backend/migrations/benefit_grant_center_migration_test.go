package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBenefitGrantCenterMigrationCreatesDurableBatchAndRecipientTables(t *testing.T) {
	content, err := FS.ReadFile("193_benefit_grant_center.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS benefit_grant_campaigns")
	require.Contains(t, sql, "UNIQUE (created_by, operation_key)")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS benefit_grant_recipients")
	require.Contains(t, sql, "UNIQUE (campaign_id, user_id)")
	require.Contains(t, sql, "group_name_snapshot VARCHAR(100)")
	require.Contains(t, sql, "balance_before DECIMAL(20, 8)")
	require.Contains(t, sql, "balance_after DECIMAL(20, 8)")
}

func TestRecentRegisteredAudienceIndexMigrationIsOnline(t *testing.T) {
	content, err := FS.ReadFile("194_users_recent_registered_index_notx.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_recent_registered_audience")
	require.Contains(t, sql, "ON users (created_at, id)")
	require.Contains(t, sql, "deleted_at IS NULL")
	require.Contains(t, sql, "status = 'active'")
	require.Contains(t, sql, "role = 'user'")
	require.False(t, strings.Contains(strings.ToUpper(sql), "BEGIN;"))
}

func TestBenefitGrantActivityWindowMigrationAddsDeliveryModeAndAnnouncementLink(t *testing.T) {
	content, err := FS.ReadFile("195_benefit_grant_activity_window.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "delivery_mode VARCHAR(32)")
	require.Contains(t, sql, "'activity_window'")
	require.Contains(t, sql, "'scheduled'")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS benefit_grant_campaign_announcements")
	require.Contains(t, sql, "announcement_id BIGINT NOT NULL UNIQUE")
}
