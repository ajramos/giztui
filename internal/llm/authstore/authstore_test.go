package authstore

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRoundTripAndPerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "llm-auth.json")
	s := New(path)

	if _, ok, err := s.Get("chatgpt"); err != nil || ok {
		t.Fatalf("empty store: ok=%v err=%v", ok, err)
	}

	tok := Token{AccessToken: "at", RefreshToken: "rt", AccountID: "acct-1",
		Expiry: time.Now().Add(time.Hour).UTC().Truncate(time.Second)}
	if err := s.Put("chatgpt", tok); err != nil {
		t.Fatalf("put: %v", err)
	}

	// File must be 0600.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}

	got, ok, err := s.Get("chatgpt")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.AccessToken != "at" || got.RefreshToken != "rt" || got.AccountID != "acct-1" {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	// A second provider coexists without clobbering the first.
	if err := s.Put("other", Token{AccessToken: "x"}); err != nil {
		t.Fatalf("put other: %v", err)
	}
	if got, ok, _ := s.Get("chatgpt"); !ok || got.AccessToken != "at" {
		t.Errorf("first provider clobbered: ok=%v %+v", ok, got)
	}

	if err := s.Delete("chatgpt"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, _ := s.Get("chatgpt"); ok {
		t.Error("token still present after delete")
	}
	if _, ok, _ := s.Get("other"); !ok {
		t.Error("delete removed the wrong provider")
	}
	// Deleting an absent provider is a no-op, not an error.
	if err := s.Delete("nope"); err != nil {
		t.Errorf("delete absent: %v", err)
	}
}

func TestTokenExpired(t *testing.T) {
	if (Token{}).Expired(time.Minute) {
		t.Error("zero expiry must be treated as valid")
	}
	past := Token{Expiry: time.Now().Add(-time.Minute)}
	if !past.Expired(0) {
		t.Error("past expiry must be expired")
	}
	soon := Token{Expiry: time.Now().Add(30 * time.Second)}
	if !soon.Expired(time.Minute) {
		t.Error("within skew must be treated as expired")
	}
	if soon.Expired(0) {
		t.Error("30s ahead with no skew must be valid")
	}
}
