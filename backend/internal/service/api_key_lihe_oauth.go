package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	LiheOAuthClientName          = "lihe.chat"
	LiheOAuthScopeModelsRead     = "models:read"
	LiheOAuthScopeChatWrite      = "chat:write"
	LiheOAuthScopes              = LiheOAuthScopeModelsRead + " " + LiheOAuthScopeChatWrite
	LiheAccessTokenPrefix        = "lihe_"
	LiheInternalAPIKeyPrefix     = "lihe-internal-"
	liheAuthorizationCodeTTL     = 60 * time.Second
	lihePKCEChallengeMethod      = "S256"
	liheOAuthRandomCredentialLen = 32
)

var (
	ErrLiheOAuthDisabled  = infraerrors.NotFound("LIHE_OAUTH_DISABLED", "Lihe Chat integration is not enabled")
	ErrLiheInvalidRequest = infraerrors.BadRequest(
		"LIHE_OAUTH_INVALID_REQUEST",
		"invalid Lihe OAuth request",
	)
	ErrLiheInvalidGrant = infraerrors.Unauthorized(
		"LIHE_OAUTH_INVALID_GRANT",
		"authorization code is invalid or expired",
	)
	ErrLiheInvalidToken = infraerrors.Unauthorized(
		"LIHE_OAUTH_INVALID_TOKEN",
		"Lihe access token is invalid or revoked",
	)
	ErrLiheProviderRequired = infraerrors.BadRequest(
		"LIHE_PROVIDER_REQUIRED",
		"X-Lihe-Provider is required for Lihe access tokens",
	)
	ErrLiheProviderNotAllowed = infraerrors.Forbidden(
		"LIHE_PROVIDER_NOT_ALLOWED",
		"the requested provider is not available for this Lihe connection",
	)
	ErrLiheScopeNotAllowed = infraerrors.Forbidden(
		"LIHE_SCOPE_NOT_ALLOWED",
		"Lihe access tokens cannot access this endpoint",
	)
	ErrLiheNoProviders = infraerrors.Forbidden(
		"LIHE_NO_PROVIDERS",
		"this account has no available model providers",
	)
	ErrLiheTokenNotFound = infraerrors.NotFound(
		"LIHE_TOKEN_NOT_FOUND",
		"Lihe connection not found",
	)
	ErrLiheOAuthRepositoryUnavailable = infraerrors.New(
		http.StatusServiceUnavailable,
		"LIHE_OAUTH_UNAVAILABLE",
		"Lihe Chat integration is temporarily unavailable",
	)

	lihePKCEValuePattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)
	liheProviderOrder    = []string{
		PlatformOpenAI,
		PlatformAnthropic,
		PlatformGemini,
		PlatformAntigravity,
		PlatformGrok,
	}
)

type LiheAuthorizationCode struct {
	ID                  int64
	UserID              int64
	CodeHash            string
	ClientID            string
	RedirectURI         string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
	Used                bool
	UsedAt              *time.Time
	CreatedAt           time.Time
}

type LiheAuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type LiheAuthorizeResult struct {
	RedirectTo string `json:"redirect_to"`
	ExpiresIn  int    `json:"expires_in"`
}

type LiheTokenBindingInput struct {
	Provider string
	GroupID  int64
	APIKey   string
}

type LiheTokenExchangeInput struct {
	CodeHash      string
	UserID        int64
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	TokenHash     string
	Scopes        []string
	Bindings      []LiheTokenBindingInput
}

type LiheTokenExchangeResult struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	Providers   []string  `json:"providers"`
	CreatedAt   time.Time `json:"created_at"`
}

type LiheAccessToken struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"-"`
	Name       string     `json:"name"`
	ClientID   string     `json:"-"`
	Scopes     []string   `json:"scopes"`
	Providers  []string   `json:"providers"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
}

type LiheResolvedAccess struct {
	TokenID     int64
	TokenUserID int64
	Scopes      []string
	APIKey      *APIKey
}

// LiheOAuthRepository is implemented by the concrete API-key repository. It
// stays separate from APIKeyRepository so existing service and handler stubs do
// not need to implement integration-only persistence methods.
type LiheOAuthRepository interface {
	CreateLiheAuthorizationCode(ctx context.Context, code *LiheAuthorizationCode) error
	GetLiheAuthorizationCode(ctx context.Context, codeHash string) (*LiheAuthorizationCode, error)
	ExchangeLiheAuthorizationCode(ctx context.Context, input LiheTokenExchangeInput) (*LiheAccessToken, error)
	ListLiheAccessTokens(ctx context.Context, userID int64) ([]LiheAccessToken, error)
	RevokeLiheAccessTokenByID(ctx context.Context, tokenID, userID int64) (bool, error)
	RevokeLiheAccessTokenByIDAsAdmin(ctx context.Context, tokenID int64) (bool, error)
	RevokeLiheAccessTokenByHash(ctx context.Context, tokenHash, clientID string) (bool, error)
	ResolveLiheAccessToken(ctx context.Context, tokenHash, clientID, provider string) (*LiheResolvedAccess, error)
}

