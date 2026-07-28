package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	InstallClientClaudeCode = "claude-code"
	InstallClientCodex      = "codex"
	InstallClientGeminiCLI  = "gemini-cli"

	installTokenTTL          = 20 * time.Minute
	installReceiptTTL        = 10 * time.Minute
	installCredentialTombTTL = 30 * time.Minute

	installTokenPrefix   = "tcinst_"
	installReceiptPrefix = "tcrcp_"

	installCredentialPending  = "pending"
	installCredentialRedeemed = "redeemed"
	installCredentialExpired  = "expired"
	installCredentialRevoked  = "revoked"

	installCredentialKindToken   = "token"
	installCredentialKindReceipt = "receipt"
)

var (
	ErrInstallClientInvalid = infraerrors.BadRequest(
		"client_invalid",
		"unsupported install client",
	)
	ErrInstallKeyUnavailable = infraerrors.Forbidden(
		"key_disabled",
		"the selected API key is unavailable",
	)
	ErrInstallClientMismatch = infraerrors.Forbidden(
		"client_mismatch",
		"the selected API key group is not compatible with this client",
	)
	ErrInstallNoCredit = infraerrors.New(
		http.StatusPaymentRequired,
		"no_credit",
		"add balance or activate a subscription before creating an install command",
	)
	ErrInstallTokenInvalid = infraerrors.BadRequest(
		"token_invalid",
		"invalid install token",
	)
	ErrInstallTokenNotFound = infraerrors.NotFound(
		"token_not_found",
		"install token not found",
	)
	ErrInstallTokenExpired = infraerrors.New(
		http.StatusGone,
		"token_expired",
		"install token expired; refresh the command in Quick Start",
	)
	ErrInstallTokenUsed = infraerrors.New(
		http.StatusGone,
		"token_used",
		"install token has already been used; refresh the command in Quick Start",
	)
	ErrInstallTokenRevoked = infraerrors.New(
		http.StatusGone,
		"token_revoked",
		"install token was revoked; refresh the command in Quick Start",
	)
	ErrInstallTokenRateLimited = infraerrors.TooManyRequests(
		"install_token_rate_limited",
		"too many install token requests; try again shortly",
	)
)

type installAPIKeyReader interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
	CheckAPIKeyQuotaAndExpiry(apiKey *APIKey) error
}

type installSubscriptionReader interface {
	GetActiveSubscription(ctx context.Context, userID, groupID int64) (*UserSubscription, error)
}

type installSettingsReader interface {
	GetPublicSettings(ctx context.Context) (*PublicSettings, error)
}

type InstallTokenService struct {
	redis         *redis.Client
	apiKeys       installAPIKeyReader
	subscriptions installSubscriptionReader
	settings      installSettingsReader
	cfg           *config.Config
	now           func() time.Time
}

type InstallTokenCommands struct {
	Unix    string `json:"unix"`
	Windows string `json:"windows"`
}

type InstallTokenKeySummary struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Prefix         string  `json:"prefix"`
	GroupID        int64   `json:"group_id"`
	GroupName      string  `json:"group_name"`
	Platform       string  `json:"platform"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type InstallTokenIssueResult struct {
	Token       string                 `json:"token"`
	Client      string                 `json:"client"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Commands    InstallTokenCommands   `json:"commands"`
	FallbackURL string                 `json:"fallback_url"`
	Key         InstallTokenKeySummary `json:"key"`
}

type InstallTokenPeekResult struct {
	Client       string                 `json:"client"`
	ExpiresAt    time.Time              `json:"expires_at"`
	ProviderName string                 `json:"provider_name"`
	Endpoint     string                 `json:"endpoint"`
	Key          InstallTokenKeySummary `json:"key"`
}

type InstallTokenRedeemResult struct {
	Client       string `json:"client"`
	App          string `json:"app"`
	ProviderName string `json:"provider_name"`
	Homepage     string `json:"homepage"`
	Endpoint     string `json:"endpoint"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model,omitempty"`
	UsageScript  string `json:"usage_script"`
	Deeplink     string `json:"deeplink"`
	ConfirmURL   string `json:"confirm_url,omitempty"`
	KeyName      string `json:"key_name"`
}

