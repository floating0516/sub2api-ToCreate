package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"
)

const channelMonitorV2CatalogTTL = 30 * time.Second

// ChannelMonitorV2CatalogSource loads the probe-model inventory used to seed
// Channel Status dimensions when operators leave platform model lists empty.
type ChannelMonitorV2CatalogSource interface {
	ChannelMonitorV2Catalog(ctx context.Context) (map[string][]string, error)
}

type channelMonitorV2ProbeCatalog struct {
	monitors ChannelMonitorRepository
}

// NewChannelMonitorV2ProbeCatalog uses enabled V1 channel monitors
// (primary_model + extra_models) as the only named models on Channel Status.
func NewChannelMonitorV2ProbeCatalog(monitors ChannelMonitorRepository) ChannelMonitorV2CatalogSource {
	return &channelMonitorV2ProbeCatalog{monitors: monitors}
}

func (l *channelMonitorV2ProbeCatalog) ChannelMonitorV2Catalog(ctx context.Context) (map[string][]string, error) {
	if l == nil || l.monitors == nil {
		return nil, nil
	}
	listed, err := l.monitors.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return BuildChannelMonitorV2ProbeCatalog(listed), nil
}

// BuildChannelMonitorV2ProbeCatalog is the pure inventory used by Channel Status.
func BuildChannelMonitorV2ProbeCatalog(monitors []*ChannelMonitor) map[string][]string {
	sets := map[string]map[string]struct{}{}
	add := func(platform, model string) {
		platform = strings.ToLower(strings.TrimSpace(platform))
		model = strings.TrimSpace(model)
		if platform == "" || model == "" {
			return
		}
		if _, wild := splitWildcardSuffix(model); wild {
			return
		}
		if sets[platform] == nil {
			sets[platform] = map[string]struct{}{}
		}
		sets[platform][model] = struct{}{}
	}
	for _, monitor := range monitors {
		if monitor == nil || !monitor.Enabled {
			continue
		}
		models := append([]string{monitor.PrimaryModel}, monitor.ExtraModels...)
		for _, model := range models {
			add(channelMonitorV2ProbePlatform(monitor.Provider, model), model)
		}
	}
	out := make(map[string][]string, len(sets))
	for platform, models := range sets {
		list := make([]string, 0, len(models))
		for model := range models {
			list = append(list, model)
		}
		sort.Strings(list)
		out[platform] = list
	}
	return out
}

func channelMonitorV2ProbePlatform(provider, model string) string {
	if platform, ok := DetectModelPlatform(model); ok {
		return platform
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case MonitorProviderAnthropic, MonitorProviderOpenAI, MonitorProviderGemini, MonitorProviderGrok:
		return provider
	default:
		return provider
	}
}

func (s *ChannelMonitorV2Service) attachRuntimeCatalog(ctx context.Context, cfg *ChannelMonitorV2Config) {
	if s == nil || cfg == nil {
		return
	}
	models, err := s.cachedCatalog(ctx)
	if err != nil {
		slog.Warn("channel_monitor_v2_catalog_failed", "error", err)
		return
	}
	if len(models) == 0 {
		return
	}
	cfg.RuntimeCatalog = models
}

func (s *ChannelMonitorV2Service) cachedCatalog(ctx context.Context) (map[string][]string, error) {
	if s == nil || s.catalog == nil {
		return nil, nil
	}
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	if s.catalogCache != nil && !s.catalogAt.IsZero() && now.Sub(s.catalogAt) < channelMonitorV2CatalogTTL {
		return s.catalogCache, nil
	}
	models, err := s.catalog.ChannelMonitorV2Catalog(ctx)
	if err != nil {
		return nil, err
	}
	s.catalogCache = models
	s.catalogAt = now
	return models, nil
}

// ChannelMonitorV2ModelAliases returns identity plus dated/thinking family names.
func ChannelMonitorV2ModelAliases(model string) []string {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	add(model)
	withoutThinking := strings.TrimSuffix(model, "-thinking")
	add(withoutThinking)
	add(stripChannelMonitorV2DateSuffix(model))
	add(stripChannelMonitorV2DateSuffix(withoutThinking))
	return out
}

func stripChannelMonitorV2DateSuffix(model string) string {
	if i := strings.LastIndex(model, "@"); i > 0 && channelMonitorV2IsDateSuffix(model[i+1:]) {
		return model[:i]
	}
	if i := strings.LastIndex(model, "-"); i > 0 && channelMonitorV2IsDateSuffix(model[i+1:]) {
		return model[:i]
	}
	return model
}

func channelMonitorV2IsDateSuffix(value string) bool {
	if len(value) != 8 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

// ChannelMonitorV2CanonicalModel maps a raw traffic model onto the longest
// listed catalog/allow-list name among its dated/thinking aliases.
func ChannelMonitorV2CanonicalModel(listed []string, model string) (string, bool) {
	aliases := ChannelMonitorV2ModelAliases(model)
	if len(aliases) == 0 || len(listed) == 0 {
		return "", false
	}
	aliasSet := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		aliasSet[strings.ToLower(alias)] = struct{}{}
	}
	best := ""
	for _, item := range listed {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := aliasSet[strings.ToLower(item)]; !ok {
			continue
		}
		if len(item) > len(best) {
			best = item
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}
