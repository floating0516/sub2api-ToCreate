package oidcprovider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"
)

const (
	AuthorizationCodeTTL = 60 * time.Second
	AccessTokenTTL       = 5 * time.Minute
	IDTokenTTL           = 5 * time.Minute
	Scope                = "openid profile email"
)

var (
	ErrProviderDisabled      = errors.New("lihe OIDC provider is disabled")
	ErrInvalidRequest        = errors.New("invalid OIDC authorization request")
	ErrInvalidBrowserBinding = errors.New("invalid OIDC browser binding")

	pkceChallengePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43,128}$`)
	opaqueHandlePattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

	oidcScopes                     = fosite.Arguments{"openid", "profile", "email"}
	allowedAuthorizationParameters = map[string]struct{}{
		"response_type":         {},
		"client_id":             {},
		"redirect_uri":          {},
		"scope":                 {},
		"state":                 {},
		"nonce":                 {},
		"code_challenge":        {},
		"code_challenge_method": {},
		"response_mode":         {},
		"prompt":                {},
	}
)

type DiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgsSupported       []string `json:"id_token_signing_alg_values_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
}

type PreparedAuthorization struct {
	RequestID                 string `json:"request_id"`
	ExpiresIn                 int    `json:"expires_in"`
	UnauthenticatedRedirectTo string `json:"unauthenticated_redirect_to,omitempty"`
}

type AuthorizationResult struct {
	RedirectTo     string `json:"redirect_to,omitempty"`
	ExpiresIn      int    `json:"expires_in,omitempty"`
	Reauthenticate bool   `json:"reauthenticate,omitempty"`
}

type Provider struct {
	enabled bool
	config  config.LiheOIDCConfig
	oauth   fosite.OAuth2Provider
	store   *Store
	keys    *KeyRing
}

func NewProvider(db *sql.DB, cfg *config.Config) (*Provider, error) {
	provider := &Provider{}
	if cfg == nil {
		return provider, nil
	}
	provider.enabled = cfg.LiheOIDC.Enabled
	provider.config = cfg.LiheOIDC
	if !provider.enabled {
		return provider, nil
	}

	keys, err := NewKeyRing(cfg.LiheOIDC.KeyDirectory, cfg.LiheOIDC.KeyRotationDays)
	if err != nil {
		return nil, fmt.Errorf("initialize Lihe OIDC signing keys: %w", err)
	}
	fositeConfig := &fosite.Config{
		GlobalSecret:                   []byte(cfg.LiheOIDC.HMACSecret),
		AccessTokenLifespan:            AccessTokenTTL,
		AuthorizeCodeLifespan:          AuthorizationCodeTTL,
		IDTokenLifespan:                IDTokenTTL,
		IDTokenIssuer:                  cfg.LiheOIDC.Issuer,
		AccessTokenIssuer:              cfg.LiheOIDC.Issuer,
		TokenURL:                       cfg.LiheOIDC.Issuer + "/oidc/token",
		ScopeStrategy:                  fosite.ExactScopeStrategy,
		EnforcePKCE:                    true,
		EnforcePKCEForPublicClients:    true,
		EnablePKCEPlainChallengeMethod: false,
		RefreshTokenScopes:             []string{"offline_access"},
		MinParameterEntropy:            16,
		SendDebugMessagesToClients:     false,
		SanitationWhiteList: []string{
			"redirect_uri",
			"code_challenge",
			"code_challenge_method",
			"nonce",
			"prompt",
		},
	}
	hashedSecret, err := (&fosite.BCrypt{Config: fositeConfig}).Hash(context.Background(), []byte(cfg.LiheOIDC.ClientSecret))
	if err != nil {
		return nil, fmt.Errorf("hash Lihe OIDC client secret: %w", err)
	}
	client := &fosite.DefaultOpenIDConnectClient{
		DefaultClient: &fosite.DefaultClient{
			ID:            cfg.LiheOIDC.ClientID,
			Secret:        hashedSecret,
			RedirectURIs:  []string{cfg.LiheOIDC.RedirectURI},
			GrantTypes:    []string{"authorization_code"},
			ResponseTypes: []string{"code"},
			Scopes:        []string(oidcScopes),
			Audience:      []string{cfg.LiheOIDC.ClientID},
			Public:        false,
		},
		TokenEndpointAuthMethod: "client_secret_basic",
	}
	store, err := NewStore(db, client, time.Duration(cfg.LiheOIDC.PendingRequestTTLSeconds)*time.Second, cfg.LiheOIDC.HMACSecret)
	if err != nil {
		return nil, err
	}
	keySigner := &jwt.DefaultSigner{GetPrivateKey: keys.CurrentPrivateKey}
	strategy := &compose.CommonStrategy{
		CoreStrategy:               compose.NewOAuth2HMACStrategy(fositeConfig),
		OpenIDConnectTokenStrategy: compose.NewOpenIDConnectStrategy(keys.CurrentPrivateKey, fositeConfig),
		Signer:                     keySigner,
	}
	provider.oauth = compose.Compose(
		fositeConfig,
		store,
		strategy,
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OpenIDConnectExplicitFactory,
		compose.OAuth2PKCEFactory,
		compose.OAuth2TokenIntrospectionFactory,
	)
	provider.store = store
	provider.keys = keys
	return provider, nil
}

