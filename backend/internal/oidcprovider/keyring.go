package oidcprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v3"
)

const (
	currentPrivateKeyFile  = "current.pem"
	previousPrivateKeyFile = "previous.pem"
	rsaKeyBits             = 2048
	keyRotationCheckEvery  = time.Minute
)

// KeyRing keeps the active and immediately previous RS256 key on persistent
// storage. Private material never leaves this type; JWKS receives public copies.
type KeyRing struct {
	directory      string
	rotationPeriod time.Duration

	mu               sync.Mutex
	current          *jose.JSONWebKey
	previous         *jose.JSONWebKey
	currentCreatedAt time.Time
	lastCheckedAt    time.Time
}

func NewKeyRing(directory string, rotationDays int) (*KeyRing, error) {
	if directory == "" {
		return nil, errors.New("OIDC key directory is required")
	}
	if rotationDays < 1 {
		return nil, errors.New("OIDC key rotation period must be positive")
	}
	ring := &KeyRing{
		directory:      directory,
		rotationPeriod: time.Duration(rotationDays) * 24 * time.Hour,
	}
	if err := ring.ensure(time.Now().UTC(), true); err != nil {
		return nil, err
	}
	return ring, nil
}

// CurrentPrivateKey is the key getter used by Fosite's RS256 strategy.
func (r *KeyRing) CurrentPrivateKey(_ context.Context) (any, error) {
	if err := r.ensure(time.Now().UTC(), false); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return nil, errors.New("OIDC current signing key is unavailable")
	}
	return r.current, nil
}

func (r *KeyRing) PublicJWKS() (*jose.JSONWebKeySet, error) {
	if err := r.ensure(time.Now().UTC(), false); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]jose.JSONWebKey, 0, 2)
	if r.current != nil {
		keys = append(keys, publicJWK(r.current))
	}
	if r.previous != nil && (r.current == nil || r.previous.KeyID != r.current.KeyID) {
		keys = append(keys, publicJWK(r.previous))
	}
	return &jose.JSONWebKeySet{Keys: keys}, nil
}

func (r *KeyRing) ensure(now time.Time, force bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(r.directory, 0o700); err != nil {
		return fmt.Errorf("create OIDC key directory: %w", err)
	}
	if err := os.Chmod(r.directory, 0o700); err != nil {
		return fmt.Errorf("secure OIDC key directory: %w", err)
	}
	if r.current == nil {
		if err := r.loadOrCreateLocked(now); err != nil {
			return err
		}
	}
	if !force && !r.lastCheckedAt.IsZero() && now.Sub(r.lastCheckedAt) < keyRotationCheckEvery {
		return nil
	}
	r.lastCheckedAt = now
	if now.Sub(r.currentCreatedAt) < r.rotationPeriod {
		return nil
	}
	return r.rotateLocked(now)
}

func (r *KeyRing) loadOrCreateLocked(now time.Time) error {
	currentPath := filepath.Join(r.directory, currentPrivateKeyFile)
	key, createdAt, err := readPrivateJWK(currentPath)
	if errors.Is(err, os.ErrNotExist) {
		privateKey, generateErr := rsa.GenerateKey(rand.Reader, rsaKeyBits)
		if generateErr != nil {
			return fmt.Errorf("generate OIDC signing key: %w", generateErr)
		}
		key, err = privateJWK(privateKey)
		if err != nil {
			return err
		}
		if err = writePrivateKeyAtomically(currentPath, privateKey); err != nil {
			return fmt.Errorf("persist OIDC signing key: %w", err)
		}
		createdAt = now
	} else if err != nil {
		return fmt.Errorf("load OIDC current signing key: %w", err)
	}
	if err := os.Chmod(currentPath, 0o600); err != nil {
		return fmt.Errorf("secure OIDC current signing key: %w", err)
	}
	r.current = key
	r.currentCreatedAt = createdAt.UTC()

	previousPath := filepath.Join(r.directory, previousPrivateKeyFile)
	previous, _, err := readPrivateJWK(previousPath)
	if err == nil {
		if chmodErr := os.Chmod(previousPath, 0o600); chmodErr != nil {
			return fmt.Errorf("secure OIDC previous signing key: %w", chmodErr)
		}
		r.previous = previous
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load OIDC previous signing key: %w", err)
	}
	return nil
}

func (r *KeyRing) rotateLocked(now time.Time) error {
	currentPrivate, ok := r.current.Key.(*rsa.PrivateKey)
	if !ok || currentPrivate == nil {
		return errors.New("OIDC current signing key is not RSA private material")
	}
	newPrivate, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return fmt.Errorf("rotate OIDC signing key: %w", err)
	}
	newCurrent, err := privateJWK(newPrivate)
	if err != nil {
		return err
	}
	previousPath := filepath.Join(r.directory, previousPrivateKeyFile)
	if err := writePrivateKeyAtomically(previousPath, currentPrivate); err != nil {
		return fmt.Errorf("persist OIDC previous signing key: %w", err)
	}
	currentPath := filepath.Join(r.directory, currentPrivateKeyFile)
	if err := writePrivateKeyAtomically(currentPath, newPrivate); err != nil {
		return fmt.Errorf("persist rotated OIDC signing key: %w", err)
	}
	r.previous = r.current
	r.current = newCurrent
	r.currentCreatedAt = now
	return nil
}

func readPrivateJWK(path string) (*jose.JSONWebKey, time.Time, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, time.Time{}, errors.New("invalid PEM data")
	}
	var privateKey *rsa.PrivateKey
	if parsed, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes); parseErr == nil {
		var ok bool
		privateKey, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, time.Time{}, errors.New("PEM key is not RSA")
		}
	} else {
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("parse RSA private key: %w", err)
		}
	}
	if err := privateKey.Validate(); err != nil {
		return nil, time.Time{}, fmt.Errorf("validate RSA private key: %w", err)
	}
	jwk, err := privateJWK(privateKey)
	if err != nil {
		return nil, time.Time{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	return jwk, info.ModTime(), nil
}

func privateJWK(key *rsa.PrivateKey) (*jose.JSONWebKey, error) {
	if key == nil {
		return nil, errors.New("nil RSA private key")
	}
	kid, err := rsaKeyID(&key.PublicKey)
	if err != nil {
		return nil, err
	}
	return &jose.JSONWebKey{
		Key:       key,
		KeyID:     kid,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}, nil
}

func publicJWK(private *jose.JSONWebKey) jose.JSONWebKey {
	public := private.Public()
	public.Algorithm = string(jose.RS256)
	public.Use = "sig"
	return public
}

func rsaKeyID(publicKey *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("marshal OIDC public key: %w", err)
	}
	digest := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func writePrivateKeyAtomically(path string, key *rsa.PrivateKey) (err error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	raw := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	tmpPath := filepath.Join(filepath.Dir(path), fmt.Sprintf(".%s.%d.tmp", filepath.Base(path), time.Now().UnixNano()))
	file, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err = file.Write(raw); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
