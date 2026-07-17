package oidcprovider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/handler/pkce"
	fositestorage "github.com/ory/fosite/storage"
)

var (
	ErrPendingRequestNotFound = errors.New("OIDC pending authorization request is invalid or expired")
	ErrUserInactive           = errors.New("OIDC user is inactive")
)

type UserProfile struct {
	UserID            int64
	Subject           string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Name              string
}

type Store struct {
	db         *sql.DB
	client     fosite.Client
	pendingTTL time.Duration
	payloads   *payloadCipher
}

func NewStore(db *sql.DB, client fosite.Client, pendingTTL time.Duration, storageSecret string) (*Store, error) {
	if db == nil {
		return nil, errors.New("OIDC database is required")
	}
	if client == nil {
		return nil, errors.New("OIDC client is required")
	}
	if pendingTTL <= 0 {
		return nil, errors.New("OIDC pending request TTL must be positive")
	}
	payloads, err := newPayloadCipher(storageSecret)
	if err != nil {
		return nil, err
	}
	return &Store{db: db, client: client, pendingTTL: pendingTTL, payloads: payloads}, nil
}

func (s *Store) GetClient(_ context.Context, id string) (fosite.Client, error) {
	if s == nil || s.client == nil || id != s.client.GetID() {
		return nil, fosite.ErrNotFound
	}
	return s.client, nil
}

func (s *Store) ClientAssertionJWTValid(context.Context, string) error {
	return fosite.ErrNotFound
}

func (s *Store) SetClientAssertionJWT(context.Context, string, time.Time) error {
	return fosite.ErrInvalidClient
}

type oidcTxContextKey struct{}

type oidcTxState struct {
	tx   *sql.Tx
	done bool
}

func (s *Store) BeginTX(ctx context.Context) (context.Context, error) {
	if existing, ok := ctx.Value(oidcTxContextKey{}).(*oidcTxState); ok && existing != nil && !existing.done {
		return ctx, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, oidcTxContextKey{}, &oidcTxState{tx: tx}), nil
}

func (s *Store) Commit(ctx context.Context) error {
	state, ok := ctx.Value(oidcTxContextKey{}).(*oidcTxState)
	if !ok || state == nil || state.tx == nil || state.done {
		return errors.New("OIDC transaction is unavailable")
	}
	state.done = true
	return state.tx.Commit()
}

func (s *Store) Rollback(ctx context.Context) error {
	state, ok := ctx.Value(oidcTxContextKey{}).(*oidcTxState)
	if !ok || state == nil || state.tx == nil || state.done {
		return nil
	}
	state.done = true
	return state.tx.Rollback()
}

type oidcSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) executor(ctx context.Context) oidcSQLExecutor {
	if state, ok := ctx.Value(oidcTxContextKey{}).(*oidcTxState); ok && state != nil && state.tx != nil && !state.done {
		return state.tx
	}
	return s.db
}

func (s *Store) CreatePendingRequest(ctx context.Context, params url.Values, browserBinding string) (string, time.Time, error) {
	handle, err := randomOpaqueValue()
	if err != nil {
		return "", time.Time{}, err
	}
	payload, err := s.payloads.Seal(params)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encrypt OIDC pending request: %w", err)
	}
	expiresAt := time.Now().UTC().Add(s.pendingTTL)
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM lihe_oidc_pending_requests
		WHERE expires_at < NOW() - INTERVAL '1 day'
	`)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO lihe_oidc_pending_requests (
			request_hash, browser_binding_hash, request_params, expires_at
		) VALUES ($1, $2, $3::jsonb, $4)
	`, hashOIDCCredential(handle), hashOIDCCredential(browserBinding), payload, expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("store OIDC pending request: %w", err)
	}
	return handle, expiresAt, nil
}