func (p *Provider) Enabled() bool {
	return p != nil && p.enabled
}

func (p *Provider) Discovery() (DiscoveryDocument, error) {
	if !p.Enabled() {
		return DiscoveryDocument{}, ErrProviderDisabled
	}
	issuer := p.config.Issuer
	return DiscoveryDocument{
		Issuer:                            issuer,
		AuthorizationEndpoint:             issuer + "/oidc/authorize",
		TokenEndpoint:                     issuer + "/oidc/token",
		UserInfoEndpoint:                  issuer + "/oidc/userinfo",
		JWKSURI:                           issuer + "/oidc/jwks",
		ScopesSupported:                   []string(oidcScopes),
		ResponseTypesSupported:            []string{"code"},
		ResponseModesSupported:            []string{"query"},
		GrantTypesSupported:               []string{"authorization_code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgsSupported:       []string{"RS256"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		ClaimsSupported: []string{
			"iss", "sub", "aud", "exp", "iat", "nonce",
			"email", "email_verified", "preferred_username", "name",
		},
	}, nil
}

func (p *Provider) JWKS() (*jose.JSONWebKeySet, error) {
	if !p.Enabled() {
		return nil, ErrProviderDisabled
	}
	return p.keys.PublicJWKS()
}

func (p *Provider) PrepareAuthorization(ctx context.Context, params url.Values, browserBinding string) (*PreparedAuthorization, error) {
	if !p.Enabled() {
		return nil, ErrProviderDisabled
	}
	if !opaqueHandlePattern.MatchString(browserBinding) {
		return nil, ErrInvalidBrowserBinding
	}
	if err := p.validateAuthorizationParameters(params); err != nil {
		return nil, err
	}
	request, err := p.newAuthorizationHTTPRequest(ctx, params)
	if err != nil {
		return nil, err
	}
	if _, err := p.oauth.NewAuthorizeRequest(ctx, request); err != nil {
		return nil, err
	}
	handle, expiresAt, err := p.store.CreatePendingRequest(ctx, params, browserBinding)
	if err != nil {
		return nil, err
	}
	prepared := &PreparedAuthorization{
		RequestID: handle,
		ExpiresIn: int(time.Until(expiresAt).Seconds()),
	}
	if params.Get("prompt") == "none" {
		prepared.UnauthenticatedRedirectTo = p.authorizationErrorRedirect(params, "login_required")
	}
	return prepared, nil
}

