package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DeterministicRule is one Gmail-query → action rule, applied first-match-wins in
// creation order. label is set when action == "label"; prompt_id when action == "prompt";
// gmail_filter_id is non-empty when the rule is mirrored as a server-side Gmail filter.
type DeterministicRule struct {
	ID            int64  `json:"id"`
	AccountEmail  string `json:"account_email"`
	Query         string `json:"query"`
	Action        string `json:"action"`
	Label         string `json:"label"`
	PromptID      int64  `json:"prompt_id"`
	GmailFilterID string `json:"gmail_filter_id"`
	CreatedAt     int64  `json:"created_at"`
}

// validRuleActions is the closed set of rule actions the store accepts.
var validRuleActions = map[string]bool{
	"archive": true, "mark_read": true, "trash": true, "label": true, "prompt": true,
}

// DeterministicRulesStore handles persistence of deterministic rules.
type DeterministicRulesStore struct {
	db *sql.DB
}

// NewDeterministicRulesStore creates a new deterministic rules store.
func NewDeterministicRulesStore(store *Store) *DeterministicRulesStore {
	return &DeterministicRulesStore{db: store.DB()}
}

// SaveRule inserts a new rule for the account and returns it.
func (s *DeterministicRulesStore) SaveRule(ctx context.Context, accountEmail, query, action, label string, promptID int64) (*DeterministicRule, error) {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("account_email and query cannot be empty")
	}
	if !validRuleActions[action] {
		return nil, fmt.Errorf("unknown action %q", action)
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO deterministic_rules (account_email, query, action, label, prompt_id, gmail_filter_id, created_at)
		VALUES (?, ?, ?, ?, ?, '', ?)`,
		accountEmail, strings.TrimSpace(query), action, strings.TrimSpace(label), promptID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to save rule: %w", err)
	}
	id, _ := res.LastInsertId()
	return &DeterministicRule{
		ID:           id,
		AccountEmail: accountEmail,
		Query:        strings.TrimSpace(query),
		Action:       action,
		Label:        strings.TrimSpace(label),
		PromptID:     promptID,
		CreatedAt:    now,
	}, nil
}

// ListRules returns all rules for an account in CREATION order (first-match-wins order).
func (s *DeterministicRulesStore) ListRules(ctx context.Context, accountEmail string) ([]*DeterministicRule, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_email, query, action, label, prompt_id, gmail_filter_id, created_at
		FROM deterministic_rules
		WHERE account_email = ?
		ORDER BY created_at ASC, id ASC`, accountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*DeterministicRule
	for rows.Next() {
		r := &DeterministicRule{}
		if err := rows.Scan(&r.ID, &r.AccountEmail, &r.Query, &r.Action, &r.Label, &r.PromptID, &r.GmailFilterID, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return out, nil
}

// UpdateRule rewrites the query/action/label/prompt of an existing rule.
func (s *DeterministicRulesStore) UpdateRule(ctx context.Context, accountEmail string, id int64, query, action, label string, promptID int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 || strings.TrimSpace(query) == "" {
		return fmt.Errorf("account_email, id and query are required")
	}
	if !validRuleActions[action] {
		return fmt.Errorf("unknown action %q", action)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE deterministic_rules SET query = ?, action = ?, label = ?, prompt_id = ?
		WHERE account_email = ? AND id = ?`,
		strings.TrimSpace(query), action, strings.TrimSpace(label), promptID, accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// SetGmailFilterID records (or clears, with "") the mirrored Gmail filter's ID.
func (s *DeterministicRulesStore) SetGmailFilterID(ctx context.Context, accountEmail string, id int64, filterID string) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE deterministic_rules SET gmail_filter_id = ?
		WHERE account_email = ? AND id = ?`, filterID, accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to set gmail filter id: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}

// DeleteRule removes a rule by id for the account.
func (s *DeterministicRulesStore) DeleteRule(ctx context.Context, accountEmail string, id int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM deterministic_rules WHERE account_email = ? AND id = ?`, accountEmail, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("rule not found")
	}
	return nil
}
