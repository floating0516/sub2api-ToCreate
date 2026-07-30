package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const benefitGrantCampaignSelect = `
	SELECT
		c.id,
		c.operation_key,
		c.request_hash,
		c.audience_type,
		c.audience_date::text,
		c.audience_days,
		c.timezone,
		c.window_start,
		c.window_end,
		c.benefit_type,
		c.conflict_policy,
		c.group_id,
		COALESCE(NULLIF(c.group_name_snapshot, ''), g.name, ''),
		c.validity_days,
		c.balance_amount::double precision,
		c.notes,
		c.marker,
		c.status,
		c.matched_count,
		c.eligible_count,
		c.already_granted_count,
		c.conflict_count,
		c.granted_count,
		c.skipped_count,
		c.failed_count,
		c.created_count,
		c.renewed_count,
		c.extended_count,
		c.balance_granted_count,
		c.created_by,
		c.started_at,
		c.completed_at,
		c.created_at,
		c.updated_at
	FROM benefit_grant_campaigns c
	LEFT JOIN groups g ON g.id = c.group_id
`

const benefitGrantRecipientSelect = `
	SELECT
		id,
		campaign_id,
		user_id,
		email_snapshot,
		username_snapshot,
		eligibility,
		planned_action,
		status,
		COALESCE(result_type, ''),
		subscription_id,
		balance_before::double precision,
		balance_after::double precision,
		COALESCE(error, ''),
		attempt_count,
		last_attempt_at,
		created_at,
		updated_at
	FROM benefit_grant_recipients
`

type benefitGrantRepository struct {
	db *sql.DB
}

func NewBenefitGrantRepository(db *sql.DB) service.BenefitGrantRepository {
	return &benefitGrantRepository{db: db}
}

func (r *benefitGrantRepository) ListAudience(
	ctx context.Context,
	audienceType string,
	windowStart, windowEnd time.Time,
) ([]service.BenefitGrantAudienceUser, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}

	field := "last_active_at"
	switch audienceType {
	case service.BenefitGrantAudienceTodayActive, service.BenefitGrantAudienceRecentActive:
	case service.BenefitGrantAudienceRecentRegistered:
		field = "created_at"
	default:
		return nil, fmt.Errorf("unsupported benefit grant audience: %s", audienceType)
	}

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, email, COALESCE(username, '')
		FROM users
		WHERE deleted_at IS NULL
			AND status = 'active'
			AND role = 'user'
			AND %s >= $1
			AND %s < $2
		ORDER BY id ASC
	`, field, field), windowStart, windowEnd)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	users := make([]service.BenefitGrantAudienceUser, 0)
	for rows.Next() {
		var user service.BenefitGrantAudienceUser
		if err := rows.Scan(&user.ID, &user.Email, &user.Username); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *benefitGrantRepository) ListSubscriptionStates(
	ctx context.Context,
	userIDs []int64,
	groupID int64,
) (map[int64]service.BenefitGrantSubscriptionState, error) {
	states := make(map[int64]service.BenefitGrantSubscriptionState, len(userIDs))
	if len(userIDs) == 0 {
		return states, nil
	}
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, status, expires_at, COALESCE(notes, '')
		FROM user_subscriptions
		WHERE user_id = ANY($1)
			AND group_id = $2
			AND deleted_at IS NULL
	`, pq.Array(userIDs), groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var state service.BenefitGrantSubscriptionState
		if err := rows.Scan(
			&state.ID,
			&state.UserID,
			&state.Status,
			&state.ExpiresAt,
			&state.Notes,
		); err != nil {
			return nil, err
		}
		states[state.UserID] = state
	}
	return states, rows.Err()
}

