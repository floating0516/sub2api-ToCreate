package service

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type fakeInstallAPIKeyReader struct {
	key *APIKey
	err error
}

func (f *fakeInstallAPIKeyReader) GetByID(context.Context, int64) (*APIKey, error) {
	return f.key, f.err
}

func (f *fakeInstallAPIKeyReader) CheckAPIKeyQuotaAndExpiry(key *APIKey) error {
	if key.IsExpired() {
		return ErrAPIKeyExpired
	}
	if key.IsQuotaExhausted() {
		return ErrAPIKeyQuotaExhausted
	}
	return nil
}

type fakeInstallSubscriptionReader struct {
	sub *UserSubscription
	err error
}

func (f *fakeInstallSubscriptionReader) GetActiveSubscription(context.Context, int64, int64) (*UserSubscription, error) {
	return f.sub, f.err
}

type fakeInstallSettingsReader struct {
	settings *PublicSettings
	err      error
}

func (f *fakeInstallSettingsReader) GetPublicSettings(context.Context) (*PublicSettings, error) {
	return f.settings, f.err
}

func TestInstallTokenIssueRedeemAndConfirmAreOneTime(t *testing.T) {
	service, key, redisServer := newInstallTokenTestService(t, PlatformOpenAI)
	ctx := context.Background()

	issued, err := service.Issue(ctx, key.UserID, InstallClientCodex, key.ID, "", "https://fallback.example")
	require.NoError(t, err)
	require.NotEmpty(t, issued.Token)
	require.Contains(t, issued.Commands.Unix, issued.Token)
	require.Contains(t, issued.Commands.Windows, issued.Token)
	require.NotContains(t, issued.Commands.Unix, key.Key)
	require.NotContains(t, issued.Commands.Windows, key.Key)
	require.Contains(t, issued.Commands.Unix, "https://api.example.com/install.sh")

	for _, redisKey := range redisServer.Keys() {
		require.NotContains(t, redisKey, issued.Token)
		values, redisErr := service.redis.HGetAll(ctx, redisKey).Result()
		require.NoError(t, redisErr)
		for _, value := range values {
			require.NotContains(t, value, issued.Token)
			require.NotContains(t, value, key.Key)
		}
	}

	peeked, err := service.Peek(ctx, issued.Token, "203.0.113.5", "https://fallback.example")
	require.NoError(t, err)
	require.Equal(t, InstallClientCodex, peeked.Client)
	require.Equal(t, "https://api.example.com", peeked.Endpoint)
	require.NotContains(t, peeked.Key.Prefix, key.Key)

	redeemed, err := service.Redeem(ctx, issued.Token, "203.0.113.5", "https://fallback.example")
	require.NoError(t, err)
	require.Equal(t, key.Key, redeemed.APIKey)
	require.Equal(t, "codex", redeemed.App)
	require.Contains(t, redeemed.Deeplink, url.QueryEscape(key.Key))
	require.NotContains(t, redeemed.ConfirmURL, key.Key)

	_, err = service.Redeem(ctx, issued.Token, "203.0.113.5", "https://fallback.example")
	require.Equal(t, "token_used", infraerrors.Reason(err))

	confirmURL, err := url.Parse(redeemed.ConfirmURL)
	require.NoError(t, err)
	receipt := confirmURL.Query().Get("receipt")
	require.NotEmpty(t, receipt)

	receiptMetadata, err := service.Peek(ctx, receipt, "203.0.113.5", "https://fallback.example")
	require.NoError(t, err)
	require.Equal(t, InstallClientCodex, receiptMetadata.Client)
	require.Equal(t, key.ID, receiptMetadata.Key.ID)

	confirmed, err := service.Confirm(ctx, receipt, "203.0.113.5", "https://fallback.example")
	require.NoError(t, err)
	require.Equal(t, key.Key, confirmed.APIKey)

	_, err = service.Confirm(ctx, receipt, "203.0.113.5", "https://fallback.example")
	require.Equal(t, "token_used", infraerrors.Reason(err))
}

