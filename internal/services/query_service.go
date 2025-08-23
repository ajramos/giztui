package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/config"
	"github.com/ajramos/gmail-tui/internal/db"
)

// QueryServiceImpl implements QueryService
type QueryServiceImpl struct {
	store        *db.QueryStore
	accountEmail string
}

// NewQueryService creates a new query service
func NewQueryService(store *db.QueryStore, config *config.Config) *QueryServiceImpl {
	accountEmail := ""
	// Try to get account email from config or other sources
	// For now, we'll use empty string and it should be set by the app
	
	return &QueryServiceImpl{
		store:        store,
		accountEmail: accountEmail,
	}
}

// SetAccountEmail sets the account email for the service
func (s *QueryServiceImpl) SetAccountEmail(email string) {
	s.accountEmail = email
}

// GetAccountEmail returns the current account email
func (s *QueryServiceImpl) GetAccountEmail() string {
	return s.accountEmail
}

// SaveQuery saves a new query or updates an existing one
func (s *QueryServiceImpl) SaveQuery(ctx context.Context, name, query, description, category string) (*SavedQueryInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return nil, fmt.Errorf("account email not set")
	}

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("query name cannot be empty")
	}

	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Default category if not provided
	if strings.TrimSpace(category) == "" {
		category = "general"
	}

	savedQuery, err := s.store.SaveQuery(ctx, s.accountEmail, name, query, description, category)
	if err != nil {
		return nil, fmt.Errorf("failed to save query: %w", err)
	}

	return s.convertToSavedQueryInfo(savedQuery), nil
}

// GetQuery retrieves a saved query by name
func (s *QueryServiceImpl) GetQuery(ctx context.Context, name string) (*SavedQueryInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return nil, fmt.Errorf("account email not set")
	}

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("query name cannot be empty")
	}

	savedQuery, err := s.store.GetQueryByName(ctx, s.accountEmail, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}

	return s.convertToSavedQueryInfo(savedQuery), nil
}

// GetQueryByID retrieves a saved query by ID
func (s *QueryServiceImpl) GetQueryByID(ctx context.Context, id int64) (*SavedQueryInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return nil, fmt.Errorf("account email not set")
	}

	if id <= 0 {
		return nil, fmt.Errorf("invalid query ID")
	}

	savedQuery, err := s.store.GetQueryByID(ctx, s.accountEmail, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get query: %w", err)
	}

	return s.convertToSavedQueryInfo(savedQuery), nil
}

// ListQueries retrieves all saved queries, optionally filtered by category
func (s *QueryServiceImpl) ListQueries(ctx context.Context, category string) ([]*SavedQueryInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return nil, fmt.Errorf("account email not set")
	}

	savedQueries, err := s.store.ListQueries(ctx, s.accountEmail, category)
	if err != nil {
		return nil, fmt.Errorf("failed to list queries: %w", err)
	}

	result := make([]*SavedQueryInfo, len(savedQueries))
	for i, sq := range savedQueries {
		result[i] = s.convertToSavedQueryInfo(sq)
	}

	return result, nil
}

// SearchQueries searches for queries by name, description, or query content
func (s *QueryServiceImpl) SearchQueries(ctx context.Context, searchTerm string) ([]*SavedQueryInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return nil, fmt.Errorf("account email not set")
	}

	savedQueries, err := s.store.SearchQueries(ctx, s.accountEmail, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search queries: %w", err)
	}

	result := make([]*SavedQueryInfo, len(savedQueries))
	for i, sq := range savedQueries {
		result[i] = s.convertToSavedQueryInfo(sq)
	}

	return result, nil
}

// DeleteQuery removes a saved query by ID
func (s *QueryServiceImpl) DeleteQuery(ctx context.Context, id int64) error {
	if s.store == nil {
		return fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return fmt.Errorf("account email not set")
	}

	if id <= 0 {
		return fmt.Errorf("invalid query ID")
	}

	if err := s.store.DeleteQuery(ctx, s.accountEmail, id); err != nil {
		return fmt.Errorf("failed to delete query: %w", err)
	}

	return nil
}

// DeleteQueryByName removes a saved query by name
func (s *QueryServiceImpl) DeleteQueryByName(ctx context.Context, name string) error {
	if s.store == nil {
		return fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return fmt.Errorf("account email not set")
	}

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("query name cannot be empty")
	}

	if err := s.store.DeleteQueryByName(ctx, s.accountEmail, name); err != nil {
		return fmt.Errorf("failed to delete query: %w", err)
	}

	return nil
}

