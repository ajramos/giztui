package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/db"
)

// AnalyzerRulesServiceImpl implements AnalyzerRulesService.
type AnalyzerRulesServiceImpl struct {
	store        *db.AnalyzerRulesStore
	accountEmail string
	mu           sync.RWMutex
}

// NewAnalyzerRulesService creates a new analyzer rules service.
func NewAnalyzerRulesService(store *db.AnalyzerRulesStore) *AnalyzerRulesServiceImpl {
	return &AnalyzerRulesServiceImpl{store: store}
}

// SetAccountEmail sets the active account for scoping.
func (s *AnalyzerRulesServiceImpl) SetAccountEmail(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accountEmail = email
}

func (s *AnalyzerRulesServiceImpl) account() (string, error) {
	s.mu.RLock()
	email := s.accountEmail
	s.mu.RUnlock()
	if strings.TrimSpace(email) == "" {
		return "", fmt.Errorf("account email not set")
	}
	return email, nil
}

func (s *AnalyzerRulesServiceImpl) SaveRule(ctx context.Context, ruleText string) error {
	if s.store == nil {
		return fmt.Errorf("analyzer rules store not available")
	}
	if strings.TrimSpace(ruleText) == "" {
		return fmt.Errorf("rule text cannot be empty")
	}
	email, err := s.account()
	if err != nil {
		return err
	}
	_, err = s.store.SaveRule(ctx, email, ruleText)
	return err
}

func (s *AnalyzerRulesServiceImpl) ListRules(ctx context.Context) ([]AnalyzerRuleInfo, error) {
	if s.store == nil {
		return nil, fmt.Errorf("analyzer rules store not available")
	}
	email, err := s.account()
	if err != nil {
		return nil, err
	}
	rows, err := s.store.ListRules(ctx, email)
	if err != nil {
		return nil, err
	}
	out := make([]AnalyzerRuleInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, AnalyzerRuleInfo{ID: r.ID, RuleText: r.RuleText, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func (s *AnalyzerRulesServiceImpl) DeleteRule(ctx context.Context, id int64) error {
	if s.store == nil {
		return fmt.Errorf("analyzer rules store not available")
	}
	email, err := s.account()
	if err != nil {
		return err
	}
	return s.store.DeleteRule(ctx, email, id)
}

// actionRuleVerb maps an action token to the verb used in a suggested rule.
func actionRuleVerb(action string) string {
	switch action {
	case "archive":
		return "archive"
	case "mark_read":
		return "mark as read"
	case "trash":
		return "trash"
	case "label":
		return "label"
	default:
		return "review"
	}
}

// senderDomain extracts the domain from a From header, falling back to the whole
// trimmed string if there is no parseable address.
func senderDomain(from string) string {
	f := strings.TrimSpace(from)
	if i := strings.LastIndex(f, "<"); i >= 0 {
		if j := strings.Index(f[i:], ">"); j >= 0 {
			f = strings.TrimSpace(f[i+1 : i+j])
		}
	}
	if at := strings.LastIndex(f, "@"); at >= 0 && at+1 < len(f) {
		return strings.ToLower(strings.TrimSpace(f[at+1:]))
	}
	return f
}

func (s *AnalyzerRulesServiceImpl) SuggestRuleFromContext(from, action string, negate bool) string {
	target := senderDomain(from)
	verb := actionRuleVerb(action)
	if negate {
		return fmt.Sprintf("Never %s emails from %s", verb, target)
	}
	return fmt.Sprintf("Always %s emails from %s", verb, target)
}
