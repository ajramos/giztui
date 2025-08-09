package tui

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// toggleAISummary shows/hides the AI summary pane and triggers generation if needed
func (a *App) toggleAISummary() {
	if a.aiSummaryVisible && a.currentFocus == "summary" {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.aiSummaryView, 0, 0)
		}
		a.aiSummaryVisible = false
		a.SetFocus(a.views["textContainer"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
		a.showStatusMessage("🙈 AI summary hidden")
		return
	}

	mid := a.getCurrentMessageID()
	if mid == "" && len(a.ids) > 0 {
		if list, ok := a.views["list"].(*tview.List); ok {
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(a.ids) {
				mid = a.ids[idx]
				go a.showMessage(mid)
			}
		}
	}
	if mid == "" {
		a.showError("No message selected")
		return
	}

	if mid != "" {
		a.showMessageWithoutFocus(mid)
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.aiSummaryView, 0, 1)
	}
	a.aiSummaryVisible = true
	a.SetFocus(a.aiSummaryView)
	a.currentFocus = "summary"
	if a.aiSummaryView != nil {
		a.aiSummaryView.SetBorderColor(tcell.ColorYellow)
	}
	a.updateFocusIndicators("summary")

	a.generateOrShowSummary(mid)
}

// generateOrShowSummary shows cached summary or triggers generation if missing
func (a *App) generateOrShowSummary(messageID string) {
	if a.aiSummaryView == nil {
		return
	}
	if sum, ok := a.aiSummaryCache[messageID]; ok && sum != "" {
		a.aiSummaryView.SetText(sum)
		a.aiSummaryView.ScrollToBeginning()
		a.setStatusPersistent("🤖 Summary loaded from cache")
		return
	}
	if a.aiInFlight[messageID] {
		a.aiSummaryView.SetText("🧠 Summarizing…")
		a.aiSummaryView.ScrollToBeginning()
		a.setStatusPersistent("🧠 Summarizing…")
		return
	}
	a.aiSummaryView.SetText("🧠 Summarizing…")
	a.aiSummaryView.ScrollToBeginning()
	a.setStatusPersistent("🧠 Summarizing…")
	a.aiInFlight[messageID] = true
	go func(id string) {
		m, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.QueueUpdateDraw(func() {
				a.aiSummaryView.SetText("⚠️ Error loading message")
				a.showStatusMessage("⚠️ Error loading message")
			})
			delete(a.aiInFlight, id)
			return
		}
		if a.LLM == nil {
			a.QueueUpdateDraw(func() { a.aiSummaryView.SetText("⚠️ LLM disabled"); a.showStatusMessage("⚠️ LLM disabled") })
			delete(a.aiInFlight, id)
			return
		}
		body := m.PlainText
		if len([]rune(body)) > 8000 {
			body = string([]rune(body)[:8000])
		}
		resp, err := a.LLM.Generate("Summarize in 3 bullet points (keep language).\n\n" + body)
		if err != nil {
			a.QueueUpdateDraw(func() { a.aiSummaryView.SetText("⚠️ LLM error"); a.showStatusMessage("⚠️ LLM error") })
			delete(a.aiInFlight, id)
			return
		}
		a.aiSummaryCache[id] = resp
		delete(a.aiInFlight, id)
		a.QueueUpdateDraw(func() {
			a.aiSummaryView.SetText(resp)
			a.aiSummaryView.ScrollToBeginning()
			a.showStatusMessage("✅ Summary ready")
		})
	}(messageID)
}

// suggestLabel suggests a label using LLM
func (a *App) suggestLabel() {
	if a.LLM == nil {
		a.showStatusMessage("⚠️ LLM disabled")
		return
	}
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}
	if cached, ok := a.aiLabelsCache[messageID]; ok && len(cached) > 0 {
		a.showLabelSuggestions(messageID, cached)
		return
	}
	a.setStatusPersistent("🏷️ Suggesting labels…")
	go func() {
		m, err := a.Client.GetMessageWithContent(messageID)
		if err != nil {
			a.showError("❌ Error loading message")
			return
		}
		labels, err := a.Client.ListLabels()
		if err != nil || len(labels) == 0 {
			a.showError("❌ Error loading labels")
			return
		}
		allowed := make([]string, 0, len(labels))
		nameToID := make(map[string]string, len(labels))
		for _, l := range labels {
			if strings.HasPrefix(l.Id, "CATEGORY_") || l.Id == "INBOX" || l.Id == "SENT" || l.Id == "DRAFT" || l.Id == "SPAM" || l.Id == "TRASH" || l.Id == "CHAT" || (strings.HasSuffix(l.Id, "_STARRED") && l.Id != "STARRED") {
				continue
			}
			allowed = append(allowed, l.Name)
			nameToID[l.Name] = l.Id
		}
		sort.Slice(allowed, func(i, j int) bool { return strings.ToLower(allowed[i]) < strings.ToLower(allowed[j]) })
		body := m.PlainText
		if len([]rune(body)) > 6000 {
			body = string([]rune(body)[:6000])
		}
		prompt := "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: " + strings.Join(allowed, ", ") + "\n\nEmail:\n" + body
		resp, err := a.LLM.Generate(prompt)
		if err != nil {
			a.showStatusMessage("⚠️ LLM error")
			return
		}
		var arr []string
		if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &arr); err != nil {
			parts := strings.Split(resp, ",")
			for _, p := range parts {
				if s := strings.TrimSpace(strings.Trim(p, "\"[]")); s != "" {
					arr = append(arr, s)
				}
			}
		}
		uniq := make([]string, 0, 3)
		seen := make(map[string]struct{})
		for _, s := range arr {
			if _, ok := nameToID[s]; !ok {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			uniq = append(uniq, s)
			if len(uniq) == 3 {
				break
			}
		}
		if len(uniq) == 0 {
			a.showStatusMessage("ℹ️ No label suggestion")
			return
		}
		a.aiLabelsCache[messageID] = uniq
		a.QueueUpdateDraw(func() { a.showLabelSuggestions(messageID, uniq); a.showStatusMessage("✅ Suggestions ready") })
	}()
}

