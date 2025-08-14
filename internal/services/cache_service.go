package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/db"
)

// CacheServiceImpl implements CacheService
type CacheServiceImpl struct {
	store *db.CacheStore
}

// NewCacheService creates a new cache service
func NewCacheService(store *db.CacheStore) *CacheServiceImpl {
	return &CacheServiceImpl{
		store: store,
	}
}

func (s *CacheServiceImpl) GetSummary(ctx context.Context, accountEmail, messageID string) (string, bool, error) {
	if s.store == nil {
		return "", false, fmt.Errorf("cache store not available")
	}

	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" {
		return "", false, fmt.Errorf("accountEmail and messageID cannot be empty")
	}

	summary, found, err := s.store.LoadAISummary(ctx, accountEmail, messageID)
	if err != nil {
		return "", false, fmt.Errorf("failed to load summary from cache: %w", err)
	}

	return summary, found, nil
}

func (s *CacheServiceImpl) SaveSummary(ctx context.Context, accountEmail, messageID, summary string) error {
	if s.store == nil {
		return fmt.Errorf("cache store not available")
	}

	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" || strings.TrimSpace(summary) == "" {
		return fmt.Errorf("accountEmail, messageID, and summary cannot be empty")
	}

	updatedAt := time.Now().Unix()

	if err := s.store.SaveAISummary(ctx, accountEmail, messageID, summary, updatedAt); err != nil {
		return fmt.Errorf("failed to save summary to cache: %w", err)
	}

	return nil
}

func (s *CacheServiceImpl) InvalidateSummary(ctx context.Context, accountEmail, messageID string) error {
	if s.store == nil {
		return fmt.Errorf("cache store not available")
	}

	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" {
		return fmt.Errorf("accountEmail and messageID cannot be empty")
	}

	if err := s.store.DeleteAISummary(ctx, accountEmail, messageID); err != nil {
		return fmt.Errorf("failed to invalidate summary: %w", err)
	}

	return nil
}

func (s *CacheServiceImpl) ClearCache(ctx context.Context, accountEmail string) error {
	if s.store == nil {
		return fmt.Errorf("cache store not available")
	}

	if strings.TrimSpace(accountEmail) == "" {
		return fmt.Errorf("accountEmail cannot be empty")
	}

	// Note: The current cache.Store interface doesn't have a method to clear all summaries
	// for an account. This would need to be added to the cache.Store interface.
	// For now, return a not implemented error.

	return fmt.Errorf("clear cache not implemented in current cache store")
}
