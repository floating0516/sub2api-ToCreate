package oidcprovider

import (
	"database/sql/driver"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/stretchr/testify/require"
)

type encryptedJSONArgument struct {
	forbidden []string
}

func (m encryptedJSONArgument) Match(value driver.Value) bool {
	payload, ok := value.(string)
	if !ok {
		return false
	}
	for _, forbidden := range m.forbidden {
		if strings.Contains(payload, forbidden) {
			return false
		}
	}
	var envelope encryptedPayload
	return json.Unmarshal([]byte(payload), &envelope) == nil &&
		envelope.Version == encryptedPayloadVersion &&
		envelope.Nonce != "" &&
		envelope.Ciphertext != ""
}

func newStoreTestClient() *fosite.DefaultOpenIDConnectClient {
	return &fosite.DefaultOpenIDConnectClient{
		DefaultClient: &fosite.DefaultClient{
			ID:            "lihe-chat-login",
			RedirectURIs:  []string{"https://lihe.chat/oauth/openid/callback"},
			GrantTypes:    []string{"authorization_code"},
			ResponseTypes: []string{"code"},
			Scopes:        []string{"openid", "profile", "email"},
		},
		TokenEndpointAuthMethod: "client_secret_basic",
	}
}

func TestCreatePendingRequestStoresEncryptedJSONStringForJSONB(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db, newStoreTestClient(), 5*time.Minute, strings.Repeat("h", 32))
	require.NoError(t, err)

	state := "state-must-not-be-plaintext"
	nonce := "nonce-must-not-be-plaintext"
	browserBinding := strings.Repeat("b", 43)
	mock.ExpectExec(`(?s)DELETE FROM lihe_oidc_pending_requests`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`(?s)INSERT INTO lihe_oidc_pending_requests`).
		WithArgs(
			sqlmock.AnyArg(),
			hashOIDCCredential(browserBinding),
			encryptedJSONArgument{forbidden: []string{state, nonce}},
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	handle, expiresAt, err := store.CreatePendingRequest(t.Context(), url.Values{
		"state": {state},
		"nonce": {nonce},
	}, browserBinding)
	require.NoError(t, err)
	require.Len(t, handle, 43)
	require.WithinDuration(t, time.Now().UTC().Add(5*time.Minute), expiresAt, 2*time.Second)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequesterPersistenceEncryptsTokenRequestSecrets(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewStore(db, newStoreTestClient(), 5*time.Minute, strings.Repeat("h", 32))
	require.NoError(t, err)

	session := openid.NewDefaultSession()
	session.Subject = "11111111-1111-4111-8111-111111111111"
	request := &fosite.Request{
		ID:          uuid.NewString(),
		RequestedAt: time.Now().UTC(),
		Client:      newStoreTestClient(),
		Form: url.Values{
			"code":          {"authorization-code-secret"},
			"code_verifier": {"pkce-verifier-secret"},
			"nonce":         {"nonce-secret"},
			"state":         {"state-secret"},
		},
		Session: session,
	}
	payload, err := store.marshalRequester(request)
	require.NoError(t, err)
	for _, secret := range request.Form {
		require.NotContains(t, payload, secret[0])
	}

	restored, err := store.unmarshalRequester(t.Context(), []byte(payload))
	require.NoError(t, err)
	require.Equal(t, request.ID, restored.GetID())
	require.Equal(t, request.Form, restored.GetRequestForm())
	require.Equal(t, session.Subject, restored.GetSession().GetSubject())
}