type installCredentialRecord struct {
	Kind      string
	Status    string
	UserID    int64
	KeyID     int64
	Client    string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
}

type installPublicContext struct {
	ProviderName string
	APIBaseURL   string
	Origin       string
}

type installImportConfig struct {
	App      string
	Endpoint string
	Model    string
}

var installTransitionScript = redis.NewScript(`
local status = redis.call("HGET", KEYS[1], "status")
if not status then
  return "missing"
end
if status ~= "pending" then
  return status
end
local expires_at = tonumber(redis.call("HGET", KEYS[1], "expires_at") or "0")
local now = tonumber(ARGV[1])
if expires_at <= now then
  redis.call("HSET", KEYS[1], "status", "expired")
  return "expired"
end
redis.call("HSET", KEYS[1], "status", ARGV[2])
if ARGV[2] == "redeemed" then
  redis.call("HSET", KEYS[1], "used_at", ARGV[1])
end
return "transitioned"
`)

var installRateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func NewInstallTokenService(
	redisClient *redis.Client,
	apiKeyService *APIKeyService,
	subscriptionService *SubscriptionService,
	settingService *SettingService,
	cfg *config.Config,
) *InstallTokenService {
	return newInstallTokenService(
		redisClient,
		apiKeyService,
		subscriptionService,
		settingService,
		cfg,
	)
}

func newInstallTokenService(
	redisClient *redis.Client,
	apiKeys installAPIKeyReader,
	subscriptions installSubscriptionReader,
	settings installSettingsReader,
	cfg *config.Config,
) *InstallTokenService {
	return &InstallTokenService{
		redis:         redisClient,
		apiKeys:       apiKeys,
		subscriptions: subscriptions,
		settings:      settings,
		cfg:           cfg,
		now:           time.Now,
	}
}

func (s *InstallTokenService) Issue(
	ctx context.Context,
	userID int64,
	client string,
	keyID int64,
	previousToken string,
	requestOrigin string,
) (*InstallTokenIssueResult, error) {
	client = normalizeInstallClient(client)
	if !isInstallClient(client) {
		return nil, ErrInstallClientInvalid
	}
	if userID <= 0 || keyID <= 0 {
		return nil, ErrInstallKeyUnavailable
	}

	if strings.TrimSpace(previousToken) != "" {
		if _, err := s.Revoke(ctx, userID, previousToken); err != nil {
			return nil, err
		}
	}

	key, err := s.validateBoundKey(ctx, userID, keyID, client)
	if err != nil {
		return nil, err
	}
	publicContext, err := s.loadPublicContext(ctx, requestOrigin)
	if err != nil {
		return nil, err
	}
	if _, err := resolveInstallImportConfig(client, key.Group.Platform, publicContext.APIBaseURL); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	rawToken, err := generateInstallCredential(installTokenPrefix)
	if err != nil {
		return nil, fmt.Errorf("generate install token: %w", err)
	}
	record := &installCredentialRecord{
		Kind:      installCredentialKindToken,
		Status:    installCredentialPending,
		UserID:    userID,
		KeyID:     keyID,
		Client:    client,
		CreatedAt: now,
		ExpiresAt: now.Add(installTokenTTL),
	}
	if err := s.saveRecord(ctx, rawToken, record); err != nil {
		return nil, err
	}

	return &InstallTokenIssueResult{
		Token:     rawToken,
		Client:    client,
		ExpiresAt: record.ExpiresAt,
		Commands:  buildInstallCommands(publicContext.Origin, rawToken),
		FallbackURL: buildInstallConfirmURL(
			publicContext.Origin,
			"token",
			rawToken,
		),
		Key: installKeySummary(key),
	}, nil
}

