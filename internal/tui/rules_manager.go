package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

const rulesManagerFooter = " a add · Enter edit · d delete · Esc close "
const rulesManagerTitle = " ⚡ Deterministic rules "

// ruleSyncOp decides what Gmail-filter operation a save requires.
// "sync" mirrors the rule, "unsync" removes a stale filter, "none" does nothing.
// Prompt rules can never be mirrored; if one previously had a filter (e.g. the
// action was edited from archive to prompt), the stale filter must be removed.
func ruleSyncOp(mirror bool, action string, hadFilter bool) string {
	switch {
	case action == "prompt":
		if hadFilter {
			return "unsync"
		}
		return "none"
	case mirror:
		return "sync"
	case hadFilter:
		return "unsync"
	default:
		return "none"
	}
}

// deterministicRuleListItem renders one rule for the manager list:
// "⚡ <verb>: <query>" plus " ☁" when the rule is mirrored as a Gmail filter.
func deterministicRuleListItem(r services.DeterministicRuleInfo, promptName string) string {
	verb := actionVerbLabel(r.Action)
	switch {
	case r.Action == "label" && strings.TrimSpace(r.Label) != "":
		verb = "Label " + r.Label
	case r.Action == "prompt" && promptName != "":
		verb = "Prompt '" + promptName + "'"
	}
	item := fmt.Sprintf("⚡ %s: %s", verb, r.Query)
	if r.GmailFilterID != "" {
		item += " ☁"
	}
	return item
}

