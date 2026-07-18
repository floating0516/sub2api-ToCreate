package oidcprovider

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func frozenTestProvider() *Provider {
	return &Provider{
		enabled: true,
		config: config.LiheOIDCConfig{
			Issuer:          "https://api.lihe.chat",
			ClientID:        "lihe-chat-login",
			RedirectURI:     "https://lihe.chat/oauth/openid/callback",
			LinkRedirectURI: "https://lihe.chat/oauth/openid/link/callback",
		},
	}
}

func validAuthorizationParams() url.Values {
	return url.Values{
		"response_type":         {"code"},
		"client_id":             {"lihe-chat-login"},
		"redirect_uri":          {"https://lihe.chat/oauth/openid/callback"},
		"scope":                 {Scope},
		"state":                 {strings.Repeat("s", 32)},
		"nonce":                 {strings.Repeat("n", 32)},
		"code_challenge":        {strings.Repeat("c", 43)},
		"code_challenge_method": {"S256"},
	}
}

func TestDiscoveryMatchesFrozenLiheContract(t *testing.T) {
	document, err := frozenTestProvider().Discovery()
	require.NoError(t, err)
	fixtureData, err := os.ReadFile("testdata/openid-configuration.json")
	require.NoError(t, err)
	var fixture DiscoveryDocument
	require.NoError(t, json.Unmarshal(fixtureData, &fixture))
	require.Equal(t, fixture, document)
	require.Equal(t, "https://api.lihe.chat", document.Issuer)
	require.Equal(t, "https://api.lihe.chat/oidc/authorize", document.AuthorizationEndpoint)
	require.Equal(t, "https://api.lihe.chat/oidc/token", document.TokenEndpoint)
	require.Equal(t, "https://api.lihe.chat/oidc/userinfo", document.UserInfoEndpoint)
	require.Equal(t, "https://api.lihe.chat/oidc/jwks", document.JWKSURI)
	require.Equal(t, []string{"code"}, document.ResponseTypesSupported)
	require.Equal(t, []string{"authorization_code"}, document.GrantTypesSupported)
	require.Equal(t, []string{"public"}, document.SubjectTypesSupported)
	require.Equal(t, []string{"RS256"}, document.IDTokenSigningAlgsSupported)
	require.Equal(t, []string{"client_secret_basic"}, document.TokenEndpointAuthMethodsSupported)
	require.Equal(t, []string{"S256"}, document.CodeChallengeMethodsSupported)
	require.NotContains(t, document.GrantTypesSupported, "refresh_token")
	for _, claim := range []string{
		"iss", "sub", "aud", "exp", "iat", "nonce", "email",
		"email_verified", "preferred_username", "name",
	} {
		require.Contains(t, document.ClaimsSupported, claim)
	}
}

func TestAuthorizationParametersEnforceFrozenClientPKCEAndNonce(t *testing.T) {
	provider := frozenTestProvider()
	require.NoError(t, provider.validateAuthorizationParameters(validAuthorizationParams()))
	linkParams := validAuthorizationParams()
	linkParams.Set("redirect_uri", "https://lihe.chat/oauth/openid/link/callback")
	require.NoError(t, provider.validateAuthorizationParameters(linkParams))

	tests := []struct {
		name   string
		mutate func(url.Values)
	}{
		{"wrong client", func(values url.Values) { values.Set("client_id", "other-client") }},
		{"wrong redirect", func(values url.Values) { values.Set("redirect_uri", "https://attacker.example/callback") }},
		{"plain PKCE", func(values url.Values) { values.Set("code_challenge_method", "plain") }},
		{"short PKCE", func(values url.Values) { values.Set("code_challenge", "short") }},
		{"missing nonce", func(values url.Values) { values.Del("nonce") }},
		{"short state", func(values url.Values) { values.Set("state", "short") }},
		{"extra scope", func(values url.Values) { values.Set("scope", Scope+" offline_access") }},
		{"unknown parameter", func(values url.Values) { values.Set("request_uri", "urn:example") }},
		{"unsupported max age", func(values url.Values) { values.Set("max_age", "300") }},
		{"duplicate parameter", func(values url.Values) { values["client_id"] = append(values["client_id"], "lihe-chat-login") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := validAuthorizationParams()
			test.mutate(values)
			require.Error(t, provider.validateAuthorizationParameters(values))
		})
	}
}

func TestPromptNoneUsesTheValidatedRequestCallback(t *testing.T) {
	provider := frozenTestProvider()
	params := validAuthorizationParams()
	params.Set("redirect_uri", "https://lihe.chat/oauth/openid/link/callback")
	params.Set("prompt", "none")
	require.NoError(t, provider.validateAuthorizationParameters(params))

	redirect := provider.authorizationErrorRedirect(params, "login_required")
	parsed, err := url.Parse(redirect)
	require.NoError(t, err)
	require.Equal(t, "https", parsed.Scheme)
	require.Equal(t, "lihe.chat", parsed.Host)
	require.Equal(t, "/oauth/openid/link/callback", parsed.Path)
	require.Equal(t, "login_required", parsed.Query().Get("error"))
	require.Equal(t, params.Get("state"), parsed.Query().Get("state"))

	params.Set("redirect_uri", "https://attacker.example/callback")
	require.Empty(t, provider.authorizationErrorRedirect(params, "login_required"))
}

func TestProfileClaimsUseStableSubjectAndReliableEmailFlag(t *testing.T) {
	claims := profileClaims(&UserProfile{
		Subject:           "11111111-1111-4111-8111-111111111111",
		Email:             "user@example.com",
		EmailVerified:     false,
		PreferredUsername: "user",
		Name:              "User",
	})
	require.Equal(t, "11111111-1111-4111-8111-111111111111", claims["sub"])
	require.Equal(t, "user@example.com", claims["email"])
	require.Equal(t, false, claims["email_verified"])
	require.Equal(t, "user", claims["preferred_username"])
	require.Equal(t, "User", claims["name"])
}

func TestProtocolTokenLifetimesAreFrozen(t *testing.T) {
	require.Equal(t, 60, int(AuthorizationCodeTTL.Seconds()))
	require.Equal(t, 300, int(AccessTokenTTL.Seconds()))
	require.Equal(t, 300, int(IDTokenTTL.Seconds()))
}
