package server

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	managedRechargeEncryptionKeyEnv     = "MANAGED_RECHARGE_ENCRYPTION_KEY"
	managedRechargeEncryptionKeyFileEnv = "MANAGED_RECHARGE_ENCRYPTION_KEY_FILE"
	managedRechargeEncryptionKeyMaxSize = 1024
)

func newManagedRechargeSecretProtector() (service.ManagedRechargeSecretProtector, error) {
	key, err := loadManagedRechargeEncryptionKey()
	if err != nil {
		return nil, err
	}
	return repository.NewAESHexEncryptor(key)
}

func loadManagedRechargeEncryptionKey() (string, error) {
	inlineKey := strings.TrimSpace(os.Getenv(managedRechargeEncryptionKeyEnv))
	keyFile := strings.TrimSpace(os.Getenv(managedRechargeEncryptionKeyFileEnv))
	if inlineKey != "" && keyFile != "" {
		return "", fmt.Errorf("configure only one of %s or %s", managedRechargeEncryptionKeyEnv, managedRechargeEncryptionKeyFileEnv)
	}
	if keyFile == "" {
		if inlineKey == "" {
			return "", fmt.Errorf("%s or %s is required", managedRechargeEncryptionKeyEnv, managedRechargeEncryptionKeyFileEnv)
		}
		return inlineKey, nil
	}

	// keyFile is a startup-only path supplied by the server operator through
	// the process environment; request data cannot influence it.
	//nolint:gosec // G703: the configured path is an intentional operator trust boundary.
	file, err := os.Open(keyFile)
	if err != nil {
		return "", fmt.Errorf("open managed recharge encryption key file: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat managed recharge encryption key file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("managed recharge encryption key file must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("managed recharge encryption key file permissions must not allow group or other access")
	}
	if info.Size() > managedRechargeEncryptionKeyMaxSize {
		return "", fmt.Errorf("managed recharge encryption key file is too large")
	}
	payload, err := io.ReadAll(io.LimitReader(file, managedRechargeEncryptionKeyMaxSize+1))
	if err != nil {
		return "", fmt.Errorf("read managed recharge encryption key file: %w", err)
	}
	if len(payload) > managedRechargeEncryptionKeyMaxSize {
		return "", fmt.Errorf("managed recharge encryption key file is too large")
	}
	key := strings.TrimSpace(string(payload))
	if key == "" {
		return "", fmt.Errorf("managed recharge encryption key file is empty")
	}
	return key, nil
}
