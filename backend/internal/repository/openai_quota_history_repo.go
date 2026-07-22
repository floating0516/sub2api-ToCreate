package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	openAIQuotaWindow7d              = "7d"
	openAIQuotaResetDropMinimum      = 0.5
	openAIQuotaResetStrongDrop       = 5.0
	openAIQuotaResetForecastAdvance  = time.Hour
	openAIQuotaResetBoundaryGrace    = 5 * time.Minute
	openAIQuotaTransientLowMaximum   = 5.0
	openAIQuotaSampleBucket          = 5 * time.Minute
	openAIQuotaSampleRetention       = 90 * 24 * time.Hour
	openAIQuotaResetEvidenceEndpoint = "reset_endpoint"
	openAIQuotaResetEvidenceElapsed  = "window_elapsed"
	openAIQuotaResetEvidenceConsumed = "credit_consumed"
	openAIQuotaResetEvidenceSame     = "credits_unchanged"
	openAIQuotaResetEvidenceExpired  = "credits_expired_only"
)

type openAIQuotaSnapshot struct {
	ObservedAt time.Time
	Used       float64
	ResetAt    *time.Time
}

type openAIQuotaActiveCycle struct {
	ID              int64
	CycleStartedAt  time.Time
	LastObservedAt  time.Time
	LastUsedPercent float64
	PeakUsedPercent float64
	ProviderResetAt *time.Time
}

func automaticOpenAIQuotaResetSource(reason string) (string, string) {
	switch reason {
	case "manual_reset":
		return service.OpenAIQuotaResetSourceManual, openAIQuotaResetEvidenceEndpoint
	case "window_elapsed":
		return service.OpenAIQuotaResetSourceProvider, openAIQuotaResetEvidenceElapsed
	default:
		return service.OpenAIQuotaResetSourceUnknown, ""
	}
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
	peakDrop := previous.PeakUsedPercent - next.Used
	forecastAdvanced := previous.ProviderResetAt != nil && next.ResetAt != nil &&
		next.ResetAt.After(previous.ProviderResetAt.Add(openAIQuotaResetForecastAdvance))

	if forecastAdvanced && (drop >= openAIQuotaResetDropMinimum || peakDrop >= openAIQuotaResetStrongDrop) {
		return "usage_drop"
	}
	if forecastAdvanced && !next.ObservedAt.Before(previous.ProviderResetAt.Add(-openAIQuotaResetBoundaryGrace)) {
		return "window_elapsed"
	}
	return ""
}

func isUncorroboratedOpenAIQuotaLow(previous *openAIQuotaActiveCycle, next *openAIQuotaSnapshot) bool {
	if previous == nil || next == nil || next.Used > openAIQuotaTransientLowMaximum {
		return false
	}
	if previous.LastUsedPercent-next.Used < openAIQuotaResetDropMinimum {
		return false
	}
	return previous.ProviderResetAt == nil || next.ResetAt == nil ||
		!next.ResetAt.After(previous.ProviderResetAt.Add(openAIQuotaResetForecastAdvance))
}

func isOpenAIQuotaHistoryAccount(ctx context.Context, q sqlQueryer, accountID int64) (bool, error) {
	var platform, accountType string
	var parentAccountID sql.NullInt64
	var quotaDimension sql.NullString
	err := scanSingleRow(ctx, q, `
		SELECT platform, type, parent_account_id, quota_dimension
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`, []any{accountID}, &platform, &accountType, &parentAccountID, &quotaDimension)
	if err != nil {
		return false, err
	}
	if platform != service.PlatformOpenAI || accountType != service.AccountTypeOAuth || parentAccountID.Valid {
		return false, nil
	}
	return !quotaDimension.Valid || quotaDimension.String == "" || quotaDimension.String == "global", nil
}

