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
// store may be nil when no database is available; CRUD methods return a clear
// error in that case. labels and filters may be nil (no Gmail client / no label
// service): CRUD and Partition still work; SyncRule/UnsyncRule fail with a
// clear error.
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
	if s.store == nil {
		return nil, fmt.Errorf("deterministic rules store not available")
	}
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
	if s.store == nil {
		return fmt.Errorf("deterministic rules store not available")
	}
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
	if s.store == nil {
		return nil, fmt.Errorf("deterministic rules store not available")
	}
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
	if s.store == nil {
		return fmt.Errorf("deterministic rules store not available")
	}
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

// partitionPageSize is the number of IDs fetched per Gmail search page.
// partitionMaxPerRule caps how many IDs a single rule can claim so a
// catch-all query cannot sweep an entire mailbox in one call.
const (
	partitionPageSize   = 100
	partitionMaxPerRule = 500
)

// Partition runs every rule's query (creation order, first-match-wins) against
// the combined scopeQuery, assigns each message to the first matching rule, and
// returns the slice of per-rule matches plus any candidate IDs left unmatched.
// When candidates is nil the scope is treated as unbounded (no candidate filter)
// and remaining is also nil. Each rule's sweep is capped at 500 messages, so
// with large candidate sets, IDs beyond the cap may land in remaining (best-effort).
func (s *DeterministicRulesServiceImpl) Partition(ctx context.Context, scopeQuery string, candidates []string) ([]RuleMatch, []string, error) {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, nil, err
	}
	var candidateSet map[string]bool
	if candidates != nil {
		candidateSet = make(map[string]bool, len(candidates))
		for _, id := range candidates {
			candidateSet[id] = true
		}
	}
	seen := make(map[string]bool)
	var matches []RuleMatch
	for _, r := range rules {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		query := strings.TrimSpace(strings.TrimSpace(scopeQuery) + " " + r.Query)
		ids, err := s.searchAllIDs(ctx, query)
		if err != nil {
			return nil, nil, fmt.Errorf("rule %d (%q): %w", r.ID, r.Query, err)
		}
		var mine []string
		for _, id := range ids {
			if seen[id] {
				continue // first-match-wins: an earlier rule already owns this message
			}
			if candidateSet != nil && !candidateSet[id] {
				continue // outside candidate set
			}
			seen[id] = true
			mine = append(mine, id)
		}
		if len(mine) > 0 {
			matches = append(matches, RuleMatch{Rule: r, MessageIDs: mine})
		}
	}
	var remaining []string
	if candidates != nil {
		remaining = make([]string, 0, len(candidates))
		for _, id := range candidates {
			if !seen[id] {
				remaining = append(remaining, id)
			}
		}
	}
	return matches, remaining, nil
}

// searchAllIDs collects message IDs for the given query across all pages, capped
// at partitionMaxPerRule to prevent unbounded sweeps. The loop is also bounded to
// maxPages iterations so a misbehaving repository returning empty pages with tokens
// cannot loop forever.
func (s *DeterministicRulesServiceImpl) searchAllIDs(ctx context.Context, query string) ([]string, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("message repository not available")
	}
	maxPages := partitionMaxPerRule/partitionPageSize + 1
	var ids []string
	pageToken := ""
	for page := 0; page < maxPages; page++ {
		res, err := s.repo.SearchMessages(ctx, query, QueryOptions{MaxResults: partitionPageSize, PageToken: pageToken})
		if err != nil {
			return nil, err
		}
		for _, m := range res.Messages {
			if m != nil {
				ids = append(ids, m.Id)
			}
		}
		if res.NextPageToken == "" || len(ids) >= partitionMaxPerRule {
			return ids, nil
		}
		pageToken = res.NextPageToken
	}
	return ids, nil
}

// SyncRule / UnsyncRule are implemented in Task 5.
func (s *DeterministicRulesServiceImpl) SyncRule(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}
func (s *DeterministicRulesServiceImpl) UnsyncRule(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

var _ DeterministicRulesService = (*DeterministicRulesServiceImpl)(nil)