func (s *Store) LoadPendingRequestForUpdate(ctx context.Context, handle, browserBinding string) (url.Values, time.Time, error) {
	var payload []byte
	var requestedAt time.Time
	err := s.executor(ctx).QueryRowContext(ctx, `
		SELECT request_params, created_at
		FROM lihe_oidc_pending_requests
		WHERE request_hash = $1
		  AND browser_binding_hash = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		FOR UPDATE
	`, hashOIDCCredential(handle), hashOIDCCredential(browserBinding)).Scan(&payload, &requestedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, ErrPendingRequestNotFound
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	params := url.Values{}
	if err := s.payloads.Open(payload, &params); err != nil {
		return nil, time.Time{}, fmt.Errorf("decode OIDC pending request: %w", err)
	}
	return params, requestedAt.UTC(), nil
}

func (s *Store) ConsumePendingRequest(ctx context.Context, handle, browserBinding string) error {
	result, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_pending_requests
		SET consumed_at = NOW()
		WHERE request_hash = $1
		  AND browser_binding_hash = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
	`, hashOIDCCredential(handle), hashOIDCCredential(browserBinding))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrPendingRequestNotFound
	}
	return nil
}

func (s *Store) GetActiveProfileByUserID(ctx context.Context, userID int64) (*UserProfile, error) {
	return s.getActiveProfile(ctx, "u.id = $1", userID)
}

func (s *Store) GetActiveProfileBySubject(ctx context.Context, subject string) (*UserProfile, error) {
	return s.getActiveProfile(ctx, "u.oidc_subject = $1::uuid", subject)
}

func (s *Store) getActiveProfile(ctx context.Context, predicate string, arg any) (*UserProfile, error) {
	return scanActiveProfile(ctx, s.executor(ctx), predicate, arg)
}

func scanActiveProfile(ctx context.Context, exec oidcSQLExecutor, predicate string, arg any) (*UserProfile, error) {
	profile := &UserProfile{}
	var username string
	var verifiedAt sql.NullTime
	query := fmt.Sprintf(`
		SELECT u.id, u.oidc_subject::text, u.email, u.username, u.email_verified_at
		FROM users u
		WHERE %s
		  AND u.deleted_at IS NULL
		  AND u.status = 'active'
	`, predicate)
	err := exec.QueryRowContext(ctx, query, arg).Scan(
		&profile.UserID,
		&profile.Subject,
		&profile.Email,
		&username,
		&verifiedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserInactive
	}
	if err != nil {
		return nil, err
	}
	profile.EmailVerified = verifiedAt.Valid
	profile.PreferredUsername = strings.TrimSpace(username)
	if profile.PreferredUsername == "" {
		profile.PreferredUsername = emailLocalPart(profile.Email)
	}
	profile.Name = profile.PreferredUsername
	return profile, nil
}

type persistedRequest struct {
	ID                string                 `json:"id"`
	RequestedAt       time.Time              `json:"requested_at"`
	ClientID          string                 `json:"client_id"`
	RequestedScope    []string               `json:"requested_scope"`
	GrantedScope      []string               `json:"granted_scope"`
	Form              url.Values             `json:"form"`
	Session           *openid.DefaultSession `json:"session"`
	RequestedAudience []string               `json:"requested_audience"`
	GrantedAudience   []string               `json:"granted_audience"`
}

func (s *Store) marshalRequester(requester fosite.Requester) (string, error) {
	if requester == nil || requester.GetClient() == nil {
		return "", errors.New("OIDC requester is incomplete")
	}
	session, ok := requester.GetSession().(*openid.DefaultSession)
	if !ok || session == nil {
		return "", errors.New("OIDC requester has an unsupported session")
	}
	persisted := persistedRequest{
		ID:                requester.GetID(),
		RequestedAt:       requester.GetRequestedAt(),
		ClientID:          requester.GetClient().GetID(),
		RequestedScope:    append([]string(nil), requester.GetRequestedScopes()...),
		GrantedScope:      append([]string(nil), requester.GetGrantedScopes()...),
		Form:              cloneURLValues(requester.GetRequestForm()),
		Session:           session,
		RequestedAudience: append([]string(nil), requester.GetRequestedAudience()...),
		GrantedAudience:   append([]string(nil), requester.GetGrantedAudience()...),
	}
	return s.payloads.Seal(&persisted)
}

func (s *Store) unmarshalRequester(ctx context.Context, payload []byte) (fosite.Requester, error) {
	var persisted persistedRequest
	if err := s.payloads.Open(payload, &persisted); err != nil {
		return nil, err
	}
	client, err := s.GetClient(ctx, persisted.ClientID)
	if err != nil {
		return nil, err
	}
	if persisted.Session == nil {
		return nil, errors.New("OIDC persisted session is missing")
	}
	return &fosite.Request{
		ID:                persisted.ID,
		RequestedAt:       persisted.RequestedAt,
		Client:            client,
		RequestedScope:    fosite.Arguments(persisted.RequestedScope),
		GrantedScope:      fosite.Arguments(persisted.GrantedScope),
		Form:              cloneURLValues(persisted.Form),
		Session:           persisted.Session,
		RequestedAudience: fosite.Arguments(persisted.RequestedAudience),
		GrantedAudience:   fosite.Arguments(persisted.GrantedAudience),
	}, nil
}

func (s *Store) CreateAuthorizeCodeSession(ctx context.Context, signature string, requester fosite.Requester) error {
	payload, err := s.marshalRequester(requester)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(requester.GetID()); err != nil {
		return fmt.Errorf("invalid OIDC request ID: %w", err)
	}
	profile, err := s.getActiveProfile(ctx, "u.oidc_subject = $1::uuid", requester.GetSession().GetSubject())
	if err != nil {
		return err
	}
	form := requester.GetRequestForm()
	expiresAt := requester.GetSession().GetExpiresAt(fosite.AuthorizeCode)
	if expiresAt.IsZero() {
		return errors.New("OIDC authorization code expiry is missing")
	}
	_, _ = s.executor(ctx).ExecContext(ctx, `
		DELETE FROM lihe_oidc_authorization_codes
		WHERE expires_at < NOW() - INTERVAL '1 day'
	`)
	_, err = s.executor(ctx).ExecContext(ctx, `
		INSERT INTO lihe_oidc_authorization_codes (
			request_id, user_id, signature_hash, client_id, redirect_uri, scopes,
			code_challenge_hash, code_challenge_method, nonce_hash, request_data, expires_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)
	`,
		requester.GetID(),
		profile.UserID,
		hashOIDCCredential(signature),
		requester.GetClient().GetID(),
		form.Get("redirect_uri"),
		pq.Array([]string(requester.GetGrantedScopes())),
		hashOIDCCredential(form.Get("code_challenge")),
		form.Get("code_challenge_method"),
		hashOIDCCredential(form.Get("nonce")),
		payload,
		expiresAt,
	)
	return err
}

func (s *Store) GetAuthorizeCodeSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	var payload []byte
	var invalidatedAt sql.NullTime
	err := s.executor(ctx).QueryRowContext(ctx, `
		SELECT request_data, invalidated_at
		FROM lihe_oidc_authorization_codes
		WHERE signature_hash = $1
		  AND expires_at > NOW()
	`, hashOIDCCredential(signature)).Scan(&payload, &invalidatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	requester, err := s.unmarshalRequester(ctx, payload)
	if err != nil {
		return nil, err
	}
	if invalidatedAt.Valid {
		return requester, fosite.ErrInvalidatedAuthorizeCode
	}
	return requester, nil
}

func (s *Store) InvalidateAuthorizeCodeSession(ctx context.Context, signature string) error {
	result, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_authorization_codes
		SET invalidated_at = NOW(), pkce_request_data = NULL
		WHERE signature_hash = $1 AND invalidated_at IS NULL
	`, hashOIDCCredential(signature))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fosite.ErrNotFound
	}
	return nil
}

