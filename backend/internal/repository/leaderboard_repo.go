package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type leaderboardRepository struct {
	db *sql.DB
}

func NewLeaderboardRepository(db *sql.DB) service.LeaderboardRepository {
	return &leaderboardRepository{db: db}
}

func (r *leaderboardRepository) GetPreference(ctx context.Context, userID int64) (*service.LeaderboardPreference, error) {
	var pref service.LeaderboardPreference
	err := r.db.QueryRowContext(ctx, `
SELECT user_id, anonymous, updated_at
FROM leaderboard_preferences
WHERE user_id = $1`, userID).Scan(&pref.UserID, &pref.Anonymous, &pref.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &service.LeaderboardPreference{UserID: userID, Anonymous: false}, nil
		}
		return nil, err
	}
	return &pref, nil
}

func (r *leaderboardRepository) SetPreference(ctx context.Context, userID int64, anonymous bool) (*service.LeaderboardPreference, error) {
	var pref service.LeaderboardPreference
	err := r.db.QueryRowContext(ctx, `
INSERT INTO leaderboard_preferences (user_id, anonymous, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id)
DO UPDATE SET anonymous = EXCLUDED.anonymous, updated_at = EXCLUDED.updated_at
RETURNING user_id, anonymous, updated_at`, userID, anonymous).Scan(&pref.UserID, &pref.Anonymous, &pref.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *leaderboardRepository) GetTodayTokens(ctx context.Context, userID int64, start, end time.Time) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0)
FROM usage_logs
WHERE user_id = $1 AND created_at >= $2 AND created_at < $3`, userID, start, end).Scan(&total)
	return total, err
}

func (r *leaderboardRepository) ListRankings(ctx context.Context, start, end time.Time, limit int) ([]service.LeaderboardEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH ranked AS (
  SELECT
    ul.user_id,
    COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0)::BIGINT AS token_count,
    RANK() OVER (ORDER BY COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) DESC, ul.user_id ASC) AS rank
  FROM usage_logs ul
  JOIN users u ON u.id = ul.user_id
  WHERE ul.created_at >= $1
    AND ul.created_at < $2
    AND u.deleted_at IS NULL
    AND u.status = 'active'
  GROUP BY ul.user_id
)
SELECT r.rank, r.user_id, COALESCE(u.username, ''), u.email, u.role, COALESCE(lp.anonymous, false), r.token_count
FROM ranked r
JOIN users u ON u.id = r.user_id
LEFT JOIN leaderboard_preferences lp ON lp.user_id = r.user_id
WHERE r.token_count > 0
ORDER BY r.rank ASC, r.user_id ASC
LIMIT $3`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []service.LeaderboardEntry
	for rows.Next() {
		var entry service.LeaderboardEntry
		if err := rows.Scan(&entry.Rank, &entry.UserID, &entry.Username, &entry.Email, &entry.Role, &entry.Anonymous, &entry.TokenCount); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *leaderboardRepository) GetUserRank(ctx context.Context, userID int64, start, end time.Time) (*service.LeaderboardEntry, error) {
	var entry service.LeaderboardEntry
	err := r.db.QueryRowContext(ctx, `
WITH ranked AS (
  SELECT
    ul.user_id,
    COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0)::BIGINT AS token_count,
    RANK() OVER (ORDER BY COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) DESC, ul.user_id ASC) AS rank
  FROM usage_logs ul
  JOIN users u ON u.id = ul.user_id
  WHERE ul.created_at >= $1
    AND ul.created_at < $2
    AND u.deleted_at IS NULL
    AND u.status = 'active'
  GROUP BY ul.user_id
)
SELECT r.rank, r.user_id, COALESCE(u.username, ''), u.email, u.role, COALESCE(lp.anonymous, false), r.token_count
FROM ranked r
JOIN users u ON u.id = r.user_id
LEFT JOIN leaderboard_preferences lp ON lp.user_id = r.user_id
WHERE r.user_id = $3 AND r.token_count > 0`, start, end, userID).
		Scan(&entry.Rank, &entry.UserID, &entry.Username, &entry.Email, &entry.Role, &entry.Anonymous, &entry.TokenCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &entry, nil
}

func (r *leaderboardRepository) CreateReward(ctx context.Context, reward *service.LeaderboardReward) error {
	if reward == nil {
		return errors.New("nil leaderboard reward")
	}
	err := r.db.QueryRowContext(ctx, `
INSERT INTO leaderboard_rewards
  (period, period_start, period_end, rank, user_id, token_count, reward_type, redeem_code_id, created_by)
VALUES
  ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, created_at`,
		reward.Period,
		reward.PeriodStart,
		reward.PeriodEnd,
		reward.Rank,
		reward.UserID,
		reward.TokenCount,
		reward.RewardType,
		reward.RedeemCodeID,
		reward.CreatedBy,
	).Scan(&reward.ID, &reward.CreatedAt)
	return err
}

func (r *leaderboardRepository) ListRewards(ctx context.Context, period string, start time.Time) ([]service.LeaderboardReward, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT lr.id, lr.period, lr.period_start, lr.period_end, lr.rank, lr.user_id, lr.token_count,
       lr.reward_type, lr.redeem_code_id, rc.code, lr.created_by, lr.created_at
FROM leaderboard_rewards lr
JOIN redeem_codes rc ON rc.id = lr.redeem_code_id
WHERE lr.period = $1 AND lr.period_start = $2
ORDER BY lr.rank ASC, lr.created_at ASC`, period, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rewards []service.LeaderboardReward
	for rows.Next() {
		var reward service.LeaderboardReward
		var createdBy sql.NullInt64
		if err := rows.Scan(
			&reward.ID,
			&reward.Period,
			&reward.PeriodStart,
			&reward.PeriodEnd,
			&reward.Rank,
			&reward.UserID,
			&reward.TokenCount,
			&reward.RewardType,
			&reward.RedeemCodeID,
			&reward.RedeemCode,
			&createdBy,
			&reward.CreatedAt,
		); err != nil {
			return nil, err
		}
		if createdBy.Valid {
			reward.CreatedBy = &createdBy.Int64
		}
		rewards = append(rewards, reward)
	}
	return rewards, rows.Err()
}
