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
// onProgress (may be nil) is called as each AI batch completes with (done,
// total) batch counts, so the UI can show real progress instead of a spinner.
func (a *API) AnalyzeInbox(ctx context.Context, inputs []AnalyzerInput, onProgress func(done, total int)) (*ActionPlanResult, error) {
	if a.analyzer == nil {
		return nil, fmt.Errorf("the inbox action plan needs an LLM provider")
	}
	msgs := make([]services.AnalyzerMessage, 0, len(inputs))
	for _, in := range inputs {
		msgs = append(msgs, services.AnalyzerMessage{
			ID: in.ID, Subject: in.Subject, From: in.From, Snippet: in.Snippet,
		})
	}

	res := &ActionPlanResult{}

	// First pass: resolve messages against the deterministic rules (the TUI's
	// DeterministicPrefilter). Matched messages become rule categories and are
	// removed from what the LLM sees.
	resolved := 0
	if a.detRules != nil {
		candidates := make([]string, len(msgs))
		for i := range msgs {
			candidates[i] = msgs[i].ID
		}
		matches, remaining, err := a.detRules.Partition(ctx, "in:inbox", candidates)
		if err == nil && len(matches) > 0 {
			for _, m := range matches {
				if len(m.MessageIDs) == 0 {
					continue
				}
				res.Categories = append(res.Categories, PlanCategory{
					Name:        deterministicRuleName(m.Rule),
					Priority:    "medium",
					Description: "Matched by rule: " + m.Rule.Query,
					Action:      m.Rule.Action,
					Label:       m.Rule.Label,
					MessageIDs:  m.MessageIDs,
					ByRule:      true,
					PromptID:    int(m.Rule.PromptID),
				})
			}
			keep := make(map[string]bool, len(remaining))
			for _, id := range remaining {
				keep[id] = true
			}
			filtered := msgs[:0]
			for _, m := range msgs {
				if keep[m.ID] {
					filtered = append(filtered, m)
				}
			}
			resolved = len(msgs) - len(filtered)
			msgs = filtered
		}
	}

	// Second pass: LLM analyzes whatever the rules didn't resolve.
	if len(msgs) > 0 {
		available := a.availableLabelNames(ctx)
		var cb func(*services.ActionPlan)
		if onProgress != nil {
			cb = func(p *services.ActionPlan) { onProgress(p.BatchesDone, p.BatchesTotal) }
		}
		// Batch/cap come from config (inbox_analyzer.*), same as the TUI; fall
		// back to the shared defaults when unset. Smaller batches yield several
		// blocks, which the bounded concurrency overlaps and the UI shows as real
		// "Batch N/M" progress.
		batchSize := a.analyzerBatchSize
		if batchSize <= 0 {
			batchSize = 50
		}
		maxBatches := a.analyzerMaxBatches
		if maxBatches <= 0 {
			maxBatches = 10
		}
		plan, err := a.analyzer.Analyze(ctx, msgs, services.InboxAnalyzerOptions{
			BatchSize:       batchSize,
			MaxBatches:      maxBatches,
			Concurrency:     4,
			AvailableLabels: available,
			StrictLabels:    a.analyzerStrictLabels,
			UserRules:       a.userRuleTexts(ctx),
		}, cb)
		if err != nil {
			return nil, err
		}
		res.TotalAnalyzed = plan.TotalAnalyzed + resolved
		res.ReadManually = len(plan.ReadManually)
		for _, c := range plan.Categories {
			res.Categories = append(res.Categories, PlanCategory{
				Name: c.Name, Priority: c.Priority, Description: c.Description,
				Action: c.Action, Label: c.Label, MessageIDs: c.MessageIDs,
			})
		}
		// Expose "read manually" as a navigable bucket (action "none"), like the
		// TUI — so you can expand it, peek/select its emails, and recategorize
		// them into a real category. Appended last so it sorts to the bottom.
		if len(plan.ReadManually) > 0 {
			ids := make([]string, len(plan.ReadManually))
			for i, m := range plan.ReadManually {
				ids[i] = m.ID
			}
			res.Categories = append(res.Categories, PlanCategory{
				Name:         "Read manually",
				Priority:     "low",
				Description:  "The AI left these for you to review",
				Action:       "none",
				MessageIDs:   ids,
				ReadManually: true,
			})
		}
	} else {
		res.TotalAnalyzed = resolved
	}
	return res, nil
}

// deterministicRuleName renders a rule as an action-plan category name, e.g.
// "Archive: from:github.com" (query capped), matching the TUI.
func deterministicRuleName(r services.DeterministicRuleInfo) string {
	verb := map[string]string{
		"archive":   "Archive",
		"trash":     "Trash",
		"mark_read": "Mark read",
		"label":     "Label",
		"prompt":    "Prompt",
	}[r.Action]
	if verb == "" {
		verb = r.Action
	}
	if r.Action == "label" && strings.TrimSpace(r.Label) != "" {
		verb = "Label " + r.Label
	}
	q := strings.TrimSpace(r.Query)
	if rq := []rune(q); len(rq) > 40 {
		q = string(rq[:40]) + "…"
	}
	return verb + ": " + q
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