// RecordQueryUsage increments use count and updates last used timestamp
func (s *QueryServiceImpl) RecordQueryUsage(ctx context.Context, id int64) error {
	if s.store == nil {
		return fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return fmt.Errorf("account email not set")
	}

	if id <= 0 {
		return fmt.Errorf("invalid query ID")
	}

	if err := s.store.UpdateQueryUsage(ctx, s.accountEmail, id); err != nil {
		return fmt.Errorf("failed to record query usage: %w", err)
	}

	return nil
}

// GetCategories returns all unique categories for the account
func (s *QueryServiceImpl) GetCategories(ctx context.Context) ([]string, error) {
	if s.store == nil {
		return nil, fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return nil, fmt.Errorf("account email not set")
	}

	categories, err := s.store.GetCategories(ctx, s.accountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	return categories, nil
}

// UpdateQueryCategory updates the category of a saved query
func (s *QueryServiceImpl) UpdateQueryCategory(ctx context.Context, id int64, category string) error {
	if s.store == nil {
		return fmt.Errorf("query store not available")
	}

	if strings.TrimSpace(s.accountEmail) == "" {
		return fmt.Errorf("account email not set")
	}

	if id <= 0 {
		return fmt.Errorf("invalid query ID")
	}

	if strings.TrimSpace(category) == "" {
		category = "general"
	}

	// Get the existing query first
	savedQuery, err := s.store.GetQueryByID(ctx, s.accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to get query: %w", err)
	}

	// Update the category by saving the query again
	_, err = s.store.SaveQuery(ctx, s.accountEmail, savedQuery.Name, savedQuery.Query, savedQuery.Description, category)
	if err != nil {
		return fmt.Errorf("failed to update query category: %w", err)
	}

	return nil
}

// convertToSavedQueryInfo converts a db.SavedQuery to SavedQueryInfo
func (s *QueryServiceImpl) convertToSavedQueryInfo(sq *db.SavedQuery) *SavedQueryInfo {
	return &SavedQueryInfo{
		ID:          sq.ID,
		Name:        sq.Name,
		Query:       sq.Query,
		Description: sq.Description,
		Category:    sq.Category,
		UseCount:    sq.UseCount,
		LastUsed:    sq.LastUsed,
		CreatedAt:   sq.CreatedAt,
	}
}

// ValidateQueryName checks if a query name is valid and unique
func (s *QueryServiceImpl) ValidateQueryName(ctx context.Context, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("query name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("query name cannot exceed 100 characters")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, "\n\r\t") {
		return fmt.Errorf("query name cannot contain newlines or tabs")
	}

	return nil
}

// GenerateQueryName generates a default name for a query based on its content
func (s *QueryServiceImpl) GenerateQueryName(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return "Untitled Query"
	}

	// Take first meaningful part of the query
	words := strings.Fields(query)
	if len(words) == 0 {
		return "Untitled Query"
	}

	// Use first 3 words or less
	nameWords := words
	if len(nameWords) > 3 {
		nameWords = nameWords[:3]
	}

	name := strings.Join(nameWords, " ")
	
	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	// Limit length
	if len(name) > 50 {
		name = name[:47] + "..."
	}

	return name
}

// GetMostUsedQueries returns the most frequently used queries
func (s *QueryServiceImpl) GetMostUsedQueries(ctx context.Context, limit int) ([]*SavedQueryInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get all queries (already sorted by use_count DESC, last_used DESC)
	queries, err := s.ListQueries(ctx, "")
	if err != nil {
		return nil, err
	}

	// Return top N queries
	if len(queries) > limit {
		queries = queries[:limit]
	}

	return queries, nil
}

// GetRecentQueries returns the most recently used queries
func (s *QueryServiceImpl) GetRecentQueries(ctx context.Context, limit int) ([]*SavedQueryInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	// Get all queries, they're already sorted by last_used DESC
	queries, err := s.ListQueries(ctx, "")
	if err != nil {
		return nil, err
	}

	// Filter to only queries that have been used (use_count > 0)
	var recentQueries []*SavedQueryInfo
	for _, q := range queries {
		if q.UseCount > 0 {
			recentQueries = append(recentQueries, q)
		}
		if len(recentQueries) >= limit {
			break
		}
	}

	return recentQueries, nil
}