// showLabelSuggestions displays a picker to apply one or all suggested labels
func (a *App) showLabelSuggestions(messageID string, suggestions []string) {
	picker := tview.NewList().ShowSecondaryText(false)
	picker.SetBorder(true)
	picker.SetTitle(" 🏷️ Apply suggested label(s) ")

	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError("❌ Error loading labels")
		return
	}
	nameToID := make(map[string]string, len(labels))
	for _, l := range labels {
		nameToID[l.Name] = l.Id
	}
	for _, name := range suggestions {
		labelName := name
		picker.AddItem(labelName, "Enter to apply", 0, func() {
			if id, ok := nameToID[labelName]; ok {
				go func() {
					if err := a.Client.ApplyLabel(messageID, id); err != nil {
						a.showError("❌ Error applying label")
						return
					}
					a.updateCachedMessageLabels(messageID, id, true)
					a.showStatusMessage("✅ Applied: " + labelName)
					a.Pages.SwitchToPage("main")
					a.restoreFocusAfterModal()
				}()
			}
		})
	}
	picker.AddItem("✅ Apply all", "Apply all suggested labels", 0, func() {
		go func() {
			for _, name := range suggestions {
				if id, ok := nameToID[name]; ok {
					_ = a.Client.ApplyLabel(messageID, id)
					a.updateCachedMessageLabels(messageID, id, true)
				}
			}
			a.showStatusMessage("✅ Applied all suggestions")
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
		}()
	})
	picker.AddItem("🗂️  Pick from all labels", "Open full label list to apply", 0, func() {
		a.showAllLabelsPicker(messageID)
	})
	picker.AddItem("➕ Add custom label", "Create or pick a label and apply", 0, func() {
		input := tview.NewInputField().
			SetLabel("Label name: ").
			SetFieldWidth(30)
		modal := tview.NewFlex().SetDirection(tview.FlexRow)
		title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
		title.SetBorder(true)
		title.SetText("Enter a label name | Enter=apply, ESC=cancel")
		modal.AddItem(title, 3, 0, false)
		modal.AddItem(input, 3, 0, true)

		input.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEscape {
				a.Pages.SwitchToPage("aiLabelSuggestions")
				a.SetFocus(picker)
				return
			}
			if key == tcell.KeyEnter {
				name := strings.TrimSpace(input.GetText())
				if name == "" {
					return
				}
				go func() {
					id, ok := nameToID[name]
					if !ok {
						for n, i := range nameToID {
							if strings.EqualFold(n, name) {
								id = i
								ok = true
								break
							}
						}
					}
					if !ok {
						created, err := a.Client.CreateLabel(name)
						if err != nil {
							a.showError("❌ Error creating label")
							return
						}
						id = created.Id
						nameToID[name] = id
					}
					if err := a.Client.ApplyLabel(messageID, id); err != nil {
						a.showError("❌ Error applying label")
						return
					}
					a.updateCachedMessageLabels(messageID, id, true)
					a.showStatusMessage("✅ Applied: " + name)
					a.QueueUpdateDraw(func() { a.Pages.SwitchToPage("main"); a.restoreFocusAfterModal() })
				}()
			}
		})

		a.Pages.AddPage("aiLabelAddCustom", modal, true, true)
		a.Pages.SwitchToPage("aiLabelAddCustom")
		a.SetFocus(input)
	})

	picker.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			return nil
		}
		return e
	})

	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select label to apply | Enter=apply, ESC=back")
	v.AddItem(title, 3, 0, false)
	v.AddItem(picker, 0, 1, true)
	a.Pages.AddPage("aiLabelSuggestions", v, true, true)
	a.Pages.SwitchToPage("aiLabelSuggestions")
	if picker.GetItemCount() > 0 {
		picker.SetCurrentItem(0)
	}
	a.SetFocus(picker)
}