func (p *Provider) Authorize(
	ctx context.Context,
	requestID string,
	browserBinding string,
	userID int64,
	authenticatedAt time.Time,
) (_ *AuthorizationResult, err error) {
	if !p.Enabled() {
		return nil, ErrProviderDisabled
	}
	if !opaqueHandlePattern.MatchString(requestID) || !opaqueHandlePattern.MatchString(browserBinding) {
		return nil, ErrPendingRequestNotFound
	}
	txCtx, err := p.store.BeginTX(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = p.store.Rollback(txCtx)
		}
	}()

	params, requestedAt, err := p.store.LoadPendingRequestForUpdate(txCtx, requestID, browserBinding)
	if err != nil {
		return nil, err
	}
	request, err := p.newAuthorizationHTTPRequest(txCtx, params)
	if err != nil {
		return nil, err
	}
	authorizeRequest, err := p.oauth.NewAuthorizeRequest(txCtx, request)
	if err != nil {
		return nil, err
	}
	profile, err := p.store.GetActiveProfileByUserID(txCtx, userID)
	if err != nil {
		return nil, err
	}
	prompt := params.Get("prompt")
	if prompt == "login" && (authenticatedAt.IsZero() || authenticatedAt.Before(requestedAt)) {
		return &AuthorizationResult{Reauthenticate: true}, nil
	}
	if prompt == "none" && (authenticatedAt.IsZero() || authenticatedAt.After(requestedAt)) {
		if err := p.store.ConsumePendingRequest(txCtx, requestID, browserBinding); err != nil {
			return nil, err
		}
		if err := p.store.Commit(txCtx); err != nil {
			return nil, err
		}
		committed = true
		return &AuthorizationResult{
			RedirectTo: p.authorizationErrorRedirect(params, "login_required"),
		}, nil
	}
	for _, scope := range oidcScopes {
		authorizeRequest.GrantScope(scope)
	}
	if authenticatedAt.IsZero() {
		authenticatedAt = time.Now().UTC()
	}
	session := openid.NewDefaultSession()
	session.Subject = profile.Subject
	session.Username = profile.PreferredUsername
	claims := session.IDTokenClaims()
	claims.Subject = profile.Subject
	claims.RequestedAt = requestedAt
	claims.AuthTime = authenticatedAt.UTC().Truncate(time.Second)
	claims.Extra = profileClaims(profile)

	authorizeResponse, err := p.oauth.NewAuthorizeResponse(txCtx, authorizeRequest, session)
	if err != nil {
		return nil, err
	}
	if err := p.store.ConsumePendingRequest(txCtx, requestID, browserBinding); err != nil {
		return nil, err
	}
	if err := p.store.Commit(txCtx); err != nil {
		return nil, err
	}
	committed = true

	callback, err := url.Parse(authorizeRequest.GetRedirectURI().String())
	if err != nil {
		return nil, err
	}
	query := callback.Query()
	for key, values := range authorizeResponse.GetParameters() {
		for _, value := range values {
			query.Set(key, value)
		}
	}
	query.Set("scope", Scope)
	callback.RawQuery = query.Encode()
	return &AuthorizationResult{
		RedirectTo: callback.String(),
		ExpiresIn:  int(AuthorizationCodeTTL / time.Second),
	}, nil
}

func (p *Provider) HandleToken(ctx context.Context, writer http.ResponseWriter, request *http.Request) (*UserProfile, error) {
	if !p.Enabled() {
		return nil, ErrProviderDisabled
	}
	accessRequest, err := p.oauth.NewAccessRequest(ctx, request, openid.NewDefaultSession())
	if err != nil {
		p.oauth.WriteAccessError(ctx, writer, accessRequest, err)
		return nil, err
	}
	profile, err := p.store.GetActiveProfileBySubject(ctx, accessRequest.GetSession().GetSubject())
	if err != nil {
		protocolErr := fosite.ErrInvalidGrant.WithHint("The resource owner account is not active.")
		p.oauth.WriteAccessError(ctx, writer, accessRequest, protocolErr)
		return nil, protocolErr
	}
	accessResponse, err := p.oauth.NewAccessResponse(ctx, accessRequest)
	if err != nil {
		p.oauth.WriteAccessError(ctx, writer, accessRequest, err)
		return nil, err
	}
	accessResponse.SetTokenType("Bearer")
	accessResponse.SetExpiresIn(AccessTokenTTL)
	accessResponse.SetScopes(oidcScopes)
	p.oauth.WriteAccessResponse(ctx, writer, accessRequest, accessResponse)
	return profile, nil
}

