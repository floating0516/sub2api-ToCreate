package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildChannelMonitorV2ProbeCatalogUsesEnabledProbeModels(t *testing.T) {
	got := BuildChannelMonitorV2ProbeCatalog([]*ChannelMonitor{
		{Enabled: true, Provider: MonitorProviderAnthropic, PrimaryModel: "claude-sonnet-5"},
		{Enabled: true, Provider: MonitorProviderOpenAI, PrimaryModel: "gpt-5.5"},
		{Enabled: true, Provider: MonitorProviderOpenAI, APIMode: MonitorAPIModeResponses, PrimaryModel: "grok-4.5"},
		{Enabled: true, Provider: MonitorProviderGemini, PrimaryModel: "gemini-3.1-flash-lite", ExtraModels: []string{"gemini-3.6-flash-low"}},
		{Enabled: false, Provider: MonitorProviderAnthropic, PrimaryModel: "claude-fable-5"},
		{Enabled: true, Provider: MonitorProviderAnthropic, PrimaryModel: "claude-sonnet-5"},
	})
	require.Equal(t, []string{"claude-sonnet-5"}, got["anthropic"])
	require.Equal(t, []string{"gpt-5.5"}, got["openai"])
	require.Equal(t, []string{"grok-4.5"}, got["grok"])
	require.Equal(t, []string{"gemini-3.1-flash-lite", "gemini-3.6-flash-low"}, got["gemini"])
	require.NotContains(t, got["anthropic"], "claude-fable-5")
}

func TestChannelMonitorV2CanonicalModelFoldsDatedThinkingWithoutVersionCollision(t *testing.T) {
	listed := []string{"claude-haiku-4-5", "claude-opus-4", "claude-opus-4-8", "claude-sonnet-5"}
	canon, ok := ChannelMonitorV2CanonicalModel(listed, "claude-haiku-4-5-20251001-thinking")
	require.True(t, ok)
	require.Equal(t, "claude-haiku-4-5", canon)

	canon, ok = ChannelMonitorV2CanonicalModel(listed, "claude-opus-4-8-thinking")
	require.True(t, ok)
	require.Equal(t, "claude-opus-4-8", canon)

	canon, ok = ChannelMonitorV2CanonicalModel(listed, "claude-sonnet-5@20260701")
	require.True(t, ok)
	require.Equal(t, "claude-sonnet-5", canon)

	_, ok = ChannelMonitorV2CanonicalModel(listed, "claude-fable-5-1")
	require.False(t, ok)
}

type stubChannelMonitorV2Catalog map[string][]string

func (s stubChannelMonitorV2Catalog) ChannelMonitorV2Catalog(context.Context) (map[string][]string, error) {
	return s, nil
}

func TestAttachRuntimeCatalogUsesProbeInventory(t *testing.T) {
	svc := NewChannelMonitorV2Service(nil)
	svc.SetCatalogSource(stubChannelMonitorV2Catalog{"anthropic": []string{"claude-sonnet-5"}})
	cfg := &ChannelMonitorV2Config{Enabled: true}
	svc.attachRuntimeCatalog(context.Background(), cfg)
	require.Equal(t, []string{"claude-sonnet-5"}, cfg.RuntimeCatalog["anthropic"])
}
