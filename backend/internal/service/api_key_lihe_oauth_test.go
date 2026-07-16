package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type liheOAuthTestAPIKeyRepo struct {
	APIKeyRepository
	code              *LiheAuthorizationCode
	exchangeInput     *LiheTokenExchangeInput
	resolved          *LiheResolvedAccess
	revokedTokenID    int64
	revokedUserID     int64
	revokedTokenHash  string
	revokedClientID   string
	listTokens        []LiheAccessToken
	apiKeys           map[int64]*APIKey
	forceExchangeUsed bool
}

func (r *liheOAuthTestAPIKeyRepo) GetByID(_ context.Context, id int64) (*APIKey, error) {
	apiKey, ok := r.apiKeys[id]
	if !ok {
		return nil, ErrAPIKeyNotFound
	}
	copyKey := *apiKey
	if apiKey.User != nil {
		copyUser := *apiKey.User
		copyKey.User = &copyUser
	}
	if apiKey.Group != nil {
		copyGroup := *apiKey.Group
		copyKey.Group = &copyGroup
	}
	return &copyKey, nil
}

func (r *liheOAuthTestAPIKeyRepo) CreateLiheAuthorizationCode(_ context.Context, code *LiheAuthorizationCode) error {
	copyCode := *code
	copyCode.Scopes = append([]string(nil), code.Scopes...)
	r.code = &copyCode
	return nil
}

func (r *liheOAuthTestAPIKeyRepo) GetLiheAuthorizationCode(_ context.Context, codeHash string) (*LiheAuthorizationCode, error) {
	if r.code == nil || r.code.CodeHash != codeHash {
		return nil, nil
	}
	return r.code, nil
}

func (r *liheOAuthTestAPIKeyRepo) ExchangeLiheAuthorizationCode(_ context.Context, input LiheTokenExchangeInput) (*LiheAccessToken, error) {
	if r.forceExchangeUsed || r.code == nil || r.code.Used || r.code.CodeHash != input.CodeHash {
		return nil, ErrLiheInvalidGrant
	}
	copyInput := input
	copyInput.Scopes = append([]string(nil), input.Scopes...)
	copyInput.Bindings = append([]LiheTokenBindingInput(nil), input.Bindings...)
	r.exchangeInput = &copyInput
	r.code.Used = true
	now := time.Now().UTC()
	binding := input.Bindings[0]
	apiKeyID := binding.APIKeyID
	apiKeyName := ""
	if apiKey := r.apiKeys[binding.APIKeyID]; apiKey != nil {
		apiKeyName = apiKey.Name
	}
	return &LiheAccessToken{
		ID:         7,
		UserID:     input.UserID,
		Providers:  []string{binding.Provider},
		APIKeyID:   &apiKeyID,
		APIKeyName: apiKeyName,
		CreatedAt:  now,
	}, nil
}

func (r *liheOAuthTestAPIKeyRepo) ListLiheAccessTokens(_ context.Context, _ int64) ([]LiheAccessToken, error) {
	return append([]LiheAccessToken(nil), r.listTokens...), nil
}

func (r *liheOAuthTestAPIKeyRepo) RevokeLiheAccessTokenByID(_ context.Context, tokenID, userID int64) (bool, error) {
	r.revokedTokenID = tokenID
	r.revokedUserID = userID
	return tokenID == 7 && userID == 11, nil
}

func (r *liheOAuthTestAPIKeyRepo) RevokeLiheAccessTokenByIDAsAdmin(_ context.Context, tokenID int64) (bool, error) {
	r.revokedTokenID = tokenID
	return tokenID == 7, nil
}

func (r *liheOAuthTestAPIKeyRepo) RevokeLiheAccessTokenByHash(_ context.Context, tokenHash, clientID string) (bool, error) {
	r.revokedTokenHash = tokenHash
	r.revokedClientID = clientID
	return true, nil
}

func (r *liheOAuthTestAPIKeyRepo) ResolveLiheAccessToken(_ context.Context, _, _, _ string) (*LiheResolvedAccess, error) {
	return r.resolved, nil
}

