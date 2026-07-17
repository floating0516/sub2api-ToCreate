package oidcprovider

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/require"
)

func TestKeyRingPublishesPublicRS256KeyAndSignerAddsKID(t *testing.T) {
	directory := t.TempDir()
	ring, err := NewKeyRing(directory, 30)
	require.NoError(t, err)

	jwks, err := ring.PublicJWKS()
	require.NoError(t, err)
	require.Len(t, jwks.Keys, 1)
	key := jwks.Keys[0]
	require.NotEmpty(t, key.KeyID)
	require.Equal(t, "RS256", key.Algorithm)
	require.Equal(t, "sig", key.Use)
	_, isPublicRSA := key.Key.(*rsa.PublicKey)
	require.True(t, isPublicRSA)

	signer := &jwt.DefaultSigner{GetPrivateKey: ring.CurrentPrivateKey}
	rawToken, _, err := signer.Generate(
		context.Background(),
		jwt.MapClaims{"sub": "test-subject"},
		jwt.NewHeaders(),
	)
	require.NoError(t, err)
	parts := strings.Split(rawToken, ".")
	require.Len(t, parts, 3)
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	var header map[string]any
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	require.Equal(t, "RS256", header["alg"])
	require.Equal(t, key.KeyID, header["kid"])

	info, err := os.Stat(filepath.Join(directory, currentPrivateKeyFile))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestKeyRingRotationPublishesCurrentAndPreviousPublicKeys(t *testing.T) {
	ring, err := NewKeyRing(t.TempDir(), 30)
	require.NoError(t, err)
	before, err := ring.PublicJWKS()
	require.NoError(t, err)
	require.Len(t, before.Keys, 1)

	now := time.Now().UTC()
	ring.mu.Lock()
	ring.rotationPeriod = time.Hour
	ring.currentCreatedAt = now.Add(-2 * time.Hour)
	ring.lastCheckedAt = time.Time{}
	ring.mu.Unlock()
	require.NoError(t, ring.ensure(now, false))

	after, err := ring.PublicJWKS()
	require.NoError(t, err)
	require.Len(t, after.Keys, 2)
	require.NotEqual(t, after.Keys[0].KeyID, after.Keys[1].KeyID)
	for _, key := range after.Keys {
		_, isPublicRSA := key.Key.(*rsa.PublicKey)
		require.True(t, isPublicRSA)
	}
}

func TestMockJWKSFixtureContainsOnlyAValidPublicKey(t *testing.T) {
	payload, err := os.ReadFile("testdata/jwks.json")
	require.NoError(t, err)
	var fixture jose.JSONWebKeySet
	require.NoError(t, json.Unmarshal(payload, &fixture))
	require.Len(t, fixture.Keys, 1)
	key := fixture.Keys[0]
	require.True(t, key.Valid())
	publicKey, ok := key.Key.(*rsa.PublicKey)
	require.True(t, ok)
	require.Equal(t, 2048, publicKey.N.BitLen())
	require.Equal(t, 65537, publicKey.E)
	wantKID, err := rsaKeyID(publicKey)
	require.NoError(t, err)
	require.Equal(t, wantKID, key.KeyID)
}
