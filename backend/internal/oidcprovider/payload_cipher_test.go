package oidcprovider

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPayloadCipherRoundTripAndConfidentiality(t *testing.T) {
	cipher, err := newPayloadCipher(strings.Repeat("s", 32))
	require.NoError(t, err)

	want := map[string]string{
		"state":         "state-must-not-be-stored-in-plaintext",
		"nonce":         "nonce-must-not-be-stored-in-plaintext",
		"code_verifier": "verifier-must-not-be-stored-in-plaintext",
	}
	payload, err := cipher.Seal(want)
	require.NoError(t, err)
	for _, secret := range want {
		require.NotContains(t, payload, secret)
	}

	var envelope encryptedPayload
	require.NoError(t, json.Unmarshal([]byte(payload), &envelope))
	require.Equal(t, encryptedPayloadVersion, envelope.Version)
	require.NotEmpty(t, envelope.Nonce)
	require.NotEmpty(t, envelope.Ciphertext)

	got := map[string]string{}
	require.NoError(t, cipher.Open([]byte(payload), &got))
	require.Equal(t, want, got)
}

func TestPayloadCipherRejectsTamperingAndWrongKey(t *testing.T) {
	cipher, err := newPayloadCipher(strings.Repeat("a", 32))
	require.NoError(t, err)
	payload, err := cipher.Seal(map[string]string{"nonce": "original-nonce"})
	require.NoError(t, err)

	var envelope encryptedPayload
	require.NoError(t, json.Unmarshal([]byte(payload), &envelope))
	if strings.HasSuffix(envelope.Ciphertext, "A") {
		envelope.Ciphertext = strings.TrimSuffix(envelope.Ciphertext, "A") + "B"
	} else {
		envelope.Ciphertext = envelope.Ciphertext[:len(envelope.Ciphertext)-1] + "A"
	}
	tampered, err := json.Marshal(envelope)
	require.NoError(t, err)
	require.Error(t, cipher.Open(tampered, &map[string]string{}))

	wrongCipher, err := newPayloadCipher(strings.Repeat("b", 32))
	require.NoError(t, err)
	require.Error(t, wrongCipher.Open([]byte(payload), &map[string]string{}))
}

func TestPayloadCipherRequiresIndependentSecretStrength(t *testing.T) {
	_, err := newPayloadCipher("short")
	require.ErrorContains(t, err, "at least 32 bytes")
}
