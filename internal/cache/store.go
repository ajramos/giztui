package cache

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

// Store wraps a SQLite database used for caching data locally
type Store struct {
	db *sql.DB
}

// Open opens (and creates/migrates) the cache database at the given path
func Open(ctx context.Context, dbPath string) (*Store, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, fmt.Errorf("empty cache path")
	}
	// Validate path to prevent directory traversal
	cleanPath := filepath.Clean(dbPath)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid cache path: contains directory traversal")
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
		_ = f.Close() // Error not actionable here - just creating empty file
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
	// user_version based migrations (v1 initializes ai_summaries, v2 adds prompt templates)
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

// PromptTemplate represents a prompt template
type PromptTemplate struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PromptText  string `json:"prompt_text"`
	Category    string `json:"category"`
	CreatedAt   int64  `json:"created_at"`
	IsFavorite  bool   `json:"is_favorite"`
	UsageCount  int    `json:"usage_count"`
}

// PromptResult represents a prompt execution result
type PromptResult struct {
	ID           int    `json:"id"`
	AccountEmail string `json:"account_email"`
	MessageID    string `json:"message_id"`
	PromptID     int    `json:"prompt_id"`
	ResultText   string `json:"result_text"`
	CreatedAt    int64  `json:"created_at"`
}

// ListPromptTemplates returns all prompt templates, optionally filtered by category
func (s *Store) ListPromptTemplates(ctx context.Context, category string) ([]*PromptTemplate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("cache store not initialized")
	}

	query := `SELECT id, name, description, prompt_text, category, created_at, is_favorite, usage_count
	          FROM prompt_templates`
	args := []interface{}{}

	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}

	query += ` ORDER BY is_favorite DESC, usage_count DESC, name ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }() // Error not actionable in defer

	var templates []*PromptTemplate
	for rows.Next() {
		t := &PromptTemplate{}
		err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.PromptText, &t.Category,
			&t.CreatedAt, &t.IsFavorite, &t.UsageCount)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	return templates, rows.Err()
}

// GetPromptTemplate returns a specific prompt template by ID
func (s *Store) GetPromptTemplate(ctx context.Context, id int) (*PromptTemplate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("cache store not initialized")
	}

	t := &PromptTemplate{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, prompt_text, category, created_at, is_favorite, usage_count
		 FROM prompt_templates WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Description, &t.PromptText, &t.Category,
			&t.CreatedAt, &t.IsFavorite, &t.UsageCount)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt template not found")
	}
	if err != nil {
		return nil, err
	}

	return t, nil
}

// IncrementPromptUsage increments the usage count for a prompt template
func (s *Store) IncrementPromptUsage(ctx context.Context, promptID int) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE prompt_templates SET usage_count = usage_count + 1 WHERE id = ?`, promptID)
	return err
}

// SavePromptResult saves a prompt execution result
func (s *Store) SavePromptResult(ctx context.Context, accountEmail, messageID string, promptID int, resultText string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("cache store not initialized")
	}

	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" || strings.TrimSpace(resultText) == "" {
		return fmt.Errorf("invalid prompt result inputs")
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO prompt_results (account_email, message_id, prompt_id, result_text, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		accountEmail, messageID, promptID, resultText, time.Now().Unix())

	return err
}
