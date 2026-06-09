package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AnalyzerRule is one free-text, LLM-interpreted preference rule for the inbox analyzer.
type AnalyzerRule struct {
	ID           int64  `json:"id"`
	AccountEmail string `json:"account_email"`
	RuleText     string `json:"rule_text"`
	CreatedAt    int64  `json:"created_at"`
}

// AnalyzerRulesStore handles persistence of analyzer preference rules.
type AnalyzerRulesStore struct {
	db *sql.DB
}

// NewAnalyzerRulesStore creates a new analyzer rules store.
func NewAnalyzerRulesStore(store *Store) *AnalyzerRulesStore {
	return &AnalyzerRulesStore{db: store.DB()}
}

// SaveRule inserts a new rule for the account and returns it.
func (s *AnalyzerRulesStore) SaveRule(ctx context.Context, accountEmail, ruleText string) (*AnalyzerRule, error) {
	if strings.TrimSpace(accountEmail) == "" || strings.TrimSpace(ruleText) == "" {
		return nil, fmt.Errorf("account_email and rule_text cannot be empty")
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO analyzer_rules (account_email, rule_text, created_at)
		VALUES (?, ?, ?)`,
		accountEmail, strings.TrimSpace(ruleText), now)
	if err != nil {
		return nil, fmt.Errorf("failed to save rule: %w", err)
	}
	id, _ := res.LastInsertId()
	return &AnalyzerRule{ID: id, AccountEmail: accountEmail, RuleText: strings.TrimSpace(ruleText), CreatedAt: now}, nil
}

// ListRules returns all rules for an account, newest first.
func (s *AnalyzerRulesStore) ListRules(ctx context.Context, accountEmail string) ([]*AnalyzerRule, error) {
	if strings.TrimSpace(accountEmail) == "" {
		return nil, fmt.Errorf("account_email cannot be empty")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_email, rule_text, created_at
		FROM analyzer_rules
		WHERE account_email = ?
		ORDER BY created_at DESC, id DESC`, accountEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []*AnalyzerRule
	for rows.Next() {
		r := &AnalyzerRule{}
		if err := rows.Scan(&r.ID, &r.AccountEmail, &r.RuleText, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return out, nil
}

// DeleteRule removes a rule by id for the account.
func (s *AnalyzerRulesStore) DeleteRule(ctx context.Context, accountEmail string, id int64) error {
	if strings.TrimSpace(accountEmail) == "" || id <= 0 {
		return fmt.Errorf("account_email cannot be empty and id must be positive")
	}
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM analyzer_rules WHERE account_email = ? AND id = ?`, accountEmail, id)
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