func TestInstallTokenConcurrentRedeemOnlyReturnsSecretOnce(t *testing.T) {
	service, key, _ := newInstallTokenTestService(t, PlatformOpenAI)
	ctx := context.Background()

	issued, err := service.Issue(ctx, key.UserID, InstallClientCodex, key.ID, "", "https://api.example.com")
	require.NoError(t, err)

	type result struct {
		payload *InstallTokenRedeemResult
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			payload, redeemErr := service.Redeem(
				ctx,
				issued.Token,
				"203.0.113.6",
				"https://api.example.com",
			)
			results <- result{payload: payload, err: redeemErr}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	usedErrors := 0
	for redeemResult := range results {
		if redeemResult.err == nil {
			successes++
			require.NotNil(t, redeemResult.payload)
			require.Equal(t, key.Key, redeemResult.payload.APIKey)
			continue
		}
		if infraerrors.Reason(redeemResult.err) == "token_used" {
			usedErrors++
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, usedErrors)
}

func TestInstallTokenRefreshRevokesPreviousCommand(t *testing.T) {
	service, key, _ := newInstallTokenTestService(t, PlatformAnthropic)
	ctx := context.Background()

	first, err := service.Issue(ctx, key.UserID, InstallClientClaudeCode, key.ID, "", "https://api.example.com")
	require.NoError(t, err)
	second, err := service.Issue(ctx, key.UserID, InstallClientClaudeCode, key.ID, first.Token, "https://api.example.com")
	require.NoError(t, err)
	require.NotEqual(t, first.Token, second.Token)

	_, err = service.Redeem(ctx, first.Token, "203.0.113.8", "https://api.example.com")
	require.Equal(t, "token_revoked", infraerrors.Reason(err))
	_, err = service.Peek(ctx, second.Token, "203.0.113.8", "https://api.example.com")
	require.NoError(t, err)
}

func TestInstallTokenRejectsExpiredAndDeletedKeys(t *testing.T) {
	service, key, _ := newInstallTokenTestService(t, PlatformGemini)
	ctx := context.Background()

	issued, err := service.Issue(ctx, key.UserID, InstallClientGeminiCLI, key.ID, "", "https://api.example.com")
	require.NoError(t, err)
	redisKey, err := installCredentialRedisKey(issued.Token, installCredentialKindToken)
	require.NoError(t, err)
	require.NoError(t, service.redis.HSet(ctx, redisKey, "expires_at", time.Now().Add(-time.Minute).Unix()).Err())

	_, err = service.Redeem(ctx, issued.Token, "203.0.113.9", "https://api.example.com")
	require.Equal(t, "token_expired", infraerrors.Reason(err))

	issued, err = service.Issue(ctx, key.UserID, InstallClientGeminiCLI, key.ID, "", "https://api.example.com")
	require.NoError(t, err)
	keyReader := service.apiKeys.(*fakeInstallAPIKeyReader)
	keyReader.key = nil
	keyReader.err = ErrAPIKeyNotFound

	_, err = service.Redeem(ctx, issued.Token, "203.0.113.9", "https://api.example.com")
	require.Equal(t, "key_disabled", infraerrors.Reason(err))
}

func TestInstallTokenRequiresCreditAndCompatibleGroup(t *testing.T) {
	service, key, _ := newInstallTokenTestService(t, PlatformOpenAI)
	ctx := context.Background()

	key.User.Balance = 0
	_, err := service.Issue(ctx, key.UserID, InstallClientCodex, key.ID, "", "https://api.example.com")
	require.Equal(t, "no_credit", infraerrors.Reason(err))

	key.User.Balance = 10
	_, err = service.Issue(ctx, key.UserID, InstallClientClaudeCode, key.ID, "", "https://api.example.com")
	require.Equal(t, "client_mismatch", infraerrors.Reason(err))
}

func TestInstallClientPlatformCompatibility(t *testing.T) {
	tests := []struct {
		client   string
		platform string
		want     bool
	}{
		{InstallClientClaudeCode, PlatformAnthropic, true},
		{InstallClientClaudeCode, PlatformAntigravity, true},
		{InstallClientClaudeCode, PlatformComposite, true},
		{InstallClientClaudeCode, PlatformOpenAI, false},
		{InstallClientCodex, PlatformOpenAI, true},
		{InstallClientCodex, PlatformGrok, false},
		{InstallClientGeminiCLI, PlatformGemini, true},
		{InstallClientGeminiCLI, PlatformAntigravity, true},
	}

	for _, test := range tests {
		require.Equal(t, test.want, isInstallClientPlatformCompatible(test.client, test.platform))
	}
}

func newInstallTokenTestService(
	t *testing.T,
	platform string,
) (*InstallTokenService, *APIKey, *miniredis.Miniredis) {
	t.Helper()
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})

	group := &Group{
		ID:               22,
		Name:             "Primary",
		Platform:         platform,
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeStandard,
		RateMultiplier:   0.3,
	}
	user := &User{
		ID:      11,
		Status:  StatusActive,
		Balance: 10,
	}
	key := &APIKey{
		ID:      33,
		UserID:  user.ID,
		Key:     "sk-test-install-secret",
		Name:    "Installer key",
		GroupID: &group.ID,
		Status:  StatusActive,
		User:    user,
		Group:   group,
	}
	service := newInstallTokenService(
		redisClient,
		&fakeInstallAPIKeyReader{key: key},
		&fakeInstallSubscriptionReader{},
		&fakeInstallSettingsReader{settings: &PublicSettings{
			SiteName:   "ToCreate",
			APIBaseURL: "https://api.example.com",
		}},
		&config.Config{RunMode: config.RunModeStandard},
	)
	return service, key, redisServer
}