func recordOpenAIQuotaCycle(ctx context.Context, q sqlExecutor, accountID int64, snapshot *openAIQuotaSnapshot) error {
	if q == nil || snapshot == nil || accountID <= 0 {
		return nil
	}

	eligible, err := isOpenAIQuotaHistoryAccount(ctx, q, accountID)
	if err != nil {
		return err
	}
	if !eligible {
		return nil
	}

	active, err := getOpenAIQuotaActiveCycleForUpdate(ctx, q, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		var cycleID int64
		err = scanSingleRow(ctx, q, `
			INSERT INTO openai_quota_cycles (
				account_id, window_type, cycle_started_at, last_observed_at,
				last_used_percent, peak_used_percent, provider_reset_at
			) VALUES ($1, $2, $3, $3, $4, $4, $5)
			RETURNING id
		`, []any{accountID, openAIQuotaWindow7d, snapshot.ObservedAt, snapshot.Used, snapshot.ResetAt}, &cycleID)
		if err != nil {
			return err
		}
		return recordOpenAIQuotaSample(ctx, q, accountID, cycleID, snapshot)
	}
	if err != nil {
		return err
	}
	if !snapshot.ObservedAt.After(active.LastObservedAt) {
		return nil
	}

	if reason := detectOpenAIQuotaReset(active, snapshot); reason != "" {
		resetSource, evidence := automaticOpenAIQuotaResetSource(reason)
		if _, err = q.ExecContext(ctx, `
			UPDATE openai_quota_cycles
			SET reset_observed_at = $1,
				reset_to_percent = $2,
				detection_reason = $3,
				detected_reset_source = $4,
				reset_source_evidence = NULLIF($5, ''),
				updated_at = NOW()
			WHERE id = $6 AND reset_observed_at IS NULL
		`, snapshot.ObservedAt, snapshot.Used, reason, resetSource, evidence, active.ID); err != nil {
			return err
		}
		var cycleID int64
		err = scanSingleRow(ctx, q, `
			INSERT INTO openai_quota_cycles (
				account_id, window_type, cycle_started_at, last_observed_at,
				last_used_percent, peak_used_percent, provider_reset_at
			) VALUES ($1, $2, $3, $3, $4, $4, $5)
			RETURNING id
		`, []any{accountID, openAIQuotaWindow7d, snapshot.ObservedAt, snapshot.Used, snapshot.ResetAt}, &cycleID)
		if err != nil {
			return err
		}
		return recordOpenAIQuotaSample(ctx, q, accountID, cycleID, snapshot)
	}
	// Startup probes can briefly emit 0% without moving the provider's reset
	// forecast. Keep the last corroborated point until a complete snapshot arrives.
	if isUncorroboratedOpenAIQuotaLow(active, snapshot) {
		return nil
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
	if err != nil {
		return err
	}
	return recordOpenAIQuotaSample(ctx, q, accountID, active.ID, snapshot)
}

func recordOpenAIQuotaManualReset(ctx context.Context, q sqlExecutor, accountID int64, observedAt time.Time) error {
	if q == nil || accountID <= 0 {
		return nil
	}
	eligible, err := isOpenAIQuotaHistoryAccount(ctx, q, accountID)
	if err != nil {
		return err
	}
	if !eligible {
		return nil
	}

	observedAt = observedAt.UTC()
	snapshot := &openAIQuotaSnapshot{ObservedAt: observedAt, Used: 0}
	active, err := getOpenAIQuotaActiveCycleForUpdate(ctx, q, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		var cycleID int64
		err = scanSingleRow(ctx, q, `
			INSERT INTO openai_quota_cycles (
				account_id, window_type, cycle_started_at, last_observed_at,
				last_used_percent, peak_used_percent
			) VALUES ($1, $2, $3, $3, 0, 0)
			RETURNING id
		`, []any{accountID, openAIQuotaWindow7d, observedAt}, &cycleID)
		if err != nil {
			return err
		}
		return recordOpenAIQuotaSample(ctx, q, accountID, cycleID, snapshot)
	}
	if err != nil {
		return err
	}
	if !observedAt.After(active.LastObservedAt) {
		observedAt = active.LastObservedAt.Add(time.Nanosecond)
		snapshot.ObservedAt = observedAt
	}

	if _, err = q.ExecContext(ctx, `
		UPDATE openai_quota_cycles
		SET reset_observed_at = $1,
			reset_to_percent = 0,
			detection_reason = 'manual_reset',
			detected_reset_source = 'manual',
			reset_source_evidence = $2,
			updated_at = NOW()
		WHERE id = $3 AND reset_observed_at IS NULL
	`, observedAt, openAIQuotaResetEvidenceEndpoint, active.ID); err != nil {
		return err
	}
	var cycleID int64
	err = scanSingleRow(ctx, q, `
		INSERT INTO openai_quota_cycles (
			account_id, window_type, cycle_started_at, last_observed_at,
			last_used_percent, peak_used_percent
		) VALUES ($1, $2, $3, $3, 0, 0)
		RETURNING id
	`, []any{accountID, openAIQuotaWindow7d, observedAt}, &cycleID)
	if err != nil {
		return err
	}
	return recordOpenAIQuotaSample(ctx, q, accountID, cycleID, snapshot)
}

func recordOpenAIQuotaSample(
	ctx context.Context,
	q sqlExecutor,
	accountID, cycleID int64,
	snapshot *openAIQuotaSnapshot,
) error {
	if q == nil || snapshot == nil || accountID <= 0 || cycleID <= 0 {
		return nil
	}

	bucketStartedAt := snapshot.ObservedAt.UTC().Truncate(openAIQuotaSampleBucket)
	result, err := q.ExecContext(ctx, `
		INSERT INTO openai_quota_samples (
			account_id, cycle_id, bucket_started_at, observed_at, used_percent
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (cycle_id, bucket_started_at) DO NOTHING
	`, accountID, cycleID, bucketStartedAt, snapshot.ObservedAt, snapshot.Used)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		_, err = q.ExecContext(ctx, `
			UPDATE openai_quota_samples
			SET observed_at = $1,
				used_percent = $2,
				updated_at = NOW()
			WHERE cycle_id = $3
				AND bucket_started_at = $4
				AND observed_at < $1
		`, snapshot.ObservedAt, snapshot.Used, cycleID, bucketStartedAt)
		return err
	}

	_, err = q.ExecContext(ctx, `
		DELETE FROM openai_quota_samples
		WHERE account_id = $1 AND observed_at < $2
	`, accountID, snapshot.ObservedAt.Add(-openAIQuotaSampleRetention))
	return err
}

