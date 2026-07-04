package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// moveTarget describes a destination chosen in the move picker: either a standard
// action ("archive"/"trash"/"mark_read"/"keep"/"none") or an existing category (by name).
type moveTarget struct {
	label   string
	kind    string // "action" | "category"
	action  string // when kind=="action"
	catName string // when kind=="category"
}

// removeID returns ids without msgID, preserving order.
func removeID(ids []string, msgID string) []string {
	out := ids[:0:0]
	for _, id := range ids {
		if id != msgID {
			out = append(out, id)
		}
	}
	return out
}

// removeReadManuallyByID drops the message with the given ID from a ReadManually slice.
func removeReadManuallyByID(msgs []services.AnalyzerMessage, msgID string) []services.AnalyzerMessage {
	out := msgs[:0:0]
	for _, m := range msgs {
		if m.ID != msgID {
			out = append(out, m)
		}
	}
	return out
}

// categoryIndexByName returns the index of the first category with the given name, or -1.
func categoryIndexByName(plan *services.ActionPlan, name string) int {
	for i := range plan.Categories {
		if plan.Categories[i].Name == name {
			return i
		}
	}
	return -1
}

// firstCategoryWithAction returns the index of the first category with the given action, or -1.
func firstCategoryWithAction(plan *services.ActionPlan, action string) int {
	for i := range plan.Categories {
		if plan.Categories[i].Action == action {
			return i
		}
	}
	return -1
}