func (s *APIKeyService) liheOAuthRepository() (LiheOAuthRepository, error) {
	repo, ok := s.apiKeyRepo.(LiheOAuthRepository)
	if !ok || repo == nil {
		return nil, ErrLiheOAuthRepositoryUnavailable
	}
	return repo, nil
}

func (s *APIKeyService) liheOAuthEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.LiheOAuth.Enabled
}

func (s *APIKeyService) LiheOAuthPublicConfig() (enabled bool, connectURL string) {
	if !s.liheOAuthEnabled() {
		return false, ""
	}
	return true, s.cfg.LiheOAuth.ConnectURL
}

func (s *APIKeyService) AuthenticateLiheOAuthClient(clientID, clientSecret string) bool {
	if !s.liheOAuthEnabled() {
		return false
	}
	configuredID := sha256.Sum256([]byte(s.cfg.LiheOAuth.ClientID))
	configuredSecret := sha256.Sum256([]byte(s.cfg.LiheOAuth.ClientSecret))
	providedID := sha256.Sum256([]byte(clientID))
	providedSecret := sha256.Sum256([]byte(clientSecret))
	return subtle.ConstantTimeCompare(configuredID[:], providedID[:]) == 1 &&
		subtle.ConstantTimeCompare(configuredSecret[:], providedSecret[:]) == 1
}

func (s *APIKeyService) CreateLiheAuthorizationCode(
	ctx context.Context,
	userID int64,
	req LiheAuthorizeRequest,
) (*LiheAuthorizeResult, error) {
	if !s.liheOAuthEnabled() {
		return nil, ErrLiheOAuthDisabled
	}
	if err := s.validateLiheAuthorizeRequest(req); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get Lihe OAuth user: %w", err)
	}
	if user == nil || !user.IsActive() {
		return nil, infraerrors.Forbidden("LIHE_USER_INACTIVE", "user account is not active")
	}

	plainCode, err := generateLiheCredential("")
	if err != nil {
		return nil, fmt.Errorf("generate Lihe authorization code: %w", err)
	}
	now := time.Now().UTC()
	record := &LiheAuthorizationCode{
		UserID:              userID,
		CodeHash:            hashLiheCredential(plainCode),
		ClientID:            s.cfg.LiheOAuth.ClientID,
		RedirectURI:         s.cfg.LiheOAuth.RedirectURI,
		Scopes:              liheOAuthScopeList(),
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: lihePKCEChallengeMethod,
		ExpiresAt:           now.Add(liheAuthorizationCodeTTL),
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return nil, err
	}
	if err := repo.CreateLiheAuthorizationCode(ctx, record); err != nil {
		return nil, fmt.Errorf("create Lihe authorization code: %w", err)
	}

	callback, err := url.Parse(s.cfg.LiheOAuth.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("parse configured Lihe redirect URI: %w", err)
	}
	query := callback.Query()
	query.Set("code", plainCode)
	query.Set("state", req.State)
	callback.RawQuery = query.Encode()
	return &LiheAuthorizeResult{
		RedirectTo: callback.String(),
		ExpiresIn:  int(liheAuthorizationCodeTTL / time.Second),
	}, nil
}

func (s *APIKeyService) validateLiheAuthorizeRequest(req LiheAuthorizeRequest) error {
	if req.ResponseType != "code" ||
		req.ClientID != s.cfg.LiheOAuth.ClientID ||
		req.RedirectURI != s.cfg.LiheOAuth.RedirectURI ||
		req.CodeChallengeMethod != lihePKCEChallengeMethod ||
		!lihePKCEValuePattern.MatchString(req.CodeChallenge) ||
		len(req.State) < 16 || len(req.State) > 1024 || strings.ContainsAny(req.State, "\r\n") {
		return ErrLiheInvalidRequest
	}
	if !equalLiheScopes(req.Scope) {
		return infraerrors.BadRequest("LIHE_OAUTH_INVALID_SCOPE", "requested scopes are not allowed")
	}
	return nil
}

