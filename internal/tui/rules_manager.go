package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

const rulesManagerFooter = " a add  |  Enter/e edit  |  t test  |  d delete  |  Esc close "
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
// "⚡ <verb>: <query>" plus " ☁️" when the rule is mirrored as a Gmail filter.
// U+FE0F (emoji presentation) matters: the bare text-presentation cloud renders
// as a tiny, near-invisible glyph on some terminals (reported on macOS).
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
		item += " ☁️"
	}
	return item
}

// gmailOnlyListItem renders a Gmail filter the rule model can't represent — shown
// read-only after the rules so the list stays a complete picture of the inbox.
func gmailOnlyListItem(f services.GmailOnlyFilter) string {
	return fmt.Sprintf("☁️ %s  (Gmail only)", f.Description)
}

// openRulesManager shows the deterministic rules as an in-place side-panel picker
// (the openAnalyzerRulesManager pattern). 'a' adds, Enter edits, 'd' deletes, Esc
// closes. Add/edit body-swap the list for a form inside the same container.
func (a *App) openRulesManager() {
	a.openRulesManagerOpts(false, "")
}

// openRulesManagerNewRule opens the manager directly on the New-rule form, with the
// Query pre-filled (create-a-rule-from-the-active-search flow; "" = blank form).
func (a *App) openRulesManagerNewRule(prefillQuery string) {
	a.openRulesManagerOpts(true, prefillQuery)
}

// activeSearchPrefill returns the query to seed a new rule with: the user-typed text
// of the active remote search, or "" when nothing (or only a local filter) is active.
// Local filters are not Gmail queries, so they can't become a rule verbatim.
func (a *App) activeSearchPrefill() string {
	if a.search.Mode() != "remote" {
		return ""
	}
	if q := a.search.Original(); q != "" {
		return q
	}
	return a.search.Query()
}

