package repository

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExtractOpenAIQuotaSnapshot(t *testing.T) {
	observedAt := time.Date(2026, 7, 18, 11, 43, 25, 0, time.FixedZone("CST", 8*60*60))
	resetAt := observedAt.Add(7 * 24 * time.Hour)

	snapshot, ok := extractOpenAIQuotaSnapshot(map[string]any{
		"codex_7d_used_percent":  "12.5",
		"codex_7d_reset_at":      resetAt.Format(time.RFC3339),
		"codex_usage_updated_at": observedAt.Format(time.RFC3339),
	})

	require.True(t, ok)
	require.Equal(t, 12.5, snapshot.Used)
	require.Equal(t, observedAt.UTC(), snapshot.ObservedAt)
	require.NotNil(t, snapshot.ResetAt)
	require.Equal(t, resetAt.UTC(), *snapshot.ResetAt)
}

func TestExtractOpenAIQuotaSnapshotRejectsInvalidPercent(t *testing.T) {
	for _, value := range []any{nil, "bad", -1, 101, math.NaN()} {
		_, ok := extractOpenAIQuotaSnapshot(map[string]any{"codex_7d_used_percent": value})
		require.False(t, ok, "value=%v", value)
	}
}

func TestOpenAIQuotaSampleBucketStart(t *testing.T) {
	observedAt := time.Date(2026, 7, 18, 11, 43, 25, 0, time.UTC)
	require.Equal(t, time.Date(2026, 7, 18, 11, 40, 0, 0, time.UTC), observedAt.Truncate(openAIQuotaSampleBucket))
}

func TestDetectOpenAIQuotaReset(t *testing.T) {
	base := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	previousReset := base.Add(2 * time.Hour)
	weekLater := previousReset.Add(7 * 24 * time.Hour)

	tests := []struct {
		name         string
		previousUsed float64
		nextUsed     float64
		observedAt   time.Time
		previousAt   *time.Time
		nextAt       *time.Time
		want         string
	}{
		{
			name:         "normal increase",
			previousUsed: 20,
			nextUsed:     28,
			observedAt:   base.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       &weekLater,
		},
		{
			name:         "small reset forecast drift",
			previousUsed: 20,
			nextUsed:     20,
			observedAt:   base.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       timePointer(previousReset.Add(30 * time.Second)),
		},
		{
			name:         "small usage jitter",
			previousUsed: 20,
			nextUsed:     19.7,
			observedAt:   base.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       &previousReset,
		},
		{
			name:         "confirmed small drop",
			previousUsed: 1,
			nextUsed:     0,
			observedAt:   base.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       &weekLater,
			want:         "usage_drop",
		},
		{
			name:         "uncorroborated strong early drop",
			previousUsed: 36,
			nextUsed:     9,
			observedAt:   base.Add(time.Minute),
		},
		{
			name:         "strong drop with unchanged forecast",
			previousUsed: 36,
			nextUsed:     0,
			observedAt:   base.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       &previousReset,
		},
		{
			name:         "strong drop with advanced forecast",
			previousUsed: 36,
			nextUsed:     9,
			observedAt:   base.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       &weekLater,
			want:         "usage_drop",
		},
		{
			name:         "zero usage scheduled boundary",
			previousUsed: 0,
			nextUsed:     0,
			observedAt:   previousReset.Add(time.Minute),
			previousAt:   &previousReset,
			nextAt:       &weekLater,
			want:         "window_elapsed",
		},
		{
			name:         "forecast moves before boundary",
			previousUsed: 10,
			nextUsed:     10,
			observedAt:   base,
			previousAt:   &previousReset,
			nextAt:       &weekLater,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous := &openAIQuotaActiveCycle{
				LastUsedPercent: tt.previousUsed,
				ProviderResetAt: tt.previousAt,
			}
			next := &openAIQuotaSnapshot{
				ObservedAt: tt.observedAt,
				Used:       tt.nextUsed,
				ResetAt:    tt.nextAt,
			}
			require.Equal(t, tt.want, detectOpenAIQuotaReset(previous, next))
		})
	}
}

func TestIsUncorroboratedOpenAIQuotaLow(t *testing.T) {
	base := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)
	resetAt := base.Add(7 * 24 * time.Hour)
	previous := &openAIQuotaActiveCycle{
		LastUsedPercent: 25,
		ProviderResetAt: &resetAt,
	}

	require.True(t, isUncorroboratedOpenAIQuotaLow(previous, &openAIQuotaSnapshot{
		ObservedAt: base.Add(time.Minute),
		Used:       0,
		ResetAt:    &resetAt,
	}))
	require.False(t, isUncorroboratedOpenAIQuotaLow(previous, &openAIQuotaSnapshot{
		ObservedAt: base.Add(time.Minute),
		Used:       0,
		ResetAt:    timePointer(resetAt.Add(7 * 24 * time.Hour)),
	}))
	require.False(t, isUncorroboratedOpenAIQuotaLow(previous, &openAIQuotaSnapshot{
		ObservedAt: base.Add(time.Minute),
		Used:       24.7,
		ResetAt:    &resetAt,
	}))
}

func timePointer(value time.Time) *time.Time {
	return &value
}
