package tui

import "github.com/ajramos/giztui/internal/services"

// dropMessagesFromPlan removes every id in ids from every category and from ReadManually,
// then prunes emptied categories. Mutates plan in place and reports whether anything changed.
// This is the removal half of applyActionPlanMove, reused so that acting on a plan email from
// the list/reader (trash/archive) keeps the open plan in sync.
func dropMessagesFromPlan(plan *services.ActionPlan, ids []string) bool {
	if plan == nil || len(ids) == 0 {
		return false
	}
	changed := false
	for _, id := range ids {
		for i := range plan.Categories {
			before := len(plan.Categories[i].MessageIDs)
			plan.Categories[i].MessageIDs = removeID(plan.Categories[i].MessageIDs, id)
			if len(plan.Categories[i].MessageIDs) != before {
				changed = true
			}
		}
		before := len(plan.ReadManually)
		plan.ReadManually = removeReadManuallyByID(plan.ReadManually, id)
		if len(plan.ReadManually) != before {
			changed = true
		}
	}
	if changed {
		plan.Categories = pruneEmptyCategories(plan.Categories)
	}
	return changed
}

// syncActionPlanRemovedIDs drops externally-removed message IDs (trashed/archived from the
// list or reader) from the open Action Plan and rebuilds its tree, so the plan never shows an
// email that no longer lives in the inbox. No-op when the plan panel isn't open. UI-thread
// only — every caller (the list-removal primitives) already runs inside QueueUpdateDraw.
func (a *App) syncActionPlanRemovedIDs(ids ...string) {
	if !a.isActionPlanActive() || a.actionPlanState == nil || a.actionPlanState.plan == nil {
		return
	}
	if dropMessagesFromPlan(a.actionPlanState.plan, ids) {
		a.rebuildActionPlanTree(a.actionPlanState)
	}
}
