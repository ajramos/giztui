package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCredentialsJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		ok   bool
	}{
		{"desktop installed block", `{"installed":{"client_id":"abc.apps.googleusercontent.com","client_secret":"x"}}`, true},
		{"web block", `{"web":{"client_id":"abc.apps.googleusercontent.com"}}`, true},
		{"not json", `not json at all`, false},
		{"json but no oauth block", `{"foo":"bar"}`, false},
		{"installed without client_id", `{"installed":{"client_secret":"x"}}`, false},
		{"token file (wrong file)", `{"access_token":"ya29","refresh_token":"1//0"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCredentialsJSON([]byte(tc.in))
			if tc.ok && err != nil {
				t.Fatalf("expected valid, got error: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestInstallCredentials(t *testing.T) {
	dir := t.TempDir()
	// Point the resolved credentials path into the temp dir via the explicit
	// option so we don't touch the real ~/.config/giztui.
	dest := filepath.Join(dir, "config", "credentials.json")
	opts := Options{CredentialsPath: dest}

	src := filepath.Join(dir, "downloaded.json")
	good := `{"installed":{"client_id":"abc.apps.googleusercontent.com","client_secret":"s"}}`
	if err := os.WriteFile(src, []byte(good), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := InstallCredentials(opts, src)
	if err != nil {
		t.Fatalf("InstallCredentials: %v", err)
	}
	if got != dest {
		t.Fatalf("dest = %q, want %q", got, dest)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("copied file not readable: %v", err)
	}
	if string(data) != good {
		t.Fatalf("copied content mismatch")
	}
	// The parent directory should be created with tight perms and the file
	// should not be world-readable (it holds an OAuth client secret).
	if info, err := os.Stat(dest); err == nil {
		if perm := info.Mode().Perm(); perm&0o077 != 0 {
			t.Fatalf("credentials file perms too open: %v", perm)
		}
	}
}

func TestInstallCredentialsRejectsWrongFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "token.json")
	if err := os.WriteFile(src, []byte(`{"access_token":"ya29"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "credentials.json")
	if _, err := InstallCredentials(Options{CredentialsPath: dest}, src); err == nil {
		t.Fatalf("expected InstallCredentials to reject a non-credentials file")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("a rejected file must not be written to the destination")
	}
}

func TestInstallCredentialsMissingSource(t *testing.T) {
	if _, err := InstallCredentials(Options{}, ""); err == nil {
		t.Fatalf("expected error for empty source path")
	}
}
