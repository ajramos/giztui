package tui

import (
	"fmt"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

const actionPlanRulePage = "actionPlanRule"
const analyzerRulesPage = "analyzerRules"

// showRememberRuleModal opens an editable input pre-seeded with a suggested rule.
// Enter saves via AnalyzerRulesService; Esc cancels. Synchronous open/close.
func (a *App) showRememberRuleModal(suggestion string) {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	colors := a.GetComponentColors("ai")

	input := tview.NewInputField().
		SetLabel(" Rule: ").
		SetText(suggestion).
		SetFieldWidth(0)
	input.SetBackgroundColor(colors.Background.Color())
	input.SetFieldBackgroundColor(colors.Background.Color())
	input.SetFieldTextColor(colors.Text.Color())

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).
		SetTitle(" 🧠 Remember preference ").
		SetTitleColor(colors.Title.Color()).
		SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	box.AddItem(input, 1, 0, true)
	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Enter save · Esc cancel")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	prev := a.GetFocus()
	closeModal := func() {
		a.Pages.RemovePage(actionPlanRulePage)
		if prev != nil {
			a.SetFocus(prev)
		}
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := input.GetText()
			closeModal()
			go func() {
				if err := svc.SaveRule(a.ctx, text); err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
					return
				}
				a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved — applies on next analysis")
			}()
			return
		}
		closeModal() // Esc / Tab
	})

	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(box, 0, 6, true).
			AddItem(nil, 0, 1, false), 5, 0, true).
		AddItem(nil, 0, 1, false)

	a.Pages.AddPage(actionPlanRulePage, centered, true, true)
	a.SetFocus(input)
}

// openAnalyzerRulesManager lists existing rules with add ('a') / delete ('d') / Esc.
func (a *App) openAnalyzerRulesManager() {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	colors := a.GetComponentColors("ai")
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())

	var rules []services.AnalyzerRuleInfo
	reload := func() {
		list.Clear()
		rs, err := svc.ListRules(a.ctx)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", err))
			return
		}
		rules = rs
		if len(rs) == 0 {
			list.AddItem("(no rules yet — press 'a' to add)", "", 0, nil)
			return
		}
		for _, r := range rs {
			list.AddItem(r.RuleText, "", 0, nil)
		}
	}
	reload()

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).SetTitle(" 🧠 Analyzer rules ").
		SetTitleColor(colors.Title.Color()).SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	box.AddItem(list, 0, 1, true)
	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("a add · d delete · Esc close")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	prev := a.GetFocus()
	closeModal := func() {
		a.Pages.RemovePage(analyzerRulesPage)
		if prev != nil {
			a.SetFocus(prev)
		}
	}
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			closeModal()
			return nil
		case ev.Rune() == 'a':
			closeModal()
			a.showRememberRuleModal("")
			return nil
		case ev.Rune() == 'd':
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(rules) {
				id := rules[idx].ID
				go func() {
					if err := svc.DeleteRule(a.ctx, id); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Delete failed: %v", err))
						return
					}
					a.QueueUpdateDraw(reload)
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule deleted")
				}()
			}
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
	a.Pages.AddPage(analyzerRulesPage, centered, true, true)
	a.SetFocus(list)
}