func getOpenAIQuotaActiveCycleForUpdate(ctx context.Context, q sqlQueryer, accountID int64) (*openAIQuotaActiveCycle, error) {
	var cycle openAIQuotaActiveCycle
	var providerResetAt sql.NullTime
	err := scanSingleRow(ctx, q, `
		SELECT id, cycle_started_at, last_observed_at, last_used_percent, peak_used_percent, provider_reset_at
		FROM openai_quota_cycles
		WHERE account_id = $1 AND window_type = $2 AND reset_observed_at IS NULL
		FOR UPDATE
	`, []any{accountID, openAIQuotaWindow7d},
		&cycle.ID,
		&cycle.CycleStartedAt,
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

func recordOpenAIQuotaObservation(
	ctx context.Context,
	q sqlExecutor,
	accountID int64,
	observation *service.OpenAIQuotaObservation,
) error {
	if q == nil || observation == nil || accountID <= 0 {
		return nil
	}
	if observation.UsedPercent != nil {
		used := *observation.UsedPercent
		if !math.IsNaN(used) && !math.IsInf(used, 0) && used >= 0 && used <= 100 {
			if err := recordOpenAIQuotaCycle(ctx, q, accountID, &openAIQuotaSnapshot{
				ObservedAt: observation.ObservedAt.UTC(),
				Used:       used,
				ResetAt:    observation.ProviderResetAt,
			}); err != nil {
				return err
			}
		}
	}
	if !observation.CreditSnapshotKnown {
		return nil
	}
	return recordOpenAIQuotaCreditSnapshot(
		ctx,
		q,
		accountID,
		observation.ObservedAt.UTC(),
		observation.CreditExpiresAt,
	)
}

func recordOpenAIQuotaCreditSnapshot(
	ctx context.Context,
	q sqlExecutor,
	accountID int64,
	observedAt time.Time,
	expiresAt []time.Time,
) error {
	eligible, err := isOpenAIQuotaHistoryAccount(ctx, q, accountID)
	if err != nil || !eligible {
		return err
	}
	active, err := getOpenAIQuotaActiveCycleForUpdate(ctx, q, accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if observedAt.Before(active.CycleStartedAt) {
		return nil
	}

	normalized := normalizeOpenAIQuotaCreditExpirations(expiresAt)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	result, err := q.ExecContext(ctx, `
		UPDATE openai_quota_cycles
		SET reset_credit_snapshot_at = $1,
			reset_credit_expirations = $2::jsonb,
			updated_at = NOW()
		WHERE id = $3
			AND (reset_credit_snapshot_at IS NULL OR reset_credit_snapshot_at < $1)
	`, observedAt, string(payload), active.ID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated == 0 {
		return err
	}

	var previousID int64
	var resetObservedAt time.Time
	var previousSnapshotAt sql.NullTime
	var previousPayload []byte
	err = scanSingleRow(ctx, q, `
		SELECT id, reset_observed_at, reset_credit_snapshot_at, reset_credit_expirations
		FROM openai_quota_cycles
		WHERE account_id = $1
			AND window_type = $2
			AND reset_observed_at IS NOT NULL
			AND reset_observed_at <= $3
		ORDER BY reset_observed_at DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`, []any{accountID, openAIQuotaWindow7d, active.CycleStartedAt},
		&previousID,
		&resetObservedAt,
		&previousSnapshotAt,
		&previousPayload,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !previousSnapshotAt.Valid || previousSnapshotAt.Time.After(resetObservedAt) || observedAt.Before(resetObservedAt) {
		return nil
	}
	previousExpirations, err := decodeOpenAIQuotaCreditExpirations(previousPayload)
	if err != nil {
		return err
	}
	resetSource, evidence := classifyOpenAIQuotaResetFromCredits(previousExpirations, expiresAt, observedAt)
	_, err = q.ExecContext(ctx, `
		UPDATE openai_quota_cycles
		SET detected_reset_source = $1,
			reset_source_evidence = $2,
			updated_at = NOW()
		WHERE id = $3 AND detected_reset_source = 'unknown'
	`, resetSource, evidence, previousID)
	return err
}

func normalizeOpenAIQuotaCreditExpirations(values []time.Time) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, value.UTC().Format(time.RFC3339Nano))
	}
	sort.Strings(normalized)
	return normalized
}

func decodeOpenAIQuotaCreditExpirations(payload []byte) ([]time.Time, error) {
	var values []string
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}
	result := make([]time.Time, 0, len(values))
	for _, value := range values {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed.UTC())
	}
	return result, nil
}