func (s *InstallTokenService) Peek(
	ctx context.Context,
	rawCredential string,
	clientIP string,
	requestOrigin string,
) (*InstallTokenPeekResult, error) {
	if err := s.checkAccessRate(ctx, "peek", rawCredential, clientIP, 20); err != nil {
		return nil, err
	}
	record, err := s.loadUsableRecord(
		ctx,
		rawCredential,
		installCredentialKindForRaw(rawCredential),
	)
	if err != nil {
		return nil, err
	}
	key, err := s.validateBoundKey(ctx, record.UserID, record.KeyID, record.Client)
	if err != nil {
		return nil, err
	}
	publicContext, err := s.loadPublicContext(ctx, requestOrigin)
	if err != nil {
		return nil, err
	}
	importConfig, err := resolveInstallImportConfig(record.Client, key.Group.Platform, publicContext.APIBaseURL)
	if err != nil {
		return nil, err
	}

	return &InstallTokenPeekResult{
		Client:       record.Client,
		ExpiresAt:    record.ExpiresAt,
		ProviderName: publicContext.ProviderName,
		Endpoint:     importConfig.Endpoint,
		Key:          installKeySummary(key),
	}, nil
}

func (s *InstallTokenService) Redeem(
	ctx context.Context,
	rawToken string,
	clientIP string,
	requestOrigin string,
) (*InstallTokenRedeemResult, error) {
	if err := s.checkAccessRate(ctx, "redeem", rawToken, clientIP, 10); err != nil {
		return nil, err
	}
	record, err := s.loadUsableRecord(ctx, rawToken, installCredentialKindToken)
	if err != nil {
		return nil, err
	}
	key, err := s.validateBoundKey(ctx, record.UserID, record.KeyID, record.Client)
	if err != nil {
		return nil, err
	}
	publicContext, err := s.loadPublicContext(ctx, requestOrigin)
	if err != nil {
		return nil, err
	}
	payload, err := buildInstallRedeemPayload(record.Client, key, publicContext)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	receipt, err := generateInstallCredential(installReceiptPrefix)
	if err != nil {
		return nil, fmt.Errorf("generate install confirmation receipt: %w", err)
	}
	receiptRecord := &installCredentialRecord{
		Kind:      installCredentialKindReceipt,
		Status:    installCredentialPending,
		UserID:    record.UserID,
		KeyID:     record.KeyID,
		Client:    record.Client,
		CreatedAt: now,
		ExpiresAt: now.Add(installReceiptTTL),
	}
	if err := s.saveRecord(ctx, receipt, receiptRecord); err != nil {
		return nil, err
	}

	if err := s.claimRecord(ctx, rawToken, installCredentialKindToken); err != nil {
		s.deleteRecord(ctx, receipt, installCredentialKindReceipt)
		return nil, err
	}
	payload.ConfirmURL = buildInstallConfirmURL(publicContext.Origin, "receipt", receipt)
	return payload, nil
}

func (s *InstallTokenService) Confirm(
	ctx context.Context,
	receipt string,
	clientIP string,
	requestOrigin string,
) (*InstallTokenRedeemResult, error) {
	if err := s.checkAccessRate(ctx, "confirm", receipt, clientIP, 10); err != nil {
		return nil, err
	}
	record, err := s.loadUsableRecord(ctx, receipt, installCredentialKindReceipt)
	if err != nil {
		return nil, err
	}
	key, err := s.validateBoundKey(ctx, record.UserID, record.KeyID, record.Client)
	if err != nil {
		return nil, err
	}
	publicContext, err := s.loadPublicContext(ctx, requestOrigin)
	if err != nil {
		return nil, err
	}
	payload, err := buildInstallRedeemPayload(record.Client, key, publicContext)
	if err != nil {
		return nil, err
	}
	if err := s.claimRecord(ctx, receipt, installCredentialKindReceipt); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *InstallTokenService) Revoke(
	ctx context.Context,
	userID int64,
	rawToken string,
) (string, error) {
	record, err := s.loadRecord(ctx, rawToken, installCredentialKindToken)
	if err != nil {
		return "", err
	}
	if record.UserID != userID {
		return "", ErrInstallTokenNotFound
	}
	switch record.Status {
	case installCredentialRevoked, installCredentialExpired, installCredentialRedeemed:
		return record.Status, nil
	case installCredentialPending:
	default:
		return "", ErrInstallTokenInvalid
	}

	key, err := installCredentialRedisKey(rawToken, installCredentialKindToken)
	if err != nil {
		return "", err
	}
	status, err := s.transitionRecord(ctx, key, installCredentialRevoked)
	if err != nil {
		return "", err
	}
	switch status {
	case "transitioned":
		return installCredentialRevoked, nil
	case "missing":
		return "", ErrInstallTokenNotFound
	case installCredentialExpired, installCredentialRevoked, installCredentialRedeemed:
		return status, nil
	default:
		return "", ErrInstallTokenInvalid
	}
}

