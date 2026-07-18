package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

// ActionPlanEnabled reports whether the AI inbox action plan is available.
func (a *API) ActionPlanEnabled() bool { return a.analyzer != nil }

// AnalyzeInbox runs the AI inbox analyzer over the given messages and returns a
// categorized action plan. The frontend passes the message data it already has
// to avoid re-fetching bodies.
func (a *API) AnalyzeInbox(ctx context.Context, inputs []AnalyzerInput) (*ActionPlanResult, error) {
	if a.analyzer == nil {
		return nil, fmt.Errorf("the inbox action plan needs an LLM provider")
	}
	msgs := make([]services.AnalyzerMessage, 0, len(inputs))
	for _, in := range inputs {
		msgs = append(msgs, services.AnalyzerMessage{
			ID: in.ID, Subject: in.Subject, From: in.From, Snippet: in.Snippet,
		})
	}
	available := a.availableLabelNames(ctx)
	plan, err := a.analyzer.Analyze(ctx, msgs, services.InboxAnalyzerOptions{
		BatchSize:       50,
		MaxBatches:      5,
		AvailableLabels: available,
		UserRules:       a.userRuleTexts(ctx),
	}, nil)
	if err != nil {
		return nil, err
	}
	res := &ActionPlanResult{
		TotalAnalyzed: plan.TotalAnalyzed,
		ReadManually:  len(plan.ReadManually),
	}
	for _, c := range plan.Categories {
		res.Categories = append(res.Categories, PlanCategory{
			Name: c.Name, Priority: c.Priority, Description: c.Description,
			Action: c.Action, Label: c.Label, MessageIDs: c.MessageIDs,
		})
	}
	return res, nil
}

// availableLabelNames returns the user's non-system label names for the analyzer.
func (a *API) availableLabelNames(ctx context.Context) []string {
	var available []string
	if ls, err := a.labels.ListLabels(ctx); err == nil {
		for _, l := range ls {
			if l == nil {
				continue
			}
			if _, sys := systemLabels[l.Id]; sys || strings.HasPrefix(l.Id, "CATEGORY_") {
				continue
			}
			available = append(available, l.Name)
		}
	}
	return available
}

// userRuleTexts returns the stored analyzer preference rules as plain strings.
func (a *API) userRuleTexts(ctx context.Context) []string {
	if a.rules == nil {
		return nil
	}
	rs, err := a.rules.ListRules(ctx)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.RuleText)
	}
	return out
}

// AnalyzerRulesEnabled reports whether analyzer preference rules are available.
func (a *API) AnalyzerRulesEnabled() bool { return a.rules != nil }

// ListAnalyzerRules returns the stored analyzer preference rules.
func (a *API) ListAnalyzerRules(ctx context.Context) ([]AnalyzerRule, error) {
	if a.rules == nil {
		return []AnalyzerRule{}, nil
	}
	rs, err := a.rules.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AnalyzerRule, 0, len(rs))
	for _, r := range rs {
		out = append(out, AnalyzerRule{ID: r.ID, Text: r.RuleText})
	}
	return out, nil
}

// SaveAnalyzerRule persists a new free-text analyzer preference rule.
func (a *API) SaveAnalyzerRule(ctx context.Context, text string) error {
	if a.rules == nil {
		return fmt.Errorf("analyzer rules are not available")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("rule text is required")
	}
	return a.rules.SaveRule(ctx, text)
}

// DeleteAnalyzerRule removes a stored analyzer preference rule.
func (a *API) DeleteAnalyzerRule(ctx context.Context, id int64) error {
	if a.rules == nil {
		return fmt.Errorf("analyzer rules are not available")
	}
	return a.rules.DeleteRule(ctx, id)
}

// SuggestAnalyzerRule builds an editable default rule string from a sender and
// action (e.g. "Always archive emails from tldr.tech").
func (a *API) SuggestAnalyzerRule(from, action string, negate bool) string {
	if a.rules == nil {
		return ""
	}
	return a.rules.SuggestRuleFromContext(from, action, negate)
}

// ViewAnalyzerPrompt returns the effective analyzer prompt (rules block + base
// prompt) that Analyze would send, for inspection.
func (a *API) ViewAnalyzerPrompt(ctx context.Context) (string, error) {
	if a.analyzer == nil {
		return "", fmt.Errorf("the inbox action plan needs an LLM provider")
	}
	return a.analyzer.BuildPromptPreview(services.InboxAnalyzerOptions{
		AvailableLabels: a.availableLabelNames(ctx),
		UserRules:       a.userRuleTexts(ctx),
	}), nil
}

// BulkApplyLabelByName applies a label (by name, creating it if needed) to many
// messages — used when applying the "label" action of a plan category.
func (a *API) BulkApplyLabelByName(ctx context.Context, ids []string, name string) error {
	if len(ids) == 0 {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("label name is required")
	}
	var labelID string
	ls, err := a.labels.ListLabels(ctx)
	if err != nil {
		return err
	}
	for _, l := range ls {
		if l != nil && strings.EqualFold(l.Name, name) {
			labelID = l.Id
			break
		}
	}
	if labelID == "" {
		created, err := a.labels.CreateLabel(ctx, name)
		if err != nil {
			return err
		}
		labelID = created.Id
	}
	return a.labels.BulkApplyLabel(ctx, ids, labelID)
}
