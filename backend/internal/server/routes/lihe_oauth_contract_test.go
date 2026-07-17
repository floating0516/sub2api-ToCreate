package routes

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type liheContractRepository struct {
	service.APIKeyRepository
	apiKey    *service.APIKey
	code      *service.LiheAuthorizationCode
	tokenHash string
}

func (r *liheContractRepository) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	if r.apiKey == nil || r.apiKey.ID != id {
		return nil, service.ErrAPIKeyNotFound
	}
	return cloneLiheContractAPIKey(r.apiKey), nil
}

func (r *liheContractRepository) UpdateLastUsed(context.Context, int64, time.Time) error {
	return nil
}

func (r *liheContractRepository) CreateLiheAuthorizationCode(_ context.Context, code *service.LiheAuthorizationCode) error {
	r.code = code
	return nil
}

func (r *liheContractRepository) GetLiheAuthorizationCode(_ context.Context, codeHash string) (*service.LiheAuthorizationCode, error) {
	if r.code == nil || r.code.CodeHash != codeHash {
		return nil, nil
	}
	return r.code, nil
}

func (r *liheContractRepository) ExchangeLiheAuthorizationCode(
	_ context.Context,
	input service.LiheTokenExchangeInput,
) (*service.LiheAccessToken, error) {
	if r.code == nil || r.code.Used || input.CodeHash != r.code.CodeHash || len(input.Bindings) != 1 {
		return nil, service.ErrLiheInvalidGrant
	}
	r.code.Used = true
	r.tokenHash = input.TokenHash
	apiKeyID := input.Bindings[0].APIKeyID
	return &service.LiheAccessToken{
		ID:         7,
		UserID:     input.UserID,
		Providers:  []string{input.Bindings[0].Provider},
		APIKeyID:   &apiKeyID,
		APIKeyName: r.apiKey.Name,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func (r *liheContractRepository) ListLiheAccessTokens(context.Context, int64) ([]service.LiheAccessToken, error) {
	return nil, nil
}

func (r *liheContractRepository) RevokeLiheAccessTokenByID(context.Context, int64, int64) (bool, error) {
	return false, nil
}

func (r *liheContractRepository) RevokeLiheAccessTokenByIDAsAdmin(context.Context, int64) (bool, error) {
	return false, nil
}

func (r *liheContractRepository) RevokeLiheAccessTokenByHash(context.Context, string, string) (bool, error) {
	return false, nil
}

func (r *liheContractRepository) ResolveLiheAccessToken(
	_ context.Context,
	tokenHash, clientID string,
) (*service.LiheResolvedAccess, error) {
	if r.tokenHash == "" || tokenHash != r.tokenHash || clientID != "lihe-chat" {
		return nil, nil
	}
	return &service.LiheResolvedAccess{
		TokenID:         7,
		TokenUserID:     r.apiKey.UserID,
		Scopes:          []string{service.LiheOAuthScopeModelsRead, service.LiheOAuthScopeChatWrite},
		BindingFound:    true,
		BindingProvider: r.apiKey.Group.Platform,
		BindingGroupID:  r.apiKey.Group.ID,
		APIKey:          cloneLiheContractAPIKey(r.apiKey),
	}, nil
}

type liheContractUserRepository struct {
	service.UserRepository
	user *service.User
}

func (r *liheContractUserRepository) GetByID(_ context.Context, id int64) (*service.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, service.ErrUserNotFound
	}
	copyUser := *r.user
	return &copyUser, nil
}

type liheContractGroupRepository struct {
	service.GroupRepository
	group *service.Group
}

func (r *liheContractGroupRepository) ListActive(context.Context) ([]service.Group, error) {
	return []service.Group{*r.group}, nil
}

type liheContractSubscriptionRepository struct {
	service.UserSubscriptionRepository
}

func (r *liheContractSubscriptionRepository) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, nil
}

type liheContractAccountRepository struct {
	service.AccountRepository
}

func (r *liheContractAccountRepository) ListSchedulableByGroupID(context.Context, int64) ([]service.Account, error) {
	return nil, nil
}

type libreChatTokenResponseSchema struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	Providers   []string  `json:"providers"`
	APIKeyID    int64     `json:"api_key_id"`
	APIKeyName  string    `json:"api_key_name"`
	CreatedAt   time.Time `json:"created_at"`
}

func TestLiheOAuthLibreChatContractEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	group := &service.Group{
		ID:                 101,
		Name:               "OpenAI",
		Platform:           service.PlatformOpenAI,
		Status:             service.StatusActive,
		Hydrated:           true,
		ActiveAccountCount: 1,
	}
	user := &service.User{
		ID:          11,
		Email:       "lihe-contract@example.test",
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Balance:     10,
		Concurrency: 1,
	}
	apiKey := &service.APIKey{
		ID:      501,
		UserID:  user.ID,
		Key:     "sk-lihe-contract-source",
		Name:    "Contract source key",
		GroupID: &group.ID,
		Status:  service.StatusAPIKeyActive,
		User:    user,
		Group:   group,
	}

	const (
		clientID     = "lihe-chat"
		clientSecret = "0123456789abcdef0123456789abcdef"
		redirectURI  = "https://lihe.chat/api/integrations/lihe/callback"
	)
	verifier := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	challengeDigest := sha256.Sum256([]byte(verifier))
	code := "lihe-contract-authorization-code"
	repo := &liheContractRepository{
		apiKey: apiKey,
		code: &service.LiheAuthorizationCode{
			UserID:              user.ID,
			APIKeyID:            apiKey.ID,
			CodeHash:            liheContractHash(code),
			ClientID:            clientID,
			RedirectURI:         redirectURI,
			Scopes:              []string{service.LiheOAuthScopeModelsRead, service.LiheOAuthScopeChatWrite},
			CodeChallenge:       base64.RawURLEncoding.EncodeToString(challengeDigest[:]),
			CodeChallengeMethod: "S256",
			ExpiresAt:           time.Now().Add(time.Minute),
		},
	}
	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		LiheOAuth: config.LiheOAuthConfig{
			Enabled:      true,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  redirectURI,
			ConnectURL:   "https://lihe.chat/connect/lihe",
		},
	}
	apiKeyService := service.NewAPIKeyService(
		repo,
		&liheContractUserRepository{user: user},
		&liheContractGroupRepository{group: group},
		&liheContractSubscriptionRepository{},
		nil,
		nil,
		cfg,
	)
	gatewayService := service.NewGatewayService(
		&liheContractAccountRepository{}, nil, nil, nil, nil, nil, nil, nil, cfg,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil,
	)
	gatewayHandler := handler.NewGatewayHandler(
		gatewayService, nil, nil, nil, nil, nil, nil, nil, apiKeyService, nil, nil,
		nil, nil, cfg, nil,
	)
	handlers := &handler.Handlers{
		APIKey:        handler.NewAPIKeyHandler(apiKeyService),
		Gateway:       gatewayHandler,
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
		AsyncImage:    handler.NewAsyncImageHandler(nil, nil),
	}
	router := gin.New()
	router.POST("/oauth/token", handlers.APIKey.ExchangeLiheOAuthToken)
	RegisterGatewayRoutes(
		router,
		handlers,
		servermiddleware.NewAPIKeyAuthMiddleware(apiKeyService, nil, cfg),
		apiKeyService,
		nil,
		nil,
		nil,
		cfg,
	)

	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	tokenRequest := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenRequest.SetBasicAuth(clientID, clientSecret)
	tokenRecorder := httptest.NewRecorder()

	router.ServeHTTP(tokenRecorder, tokenRequest)

	require.Equal(t, http.StatusOK, tokenRecorder.Code)
	tokenResponse := requireLibreChatTokenResponse(t, tokenRecorder)

	modelsRequest := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	modelsRequest.Header.Set("Authorization", "Bearer "+tokenResponse.AccessToken)
	modelsRecorder := httptest.NewRecorder()

	router.ServeHTTP(modelsRecorder, modelsRequest)

	require.Equal(t, http.StatusOK, modelsRecorder.Code)
	require.Empty(t, modelsRequest.Header.Get("X-Lihe-Provider"))
	var modelsResponse struct {
		Object string            `json:"object"`
		Data   []json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(modelsRecorder.Body.Bytes(), &modelsResponse))
	require.Equal(t, "list", modelsResponse.Object)
	require.NotEmpty(t, modelsResponse.Data)
}

func requireLibreChatTokenResponse(t *testing.T, recorder *httptest.ResponseRecorder) libreChatTokenResponseSchema {
	t.Helper()
	var response libreChatTokenResponseSchema
	decoder := json.NewDecoder(recorder.Body)
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&response))
	require.True(t, strings.HasPrefix(response.AccessToken, service.LiheAccessTokenPrefix))
	require.Equal(t, "Bearer", response.TokenType)
	require.Equal(t, service.LiheOAuthScopes, response.Scope)
	require.Len(t, response.Providers, 1)
	require.Contains(t, map[string]struct{}{
		service.LiheOAuthProviderOpenAI:    {},
		service.LiheOAuthProviderAnthropic: {},
		service.LiheOAuthProviderGoogle:    {},
	}, response.Providers[0])
	require.Equal(t, service.LiheOAuthProviderOpenAI, response.Providers[0])
	require.Positive(t, response.APIKeyID)
	require.NotEmpty(t, response.APIKeyName)
	require.False(t, response.CreatedAt.IsZero())
	return response
}

func cloneLiheContractAPIKey(apiKey *service.APIKey) *service.APIKey {
	copyKey := *apiKey
	copyUser := *apiKey.User
	copyGroup := *apiKey.Group
	copyKey.User = &copyUser
	copyKey.Group = &copyGroup
	return &copyKey
}

func liheContractHash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

var _ service.LiheOAuthRepository = (*liheContractRepository)(nil)
