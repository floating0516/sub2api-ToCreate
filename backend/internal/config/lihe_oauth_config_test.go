package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadLiheOAuthDefaultsDisabled(t *testing.T) {
	resetViperWithJWTSecret(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.LiheOAuth.Enabled)
	require.Equal(t, "lihe-chat", cfg.LiheOAuth.ClientID)
	require.Equal(t, "https://lihe.chat/api/integrations/lihe/callback", cfg.LiheOAuth.RedirectURI)
	require.Equal(t, "https://lihe.chat/connect/lihe", cfg.LiheOAuth.ConnectURL)
}

func TestLoadLiheOAuthRequiresStrongConfidentialClientSecret(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("LIHE_OAUTH_ENABLED", "true")
	t.Setenv("LIHE_OAUTH_CLIENT_SECRET", "short")
	_, err := Load()
	require.ErrorContains(t, err, "lihe_oauth.client_secret must be at least 32 characters")
}

func TestLoadLiheOAuthAcceptsHTTPSStagingURLs(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("LIHE_OAUTH_ENABLED", "true")
	t.Setenv("LIHE_OAUTH_CLIENT_SECRET", strings.Repeat("s", 32))
	t.Setenv("LIHE_OAUTH_REDIRECT_URI", "https://chat.lihe.chat/api/integrations/lihe/callback")
	t.Setenv("LIHE_OAUTH_CONNECT_URL", "https://chat.lihe.chat/connect/lihe")
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.LiheOAuth.Enabled)
	require.Equal(t, "https://chat.lihe.chat/connect/lihe", cfg.LiheOAuth.ConnectURL)
}

func TestLoadLiheOAuthRejectsInsecureOrParameterizedURLs(t *testing.T) {
	tests := []struct {
		name        string
		envName     string
		value       string
		errorSubstr string
	}{
		{"http redirect", "LIHE_OAUTH_REDIRECT_URI", "http://chat.lihe.chat/callback", "must use https"},
		{"redirect query", "LIHE_OAUTH_REDIRECT_URI", "https://chat.lihe.chat/callback?next=evil", "must not include query"},
		{"connect userinfo", "LIHE_OAUTH_CONNECT_URL", "https://user@chat.lihe.chat/connect/lihe", "must not include userinfo"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			t.Setenv("LIHE_OAUTH_ENABLED", "true")
			t.Setenv("LIHE_OAUTH_CLIENT_SECRET", strings.Repeat("s", 32))
			t.Setenv(test.envName, test.value)
			_, err := Load()
			require.ErrorContains(t, err, test.errorSubstr)
		})
	}
}

func TestLoadRejectsReservedLiheAPIKeyPrefix(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("DEFAULT_API_KEY_PREFIX", "lihe_")
	_, err := Load()
	require.ErrorContains(t, err, "reserved Lihe OAuth prefix")
}
