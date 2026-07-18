package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	openAIQuotaWindow7d             = "7d"
	openAIQuotaResetDropMinimum     = 0.5
	openAIQuotaResetStrongDrop      = 5.0
	openAIQuotaResetForecastAdvance = time.Hour
	openAIQuotaResetBoundaryGrace   = 5 * time.Minute
)

type openAIQuotaSnapshot struct {
	ObservedAt time.Time
	Used       float64
	ResetAt    *time.Time
}

type openAIQuotaActiveCycle struct {
	ID              int64
	LastObservedAt  time.Time
	LastUsedPercent float64
	PeakUsedPercent float64
	ProviderResetAt *time.Time
}

func extractOpenAIQuotaSnapshot(updates map[string]any) (*openAIQuotaSnapshot, bool) {
	used, ok := quotaHistoryFloat(updates["codex_7d_used_percent"])
	if !ok || math.IsNaN(used) || math.IsInf(used, 0) || used < 0 || used > 100 {
		return nil, false
	}

	observedAt, ok := quotaHistoryTime(updates["codex_usage_updated_at"])
	if !ok {
		observedAt = time.Now().UTC()
	}
	observedAt = observedAt.UTC()

	var resetAt *time.Time
	if parsed, parsedOK := quotaHistoryTime(updates["codex_7d_reset_at"]); parsedOK {
		parsed = parsed.UTC()
		resetAt = &parsed
	}

	return &openAIQuotaSnapshot{
		ObservedAt: observedAt,
		Used:       used,
		ResetAt:    resetAt,
	}, true
}

func quotaHistoryFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func quotaHistoryTime(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return time.Time{}, false
	}
}

func detectOpenAIQuotaReset(previous *openAIQuotaActiveCycle, next *openAIQuotaSnapshot) string {
	if previous == nil || next == nil {
		return ""
	}

	drop := previous.LastUsedPercent - next.Used
	forecastAdvanced := previous.ProviderResetAt != nil && next.ResetAt != nil &&
		next.ResetAt.After(previous.ProviderResetAt.Add(openAIQuotaResetForecastAdvance))

	if drop >= openAIQuotaResetStrongDrop {
		return "usage_drop"
	}
	if drop >= openAIQuotaResetDropMinimum && forecastAdvanced {
		return "usage_drop"
	}
	if forecastAdvanced && !next.ObservedAt.Before(previous.ProviderResetAt.Add(-openAIQuotaResetBoundaryGrace)) {
		return "window_elapsed"
	}
	return ""
}

func recordOpenAIQuotaCycle(ctx context.Context, q sqlExecutor, accountID int64, snapshot *openAIQuotaSnapshot) error {
	if q == nil || snapshot == nil || accountID <= 0 {
		return nil
	}

	var platform, accountType string
	var parentAccountID sql.NullInt64
	var quotaDimension sql.NullString
	err := scanSingleRow(ctx, q, `
		SELECT platform, type, parent_account_id, quota_dimension
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`, []any{accountID}, &platform, &accountType, &parentAccountID, &quotaDimension)
	if err != nil {
		return err
	}
	if platform != service.PlatformOpenAI || accountType != service.AccountTypeOAuth || parentAccountID.Valid {
		return nil
	}
	if quotaDimension.Valid && quotaDimension.String != "" && quotaDimension.String != "global" {
		return nil
	}

	active, err := getOpenAIQuotaActiveCycleForUpdate(ctx, q, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = q.ExecContext(ctx, `
			INSERT INTO openai_quota_cycles (
				account_id, window_type, cycle_started_at, last_observed_at,
				last_used_percent, peak_used_percent, provider_reset_at
			) VALUES ($1, $2, $3, $3, $4, $4, $5)
		`, accountID, openAIQuotaWindow7d, snapshot.ObservedAt, snapshot.Used, snapshot.ResetAt)
		return err
	}
	if err != nil {
		return err
	}
	if !snapshot.ObservedAt.After(active.LastObservedAt) {
		return nil
	}

	if reason := detectOpenAIQuotaReset(active, snapshot); reason != "" {
		if _, err = q.ExecContext(ctx, `
			UPDATE openai_quota_cycles
			SET reset_observed_at = $1,
				reset_to_percent = $2,
				detection_reason = $3,
				updated_at = NOW()
			WHERE id = $4 AND reset_observed_at IS NULL
		`, snapshot.ObservedAt, snapshot.Used, reason, active.ID); err != nil {
			return err
		}
		_, err = q.ExecContext(ctx, `
			INSERT INTO openai_quota_cycles (
				account_id, window_type, cycle_started_at, last_observed_at,
				last_used_percent, peak_used_percent, provider_reset_at
			) VALUES ($1, $2, $3, $3, $4, $4, $5)
		`, accountID, openAIQuotaWindow7d, snapshot.ObservedAt, snapshot.Used, snapshot.ResetAt)
		return err
	}

	_, err = q.ExecContext(ctx, `
		UPDATE openai_quota_cycles
		SET last_observed_at = $1,
			last_used_percent = $2,
			peak_used_percent = GREATEST(peak_used_percent, $2),
			provider_reset_at = $3,
			updated_at = NOW()
		WHERE id = $4 AND reset_observed_at IS NULL
	`, snapshot.ObservedAt, snapshot.Used, snapshot.ResetAt, active.ID)
	return err
}

