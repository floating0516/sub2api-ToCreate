package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiheOIDCProviderMigrationFreezesSubjectAndHashesCredentials(t *testing.T) {
	content, err := FS.ReadFile("183_lihe_oidc_provider.sql")
	require.NoError(t, err)
	sql := string(content)

	require.Contains(t, sql, "oidc_subject UUID DEFAULT gen_random_uuid()")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_subject_unique")
	indexStart := strings.Index(sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_subject_unique")
	require.NotEqual(t, -1, indexStart)
	indexEnd := strings.Index(sql[indexStart:], ";")
	require.NotEqual(t, -1, indexEnd)
	require.NotContains(t, strings.ToLower(sql[indexStart:indexStart+indexEnd]), " where ")
	require.Contains(t, sql, "prevent_user_oidc_subject_change")
	require.Contains(t, sql, "users.oidc_subject is immutable")

	require.Contains(t, sql, "request_hash CHAR(64)")
	require.Contains(t, sql, "browser_binding_hash CHAR(64)")
	require.Contains(t, sql, "signature_hash CHAR(64)")
	require.Contains(t, sql, "code_hash CHAR(64)")
	require.Contains(t, sql, "nonce_hash CHAR(64)")
	require.Contains(t, sql, "code_challenge_hash CHAR(64)")
	require.NotContains(t, sql, "code_challenge VARCHAR")
	require.Contains(t, sql, "request_data->>'v' = '1'")

	require.Contains(t, sql, "lihe_oidc_authorization_codes")
	require.Contains(t, sql, "lihe_oidc_access_tokens")
	require.NotContains(t, sql, "lihe_oauth_access_tokens (")
}

func TestLiheOIDCProviderMigrationAddsReliableEmailEvidence(t *testing.T) {
	content, err := FS.ReadFile("183_lihe_oidc_provider.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "email_verified_at TIMESTAMPTZ")
	require.Contains(t, sql, "email_verification_source VARCHAR(64)")
	require.Contains(t, sql, "users_email_verification_evidence_consistent")
	require.Contains(t, sql, "clear_user_email_verification_on_change")
	require.Contains(t, sql, "BEFORE UPDATE OF email ON users")
}
