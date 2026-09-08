package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCollapseChannelMonitorV2DisplayGroupsMergesProductRows(t *testing.T) {
	id15, id18, id2, id37 := int64(15), int64(18), int64(2), int64(37)
	start := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	matrix := &ChannelMonitorV2Matrix{
		GroupBy: ChannelMonitorV2GroupByPlatformGroup,
		Items: []ChannelMonitorV2MatrixRow{
			{
				Platform:  "openai",
				GroupID:   &id15,
				GroupName: "GPT_0.14",
				Metrics:   ChannelMonitorV2Metric{SuccessRequests: 90, ErrorRequests: 10, RequestCount: 100, SuccessRate: 0.9, ErrorRate: 0},
				Health:    ChannelMonitorV2HealthFor(ChannelMonitorV2Metric{RequestCount: 100, ErrorRate: 0}),
				Buckets: []ChannelMonitorV2TrendPoint{{
					BucketStart: start,
					Metrics:     ChannelMonitorV2Metric{SuccessRequests: 90, ErrorRequests: 10, RequestCount: 100, SuccessRate: 0.9, ErrorRate: 0},
				}},
			},
			{
				Platform:  "openai",
				GroupID:   &id18,
				GroupName: "pro拼车",
				Metrics:   ChannelMonitorV2Metric{SuccessRequests: 10, ErrorRequests: 10, RequestCount: 20, SuccessRate: 0.5, ErrorRate: 0.1},
				Health:    ChannelMonitorV2HealthFor(ChannelMonitorV2Metric{RequestCount: 20, ErrorRate: 0.1}),
				Buckets: []ChannelMonitorV2TrendPoint{{
					BucketStart: start,
					Metrics:     ChannelMonitorV2Metric{SuccessRequests: 10, ErrorRequests: 10, RequestCount: 20, SuccessRate: 0.5, ErrorRate: 0.1},
				}},
			},
			{
				Platform:  "anthropic",
				GroupID:   &id2,
				GroupName: "claude_0.6",
				Metrics:   ChannelMonitorV2Metric{SuccessRequests: 50, RequestCount: 50, SuccessRate: 1, ErrorRate: 0},
				Health:    ChannelMonitorV2HealthFor(ChannelMonitorV2Metric{RequestCount: 50, ErrorRate: 0}),
			},
			{
				Platform:  "anthropic",
				GroupID:   &id37,
				GroupName: "claude-cursor逆向",
				Metrics:   ChannelMonitorV2Metric{SuccessRequests: 0, ErrorRequests: 5, RequestCount: 5, SuccessRate: 0, ErrorRate: 1},
				Health:    ChannelMonitorV2HealthFor(ChannelMonitorV2Metric{RequestCount: 5, ErrorRate: 1}),
			},
		},
	}

	CollapseChannelMonitorV2DisplayGroups(matrix, ChannelMonitorV2GroupByPlatformGroup)

	require.Len(t, matrix.Items, 3)
	require.Equal(t, "Pro 渠道", matrix.Items[0].GroupName)
	require.Equal(t, "Claude Opus", matrix.Items[1].GroupName)
	require.Equal(t, "Claude 0.6", matrix.Items[2].GroupName)
	require.InDelta(t, 100.0/120.0, matrix.Items[0].Metrics.SuccessRate, 1e-9)
	require.Equal(t, int64(120), matrix.Items[0].Metrics.RequestCount)
	require.Equal(t, int64(0), matrix.Items[1].Metrics.SuccessRequests)
	require.Equal(t, int64(50), matrix.Items[2].Metrics.RequestCount)
	require.Len(t, matrix.Items[0].Buckets, 1)
	require.Equal(t, int64(120), matrix.Items[0].Buckets[0].Metrics.RequestCount)
}

func TestCollapseChannelMonitorV2DisplayGroupsSkipsPlatformView(t *testing.T) {
	matrix := &ChannelMonitorV2Matrix{
		Items: []ChannelMonitorV2MatrixRow{{Platform: "openai", Metrics: ChannelMonitorV2Metric{RequestCount: 3}}},
	}
	CollapseChannelMonitorV2DisplayGroups(matrix, ChannelMonitorV2GroupByPlatform)
	require.Len(t, matrix.Items, 1)
	require.Equal(t, "openai", matrix.Items[0].Platform)
}
