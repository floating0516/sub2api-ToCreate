package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
			user_id, api_key_id, code_hash, client_id, redirect_uri, scopes,
			code_challenge, code_challenge_method, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	return scanSingleRow(
		ctx,
		r.sql,
		query,
		[]any{
			code.UserID,
			code.APIKeyID,
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
	var apiKeyID sql.NullInt64
	var usedAt sql.NullTime
	err := scanSingleRow(ctx, r.sql, `
		SELECT id, user_id, api_key_id, code_hash, client_id, redirect_uri, scopes,
			code_challenge, code_challenge_method, expires_at, used, used_at, created_at
		FROM lihe_oauth_authorization_codes
		WHERE code_hash = $1
	`, []any{codeHash},
		&record.ID,
		&record.UserID,
		&apiKeyID,
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
	if apiKeyID.Valid {
		record.APIKeyID = apiKeyID.Int64
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
	if len(input.Bindings) != 1 {
		return nil, service.ErrLiheInvalidGrant
	}
	binding := input.Bindings[0]
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
	var codeAPIKeyID sql.NullInt64
	var clientID string
	var redirectURI string
	var codeChallenge string
	var expiresAt time.Time
	var used bool
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, api_key_id, client_id, redirect_uri, code_challenge, expires_at, used
		FROM lihe_oauth_authorization_codes
		WHERE code_hash = $1
		FOR UPDATE
	`, input.CodeHash).Scan(
		&codeID,
		&codeUserID,
		&codeAPIKeyID,
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
		!codeAPIKeyID.Valid || codeAPIKeyID.Int64 != binding.APIKeyID ||
		clientID != input.ClientID || redirectURI != input.RedirectURI || codeChallenge != input.CodeChallenge {
		return nil, service.ErrLiheInvalidGrant
	}

	var apiKeyName string
	var apiKeyStatus string
	var groupID int64
	var groupStatus string
	var provider string
	err = tx.QueryRowContext(ctx, `
		SELECT k.name, k.status, k.group_id, g.status, g.platform
		FROM api_keys k
		JOIN groups g ON g.id = k.group_id
		WHERE k.id = $1 AND k.user_id = $2 AND k.deleted_at IS NULL
		FOR SHARE OF k, g
	`, binding.APIKeyID, input.UserID).Scan(
		&apiKeyName,
		&apiKeyStatus,
		&groupID,
		&groupStatus,
		&provider,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrLiheInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if apiKeyStatus != service.StatusAPIKeyActive || groupStatus != service.StatusActive ||
		groupID != binding.GroupID || provider != binding.Provider {
		return nil, service.ErrLiheInvalidGrant
	}

	result := &service.LiheAccessToken{
		UserID:     input.UserID,
		Name:       service.LiheOAuthClientName,
		ClientID:   input.ClientID,
		Scopes:     append([]string(nil), input.Scopes...),
		Providers:  []string{provider},
		APIKeyID:   &binding.APIKeyID,
		APIKeyName: apiKeyName,
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

	// New bindings use source_api_key_id. The legacy api_key_id stays NULL so
	// an older application rollback cannot disable the user's source key.
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO lihe_oauth_token_bindings (token_id, provider, group_id, source_api_key_id)
		VALUES ($1, $2, $3, $4)
	`, result.ID, provider, binding.GroupID, binding.APIKeyID); err != nil {
		return nil, err
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
			) AS providers,
			MIN(COALESCE(b.source_api_key_id, b.api_key_id)) AS api_key_id,
			COALESCE(MIN(k.name), '') AS api_key_name
		FROM lihe_oauth_access_tokens t
		LEFT JOIN lihe_oauth_token_bindings b ON b.token_id = t.id
		LEFT JOIN api_keys k ON k.id = COALESCE(b.source_api_key_id, b.api_key_id)
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
		var apiKeyID sql.NullInt64
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
			&apiKeyID,
			&token.APIKeyName,
		); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			token.LastUsedAt = &lastUsedAt.Time
		}
		if revokedAt.Valid {
			token.RevokedAt = &revokedAt.Time
		}
		if apiKeyID.Valid {
			token.APIKeyID = &apiKeyID.Int64
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
				AND k.key LIKE $%d
			RETURNING k.id
		)
		SELECT EXISTS(SELECT 1 FROM target)
	`, predicate, len(args)+1, len(args)+2)
	queryArgs := append(
		append([]any(nil), args...),
		service.StatusAPIKeyDisabled,
		service.LiheInternalAPIKeyPrefix+"%",
	)
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
	var bindingID sql.NullInt64
	var bindingGroupID sql.NullInt64
	var apiKeyID sql.NullInt64
	err := scanSingleRow(ctx, r.sql, `
		WITH resolved AS (
			SELECT t.id AS token_id, t.user_id, t.scopes,
				b.id AS binding_id, b.group_id AS binding_group_id, k.id AS api_key_id
			FROM lihe_oauth_access_tokens t
			LEFT JOIN lihe_oauth_token_bindings b
				ON b.token_id = t.id AND b.provider = $3
			LEFT JOIN groups g
				ON g.id = b.group_id AND LOWER(TRIM(g.platform)) = b.provider
			LEFT JOIN api_keys k
				ON k.id = COALESCE(b.source_api_key_id, b.api_key_id)
				AND k.user_id = t.user_id
				AND k.group_id = b.group_id
				AND k.deleted_at IS NULL
				AND g.id IS NOT NULL
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
		SELECT token_id, user_id, scopes, binding_id, binding_group_id, api_key_id
		FROM resolved
	`, []any{tokenHash, clientID, provider},
		&resolved.TokenID,
		&resolved.TokenUserID,
		pq.Array(&resolved.Scopes),
		&bindingID,
		&bindingGroupID,
		&apiKeyID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if bindingID.Valid && bindingGroupID.Valid {
		resolved.BindingFound = true
		resolved.BindingGroupID = bindingGroupID.Int64
	}
	if !apiKeyID.Valid {
		return resolved, nil
	}
	apiKey, err := r.getLiheBoundAPIKeyForAuth(ctx, apiKeyID.Int64)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			return resolved, nil
		}
		return nil, err
	}
	resolved.APIKey = apiKey
	return resolved, nil
}

func (r *apiKeyRepository) getLiheBoundAPIKeyForAuth(ctx context.Context, id int64) (*service.APIKey, error) {
	m, err := r.client.APIKey.Query().
		Where(
			apikey.IDEQ(id),
			apikey.DeletedAtIsNil(),
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