func getOpenAIQuotaActiveCycleForUpdate(ctx context.Context, q sqlQueryer, accountID int64) (*openAIQuotaActiveCycle, error) {
	var cycle openAIQuotaActiveCycle
	var providerResetAt sql.NullTime
	err := scanSingleRow(ctx, q, `
		SELECT id, last_observed_at, last_used_percent, peak_used_percent, provider_reset_at
		FROM openai_quota_cycles
		WHERE account_id = $1 AND window_type = $2 AND reset_observed_at IS NULL
		FOR UPDATE
	`, []any{accountID, openAIQuotaWindow7d},
		&cycle.ID,
		&cycle.LastObservedAt,
		&cycle.LastUsedPercent,
		&cycle.PeakUsedPercent,
		&providerResetAt,
	)
	if err != nil {
		return nil, err
	}
	if providerResetAt.Valid {
		resetAt := providerResetAt.Time
		cycle.ProviderResetAt = &resetAt
	}
	return &cycle, nil
}

func (r *accountRepository) GetOpenAIQuotaHistory(ctx context.Context, accountID int64, limit int) (*service.OpenAIQuotaHistoryResponse, error) {
	if r == nil || r.sql == nil {
		return nil, fmt.Errorf("nil account repository")
	}
	if limit <= 0 {
		limit = 20
	}

	result := &service.OpenAIQuotaHistoryResponse{History: []service.OpenAIQuotaCycle{}}
	current, err := scanOpenAIQuotaCycle(ctx, r.sql, `
		SELECT id, cycle_started_at, last_observed_at, last_used_percent,
			peak_used_percent, provider_reset_at, reset_observed_at,
			reset_to_percent, COALESCE(detection_reason, '')
		FROM openai_quota_cycles
		WHERE account_id = $1 AND window_type = $2 AND reset_observed_at IS NULL
		LIMIT 1
	`, accountID, openAIQuotaWindow7d)
	if err == nil {
		result.Current = current
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, cycle_started_at, last_observed_at, last_used_percent,
			peak_used_percent, provider_reset_at, reset_observed_at,
			reset_to_percent, COALESCE(detection_reason, '')
		FROM openai_quota_cycles
		WHERE account_id = $1 AND window_type = $2 AND reset_observed_at IS NOT NULL
		ORDER BY reset_observed_at DESC, id DESC
		LIMIT $3
	`, accountID, openAIQuotaWindow7d, limit+1)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		cycle, scanErr := scanOpenAIQuotaCycleRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		if len(result.History) == limit {
			result.HasMore = true
			continue
		}
		result.History = append(result.History, *cycle)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanOpenAIQuotaCycle(ctx context.Context, q sqlQueryer, query string, args ...any) (*service.OpenAIQuotaCycle, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	return scanOpenAIQuotaCycleRow(rows)
}

func scanOpenAIQuotaCycleRow(scanner interface{ Scan(dest ...any) error }) (*service.OpenAIQuotaCycle, error) {
	var cycle service.OpenAIQuotaCycle
	var providerResetAt, resetObservedAt sql.NullTime
	var resetToPercent sql.NullFloat64
	if err := scanner.Scan(
		&cycle.ID,
		&cycle.CycleStartedAt,
		&cycle.LastObservedAt,
		&cycle.LastUsedPercent,
		&cycle.PeakUsedPercent,
		&providerResetAt,
		&resetObservedAt,
		&resetToPercent,
		&cycle.DetectionReason,
	); err != nil {
		return nil, err
	}
	if providerResetAt.Valid {
		value := providerResetAt.Time
		cycle.ProviderResetAt = &value
	}
	if resetObservedAt.Valid {
		value := resetObservedAt.Time
		cycle.ResetObservedAt = &value
	}
	if resetToPercent.Valid {
		value := resetToPercent.Float64
		cycle.ResetToPercent = &value
	}
	return &cycle, nil
}
