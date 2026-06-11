package tui

import (
	"fmt"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// showActionPlanPromptView body-swaps the Action Plan tree for a read-only view of the
// effective analyzer prompt (rules + base + literal {{messages}}), so the user can see exactly
// what is assembled before the per-batch payload is injected. Esc returns to the tree.
func (a *App) showActionPlanPromptView(state *actionPlanState) {
	svc := a.GetInboxAnalyzerService()
	if svc == nil || state == nil {
		return
	}

	var userRules []string
	if rsvc := a.GetAnalyzerRulesService(); rsvc != nil {
		if rs, err := rsvc.ListRules(a.ctx); err == nil {
			for _, r := range rs {
				userRules = append(userRules, r.RuleText)
			}
		}
	}
	opts := services.InboxAnalyzerOptions{
		CustomPromptText: state.customPromptText,
		UserRules:        userRules,
		BodyCharLimit:    a.Config.InboxAnalyzer.BodyCharLimit,
	}
	prompt := svc.BuildPromptPreview(opts)

	note := fmt.Sprintf("{{messages}} is replaced per batch with each email's subject / from / body (up to %d chars).", a.Config.InboxAnalyzer.BodyCharLimit)
	if !a.Config.InboxAnalyzer.IncludeBody {
		note = "{{messages}} is replaced per batch with each email's subject / from / snippet."
	}

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(true)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(tview.Escape(note + "\n\n" + prompt))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
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
	state.container.SetTitle(" 🔎 Effective analyzer prompt ")
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.currentFocus = "action_plan_prompt"
	a.SetFocus(view)
}
