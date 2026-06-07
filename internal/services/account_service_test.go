package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/stretchr/testify/assert"
)

// writeDefaultCredentialFiles creates ~/.config/giztui/{credentials,token}.json under the
// given home directory and returns the giztui config dir.
func writeDefaultCredentialFiles(t *testing.T, home string, withToken bool) string {
	t.Helper()
	dir := filepath.Join(home, ".config", "giztui")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte(`{"installed":{}}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	if withToken {
		if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(`{"access_token":"x"}`), 0o600); err != nil {
			t.Fatalf("write token: %v", err)
		}
	}
	return dir
}

// TestAccountService_LegacyFallback_DefaultFilesExist verifies the regression from issue #42:
// with no `accounts` and no `credentials`/`token` in config, but the default credential files
// present, a default account is created so the database can initialize.
func TestAccountService_LegacyFallback_DefaultFilesExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeDefaultCredentialFiles(t, home, true)

	cfg := &config.Config{} // no Accounts, no Credentials, no Token
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, "default", acc.ID)
	assert.True(t, acc.IsActive)
	assert.Contains(t, acc.CredPath, filepath.Join(".config", "giztui", "credentials.json"))
	assert.Contains(t, acc.TokenPath, filepath.Join(".config", "giztui", "token.json"))
}

// TestAccountService_LegacyFallback_NoFiles verifies that without any credential files and no
// config, no account is created (and GetActiveAccount errors), rather than a phantom account.
func TestAccountService_LegacyFallback_NoFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Intentionally do NOT create any credential files.

	cfg := &config.Config{}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.Error(t, err)
	assert.Nil(t, acc)
}

// TestAccountService_LegacyFallback_ExplicitConfigStillWins verifies that an explicit config
// Credentials path is honored (and the existing behavior is preserved).
func TestAccountService_LegacyFallback_ExplicitConfigWins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Put the credential file at a non-default location and point config at it.
	customDir := filepath.Join(home, "custom")
	if err := os.MkdirAll(customDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	credPath := filepath.Join(customDir, "creds.json")
	if err := os.WriteFile(credPath, []byte(`{"installed":{}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := &config.Config{Credentials: credPath}
	svc := NewAccountService(cfg, nil)

	acc, err := svc.GetActiveAccount(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, credPath, acc.CredPath)
}

func TestResolveLegacyCredentialPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Empty configured -> default ~/.config/giztui/<file>, ~ expanded.
	got := resolveLegacyCredentialPath("", "token.json")
	assert.Equal(t, filepath.Join(home, ".config", "giztui", "token.json"), got)

	// Configured absolute path passes through unchanged.
	abs := filepath.Join(home, "x", "creds.json")
	assert.Equal(t, abs, resolveLegacyCredentialPath(abs, "credentials.json"))

	// Configured ~ path is expanded.
	assert.Equal(t, filepath.Join(home, "creds.json"), resolveLegacyCredentialPath("~/creds.json", "credentials.json"))
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f.json")
	assert.False(t, fileExists(f))
	assert.False(t, fileExists(""))
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	assert.True(t, fileExists(f))
	assert.False(t, fileExists(dir)) // directory is not a regular file
}
