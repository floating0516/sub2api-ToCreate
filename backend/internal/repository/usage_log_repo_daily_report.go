package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

const dailyReportAccountCostExpr = "COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)"

func (r *usageLogRepository) GetDailyReport(ctx context.Context, startTime, endTime, previousStart, trendStart time.Time, timezoneName string) (*usagestats.DailyReport, error) {
	report := &usagestats.DailyReport{
		Date: startTime.Format("2006-01-02"), Timezone: timezoneName, StartTime: startTime, EndTime: endTime,
		Trend: []usagestats.DailyReportTrendPoint{}, Multipliers: []usagestats.DailyReportMultiplierStat{},
		Groups: []usagestats.DailyReportGroupStat{}, Users: []usagestats.DailyReportUserStat{},
	}
	if err := r.scanDailyReportSummary(ctx, previousStart, startTime, &report.PreviousSummary); err != nil {
		return nil, err
	}
	if err := r.scanDailyReportSummary(ctx, startTime, endTime, &report.Summary); err != nil {
		return nil, err
	}
	if err := r.scanDailyReportTrend(ctx, trendStart, endTime, timezoneName, &report.Trend); err != nil {
		return nil, err
	}
	if err := r.scanDailyReportMultipliers(ctx, startTime, endTime, &report.Multipliers); err != nil {
		return nil, err
	}
	if err := r.scanDailyReportGroups(ctx, startTime, endTime, &report.Groups); err != nil {
		return nil, err
	}
	if err := r.scanDailyReportUsers(ctx, startTime, endTime, &report.Users); err != nil {
		return nil, err
	}
	return report, nil
}

func (r *usageLogRepository) scanDailyReportSummary(ctx context.Context, startTime, endTime time.Time, out *usagestats.DailyReportSummary) error {
	query := fmt.Sprintf(`SELECT COUNT(DISTINCT user_id), COUNT(*), COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0), COALESCE(SUM(total_cost), 0), COALESCE(SUM(actual_cost), 0), COALESCE(SUM(%s), 0) FROM usage_logs WHERE created_at >= $1 AND created_at < $2`, dailyReportAccountCostExpr)
	return scanSingleRow(ctx, r.sql, query, []any{startTime, endTime}, &out.ActiveUsers, &out.TotalRequests, &out.TotalTokens, &out.TotalCost, &out.TotalActualCost, &out.TotalAccountCost)
}

func (r *usageLogRepository) scanDailyReportTrend(ctx context.Context, startTime, endTime time.Time, timezoneName string, out *[]usagestats.DailyReportTrendPoint) error {
	rows, err := r.sql.QueryContext(ctx, `SELECT TO_CHAR(created_at AT TIME ZONE $3, 'YYYY-MM-DD'), COUNT(DISTINCT user_id), COUNT(*), COALESCE(SUM(actual_cost), 0) FROM usage_logs WHERE created_at >= $1 AND created_at < $2 GROUP BY 1 ORDER BY 1`, startTime, endTime, timezoneName)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var item usagestats.DailyReportTrendPoint
		if err := rows.Scan(&item.Date, &item.ActiveUsers, &item.Requests, &item.ActualCost); err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return rows.Err()
}

func (r *usageLogRepository) scanDailyReportMultipliers(ctx context.Context, startTime, endTime time.Time, out *[]usagestats.DailyReportMultiplierStat) error {
	query := fmt.Sprintf(`SELECT COALESCE(rate_multiplier, 1), COUNT(*), COUNT(DISTINCT user_id), COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0), COALESCE(SUM(total_cost), 0), COALESCE(SUM(actual_cost), 0), COALESCE(SUM(%s), 0) FROM usage_logs WHERE created_at >= $1 AND created_at < $2 GROUP BY 1 ORDER BY 6 DESC`, dailyReportAccountCostExpr)
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var item usagestats.DailyReportMultiplierStat
		if err := rows.Scan(&item.RateMultiplier, &item.Requests, &item.ActiveUsers, &item.TotalTokens, &item.Cost, &item.ActualCost, &item.AccountCost); err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return rows.Err()
}

