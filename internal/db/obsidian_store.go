package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ajramos/giztui/internal/obsidian"
)

// ObsidianStore handles Obsidian forward history operations
type ObsidianStore struct {
	db *sql.DB
}

// NewObsidianStore creates a new Obsidian store from a base store
func NewObsidianStore(store *Store) *ObsidianStore {
	if store == nil {
		return nil
	}
	return &ObsidianStore{db: store.DB()}
}

// RecordForward saves a record of an email forwarded to Obsidian
func (os *ObsidianStore) RecordForward(ctx context.Context, record *obsidian.ObsidianForwardRecord) error {
	if os == nil || os.db == nil {
		return fmt.Errorf("obsidian store not initialized")
	}

	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `INSERT INTO obsidian_forward_history
	          (message_id, account_email, obsidian_path, template_used, forward_date, status, error_message, file_size, metadata)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = os.db.ExecContext(ctx, query,
		record.MessageID,
		record.AccountEmail,
		record.ObsidianPath,
		record.TemplateUsed,
		record.ForwardDate,
		record.Status,
		record.ErrorMessage,
		record.FileSize,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to record forward: %w", err)
	}

	return nil
}

// GetForwardHistory retrieves the forward history for a specific message
func (os *ObsidianStore) GetForwardHistory(ctx context.Context, messageID string) (*obsidian.ObsidianForwardRecord, error) {
	if os == nil || os.db == nil {
		return nil, fmt.Errorf("obsidian store not initialized")
	}

	query := `SELECT id, message_id, account_email, obsidian_path, template_used, forward_date, status, error_message, file_size, metadata
	          FROM obsidian_forward_history
	          WHERE message_id = ?
	          ORDER BY forward_date DESC
	          LIMIT 1`

	row := os.db.QueryRowContext(ctx, query, messageID)

	record := &obsidian.ObsidianForwardRecord{}
	var metadataJSON []byte
	var forwardDateStr string

	err := row.Scan(
		&record.ID,
		&record.MessageID,
		&record.AccountEmail,
		&record.ObsidianPath,
		&record.TemplateUsed,
		&forwardDateStr,
		&record.Status,
		&record.ErrorMessage,
		&record.FileSize,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("forward record not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan forward record: %w", err)
	}

	// Parse forward date
	if forwardDateStr != "" {
		record.ForwardDate, err = time.Parse("2006-01-02 15:04:05", forwardDateStr)
		if err != nil {
			// Try alternative format
			record.ForwardDate, err = time.Parse(time.RFC3339, forwardDateStr)
			if err != nil {
				record.ForwardDate = time.Now() // Fallback to current time
			}
		}
	}

	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		err = json.Unmarshal(metadataJSON, &record.Metadata)
		if err != nil {
			record.Metadata = make(map[string]interface{})
		}
	}

	return record, nil
}

// CheckIfAlreadyForwarded checks if a message has already been forwarded
func (os *ObsidianStore) CheckIfAlreadyForwarded(ctx context.Context, messageID, accountEmail string) (bool, error) {
	if os == nil || os.db == nil {
		return false, fmt.Errorf("obsidian store not initialized")
	}

	query := `SELECT COUNT(*) FROM obsidian_forward_history
	          WHERE message_id = ? AND account_email = ? AND status = 'success'`

	var count int
	err := os.db.QueryRowContext(ctx, query, messageID, accountEmail).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check forward status: %w", err)
	}

	return count > 0, nil
}

// ListRecentForwards retrieves recent forward records
func (os *ObsidianStore) ListRecentForwards(ctx context.Context, limit int) ([]*obsidian.ObsidianForwardRecord, error) {
	if os == nil || os.db == nil {
		return nil, fmt.Errorf("obsidian store not initialized")
	}

	if limit <= 0 {
		limit = 50 // Default limit
	}

	query := `SELECT id, message_id, account_email, obsidian_path, template_used, forward_date, status, error_message, file_size, metadata
	          FROM obsidian_forward_history
	          ORDER BY forward_date DESC
	          LIMIT ?`

	rows, err := os.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent forwards: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	var records []*obsidian.ObsidianForwardRecord
	for rows.Next() {
		record := &obsidian.ObsidianForwardRecord{}
		var metadataJSON []byte
		var forwardDateStr string

		err := rows.Scan(
			&record.ID,
			&record.MessageID,
			&record.AccountEmail,
			&record.ObsidianPath,
			&record.TemplateUsed,
			&forwardDateStr,
			&record.Status,
			&record.ErrorMessage,
			&record.FileSize,
			&metadataJSON,
		)
		if err != nil {
			continue // Skip malformed records
		}

		// Parse forward date
		if forwardDateStr != "" {
			record.ForwardDate, err = time.Parse("2006-01-02 15:04:05", forwardDateStr)
			if err != nil {
				record.ForwardDate, err = time.Parse(time.RFC3339, forwardDateStr)
				if err != nil {
					record.ForwardDate = time.Now()
				}
			}
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			err = json.Unmarshal(metadataJSON, &record.Metadata)
			if err != nil {
				record.Metadata = make(map[string]interface{})
			}
		}

		records = append(records, record)
	}

	return records, rows.Err()
}

// UpdateForwardStatus updates the status of a forward record
func (os *ObsidianStore) UpdateForwardStatus(ctx context.Context, id int, status, errorMessage string) error {
	if os == nil || os.db == nil {
		return fmt.Errorf("obsidian store not initialized")
	}

	query := `UPDATE obsidian_forward_history
	          SET status = ?, error_message = ?
	          WHERE id = ?`

	_, err := os.db.ExecContext(ctx, query, status, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update forward status: %w", err)
	}

	return nil
}

// GetForwardStats retrieves statistics about forwards
func (os *ObsidianStore) GetForwardStats(ctx context.Context, accountEmail string, days int) (map[string]interface{}, error) {
	if os == nil || os.db == nil {
		return nil, fmt.Errorf("obsidian store not initialized")
	}

	stats := make(map[string]interface{})

	// Total forwards
	var totalForwards int
	query := `SELECT COUNT(*) FROM obsidian_forward_history WHERE account_email = ?`
	err := os.db.QueryRowContext(ctx, query, accountEmail).Scan(&totalForwards)
	if err != nil {
		return nil, fmt.Errorf("failed to get total forwards: %w", err)
	}
	stats["total_forwards"] = totalForwards

	// Recent forwards (last N days)
	if days > 0 {
		var recentForwards int
		query = `SELECT COUNT(*) FROM obsidian_forward_history
		         WHERE account_email = ? AND forward_date >= datetime('now', '-? days')`
		err = os.db.QueryRowContext(ctx, query, accountEmail, days).Scan(&recentForwards)
		if err != nil {
			return nil, fmt.Errorf("failed to get recent forwards: %w", err)
		}
		stats["recent_forwards"] = recentForwards
	}

	// Success rate
	var successCount int
	query = `SELECT COUNT(*) FROM obsidian_forward_history
	         WHERE account_email = ? AND status = 'success'`
	err = os.db.QueryRowContext(ctx, query, accountEmail).Scan(&successCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get success count: %w", err)
	}

	successRate := 0.0
	if totalForwards > 0 {
		successRate = float64(successCount) / float64(totalForwards) * 100
	}
	stats["success_rate"] = successRate
	stats["success_count"] = successCount

	// Template usage
	query = `SELECT template_used, COUNT(*) as count
	         FROM obsidian_forward_history
	         WHERE account_email = ? AND template_used IS NOT NULL
	         GROUP BY template_used
	         ORDER BY count DESC`

	rows, err := os.db.QueryContext(ctx, query, accountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get template usage: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the operation
			_ = err
		}
	}()

	templateUsage := make(map[string]int)
	for rows.Next() {
		var template string
		var count int
		if err := rows.Scan(&template, &count); err != nil {
			continue
		}
		templateUsage[template] = count
	}
	stats["template_usage"] = templateUsage

	return stats, nil
}

// InitializeTable creates the obsidian_forward_history table if it doesn't exist
func (os *ObsidianStore) InitializeTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS obsidian_forward_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL,
			account_email TEXT NOT NULL,
			obsidian_path TEXT NOT NULL,
			template_used TEXT,
			forward_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			status TEXT DEFAULT 'success',
			error_message TEXT,
			file_size INTEGER,
			metadata TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_obsidian_history_message_id ON obsidian_forward_history(message_id);
		CREATE INDEX IF NOT EXISTS idx_obsidian_history_date ON obsidian_forward_history(forward_date);
		CREATE INDEX IF NOT EXISTS idx_obsidian_history_status ON obsidian_forward_history(status);
		CREATE INDEX IF NOT EXISTS idx_obsidian_history_account ON obsidian_forward_history(account_email);
	`

	_, err := os.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create obsidian_forward_history table: %w", err)
	}

	return nil
}