func (s *InstallTokenService) validateBoundKey(
	ctx context.Context,
	userID int64,
	keyID int64,
	client string,
) (*APIKey, error) {
	if s.apiKeys == nil {
		return nil, fmt.Errorf("install token API key reader is not configured")
	}
	key, err := s.apiKeys.GetByID(ctx, keyID)
	if err != nil || key == nil || key.UserID != userID {
		return nil, ErrInstallKeyUnavailable
	}
	if !key.IsActive() || s.apiKeys.CheckAPIKeyQuotaAndExpiry(key) != nil {
		return nil, ErrInstallKeyUnavailable
	}
	if key.User == nil || !key.User.IsActive() || key.Group == nil || key.Group.Status != StatusActive {
		return nil, ErrInstallKeyUnavailable
	}
	if !isInstallClientPlatformCompatible(client, key.Group.Platform) {
		return nil, ErrInstallClientMismatch
	}

	if key.Group.IsSubscriptionType() {
		if s.subscriptions == nil {
			return nil, fmt.Errorf("install token subscription reader is not configured")
		}
		subscription, err := s.subscriptions.GetActiveSubscription(ctx, userID, key.Group.ID)
		if err != nil || subscription == nil || !subscription.IsActive() {
			return nil, ErrInstallNoCredit
		}
	} else if s.cfg == nil || s.cfg.RunMode != config.RunModeSimple {
		if key.User.Balance-key.User.FrozenBalance <= 0 {
			return nil, ErrInstallNoCredit
		}
	}

	return key, nil
}

func (s *InstallTokenService) loadPublicContext(
	ctx context.Context,
	requestOrigin string,
) (installPublicContext, error) {
	if s.settings == nil {
		return installPublicContext{}, fmt.Errorf("install token settings reader is not configured")
	}
	settings, err := s.settings.GetPublicSettings(ctx)
	if err != nil {
		return installPublicContext{}, fmt.Errorf("load install settings: %w", err)
	}

	requestOrigin = normalizeInstallOrigin(requestOrigin)
	apiBaseURL := normalizeInstallBaseURL(settings.APIBaseURL)
	if apiBaseURL == "" {
		apiBaseURL = requestOrigin
	}
	if apiBaseURL == "" {
		return installPublicContext{}, fmt.Errorf("public API base URL is not configured")
	}
	origin := installURLOrigin(apiBaseURL)
	if origin == "" {
		origin = requestOrigin
	}
	if origin == "" {
		return installPublicContext{}, fmt.Errorf("public install origin is not configured")
	}

	providerName := strings.TrimSpace(settings.SiteName)
	if providerName == "" {
		providerName = "ToCreate"
	}
	return installPublicContext{
		ProviderName: providerName,
		APIBaseURL:   strings.TrimRight(apiBaseURL, "/"),
		Origin:       strings.TrimRight(origin, "/"),
	}, nil
}

