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
	if ver == 1 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS messages_meta (
  id TEXT PRIMARY KEY,
  thread_id TEXT,
  snippet TEXT,
  internal_date INTEGER,
  label_ids TEXT,
  history_id INTEGER,
  etag TEXT,
  updated_at INTEGER,
  last_accessed INTEGER
);
CREATE INDEX IF NOT EXISTS idx_meta_date ON messages_meta(internal_date DESC);

CREATE TABLE IF NOT EXISTS messages_body (
  id TEXT PRIMARY KEY,
  plain TEXT,
  html TEXT,
  fetched_at INTEGER
);

CREATE TABLE IF NOT EXISTS labels (
  id TEXT PRIMARY KEY,
  name TEXT,
  type TEXT
);

CREATE TABLE IF NOT EXISTS sync_state (
  key TEXT PRIMARY KEY,
  value TEXT
);
`)
		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=2;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v2: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 2
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

// --- Messages & Labels cache ---

type MessageMeta struct {
	ID           string
	ThreadID     string
	Snippet      string
	InternalDate int64
	LabelIDsJSON string
	HistoryID    int64
	ETag         string
	UpdatedAt    int64
	LastAccessed int64
}

func (s *Store) UpsertMessageMeta(ctx context.Context, m MessageMeta) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO messages_meta(id, thread_id, snippet, internal_date, label_ids, history_id, etag, updated_at, last_accessed)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET thread_id=excluded.thread_id, snippet=excluded.snippet, internal_date=excluded.internal_date, label_ids=excluded.label_ids, history_id=excluded.history_id, etag=excluded.etag, updated_at=excluded.updated_at, last_accessed=excluded.last_accessed;
`, m.ID, m.ThreadID, m.Snippet, m.InternalDate, m.LabelIDsJSON, m.HistoryID, m.ETag, m.UpdatedAt, m.LastAccessed)
	return err
}

func (s *Store) UpsertMessageBody(ctx context.Context, id, plain, html string, fetchedAt int64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO messages_body(id, plain, html, fetched_at)
VALUES(?,?,?,?)
ON CONFLICT(id) DO UPDATE SET plain=excluded.plain, html=excluded.html, fetched_at=excluded.fetched_at;
`, id, plain, html, fetchedAt)
	return err
}

func (s *Store) GetMessageBody(ctx context.Context, id string) (plain, html string, ok bool, err error) {
	if s == nil || s.db == nil {
		return "", "", false, fmt.Errorf("cache store not initialized")
	}
	err = s.db.QueryRowContext(ctx, `SELECT plain, html FROM messages_body WHERE id=?`, id).Scan(&plain, &html)
	if err == sql.ErrNoRows {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return plain, html, true, nil
}

func (s *Store) LoadRecentMessageMetas(ctx context.Context, limit int) ([]MessageMeta, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("cache store not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, thread_id, snippet, internal_date, label_ids, history_id, etag, updated_at, last_accessed FROM messages_meta ORDER BY internal_date DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MessageMeta, 0, limit)
	for rows.Next() {
		var m MessageMeta
		if err := rows.Scan(&m.ID, &m.ThreadID, &m.Snippet, &m.InternalDate, &m.LabelIDsJSON, &m.HistoryID, &m.ETag, &m.UpdatedAt, &m.LastAccessed); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *Store) SaveLabels(ctx context.Context, labels map[string]struct{ Name, Type string }) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for id, v := range labels {
		if _, err := tx.ExecContext(ctx, `INSERT INTO labels(id, name, type) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, type=excluded.type;`, id, v.Name, v.Type); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

type LabelRow struct{ ID, Name, Type string }

func (s *Store) LoadLabels(ctx context.Context) ([]LabelRow, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("cache store not initialized")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, type FROM labels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LabelRow
	for rows.Next() {
		var r LabelRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Type); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) GetSyncState(ctx context.Context, key string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, fmt.Errorf("cache store not initialized")
	}
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM sync_state WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (s *Store) SetSyncState(ctx context.Context, key, value string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_state(key, value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value;`, key, value)
	return err
}

// ClearAISummaries deletes all AI summaries
func (s *Store) ClearAISummaries(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM ai_summaries;`); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `VACUUM;`)
	return nil
}

// ClearMessages deletes cached message bodies and metadata
func (s *Store) ClearMessages(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages_body;`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages_meta;`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `VACUUM;`)
	return nil
}

// ClearSyncState deletes sync_state rows
func (s *Store) ClearSyncState(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM sync_state;`)
	return err
}

// ClearAll clears summaries, messages, and sync state
func (s *Store) ClearAll(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ai_summaries;`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages_body;`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages_meta;`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_state;`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `VACUUM;`)
	return nil
}