func (s *Store) CreateOpenIDConnectSession(ctx context.Context, code string, requester fosite.Requester) error {
	payload, err := s.marshalRequester(requester)
	if err != nil {
		return err
	}
	result, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_authorization_codes
		SET code_hash = $1, oidc_request_data = $2::jsonb
		WHERE request_id = $3::uuid AND invalidated_at IS NULL
	`, hashOIDCCredential(code), payload, requester.GetID())
	if err != nil {
		return err
	}
	return requireOneAffected(result)
}

func (s *Store) GetOpenIDConnectSession(ctx context.Context, code string, _ fosite.Requester) (fosite.Requester, error) {
	var payload []byte
	err := s.executor(ctx).QueryRowContext(ctx, `
		SELECT oidc_request_data
		FROM lihe_oidc_authorization_codes
		WHERE code_hash = $1
		  AND oidc_request_data IS NOT NULL
		  AND expires_at > NOW()
	`, hashOIDCCredential(code)).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.unmarshalRequester(ctx, payload)
}

func (s *Store) DeleteOpenIDConnectSession(ctx context.Context, code string) error {
	_, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_authorization_codes
		SET oidc_request_data = NULL
		WHERE code_hash = $1
	`, hashOIDCCredential(code))
	return err
}

func (s *Store) CreatePKCERequestSession(ctx context.Context, signature string, requester fosite.Requester) error {
	payload, err := s.marshalRequester(requester)
	if err != nil {
		return err
	}
	result, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_authorization_codes
		SET pkce_request_data = $1::jsonb
		WHERE signature_hash = $2
		  AND request_id = $3::uuid
		  AND invalidated_at IS NULL
	`, payload, hashOIDCCredential(signature), requester.GetID())
	if err != nil {
		return err
	}
	return requireOneAffected(result)
}

func (s *Store) GetPKCERequestSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	var payload []byte
	err := s.executor(ctx).QueryRowContext(ctx, `
		SELECT pkce_request_data
		FROM lihe_oidc_authorization_codes
		WHERE signature_hash = $1
		  AND pkce_request_data IS NOT NULL
		  AND invalidated_at IS NULL
		  AND expires_at > NOW()
	`, hashOIDCCredential(signature)).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.unmarshalRequester(ctx, payload)
}

func (s *Store) DeletePKCERequestSession(context.Context, string) error {
	// Fosite calls this before it starts the authorization-code exchange
	// transaction. The code invalidation removes the PKCE row atomically after
	// a successful verifier check, so a transient later failure remains retryable.
	return nil
}

func (s *Store) CreateAccessTokenSession(ctx context.Context, signature string, requester fosite.Requester) error {
	payload, err := s.marshalRequester(requester)
	if err != nil {
		return err
	}
	profile, err := s.getActiveProfile(ctx, "u.oidc_subject = $1::uuid", requester.GetSession().GetSubject())
	if err != nil {
		return err
	}
	expiresAt := requester.GetSession().GetExpiresAt(fosite.AccessToken)
	if expiresAt.IsZero() {
		return errors.New("OIDC access token expiry is missing")
	}
	_, _ = s.executor(ctx).ExecContext(ctx, `
		DELETE FROM lihe_oidc_access_tokens
		WHERE expires_at < NOW() - INTERVAL '1 day'
	`)
	_, err = s.executor(ctx).ExecContext(ctx, `
		INSERT INTO lihe_oidc_access_tokens (
			request_id, user_id, oidc_subject, signature_hash, client_id,
			scopes, request_data, expires_at
		) VALUES ($1::uuid, $2, $3::uuid, $4, $5, $6, $7::jsonb, $8)
	`,
		requester.GetID(),
		profile.UserID,
		profile.Subject,
		hashOIDCCredential(signature),
		requester.GetClient().GetID(),
		pq.Array([]string(requester.GetGrantedScopes())),
		payload,
		expiresAt,
	)
	return err
}

func (s *Store) GetAccessTokenSession(ctx context.Context, signature string, _ fosite.Session) (fosite.Requester, error) {
	var payload []byte
	err := s.executor(ctx).QueryRowContext(ctx, `
		SELECT t.request_data
		FROM lihe_oidc_access_tokens t
		JOIN users u ON u.id = t.user_id AND u.oidc_subject = t.oidc_subject
		WHERE t.signature_hash = $1
		  AND t.revoked_at IS NULL
		  AND t.expires_at > NOW()
		  AND u.deleted_at IS NULL
		  AND u.status = 'active'
	`, hashOIDCCredential(signature)).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fosite.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.unmarshalRequester(ctx, payload)
}

func (s *Store) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	_, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_access_tokens
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE signature_hash = $1
	`, hashOIDCCredential(signature))
	return err
}