// openRulesManager shows the deterministic rules as an in-place side-panel picker
// (the openAnalyzerRulesManager pattern). 'a' adds, Enter edits, 'd' deletes, Esc
// closes. Add/edit body-swap the list for a form inside the same container.
func (a *App) openRulesManager() {
	svc := a.GetDeterministicRulesService()
	if svc == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}
	colors := a.GetComponentColors("ai")

	// Prompt names for list display and the form dropdown ("" category = all).
	_, _, _, _, _, _, promptSvc, _, _, _, _, _ := a.GetServices()
	promptNameByID := map[int64]string{}
	promptNames := []string{"(none)"}
	promptIDs := []int64{0}
	if promptSvc != nil {
		if pts, err := promptSvc.ListPrompts(a.ctx, ""); err == nil {
			for _, p := range pts {
				promptNameByID[int64(p.ID)] = p.Name
				promptNames = append(promptNames, p.Name)
				promptIDs = append(promptIDs, int64(p.ID))
			}
		}
	}

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(colors.Background.Color())
	container.SetBorder(true)
	container.SetTitle(rulesManagerTitle)
	container.SetTitleColor(colors.Title.Color())
	container.SetBorderColor(colors.Border.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	footer.SetText(rulesManagerFooter)

	var rules []services.DeterministicRuleInfo
	reload := func() {
		list.Clear()
		rs, err := svc.ListRules(a.ctx)
		if err != nil {
			rules = nil
			list.AddItem("(failed to load rules)", "", 0, nil)
			go a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", err))
			return
		}
		rules = rs
		if len(rs) == 0 {
			list.AddItem("(no rules yet — press 'a' to add)", "", 0, nil)
			return
		}
		for _, r := range rs {
			list.AddItem(deterministicRuleListItem(r, promptNameByID[r.PromptID]), "", 0, nil)
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
		a.markFocus("list")
	}

	// showRuleForm body-swaps the list for an add/edit form. existing == nil → new rule.
	showRuleForm := func(existing *services.DeterministicRuleInfo) {
		actionsDisplay := []string{"Archive", "Mark read", "Trash", "Label", "Prompt"}
		actionTokens := []string{"archive", "mark_read", "trash", "label", "prompt"}

		// Seed values from the rule under edit (defaults for a new rule).
		queryText, labelText := "", ""
		actionIdx, promptIdx := 0, 0
		mirrored := false
		if existing != nil {
			queryText, labelText = existing.Query, existing.Label
			for i, tok := range actionTokens {
				if tok == existing.Action {
					actionIdx = i
					break
				}
			}
			for i, pid := range promptIDs {
				if pid != 0 && pid == existing.PromptID {
					promptIdx = i
					break
				}
			}
			mirrored = existing.GmailFilterID != ""
		}

		// Live form values, captured via changed-callbacks (no form-item lookups).
		action := actionTokens[actionIdx]
		promptID := promptIDs[promptIdx]
		mirror := mirrored

		form := tview.NewForm()
		form.SetBackgroundColor(colors.Background.Color())
		form.SetFieldBackgroundColor(colors.Background.Color())
		form.SetFieldTextColor(colors.Text.Color())
		form.SetLabelColor(colors.Title.Color())
		form.SetButtonBackgroundColor(colors.Background.Color())
		form.SetButtonTextColor(colors.Text.Color())
		form.AddInputField("Query", queryText, 0, nil, func(text string) { queryText = text })
		form.AddDropDown("Action", actionsDisplay, actionIdx, func(_ string, idx int) {
			if idx >= 0 && idx < len(actionTokens) {
				action = actionTokens[idx]
			}
		})
		form.AddInputField("Label (for Label action)", labelText, 0, nil, func(text string) { labelText = text })
		form.AddDropDown("Prompt (for Prompt action)", promptNames, promptIdx, func(_ string, idx int) {
			if idx >= 0 && idx < len(promptIDs) {
				promptID = promptIDs[idx]
			}
		})
		form.AddCheckbox("Also in Gmail", mirrored, func(_ string, checked bool) { mirror = checked })

		restore := func() {
			container.RemoveItem(form)
			container.RemoveItem(footer)
			container.AddItem(list, 0, 1, true)
			container.AddItem(footer, 1, 0, false)
			container.SetTitle(rulesManagerTitle)
			footer.SetText(rulesManagerFooter)
			a.focus.set("rules_manager")
			a.SetFocus(list)
		}

		save := func() {
			q := strings.TrimSpace(queryText)
			lbl := strings.TrimSpace(labelText)
			act := action
			pid := promptID
			if act != "prompt" {
				pid = 0
			}
			if q == "" {
				go a.GetErrorHandler().ShowWarning(a.ctx, "Query cannot be empty")
				return
			}
			if act == "label" && lbl == "" {
				go a.GetErrorHandler().ShowWarning(a.ctx, "Label action needs a label name")
				return
			}
			if act == "prompt" && pid == 0 {
				go a.GetErrorHandler().ShowWarning(a.ctx, "Prompt action needs a prompt — pick one in the Prompt dropdown")
				return
			}
			mir := mirror
			var existingID int64
			hadFilter := false
			if existing != nil {
				existingID = existing.ID
				hadFilter = existing.GmailFilterID != ""
			}
			restore()
			go func() {
				var (
					id  int64
					err error
				)
				if existing == nil {
					var saved *services.DeterministicRuleInfo
					saved, err = svc.SaveRule(a.ctx, q, act, lbl, pid)
					if saved != nil {
						id = saved.ID
					}
				} else {
					id = existingID
					err = svc.UpdateRule(a.ctx, id, q, act, lbl, pid)
				}
				if err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
					return
				}
				// Gmail mirroring. Prompt rules cannot exist as Gmail filters.
				warned := false
				if mir && act == "prompt" {
					a.GetErrorHandler().ShowWarning(a.ctx, "Prompt rules can't be mirrored to Gmail — saved locally only")
					warned = true
				}
				switch ruleSyncOp(mir, act, hadFilter) {
				case "sync":
					if serr := svc.SyncRule(a.ctx, id); serr != nil {
						a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Rule saved locally, but the Gmail mirror failed: %v", serr))
						warned = true
					}
				case "unsync":
					if serr := svc.UnsyncRule(a.ctx, id); serr != nil {
						a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Rule saved, but the Gmail filter could not be removed: %v", serr))
						warned = true
					}
				}
				a.QueueUpdateDraw(reload)
				if !warned {
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved")
				}
			}()
		}

		form.AddButton("Save", save)
		form.AddButton("Cancel", restore)
		form.SetCancelFunc(restore) // Esc anywhere in the form

		container.RemoveItem(list)
		container.RemoveItem(footer)
		container.AddItem(form, 0, 1, true)
		container.AddItem(footer, 1, 0, false)
		if existing == nil {
			container.SetTitle(" ⚡ New rule ")
		} else {
			container.SetTitle(" ⚡ Edit rule ")
		}
		footer.SetText(" Tab fields · Save button saves · Esc cancel ")
		a.focus.set("rules_manager_form")
		a.SetFocus(form)
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx >= 0 && idx < len(rules) {
			r := rules[idx]
			showRuleForm(&r)
		}
	})
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			closePicker()
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleAdd):
			showRuleForm(nil)
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleDelete):
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(rules) {
				id := rules[idx].ID
				go func() {
					// DeleteRule also removes the mirrored Gmail filter (service layer).
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
	a.setActivePicker(PickerRules)
	a.focus.set("rules_manager")
	a.SetFocus(list)
	// :rules runs during command execution; hideCommandBar()'s restoreFocusAfterModal()
	// would otherwise re-focus the message list afterward. "keep" leaves our focus alone.
	a.cmd.focusOverride = "keep"
}
