package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CacheStore handles AI summary cache operations
type CacheStore struct {
	db *sql.DB
}

// NewCacheStore creates a new cache store from a base store
func NewCacheStore(store *Store) *CacheStore {
	if store == nil {
		return nil
	}
	return &CacheStore{db: store.DB()}
}

// SaveAISummary upserts a summary for (account_email, message_id)
func (cs *CacheStore) SaveAISummary(ctx context.Context, accountEmail, messageID, summary string, updatedAt int64) error {
	if cs == nil || cs.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(messageID) == "" || strings.TrimSpace(summary) == "" {
		return fmt.Errorf("invalid summary inputs")
	}
	_, err := cs.db.ExecContext(ctx, `INSERT INTO ai_summaries(account_email, message_id, summary, updated_at)
VALUES(?,?,?,?)
ON CONFLICT(account_email, message_id) DO UPDATE SET summary=excluded.summary, updated_at=excluded.updated_at;
`, accountEmail, messageID, summary, updatedAt)
	return err
}

// LoadAISummary returns a cached summary if present
func (cs *CacheStore) LoadAISummary(ctx context.Context, accountEmail, messageID string) (string, bool, error) {
	if cs == nil || cs.db == nil {
		return "", false, fmt.Errorf("cache store not initialized")
	}
	var out string
	err := cs.db.QueryRowContext(ctx, `SELECT summary FROM ai_summaries WHERE account_email=? AND message_id=?`, accountEmail, messageID).Scan(&out)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return out, true, nil
}

// DeleteAISummary removes a cached summary for (account_email, message_id)
func (cs *CacheStore) DeleteAISummary(ctx context.Context, accountEmail, messageID string) error {
	if cs == nil || cs.db == nil {
		return fmt.Errorf("cache store not initialized")
	}
	_, err := cs.db.ExecContext(ctx, `DELETE FROM ai_summaries WHERE account_email=? AND message_id=?`, accountEmail, messageID)
	return err
}
