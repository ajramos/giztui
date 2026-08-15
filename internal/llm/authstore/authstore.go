// Package authstore persists OAuth tokens for subscription-based LLM providers
// (e.g. ChatGPT reused from a Plus/Pro subscription) in a dedicated file, keyed
// by provider, separate from config.json so secrets never live in the main
// config. Tokens are machine-wide: one login is reused across every Gmail
// account that selects the provider.
package authstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Token is a stored OAuth credential for one provider.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	AccountID    string    `json:"account_id,omitempty"` // e.g. ChatGPT account id from the id_token
	Expiry       time.Time `json:"expiry,omitempty"`
}

// Expired reports whether the access token is at or within skew of expiry. A
// zero Expiry is treated as valid (unknown lifetime → refresh reactively on a
// 401 instead of proactively).
func (t Token) Expired(skew time.Duration) bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().Add(skew).After(t.Expiry)
}

// Store is a provider→Token map persisted to a 0600 JSON file.
type Store struct {
	path string
	mu   sync.Mutex
}

// DefaultPath returns ~/.config/giztui/llm-auth.json (empty if home is unknown).
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "giztui", "llm-auth.json")
}

// New creates a store backed by the given file path.
func New(path string) *Store { return &Store{path: path} }

func (s *Store) readAll() (map[string]Token, error) {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return map[string]Token{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]Token{}
	if len(b) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return m, nil
}

func (s *Store) writeAll(m map[string]Token) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Get returns the token for provider; ok is false when absent.
func (s *Store) Get(provider string) (Token, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.readAll()
	if err != nil {
		return Token{}, false, err
	}
	t, ok := m[provider]
	return t, ok, nil
}

// Put stores (overwriting) the token for provider; the file is written 0600.
func (s *Store) Put(provider string, t Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.readAll()
	if err != nil {
		return err
	}
	m[provider] = t
	return s.writeAll(m)
}

// Delete removes the token for provider (no error when absent).
func (s *Store) Delete(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, err := s.readAll()
	if err != nil {
		return err
	}
	if _, ok := m[provider]; !ok {
		return nil
	}
	delete(m, provider)
	return s.writeAll(m)
}
