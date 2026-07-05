package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	gmailapi "google.golang.org/api/gmail/v1"
)

// deterministicRuleCategoryName renders a rule as an Action Plan category header:
// "⚡ <verb>: <query>" with the query capped at 40 runes.
func deterministicRuleCategoryName(r services.DeterministicRuleInfo) string {
	verb := actionVerbLabel(r.Action)
	if r.Action == "label" && strings.TrimSpace(r.Label) != "" {
		verb = "Label " + r.Label
	}
	q := strings.TrimSpace(r.Query)
	if rq := []rune(q); len(rq) > 40 {
		q = string(rq[:40]) + "…"
	}
	return fmt.Sprintf("⚡ %s: %s", verb, q)
}

// buildDeterministicPlan converts rule matches into an ActionPlan without any LLM
// involvement. Rules with no matched messages are dropped; categories are sorted
// with the same criterion as AI plans (action first, then name). Duplicate display
// names get a " (n)" suffix — the panel keys expansion/removal by category NAME, so
// two rules colliding (same action + same 40-rune query prefix) would otherwise be
// expanded/removed together, silently skipping the second rule's messages.
func buildDeterministicPlan(matches []services.RuleMatch) *services.ActionPlan {
	cats := make([]services.ActionPlanCategory, 0, len(matches))
	seen := make(map[string]int, len(matches))
	for _, m := range matches {
		if len(m.MessageIDs) == 0 {
			continue
		}
		name := deterministicRuleCategoryName(m.Rule)
		seen[name]++
		if n := seen[name]; n > 1 {
			name = fmt.Sprintf("%s (%d)", name, n)
		}
		cats = append(cats, services.ActionPlanCategory{
			Name:       name,
			Priority:   "medium",
			Action:     m.Rule.Action,
			Label:      m.Rule.Label,
			PromptID:   m.Rule.PromptID,
			MessageIDs: m.MessageIDs,
		})
	}
	services.SortCategories(cats)
	return &services.ActionPlan{Categories: cats}
}

// openDeterministicPlan opens the Action Plan panel populated purely by
// deterministic rules (":rules plan"). Runs on a background goroutine (network
// searches); mounting/rendering marshal onto the UI thread via the Task 9 helpers.
func (a *App) openDeterministicPlan() {
	svc := a.GetDeterministicRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}

	// Scope mirrors the AI Action Plan: the bulk selection if any, else the messages
	// currently loaded in the list — never the whole remote inbox. Partition intersects
	// each rule's search results with these candidates.
	a.mu.RLock()
	metaByID := make(map[string]*gmailapi.Message, len(a.messagesMeta))
	for _, m := range a.messagesMeta {
		if m != nil {
			metaByID[m.Id] = m
		}
	}
	selected := a.bulk.ids()
	a.mu.RUnlock()

	var candidates []string
	scopeNote := ""
	if len(selected) > 0 {
		candidates = selected
		scopeNote = fmt.Sprintf("%d selected", len(selected))
	} else {
		candidates = make([]string, 0, len(metaByID))
		for id := range metaByID {
			candidates = append(candidates, id)
		}
		scopeNote = fmt.Sprintf("%d loaded", len(candidates))
	}
	if len(candidates) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, "No messages to match — load or select messages first")
		return
	}

	a.GetErrorHandler().ShowProgress(a.ctx, "Matching messages against your rules…")
	matches, _, err := svc.Partition(a.ctx, "in:inbox", candidates)
	a.GetErrorHandler().ClearPersistentMessage()
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Rules search failed — check connection")
		return
	}
	plan := buildDeterministicPlan(matches)
	if len(plan.Categories) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("No rules matched (%s)", scopeNote))
		return
	}

	total := 0
	for _, c := range plan.Categories {
		total += len(c.MessageIDs)
	}

	scopeLabel := fmt.Sprintf("⚡ %d of %s by rules (no AI)", total, scopeNote)
	state := a.buildActionPlanPanelState("", scopeLabel, metaByID, false)
	state.plan = plan
	a.mountActionPlanPanel(state)
	a.QueueUpdateDraw(func() {
		if a.actionPlanState == state {
			a.renderActionPlanPanel(state)
		}
	})
}
