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
// with the same criterion as AI plans (action first, then name).
func buildDeterministicPlan(matches []services.RuleMatch) *services.ActionPlan {
	cats := make([]services.ActionPlanCategory, 0, len(matches))
	for _, m := range matches {
		if len(m.MessageIDs) == 0 {
			continue
		}
		cats = append(cats, services.ActionPlanCategory{
			Name:       deterministicRuleCategoryName(m.Rule),
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

	a.GetErrorHandler().ShowProgress(a.ctx, "Matching inbox against your rules…")
	matches, _, err := svc.Partition(a.ctx, "in:inbox", nil)
	a.GetErrorHandler().ClearPersistentMessage()
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Rules search failed — check connection")
		return
	}
	plan := buildDeterministicPlan(matches)
	if len(plan.Categories) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, "No rules matched any inbox messages")
		return
	}

	total := 0
	for _, c := range plan.Categories {
		total += len(c.MessageIDs)
	}

	// metaByID from the in-memory list; rule searches can surface messages beyond the
	// loaded page, so fetch metadata for the gap (subjects/senders in the tree).
	a.mu.RLock()
	metaByID := make(map[string]*gmailapi.Message, len(a.messagesMeta))
	for _, m := range a.messagesMeta {
		if m != nil {
			metaByID[m.Id] = m
		}
	}
	a.mu.RUnlock()
	var missing []string
	for _, c := range plan.Categories {
		for _, id := range c.MessageIDs {
			if _, ok := metaByID[id]; !ok {
				missing = append(missing, id)
			}
		}
	}
	if len(missing) > 0 && a.Client != nil {
		if fetched, ferr := a.Client.GetMessagesMetadataParallel(missing, 5); ferr == nil {
			for _, m := range fetched {
				if m != nil {
					metaByID[m.Id] = m
				}
			}
		}
	}

	scopeLabel := fmt.Sprintf("⚡ %d by rules (no AI)", total)
	state := a.buildActionPlanPanelState("", scopeLabel, metaByID, false)
	state.plan = plan
	a.mountActionPlanPanel(state)
	a.QueueUpdateDraw(func() {
		if a.actionPlanState == state {
			a.renderActionPlanPanel(state)
		}
	})
}
