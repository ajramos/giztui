package tui

import (
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// showActionPlanKeyHelp body-swaps the Action Plan tree for a read-only cheat-sheet of the
// panel's keys. Esc returns to the tree. Mirrors showActionPlanPromptView (the `v` viewer):
// synchronous event-loop UI work only — no goroutine / QueueUpdateDraw / ErrorHandler.
func (a *App) showActionPlanKeyHelp(state *actionPlanState) {
	if state == nil {
		return
	}
	fk := actionPlanFooterKeys{
		viewPrompt: a.Keys.ViewPrompt,
		remember:   a.Keys.RememberRule,
		move:       a.Keys.Move,
		skip:       a.Keys.BulkSelect,
		archive:    a.Keys.Archive,
		trash:      a.Keys.Trash,
		label:      a.Keys.ManageLabels,
		toggleRead: a.Keys.ToggleRead,
		confirm:    a.Keys.ConfirmPlan,
	}
	text := formatKeyHelp("Action Plan — keys", actionPlanKeyHints(fk))

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(false)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(tview.Escape(text))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.focus.set("action_plan")
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state)
	}
	view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev // arrows scroll the TextView
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(view, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(" ⌨️  Action Plan keys ")
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.focus.set("action_plan_key_help")
	a.SetFocus(view)
}

// actionPlanKeyHints builds the full ordered cheat-sheet for the Action Plan panel from the
// same actionPlanFooterKeys the footer uses, so footer teaser and cheat-sheet never drift.
// Fixed tview keys (arrows/Enter/Tab/Esc) are literals; configured keys use prettyKeyLabel.
func actionPlanKeyHints(keys actionPlanFooterKeys) []KeyHint {
	return []KeyHint{
		{Key: "↑/↓", Desc: "Move between nodes"},
		{Key: "Enter/→", Desc: "Expand category / open email"},
		{Key: "←", Desc: "Collapse category"},
		{Key: prettyKeyLabel(keys.skip), Desc: "Exclude / include email"},
		{Key: prettyKeyLabel(keys.archive), Desc: "Archive the category's checked emails"},
		{Key: prettyKeyLabel(keys.trash), Desc: "Trash the category's checked emails"},
		{Key: prettyKeyLabel(keys.label), Desc: "Apply the category's label"},
		{Key: prettyKeyLabel(keys.toggleRead), Desc: "Mark the category's checked emails read"},
		{Key: prettyKeyLabel(keys.move), Desc: "Move email / category to another label"},
		{Key: prettyKeyLabel(keys.viewPrompt), Desc: "View the effective analyzer prompt"},
		{Key: prettyKeyLabel(keys.remember), Desc: "Remember a rule / interest"},
		{Key: prettyKeyLabel(keys.confirm), Desc: "Confirm & apply the whole plan (two-press)"},
		{Key: "Tab", Desc: "Move focus to the inbox"},
		{Key: "Esc", Desc: "Close the panel"},
	}
}
