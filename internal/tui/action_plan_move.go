package tui

import (
	"fmt"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

const actionPlanMovePage = "actionPlanMove"

// moveTarget describes a destination chosen in the move picker: either a standard
// action ("archive"/"trash"/"mark_read"/"keep") or an existing category (by name).
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
}

// actionPlanMoveTargets builds the destination list for the move picker: the standard
// actions first, then the existing categories (excluding the source category by name).
func actionPlanMoveTargets(plan *services.ActionPlan, srcCatName string) []moveTarget {
	targets := []moveTarget{
		{label: "Archive", kind: "action", action: "archive"},
		{label: "Trash", kind: "action", action: "trash"},
		{label: "Mark read", kind: "action", action: "mark_read"},
		{label: "Keep (read manually)", kind: "action", action: "keep"},
	}
	for _, c := range plan.Categories {
		if c.Name == srcCatName {
			continue
		}
		targets = append(targets, moveTarget{
			label:   fmt.Sprintf("%s · %s", actionVerbLabel(c.Action), c.Name),
			kind:    "category",
			catName: c.Name,
		})
	}
	return targets
}

// openActionPlanMovePicker opens an overlay to recategorize the email msgID (currently in
// the category at srcCatIdx, or -1 for read-manually). Selecting a destination reassigns
// the message in the plan (no action runs yet) and re-renders the tree. Esc cancels.
func (a *App) openActionPlanMovePicker(state *actionPlanState, srcCatIdx int, msgID string) {
	colors := a.GetComponentColors("ai")

	srcCatName := ""
	if srcCatIdx >= 0 && srcCatIdx < len(state.plan.Categories) {
		srcCatName = state.plan.Categories[srcCatIdx].Name
	}
	targets := actionPlanMoveTargets(state.plan, srcCatName)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())
	for _, tg := range targets {
		list.AddItem(tg.label, "", 0, nil)
	}

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).SetTitle(" ➫ Move message to ").
		SetTitleColor(colors.Title.Color()).SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	box.AddItem(list, 0, 1, true)
	footer := tview.NewTextView().SetTextAlign(tview.AlignRight).SetText(" Enter to move  |  Esc to cancel ")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	prev := a.GetFocus()
	closeModal := func() {
		a.Pages.RemovePage(actionPlanMovePage)
		if prev != nil {
			a.SetFocus(prev)
		}
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx < 0 || idx >= len(targets) {
			closeModal()
			return
		}
		target := targets[idx]
		closeModal()
		if state.plan == nil || a.actionPlanState != state {
			return
		}
		applyActionPlanMove(state.plan, state.metaByID, msgID, target)
		delete(state.excluded, msgID) // a deliberately-placed message starts checked
		state.selectedMsgID = ""      // land the cursor on a category header after the move
		a.renderActionPlanPanel(state)
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Moved to %s — applies when you dispatch that group", target.label))
	})
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			closeModal()
			return nil
		}
		return ev
	})

	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(box, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)
	a.Pages.AddPage(actionPlanMovePage, centered, true, true)
	a.SetFocus(list)
}
