package service

import (
	"context"
	"database/sql"
	"math"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

const (
	subscriptionSettlementExpired = "expired"
	subscriptionSettlementRenewed = "renewed"
	subscriptionSettlementRevoked = "revoked"
)

type subscriptionSnapshotExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *SubscriptionService) snapshotSubscriptionTerm(ctx context.Context, sub *UserSubscription, reason string) error {
	if s == nil || s.entClient == nil || sub == nil || sub.Group == nil {
		return nil
	}

	capacity, ok := subscriptionTermQuotaCapacity(sub)
	if !ok {
		capacity = 0
	}
	usageBySubscription, err := s.subscriptionTermUsage(ctx, []UserSubscription{*sub})
	if err != nil {
		return err
	}
	used, unused, overage := calculateSubscriptionQuotaOutcome(
		capacity,
		math.Max(usageBySubscription[sub.ID], currentSubscriptionUsageFloor(sub)),
	)

	var executor subscriptionSnapshotExecutor = s.entClient
	if tx := dbent.TxFromContext(ctx); tx != nil {
		executor = tx.Client()
	}
	return insertSubscriptionTermSnapshot(ctx, executor, sub, reason, capacity, used, unused, overage)
}

func insertSubscriptionTermSnapshot(ctx context.Context, executor subscriptionSnapshotExecutor, sub *UserSubscription, reason string, capacity, used, unused, overage float64) error {
	if executor == nil || sub == nil {
		return nil
	}
	_, err := executor.ExecContext(ctx, `
		INSERT INTO subscription_term_snapshots (
			subscription_id, user_id, group_id, starts_at, expires_at,
			settled_at, settlement_reason, total_quota_usd, used_quota_usd, unused_quota_usd, overage_usd
		) VALUES ($1, $2, $3, $4, $5, NOW(), $6, $7, $8, $9, $10)
		ON CONFLICT (subscription_id, starts_at, expires_at) DO UPDATE SET
			settled_at = EXCLUDED.settled_at,
			settlement_reason = EXCLUDED.settlement_reason,
			total_quota_usd = EXCLUDED.total_quota_usd,
			used_quota_usd = EXCLUDED.used_quota_usd,
			unused_quota_usd = EXCLUDED.unused_quota_usd,
			overage_usd = EXCLUDED.overage_usd
	`, sub.ID, sub.UserID, sub.GroupID, sub.StartsAt, sub.ExpiresAt, reason, capacity, used, unused, overage)
	return err
}

func snapshotExpiredSubscriptionTerms(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT
			us.id, us.user_id, us.group_id, us.starts_at, us.expires_at,
			us.daily_usage_usd, us.weekly_usage_usd, us.monthly_usage_usd,
			g.daily_limit_usd, g.weekly_limit_usd, g.monthly_limit_usd,
			COALESCE(SUM(ul.actual_cost), 0)
		FROM user_subscriptions us
		JOIN groups g ON g.id = us.group_id AND g.deleted_at IS NULL
		LEFT JOIN usage_logs ul
			ON ul.subscription_id = us.id
			AND ul.created_at >= us.starts_at
			AND ul.created_at < us.expires_at
		WHERE us.status = 'active'
			AND us.expires_at <= NOW()
			AND us.deleted_at IS NULL
		GROUP BY us.id, g.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type expiredTerm struct {
		sub       UserSubscription
		daily     sql.NullFloat64
		weekly    sql.NullFloat64
		monthly   sql.NullFloat64
		loggedUse float64
	}
	terms := make([]expiredTerm, 0)
	for rows.Next() {
		var term expiredTerm
		if err := rows.Scan(
			&term.sub.ID, &term.sub.UserID, &term.sub.GroupID, &term.sub.StartsAt, &term.sub.ExpiresAt,
			&term.sub.DailyUsageUSD, &term.sub.WeeklyUsageUSD, &term.sub.MonthlyUsageUSD,
			&term.daily, &term.weekly, &term.monthly, &term.loggedUse,
		); err != nil {
			return err
		}
		term.sub.Group = &Group{ID: term.sub.GroupID}
		if term.daily.Valid {
			term.sub.Group.DailyLimitUSD = &term.daily.Float64
		}
		if term.weekly.Valid {
			term.sub.Group.WeeklyLimitUSD = &term.weekly.Float64
		}
		if term.monthly.Valid {
			term.sub.Group.MonthlyLimitUSD = &term.monthly.Float64
		}
		terms = append(terms, term)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range terms {
		term := &terms[i]
		capacity, _ := subscriptionTermQuotaCapacity(&term.sub)
		used, unused, overage := calculateSubscriptionQuotaOutcome(
			capacity,
			math.Max(term.loggedUse, currentSubscriptionUsageFloor(&term.sub)),
		)
		if err := insertSubscriptionTermSnapshot(ctx, db, &term.sub, subscriptionSettlementExpired, capacity, used, unused, overage); err != nil {
			return err
		}
	}
	return nil
}
