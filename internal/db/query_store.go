package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SavedQuery represents a saved search query
type SavedQuery struct {
	ID           int64  `json:"id"`
	AccountEmail string `json:"account_email"`
	Name         string `json:"name"`
	Query        string `json:"query"`
	Description  string `json:"description"`
	CreatedAt    int64  `json:"created_at"`
	LastUsed     int64  `json:"last_used"`
	UseCount     int    `json:"use_count"`
	Category     string `json:"category"`
}

// QueryStore handles database operations for saved queries
type QueryStore struct {
	db *sql.DB
}

// NewQueryStore creates a new query store
func NewQueryStore(store *Store) *QueryStore {
	return &QueryStore{
		db: store.DB(),
	}
}

// SaveQuery saves a new query or updates an existing one
func (s *QueryStore) SaveQuery(ctx context.Context, accountEmail, name, query, description, category string) (*SavedQuery, error) {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("account_email, name, and query cannot be empty")
	}

	now := time.Now().Unix()

	// Try to insert new query
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO saved_queries (account_email, name, query, description, created_at, last_used, use_count, category)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?)
		ON CONFLICT(account_email, name) DO UPDATE SET
			query = excluded.query,
			description = excluded.description,
			last_used = excluded.last_used,
			category = excluded.category`,
		accountEmail, name, query, description, now, now, category)

	if err != nil {
		return nil, fmt.Errorf("failed to save query: %w", err)
	}

	// Get the saved query
	savedQuery, err := s.GetQueryByName(ctx, accountEmail, name)
	if err != nil {
		// If we can't get the query back, create a minimal response
		id, _ := result.LastInsertId()
		return &SavedQuery{
			ID:           id,
			AccountEmail: accountEmail,
			Name:         name,
			Query:        query,
			Description:  description,
			CreatedAt:    now,
			LastUsed:     now,
			UseCount:     0,
			Category:     category,
		}, nil
	}

	return savedQuery, nil
}

// GetQueryByName retrieves a saved query by name
func (s *QueryStore) GetQueryByName(ctx context.Context, accountEmail, name string) (*SavedQuery, error) {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("account_email and name cannot be empty")
	}

	query := &SavedQuery{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_email, name, query, description, created_at, last_used, use_count, category
		FROM saved_queries
		WHERE account_email = ? AND name = ?`,
		accountEmail, name).Scan(
		&query.ID, &query.AccountEmail, &query.Name, &query.Query,
		&query.Description, &query.CreatedAt, &query.LastUsed, &query.UseCount, &query.Category)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("query not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}

	return query, nil
}

// GetQueryByID retrieves a saved query by ID
func (s *QueryStore) GetQueryByID(ctx context.Context, accountEmail string, id int64) (*SavedQuery, error) {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return nil, fmt.Errorf("account_email cannot be empty and id must be positive")
	}

	query := &SavedQuery{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, account_email, name, query, description, created_at, last_used, use_count, category
		FROM saved_queries
		WHERE account_email = ? AND id = ?`,
		accountEmail, id).Scan(
		&query.ID, &query.AccountEmail, &query.Name, &query.Query,
		&query.Description, &query.CreatedAt, &query.LastUsed, &query.UseCount, &query.Category)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("query not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}

	return query, nil
}

// ListQueries retrieves all saved queries for an account, optionally filtered by category
func (s *QueryStore) ListQueries(ctx context.Context, accountEmail, category string) ([]*SavedQuery, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}

	var rows *sql.Rows
	var err error

	if strings.TrimSpace(category) == "" {
		// Get all queries
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, account_email, name, query, description, created_at, last_used, use_count, category
			FROM saved_queries
			WHERE account_email = ?
			ORDER BY last_used DESC, use_count DESC, name ASC`,
			accountEmail)
	} else {
		// Filter by category
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, account_email, name, query, description, created_at, last_used, use_count, category
			FROM saved_queries
			WHERE account_email = ? AND category = ?
			ORDER BY last_used DESC, use_count DESC, name ASC`,
			accountEmail, category)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list queries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	var queries []*SavedQuery
	for rows.Next() {
		query := &SavedQuery{}
		err := rows.Scan(&query.ID, &query.AccountEmail, &query.Name, &query.Query,
			&query.Description, &query.CreatedAt, &query.LastUsed, &query.UseCount, &query.Category)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query: %w", err)
		}
		queries = append(queries, query)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return queries, nil
}

// UpdateQueryUsage increments use count and updates last used timestamp
func (s *QueryStore) UpdateQueryUsage(ctx context.Context, accountEmail string, id int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE saved_queries
		SET use_count = use_count + 1, last_used = ?
		WHERE account_email = ? AND id = ?`,
		time.Now().Unix(), accountEmail, id)

	if err != nil {
		return fmt.Errorf("failed to update query usage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("query not found")
	}

	return nil
}

// DeleteQuery removes a saved query
func (s *QueryStore) DeleteQuery(ctx context.Context, accountEmail string, id int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM saved_queries
		WHERE account_email = ? AND id = ?`,
		accountEmail, id)

	if err != nil {
		return fmt.Errorf("failed to delete query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("query not found")
	}

	return nil
}

// DeleteQueryByName removes a saved query by name
func (s *QueryStore) DeleteQueryByName(ctx context.Context, accountEmail, name string) error {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(name) == "" {
		return fmt.Errorf("account_email and name cannot be empty")
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM saved_queries
		WHERE account_email = ? AND name = ?`,
		accountEmail, name)

	if err != nil {
		return fmt.Errorf("failed to delete query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("query not found")
	}

	return nil
}

// SearchQueries searches for queries by name or description
func (s *QueryStore) SearchQueries(ctx context.Context, accountEmail, searchTerm string) ([]*SavedQuery, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}

	searchPattern := "%" + strings.TrimSpace(searchTerm) + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_email, name, query, description, created_at, last_used, use_count, category
		FROM saved_queries
		WHERE account_email = ? AND (name LIKE ? OR description LIKE ? OR query LIKE ?)
		ORDER BY use_count DESC, last_used DESC, name ASC`,
		accountEmail, searchPattern, searchPattern, searchPattern)

	if err != nil {
		return nil, fmt.Errorf("failed to search queries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	var queries []*SavedQuery
	for rows.Next() {
		query := &SavedQuery{}
		err := rows.Scan(&query.ID, &query.AccountEmail, &query.Name, &query.Query,
			&query.Description, &query.CreatedAt, &query.LastUsed, &query.UseCount, &query.Category)
		if err != nil {
			return nil, fmt.Errorf("failed to scan query: %w", err)
		}
		queries = append(queries, query)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return queries, nil
}

// GetCategories returns all unique categories for an account
func (s *QueryStore) GetCategories(ctx context.Context, accountEmail string) ([]string, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT category
		FROM saved_queries
		WHERE account_email = ?
		ORDER BY category ASC`,
		accountEmail)

	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return categories, nil
}
