package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TelemetryStore handles local usage-analytics events. All data is local-only.
type TelemetryStore struct {
	db *sql.DB
}

// TelemetryEvent is a single recorded usage event.
type TelemetryEvent struct {
	TS           int64  // unix seconds
	AccountEmail string // may be empty
	Kind         string // e.g. "command", "shortcut", "error"
	Name         string // e.g. "archive", "a", "summarize"
	OK           bool
}

// NameCount is an aggregate row (a name and how many times it occurred).
type NameCount struct {
	Name  string
	Count int
}

// NewTelemetryStore creates a telemetry store from a base store.
func NewTelemetryStore(store *Store) *TelemetryStore {
	if store == nil {
		return nil
	}
	return &TelemetryStore{db: store.DB()}
}

// InsertEvents batch-inserts events in a single transaction.
func (ts *TelemetryStore) InsertEvents(ctx context.Context, events []TelemetryEvent) error {
	if ts == nil || ts.db == nil {
		return fmt.Errorf("telemetry store not initialized")
	}
	if len(events) == 0 {
		return nil
	}
	tx, err := ts.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO telemetry_events(ts, account_email, kind, name, ok) VALUES(?,?,?,?,?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, e := range events {
		okVal := 1
		if !e.OK {
			okVal = 0
		}
		if _, err := stmt.ExecContext(ctx, e.TS, e.AccountEmail, e.Kind, e.Name, okVal); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// TopByKind returns the most frequent event names of a kind since `since`
// (unix seconds), for an account (empty account_email = all accounts), limited
// to `limit` rows, ordered by count descending.
func (ts *TelemetryStore) TopByKind(ctx context.Context, accountEmail, kind string, since int64, limit int) ([]NameCount, error) {
	if ts == nil || ts.db == nil {
		return nil, fmt.Errorf("telemetry store not initialized")
	}
	if limit <= 0 {
		limit = 10
	}
	query := `SELECT name, COUNT(*) c FROM telemetry_events WHERE kind=? AND ts>=?`
	args := []any{kind, since}
	if strings.TrimSpace(accountEmail) != "" {
		query += ` AND account_email=?`
		args = append(args, accountEmail)
	}
	query += ` GROUP BY name ORDER BY c DESC, name ASC LIMIT ?`
	args = append(args, limit)

	rows, err := ts.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]NameCount, 0, limit)
	for rows.Next() {
		var nc NameCount
		if err := rows.Scan(&nc.Name, &nc.Count); err != nil {
			return nil, err
		}
		out = append(out, nc)
	}
	return out, rows.Err()
}

// Totals returns the total number of events and the number of error events
// since `since` (unix seconds) for an account (empty = all accounts).
func (ts *TelemetryStore) Totals(ctx context.Context, accountEmail string, since int64) (total int, errors int, err error) {
	if ts == nil || ts.db == nil {
		return 0, 0, fmt.Errorf("telemetry store not initialized")
	}
	query := `SELECT COUNT(*), COALESCE(SUM(CASE WHEN ok=0 THEN 1 ELSE 0 END),0) FROM telemetry_events WHERE ts>=?`
	args := []any{since}
	if strings.TrimSpace(accountEmail) != "" {
		query += ` AND account_email=?`
		args = append(args, accountEmail)
	}
	err = ts.db.QueryRowContext(ctx, query, args...).Scan(&total, &errors)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return total, errors, err
}

// Prune deletes events older than cutoff (unix seconds). Returns rows removed.
func (ts *TelemetryStore) Prune(ctx context.Context, cutoff int64) (int64, error) {
	if ts == nil || ts.db == nil {
		return 0, fmt.Errorf("telemetry store not initialized")
	}
	res, err := ts.db.ExecContext(ctx, `DELETE FROM telemetry_events WHERE ts < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// Clear removes all telemetry events for an account (empty = all accounts).
func (ts *TelemetryStore) Clear(ctx context.Context, accountEmail string) error {
	if ts == nil || ts.db == nil {
		return fmt.Errorf("telemetry store not initialized")
	}
	if strings.TrimSpace(accountEmail) == "" {
		_, err := ts.db.ExecContext(ctx, `DELETE FROM telemetry_events`)
		return err
	}
	_, err := ts.db.ExecContext(ctx, `DELETE FROM telemetry_events WHERE account_email=?`, accountEmail)
	return err
}