// pruneEmptyCategories drops categories left with no messages (e.g. after a move).
func pruneEmptyCategories(cats []services.ActionPlanCategory) []services.ActionPlanCategory {
	out := cats[:0:0]
	for _, c := range cats {
		if len(c.MessageIDs) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// analyzerMessageFor builds an AnalyzerMessage for a message ID from in-memory metadata.
func analyzerMessageFor(metaByID map[string]*gmailapi.Message, msgID string) services.AnalyzerMessage {
	am := services.AnalyzerMessage{ID: msgID}
	if m := metaByID[msgID]; m != nil {
		am.Subject = extractHeaderValue(m, "Subject")
		am.From = extractHeaderValue(m, "From")
		am.Snippet = m.Snippet
	}
	return am
}

// applyActionPlanMove reassigns msgID to the chosen target, mutating plan in place. It is
// robust to index shifts: the message is removed from wherever it currently lives (any
// category or the read-manually list) and re-added to the target, which is resolved by
// name/action rather than a possibly-stale index. Empty categories are pruned afterwards.
func applyActionPlanMove(plan *services.ActionPlan, metaByID map[string]*gmailapi.Message, msgID string, target moveTarget) {
	for i := range plan.Categories {
		plan.Categories[i].MessageIDs = removeID(plan.Categories[i].MessageIDs, msgID)
	}
	plan.ReadManually = removeReadManuallyByID(plan.ReadManually, msgID)

	switch target.kind {
	case "category":
		if idx := categoryIndexByName(plan, target.catName); idx >= 0 {
			plan.Categories[idx].MessageIDs = append(plan.Categories[idx].MessageIDs, msgID)
		}
	case "action":
		if target.action == "keep" {
			plan.ReadManually = append(plan.ReadManually, analyzerMessageFor(metaByID, msgID))
		} else if idx := firstCategoryWithAction(plan, target.action); idx >= 0 {
			plan.Categories[idx].MessageIDs = append(plan.Categories[idx].MessageIDs, msgID)
		} else {
			plan.Categories = append(plan.Categories, services.ActionPlanCategory{
				Name:       actionVerbLabel(target.action),
				Priority:   "medium",
				Action:     target.action,
				MessageIDs: []string{msgID},
			})
		}
	}
	plan.Categories = pruneEmptyCategories(plan.Categories)
	// A move can create a brand-new category (appended above) — re-sort so the tree
	// and the move chooser keep showing categories in display order (live feedback).
	services.SortCategories(plan.Categories)
}

// applyActionPlanBulkMove reassigns every message in the source group (a category by index,
// or ReadManually when srcCatIdx == -1) to target, returning the number moved. It loops the
// index-safe applyActionPlanMove over a pre-collected ID list, so mid-loop pruning/re-indexing
// is harmless (targets resolve by name/action). It does NOT touch any excluded map: msgIDs are
// unchanged, so the caller's state.excluded keys still apply to the moved emails.
func applyActionPlanBulkMove(plan *services.ActionPlan, metaByID map[string]*gmailapi.Message, srcCatIdx int, target moveTarget) int {
	var ids []string
	switch {
	case srcCatIdx == -1:
		for _, m := range plan.ReadManually {
			ids = append(ids, m.ID)
		}
	case srcCatIdx >= 0 && srcCatIdx < len(plan.Categories):
		ids = append(ids, plan.Categories[srcCatIdx].MessageIDs...)
	}
	for _, id := range ids {
		applyActionPlanMove(plan, metaByID, id, target)
	}
	return len(ids)
}

// actionPlanMoveTargets builds the destination list for the move picker: the standard
// actions and the existing categories (excluding the source category by name), as ONE
// list sorted by label — fixed actions and categories interleaved. Two separately-sorted
// blocks glued together read as unsorted (live feedback: "dos grupos distintos ordenados
// pero pegados"). "No action" keeps the email grouped in a themed category with nothing
// applied — distinct from the read-manually pile (live feedback: both destinations wanted).
func actionPlanMoveTargets(plan *services.ActionPlan, srcCatName string) []moveTarget {
	targets := []moveTarget{
		{label: "Archive", kind: "action", action: "archive"},
		{label: "Keep (read manually)", kind: "action", action: "keep"},
		{label: "Mark read", kind: "action", action: "mark_read"},
		{label: "No action", kind: "action", action: "none"},
		{label: "Trash", kind: "action", action: "trash"},
	}
	for _, c := range plan.Categories {
		if c.Name == srcCatName {
			continue
		}
		// Categories auto-created by a move are named after their action verb; the
		// "verb · name" format would read "Mark read · Mark read" (live feedback:
		// "cosas raras que se repiten") — render the label once in that case.
		label := c.Name
		if verb := actionVerbLabel(c.Action); verb != c.Name {
			label = fmt.Sprintf("%s · %s", verb, c.Name)
		}
		targets = append(targets, moveTarget{
			label:   label,
			kind:    "category",
			catName: c.Name,
		})
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return strings.ToLower(targets[i].label) < strings.ToLower(targets[j].label)
	})
	// Drop label duplicates: a verb-named category (e.g. "Mark read") and its fixed action
	// resolve to the same destination (firstCategoryWithAction), so showing both just reads
	// as a repeat. Stable sort keeps the fixed action first; it wins.
	deduped := targets[:0]
	for _, tg := range targets {
		if n := len(deduped); n > 0 && strings.EqualFold(deduped[n-1].label, tg.label) {
			continue
		}
		deduped = append(deduped, tg)
	}
	return deduped
}

// showActionPlanMoveChooser swaps the panel's tree for a destination chooser inside the SAME
// container (body-swap), so recategorizing reads as deeper navigation rather than a floating
// modal — and stays inside the panel's focus, avoiding the global-capture key swallow (see the
// currentFocus="action_plan_move" pass-through in keys.go). Shared by single-email and whole-
// group move: srcCatName is excluded from the destination list, title is shown while choosing,
// and onChosen runs the actual reassignment + feedback. Esc returns to the tree.
func (a *App) showActionPlanMoveChooser(state *actionPlanState, srcCatName, title string, onChosen func(target moveTarget)) {
	colors := a.GetComponentColors("ai")
	targets := actionPlanMoveTargets(state.plan, srcCatName)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())
	for i, tg := range targets {
		// Number the first 9 destinations so "m then a digit" picks one directly
		// (live feedback: no cursor navigation). tview renders the rune as "(1) …"
		// and pressing it fires the selected func; later targets keep Enter-only.
		var shortcut rune
		if i < 9 {
			shortcut = rune('1' + i)
		}
		list.AddItem(tg.label, "", shortcut, nil)
	}

	restore := func() {
		state.container.RemoveItem(list)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.focus.set("action_plan")
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx < 0 || idx >= len(targets) {
			restore()
			return
		}
		if state.plan == nil || a.actionPlanState != state {
			restore()
			return
		}
		onChosen(targets[idx])
		state.selectedMsgID = "" // land the cursor on a category header after the move
		restore()
	})
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(list, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(title)
	state.footer.SetText(" 1-9 or Enter to move  |  Esc to go back ")
	a.focus.set("action_plan_move")
	a.SetFocus(list)
}