func (r *benefitGrantRepository) GetCampaignByOperationKey(
	ctx context.Context,
	createdBy int64,
	operationKey string,
) (*service.BenefitGrantCampaign, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}
	campaign, err := scanBenefitGrantCampaign(r.db.QueryRowContext(
		ctx,
		benefitGrantCampaignSelect+`
			WHERE c.created_by = $1 AND c.operation_key = $2
		`,
		createdBy,
		operationKey,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return campaign, err
}

func (r *benefitGrantRepository) CreateCampaign(
	ctx context.Context,
	campaign *service.BenefitGrantCampaign,
	recipients []service.BenefitGrantRecipient,
) (*service.BenefitGrantCampaign, bool, error) {
	if r == nil || r.db == nil || campaign == nil {
		return nil, false, service.ErrBenefitGrantUnavailable
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	err = tx.QueryRowContext(ctx, `
		INSERT INTO benefit_grant_campaigns (
			operation_key,
			request_hash,
			audience_type,
			audience_date,
			audience_days,
			timezone,
			window_start,
			window_end,
			benefit_type,
			conflict_policy,
			group_id,
			group_name_snapshot,
			validity_days,
			balance_amount,
			notes,
			marker,
			status,
			matched_count,
			eligible_count,
			already_granted_count,
			conflict_count,
			skipped_count,
			created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		ON CONFLICT (created_by, operation_key) DO NOTHING
		RETURNING id
	`,
		campaign.OperationKey,
		campaign.RequestHash,
		campaign.AudienceType,
		campaign.AudienceDate,
		campaign.AudienceDays,
		campaign.Timezone,
		campaign.WindowStart,
		campaign.WindowEnd,
		campaign.BenefitType,
		campaign.ConflictPolicy,
		nullableInt64(campaign.GroupID),
		campaign.GroupName,
		nullableInt(campaign.ValidityDays),
		nullableFloat64(campaign.BalanceAmount),
		campaign.Notes,
		campaign.Marker,
		service.BenefitGrantStatusRunning,
		campaign.MatchedCount,
		campaign.EligibleCount,
		campaign.AlreadyGrantedCount,
		campaign.ConflictCount,
		campaign.SkippedCount,
		campaign.CreatedBy,
	).Scan(&campaign.ID)
	if errors.Is(err, sql.ErrNoRows) {
		// Release the transaction before querying through the pool. Keeping the
		// transaction open can deadlock when the pool has a single connection.
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return nil, false, rollbackErr
		}
		returnExisting, getErr := r.GetCampaignByOperationKey(
			ctx,
			campaign.CreatedBy,
			campaign.OperationKey,
		)
		return returnExisting, false, getErr
	}
	if err != nil {
		return nil, false, err
	}

	if len(recipients) > 0 {
		stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
			"benefit_grant_recipients",
			"campaign_id",
			"user_id",
			"email_snapshot",
			"username_snapshot",
			"eligibility",
			"planned_action",
			"status",
			"result_type",
		))
		if err != nil {
			return nil, false, err
		}
		for i := range recipients {
			recipient := recipients[i]
			if _, err := stmt.ExecContext(
				ctx,
				campaign.ID,
				recipient.UserID,
				recipient.EmailSnapshot,
				recipient.UsernameSnapshot,
				recipient.Eligibility,
				recipient.PlannedAction,
				recipient.Status,
				nullableString(recipient.ResultType),
			); err != nil {
				_ = stmt.Close()
				return nil, false, err
			}
		}
		if _, err := stmt.ExecContext(ctx); err != nil {
			_ = stmt.Close()
			return nil, false, err
		}
		if err := stmt.Close(); err != nil {
			return nil, false, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	created, err := r.GetCampaign(ctx, campaign.ID)
	return created, true, err
}

func (r *benefitGrantRepository) GetCampaign(
	ctx context.Context,
	id int64,
) (*service.BenefitGrantCampaign, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}
	campaign, err := scanBenefitGrantCampaign(r.db.QueryRowContext(
		ctx,
		benefitGrantCampaignSelect+` WHERE c.id = $1`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrBenefitGrantNotFound
	}
	return campaign, err
}

func (r *benefitGrantRepository) ListCampaigns(
	ctx context.Context,
	page, pageSize int,
) ([]service.BenefitGrantCampaign, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, service.ErrBenefitGrantUnavailable
	}
	page, pageSize = normalizeBenefitGrantPagination(page, pageSize)

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM benefit_grant_campaigns`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(
		ctx,
		benefitGrantCampaignSelect+`
			ORDER BY c.created_at DESC, c.id DESC
			LIMIT $1 OFFSET $2
		`,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	campaigns := make([]service.BenefitGrantCampaign, 0, pageSize)
	for rows.Next() {
		campaign, err := scanBenefitGrantCampaign(rows)
		if err != nil {
			return nil, 0, err
		}
		campaigns = append(campaigns, *campaign)
	}
	return campaigns, total, rows.Err()
}

func (r *benefitGrantRepository) ListRecipients(
	ctx context.Context,
	campaignID int64,
	page, pageSize int,
	status string,
) ([]service.BenefitGrantRecipient, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, service.ErrBenefitGrantUnavailable
	}
	page, pageSize = normalizeBenefitGrantPagination(page, pageSize)

	countQuery := `SELECT COUNT(*) FROM benefit_grant_recipients WHERE campaign_id = $1`
	listQuery := benefitGrantRecipientSelect + ` WHERE campaign_id = $1`
	args := []any{campaignID}
	if status != "" {
		countQuery += ` AND status = $2`
		listQuery += ` AND status = $2`
		args = append(args, status)
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limitPosition := len(args) + 1
	offsetPosition := len(args) + 2
	listQuery += fmt.Sprintf(
		` ORDER BY id ASC LIMIT $%d OFFSET $%d`,
		limitPosition,
		offsetPosition,
	)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	recipients := make([]service.BenefitGrantRecipient, 0, pageSize)
	for rows.Next() {
		recipient, err := scanBenefitGrantRecipient(rows)
		if err != nil {
			return nil, 0, err
		}
		recipients = append(recipients, *recipient)
	}
	return recipients, total, rows.Err()
}

func (r *benefitGrantRepository) ListActionableRecipients(
	ctx context.Context,
	campaignID int64,
	includeFailed bool,
	staleBefore time.Time,
	limit int,
) ([]service.BenefitGrantRecipient, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}
	if limit <= 0 || limit > 100000 {
		limit = 100000
	}
	rows, err := r.db.QueryContext(ctx, benefitGrantRecipientSelect+`
		WHERE campaign_id = $1
			AND (
				status = 'pending'
				OR (status = 'processing' AND last_attempt_at < $2)
				OR ($3 AND status = 'failed')
			)
		ORDER BY id ASC
		LIMIT $4
	`, campaignID, staleBefore, includeFailed, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	recipients := make([]service.BenefitGrantRecipient, 0)
	for rows.Next() {
		recipient, err := scanBenefitGrantRecipient(rows)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, *recipient)
	}
	return recipients, rows.Err()
}

func (r *benefitGrantRepository) ClaimRecipient(
	ctx context.Context,
	campaignID, userID int64,
	includeFailed bool,
	staleBefore time.Time,
) (bool, error) {
	if r == nil || r.db == nil {
		return false, service.ErrBenefitGrantUnavailable
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE benefit_grant_recipients
		SET status = 'processing',
			error = NULL,
			attempt_count = attempt_count + 1,
			last_attempt_at = NOW(),
			updated_at = NOW()
		WHERE campaign_id = $1
			AND user_id = $2
			AND (
				status = 'pending'
				OR (status = 'processing' AND last_attempt_at < $3)
				OR ($4 AND status = 'failed')
			)
	`, campaignID, userID, staleBefore, includeFailed)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (r *benefitGrantRepository) MarkRecipientGranted(
	ctx context.Context,
	campaignID, userID int64,
	resultType string,
	subscriptionID *int64,
) error {
	if r == nil || r.db == nil {
		return service.ErrBenefitGrantUnavailable
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE benefit_grant_recipients
		SET status = 'granted',
			result_type = $3,
			subscription_id = $4,
			error = NULL,
			updated_at = NOW()
		WHERE campaign_id = $1
			AND user_id = $2
			AND status = 'processing'
	`, campaignID, userID, resultType, nullableInt64(subscriptionID))
	return err
}

func (r *benefitGrantRepository) MarkRecipientFailed(
	ctx context.Context,
	campaignID, userID int64,
	message string,
) error {
	if r == nil || r.db == nil {
		return service.ErrBenefitGrantUnavailable
	}
	message = truncateBenefitGrantError(message, 2000)
	_, err := r.db.ExecContext(ctx, `
		UPDATE benefit_grant_recipients
		SET status = 'failed',
			error = $3,
			updated_at = NOW()
		WHERE campaign_id = $1
			AND user_id = $2
			AND status = 'processing'
	`, campaignID, userID, message)
	return err
}

func (r *benefitGrantRepository) ApplyBalanceRecipient(
	ctx context.Context,
	campaignID, userID int64,
	amount float64,
	includeFailed bool,
	staleBefore time.Time,
) (*service.BenefitGrantBalanceApplyResult, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	claim, err := tx.ExecContext(ctx, `
		UPDATE benefit_grant_recipients
		SET status = 'processing',
			error = NULL,
			attempt_count = attempt_count + 1,
			last_attempt_at = NOW(),
			updated_at = NOW()
		WHERE campaign_id = $1
			AND user_id = $2
			AND (
				status = 'pending'
				OR (status = 'processing' AND last_attempt_at < $3)
				OR ($4 AND status = 'failed')
			)
	`, campaignID, userID, staleBefore, includeFailed)
	if err != nil {
		return nil, err
	}
	affected, err := claim.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return &service.BenefitGrantBalanceApplyResult{}, nil
	}

	result := &service.BenefitGrantBalanceApplyResult{Claimed: true}
	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1,
			updated_at = NOW()
		WHERE id = $2
			AND deleted_at IS NULL
			AND status = 'active'
			AND role = 'user'
		RETURNING (balance - $1)::double precision, balance::double precision
	`, amount, userID).Scan(&result.BalanceBefore, &result.BalanceAfter)
	if errors.Is(err, sql.ErrNoRows) {
		result.Error = "user is no longer eligible for this grant"
		if _, updateErr := tx.ExecContext(ctx, `
			UPDATE benefit_grant_recipients
			SET status = 'failed',
				error = $3,
				updated_at = NOW()
			WHERE campaign_id = $1 AND user_id = $2
		`, campaignID, userID, result.Error); updateErr != nil {
			return nil, updateErr
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	if err != nil {
		return nil, err
	}

	// Balance is part of API-key auth snapshots. Queue durable invalidation in
	// the same transaction so a successful grant cannot leave stale credentials.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO auth_cache_invalidation_outbox (cache_key)
		SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
		FROM api_keys k
		WHERE k.user_id = $1
			AND k.deleted_at IS NULL
			AND k.key <> ''
	`, userID); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE benefit_grant_recipients
		SET status = 'granted',
			result_type = 'balance_added',
			balance_before = $3,
			balance_after = $4,
			error = NULL,
			updated_at = NOW()
		WHERE campaign_id = $1 AND user_id = $2
	`, campaignID, userID, result.BalanceBefore, result.BalanceAfter); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result.Granted = true
	return result, nil
}

func (r *benefitGrantRepository) RefreshCampaign(
	ctx context.Context,
	campaignID int64,
) (*service.BenefitGrantCampaign, error) {
	if r == nil || r.db == nil {
		return nil, service.ErrBenefitGrantUnavailable
	}
	result, err := r.db.ExecContext(ctx, `
		WITH counts AS (
			SELECT
				COUNT(*)::integer AS matched_count,
				COUNT(*) FILTER (WHERE eligibility = 'eligible')::integer AS eligible_count,
				COUNT(*) FILTER (WHERE eligibility = 'already_granted')::integer AS already_granted_count,
				COUNT(*) FILTER (WHERE eligibility = 'conflict')::integer AS conflict_count,
				COUNT(*) FILTER (WHERE status = 'granted')::integer AS granted_count,
				COUNT(*) FILTER (WHERE status = 'skipped')::integer AS skipped_count,
				COUNT(*) FILTER (WHERE status = 'failed')::integer AS failed_count,
				COUNT(*) FILTER (WHERE status IN ('pending', 'processing'))::integer AS pending_count,
				COUNT(*) FILTER (WHERE result_type = 'created')::integer AS created_count,
				COUNT(*) FILTER (WHERE result_type = 'renewed')::integer AS renewed_count,
				COUNT(*) FILTER (WHERE result_type = 'extended')::integer AS extended_count,
				COUNT(*) FILTER (WHERE result_type = 'balance_added')::integer AS balance_granted_count
			FROM benefit_grant_recipients
			WHERE campaign_id = $1
		)
		UPDATE benefit_grant_campaigns c
		SET matched_count = counts.matched_count,
			eligible_count = counts.eligible_count,
			already_granted_count = counts.already_granted_count,
			conflict_count = counts.conflict_count,
			granted_count = counts.granted_count,
			skipped_count = counts.skipped_count,
			failed_count = counts.failed_count,
			created_count = counts.created_count,
			renewed_count = counts.renewed_count,
			extended_count = counts.extended_count,
			balance_granted_count = counts.balance_granted_count,
			status = CASE
				WHEN counts.pending_count > 0 THEN 'running'
				WHEN counts.failed_count > 0 AND counts.granted_count > 0 THEN 'partial'
				WHEN counts.failed_count > 0 THEN 'failed'
				ELSE 'completed'
			END,
			completed_at = CASE
				WHEN counts.pending_count = 0 THEN COALESCE(c.completed_at, NOW())
				ELSE NULL
			END,
			updated_at = NOW()
		FROM counts
		WHERE c.id = $1
	`, campaignID)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrBenefitGrantNotFound
	}
	return r.GetCampaign(ctx, campaignID)
}