func (r *usageLogRepository) scanDailyReportGroups(ctx context.Context, startTime, endTime time.Time, out *[]usagestats.DailyReportGroupStat) error {
	query := `WITH base AS (
		SELECT ul.*, COALESCE(ul.group_id, 0) AS report_group_id
		FROM usage_logs ul
		WHERE ul.created_at >= $1 AND ul.created_at < $2
	), group_users AS (
		SELECT report_group_id, COUNT(DISTINCT user_id) AS active_users
		FROM base
		GROUP BY report_group_id
	)
	SELECT b.report_group_id,
		COALESCE(g.name, CASE WHEN b.report_group_id = 0 THEN 'Ungrouped' ELSE 'Deleted group #' || b.report_group_id::text END),
		COALESCE(b.rate_multiplier, 1), gu.active_users, COUNT(*), COUNT(DISTINCT b.user_id),
		COALESCE(SUM(b.input_tokens + b.output_tokens + b.cache_creation_tokens + b.cache_read_tokens), 0),
		COALESCE(SUM(b.total_cost), 0), COALESCE(SUM(b.actual_cost), 0),
		COALESCE(SUM(COALESCE(b.account_stats_cost, b.total_cost) * COALESCE(b.account_rate_multiplier, 1)), 0)
	FROM base b
	LEFT JOIN groups g ON g.id = NULLIF(b.report_group_id, 0)
	JOIN group_users gu ON gu.report_group_id = b.report_group_id
	GROUP BY b.report_group_id, g.name, b.rate_multiplier, gu.active_users
	ORDER BY 9 DESC`
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	groupIndex := map[int64]int{}
	for rows.Next() {
		var groupID int64
		var groupName string
		var groupActiveUsers int64
		var multiplier usagestats.DailyReportMultiplierStat
		if err := rows.Scan(&groupID, &groupName, &multiplier.RateMultiplier, &groupActiveUsers, &multiplier.Requests, &multiplier.ActiveUsers, &multiplier.TotalTokens, &multiplier.Cost, &multiplier.ActualCost, &multiplier.AccountCost); err != nil {
			return err
		}
		idx, ok := groupIndex[groupID]
		if !ok {
			idx = len(*out)
			groupIndex[groupID] = idx
			*out = append(*out, usagestats.DailyReportGroupStat{GroupID: groupID, GroupName: groupName, ActiveUsers: groupActiveUsers, Multipliers: []usagestats.DailyReportMultiplierStat{}})
		}
		group := &(*out)[idx]
		group.Requests += multiplier.Requests
		group.TotalTokens += multiplier.TotalTokens
		group.Cost += multiplier.Cost
		group.ActualCost += multiplier.ActualCost
		group.AccountCost += multiplier.AccountCost
		group.Multipliers = append(group.Multipliers, multiplier)
	}
	return rows.Err()
}

func (r *usageLogRepository) scanDailyReportUsers(ctx context.Context, startTime, endTime time.Time, out *[]usagestats.DailyReportUserStat) error {
	rows, err := r.sql.QueryContext(ctx, `SELECT ul.user_id, COALESCE(u.email, 'Deleted user #' || ul.user_id::text), COALESCE(u.username, ''), COUNT(*), COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0), COALESCE(SUM(ul.total_cost), 0), COALESCE(SUM(ul.actual_cost), 0) FROM usage_logs ul LEFT JOIN users u ON u.id = ul.user_id WHERE ul.created_at >= $1 AND ul.created_at < $2 GROUP BY ul.user_id, u.email, u.username ORDER BY 7 DESC LIMIT 20`, startTime, endTime)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var item usagestats.DailyReportUserStat
		if err := rows.Scan(&item.UserID, &item.Email, &item.Username, &item.Requests, &item.TotalTokens, &item.Cost, &item.ActualCost); err != nil {
			return err
		}
		*out = append(*out, item)
	}
	return rows.Err()
}