func (p *Provider) UserInfo(ctx context.Context, accessToken string) (map[string]any, *UserProfile, error) {
	if !p.Enabled() {
		return nil, nil, ErrProviderDisabled
	}
	tokenUse, requester, err := p.oauth.IntrospectToken(
		ctx,
		accessToken,
		fosite.AccessToken,
		openid.NewDefaultSession(),
		"openid",
	)
	if err != nil || tokenUse != fosite.AccessToken || requester == nil {
		if err == nil {
			err = fosite.ErrInvalidTokenFormat
		}
		return nil, nil, err
	}
	if requester.GetClient().GetID() != p.config.ClientID || !requester.GetGrantedScopes().Has("openid") {
		return nil, nil, fosite.ErrScopeNotGranted
	}
	profile, err := p.store.GetActiveProfileBySubject(ctx, requester.GetSession().GetSubject())
	if err != nil {
		return nil, nil, fosite.ErrInvalidTokenFormat
	}
	return profileClaims(profile), profile, nil
}

func (p *Provider) validateAuthorizationParameters(params url.Values) error {
	if params == nil {
		return ErrInvalidRequest
	}
	for key, values := range params {
		if _, ok := allowedAuthorizationParameters[key]; !ok || len(values) != 1 {
			return ErrInvalidRequest
		}
		if strings.ContainsAny(values[0], "\r\n") {
			return ErrInvalidRequest
		}
	}
	if params.Get("response_type") != "code" ||
		params.Get("client_id") != p.config.ClientID ||
		params.Get("redirect_uri") != p.config.RedirectURI ||
		params.Get("code_challenge_method") != "S256" ||
		!pkceChallengePattern.MatchString(params.Get("code_challenge")) ||
		!validEntropyParameter(params.Get("state")) ||
		!validEntropyParameter(params.Get("nonce")) ||
		!equalOIDCScopes(params.Get("scope")) {
		return ErrInvalidRequest
	}
	if mode := params.Get("response_mode"); mode != "" && mode != "query" {
		return ErrInvalidRequest
	}
	if prompt := params.Get("prompt"); prompt != "" && prompt != "none" && prompt != "login" {
		return ErrInvalidRequest
	}
	return nil
}

func (p *Provider) authorizationErrorRedirect(params url.Values, errorCode string) string {
	callback, err := url.Parse(p.config.RedirectURI)
	if err != nil {
		return ""
	}
	query := callback.Query()
	query.Set("error", errorCode)
	query.Set("state", params.Get("state"))
	callback.RawQuery = query.Encode()
	return callback.String()
}

func (p *Provider) newAuthorizationHTTPRequest(ctx context.Context, params url.Values) (*http.Request, error) {
	endpoint := p.config.Issuer + "/oidc/authorize?" + params.Encode()
	return http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
}

func profileClaims(profile *UserProfile) map[string]any {
	return map[string]any{
		"sub":                profile.Subject,
		"email":              profile.Email,
		"email_verified":     profile.EmailVerified,
		"preferred_username": profile.PreferredUsername,
		"name":               profile.Name,
	}
}

func validEntropyParameter(value string) bool {
	return len(value) >= 16 && len(value) <= 1024 && !strings.ContainsAny(value, "\r\n")
}

func equalOIDCScopes(raw string) bool {
	got := strings.Fields(raw)
	if len(got) != len(oidcScopes) {
		return false
	}
	want := append([]string(nil), []string(oidcScopes)...)
	sort.Strings(got)
	sort.Strings(want)
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func ProtocolErrorCode(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrProviderDisabled):
		return "provider_disabled"
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidBrowserBinding), errors.Is(err, ErrPendingRequestNotFound):
		return "invalid_request"
	case errors.Is(err, ErrUserInactive):
		return "access_denied"
	}
	return fosite.ErrorToRFC6749Error(err).ErrorField
}