type benefitGrantScanner interface {
	Scan(dest ...any) error
}

func scanBenefitGrantCampaign(scanner benefitGrantScanner) (*service.BenefitGrantCampaign, error) {
	var (
		campaign      service.BenefitGrantCampaign
		groupID       sql.NullInt64
		validityDays  sql.NullInt64
		balanceAmount sql.NullFloat64
		completedAt   sql.NullTime
	)
	err := scanner.Scan(
		&campaign.ID,
		&campaign.OperationKey,
		&campaign.RequestHash,
		&campaign.AudienceType,
		&campaign.AudienceDate,
		&campaign.AudienceDays,
		&campaign.Timezone,
		&campaign.WindowStart,
		&campaign.WindowEnd,
		&campaign.BenefitType,
		&campaign.ConflictPolicy,
		&groupID,
		&campaign.GroupName,
		&validityDays,
		&balanceAmount,
		&campaign.Notes,
		&campaign.Marker,
		&campaign.Status,
		&campaign.MatchedCount,
		&campaign.EligibleCount,
		&campaign.AlreadyGrantedCount,
		&campaign.ConflictCount,
		&campaign.GrantedCount,
		&campaign.SkippedCount,
		&campaign.FailedCount,
		&campaign.CreatedCount,
		&campaign.RenewedCount,
		&campaign.ExtendedCount,
		&campaign.BalanceGrantedCount,
		&campaign.CreatedBy,
		&campaign.StartedAt,
		&completedAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if groupID.Valid {
		value := groupID.Int64
		campaign.GroupID = &value
	}
	if validityDays.Valid {
		value := int(validityDays.Int64)
		campaign.ValidityDays = &value
	}
	if balanceAmount.Valid {
		value := balanceAmount.Float64
		campaign.BalanceAmount = &value
	}
	if completedAt.Valid {
		value := completedAt.Time
		campaign.CompletedAt = &value
	}
	return &campaign, nil
}

func scanBenefitGrantRecipient(scanner benefitGrantScanner) (*service.BenefitGrantRecipient, error) {
	var (
		recipient      service.BenefitGrantRecipient
		subscriptionID sql.NullInt64
		balanceBefore  sql.NullFloat64
		balanceAfter   sql.NullFloat64
		lastAttemptAt  sql.NullTime
	)
	err := scanner.Scan(
		&recipient.ID,
		&recipient.CampaignID,
		&recipient.UserID,
		&recipient.EmailSnapshot,
		&recipient.UsernameSnapshot,
		&recipient.Eligibility,
		&recipient.PlannedAction,
		&recipient.Status,
		&recipient.ResultType,
		&subscriptionID,
		&balanceBefore,
		&balanceAfter,
		&recipient.Error,
		&recipient.AttemptCount,
		&lastAttemptAt,
		&recipient.CreatedAt,
		&recipient.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if subscriptionID.Valid {
		value := subscriptionID.Int64
		recipient.SubscriptionID = &value
	}
	if balanceBefore.Valid {
		value := balanceBefore.Float64
		recipient.BalanceBefore = &value
	}
	if balanceAfter.Valid {
		value := balanceAfter.Float64
		recipient.BalanceAfter = &value
	}
	if lastAttemptAt.Valid {
		value := lastAttemptAt.Time
		recipient.LastAttemptAt = &value
	}
	return &recipient, nil
}

func normalizeBenefitGrantPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat64(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func truncateBenefitGrantError(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
