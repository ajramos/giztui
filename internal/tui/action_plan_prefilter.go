package tui

import "github.com/ajramos/giztui/internal/services"

// applyPrefilterToMessages keeps only the messages whose IDs appear in remaining,
// preserving the original input order (not the order of remaining).
func applyPrefilterToMessages(messages []services.AnalyzerMessage, remaining []string) []services.AnalyzerMessage {
	keep := make(map[string]bool, len(remaining))
	for _, id := range remaining {
		keep[id] = true
	}
	out := make([]services.AnalyzerMessage, 0, len(remaining))
	for _, m := range messages {
		if keep[m.ID] {
			out = append(out, m)
		}
	}
	return out
}

// mergePreResolved returns a copy of p with the rule-resolved categories prepended.
// The input plan is never mutated (the streaming callback re-merges on every batch).
func mergePreResolved(p *services.ActionPlan, preResolved []services.ActionPlanCategory) *services.ActionPlan {
	if p == nil || len(preResolved) == 0 {
		return p
	}
	merged := *p
	merged.Categories = append(append([]services.ActionPlanCategory{}, preResolved...), p.Categories...)
	return &merged
}
