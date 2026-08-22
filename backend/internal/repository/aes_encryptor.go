package repository

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AESEncryptor implements SecretEncryptor using AES-256-GCM
type AESEncryptor struct {
	key           []byte
	blindIndexKey []byte
}

// NewAESEncryptor creates a new AES encryptor
func NewAESEncryptor(cfg *config.Config) (service.SecretEncryptor, error) {
	encryptor, err := NewAESHexEncryptor(cfg.Totp.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid totp encryption key: %w", err)
	}
	return encryptor, nil
}

// NewAESHexEncryptor creates an AES encryptor from a dedicated 32-byte hex key.
func NewAESHexEncryptor(keyHex string) (*AESEncryptor, error) {
	key, err := hex.DecodeString(strings.TrimSpace(keyHex))
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	deriver := hmac.New(sha256.New, key)
	_, _ = deriver.Write([]byte("sub2api/managed-recharge/blind-index/v1"))
	return &AESEncryptor{key: key, blindIndexKey: deriver.Sum(nil)}, nil
}

// BlindIndex returns a keyed, non-reversible index suitable for deduplication.
func (e *AESEncryptor) BlindIndex(plaintext string) string {
	mac := hmac.New(sha256.New, e.blindIndexKey)
	_, _ = mac.Write([]byte(plaintext))
	return hex.EncodeToString(mac.Sum(nil))
}

// Encrypt encrypts plaintext using AES-256-GCM
// Output format: base64(nonce + ciphertext + tag)
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// Encrypt the plaintext
	// Seal appends the ciphertext and tag to the nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertextData := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
