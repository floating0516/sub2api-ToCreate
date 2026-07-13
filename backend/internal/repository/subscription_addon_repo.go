package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type subscriptionAddonRepository struct {
	db *sql.DB
}

func NewSubscriptionAddonRepository(db *sql.DB) service.SubscriptionAddonRepository {
	return &subscriptionAddonRepository{db: db}
}

func (r *subscriptionAddonRepository) Create(ctx context.Context, pack *service.SubscriptionAddonPack) error {
	if r == nil || r.db == nil || pack == nil {
		return service.ErrSubscriptionAddonInvalid
	}
	return r.db.QueryRowContext(ctx, `
		INSERT INTO subscription_addon_packs (
			subscription_id, user_id, group_id, quota_usd, used_usd,
			starts_at, expires_at, status, assigned_by, notes
		) VALUES ($1, $2, $3, $4, 0, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, pack.SubscriptionID, pack.UserID, pack.GroupID, pack.QuotaUSD,
		pack.StartsAt, pack.ExpiresAt, pack.Status, pack.AssignedBy, pack.Notes,
	).Scan(&pack.ID, &pack.CreatedAt, &pack.UpdatedAt)
}

func (r *subscriptionAddonRepository) GetByID(ctx context.Context, id int64) (*service.SubscriptionAddonPack, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrSubscriptionAddonInvalid
	}
	pack, err := scanSubscriptionAddon(r.db.QueryRowContext(ctx, `
		SELECT id, subscription_id, user_id, group_id, quota_usd, used_usd,
			starts_at, expires_at, status, assigned_by, notes, revoked_at, created_at, updated_at
		FROM subscription_addon_packs
		WHERE id = $1
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSubscriptionAddonNotFound
	}
	return pack, err
}

func (r *subscriptionAddonRepository) GetUsableForSubscription(ctx context.Context, subscriptionID int64, now time.Time) (*service.SubscriptionAddonPack, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrSubscriptionAddonInvalid
	}
	pack, err := scanSubscriptionAddon(r.db.QueryRowContext(ctx, `
		SELECT id, subscription_id, user_id, group_id, quota_usd, used_usd,
			starts_at, expires_at, status, assigned_by, notes, revoked_at, created_at, updated_at
		FROM subscription_addon_packs
		WHERE subscription_id = $1
			AND status = 'active'
			AND starts_at <= $2
			AND expires_at > $2
			AND used_usd < quota_usd
		ORDER BY expires_at ASC, id ASC
		LIMIT 1
	`, subscriptionID, now))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSubscriptionAddonNotFound
	}
	return pack, err
}

func (r *subscriptionAddonRepository) ListBySubscriptionID(ctx context.Context, subscriptionID int64) ([]service.SubscriptionAddonPack, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrSubscriptionAddonInvalid
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, subscription_id, user_id, group_id, quota_usd, used_usd,
			starts_at, expires_at, status, assigned_by, notes, revoked_at, created_at, updated_at
		FROM subscription_addon_packs
		WHERE subscription_id = $1
		ORDER BY created_at DESC, id DESC
	`, subscriptionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make([]service.SubscriptionAddonPack, 0)
	for rows.Next() {
		pack, scanErr := scanSubscriptionAddon(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *pack)
	}
	return result, rows.Err()
}

func (r *subscriptionAddonRepository) GetActiveSummaries(ctx context.Context, subscriptionIDs []int64, now time.Time) (map[int64]service.SubscriptionAddonSummary, error) {
	result := make(map[int64]service.SubscriptionAddonSummary)
	if len(subscriptionIDs) == 0 {
		return result, nil
	}
	if r == nil || r.db == nil {
		return nil, service.ErrSubscriptionAddonInvalid
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT subscription_id,
			SUM(quota_usd),
			SUM(LEAST(used_usd, quota_usd)),
			SUM(GREATEST(quota_usd - used_usd, 0)),
			COUNT(*),
			MIN(expires_at)
		FROM subscription_addon_packs
		WHERE subscription_id = ANY($1)
			AND status = 'active'
			AND starts_at <= $2
			AND expires_at > $2
			AND used_usd < quota_usd
		GROUP BY subscription_id
	`, pq.Array(subscriptionIDs), now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var subscriptionID int64
		var summary service.SubscriptionAddonSummary
		var activePackCount int64
		var nearestExpiresAt time.Time
		if err := rows.Scan(
			&subscriptionID,
			&summary.TotalQuotaUSD,
			&summary.UsedUSD,
			&summary.RemainingUSD,
			&activePackCount,
			&nearestExpiresAt,
		); err != nil {
			return nil, err
		}
		summary.ActivePackCount = int(activePackCount)
		summary.NearestExpiresAt = &nearestExpiresAt
		result[subscriptionID] = summary
	}
	return result, rows.Err()
}