type liheOAuthTestUserRepo struct {
	UserRepository
	users map[int64]*User
}

func (r *liheOAuthTestUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	copyUser := *user
	return &copyUser, nil
}

type liheOAuthTestGroupRepo struct {
	GroupRepository
	groups []Group
}

func (r *liheOAuthTestGroupRepo) ListActive(_ context.Context) ([]Group, error) {
	return append([]Group(nil), r.groups...), nil
}

type liheOAuthTestSubscriptionRepo struct {
	UserSubscriptionRepository
}

func (r *liheOAuthTestSubscriptionRepo) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return []UserSubscription{}, nil
}

func newLiheOAuthTestService(repo *liheOAuthTestAPIKeyRepo) *APIKeyService {
	if repo.apiKeys == nil {
		groupID := int64(101)
		repo.apiKeys = map[int64]*APIKey{
			501: {
				ID:      501,
				UserID:  11,
				Key:     "sk-selected-key",
				Name:    "Selected key",
				GroupID: &groupID,
				Status:  StatusAPIKeyActive,
				User:    &User{ID: 11, Email: "user@example.com", Status: StatusActive},
				Group: &Group{
					ID:                 groupID,
					Platform:           PlatformOpenAI,
					Status:             StatusActive,
					ActiveAccountCount: 1,
				},
			},
		}
	}
	cfg := &config.Config{
		LiheOAuth: config.LiheOAuthConfig{
			Enabled:      true,
			ClientID:     "lihe-chat",
			ClientSecret: "0123456789abcdef0123456789abcdef",
			RedirectURI:  "https://lihe.chat/api/integrations/lihe/callback",
			ConnectURL:   "https://lihe.chat/connect/lihe",
		},
	}
	return NewAPIKeyService(
		repo,
		&liheOAuthTestUserRepo{users: map[int64]*User{
			11: {ID: 11, Email: "user@example.com", Status: StatusActive},
		}},
		&liheOAuthTestGroupRepo{groups: []Group{
			{ID: 101, Platform: PlatformOpenAI, Status: StatusActive, ActiveAccountCount: 1},
			{ID: 102, Platform: PlatformAnthropic, Status: StatusActive, ActiveAccountCount: 1},
		}},
		&liheOAuthTestSubscriptionRepo{},
		nil,
		nil,
		cfg,
	)
}

func liheOAuthTestVerifier() string {
	return base64.RawURLEncoding.EncodeToString(make([]byte, 32))
}

func liheOAuthTestChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func liheOAuthTestAuthorizeRequest(verifier string) LiheAuthorizeRequest {
	return LiheAuthorizeRequest{
		APIKeyID:            501,
		ResponseType:        "code",
		ClientID:            "lihe-chat",
		RedirectURI:         "https://lihe.chat/api/integrations/lihe/callback",
		Scope:               "chat:write models:read",
		State:               "0123456789abcdef0123456789abcdef",
		CodeChallenge:       liheOAuthTestChallenge(verifier),
		CodeChallengeMethod: "S256",
	}
}

func issueLiheOAuthTestCode(t *testing.T, svc *APIKeyService, verifier string) string {
	t.Helper()
	result, err := svc.CreateLiheAuthorizationCode(
		context.Background(),
		11,
		liheOAuthTestAuthorizeRequest(verifier),
	)
	require.NoError(t, err)
	callback, err := url.Parse(result.RedirectTo)
	require.NoError(t, err)
	require.Equal(t, "0123456789abcdef0123456789abcdef", callback.Query().Get("state"))
	return callback.Query().Get("code")
}

func TestLiheOAuthAuthorizationCodeIsHashedAndExpiresIn60Seconds(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{}
	svc := newLiheOAuthTestService(repo)
	before := time.Now()
	plainCode := issueLiheOAuthTestCode(t, svc, liheOAuthTestVerifier())

	require.NotEmpty(t, plainCode)
	require.NotNil(t, repo.code)
	require.NotEqual(t, plainCode, repo.code.CodeHash)
	require.Equal(t, hashLiheCredential(plainCode), repo.code.CodeHash)
	require.Equal(t, int64(501), repo.code.APIKeyID)
	require.WithinDuration(t, before.Add(60*time.Second), repo.code.ExpiresAt, time.Second)
}

