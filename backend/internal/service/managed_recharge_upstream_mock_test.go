package service

import (
	"context"
	"testing"
	"time"
)

func TestManagedRechargeMockProviderRequiresStagingEnvironment(t *testing.T) {
	t.Setenv(managedRechargeProviderModeEnv, "mock")
	t.Setenv(managedRechargeEnvironmentEnv, "production")

	upstream, mockMode, _, err := newManagedRechargeUpstreamFromEnvironment()
	if err == nil {
		t.Fatal("mock provider was enabled outside staging")
	}
	if upstream != nil || mockMode {
		t.Fatalf("invalid mock configuration returned upstream=%T mockMode=%t", upstream, mockMode)
	}
}

func TestManagedRechargeMockProviderUsesConfiguredStep(t *testing.T) {
	t.Setenv(managedRechargeProviderModeEnv, "mock")
	t.Setenv(managedRechargeEnvironmentEnv, managedRechargeMockEnvironment)
	t.Setenv(managedRechargeMockStepSecondsEnv, "7")

	upstream, mockMode, stepSeconds, err := newManagedRechargeUpstreamFromEnvironment()
	if err != nil {
		t.Fatalf("create mock provider: %v", err)
	}
	if _, ok := upstream.(*managedRechargeMockUpstream); !ok || !mockMode || stepSeconds != 7 {
		t.Fatalf("unexpected mock provider selection: upstream=%T mockMode=%t step=%d", upstream, mockMode, stepSeconds)
	}
}

func TestManagedRechargeFulfillmentConfigDefaultsToProxy(t *testing.T) {
	t.Setenv(managedRechargeFulfillmentModeEnv, "")
	t.Setenv(managedRechargeExternalRedeemURLEnv, "")

	mode, redeemURL, err := managedRechargeFulfillmentConfigFromEnvironment()
	if err != nil {
		t.Fatalf("read default fulfillment config: %v", err)
	}
	if mode != ManagedRechargeFulfillmentProxy || redeemURL != managedRechargeDefaultRedeemURL {
		t.Fatalf("default fulfillment config = %q %q", mode, redeemURL)
	}
}

func TestManagedRechargeFulfillmentConfigAcceptsExternalHTTPSURL(t *testing.T) {
	t.Setenv(managedRechargeFulfillmentModeEnv, "external")
	t.Setenv(managedRechargeExternalRedeemURLEnv, "https://redeem.example.test/recharge")

	mode, redeemURL, err := managedRechargeFulfillmentConfigFromEnvironment()
	if err != nil {
		t.Fatalf("read external fulfillment config: %v", err)
	}
	if mode != ManagedRechargeFulfillmentExternal || redeemURL != "https://redeem.example.test/recharge" {
		t.Fatalf("external fulfillment config = %q %q", mode, redeemURL)
	}
}

func TestManagedRechargeFulfillmentConfigRejectsUnsafeValues(t *testing.T) {
	for _, test := range []struct {
		name string
		mode string
		url  string
	}{
		{name: "mode", mode: "iframe", url: managedRechargeDefaultRedeemURL},
		{name: "http", mode: "external", url: "http://redeem.example.test/recharge"},
		{name: "credentials", mode: "external", url: "https://user:pass@redeem.example.test/recharge"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(managedRechargeFulfillmentModeEnv, test.mode)
			t.Setenv(managedRechargeExternalRedeemURLEnv, test.url)
			if _, _, err := managedRechargeFulfillmentConfigFromEnvironment(); err == nil {
				t.Fatal("unsafe fulfillment configuration was accepted")
			}
		})
	}
}

func TestManagedRechargeRealFulfillmentDefaultsDisabled(t *testing.T) {
	t.Setenv(managedRechargeProviderModeEnv, "real")
	t.Setenv(managedRechargeRealFulfillmentEnv, "")

	enabled, err := managedRechargeFulfillmentEnabledFromEnvironment(false)
	if err != nil {
		t.Fatalf("read real fulfillment default: %v", err)
	}
	if enabled {
		t.Fatal("real fulfillment was enabled without explicit opt-in")
	}
}

func TestManagedRechargeRealFulfillmentRequiresExplicitTrue(t *testing.T) {
	t.Setenv(managedRechargeProviderModeEnv, "real")
	t.Setenv(managedRechargeRealFulfillmentEnv, "true")

	enabled, err := managedRechargeFulfillmentEnabledFromEnvironment(false)
	if err != nil {
		t.Fatalf("read real fulfillment opt-in: %v", err)
	}
	if !enabled {
		t.Fatal("explicit real fulfillment opt-in was ignored")
	}
}