func (s *APIKeyService) ExchangeLiheAuthorizationCode(
	ctx context.Context,
	code, redirectURI, codeVerifier string,
) (*LiheTokenExchangeResult, error) {
	if !s.liheOAuthEnabled() {
		return nil, ErrLiheOAuthDisabled
	}
	if code == "" || redirectURI != s.cfg.LiheOAuth.RedirectURI || !lihePKCEValuePattern.MatchString(codeVerifier) {
		return nil, ErrLiheInvalidGrant
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return nil, err
	}
	codeHash := hashLiheCredential(code)
	record, err := repo.GetLiheAuthorizationCode(ctx, codeHash)
	if err != nil {
		return nil, fmt.Errorf("get Lihe authorization code: %w", err)
	}
	if record == nil || record.Used || !record.ExpiresAt.After(time.Now()) ||
		record.ClientID != s.cfg.LiheOAuth.ClientID || record.RedirectURI != redirectURI ||
		record.CodeChallengeMethod != lihePKCEChallengeMethod || !equalLiheScopeSlices(record.Scopes) {
		return nil, ErrLiheInvalidGrant
	}
	verifierDigest := sha256.Sum256([]byte(codeVerifier))
	challenge := base64.RawURLEncoding.EncodeToString(verifierDigest[:])
	if subtle.ConstantTimeCompare([]byte(challenge), []byte(record.CodeChallenge)) != 1 {
		return nil, ErrLiheInvalidGrant
	}

	user, err := s.userRepo.GetByID(ctx, record.UserID)
	if err != nil {
		return nil, fmt.Errorf("get Lihe token user: %w", err)
	}
	if user == nil || !user.IsActive() {
		return nil, infraerrors.Forbidden("LIHE_USER_INACTIVE", "user account is not active")
	}
	bindings, err := s.buildLiheTokenBindings(ctx, record.UserID)
	if err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, ErrLiheNoProviders
	}

	plainToken, err := generateLiheCredential(LiheAccessTokenPrefix)
	if err != nil {
		return nil, fmt.Errorf("generate Lihe access token: %w", err)
	}
	input := LiheTokenExchangeInput{
		CodeHash:      codeHash,
		UserID:        record.UserID,
		ClientID:      s.cfg.LiheOAuth.ClientID,
		RedirectURI:   redirectURI,
		CodeChallenge: record.CodeChallenge,
		TokenHash:     hashLiheCredential(plainToken),
		Scopes:        liheOAuthScopeList(),
		Bindings:      bindings,
	}
	issued, err := repo.ExchangeLiheAuthorizationCode(ctx, input)
	if err != nil {
		if infraerrors.Reason(err) == infraerrors.Reason(ErrLiheInvalidGrant) {
			return nil, ErrLiheInvalidGrant
		}
		return nil, fmt.Errorf("exchange Lihe authorization code: %w", err)
	}
	providers := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		providers = append(providers, binding.Provider)
	}
	return &LiheTokenExchangeResult{
		AccessToken: plainToken,
		TokenType:   "Bearer",
		Scope:       LiheOAuthScopes,
		Providers:   providers,
		CreatedAt:   issued.CreatedAt,
	}, nil
}

func (s *APIKeyService) buildLiheTokenBindings(ctx context.Context, userID int64) ([]LiheTokenBindingInput, error) {
	groups, err := s.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get Lihe provider groups: %w", err)
	}
	firstGroupByProvider := make(map[string]int64, len(liheProviderOrder))
	for i := range groups {
		if groups[i].ActiveAccountCount <= 0 {
			continue
		}
		provider := strings.ToLower(strings.TrimSpace(groups[i].Platform))
		if !isLiheProvider(provider) {
			continue
		}
		if _, exists := firstGroupByProvider[provider]; !exists {
			firstGroupByProvider[provider] = groups[i].ID
		}
	}
	bindings := make([]LiheTokenBindingInput, 0, len(firstGroupByProvider))
	for _, provider := range liheProviderOrder {
		groupID, ok := firstGroupByProvider[provider]
		if !ok {
			continue
		}
		internalKey, err := generateLiheCredential(LiheInternalAPIKeyPrefix)
		if err != nil {
			return nil, fmt.Errorf("generate internal Lihe provider key: %w", err)
		}
		bindings = append(bindings, LiheTokenBindingInput{
			Provider: provider,
			GroupID:  groupID,
			APIKey:   internalKey,
		})
	}
	return bindings, nil
}

func (s *APIKeyService) ListLiheAccessTokens(ctx context.Context, userID int64) ([]LiheAccessToken, error) {
	if !s.liheOAuthEnabled() {
		return nil, ErrLiheOAuthDisabled
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return nil, err
	}
	tokens, err := repo.ListLiheAccessTokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list Lihe access tokens: %w", err)
	}
	for i := range tokens {
		sort.SliceStable(tokens[i].Providers, func(a, b int) bool {
			return liheProviderRank(tokens[i].Providers[a]) < liheProviderRank(tokens[i].Providers[b])
		})
	}
	return tokens, nil
}

func (s *APIKeyService) RevokeLiheAccessTokenByID(ctx context.Context, tokenID, userID int64) error {
	if !s.liheOAuthEnabled() {
		return ErrLiheOAuthDisabled
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return err
	}
	revoked, err := repo.RevokeLiheAccessTokenByID(ctx, tokenID, userID)
	if err != nil {
		return fmt.Errorf("revoke Lihe access token: %w", err)
	}
	if !revoked {
		return ErrLiheTokenNotFound
	}
	return nil
}