func TestLiheOAuthExchangeUsesPKCEAndRejectsReplay(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{}
	svc := newLiheOAuthTestService(repo)
	verifier := liheOAuthTestVerifier()
	plainCode := issueLiheOAuthTestCode(t, svc, verifier)

	_, err := svc.ExchangeLiheAuthorizationCode(
		context.Background(),
		plainCode,
		"https://lihe.chat/api/integrations/lihe/callback",
		stringsOfLength('x', 43),
	)
	require.ErrorIs(t, err, ErrLiheInvalidGrant)
	require.False(t, repo.code.Used)

	result, err := svc.ExchangeLiheAuthorizationCode(
		context.Background(),
		plainCode,
		"https://lihe.chat/api/integrations/lihe/callback",
		verifier,
	)
	require.NoError(t, err)
	require.True(t, repo.code.Used)
	require.True(t, strings.HasPrefix(result.AccessToken, LiheAccessTokenPrefix))
	require.Equal(t, []string{PlatformOpenAI}, result.Providers)
	require.Equal(t, int64(501), result.APIKeyID)
	require.Equal(t, "Selected key", result.APIKeyName)
	require.NotNil(t, repo.exchangeInput)
	require.Equal(t, hashLiheCredential(result.AccessToken), repo.exchangeInput.TokenHash)
	require.NotEqual(t, result.AccessToken, repo.exchangeInput.TokenHash)
	require.Len(t, repo.exchangeInput.Bindings, 1)
	require.Equal(t, int64(501), repo.exchangeInput.Bindings[0].APIKeyID)
	require.Equal(t, int64(101), repo.exchangeInput.Bindings[0].GroupID)
	require.Equal(t, PlatformOpenAI, repo.exchangeInput.Bindings[0].Provider)
	encodedResult, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(encodedResult), "sk-selected-key")
	encodedInput, err := json.Marshal(repo.exchangeInput)
	require.NoError(t, err)
	require.NotContains(t, string(encodedInput), "sk-selected-key")

	_, err = svc.ExchangeLiheAuthorizationCode(
		context.Background(),
		plainCode,
		"https://lihe.chat/api/integrations/lihe/callback",
		verifier,
	)
	require.ErrorIs(t, err, ErrLiheInvalidGrant)
}

func TestLiheOAuthExchangeRejectsKeyDisabledAfterAuthorization(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{}
	svc := newLiheOAuthTestService(repo)
	verifier := liheOAuthTestVerifier()
	plainCode := issueLiheOAuthTestCode(t, svc, verifier)
	repo.apiKeys[501].Status = StatusAPIKeyDisabled

	_, err := svc.ExchangeLiheAuthorizationCode(
		context.Background(),
		plainCode,
		"https://lihe.chat/api/integrations/lihe/callback",
		verifier,
	)

	require.ErrorIs(t, err, ErrLiheAPIKeyUnavailable)
	require.False(t, repo.code.Used)
	require.Nil(t, repo.exchangeInput)
}

func TestLiheOAuthExchangeRejectsExpiredCode(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{}
	svc := newLiheOAuthTestService(repo)
	verifier := liheOAuthTestVerifier()
	plainCode := issueLiheOAuthTestCode(t, svc, verifier)
	repo.code.ExpiresAt = time.Now().Add(-time.Second)

	_, err := svc.ExchangeLiheAuthorizationCode(
		context.Background(),
		plainCode,
		"https://lihe.chat/api/integrations/lihe/callback",
		verifier,
	)
	require.ErrorIs(t, err, ErrLiheInvalidGrant)
	require.False(t, repo.code.Used)
}

