package oidcprovider

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	encryptedPayloadVersion = 1
	payloadKeyDomain         = "lihe-oidc-storage-key-v1\x00"
	payloadAAD               = "lihe-oidc-storage-payload-v1"
)

type encryptedPayload struct {
	Version    int    `json:"v"`
	Nonce      string `json:"n"`
	Ciphertext string `json:"c"`
}

type payloadCipher struct {
	aead cipher.AEAD
}

func newPayloadCipher(secret string) (*payloadCipher, error) {
	if len(secret) < 32 {
		return nil, errors.New("OIDC storage secret must be at least 32 bytes")
	}
	key := sha256.Sum256(append([]byte(payloadKeyDomain), []byte(secret)...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("initialize OIDC storage cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize OIDC storage AEAD: %w", err)
	}
	return &payloadCipher{aead: aead}, nil
}

func (c *payloadCipher) Seal(value any) (string, error) {
	if c == nil || c.aead == nil {
		return "", errors.New("OIDC storage cipher is unavailable")
	}
	plaintext, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal OIDC storage payload: %w", err)
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate OIDC storage nonce: %w", err)
	}
	ciphertext := c.aead.Seal(nil, nonce, plaintext, []byte(payloadAAD))
	envelope, err := json.Marshal(encryptedPayload{
		Version:    encryptedPayloadVersion,
		Nonce:      base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(ciphertext),
	})
	if err != nil {
		return "", fmt.Errorf("marshal OIDC encrypted payload: %w", err)
	}
	return string(envelope), nil
}

func (c *payloadCipher) Open(payload []byte, target any) error {
	if c == nil || c.aead == nil {
		return errors.New("OIDC storage cipher is unavailable")
	}
	if target == nil {
		return errors.New("OIDC storage payload target is required")
	}
	var envelope encryptedPayload
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode OIDC encrypted payload: %w", err)
	}
	if envelope.Version != encryptedPayloadVersion {
		return fmt.Errorf("unsupported OIDC encrypted payload version: %d", envelope.Version)
	}
	nonce, err := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	if err != nil || len(nonce) != c.aead.NonceSize() {
		return errors.New("invalid OIDC encrypted payload nonce")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Ciphertext)
	if err != nil || len(ciphertext) < c.aead.Overhead() {
		return errors.New("invalid OIDC encrypted payload ciphertext")
	}
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, []byte(payloadAAD))
	if err != nil {
		return errors.New("authenticate OIDC encrypted payload")
	}
	if err := json.Unmarshal(plaintext, target); err != nil {
		return fmt.Errorf("decode OIDC storage payload: %w", err)
	}
	return nil
}