func (r *subscriptionAddonRepository) GetCurrentTermQuotaTotals(ctx context.Context, subscriptionIDs []int64, now time.Time) (map[int64]service.SubscriptionAddonQuotaTotal, error) {
	result := make(map[int64]service.SubscriptionAddonQuotaTotal)
	if len(subscriptionIDs) == 0 {
		return result, nil
	}
	if r == nil || r.db == nil {
		return nil, service.ErrSubscriptionAddonInvalid
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.subscription_id,
			SUM(CASE
				WHEN p.status IN ('active', 'exhausted') AND p.expires_at > $2
					THEN p.quota_usd
				ELSE LEAST(p.used_usd, p.quota_usd)
			END),
			SUM(p.used_usd)
		FROM subscription_addon_packs p
		JOIN user_subscriptions us ON us.id = p.subscription_id
		WHERE p.subscription_id = ANY($1)
			AND p.starts_at >= us.starts_at
			AND p.starts_at < us.expires_at
		GROUP BY p.subscription_id
	`, pq.Array(subscriptionIDs), now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var subscriptionID int64
		var total service.SubscriptionAddonQuotaTotal
		if err := rows.Scan(&subscriptionID, &total.EffectiveQuotaUSD, &total.UsedUSD); err != nil {
			return nil, err
		}
		result[subscriptionID] = total
	}
	return result, rows.Err()
}

func (r *subscriptionAddonRepository) GetGrantedQuotaForTerm(ctx context.Context, subscriptionID int64, startsAt, expiresAt time.Time) (float64, error) {
	if r == nil || r.db == nil {
		return 0, service.ErrSubscriptionAddonInvalid
	}
	var quotaUSD float64
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(quota_usd), 0)
		FROM subscription_addon_packs
		WHERE subscription_id = $1
			AND starts_at >= $2
			AND starts_at < $3
	`, subscriptionID, startsAt, expiresAt).Scan(&quotaUSD)
	return quotaUSD, err
}

func (r *subscriptionAddonRepository) Revoke(ctx context.Context, id int64, revokedAt time.Time) error {
	if r == nil || r.db == nil {
		return service.ErrSubscriptionAddonInvalid
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE subscription_addon_packs
		SET status = 'revoked', revoked_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'active'
	`, id, revokedAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrSubscriptionAddonNotFound
	}
	return nil
}

type subscriptionAddonScanner interface {
	Scan(dest ...any) error
}

func scanSubscriptionAddon(scanner subscriptionAddonScanner) (*service.SubscriptionAddonPack, error) {
	var pack service.SubscriptionAddonPack
	var assignedBy sql.NullInt64
	var revokedAt sql.NullTime
	if err := scanner.Scan(
		&pack.ID,
		&pack.SubscriptionID,
		&pack.UserID,
		&pack.GroupID,
		&pack.QuotaUSD,
		&pack.UsedUSD,
		&pack.StartsAt,
		&pack.ExpiresAt,
		&pack.Status,
		&assignedBy,
		&pack.Notes,
		&revokedAt,
		&pack.CreatedAt,
		&pack.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if assignedBy.Valid {
		pack.AssignedBy = &assignedBy.Int64
	}
	if revokedAt.Valid {
		pack.RevokedAt = &revokedAt.Time
	}
	return &pack, nil
}