func TestLiheOAuthAuthorizationRejectsAPIKeyOwnedByAnotherUser(t *testing.T) {
	groupID := int64(101)
	repo := &liheOAuthTestAPIKeyRepo{apiKeys: map[int64]*APIKey{
		502: {
			ID:      502,
			UserID:  12,
			Name:    "Another user's key",
			GroupID: &groupID,
			Status:  StatusAPIKeyActive,
			Group: &Group{
				ID:                 groupID,
				Platform:           PlatformOpenAI,
				Status:             StatusActive,
				ActiveAccountCount: 1,
			},
		},
	}}
	svc := newLiheOAuthTestService(repo)
	request := liheOAuthTestAuthorizeRequest(liheOAuthTestVerifier())
	request.APIKeyID = 502

	_, err := svc.CreateLiheAuthorizationCode(context.Background(), 11, request)

	require.ErrorIs(t, err, ErrLiheAPIKeyUnavailable)
	require.Nil(t, repo.code)
}

func TestLiheOAuthAuthorizationRejectsUnavailableAPIKey(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*APIKey)
	}{
		{
			name: "disabled",
			mutate: func(key *APIKey) {
				key.Status = StatusAPIKeyDisabled
			},
		},
		{
			name: "without group",
			mutate: func(key *APIKey) {
				key.GroupID = nil
				key.Group = nil
			},
		},
		{
			name: "unsupported provider",
			mutate: func(key *APIKey) {
				key.Group.Platform = "unsupported"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &liheOAuthTestAPIKeyRepo{}
			svc := newLiheOAuthTestService(repo)
			test.mutate(repo.apiKeys[501])

			_, err := svc.CreateLiheAuthorizationCode(
				context.Background(),
				11,
				liheOAuthTestAuthorizeRequest(liheOAuthTestVerifier()),
			)

			require.Error(t, err)
			require.Nil(t, repo.code)
		})
	}
}

func TestLiheOAuthRejectsCrossUserBinding(t *testing.T) {
	groupID := int64(101)
	repo := &liheOAuthTestAPIKeyRepo{
		resolved: &LiheResolvedAccess{
			TokenID:       7,
			TokenUserID:   11,
			Scopes:        liheOAuthScopeList(),
			BindingFound:  true,
			BindingGroupID: groupID,
			APIKey: &APIKey{
				ID:      9,
				UserID:  12,
				GroupID: &groupID,
				User:    &User{ID: 12, Status: StatusActive},
				Group:   &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive},
			},
		},
	}
	svc := newLiheOAuthTestService(repo)

	_, err := svc.ResolveLiheAccessToken(
		context.Background(),
		LiheAccessTokenPrefix+"token",
		PlatformOpenAI,
		http.MethodGet,
		"/v1/models",
	)
	require.ErrorIs(t, err, ErrLiheInvalidToken)
}

func TestLiheOAuthRejectsChangedOrDeletedBoundAPIKey(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{resolved: &LiheResolvedAccess{
		TokenID:       7,
		TokenUserID:   11,
		Scopes:        liheOAuthScopeList(),
		BindingFound:  true,
		BindingGroupID: 101,
		APIKey:        nil,
	}}
	svc := newLiheOAuthTestService(repo)

	_, err := svc.ResolveLiheAccessToken(
		context.Background(),
		LiheAccessTokenPrefix+"token",
		PlatformOpenAI,
		http.MethodGet,
		"/v1/models",
	)

	require.ErrorIs(t, err, ErrLiheInvalidToken)
}

func TestLiheOAuthRejectsProviderNotBoundToSelectedKey(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{resolved: &LiheResolvedAccess{
		TokenID:     7,
		TokenUserID: 11,
		Scopes:      liheOAuthScopeList(),
	}}
	svc := newLiheOAuthTestService(repo)

	_, err := svc.ResolveLiheAccessToken(
		context.Background(),
		LiheAccessTokenPrefix+"token",
		PlatformAnthropic,
		http.MethodGet,
		"/v1/models",
	)

	require.ErrorIs(t, err, ErrLiheProviderNotAllowed)
}