func (s *InstallTokenService) saveRecord(
	ctx context.Context,
	raw string,
	record *installCredentialRecord,
) error {
	if s.redis == nil {
		return fmt.Errorf("install token storage is not configured")
	}
	key, err := installCredentialRedisKey(raw, record.Kind)
	if err != nil {
		return err
	}
	ttl := time.Until(record.ExpiresAt) + installCredentialTombTTL
	if ttl < installCredentialTombTTL {
		ttl = installCredentialTombTTL
	}
	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"kind":       record.Kind,
		"status":     record.Status,
		"user_id":    record.UserID,
		"key_id":     record.KeyID,
		"client":     record.Client,
		"created_at": record.CreatedAt.Unix(),
		"expires_at": record.ExpiresAt.Unix(),
	})
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store install credential: %w", err)
	}
	return nil
}

func (s *InstallTokenService) loadRecord(
	ctx context.Context,
	raw string,
	kind string,
) (*installCredentialRecord, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("install token storage is not configured")
	}
	key, err := installCredentialRedisKey(raw, kind)
	if err != nil {
		return nil, err
	}
	values, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("load install credential: %w", err)
	}
	if len(values) == 0 {
		return nil, ErrInstallTokenNotFound
	}

	userID, err := strconv.ParseInt(values["user_id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid install credential user")
	}
	keyID, err := strconv.ParseInt(values["key_id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid install credential key")
	}
	createdUnix, err := strconv.ParseInt(values["created_at"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid install credential creation time")
	}
	expiresUnix, err := strconv.ParseInt(values["expires_at"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid install credential expiry")
	}
	record := &installCredentialRecord{
		Kind:      values["kind"],
		Status:    values["status"],
		UserID:    userID,
		KeyID:     keyID,
		Client:    values["client"],
		CreatedAt: time.Unix(createdUnix, 0).UTC(),
		ExpiresAt: time.Unix(expiresUnix, 0).UTC(),
	}
	if record.Kind != kind {
		return nil, ErrInstallTokenInvalid
	}
	if used := values["used_at"]; used != "" {
		usedUnix, parseErr := strconv.ParseInt(used, 10, 64)
		if parseErr == nil {
			usedAt := time.Unix(usedUnix, 0).UTC()
			record.UsedAt = &usedAt
		}
	}
	if record.Status == installCredentialPending && !record.ExpiresAt.After(s.now()) {
		record.Status = installCredentialExpired
		_ = s.redis.HSet(ctx, key, "status", installCredentialExpired).Err()
	}
	return record, nil
}

func (s *InstallTokenService) loadUsableRecord(
	ctx context.Context,
	raw string,
	kind string,
) (*installCredentialRecord, error) {
	record, err := s.loadRecord(ctx, raw, kind)
	if err != nil {
		return nil, err
	}
	switch record.Status {
	case installCredentialPending:
		return record, nil
	case installCredentialExpired:
		return nil, ErrInstallTokenExpired
	case installCredentialRevoked:
		return nil, ErrInstallTokenRevoked
	case installCredentialRedeemed:
		return nil, ErrInstallTokenUsed
	default:
		return nil, ErrInstallTokenInvalid
	}
}

func (s *InstallTokenService) claimRecord(
	ctx context.Context,
	raw string,
	kind string,
) error {
	key, err := installCredentialRedisKey(raw, kind)
	if err != nil {
		return err
	}
	status, err := s.transitionRecord(ctx, key, installCredentialRedeemed)
	if err != nil {
		return err
	}
	switch status {
	case "transitioned":
		return nil
	case "missing":
		return ErrInstallTokenNotFound
	case installCredentialExpired:
		return ErrInstallTokenExpired
	case installCredentialRevoked:
		return ErrInstallTokenRevoked
	default:
		return ErrInstallTokenUsed
	}
}

func (s *InstallTokenService) transitionRecord(
	ctx context.Context,
	key string,
	nextStatus string,
) (string, error) {
	if s.redis == nil {
		return "", fmt.Errorf("install token storage is not configured")
	}
	result, err := installTransitionScript.Run(
		ctx,
		s.redis,
		[]string{key},
		s.now().UTC().Unix(),
		nextStatus,
	).Result()
	if err != nil {
		return "", fmt.Errorf("update install credential: %w", err)
	}
	return fmt.Sprint(result), nil
}

func (s *InstallTokenService) deleteRecord(
	ctx context.Context,
	raw string,
	kind string,
) {
	if s.redis == nil {
		return
	}
	key, err := installCredentialRedisKey(raw, kind)
	if err == nil {
		_ = s.redis.Del(ctx, key).Err()
	}
}

func (s *InstallTokenService) checkAccessRate(
	ctx context.Context,
	scope string,
	raw string,
	clientIP string,
	tokenLimit int64,
) error {
	redisKey, err := installCredentialRedisKey(raw, installCredentialKindForRaw(raw))
	if err != nil {
		return err
	}
	if s.redis == nil {
		return fmt.Errorf("install token storage is not configured")
	}

	tokenDigest := redisKey[strings.LastIndex(redisKey, ":")+1:]
	if err := s.enforceRateLimit(
		ctx,
		"install:rate:"+scope+":token:"+tokenDigest,
		tokenLimit,
		time.Minute,
	); err != nil {
		return err
	}
	if normalizedIP := strings.TrimSpace(clientIP); normalizedIP != "" {
		ipDigest := sha256.Sum256([]byte(normalizedIP))
		if err := s.enforceRateLimit(
			ctx,
			"install:rate:"+scope+":ip:"+hex.EncodeToString(ipDigest[:]),
			60,
			time.Minute,
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *InstallTokenService) enforceRateLimit(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) error {
	result, err := installRateLimitScript.Run(
		ctx,
		s.redis,
		[]string{key},
		window.Milliseconds(),
	).Int64()
	if err != nil {
		return fmt.Errorf("check install token rate limit: %w", err)
	}
	if result > limit {
		return ErrInstallTokenRateLimited
	}
	return nil
}

func generateInstallCredential(prefix string) (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(random), nil
}

func installCredentialRedisKey(raw string, kind string) (string, error) {
	raw = strings.TrimSpace(raw)
	expectedPrefix := installTokenPrefix
	redisPrefix := "install:token:v1:"
	if kind == installCredentialKindReceipt {
		expectedPrefix = installReceiptPrefix
		redisPrefix = "install:receipt:v1:"
	}
	if !strings.HasPrefix(raw, expectedPrefix) || len(raw) < len(expectedPrefix)+32 || len(raw) > 160 {
		return "", ErrInstallTokenInvalid
	}
	digest := sha256.Sum256([]byte(raw))
	return redisPrefix + hex.EncodeToString(digest[:]), nil
}

func installCredentialKindForRaw(raw string) string {
	if strings.HasPrefix(strings.TrimSpace(raw), installReceiptPrefix) {
		return installCredentialKindReceipt
	}
	return installCredentialKindToken
}

func normalizeInstallClient(client string) string {
	return strings.ToLower(strings.TrimSpace(client))
}

func isInstallClient(client string) bool {
	switch client {
	case InstallClientClaudeCode, InstallClientCodex, InstallClientGeminiCLI:
		return true
	default:
		return false
	}
}

func isInstallClientPlatformCompatible(client string, platform string) bool {
	switch normalizeInstallClient(client) {
	case InstallClientClaudeCode:
		return platform == PlatformAnthropic ||
			platform == PlatformAntigravity ||
			platform == PlatformComposite
	case InstallClientCodex:
		return platform == PlatformOpenAI
	case InstallClientGeminiCLI:
		return platform == PlatformGemini || platform == PlatformAntigravity
	default:
		return false
	}
}

func resolveInstallImportConfig(
	client string,
	platform string,
	baseURL string,
) (installImportConfig, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if !isInstallClientPlatformCompatible(client, platform) {
		return installImportConfig{}, ErrInstallClientMismatch
	}
	switch client {
	case InstallClientClaudeCode:
		endpoint := baseURL
		if platform == PlatformAntigravity {
			endpoint += "/antigravity"
		}
		return installImportConfig{App: "claude", Endpoint: endpoint}, nil
	case InstallClientCodex:
		return installImportConfig{
			App:      "codex",
			Endpoint: baseURL,
			Model:    "gpt-5.5",
		}, nil
	case InstallClientGeminiCLI:
		endpoint := baseURL
		if platform == PlatformAntigravity {
			endpoint += "/antigravity"
		}
		return installImportConfig{App: "gemini", Endpoint: endpoint}, nil
	default:
		return installImportConfig{}, ErrInstallClientInvalid
	}
}

func buildInstallRedeemPayload(
	client string,
	key *APIKey,
	publicContext installPublicContext,
) (*InstallTokenRedeemResult, error) {
	config, err := resolveInstallImportConfig(client, key.Group.Platform, publicContext.APIBaseURL)
	if err != nil {
		return nil, err
	}
	usageScript := `({
  request: {
    url: "{{baseUrl}}/v1/usage",
    method: "GET",
    headers: { "Authorization": "Bearer {{apiKey}}" }
  },
  extractor: function(response) {
    const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
    const unit = response?.unit ?? response?.quota?.unit ?? "USD";
    return {
      isValid: response?.is_active ?? response?.isValid ?? true,
      remaining,
      unit
    };
  }
})`
	values := url.Values{}
	values.Set("resource", "provider")
	values.Set("app", config.App)
	values.Set("name", publicContext.ProviderName)
	values.Set("homepage", publicContext.APIBaseURL)
	values.Set("endpoint", config.Endpoint)
	values.Set("apiKey", key.Key)
	values.Set("configFormat", "json")
	values.Set("usageEnabled", "true")
	values.Set("usageScript", base64.StdEncoding.EncodeToString([]byte(usageScript)))
	values.Set("usageAutoInterval", "30")
	if config.Model != "" {
		values.Set("model", config.Model)
	}

	return &InstallTokenRedeemResult{
		Client:       client,
		App:          config.App,
		ProviderName: publicContext.ProviderName,
		Homepage:     publicContext.APIBaseURL,
		Endpoint:     config.Endpoint,
		APIKey:       key.Key,
		Model:        config.Model,
		UsageScript:  usageScript,
		Deeplink:     "ccswitch://v1/import?" + values.Encode(),
		KeyName:      key.Name,
	}, nil
}

func installKeySummary(key *APIKey) InstallTokenKeySummary {
	groupID := int64(0)
	groupName := ""
	platform := ""
	rate := float64(0)
	if key.Group != nil {
		groupID = key.Group.ID
		groupName = key.Group.Name
		platform = key.Group.Platform
		rate = key.Group.RateMultiplier
	}
	return InstallTokenKeySummary{
		ID:             key.ID,
		Name:           key.Name,
		Prefix:         maskInstallAPIKey(key.Key),
		GroupID:        groupID,
		GroupName:      groupName,
		Platform:       platform,
		RateMultiplier: rate,
	}
}

func maskInstallAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 10 {
		return "******"
	}
	return key[:8] + "****"
}

func buildInstallCommands(origin string, token string) InstallTokenCommands {
	origin = strings.TrimRight(origin, "/")
	return InstallTokenCommands{
		Unix: fmt.Sprintf(
			"curl -fsSL %s/install.sh | bash -s -- --token %s",
			shellQuote(origin),
			shellQuote(token),
		),
		Windows: fmt.Sprintf(
			"& ([scriptblock]::Create((irm %s))) -Token %s",
			powerShellQuote(origin+"/install.ps1"),
			powerShellQuote(token),
		),
	}
}

func buildInstallConfirmURL(origin string, parameter string, credential string) string {
	values := url.Values{}
	values.Set(parameter, credential)
	return strings.TrimRight(origin, "/") + "/custom/install-confirm?" + values.Encode()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func normalizeInstallBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func normalizeInstallOrigin(raw string) string {
	base := normalizeInstallBaseURL(raw)
	return installURLOrigin(base)
}

func installURLOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