// showActionPlanMoveInline opens the destination chooser for a SINGLE email. srcCatIdx is the
// email's current category (-1 for read-manually). Enter reassigns (the action runs at dispatch).
func (a *App) showActionPlanMoveInline(state *actionPlanState, srcCatIdx int, msgID string) {
	srcCatName := ""
	if srcCatIdx >= 0 && srcCatIdx < len(state.plan.Categories) {
		srcCatName = state.plan.Categories[srcCatIdx].Name
	}

	subj := "message"
	if m := state.metaByID[msgID]; m != nil {
		if s := extractHeaderValue(m, "Subject"); s != "" {
			subj = s
		}
	}
	if utf8.RuneCountInString(subj) > 40 {
		r := []rune(subj)
		subj = string(r[:40]) + "…"
	}

	a.showActionPlanMoveChooser(state, srcCatName, fmt.Sprintf(" ➫ Move %q to ", subj), func(target moveTarget) {
		applyActionPlanMove(state.plan, state.metaByID, msgID, target)
		delete(state.excluded, msgID) // a deliberately-placed message starts checked
		// ErrorHandler.ShowSuccess wraps QueueUpdateDraw; calling it synchronously here (on
		// the UI goroutine) would deadlock — dispatch it off-thread (links.go pattern).
		go a.GetErrorHandler().ShowSuccess(a.ctx,
			fmt.Sprintf("Moved to %s — applies when you dispatch that group", target.label))
	})
}

// showActionPlanBulkMoveInline opens the destination chooser for a WHOLE group: a category by
// index, or the read-manually pile when srcCatIdx == -1. Excluded flags are preserved across the
// move (applyActionPlanBulkMove leaves state.excluded untouched). Empty groups are a no-op.
func (a *App) showActionPlanBulkMoveInline(state *actionPlanState, srcCatIdx int) {
	srcName, count := "Read manually", 0
	if srcCatIdx == -1 {
		count = len(state.plan.ReadManually)
	} else if srcCatIdx >= 0 && srcCatIdx < len(state.plan.Categories) {
		srcName = state.plan.Categories[srcCatIdx].Name
		count = len(state.plan.Categories[srcCatIdx].MessageIDs)
	}
	if count == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "This group is empty — nothing to move")
		return
	}

	// Exclude a real category from its own destination list; read-manually (-1) excludes nothing.
	srcCatName := ""
	if srcCatIdx >= 0 {
		srcCatName = srcName
	}

	a.showActionPlanMoveChooser(state, srcCatName, fmt.Sprintf(" ➫ Move %q (%d) to ", srcName, count), func(target moveTarget) {
		n := applyActionPlanBulkMove(state.plan, state.metaByID, srcCatIdx, target)
		// Excluded flags are intentionally preserved (msgIDs unchanged).
		go a.GetErrorHandler().ShowSuccess(a.ctx,
			fmt.Sprintf("Moved %d emails to %s — applies when you dispatch that group", n, target.label))
	})
}