func TestManagedRechargeMockSuccessProgression(t *testing.T) {
	upstream, advance := newManagedRechargeMockUpstreamForTest(t)
	ctx := context.Background()
	code := "MOCK-PLUS-SUCCESS-001"

	verified, err := upstream.verifyCDK(ctx, code)
	if err != nil || !verified.Valid || verified.PlanType != "plus" {
		t.Fatalf("verify result=%+v err=%v", verified, err)
	}
	created, err := upstream.createTask(ctx, code, managedRechargeMockSessionForTest())
	if err != nil || created.TaskID == "" {
		t.Fatalf("create result=%+v err=%v", created, err)
	}
	confirmed, err := upstream.confirmTask(ctx, created.TaskID)
	if err != nil || confirmed.Status != "queued" {
		t.Fatalf("confirm result=%+v err=%v", confirmed, err)
	}

	assertManagedRechargeMockLookup(t, upstream, code, "queued", "")
	advance(10 * time.Second)
	assertManagedRechargeMockLookup(t, upstream, code, "processing", "")
	advance(10 * time.Second)
	assertManagedRechargeMockLookup(t, upstream, code, "completed", "processing")
	advance(10 * time.Second)
	assertManagedRechargeMockLookup(t, upstream, code, "completed", "completed")
}

func TestManagedRechargeMockSessionValidation(t *testing.T) {
	upstream := newManagedRechargeMockUpstream(10 * time.Second)
	result, err := upstream.validateSession(context.Background(), managedRechargeMockSessionForTest())
	if err != nil || !result.Valid || result.Email != managedRechargeMockSessionEmail {
		t.Fatalf("validate mock Session result=%+v err=%v", result, err)
	}

	result, err = upstream.validateSession(context.Background(), `{"user":{"email":"real@example.com"},"accessToken":"real-token"}`)
	if err != nil || result.Valid {
		t.Fatalf("mock provider accepted real Session result=%+v err=%v", result, err)
	}
}

func TestManagedRechargeMockFailureTriggersDefinitiveFailure(t *testing.T) {
	upstream, advance := newManagedRechargeMockUpstreamForTest(t)
	ctx := context.Background()
	code := "MOCK-FAIL-REFUND"
	if _, err := upstream.createTask(ctx, code, managedRechargeMockSessionForTest()); err != nil {
		t.Fatalf("create mock failure task: %v", err)
	}
	advance(20 * time.Second)
	result, err := upstream.lookupTask(ctx, code)
	if err != nil {
		t.Fatalf("lookup mock failure: %v", err)
	}
	if result.TaskStatus != "failed" || result.FailureReason != "session_invalid" {
		t.Fatalf("unexpected failure result: %+v", result)
	}
}

func TestManagedRechargeMockReplacementSessionCompletes(t *testing.T) {
	upstream, advance := newManagedRechargeMockUpstreamForTest(t)
	ctx := context.Background()
	code := "MOCK-SESSION-REQUIRED"
	if _, err := upstream.createTask(ctx, code, managedRechargeMockSessionForTest()); err != nil {
		t.Fatalf("create replacement task: %v", err)
	}
	advance(20 * time.Second)
	assertManagedRechargeMockLookup(t, upstream, code, "completed", "action_required")

	replaced, err := upstream.submitReplacementSession(ctx, code, managedRechargeMockSessionForTest())
	if err != nil || replaced.PostProcessStatus != "processing" {
		t.Fatalf("replacement result=%+v err=%v", replaced, err)
	}
	assertManagedRechargeMockLookup(t, upstream, code, "completed", "processing")
	advance(10 * time.Second)
	assertManagedRechargeMockLookup(t, upstream, code, "completed", "completed")
}

func TestManagedRechargeMockRejectsRealSessionShape(t *testing.T) {
	upstream := newManagedRechargeMockUpstream(10 * time.Second)
	result, err := upstream.createTask(context.Background(), "MOCK-PLUS-SUCCESS", `{"user":{"email":"real@example.com"},"accessToken":"real-token"}`)
	if err != nil {
		t.Fatalf("mock rejection returned transport error: %v", err)
	}
	if result.TaskID != "" || result.Error != "mock_session_required" {
		t.Fatalf("mock accepted a non-test Session: %+v", result)
	}
}

func newManagedRechargeMockUpstreamForTest(t *testing.T) (*managedRechargeMockUpstream, func(time.Duration)) {
	t.Helper()
	current := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	upstream := newManagedRechargeMockUpstream(10 * time.Second)
	upstream.now = func() time.Time { return current }
	return upstream, func(duration time.Duration) { current = current.Add(duration) }
}

func assertManagedRechargeMockLookup(t *testing.T, upstream *managedRechargeMockUpstream, code, taskStatus, postProcessStatus string) {
	t.Helper()
	result, err := upstream.lookupTask(context.Background(), code)
	if err != nil {
		t.Fatalf("lookup %s: %v", code, err)
	}
	if result.TaskStatus != taskStatus || result.PostProcessStatus != postProcessStatus {
		t.Fatalf("lookup %s returned task=%q post_process=%q, want task=%q post_process=%q", code,
			result.TaskStatus, result.PostProcessStatus, taskStatus, postProcessStatus)
	}
}

func managedRechargeMockSessionForTest() string {
	return `{"user":{"email":"mock@example.com"},"accessToken":"mock-access-token"}`
}
