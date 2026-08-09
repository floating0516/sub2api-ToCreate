package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestGetStatsSummaryWithFiltersRunsOnlySummaryQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)

	mock.ExpectQuery("SELECT[[:space:]]+COUNT\\(\\*\\) as total_requests").
		WithArgs(int64(42), start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_tokens",
			"total_cache_creation_tokens",
			"total_cache_read_tokens",
			"total_cost",
			"total_actual_cost",
			"total_account_cost",
			"avg_duration_ms",
		}).AddRow(12, 100, 50, 25, 10, 15, 1.5, 1.2, 0.8, 240.0))

	stats, err := repo.GetStatsSummaryWithFilters(context.Background(), usagestats.UsageLogFilters{
		UserID:    42,
		StartTime: &start,
		EndTime:   &end,
	})
	require.NoError(t, err)
	require.Equal(t, int64(175), stats.TotalTokens)
	require.Nil(t, stats.Endpoints)
	require.Nil(t, stats.UpstreamEndpoints)
	require.Nil(t, stats.EndpointPaths)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserDashboardSummaryStatsReturnsLifetimeOverview(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	todayStart := time.Date(2026, 8, 9, 0, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))

	mock.ExpectQuery("(?s)SELECT[[:space:]]+COUNT\\(\\*\\) AS total_requests.*FILTER \\(WHERE created_at >= \\$3\\).*AS today_actual_cost").
		WithArgs(int64(42), sqlmock.AnyArg(), todayStart).
		WillReturnRows(sqlmock.NewRows([]string{
			"total_requests",
			"total_input_tokens",
			"total_output_tokens",
			"total_cache_creation_tokens",
			"total_cache_read_tokens",
			"total_cost",
			"total_actual_cost",
			"today_actual_cost",
			"average_duration_ms",
			"recent_requests",
			"recent_tokens",
		}).AddRow(240, 600000, 300000, 50000, 37654, 15.5, 12.25, 1.75, 820.5, 25, 5000))

	stats, err := repo.GetUserDashboardSummaryStats(context.Background(), 42, todayStart)
	require.NoError(t, err)
	require.Equal(t, int64(987654), stats.TotalTokens)
	require.Equal(t, int64(240), stats.TotalRequests)
	require.Equal(t, int64(600000), stats.TotalInputTokens)
	require.Equal(t, int64(300000), stats.TotalOutputTokens)
	require.Equal(t, int64(87654), stats.TotalCacheCreationTokens+stats.TotalCacheReadTokens)
	require.InDelta(t, 12.25, stats.TotalActualCost, 0.000001)
	require.InDelta(t, 1.75, stats.TodayActualCost, 0.000001)
	require.InDelta(t, 820.5, stats.AverageDurationMs, 0.000001)
	require.Equal(t, int64(5), stats.Rpm)
	require.Equal(t, int64(1000), stats.Tpm)
	require.Empty(t, stats.ByPlatform)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserModelUsageTrendReturnsRankedSeriesFromOneQuery(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(7 * 24 * time.Hour)

	mock.ExpectQuery("WITH filtered AS").
		WithArgs(int64(42), start, end, 8).
		WillReturnRows(sqlmock.NewRows([]string{
			"date",
			"model",
			"rank",
			"requests",
			"total_tokens",
		}).
			AddRow("2026-08-01", "gpt-5.6-sol", 1, 10, 1000).
			AddRow("2026-08-01", "claude-opus-4-8", 2, 4, 300))

	trend, err := repo.GetUserModelUsageTrend(context.Background(), 42, start, end, "day", 8)
	require.NoError(t, err)
	require.Len(t, trend, 2)
	require.Equal(t, "gpt-5.6-sol", trend[0].Model)
	require.Equal(t, 1, trend[0].Rank)
	require.Equal(t, int64(300), trend[1].TotalTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}
