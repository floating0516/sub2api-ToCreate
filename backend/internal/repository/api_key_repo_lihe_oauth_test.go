package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGetLiheOIDCSubjectRequiresActiveCurrentUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := newAPIKeyRepositoryWithSQL(nil, db)

	want := "11111111-1111-4111-8111-111111111111"
	mock.ExpectQuery(`(?s)SELECT oidc_subject::text.*deleted_at IS NULL.*status = \$2`).
		WithArgs(int64(42), service.StatusActive).
		WillReturnRows(sqlmock.NewRows([]string{"oidc_subject"}).AddRow(want))

	got, err := repo.GetLiheOIDCSubject(t.Context(), 42)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExchangeLiheAuthorizationCodeBindsExistingAPIKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := newAPIKeyRepositoryWithSQL(nil, db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT id, user_id, api_key_id, client_id.*FOR UPDATE`).
		WithArgs("code-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "api_key_id", "client_id", "redirect_uri", "code_challenge", "expires_at", "used",
		}).AddRow(int64(7), int64(11), int64(501), "lihe-chat", "https://lihe.chat/callback", "challenge", now.Add(time.Minute), false))
	mock.ExpectQuery(`(?s)SELECT k.name, k.status, k.group_id, g.status, g.platform.*WHERE k.id = \$1 AND k.user_id = \$2`).
		WithArgs(int64(501), int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"name", "status", "group_id", "group_status", "platform"}).
			AddRow("Selected key", service.StatusAPIKeyActive, int64(101), service.StatusActive, service.PlatformOpenAI))
	mock.ExpectQuery(`(?s)INSERT INTO lihe_oauth_access_tokens.*RETURNING id, created_at`).
		WithArgs(int64(11), "token-hash", service.LiheOAuthClientName, "lihe-chat", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(19), now))
	mock.ExpectExec(`(?s)INSERT INTO lihe_oauth_token_bindings \(token_id, provider, group_id, source_api_key_id\)`).
		WithArgs(int64(19), service.PlatformOpenAI, int64(101), int64(501)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?s)UPDATE lihe_oauth_authorization_codes.*used = TRUE`).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.ExchangeLiheAuthorizationCode(context.Background(), service.LiheTokenExchangeInput{
		CodeHash:      "code-hash",
		UserID:        11,
		ClientID:      "lihe-chat",
		RedirectURI:   "https://lihe.chat/callback",
		CodeChallenge: "challenge",
		TokenHash:     "token-hash",
		Scopes:        []string{service.LiheOAuthScopeModelsRead, service.LiheOAuthScopeChatWrite},
		Bindings: []service.LiheTokenBindingInput{{
			Provider: service.PlatformOpenAI,
			GroupID:  101,
			APIKeyID: 501,
		}},
	})

	require.NoError(t, err)
	require.Equal(t, int64(19), result.ID)
	require.Equal(t, []string{service.PlatformOpenAI}, result.Providers)
	require.NotNil(t, result.APIKeyID)
	require.Equal(t, int64(501), *result.APIKeyID)
	require.Equal(t, "Selected key", result.APIKeyName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExchangeLiheAuthorizationCodeRejectsDifferentSelectedKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := newAPIKeyRepositoryWithSQL(nil, db)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT id, user_id, api_key_id, client_id.*FOR UPDATE`).
		WithArgs("code-hash").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "api_key_id", "client_id", "redirect_uri", "code_challenge", "expires_at", "used",
		}).AddRow(int64(7), int64(11), int64(999), "lihe-chat", "https://lihe.chat/callback", "challenge", time.Now().Add(time.Minute), false))
	mock.ExpectRollback()

	_, err = repo.ExchangeLiheAuthorizationCode(context.Background(), service.LiheTokenExchangeInput{
		CodeHash:      "code-hash",
		UserID:        11,
		ClientID:      "lihe-chat",
		RedirectURI:   "https://lihe.chat/callback",
		CodeChallenge: "challenge",
		Bindings: []service.LiheTokenBindingInput{{
			Provider: service.PlatformOpenAI,
			GroupID:  101,
			APIKeyID: 501,
		}},
	})

	require.ErrorIs(t, err, service.ErrLiheInvalidGrant)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRevokeLiheAccessTokenDisablesOnlyLegacyInternalKeys(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := newAPIKeyRepositoryWithSQL(nil, db)

	mock.ExpectQuery(`(?s)WITH target AS.*UPDATE api_keys k.*SELECT b.api_key_id.*k.key LIKE \$4`).
		WithArgs(int64(19), int64(11), service.StatusAPIKeyDisabled, service.LiheInternalAPIKeyPrefix+"%").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	revoked, err := repo.RevokeLiheAccessTokenByID(context.Background(), 19, 11)

	require.NoError(t, err)
	require.True(t, revoked)
	require.NoError(t, mock.ExpectationsWereMet())
}
