package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionAddonProductsMigrationSeedsCatalogAndIdempotencyIndex(t *testing.T) {
	content, err := FS.ReadFile("177_subscription_addon_products.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS subscription_addon_products")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS purchase_order_id BIGINT")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS subscription_addon_packs_purchase_order_unique_idx")
	require.Contains(t, sql, "WHERE purchase_order_id IS NOT NULL")
	require.Contains(t, sql, "'addon-usd-10'")
	require.Contains(t, sql, "10, 2.99, 10)")
	require.Contains(t, sql, "'addon-usd-30'")
	require.Contains(t, sql, "30, 7.99, 20)")
	require.Contains(t, sql, "'addon-usd-50'")
	require.Contains(t, sql, "50, 12.99, 30)")
	require.Contains(t, sql, "'addon-usd-100'")
	require.Contains(t, sql, "100, 23.99, 40)")
	require.Contains(t, sql, "'addon-usd-200'")
	require.Contains(t, sql, "200, 44.99, 50)")
	require.Contains(t, sql, "ON CONFLICT (sku) DO NOTHING")
}
