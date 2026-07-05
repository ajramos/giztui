package tui

import (
	"fmt"

	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// dispatchActionPlanPrompt runs the category's saved prompt (rule action "prompt")
// over the checked emails and shows the result in-place — the same body-swap
// pattern as dispatchActionPlanSummarize. Esc returns to the tree.
func (a *App) dispatchActionPlanPrompt(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil || cat.Action != "prompt" {
		return
	}
	if cat.PromptID == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "This rule has no prompt attached — edit it in :rules")
		return
	}
	ids := checkedIDs(cat.MessageIDs, state.excluded)
	if len(ids) == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "All emails in this category are excluded — nothing to run")
		return
	}

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(true)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(fmt.Sprintf("⏳ Running prompt on %d email(s)…", len(ids)))

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
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(view, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(fmt.Sprintf(" 🧠 Prompt on %q ", cat.Name))
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.focus.set("action_plan_prompt_run")
	a.SetFocus(view)

	promptID := int(cat.PromptID) // ApplyBulkPrompt takes int; rules store int64
	_, _, _, _, _, _, promptSvc, _, _, _, _, _ := a.GetServices()
	go func() {
		if promptSvc == nil {
			a.QueueUpdateDraw(func() {
				if a.actionPlanState == state {
					view.SetText("⚠️ Prompt service not available")
				}
			})
			return
		}
		res, err := promptSvc.ApplyBulkPrompt(a.ctx, a.getActiveAccountEmail(), ids, promptID, map[string]string{})
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state || a.ctx.Err() != nil {
				return
			}
			if err != nil {
				view.SetText(fmt.Sprintf("⚠️ Prompt failed: %v", err))
				return
			}
			view.SetText(a.renderPromptResult(res.Summary))
		})
	}()
}