func classifyOpenAIQuotaResetFromCredits(before, after []time.Time, afterObservedAt time.Time) (string, string) {
	afterCounts := make(map[string]int, len(after))
	for _, expiry := range after {
		afterCounts[expiry.UTC().Format(time.RFC3339Nano)]++
	}
	expiredOnly := false
	for _, expiry := range before {
		key := expiry.UTC().Format(time.RFC3339Nano)
		if afterCounts[key] > 0 {
			afterCounts[key]--
			continue
		}
		if expiry.After(afterObservedAt) {
			return service.OpenAIQuotaResetSourceManual, openAIQuotaResetEvidenceConsumed
		}
		expiredOnly = true
	}
	if expiredOnly {
		return service.OpenAIQuotaResetSourceProvider, openAIQuotaResetEvidenceExpired
	}
	return service.OpenAIQuotaResetSourceProvider, openAIQuotaResetEvidenceSame
}

func (r *accountRepository) GetOpenAIQuotaHistory(ctx context.Context, accountID int64, limit int) (*service.OpenAIQuotaHistoryResponse, error) {
	if r == nil || r.sql == nil {
		return nil, fmt.Errorf("nil account repository")
	}
	if limit <= 0 {
		limit = 20
	}

	result := &service.OpenAIQuotaHistoryResponse{
		History: []service.OpenAIQuotaCycle{},
		Samples: []service.OpenAIQuotaSample{},
	}
	current, err := scanOpenAIQuotaCycle(ctx, r.sql, `
		SELECT id, cycle_started_at, last_observed_at, last_used_percent,
			peak_used_percent, provider_reset_at, reset_observed_at,
			reset_to_percent, COALESCE(detection_reason, ''),
			COALESCE(detected_reset_source, 'unknown'), reset_source_override,
			COALESCE(reset_source_evidence, '')
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
		WITH ordered_cycles AS (
			SELECT id, cycle_started_at, last_observed_at, last_used_percent,
				peak_used_percent, provider_reset_at, reset_observed_at,
				reset_to_percent, COALESCE(detection_reason, '') AS detection_reason,
				COALESCE(detected_reset_source, 'unknown') AS detected_reset_source,
				reset_source_override,
				COALESCE(reset_source_evidence, '') AS reset_source_evidence,
				LEAD(cycle_started_at) OVER (
					ORDER BY cycle_started_at ASC, id ASC
				) AS next_cycle_started_at,
				LEAD(provider_reset_at) OVER (
					ORDER BY cycle_started_at ASC, id ASC
				) AS next_provider_reset_at
			FROM openai_quota_cycles
			WHERE account_id = $1 AND window_type = $2
		)
		SELECT id, cycle_started_at, last_observed_at, last_used_percent,
			peak_used_percent, provider_reset_at, reset_observed_at,
			reset_to_percent, detection_reason, detected_reset_source,
			reset_source_override, reset_source_evidence
		FROM ordered_cycles
		WHERE reset_observed_at IS NOT NULL
			AND NOT (
				detection_reason = 'usage_drop'
				AND reset_source_override IS NULL
				AND next_cycle_started_at = reset_observed_at
				AND provider_reset_at IS NOT NULL
				AND next_provider_reset_at IS NOT NULL
				AND next_provider_reset_at <= provider_reset_at + INTERVAL '1 hour'
			)
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

	sampleRows, err := r.sql.QueryContext(ctx, `
		WITH chart_samples AS (
			SELECT DISTINCT ON (
				cycle_id,
				date_bin('15 minutes', observed_at, TIMESTAMPTZ '1970-01-01 00:00:00+00')
			)
				id, cycle_id, observed_at, used_percent
			FROM openai_quota_samples
			WHERE account_id = $1
				AND observed_at >= NOW() - INTERVAL '30 days'
			ORDER BY
				cycle_id,
				date_bin('15 minutes', observed_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
				observed_at DESC,
				id DESC
		), sampled_cycles AS (
			SELECT
				cycles.id,
				CASE
					WHEN cycles.provider_reset_at IS NOT NULL
						AND cycles.provider_reset_at > cycles.cycle_started_at
						AND cycles.provider_reset_at <= cycles.cycle_started_at + INTERVAL '7 days'
					THEN cycles.provider_reset_at - INTERVAL '7 days'
					ELSE cycles.cycle_started_at
				END AS usage_started_at,
				MAX(chart_samples.observed_at) AS usage_ended_at
			FROM openai_quota_cycles AS cycles
			JOIN chart_samples ON chart_samples.cycle_id = cycles.id
			GROUP BY cycles.id, cycles.cycle_started_at, cycles.provider_reset_at
		), usage_timeline AS (
			SELECT
				chart_samples.cycle_id,
				chart_samples.observed_at AS event_at,
				1 AS event_order,
				chart_samples.id AS event_id,
				chart_samples.id AS sample_id,
				chart_samples.used_percent,
				0::NUMERIC AS cost_delta
			FROM chart_samples

			UNION ALL

			SELECT
				sampled_cycles.id AS cycle_id,
				usage_logs.created_at AS event_at,
				0 AS event_order,
				usage_logs.id AS event_id,
				NULL::BIGINT AS sample_id,
				NULL::DOUBLE PRECISION AS used_percent,
				COALESCE(usage_logs.account_stats_cost, usage_logs.total_cost)
					* COALESCE(usage_logs.account_rate_multiplier, 1) AS cost_delta
			FROM sampled_cycles
			JOIN usage_logs
				ON usage_logs.account_id = $1
				AND usage_logs.created_at >= sampled_cycles.usage_started_at
				AND usage_logs.created_at <= sampled_cycles.usage_ended_at
		), samples_with_usage AS (
			SELECT
				cycle_id,
				event_at,
				event_order,
				event_id,
				sample_id,
				used_percent,
				CAST(SUM(cost_delta) OVER (
					PARTITION BY cycle_id
					ORDER BY event_at ASC, event_order ASC, event_id ASC
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) AS DOUBLE PRECISION) AS local_cost_usd
			FROM usage_timeline
		)
		SELECT cycle_id, event_at, used_percent, local_cost_usd
		FROM samples_with_usage
		WHERE sample_id IS NOT NULL
		ORDER BY event_at ASC, event_id ASC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sampleRows.Close() }()

	for sampleRows.Next() {
		var sample service.OpenAIQuotaSample
		if err = sampleRows.Scan(
			&sample.CycleID,
			&sample.ObservedAt,
			&sample.UsedPercent,
			&sample.LocalCostUSD,
		); err != nil {
			return nil, err
		}
		result.Samples = append(result.Samples, sample)
	}
	if err = sampleRows.Err(); err != nil {
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
	var resetSourceOverride sql.NullString
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
		&cycle.AutomaticResetSource,
		&resetSourceOverride,
		&cycle.ResetSourceEvidence,
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
	if cycle.AutomaticResetSource == "" {
		cycle.AutomaticResetSource = service.OpenAIQuotaResetSourceUnknown
	}
	cycle.ResetSource = cycle.AutomaticResetSource
	if resetSourceOverride.Valid {
		value := resetSourceOverride.String
		cycle.ResetSourceOverride = &value
		cycle.ResetSource = value
	}
	return &cycle, nil
}

func (r *accountRepository) SetOpenAIQuotaResetSource(
	ctx context.Context,
	accountID, cycleID int64,
	resetSourceOverride *string,
) error {
	if r == nil || r.client == nil {
		return errors.New("nil account repository")
	}
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, `
		UPDATE openai_quota_cycles
		SET reset_source_override = $1,
			updated_at = NOW()
		WHERE id = $2
			AND account_id = $3
			AND window_type = $4
			AND reset_observed_at IS NOT NULL
	`, resetSourceOverride, cycleID, accountID, openAIQuotaWindow7d)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return service.ErrOpenAIQuotaCycleNotFound
	}
	return nil
}
