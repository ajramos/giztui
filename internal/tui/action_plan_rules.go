package tui

import (
	"fmt"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

const actionPlanRulePage = "actionPlanRule"
const analyzerRulesPage = "analyzerRules"

// showRememberRuleModal body-swaps the Action Plan tree for a pre-seeded input field inside
// the panel container (the showActionPlanMoveChooser pattern), so saving a preference reads as
// deeper navigation rather than a floating modal. Enter saves; Esc returns to the tree.
func (a *App) showRememberRuleModal(suggestion string) {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	state := a.actionPlanState
	if state == nil {
		return
	}
	colors := a.GetComponentColors("ai")

	input := tview.NewInputField().SetLabel(" Rule: ").SetText(suggestion).SetFieldWidth(0)
	input.SetBackgroundColor(colors.Background.Color())
	input.SetFieldBackgroundColor(colors.Background.Color())
	input.SetFieldTextColor(colors.Text.Color())

	restore := func() {
		state.container.RemoveItem(input)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return // Esc is handled by the input capture below
		}
		text := input.GetText()
		restore()
		go func() {
			if err := svc.SaveRule(a.ctx, text); err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
				return
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved — applies on next analysis")
		}()
	})
	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(input, 1, 0, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(" 🧠 Remember preference ")
	state.footer.SetText(" Enter save · Esc cancel ")
	a.currentFocus = "action_plan_rule"
	a.SetFocus(input)
}

// openAnalyzerRulesManager shows the analyzer rules as an in-place side-panel picker
// (mounted as a.labelsView in contentSplit, like the Action Plan). 'a' adds a rule via an
// embedded input (body-swap), 'd' deletes the highlighted rule, Esc closes. No floating modal.
func (a *App) openAnalyzerRulesManager() {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	// The rules picker and the Action Plan both occupy a.labelsView; close the plan first.
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}
	colors := a.GetComponentColors("ai")

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(colors.Background.Color())
	container.SetBorder(true)
	container.SetTitle(" 🧠 Analyzer rules ")
	container.SetTitleColor(colors.Title.Color())
	container.SetBorderColor(colors.Border.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	footer.SetText(" a add · d delete · Esc close ")

	var rules []services.AnalyzerRuleInfo
	reload := func() {
		list.Clear()
		rs, err := svc.ListRules(a.ctx)
		if err != nil {
			go a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", err))
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

	container.AddItem(list, 0, 1, true)
	container.AddItem(footer, 1, 0, false)

	closePicker := func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.ResizeItem(a.labelsView, 0, 0)
			}
		}
		a.setActivePicker(PickerNone)
		if l, ok := a.views["list"].(*tview.Table); ok {
			a.SetFocus(l)
		}
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	}

	// showAddInput body-swaps the list for an input field inside the same container.
	showAddInput := func() {
		input := tview.NewInputField().SetLabel(" New rule: ").SetFieldWidth(0)
		input.SetBackgroundColor(colors.Background.Color())
		input.SetFieldBackgroundColor(colors.Background.Color())
		input.SetFieldTextColor(colors.Text.Color())

		restore := func() {
			container.RemoveItem(input)
			container.RemoveItem(footer)
			container.AddItem(list, 0, 1, true)
			container.AddItem(footer, 1, 0, false)
			footer.SetText(" a add · d delete · Esc close ")
			a.currentFocus = "analyzer_rules"
			a.SetFocus(list)
		}
		input.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				text := input.GetText()
				restore()
				go func() {
					if err := svc.SaveRule(a.ctx, text); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
						return
					}
					a.QueueUpdateDraw(reload)
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved — applies on next analysis")
				}()
				return
			}
			restore() // Esc / Tab
		})

		container.RemoveItem(list)
		container.RemoveItem(footer)
		container.AddItem(input, 1, 0, true)
		container.AddItem(footer, 1, 0, false)
		footer.SetText(" Enter save · Esc cancel ")
		a.currentFocus = "analyzer_rules_add"
		a.SetFocus(input)
	}

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			closePicker()
			return nil
		case ev.Rune() == 'a':
			showAddInput()
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

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerAnalyzerRules)
	a.currentFocus = "analyzer_rules"
	a.SetFocus(list)
}
