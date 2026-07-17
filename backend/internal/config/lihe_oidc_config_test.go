package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadLiheOIDCDefaultsDisabled(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.LiheOIDC.Enabled)
	require.Equal(t, "https://api.lihe.chat", cfg.LiheOIDC.Issuer)
	require.Equal(t, "lihe-chat-login", cfg.LiheOIDC.ClientID)
	require.Equal(t, "https://lihe.chat/oauth/openid/callback", cfg.LiheOIDC.RedirectURI)
	require.Equal(t, "/app/data/oidc-keys", cfg.LiheOIDC.KeyDirectory)
}

func TestLoadLiheOIDCRequiresIndependentSecrets(t *testing.T) {
	for _, test := range []struct {
		name        string
		client      string
		hmac        string
		liheOAuth   string
		errorSubstr string
	}{
		{"short client secret", "short", strings.Repeat("h", 32), "", "client_secret must be at least 32"},
		{"short hmac secret", strings.Repeat("c", 32), "short", "", "hmac_secret must be at least 32"},
		{"same provider secrets", strings.Repeat("x", 32), strings.Repeat("x", 32), "", "hmac_secret must be independent"},
		{"reused api key oauth secret", strings.Repeat("x", 32), strings.Repeat("h", 32), strings.Repeat("x", 32), "must be independent from lihe_oauth"},
		{"reused api key oauth hmac", strings.Repeat("c", 32), strings.Repeat("x", 32), strings.Repeat("x", 32), "hmac_secret must be independent from lihe_oauth"},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			t.Setenv("LIHE_OIDC_ENABLED", "true")
			t.Setenv("LIHE_OIDC_CLIENT_SECRET", test.client)
			t.Setenv("LIHE_OIDC_HMAC_SECRET", test.hmac)
			if test.liheOAuth != "" {
				t.Setenv("LIHE_OAUTH_CLIENT_SECRET", test.liheOAuth)
			}
			_, err := Load()
			require.ErrorContains(t, err, test.errorSubstr)
		})
	}
}

func TestLoadLiheOIDCRejectsJWTSecretReuse(t *testing.T) {
	resetViperWithJWTSecret(t)
	jwtSecret := strings.Repeat("j", 32)
	t.Setenv("JWT_SECRET", jwtSecret)
	t.Setenv("LIHE_OIDC_ENABLED", "true")
	t.Setenv("LIHE_OIDC_CLIENT_SECRET", jwtSecret)
	t.Setenv("LIHE_OIDC_HMAC_SECRET", strings.Repeat("h", 32))
	_, err := Load()
	require.ErrorContains(t, err, "must be independent from jwt.secret")
}

func TestLoadLiheOIDCAcceptsFrozenProductionProtocol(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("LIHE_OIDC_ENABLED", "true")
	t.Setenv("LIHE_OIDC_CLIENT_SECRET", strings.Repeat("c", 32))
	t.Setenv("LIHE_OIDC_HMAC_SECRET", strings.Repeat("h", 32))
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.LiheOIDC.Enabled)
	require.Equal(t, "https://api.lihe.chat", cfg.LiheOIDC.Issuer)
	require.Equal(t, "lihe-chat-login", cfg.LiheOIDC.ClientID)
}

func TestLoadLiheOIDCRejectsInvalidIssuerAndKeyDirectory(t *testing.T) {
	for _, test := range []struct {
		name        string
		envName     string
		value       string
		errorSubstr string
	}{
		{"issuer scheme", "LIHE_OIDC_ISSUER", "http://api.lihe.chat", "must use https"},
		{"issuer path", "LIHE_OIDC_ISSUER", "https://api.lihe.chat/tenant", "must not include a path"},
		{"relative key directory", "LIHE_OIDC_KEY_DIRECTORY", "data/oidc", "must be an absolute path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			t.Setenv("LIHE_OIDC_ENABLED", "true")
			t.Setenv("LIHE_OIDC_CLIENT_SECRET", strings.Repeat("c", 32))
			t.Setenv("LIHE_OIDC_HMAC_SECRET", strings.Repeat("h", 32))
			t.Setenv(test.envName, test.value)
			_, err := Load()
			require.ErrorContains(t, err, test.errorSubstr)
		})
	}
}
