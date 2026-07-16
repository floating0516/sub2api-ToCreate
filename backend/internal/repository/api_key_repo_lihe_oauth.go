package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type sqlTxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (r *apiKeyRepository) CreateLiheAuthorizationCode(ctx context.Context, code *service.LiheAuthorizationCode) error {
	if code == nil || r.sql == nil {
		return service.ErrLiheOAuthRepositoryUnavailable
	}
	_, _ = r.sql.ExecContext(ctx, `
		DELETE FROM lihe_oauth_authorization_codes
		WHERE expires_at < NOW() - INTERVAL '1 day'
	`)
	query := `
		INSERT INTO lihe_oauth_authorization_codes (
			user_id, code_hash, client_id, redirect_uri, scopes,
			code_challenge, code_challenge_method, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return scanSingleRow(
		ctx,
		r.sql,
		query,
		[]any{
			code.UserID,
			code.CodeHash,
			code.ClientID,
			code.RedirectURI,
			pq.Array(code.Scopes),
			code.CodeChallenge,
			code.CodeChallengeMethod,
			code.ExpiresAt,
		},
		&code.ID,
		&code.CreatedAt,
	)
}

func (r *apiKeyRepository) GetLiheAuthorizationCode(ctx context.Context, codeHash string) (*service.LiheAuthorizationCode, error) {
	if r.sql == nil {
		return nil, service.ErrLiheOAuthRepositoryUnavailable
	}
	record := &service.LiheAuthorizationCode{}
	var usedAt sql.NullTime
	err := scanSingleRow(ctx, r.sql, `
		SELECT id, user_id, code_hash, client_id, redirect_uri, scopes,
			code_challenge, code_challenge_method, expires_at, used, used_at, created_at
		FROM lihe_oauth_authorization_codes
		WHERE code_hash = $1
	`, []any{codeHash},
		&record.ID,
		&record.UserID,
		&record.CodeHash,
		&record.ClientID,
		&record.RedirectURI,
		pq.Array(&record.Scopes),
		&record.CodeChallenge,
		&record.CodeChallengeMethod,
		&record.ExpiresAt,
		&record.Used,
		&usedAt,
		&record.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		record.UsedAt = &usedAt.Time
	}
	return record, nil
}

func (r *apiKeyRepository) ExchangeLiheAuthorizationCode(
	ctx context.Context,
	input service.LiheTokenExchangeInput,
) (*service.LiheAccessToken, error) {
	beginner, ok := r.sql.(sqlTxBeginner)
	if !ok || beginner == nil {
		return nil, service.ErrLiheOAuthRepositoryUnavailable
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var codeID int64
	var codeUserID int64
	var clientID string
	var redirectURI string
	var codeChallenge string
	var expiresAt time.Time
	var used bool
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, client_id, redirect_uri, code_challenge, expires_at, used
		FROM lihe_oauth_authorization_codes
		WHERE code_hash = $1
		FOR UPDATE
	`, input.CodeHash).Scan(
		&codeID,
		&codeUserID,
		&clientID,
		&redirectURI,
		&codeChallenge,
		&expiresAt,
		&used,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrLiheInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	if used || !expiresAt.After(time.Now()) || codeUserID != input.UserID ||
		clientID != input.ClientID || redirectURI != input.RedirectURI || codeChallenge != input.CodeChallenge {
		return nil, service.ErrLiheInvalidGrant
	}

	result := &service.LiheAccessToken{
		UserID:    input.UserID,
		Name:      service.LiheOAuthClientName,
		ClientID:  input.ClientID,
		Scopes:    append([]string(nil), input.Scopes...),
		Providers: make([]string, 0, len(input.Bindings)),
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO lihe_oauth_access_tokens (
			user_id, token_hash, name, client_id, scopes
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, input.UserID, input.TokenHash, service.LiheOAuthClientName, input.ClientID, pq.Array(input.Scopes)).Scan(
		&result.ID,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	for _, binding := range input.Bindings {
		var apiKeyID int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO api_keys (
				user_id, key, name, group_id, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			RETURNING id
		`,
			input.UserID,
			binding.APIKey,
			fmt.Sprintf("%s / %s", service.LiheOAuthClientName, binding.Provider),
			binding.GroupID,
			service.StatusAPIKeyActive,
		).Scan(&apiKeyID)
		if err != nil {
			return nil, err
		}
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO lihe_oauth_token_bindings (token_id, provider, group_id, api_key_id)
			VALUES ($1, $2, $3, $4)
		`, result.ID, binding.Provider, binding.GroupID, apiKeyID); err != nil {
			return nil, err
		}
		result.Providers = append(result.Providers, binding.Provider)
	}

	updated, err := tx.ExecContext(ctx, `
		UPDATE lihe_oauth_authorization_codes
		SET used = TRUE, used_at = NOW()
		WHERE id = $1 AND used = FALSE
	`, codeID)
	if err != nil {
		return nil, err
	}
	affected, err := updated.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, service.ErrLiheInvalidGrant
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *apiKeyRepository) ListLiheAccessTokens(ctx context.Context, userID int64) ([]service.LiheAccessToken, error) {
	if r.sql == nil {
		return nil, service.ErrLiheOAuthRepositoryUnavailable
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT t.id, t.user_id, t.name, t.client_id, t.scopes, t.last_used_at,
			t.revoked_at, t.created_at,
			COALESCE(
				ARRAY_AGG(b.provider ORDER BY b.provider) FILTER (WHERE b.provider IS NOT NULL),
				ARRAY[]::TEXT[]
			) AS providers
		FROM lihe_oauth_access_tokens t
		LEFT JOIN lihe_oauth_token_bindings b ON b.token_id = t.id
		WHERE t.user_id = $1 AND t.revoked_at IS NULL
		GROUP BY t.id
		ORDER BY t.created_at DESC, t.id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	tokens := make([]service.LiheAccessToken, 0)
	for rows.Next() {
		var token service.LiheAccessToken
		var lastUsedAt sql.NullTime
		var revokedAt sql.NullTime
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Name,
			&token.ClientID,
			pq.Array(&token.Scopes),
			&lastUsedAt,
			&revokedAt,
			&token.CreatedAt,
			pq.Array(&token.Providers),
		); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			token.LastUsedAt = &lastUsedAt.Time
		}
		if revokedAt.Valid {
			token.RevokedAt = &revokedAt.Time
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *apiKeyRepository) RevokeLiheAccessTokenByID(ctx context.Context, tokenID, userID int64) (bool, error) {
	return r.revokeLiheAccessToken(ctx, `t.id = $1 AND t.user_id = $2`, tokenID, userID)
}

func (r *apiKeyRepository) RevokeLiheAccessTokenByIDAsAdmin(ctx context.Context, tokenID int64) (bool, error) {
	return r.revokeLiheAccessToken(ctx, `t.id = $1`, tokenID)
}

func (r *apiKeyRepository) RevokeLiheAccessTokenByHash(ctx context.Context, tokenHash, clientID string) (bool, error) {
	return r.revokeLiheAccessToken(ctx, `t.token_hash = $1 AND t.client_id = $2`, tokenHash, clientID)
}

func (r *apiKeyRepository) revokeLiheAccessToken(ctx context.Context, predicate string, args ...any) (bool, error) {
	if r.sql == nil {
		return false, service.ErrLiheOAuthRepositoryUnavailable
	}
	query := fmt.Sprintf(`
		WITH target AS (
			SELECT t.id
			FROM lihe_oauth_access_tokens t
			WHERE %s
		), revoked AS (
			UPDATE lihe_oauth_access_tokens t
			SET revoked_at = COALESCE(t.revoked_at, NOW())
			WHERE t.id IN (SELECT id FROM target)
			RETURNING t.id
		), disabled AS (
			UPDATE api_keys k
			SET status = $%d, updated_at = NOW()
			WHERE k.id IN (
				SELECT b.api_key_id
				FROM lihe_oauth_token_bindings b
				JOIN target ON target.id = b.token_id
			)
			RETURNING k.id
		)
		SELECT EXISTS(SELECT 1 FROM target)
	`, predicate, len(args)+1)
	queryArgs := append(append([]any(nil), args...), service.StatusAPIKeyDisabled)
	var found bool
	if err := scanSingleRow(ctx, r.sql, query, queryArgs, &found); err != nil {
		return false, err
	}
	return found, nil
}

func (r *apiKeyRepository) ResolveLiheAccessToken(
	ctx context.Context,
	tokenHash, clientID, provider string,
) (*service.LiheResolvedAccess, error) {
	if r.sql == nil {
		return nil, service.ErrLiheOAuthRepositoryUnavailable
	}
	resolved := &service.LiheResolvedAccess{}
	var apiKeyID sql.NullInt64
	err := scanSingleRow(ctx, r.sql, `
		WITH resolved AS (
			SELECT t.id AS token_id, t.user_id, t.scopes, k.id AS api_key_id
			FROM lihe_oauth_access_tokens t
			LEFT JOIN lihe_oauth_token_bindings b
				ON b.token_id = t.id AND b.provider = $3
			LEFT JOIN api_keys k
				ON k.id = b.api_key_id AND k.user_id = t.user_id AND k.deleted_at IS NULL
			WHERE t.token_hash = $1
				AND t.client_id = $2
				AND t.revoked_at IS NULL
		), touched AS (
			UPDATE lihe_oauth_access_tokens t
			SET last_used_at = NOW()
			WHERE t.id = (SELECT token_id FROM resolved)
				AND (t.last_used_at IS NULL OR t.last_used_at < NOW() - INTERVAL '30 seconds')
			RETURNING t.id
		)
		SELECT token_id, user_id, scopes, api_key_id
		FROM resolved
	`, []any{tokenHash, clientID, provider},
		&resolved.TokenID,
		&resolved.TokenUserID,
		pq.Array(&resolved.Scopes),
		&apiKeyID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !apiKeyID.Valid {
		return resolved, nil
	}
	apiKey, err := r.getLiheInternalAPIKeyForAuth(ctx, apiKeyID.Int64)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			return resolved, nil
		}
		return nil, err
	}
	resolved.APIKey = apiKey
	return resolved, nil
}

func (r *apiKeyRepository) getLiheInternalAPIKeyForAuth(ctx context.Context, id int64) (*service.APIKey, error) {
	m, err := r.client.APIKey.Query().
		Where(
			apikey.IDEQ(id),
			apikey.DeletedAtIsNil(),
			apikey.KeyHasPrefix(service.LiheInternalAPIKeyPrefix),
		).
		Select(
			apikey.FieldID,
			apikey.FieldUserID,
			apikey.FieldKey,
			apikey.FieldGroupID,
			apikey.FieldName,
			apikey.FieldStatus,
			apikey.FieldIPWhitelist,
			apikey.FieldIPBlacklist,
			apikey.FieldQuota,
			apikey.FieldQuotaUsed,
			apikey.FieldExpiresAt,
			apikey.FieldRateLimit5h,
			apikey.FieldRateLimit1d,
			apikey.FieldRateLimit7d,
		).
		WithUser(func(q *dbent.UserQuery) {
			q.Select(
				user.FieldID,
				user.FieldEmail,
				user.FieldUsername,
				user.FieldStatus,
				user.FieldRole,
				user.FieldBalance,
				user.FieldConcurrency,
				user.FieldBalanceNotifyEnabled,
				user.FieldBalanceNotifyThresholdType,
				user.FieldBalanceNotifyThreshold,
				user.FieldBalanceNotifyExtraEmails,
				user.FieldTotalRecharged,
				user.FieldSignupSource,
				user.FieldLastLoginAt,
				user.FieldLastActiveAt,
				user.FieldRpmLimit,
			)
			q.WithAllowedGroups(func(gq *dbent.GroupQuery) {
				gq.Select(group.FieldID)
			})
		}).
		WithGroup(func(q *dbent.GroupQuery) {
			q.Select(
				group.FieldID,
				group.FieldName,
				group.FieldPlatform,
				group.FieldIsExclusive,
				group.FieldStatus,
				group.FieldSubscriptionType,
				group.FieldRateMultiplier,
				group.FieldDailyLimitUsd,
				group.FieldWeeklyLimitUsd,
				group.FieldMonthlyLimitUsd,
				group.FieldAllowImageGeneration,
				group.FieldAllowBatchImageGeneration,
				group.FieldImageRateIndependent,
				group.FieldImageRateMultiplier,
				group.FieldImagePrice1k,
				group.FieldImagePrice2k,
				group.FieldImagePrice4k,
				group.FieldVideoRateIndependent,
				group.FieldVideoRateMultiplier,
				group.FieldVideoPrice480p,
				group.FieldVideoPrice720p,
				group.FieldVideoPrice1080p,
				group.FieldWebSearchPricePerCall,
				group.FieldClaudeCodeOnly,
				group.FieldFallbackGroupID,
				group.FieldFallbackGroupIDOnInvalidRequest,
				group.FieldModelRoutingEnabled,
				group.FieldModelRouting,
				group.FieldMcpXMLInject,
				group.FieldSupportedModelScopes,
				group.FieldAllowMessagesDispatch,
				group.FieldDefaultMappedModel,
				group.FieldMessagesDispatchModelConfig,
				group.FieldModelsListConfig,
				group.FieldRpmLimit,
				group.FieldPeakRateEnabled,
				group.FieldPeakStart,
				group.FieldPeakEnd,
				group.FieldPeakRateMultiplier,
			)
		}).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return apiKeyEntityToService(m), nil
}
