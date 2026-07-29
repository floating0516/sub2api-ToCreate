package service

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
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

type fakeInstallCredentialStore struct {
	mu      sync.Mutex
	records map[string]*InstallCredentialRecord
	rates   map[string]int64
}

func newFakeInstallCredentialStore() *fakeInstallCredentialStore {
	return &fakeInstallCredentialStore{
		records: make(map[string]*InstallCredentialRecord),
		rates:   make(map[string]int64),
	}
}

func (s *fakeInstallCredentialStore) Save(
	_ context.Context,
	storageKey string,
	record *InstallCredentialRecord,
	_ time.Duration,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[storageKey] = cloneInstallCredentialRecord(record)
	return nil
}

func (s *fakeInstallCredentialStore) Load(
	_ context.Context,
	storageKey string,
) (*InstallCredentialRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneInstallCredentialRecord(s.records[storageKey]), nil
}

func (s *fakeInstallCredentialStore) Transition(
	_ context.Context,
	storageKey string,
	now time.Time,
	nextStatus string,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.records[storageKey]
	if record == nil {
		return "missing", nil
	}
	if record.Status != installCredentialPending {
		return record.Status, nil
	}
	if !record.ExpiresAt.After(now) {
		record.Status = installCredentialExpired
		return installCredentialExpired, nil
	}
	record.Status = nextStatus
	if nextStatus == installCredentialRedeemed {
		usedAt := now.UTC()
		record.UsedAt = &usedAt
	}
	return "transitioned", nil
}

func (s *fakeInstallCredentialStore) Delete(_ context.Context, storageKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, storageKey)
	return nil
}

func (s *fakeInstallCredentialStore) Increment(
	_ context.Context,
	storageKey string,
	_ time.Duration,
) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rates[storageKey]++
	return s.rates[storageKey], nil
}

func (s *fakeInstallCredentialStore) setExpiresAt(storageKey string, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record := s.records[storageKey]; record != nil {
		record.ExpiresAt = expiresAt
	}
}

func cloneInstallCredentialRecord(record *InstallCredentialRecord) *InstallCredentialRecord {
	if record == nil {
		return nil
	}
	cloned := *record
	if record.UsedAt != nil {
		usedAt := *record.UsedAt
		cloned.UsedAt = &usedAt
	}
	return &cloned
}

func TestInstallTokenIssueRedeemAndConfirmAreOneTime(t *testing.T) {
	service, key, store := newInstallTokenTestService(t, PlatformOpenAI)
	ctx := context.Background()

	issued, err := service.Issue(ctx, key.UserID, InstallClientCodex, key.ID, "", "https://fallback.example")
	require.NoError(t, err)
	require.NotEmpty(t, issued.Token)
	require.Contains(t, issued.Commands.Unix, issued.Token)
	require.Contains(t, issued.Commands.Windows, issued.Token)
	require.NotContains(t, issued.Commands.Unix, key.Key)
	require.NotContains(t, issued.Commands.Windows, key.Key)
	require.Contains(t, issued.Commands.Unix, "https://api.example.com/install.sh")
	require.Contains(t, issued.Commands.Unix, `. "${CODEX_HOME:-$HOME/.codex}/tocreate.env"`)

	store.mu.Lock()
	for storageKey, record := range store.records {
		require.NotContains(t, storageKey, issued.Token)
		require.NotContains(t, record.Kind, issued.Token)
		require.NotContains(t, record.Client, issued.Token)
		require.NotContains(t, record.Kind, key.Key)
		require.NotContains(t, record.Client, key.Key)
	}
	store.mu.Unlock()

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

func TestBuildInstallCommandsLoadsOnlyCodexEnvironmentInCurrentUnixShell(t *testing.T) {
	const (
		origin = "https://api.example.com/"
		token  = "tcinst_test-token"
	)

	for _, test := range []struct {
		client     string
		loadsCodex bool
	}{
		{client: InstallClientClaudeCode, loadsCodex: false},
		{client: InstallClientCodex, loadsCodex: true},
		{client: InstallClientGeminiCLI, loadsCodex: false},
	} {
		t.Run(test.client, func(t *testing.T) {
			commands := buildInstallCommands(origin, token, test.client)

			require.Contains(t, commands.Unix, "https://api.example.com/install.sh")
			require.Contains(t, commands.Unix, token)
			require.Equal(
				t,
				test.loadsCodex,
				strings.Contains(commands.Unix, `. "${CODEX_HOME:-$HOME/.codex}/tocreate.env"`),
			)
			require.NotContains(t, commands.Windows, "tocreate.env")
		})
	}
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
	service, key, store := newInstallTokenTestService(t, PlatformGemini)
	ctx := context.Background()

	issued, err := service.Issue(ctx, key.UserID, InstallClientGeminiCLI, key.ID, "", "https://api.example.com")
	require.NoError(t, err)
	redisKey, err := installCredentialRedisKey(issued.Token, installCredentialKindToken)
	require.NoError(t, err)
	store.setExpiresAt(redisKey, time.Now().Add(-time.Minute))

	_, err = service.Redeem(ctx, issued.Token, "203.0.113.9", "https://api.example.com")
	require.Equal(t, "token_expired", infraerrors.Reason(err))

	issued, err = service.Issue(ctx, key.UserID, InstallClientGeminiCLI, key.ID, "", "https://api.example.com")
	require.NoError(t, err)
	keyReader, ok := service.apiKeys.(*fakeInstallAPIKeyReader)
	require.True(t, ok)
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
) (*InstallTokenService, *APIKey, *fakeInstallCredentialStore) {
	t.Helper()
	store := newFakeInstallCredentialStore()

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
		store,
		&fakeInstallAPIKeyReader{key: key},
		&fakeInstallSubscriptionReader{},
		&fakeInstallSettingsReader{settings: &PublicSettings{
			SiteName:   "ToCreate",
			APIBaseURL: "https://api.example.com",
		}},
		&config.Config{RunMode: config.RunModeStandard},
	)
	return service, key, store
}
