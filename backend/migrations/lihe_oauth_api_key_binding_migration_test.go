package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiheOAuthAPIKeyBindingMigrationRequiresSelectedKey(t *testing.T) {
	content, err := FS.ReadFile("182_lihe_oauth_api_key_binding.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS api_key_id BIGINT")
	require.Contains(t, sql, "DELETE FROM lihe_oauth_authorization_codes WHERE api_key_id IS NULL")
	require.NotContains(t, sql, "ALTER COLUMN api_key_id SET NOT NULL")
	require.Contains(t, sql, "FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE")
	require.Contains(t, sql, "ALTER COLUMN api_key_id DROP NOT NULL")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS source_api_key_id BIGINT")
	require.Contains(t, sql, "FOREIGN KEY (source_api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS lihe_oauth_binding_api_key_unique")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS lihe_oauth_token_bindings_api_key_id_fkey")
}