func (s *APIKeyService) RevokeLiheAccessTokenByIDAsAdmin(ctx context.Context, tokenID int64) error {
	if !s.liheOAuthEnabled() {
		return ErrLiheOAuthDisabled
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return err
	}
	revoked, err := repo.RevokeLiheAccessTokenByIDAsAdmin(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("admin revoke Lihe access token: %w", err)
	}
	if !revoked {
		return ErrLiheTokenNotFound
	}
	return nil
}

func (s *APIKeyService) RevokeLiheAccessToken(ctx context.Context, token string) error {
	if !s.liheOAuthEnabled() || !strings.HasPrefix(token, LiheAccessTokenPrefix) {
		return nil
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return err
	}
	_, err = repo.RevokeLiheAccessTokenByHash(ctx, hashLiheCredential(token), s.cfg.LiheOAuth.ClientID)
	if err != nil {
		return fmt.Errorf("revoke Lihe access token: %w", err)
	}
	return nil
}

func (s *APIKeyService) ResolveLiheAccessToken(
	ctx context.Context,
	token, provider, method, path string,
) (*APIKey, error) {
	if !s.liheOAuthEnabled() || !strings.HasPrefix(token, LiheAccessTokenPrefix) {
		return nil, ErrLiheInvalidToken
	}
	requiredScope, allowed := LiheRequiredScope(method, path)
	if !allowed {
		return nil, ErrLiheScopeNotAllowed
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return nil, ErrLiheProviderRequired
	}
	if !isLiheProvider(provider) {
		return nil, ErrLiheProviderNotAllowed
	}
	repo, err := s.liheOAuthRepository()
	if err != nil {
		return nil, err
	}
	resolved, err := repo.ResolveLiheAccessToken(
		ctx,
		hashLiheCredential(token),
		s.cfg.LiheOAuth.ClientID,
		provider,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve Lihe access token: %w", err)
	}
	if resolved == nil {
		return nil, ErrLiheInvalidToken
	}
	if resolved.APIKey == nil {
		return nil, ErrLiheProviderNotAllowed
	}
	if resolved.TokenUserID != resolved.APIKey.UserID || resolved.APIKey.User == nil ||
		resolved.APIKey.User.ID != resolved.TokenUserID {
		return nil, ErrLiheInvalidToken
	}
	if !containsLiheScope(resolved.Scopes, requiredScope) {
		return nil, ErrLiheScopeNotAllowed
	}
	s.compileAPIKeyIPRules(resolved.APIKey)
	return resolved.APIKey, nil
}

func LiheRequiredScope(method, path string) (string, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	switch {
	case method == http.MethodGet && path == "/v1/models":
		return LiheOAuthScopeModelsRead, true
	case method == http.MethodPost && path == "/v1/chat/completions":
		return LiheOAuthScopeChatWrite, true
	case method == http.MethodPost && path == "/v1/messages":
		return LiheOAuthScopeChatWrite, true
	case method == http.MethodPost && path == "/v1/messages/count_tokens":
		return LiheOAuthScopeChatWrite, true
	case method == http.MethodPost && (path == "/v1/responses" || strings.HasPrefix(path, "/v1/responses/")):
		return LiheOAuthScopeChatWrite, true
	case method == http.MethodGet && path == "/v1/responses":
		return LiheOAuthScopeChatWrite, true
	case method == http.MethodPost && path == "/v1/alpha/search":
		return LiheOAuthScopeChatWrite, true
	default:
		return "", false
	}
}

func generateLiheCredential(prefix string) (string, error) {
	random := make([]byte, liheOAuthRandomCredentialLen)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(random), nil
}

func hashLiheCredential(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func liheOAuthScopeList() []string {
	return []string{LiheOAuthScopeModelsRead, LiheOAuthScopeChatWrite}
}

func equalLiheScopes(raw string) bool {
	parts := strings.Fields(raw)
	return equalLiheScopeSlices(parts)
}

func equalLiheScopeSlices(scopes []string) bool {
	if len(scopes) != 2 {
		return false
	}
	return containsLiheScope(scopes, LiheOAuthScopeModelsRead) && containsLiheScope(scopes, LiheOAuthScopeChatWrite)
}

func containsLiheScope(scopes []string, expected string) bool {
	for _, scope := range scopes {
		if scope == expected {
			return true
		}
	}
	return false
}

func isLiheProvider(provider string) bool {
	return liheProviderRank(provider) < len(liheProviderOrder)
}

func liheProviderRank(provider string) int {
	for i, candidate := range liheProviderOrder {
		if provider == candidate {
			return i
		}
	}
	return len(liheProviderOrder)
}
