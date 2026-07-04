package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/db"
	gmailapi "google.golang.org/api/gmail/v1"
)

// GmailFilterAPI is the narrow Gmail settings surface the rules service needs for
// mirroring rules as server-side filters. *gmail.Client implements it (Task 4).
type GmailFilterAPI interface {
	CreateFilter(query string, action *gmailapi.FilterAction) (string, error)
	DeleteFilter(id string) error
}

// DeterministicRulesServiceImpl implements DeterministicRulesService.
// labels and filters may be nil (no Gmail client / no label service): CRUD and
// Partition still work; SyncRule/UnsyncRule fail with a clear error.
type DeterministicRulesServiceImpl struct {
	store   *db.DeterministicRulesStore
	repo    MessageRepository
	labels  LabelService
	filters GmailFilterAPI

	mu           sync.RWMutex
	accountEmail string
}

// NewDeterministicRulesService creates the deterministic rules service.
func NewDeterministicRulesService(store *db.DeterministicRulesStore, repo MessageRepository, labels LabelService, filters GmailFilterAPI) *DeterministicRulesServiceImpl {
	return &DeterministicRulesServiceImpl{store: store, repo: repo, labels: labels, filters: filters}
}

// SetAccountEmail scopes all operations to the given account.
func (s *DeterministicRulesServiceImpl) SetAccountEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountEmail = email
}

func (s *DeterministicRulesServiceImpl) account() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if strings.TrimSpace(s.accountEmail) == "" {
		return "", fmt.Errorf("no active account")
	}
	return s.accountEmail, nil
}

// validateRuleFields checks the action/label/prompt combination before touching Gmail.
func validateRuleFields(query, action, label string, promptID int64) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("query cannot be empty")
	}
	switch action {
	case "archive", "mark_read", "trash":
		return nil
	case "label":
		if strings.TrimSpace(label) == "" {
			return fmt.Errorf("label action needs a label name")
		}
		return nil
	case "prompt":
		if promptID <= 0 {
			return fmt.Errorf("prompt action needs a saved prompt")
		}
		return nil
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}

// validateQuery runs the query through Gmail search (MaxResults 1) so a syntax error
// surfaces at save time, not later during a sweep.
func (s *DeterministicRulesServiceImpl) validateQuery(ctx context.Context, query string) error {
	if s.repo == nil {
		return fmt.Errorf("message repository not available")
	}
	if _, err := s.repo.SearchMessages(ctx, query, QueryOptions{MaxResults: 1}); err != nil {
		return fmt.Errorf("Gmail rejected the query: %w", err)
	}
	return nil
}

func ruleInfoFromDB(r *db.DeterministicRule) DeterministicRuleInfo {
	return DeterministicRuleInfo{
		ID: r.ID, Query: r.Query, Action: r.Action, Label: r.Label,
		PromptID: r.PromptID, GmailFilterID: r.GmailFilterID, CreatedAt: r.CreatedAt,
	}
}

// SaveRule validates fields + Gmail query syntax, then persists the rule.
func (s *DeterministicRulesServiceImpl) SaveRule(ctx context.Context, query, action, label string, promptID int64) (*DeterministicRuleInfo, error) {
	acct, err := s.account()
	if err != nil {
		return nil, err
	}
	if err := validateRuleFields(query, action, label, promptID); err != nil {
		return nil, err
	}
	if err := s.validateQuery(ctx, query); err != nil {
		return nil, err
	}
	saved, err := s.store.SaveRule(ctx, acct, query, action, label, promptID)
	if err != nil {
		return nil, err
	}
	info := ruleInfoFromDB(saved)
	return &info, nil
}

// UpdateRule validates and rewrites an existing rule. Re-mirroring an already-synced
// rule is the caller's job (manager calls SyncRule afterwards).
func (s *DeterministicRulesServiceImpl) UpdateRule(ctx context.Context, id int64, query, action, label string, promptID int64) error {
	acct, err := s.account()
	if err != nil {
		return err
	}
	if err := validateRuleFields(query, action, label, promptID); err != nil {
		return err
	}
	if err := s.validateQuery(ctx, query); err != nil {
		return err
	}
	return s.store.UpdateRule(ctx, acct, id, query, action, label, promptID)
}

// ListRules returns the account's rules in creation (first-match-wins) order.
func (s *DeterministicRulesServiceImpl) ListRules(ctx context.Context) ([]DeterministicRuleInfo, error) {
	acct, err := s.account()
	if err != nil {
		return nil, err
	}
	rules, err := s.store.ListRules(ctx, acct)
	if err != nil {
		return nil, err
	}
	out := make([]DeterministicRuleInfo, 0, len(rules))
	for _, r := range rules {
		out = append(out, ruleInfoFromDB(r))
	}
	return out, nil
}

// DeleteRule removes the rule; if mirrored, it deletes the Gmail filter first (best
// effort — a failed remote delete still removes the local rule but returns the error).
func (s *DeterministicRulesServiceImpl) DeleteRule(ctx context.Context, id int64) error {
	acct, err := s.account()
	if err != nil {
		return err
	}
	rule, err := s.findRule(ctx, id)
	if err != nil {
		return err
	}
	var filterErr error
	if rule.GmailFilterID != "" && s.filters != nil {
		filterErr = s.filters.DeleteFilter(rule.GmailFilterID)
	}
	if err := s.store.DeleteRule(ctx, acct, id); err != nil {
		return err
	}
	if filterErr != nil {
		return fmt.Errorf("rule deleted locally, but the Gmail filter could not be removed: %w", filterErr)
	}
	return nil
}

func (s *DeterministicRulesServiceImpl) findRule(ctx context.Context, id int64) (*DeterministicRuleInfo, error) {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i], nil
		}
	}
	return nil, fmt.Errorf("rule not found")
}

// Partition is implemented in Task 3.
func (s *DeterministicRulesServiceImpl) Partition(ctx context.Context, scopeQuery string, candidates []string) ([]RuleMatch, []string, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

// SyncRule / UnsyncRule are implemented in Task 5.
func (s *DeterministicRulesServiceImpl) SyncRule(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}
func (s *DeterministicRulesServiceImpl) UnsyncRule(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

var _ DeterministicRulesService = (*DeterministicRulesServiceImpl)(nil)
