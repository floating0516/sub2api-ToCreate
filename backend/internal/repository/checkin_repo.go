package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type checkinRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewCheckinRepository(client *dbent.Client, sqlDB *sql.DB) service.CheckinRepository {
	return &checkinRepository{client: client, sql: sqlDB}
}

func (r *checkinRepository) GetByUserAndDate(ctx context.Context, userID int64, date string) (*service.UserCheckin, error) {
	return r.queryOne(ctx, `
SELECT id, user_id, checkin_date::text, reward_amount::double precision, streak_days,
       balance_after::double precision, created_at, updated_at
FROM user_checkins
WHERE user_id = $1 AND checkin_date = $2::date
`, userID, date)
}

func (r *checkinRepository) GetLatestBeforeDate(ctx context.Context, userID int64, date string) (*service.UserCheckin, error) {
	return r.queryOne(ctx, `
SELECT id, user_id, checkin_date::text, reward_amount::double precision, streak_days,
       balance_after::double precision, created_at, updated_at
FROM user_checkins
WHERE user_id = $1 AND checkin_date < $2::date
ORDER BY checkin_date DESC
LIMIT 1
`, userID, date)
}

func (r *checkinRepository) GetLatest(ctx context.Context, userID int64) (*service.UserCheckin, error) {
	return r.queryOne(ctx, `
SELECT id, user_id, checkin_date::text, reward_amount::double precision, streak_days,
       balance_after::double precision, created_at, updated_at
FROM user_checkins
WHERE user_id = $1
ORDER BY checkin_date DESC
LIMIT 1
`, userID)
}

func (r *checkinRepository) Create(ctx context.Context, checkin *service.UserCheckin) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return fmt.Errorf("sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, `
INSERT INTO user_checkins (user_id, checkin_date, reward_amount, streak_days)
VALUES ($1, $2::date, $3, $4)
ON CONFLICT (user_id, checkin_date) DO NOTHING
RETURNING id, user_id, checkin_date::text, reward_amount::double precision, streak_days,
          balance_after::double precision, created_at, updated_at
`, checkin.UserID, checkin.CheckinDate, checkin.RewardAmount, checkin.StreakDays)
	if err != nil {
		return translateCheckinPersistenceError(err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return translateCheckinPersistenceError(err)
		}
		return service.ErrCheckinAlreadyClaimed
	}
	if err := scanCheckin(rows, checkin); err != nil {
		return err
	}
	return rows.Err()
}

func (r *checkinRepository) SetBalanceAfter(ctx context.Context, id int64, balance float64) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return fmt.Errorf("sql executor is not configured")
	}
	result, err := exec.ExecContext(ctx, `
UPDATE user_checkins
SET balance_after = $2, updated_at = NOW()
WHERE id = $1
`, id, balance)
	if err != nil {
		return translateCheckinPersistenceError(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrCheckinAlreadyClaimed
	}
	return nil
}

func (r *checkinRepository) ListRecent(ctx context.Context, userID int64, limit int) ([]service.UserCheckin, error) {
	if limit <= 0 || limit > 60 {
		limit = 30
	}
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, fmt.Errorf("sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, `
SELECT id, user_id, checkin_date::text, reward_amount::double precision, streak_days,
       balance_after::double precision, created_at, updated_at
FROM user_checkins
WHERE user_id = $1
ORDER BY checkin_date DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, translateCheckinPersistenceError(err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.UserCheckin, 0)
	for rows.Next() {
		var item service.UserCheckin
		if err := scanCheckin(rows, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *checkinRepository) queryOne(ctx context.Context, query string, args ...any) (*service.UserCheckin, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, fmt.Errorf("sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, translateCheckinPersistenceError(err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	var item service.UserCheckin
	if err := scanCheckin(rows, &item); err != nil {
		return nil, err
	}
	return &item, rows.Err()
}

type checkinScanner interface {
	Scan(dest ...any) error
}

func scanCheckin(scanner checkinScanner, item *service.UserCheckin) error {
	var balanceAfter sql.NullFloat64
	var updatedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.CheckinDate,
		&item.RewardAmount,
		&item.StreakDays,
		&balanceAfter,
		&item.CreatedAt,
		&updatedAt,
	); err != nil {
		return err
	}
	if balanceAfter.Valid {
		item.BalanceAfter = &balanceAfter.Float64
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return nil
}

func translateCheckinPersistenceError(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if ok := errors.As(err, &pqErr); ok && pqErr.Code == "23505" {
		return service.ErrCheckinAlreadyClaimed
	}
	return err
}
