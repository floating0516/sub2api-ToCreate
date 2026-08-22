//go:build unit

package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManagedRechargeEncryptionKeyFromPrivateFile(t *testing.T) {
	t.Setenv(managedRechargeEncryptionKeyEnv, "")
	path := filepath.Join(t.TempDir(), "managed-recharge.key")
	key := strings.Repeat("42", 32)
	if err := os.WriteFile(path, []byte(key+"\n"), 0o600); err != nil {
		t.Fatalf("write managed recharge key: %v", err)
	}
	t.Setenv(managedRechargeEncryptionKeyFileEnv, path)

	actual, err := loadManagedRechargeEncryptionKey()
	if err != nil {
		t.Fatalf("load managed recharge key: %v", err)
	}
	if actual != key {
		t.Fatal("loaded managed recharge key does not match")
	}
}

func TestLoadManagedRechargeEncryptionKeyRejectsPublicFile(t *testing.T) {
	t.Setenv(managedRechargeEncryptionKeyEnv, "")
	path := filepath.Join(t.TempDir(), "managed-recharge.key")
	if err := os.WriteFile(path, []byte(strings.Repeat("42", 32)), 0o644); err != nil {
		t.Fatalf("write managed recharge key: %v", err)
	}
	t.Setenv(managedRechargeEncryptionKeyFileEnv, path)

	if _, err := loadManagedRechargeEncryptionKey(); err == nil {
		t.Fatal("public managed recharge key file was accepted")
	}
}

func TestLoadManagedRechargeEncryptionKeyRejectsAmbiguousSources(t *testing.T) {
	t.Setenv(managedRechargeEncryptionKeyEnv, strings.Repeat("42", 32))
	t.Setenv(managedRechargeEncryptionKeyFileEnv, "/tmp/managed-recharge.key")

	if _, err := loadManagedRechargeEncryptionKey(); err == nil {
		t.Fatal("ambiguous managed recharge key sources were accepted")
	}
}