func TestLiheOAuthGatewayScopeAllowlist(t *testing.T) {
	tests := []struct {
		method string
		path   string
		scope  string
		ok     bool
	}{
		{http.MethodGet, "/v1/models", LiheOAuthScopeModelsRead, true},
		{http.MethodPost, "/v1/chat/completions", LiheOAuthScopeChatWrite, true},
		{http.MethodPost, "/v1/messages", LiheOAuthScopeChatWrite, true},
		{http.MethodPost, "/v1/messages/count_tokens", LiheOAuthScopeChatWrite, true},
		{http.MethodPost, "/v1/responses", LiheOAuthScopeChatWrite, true},
		{http.MethodPost, "/v1/responses/compact", LiheOAuthScopeChatWrite, true},
		{http.MethodGet, "/v1/responses", LiheOAuthScopeChatWrite, true},
		{http.MethodPost, "/v1/alpha/search", LiheOAuthScopeChatWrite, true},
		{http.MethodGet, "/v1/usage", "", false},
		{http.MethodPost, "/v1/images/generations", "", false},
		{http.MethodGet, "/api/v1/user/profile", "", false},
		{http.MethodPost, "/oauth/token", "", false},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			scope, ok := LiheRequiredScope(test.method, test.path)
			require.Equal(t, test.ok, ok)
			require.Equal(t, test.scope, scope)
		})
	}
}

func TestLiheOAuthClientAuthenticationAndReservedKeyPrefix(t *testing.T) {
	svc := newLiheOAuthTestService(&liheOAuthTestAPIKeyRepo{})
	require.True(t, svc.AuthenticateLiheOAuthClient("lihe-chat", "0123456789abcdef0123456789abcdef"))
	require.False(t, svc.AuthenticateLiheOAuthClient("lihe-chat", "wrong"))
	require.False(t, svc.AuthenticateLiheOAuthClient("wrong", "0123456789abcdef0123456789abcdef"))
	require.ErrorIs(t, svc.ValidateCustomKey(LiheAccessTokenPrefix+stringsOfLength('a', 43)), ErrAPIKeyReserved)
	require.ErrorIs(t, svc.ValidateCustomKey(LiheInternalAPIKeyPrefix+stringsOfLength('a', 43)), ErrAPIKeyReserved)
}

func TestLiheOAuthUserCanOnlyRevokeOwnToken(t *testing.T) {
	repo := &liheOAuthTestAPIKeyRepo{}
	svc := newLiheOAuthTestService(repo)
	require.NoError(t, svc.RevokeLiheAccessTokenByID(context.Background(), 7, 11))
	require.Equal(t, int64(7), repo.revokedTokenID)
	require.Equal(t, int64(11), repo.revokedUserID)

	err := svc.RevokeLiheAccessTokenByID(context.Background(), 7, 12)
	require.ErrorIs(t, err, ErrLiheTokenNotFound)
}

func stringsOfLength(char byte, length int) string {
	value := make([]byte, length)
	for i := range value {
		value[i] = char
	}
	return string(value)
}

var _ LiheOAuthRepository = (*liheOAuthTestAPIKeyRepo)(nil)
var _ APIKeyRepository = (*liheOAuthTestAPIKeyRepo)(nil)
var _ UserRepository = (*liheOAuthTestUserRepo)(nil)
var _ GroupRepository = (*liheOAuthTestGroupRepo)(nil)
var _ UserSubscriptionRepository = (*liheOAuthTestSubscriptionRepo)(nil)

func TestLiheOAuthDisabled(t *testing.T) {
	svc := newLiheOAuthTestService(&liheOAuthTestAPIKeyRepo{})
	svc.cfg.LiheOAuth.Enabled = false
	_, err := svc.CreateLiheAuthorizationCode(context.Background(), 11, liheOAuthTestAuthorizeRequest(liheOAuthTestVerifier()))
	require.True(t, errors.Is(err, ErrLiheOAuthDisabled))
}
