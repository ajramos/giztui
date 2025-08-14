package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/prompts"
)

// PromptStore handles prompt template and result operations
type PromptStore struct {
	db *sql.DB
}

// NewPromptStore creates a new prompt store from a base store
func NewPromptStore(store *Store) *PromptStore {
	if store == nil {
		return nil
	}
	return &PromptStore{db: store.DB()}
}

// ListPromptTemplates returns all prompt templates, optionally filtered by category
func (ps *PromptStore) ListPromptTemplates(ctx context.Context, category string) ([]*prompts.PromptTemplate, error) {
	if ps == nil || ps.db == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	query := `SELECT id, name, description, prompt_text, category, created_at, is_favorite, usage_count 
	          FROM prompt_templates`
	args := []interface{}{}

	if category != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}

	query += ` ORDER BY is_favorite DESC, usage_count DESC, name ASC`

	rows, err := ps.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*prompts.PromptTemplate
	for rows.Next() {
		t := &prompts.PromptTemplate{}
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
func (ps *PromptStore) GetPromptTemplate(ctx context.Context, id int) (*prompts.PromptTemplate, error) {
	if ps == nil || ps.db == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	t := &prompts.PromptTemplate{}
	err := ps.db.QueryRowContext(ctx,
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
func (ps *PromptStore) IncrementPromptUsage(ctx context.Context, promptID int) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	_, err := ps.db.ExecContext(ctx,
		`UPDATE prompt_templates SET usage_count = usage_count + 1 WHERE id = ?`, promptID)
	return err
}

// SavePromptResult saves a prompt execution result
func (ps *PromptStore) SavePromptResult(ctx context.Context, accountEmail, messageID string, promptID int, resultText string) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" || strings.TrimSpace(resultText) == "" {
		return fmt.Errorf("invalid prompt result inputs")
	}

	_, err := ps.db.ExecContext(ctx,
		`INSERT INTO prompt_results (account_email, message_id, prompt_id, result_text, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		accountEmail, messageID, promptID, resultText, time.Now().Unix())

	return err
}

// GetPromptResult retrieves a cached prompt result if available
func (ps *PromptStore) GetPromptResult(ctx context.Context, accountEmail, messageID string, promptID int) (*prompts.PromptResult, error) {
	if ps == nil || ps.db == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	result := &prompts.PromptResult{}
	err := ps.db.QueryRowContext(ctx,
		`SELECT id, account_email, message_id, prompt_id, result_text, created_at 
		 FROM prompt_results WHERE account_email = ? AND message_id = ? AND prompt_id = ? 
		 ORDER BY created_at DESC LIMIT 1`,
		accountEmail, messageID, promptID).
		Scan(&result.ID, &result.AccountEmail, &result.MessageID, &result.PromptID,
			&result.ResultText, &result.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // No cached result found
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}