func (s *Store) CreateRefreshTokenSession(context.Context, string, string, fosite.Requester) error {
	return fosite.ErrInvalidRequest
}

func (s *Store) GetRefreshTokenSession(context.Context, string, fosite.Session) (fosite.Requester, error) {
	return nil, fosite.ErrNotFound
}

func (s *Store) DeleteRefreshTokenSession(context.Context, string) error {
	return nil
}

func (s *Store) RotateRefreshToken(context.Context, string, string) error {
	return fosite.ErrInvalidRequest
}

func (s *Store) RevokeRefreshToken(context.Context, string) error {
	return nil
}

func (s *Store) RevokeAccessToken(ctx context.Context, requestID string) error {
	_, err := s.executor(ctx).ExecContext(ctx, `
		UPDATE lihe_oidc_access_tokens
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE request_id = $1::uuid
	`, requestID)
	return err
}

func hashOIDCCredential(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func randomOpaqueValue() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func cloneURLValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func requireOneAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fosite.ErrNotFound
	}
	return nil
}

func emailLocalPart(email string) string {
	local := strings.TrimSpace(email)
	if at := strings.IndexByte(local, '@'); at > 0 {
		local = local[:at]
	}
	if local == "" {
		return "user"
	}
	return local
}

var (
	_ fosite.Storage                         = (*Store)(nil)
	_ oauth2.CoreStorage                     = (*Store)(nil)
	_ oauth2.TokenRevocationStorage          = (*Store)(nil)
	_ openid.OpenIDConnectRequestStorage     = (*Store)(nil)
	_ pkce.PKCERequestStorage                = (*Store)(nil)
	_ fositestorage.Transactional            = (*Store)(nil)
)
