package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database used for local data storage
type Store struct {
	db *sql.DB
}

// Open opens (and creates/migrates) the database at the given path
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("empty database path")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o700); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	// Ensure file exists with strict perms
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create database file: %w", err)
		}
		f.Close()
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
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
	// user_version based migrations
	var ver int
	_ = s.db.QueryRowContext(ctx, "PRAGMA user_version;").Scan(&ver)

	// v1: ai_summaries table
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

	// v2: placeholder migration for existing v2 databases
	if ver == 1 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "PRAGMA user_version=2;")
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v2: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 2
	}

	// v3: prompt templates and results
	if ver == 2 {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS prompt_templates (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  name          TEXT NOT NULL UNIQUE,
  description   TEXT,
  prompt_text   TEXT NOT NULL,
  category      TEXT NOT NULL DEFAULT 'summary',
  created_at    INTEGER NOT NULL,
  is_favorite   BOOLEAN DEFAULT FALSE,
  usage_count   INTEGER DEFAULT 0
);
`)
		if err == nil {
			_, err = tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS prompt_results (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  account_email TEXT NOT NULL,
  message_id    TEXT NOT NULL,
  prompt_id     INTEGER NOT NULL,
  result_text   TEXT NOT NULL,
  created_at    INTEGER NOT NULL,
  FOREIGN KEY (prompt_id) REFERENCES prompt_templates(id)
);
`)
		}
		if err == nil {
			// Insert default prompts
			_, err = tx.ExecContext(ctx, `
INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite) VALUES
('Quick Summary', 'Brief 2-3 bullet point summary', 'Summarize this email in 2-3 bullet points:\n\n{{body}}', 'summary', ?, TRUE),
('Action Items', 'Extract specific action items and deadlines', 'Extract specific action items and deadlines from this email:\n\n{{body}}', 'analysis', ?, TRUE),
('Key Decisions', 'Identify key decisions or conclusions', 'What key decisions or conclusions are mentioned in this email?\n\n{{body}}', 'analysis', ?, FALSE),
('Meeting Summary', 'Summarize meeting details', 'Summarize the meeting details, attendees, and key points from this email:\n\n{{body}}', 'summary', ?, FALSE);
`, time.Now().Unix(), time.Now().Unix(), time.Now().Unix(), time.Now().Unix())
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, "PRAGMA user_version=3;")
		}
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate v3: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		ver = 3
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

// DB returns the underlying sql.DB for use by domain stores
func (s *Store) DB() *sql.DB {
	return s.db
}