func (a *App) openRulesManagerOpts(openForm bool, prefillQuery string) {
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
	var gmailOnly []services.GmailOnlyFilter // filters not representable as rules (read-only rows)
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
		if len(rs) == 0 && len(gmailOnly) == 0 {
			list.AddItem("(no rules yet — press 'a' to add)", "", 0, nil)
			return
		}
		for _, r := range rs {
			list.AddItem(deterministicRuleListItem(r, promptNameByID[r.PromptID]), "", 0, nil)
		}
		for _, f := range gmailOnly {
			list.AddItem(gmailOnlyListItem(f), "", 0, nil)
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

	// Delete uses a two-press status-bar confirmation, the same shape used across the app
	// (Action Plan apply, bulk ops): the first RuleDelete press arms and shows a persistent
	// prompt in the status bar, a second press on the SAME rule deletes, Esc clears it (the
	// panel stays open — a later Esc closes it). The clear is go-wrapped because
	// ClearPersistentMessage runs through QueueUpdateDraw.
	deletePendingID := int64(0)
	deletePendingGmailID := "" // armed delete for a read-only Gmail-only filter row
	clearDeletePending := func() {
		if deletePendingID != 0 || deletePendingGmailID != "" {
			deletePendingID = 0
			deletePendingGmailID = ""
			go a.GetErrorHandler().ClearPersistentMessage()
		}
	}

	// showRuleForm body-swaps the list for an add/edit form. existing == nil → new rule.
	showRuleForm := func(existing *services.DeterministicRuleInfo) {
		actionsDisplay := []string{"Archive", "Mark read", "Trash", "Label", "Prompt"}
		actionTokens := []string{"archive", "mark_read", "trash", "label", "prompt"}

		// Seed values from the rule under edit (defaults for a new rule).
		queryText, labelText := "", ""
		actionIdx, promptIdx := 0, 0
		mirrored := false
		if existing == nil && prefillQuery != "" {
			queryText = prefillQuery
			prefillQuery = "" // one-shot: a later 'a' in the same session starts blank
		}
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

		// Snapshot the search state at form open so a Preview can be undone on close.
		prevSearchMode := a.search.Mode()
		prevSearchQuery := a.search.Original()
		if prevSearchQuery == "" {
			prevSearchQuery = a.search.Query()
		}
		previewed := false

		form := tview.NewForm()
		form.SetBackgroundColor(colors.Background.Color())
		form.SetFieldBackgroundColor(colors.Background.Color())
		form.SetFieldTextColor(colors.Text.Color())
		form.SetLabelColor(colors.Title.Color())
		form.SetButtonBackgroundColor(colors.Background.Color())
		form.SetButtonTextColor(colors.Text.Color())

		// styleDropDownOptions themes a dropdown's expanded option list — tview's
		// defaults (bright green) ignore the component theme.
		styleDropDownOptions := func(label string) {
			if dd, ok := form.GetFormItemByLabel(label).(*tview.DropDown); ok {
				dd.SetListStyles(colors.Text.Color(), colors.Background.Color(),
					colors.Background.Color(), colors.Accent.Color())
			}
		}

		form.AddInputField("Query", queryText, 0, nil, func(text string) { queryText = text })

		// rebuildForAction keeps only the fields that make sense for the selected
		// action: a Label input for "label", a Prompt dropdown for "prompt", and the
		// Gmail mirror checkbox for everything except "prompt" (prompt rules can't be
		// Gmail filters). Field values persist across rebuilds via the captured vars.
		rebuildForAction := func() {
			for form.GetFormItemCount() > 2 { // keep Query (0) and Action (1)
				form.RemoveFormItem(2)
			}
			switch action {
			case "label":
				form.AddInputField("Label", labelText, 0, nil, func(text string) { labelText = text })
			case "prompt":
				pIdx := 0
				for i, pid := range promptIDs {
					if pid != 0 && pid == promptID {
						pIdx = i
						break
					}
				}
				form.AddDropDown("Prompt", promptNames, pIdx, func(_ string, idx int) {
					if idx >= 0 && idx < len(promptIDs) {
						promptID = promptIDs[idx]
					}
				})
				styleDropDownOptions("Prompt")
			}
			if action != "prompt" {
				form.AddCheckbox("Also in Gmail", mirror, func(_ string, checked bool) { mirror = checked })
			}
		}

		form.AddDropDown("Action", actionsDisplay, actionIdx, func(_ string, idx int) {
			if idx < 0 || idx >= len(actionTokens) || actionTokens[idx] == action {
				return // no-op includes the initial-selection callback at build time
			}
			action = actionTokens[idx]
			rebuildForAction()
		})
		styleDropDownOptions("Action")
		rebuildForAction()

		restore := func() {
			container.RemoveItem(form)
			container.RemoveItem(footer)
			container.AddItem(list, 0, 1, true)
			container.AddItem(footer, 1, 0, false)
			container.SetTitle(rulesManagerTitle)
			footer.SetText(rulesManagerFooter)
			a.markFocus("rules_manager")
			a.SetFocus(list)
			// If a Preview replaced the message list, put back what was there before
			// the form opened: re-run the prior remote search, or reload the inbox.
			// (A prior LOCAL filter can't be replayed from here — inbox fallback.)
			if previewed {
				previewed = false
				if prevSearchMode == "remote" && prevSearchQuery != "" {
					go a.performSearch(prevSearchQuery)
				} else {
					a.search.clear()
					a.nextPageToken = ""
					go a.reloadMessages()
				}
			}
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
				// Gmail mirroring. Prompt rules cannot exist as Gmail filters (the form
				// hides the mirror checkbox for them; ruleSyncOp enforces it regardless).
				warned := false
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

		// Preview runs the Query as a search shown in the main message list (verbatim,
		// no hidden scoping) while the form stays open and keeps focus. Re-runnable
		// after editing the query; the list is restored when the form closes.
		form.AddButton("Preview", func() {
			q := strings.TrimSpace(queryText)
			if q == "" {
				go a.GetErrorHandler().ShowWarning(a.ctx, "Query cannot be empty")
				return
			}
			previewed = true
			go a.performSearchPreview(q)
		})
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
		footer.SetText(" Tab to move  |  Preview to test the query  |  Save to confirm  |  Esc to cancel ")
		a.markFocus("rules_manager_form")
		a.SetFocus(form)
	}

	// gmailOnlyRow reports whether list index idx is one of the read-only
	// Gmail-only rows appended after the editable rules.
	gmailOnlyRow := func(idx int) bool {
		return idx >= len(rules) && idx < len(rules)+len(gmailOnly)
	}
	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if gmailOnlyRow(idx) {
			go a.GetErrorHandler().ShowInfo(a.ctx, "This filter lives only in Gmail — manage it at gmail.com → Settings → Filters")
			return
		}
		if idx >= 0 && idx < len(rules) {
			clearDeletePending()
			r := rules[idx]
			showRuleForm(&r)
		}
	})
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			if deletePendingID != 0 || deletePendingGmailID != "" {
				clearDeletePending() // cancel the pending delete only; panel stays open
				return nil
			}
			closePicker()
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleAdd):
			clearDeletePending()
			showRuleForm(nil)
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleEdit):
			// Edit the highlighted rule — parity with Enter, and with the GUI's
			// e/Shift+E convention (KEYBOARD_SHORTCUTS.md picker CRUD table).
			clearDeletePending()
			idx := list.GetCurrentItem()
			if gmailOnlyRow(idx) {
				go a.GetErrorHandler().ShowInfo(a.ctx, "This filter lives only in Gmail — manage it at gmail.com → Settings → Filters")
				return nil
			}
			if idx < 0 || idx >= len(rules) {
				return nil
			}
			r := rules[idx]
			showRuleForm(&r)
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RulePreview):
			clearDeletePending()
			idx := list.GetCurrentItem()
			if gmailOnlyRow(idx) {
				go a.GetErrorHandler().ShowInfo(a.ctx, "Test applies to rules — this row is a server-side Gmail filter")
				return nil
			}
			if idx < 0 || idx >= len(rules) {
				return nil
			}
			id := rules[idx].ID
			go func() {
				a.GetErrorHandler().ShowProgress(a.ctx, "Testing rule against inbox…")
				pv, err := svc.PreviewRule(a.ctx, id)
				if err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Preview failed: %v", err))
					return
				}
				count := fmt.Sprintf("%d", pv.MatchCount)
				if pv.Capped {
					count = fmt.Sprintf("≥%d", pv.MatchCount)
				}
				msg := fmt.Sprintf("Rule matches %s inbox message(s)", count)
				if len(pv.Sample) > 0 {
					msg += ": " + strings.Join(pv.Sample, " · ")
				}
				a.GetErrorHandler().ShowInfo(a.ctx, msg)
			}()
			return nil
		case a.matchesConfiguredKey(ev, a.Keys.RuleDelete):
			idx := list.GetCurrentItem()
			if gmailOnlyRow(idx) {
				f := gmailOnly[idx-len(rules)]
				if deletePendingGmailID == f.ID { // second press → delete the raw Gmail filter
					deletePendingGmailID = ""
					go func() {
						a.GetErrorHandler().ClearPersistentMessage()
						if err := svc.DeleteGmailFilter(a.ctx, f.ID); err != nil {
							a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Delete filter failed: %v", err))
							return
						}
						// Re-import so the deleted filter drops from the read-only rows.
						res, ierr := svc.ImportGmailFilters(a.ctx)
						a.QueueUpdateDraw(func() {
							if !a.isRulesPickerActive() {
								return
							}
							if ierr == nil {
								gmailOnly = res.Unsupported
							}
							reload()
						})
						a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Gmail filter deleted")
					}()
					return nil
				}
				// First press → arm. These are server-side filters, not local rules.
				deletePendingID = 0
				deletePendingGmailID = f.ID
				go a.GetErrorHandler().ShowPersistentMessage(a.ctx, fmt.Sprintf("Delete this Gmail filter? It's server-side, not a local rule — press '%s' again to confirm, Esc cancels", a.Keys.RuleDelete), LogLevelInfo)
				return nil
			}
			if idx < 0 || idx >= len(rules) {
				return nil
			}
			r := rules[idx]
			if deletePendingID == r.ID { // second press on the armed rule → delete
				id := r.ID
				deletePendingID = 0
				go func() {
					// DeleteRule also removes the mirrored Gmail filter (service layer).
					a.GetErrorHandler().ClearPersistentMessage()
					if err := svc.DeleteRule(a.ctx, id); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Delete failed: %v", err))
						return
					}
					a.QueueUpdateDraw(reload)
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule deleted")
				}()
				return nil
			}
			// First press → arm. The prompt adapts: a mirrored rule (☁️) also drops the
			// real Gmail filter, a local-only rule exists nowhere else.
			deletePendingID = r.ID
			msg := fmt.Sprintf("Delete this rule? It only exists here — press '%s' again to confirm, Esc cancels", a.Keys.RuleDelete)
			if r.GmailFilterID != "" {
				msg = fmt.Sprintf("Delete this rule? This also deletes the Gmail filter — press '%s' again to confirm, Esc cancels", a.Keys.RuleDelete)
			}
			go a.GetErrorHandler().ShowPersistentMessage(a.ctx, msg, LogLevelInfo)
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
	a.markFocus("rules_manager")
	a.SetFocus(list)
	// :rules runs during command execution; hideCommandBar()'s restoreFocusAfterModal()
	// would otherwise re-focus the message list afterward. "keep" leaves our focus alone.
	a.cmd.focusOverride = "keep"

	// Fold the account's Gmail filters into the list in the background (one-shot
	// per open) and reconcile: matching filters adopt/import as rules, unrepresentable
	// ones show as read-only rows, and mirrored rules whose filter vanished from Gmail
	// are dropped to follow Gmail. The panel keeps working on local rules if Gmail fails.
	go func() {
		res, err := svc.ImportGmailFilters(a.ctx)
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("rules: Gmail filter import skipped: %v", err)
			}
			a.GetErrorHandler().ShowWarning(a.ctx, "Could not check Gmail filters — showing local rules only")
			return
		}
		a.QueueUpdateDraw(func() {
			if !a.isRulesPickerActive() {
				return // panel already closed — don't touch a recycled list
			}
			gmailOnly = res.Unsupported
			reload()
		})
		added := res.Imported + res.Adopted
		switch {
		case added > 0 && res.Removed > 0:
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("☁️ Synced with Gmail: %d added, %d removed", added, res.Removed))
		case added > 0:
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("☁️ %d Gmail filter(s) added to your rules", added))
		case res.Removed > 0:
			a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("☁️ %d rule(s) removed — no longer in Gmail", res.Removed))
		}
	}()

	if openForm {
		showRuleForm(nil)
	}
}
