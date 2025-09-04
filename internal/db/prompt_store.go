package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/prompts"
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
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

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

// SaveBulkPromptResult saves a bulk prompt execution result
func (ps *PromptStore) SaveBulkPromptResult(ctx context.Context, accountEmail, cacheKey string, promptID int, messageCount int, messageIDs []string, resultText string) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(cacheKey) == "" || strings.TrimSpace(resultText) == "" {
		return fmt.Errorf("invalid bulk prompt result inputs")
	}

	messageIDsStr := strings.Join(messageIDs, ",")

	_, err := ps.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO bulk_prompt_results (account_email, cache_key, prompt_id, message_count, message_ids, result_text, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		accountEmail, cacheKey, promptID, messageCount, messageIDsStr, resultText, time.Now().Unix())

	return err
}

// GetBulkPromptResult retrieves a cached bulk prompt result if available
func (ps *PromptStore) GetBulkPromptResult(ctx context.Context, accountEmail, cacheKey string) (*prompts.BulkPromptResultDB, error) {
	if ps == nil || ps.db == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	result := &prompts.BulkPromptResultDB{}
	err := ps.db.QueryRowContext(ctx,
		`SELECT id, account_email, cache_key, prompt_id, message_count, message_ids, result_text, created_at
		 FROM bulk_prompt_results WHERE account_email = ? AND cache_key = ?
		 ORDER BY created_at DESC LIMIT 1`,
		accountEmail, cacheKey).
		Scan(&result.ID, &result.AccountEmail, &result.CacheKey, &result.PromptID,
			&result.MessageCount, &result.MessageIDs, &result.ResultText, &result.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil // No cached result found
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ClearPromptCache clears all prompt results for the given account
func (ps *PromptStore) ClearPromptCache(ctx context.Context, accountEmail string) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	if strings.TrimSpace(accountEmail) == "" {
		return fmt.Errorf("account email cannot be empty")
	}

	// Clear both single and bulk prompt results in a transaction
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	// Clear single prompt results
	_, err = tx.ExecContext(ctx, "DELETE FROM prompt_results WHERE account_email = ?", accountEmail)
	if err != nil {
		return fmt.Errorf("failed to clear single prompt results: %w", err)
	}

	// Clear bulk prompt results
	_, err = tx.ExecContext(ctx, "DELETE FROM bulk_prompt_results WHERE account_email = ?", accountEmail)
	if err != nil {
		return fmt.Errorf("failed to clear bulk prompt results: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ClearAllPromptCaches clears all prompt results for all accounts (admin function)
func (ps *PromptStore) ClearAllPromptCaches(ctx context.Context) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	// Clear both tables in a transaction
	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	// Clear all single prompt results
	_, err = tx.ExecContext(ctx, "DELETE FROM prompt_results")
	if err != nil {
		return fmt.Errorf("failed to clear all single prompt results: %w", err)
	}

	// Clear all bulk prompt results
	_, err = tx.ExecContext(ctx, "DELETE FROM bulk_prompt_results")
	if err != nil {
		return fmt.Errorf("failed to clear all bulk prompt results: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreatePromptTemplate creates a new prompt template
func (ps *PromptStore) CreatePromptTemplate(ctx context.Context, name, description, promptText, category string) (int, error) {
	if ps == nil || ps.db == nil {
		return 0, fmt.Errorf("prompt store not initialized")
	}

	if strings.TrimSpace(name) == "" || strings.TrimSpace(promptText) == "" || strings.TrimSpace(category) == "" {
		return 0, fmt.Errorf("name, prompt text, and category cannot be empty")
	}

	result, err := ps.db.ExecContext(ctx,
		`INSERT INTO prompt_templates (name, description, prompt_text, category, created_at, is_favorite, usage_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, description, promptText, category, time.Now().Unix(), false, 0)

	if err != nil {
		return 0, fmt.Errorf("failed to create prompt template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get created prompt ID: %w", err)
	}

	return int(id), nil
}

// UpdatePromptTemplate updates an existing prompt template
func (ps *PromptStore) UpdatePromptTemplate(ctx context.Context, id int, name, description, promptText, category string) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	if strings.TrimSpace(name) == "" || strings.TrimSpace(promptText) == "" || strings.TrimSpace(category) == "" {
		return fmt.Errorf("name, prompt text, and category cannot be empty")
	}

	result, err := ps.db.ExecContext(ctx,
		`UPDATE prompt_templates
		 SET name = ?, description = ?, prompt_text = ?, category = ?
		 WHERE id = ?`,
		name, description, promptText, category, id)

	if err != nil {
		return fmt.Errorf("failed to update prompt template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("prompt template with ID %d not found", id)
	}

	return nil
}

// DeletePromptTemplate deletes a prompt template
func (ps *PromptStore) DeletePromptTemplate(ctx context.Context, id int) error {
	if ps == nil || ps.db == nil {
		return fmt.Errorf("prompt store not initialized")
	}

	result, err := ps.db.ExecContext(ctx, `DELETE FROM prompt_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete prompt template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("prompt template with ID %d not found", id)
	}

	return nil
}

// FindPromptByName finds a prompt template by name
func (ps *PromptStore) FindPromptByName(ctx context.Context, name string) (*prompts.PromptTemplate, error) {
	if ps == nil || ps.db == nil {
		return nil, fmt.Errorf("prompt store not initialized")
	}

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("prompt name cannot be empty")
	}

	t := &prompts.PromptTemplate{}
	err := ps.db.QueryRowContext(ctx,
		`SELECT id, name, description, prompt_text, category, created_at, is_favorite, usage_count
		 FROM prompt_templates WHERE name = ?`, name).
		Scan(&t.ID, &t.Name, &t.Description, &t.PromptText, &t.Category,
			&t.CreatedAt, &t.IsFavorite, &t.UsageCount)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt template with name '%s' not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find prompt template: %w", err)
	}

	return t, nil
}
