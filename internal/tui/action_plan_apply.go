package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

// planApplyItem is one category's worth of whole-plan work: the suggested action
// applied to the category's non-excluded message IDs.
type planApplyItem struct {
	catName string
	action  string   // "archive" | "mark_read" | "trash" | "label"
	label   string   // label name when action == "label"
	ids     []string // non-excluded message IDs, plan order
}

// planApplySummary is a snapshot of everything the whole-plan apply will run.
type planApplySummary struct {
	items  []planApplyItem
	counts map[string]int // action → message count (all label categories aggregate under "label")
	total  int            // total messages across items
}

// buildPlanApply computes the applicable work for "apply the whole plan": every category
// with a bulk-appliable action, restricted to its non-excluded messages. Categories with
// action "none"/"summarize" (or anything unknown), label categories without a label name,
// and categories whose checked set is empty are skipped. Pure function — no App, no
// services — so it is unit-testable without the TUI harness.
func buildPlanApply(plan *services.ActionPlan, excluded map[string]bool) planApplySummary {
	s := planApplySummary{counts: map[string]int{}}
	if plan == nil {
		return s
	}
	for _, cat := range plan.Categories {
		switch cat.Action {
		case "archive", "mark_read", "trash":
			// bulk-appliable as-is
		case "label":
			if cat.Label == "" {
				continue // nothing to apply without a label name
			}
		default:
			continue // "none", "summarize", unknown → not bulk-appliable
		}
		ids := checkedIDs(cat.MessageIDs, excluded)
		if len(ids) == 0 {
			continue
		}
		s.items = append(s.items, planApplyItem{catName: cat.Name, action: cat.Action, label: cat.Label, ids: ids})
		s.counts[cat.Action] += len(ids)
		s.total += len(ids)
	}
	return s
}

// statusLine renders the confirmation prompt shown on the first press of the confirm key,
// e.g. "Apply plan: 12 archive, 3 trash, 5 label — press 'c' again to confirm, Esc cancels".
func (s planApplySummary) statusLine(confirmKey string) string {
	order := []string{"archive", "mark_read", "trash", "label"}
	names := map[string]string{"archive": "archive", "mark_read": "mark read", "trash": "trash", "label": "label"}
	parts := make([]string, 0, len(order))
	for _, act := range order {
		if n := s.counts[act]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, names[act]))
		}
	}
	return fmt.Sprintf("Apply plan: %s — press '%s' again to confirm, Esc cancels", strings.Join(parts, ", "), confirmKey)
}

// startActionPlanConfirm handles the FIRST press of the confirm-plan key (and :plan apply):
// compute the apply summary and arm the two-press confirmation. UI goroutine only.
func (a *App) startActionPlanConfirm(state *actionPlanState) {
	summary := buildPlanApply(state.plan, state.excluded)
	if summary.total == 0 {
		// go: Show* wrap QueueUpdateDraw; a synchronous call from the UI goroutine deadlocks.
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing to apply — all emails are excluded or categories have no action")
		return
	}
	state.confirmPending = true
	go a.GetErrorHandler().ShowPersistentMessage(a.ctx, summary.statusLine(a.Keys.ConfirmPlan), LogLevelInfo)
}

// executeActionPlanApply runs the whole plan: every applicable category, sequentially, in one
// worker goroutine. Failures are reported and skipped (the rest of the plan still runs);
// applied categories disappear from the tree as they complete, same as per-category apply.
// The summary snapshot is computed on the UI goroutine BEFORE the worker starts.
func (a *App) executeActionPlanApply(state *actionPlanState) {
	summary := buildPlanApply(state.plan, state.excluded)
	if summary.total == 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing to apply — all emails are excluded or categories have no action")
		return
	}
	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

	go func() {
		applied, appliedMsgs, failed := 0, 0, 0
		for i, item := range summary.items {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Applying plan (%d/%d): %s…", i+1, len(summary.items), item.catName))
			if err := a.runActionPlanBulkOp(emailService, labelService, item.action, item.ids, item.label); err != nil {
				failed++
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Action failed on %q: %v", item.catName, err))
				continue // a failing category (e.g. missing label in strict mode) must not abort the rest
			}
			applied++
			appliedMsgs += len(item.ids)
			catName := item.catName
			a.QueueUpdateDraw(func() {
				if a.actionPlanState == state { // panel may have been closed mid-run
					a.removeActionPlanCategory(state, catName)
				}
			})
		}
		a.GetErrorHandler().ClearPersistentMessage()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("✓ Plan applied: %d categories, %d messages", applied, appliedMsgs))
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Plan applied: %d categories, %d messages (%d failed)", applied, appliedMsgs, failed))
		}
	}()
}

// runActionPlanBulkOp dispatches one category's bulk operation. Shared by the per-category
// quick-action keys and the whole-plan apply so the two paths cannot drift. Must be called
// from a worker goroutine (bulkProgress and the services block).
func (a *App) runActionPlanBulkOp(emailService services.EmailService, labelService services.LabelService, action string, ids []string, label string) error {
	switch action {
	case "archive":
		return emailService.BulkArchive(a.ctx, ids, a.bulkProgress(a.ctx, "Archiving"))
	case "mark_read":
		return emailService.BulkMarkAsRead(a.ctx, ids, a.bulkProgress(a.ctx, "Marking read"))
	case "trash":
		return emailService.BulkTrash(a.ctx, ids, a.bulkProgress(a.ctx, "Trashing"))
	case "label":
		return a.applyActionPlanLabel(labelService, ids, label)
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}
