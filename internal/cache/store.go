package cache

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database used for caching data locally
type Store struct {
	db *sql.DB
}

// Open opens (and creates/migrates) the cache database at the given path
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("empty cache path")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	// Ensure file exists with strict perms
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create cache db: %w", err)
		}
		f.Close()
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open cache db: %w", err)
	}
	// Pragmas
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set WAL: %w", err)
	}
	_, _ = db.ExecContext(ctx, "PRAGMA foreign_keys=ON;")
	_, _ = db.ExecContext(ctx, "PRAGMA busy_timeout=5000;")
	_, _ = db.ExecContext(ctx, "PRAGMA synchronous=NORMAL;")
	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	// user_version based migrations (v1 initializes ai_summaries)
	var ver int
	_ = s.db.QueryRowContext(ctx, "PRAGMA user_version;").Scan(&ver)
	if ver == 0 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS ai_summaries (
  account_email TEXT NOT NULL,
  message_id    TEXT NOT NULL,
  summary       TEXT NOT NULL,
  updated_at    INTEGER NOT NULL,
  PRIMARY KEY (account_email, message_id)
);
`)
		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=1;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v1: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 1
	}
	return nil
}

// Close closes the underlying database
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SaveAISummary upserts a summary for (account_email, message_id)
func (s *Store) SaveAISummary(ctx context.Context, accountEmail, messageID, summary string, updatedAt int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" || strings.TrimSpace(summary) == "" {
		return fmt.Errorf("invalid summary inputs")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_summaries(account_email, message_id, summary, updated_at)
VALUES(?,?,?,?)
ON CONFLICT(account_email, message_id) DO UPDATE SET summary=excluded.summary, updated_at=excluded.updated_at;
`, accountEmail, messageID, summary, updatedAt)
	return err
}

// LoadAISummary returns a cached summary if present
func (s *Store) LoadAISummary(ctx context.Context, accountEmail, messageID string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, fmt.Errorf("cache store not initialized")
	}
	var out string
	err := s.db.QueryRowContext(ctx, `SELECT summary FROM ai_summaries WHERE account_email=? AND message_id=?`, accountEmail, messageID).Scan(&out)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}

// DeleteAISummary removes a cached summary for (account_email, message_id)
func (s *Store) DeleteAISummary(ctx context.Context, accountEmail, messageID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM ai_summaries WHERE account_email=? AND message_id=?`, accountEmail, messageID)
	return err
}
