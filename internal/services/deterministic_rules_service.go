package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/db"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

// isNotFound reports whether err is a Gmail 404 — the remote filter is already gone,
// which the sync paths treat as "nothing to delete".
func isNotFound(err error) bool {
	var gerr *googleapi.Error
	return errors.As(err, &gerr) && gerr.Code == http.StatusNotFound
}

// GmailFilterAPI is the narrow Gmail settings surface the rules service needs for
// mirroring rules as server-side filters. *gmail.Client implements it (Task 4).
type GmailFilterAPI interface {
	CreateFilter(query string, action *gmailapi.FilterAction) (string, error)
	DeleteFilter(id string) error
	ListFilters() ([]*gmailapi.Filter, error)
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

// FallbackAccountEmail is the placeholder the TUI uses while the Gmail profile
// hasn't resolved yet. Rules saved under it are re-keyed to the real account as
// soon as it becomes known (AdoptOrphanRules).
const FallbackAccountEmail = "user@example.com"

// SetAccountEmail scopes all operations to the given account.
func (s *DeterministicRulesServiceImpl) SetAccountEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountEmail = email
}

// AdoptOrphanRules re-keys rules stored under FallbackAccountEmail to the
// current account. At startup the profile fetch may not have resolved when the
// service was initialized, so rules created in that window were saved under the
// placeholder and would otherwise vanish from every later session.
func (s *DeterministicRulesServiceImpl) AdoptOrphanRules(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	email, err := s.account()
	if err != nil || email == FallbackAccountEmail {
		return nil // no real account yet — nothing to adopt into
	}
	return s.store.AdoptOrphanRules(ctx, FallbackAccountEmail, email)
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
		return fmt.Errorf("query rejected by Gmail: %w", err)
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
		if err := s.filters.DeleteFilter(rule.GmailFilterID); err != nil && !isNotFound(err) {
			filterErr = err
		}
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

// FilterActionForRule maps a rule action to the Gmail filter action applied to future
// incoming mail. labelID must be a resolved Gmail label ID when action == "label".
// "prompt" (and anything unknown) cannot run server-side.
func FilterActionForRule(action, labelID string) (*gmailapi.FilterAction, error) {
	switch action {
	case "archive":
		return &gmailapi.FilterAction{RemoveLabelIds: []string{"INBOX"}}, nil
	case "mark_read":
		return &gmailapi.FilterAction{RemoveLabelIds: []string{"UNREAD"}}, nil
	case "trash":
		return &gmailapi.FilterAction{AddLabelIds: []string{"TRASH"}}, nil
	case "label":
		if strings.TrimSpace(labelID) == "" {
			return nil, fmt.Errorf("label rule needs a label")
		}
		return &gmailapi.FilterAction{AddLabelIds: []string{labelID}}, nil
	default:
		return nil, fmt.Errorf("a %q rule cannot run as a Gmail filter", action)
	}
}

// resolveLabelID finds (case-insensitively) or creates the Gmail label and returns its ID.
func (s *DeterministicRulesServiceImpl) resolveLabelID(ctx context.Context, name string) (string, error) {
	if s.labels == nil {
		return "", fmt.Errorf("label service not available")
	}
	existing, err := s.labels.ListLabels(ctx)
	if err != nil {
		return "", err
	}
	for _, l := range existing {
		if l != nil && strings.EqualFold(l.Name, name) {
			return l.Id, nil
		}
	}
	created, err := s.labels.CreateLabel(ctx, name)
	if err != nil {
		return "", err
	}
	if created == nil {
		return "", fmt.Errorf("label %q could not be created", name)
	}
	return created.Id, nil
}

// SyncRule mirrors the rule as a Gmail filter; if already mirrored, the old filter is
// deleted first (edit → recreate semantics).
func (s *DeterministicRulesServiceImpl) SyncRule(ctx context.Context, id int64) error {
	if s.store == nil {
		return fmt.Errorf("deterministic rules store not available")
	}
	acct, err := s.account()
	if err != nil {
		return err
	}
	if s.filters == nil {
		return fmt.Errorf("no Gmail client available")
	}
	rule, err := s.findRule(ctx, id)
	if err != nil {
		return err
	}
	labelID := ""
	if rule.Action == "label" {
		if labelID, err = s.resolveLabelID(ctx, rule.Label); err != nil {
			return err
		}
	}
	action, err := FilterActionForRule(rule.Action, labelID)
	if err != nil {
		return err
	}
	if rule.GmailFilterID != "" {
		if err := s.filters.DeleteFilter(rule.GmailFilterID); err != nil && !isNotFound(err) {
			return fmt.Errorf("could not replace the existing Gmail filter: %w", err)
		}
		// Clear the stored ID immediately so a failed create leaves consistent state.
		_ = s.store.SetGmailFilterID(ctx, acct, id, "")
	}
	filterID, err := s.filters.CreateFilter(rule.Query, action)
	if err != nil {
		return fmt.Errorf("filter rejected by Gmail: %w", err)
	}
	return s.store.SetGmailFilterID(ctx, acct, id, filterID)
}

// UnsyncRule deletes the mirrored Gmail filter and clears gmail_filter_id.
func (s *DeterministicRulesServiceImpl) UnsyncRule(ctx context.Context, id int64) error {
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
	if rule.GmailFilterID == "" {
		return nil // nothing mirrored — no-op
	}
	if s.filters == nil {
		return fmt.Errorf("no Gmail client available")
	}
	if err := s.filters.DeleteFilter(rule.GmailFilterID); err != nil && !isNotFound(err) {
		return fmt.Errorf("could not delete the Gmail filter: %w", err)
	}
	return s.store.SetGmailFilterID(ctx, acct, id, "")
}

var _ DeterministicRulesService = (*DeterministicRulesServiceImpl)(nil)
