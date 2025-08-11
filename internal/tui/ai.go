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
		a.SetFocus(a.views["text"])
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
		a.aiSummaryView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
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
		a.aiSummaryView.SetText(sanitizeForTerminal(sum))
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
		// Build prompt from configuration template, with a sensible fallback
		template := strings.TrimSpace(a.Config.SummarizePrompt)
		if template == "" {
			template = "Resume brevemente el siguiente correo electrónico:\n\n{{body}}\n\nDevuelve el resumen en español en un párrafo."
		}
		prompt := strings.ReplaceAll(template, "{{body}}", body)
		resp, err := a.LLM.Generate(prompt)
		if err != nil {
			a.QueueUpdateDraw(func() {
				// Mostrar detalle del error en el panel de resumen de IA (más grande)
				a.aiSummaryView.SetText("⚠️ LLM error while summarizing\n\n" + strings.TrimSpace(err.Error()))
				a.aiSummaryView.ScrollToBeginning()
				a.showLLMError("summarize", err)
			})
			delete(a.aiInFlight, id)
			return
		}
		a.aiSummaryCache[id] = resp
		delete(a.aiInFlight, id)
		a.QueueUpdateDraw(func() {
			a.aiSummaryView.SetText(sanitizeForTerminal(resp))
			a.aiSummaryView.ScrollToBeginning()
			a.showStatusMessage("✅ Summary ready")
		})
	}(messageID)
}

// suggestLabel suggests a label using LLM
func (a *App) suggestLabel() {
	if a.logger != nil {
		a.logger.Printf("suggestLabel: start for %s", a.getCurrentMessageID())
	}
	if a.LLM == nil {
		// Fallback UX: abrir selector completo para no dejar al usuario sin salida
		mid := a.getCurrentMessageID()
		if mid != "" {
			a.showStatusMessage("⚠️ LLM disabled — opening all labels picker")
			a.showAllLabelsPicker(mid)
		} else {
			a.showStatusMessage("⚠️ LLM disabled")
		}
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
			if a.logger != nil {
				a.logger.Printf("suggestLabel: GetMessageWithContent error: %v", err)
			}
			a.showError("❌ Error loading message")
			return
		}
		labels, err := a.Client.ListLabels()
		if err != nil || len(labels) == 0 {
			if a.logger != nil {
				a.logger.Printf("suggestLabel: ListLabels error: %v", err)
			}
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
		// Build prompt from configuration template, with a sensible fallback
		template := strings.TrimSpace(a.Config.LabelPrompt)
		if template == "" {
			template = "From the email below, pick up to 3 labels from this list only. Return a JSON array of label names, nothing else.\n\nLabels: {{labels}}\n\nEmail:\n{{body}}"
		}
		tmpl := strings.ReplaceAll(template, "{{labels}}", strings.Join(allowed, ", "))
		prompt := strings.ReplaceAll(tmpl, "{{body}}", body)
		if a.logger != nil {
			a.logger.Printf("suggestLabel: prompt size=%d", len(prompt))
		}
		resp, err := a.LLM.Generate(prompt)
		if err != nil {
			// Fallback: mostrar selector completo para que el usuario pueda aplicar manualmente
			a.showLLMError("suggest labels", err)
			if a.logger != nil {
				a.logger.Printf("suggestLabel: LLM error: %v", err)
			}
			a.QueueUpdateDraw(func() { a.showAllLabelsPicker(messageID) })
			return
		}
		if a.logger != nil {
			a.logger.Printf("suggestLabel: raw response: %q", resp)
		}
		// Try strict JSON first; then fallback to heuristic extraction (bulleted lines, quoted names)
		var arr []string
		if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &arr); err != nil {
			arr = extractLabelsFromLLMResponse(resp)
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
		// Always show panel (even empty) to keep UX consistent
		a.aiLabelsCache[messageID] = uniq
		a.QueueUpdateDraw(func() { a.showLabelSuggestions(messageID, uniq); a.showStatusMessage("✅ Suggestions ready") })
	}()
}

// showLabelSuggestions displays a picker to apply one or all suggested labels
func (a *App) showLabelSuggestions(messageID string, suggestions []string) {
	if a.logger != nil {
		a.logger.Printf("showLabelSuggestions: start mid=%s count=%d", messageID, len(suggestions))
	}
	a.setStatusPersistent("🏷️ Showing suggested labels…")
	// Do network work off the UI thread
	go func() {
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError("❌ Error loading labels")
			return
		}
		nameToID := make(map[string]string, len(labels))
		for _, l := range labels {
			nameToID[l.Name] = l.Id
		}
		// Build UI on the UI thread
		a.QueueUpdateDraw(func() {
			body := tview.NewList().ShowSecondaryText(false)
			body.SetBorder(false)
			if len(suggestions) == 0 {
				body.AddItem("(No suggestions)", "Use Browse all or Add custom", 0, nil)
			}
			// Mark suggestions already applied with ✅
			appliedSet := make(map[string]bool)
			if meta, ok := a.messageCache[messageID]; ok && meta != nil {
				for _, ln := range meta.Labels {
					appliedSet[ln] = true
				}
			}
			for _, name := range suggestions {
				lbl := name
				prefix := "○ "
				if appliedSet[lbl] {
					prefix = "✅ "
				}
				body.AddItem(prefix+lbl, "Enter: apply", 0, func() {
					if id, ok := nameToID[lbl]; ok {
						go func() {
							if err := a.Client.ApplyLabel(messageID, id); err != nil {
								a.showError("❌ Error applying label")
								return
							}
							a.updateCachedMessageLabels(messageID, id, true)
							a.QueueUpdateDraw(func() {
								a.showStatusMessage("✅ Applied: " + lbl)
								a.refreshMessageContent(messageID)
							})
						}()
					}
				})
			}
			if len(suggestions) > 1 {
				body.AddItem("✅ Apply all", "Apply all suggested labels", 0, func() {
					go func() {
						for _, name := range suggestions {
							if id, ok := nameToID[name]; ok {
								_ = a.Client.ApplyLabel(messageID, id)
								a.updateCachedMessageLabels(messageID, id, true)
							}
						}
						a.QueueUpdateDraw(func() {
							a.showStatusMessage("✅ Applied all suggestions")
							a.refreshMessageContent(messageID)
						})
					}()
				})
			}
			// Use magnifying glass like other places
			body.AddItem("🔍 Browse all labels…", "Enter to apply 1st match | Esc to back", 0, func() { a.expandLabelsBrowse(messageID) })
			body.AddItem("➕ Add custom label…", "Create or apply", 0, func() { a.addCustomLabelInline(messageID) })
			// Remove explicit Back item; ESC hint will be shown in footer and ESC returns to quick view

			body.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyEscape {
					// Go back to quick view within the side panel
					a.labelsExpanded = false
					a.populateLabelsQuickView(messageID)
					return nil
				}
				return e
			})

			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBorder(true)
			container.SetTitle(" 🏷️  Suggested Labels ")
			container.SetTitleColor(tcell.ColorYellow)
			container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			container.AddItem(body, 0, 1, true)
			// Footer hint
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter to apply  |  Esc to back ")
			footer.SetTextColor(tcell.ColorGray)
			container.AddItem(footer, 1, 0, false)

			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}
			a.labelsVisible = true
			a.currentFocus = "labels"
			a.updateFocusIndicators("labels")
			a.SetFocus(body)
			if body.GetItemCount() > 0 {
				body.SetCurrentItem(0)
			}
			if a.logger != nil {
				a.logger.Printf("showLabelSuggestions: mounted; items=%d", body.GetItemCount())
			}
		})
	}()
}

// extractLabelsFromLLMResponse attempts to pull label names from free-form text.
// It supports bullet lists ("- name", "* name"), lines with quotes, and
// simple patterns like "label is: \"Name\"". Returns a deduplicated list.
func extractLabelsFromLLMResponse(resp string) []string {
	lines := strings.Split(resp, "\n")
	out := make([]string, 0, 6)
	seen := make(map[string]struct{})
	add := func(s string) {
		s = strings.TrimSpace(s)
		s = strings.Trim(s, "\"'`“”‘’[]")
		if s == "" {
			return
		}
		// keep short but meaningful strings; avoid generic words
		if len([]rune(s)) < 2 {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, ln := range lines {
		l := strings.TrimSpace(ln)
		if l == "" {
			continue
		}
		// bullets: - label or * label
		if strings.HasPrefix(l, "-") || strings.HasPrefix(l, "*") {
			l = strings.TrimLeft(l, "-*• ")
			// split on colon if format "Zscaler: description"
			parts := strings.SplitN(l, ":", 2)
			add(parts[0])
			continue
		}
		// lines with quotes
		if strings.Contains(l, "\"") {
			// take content within first quotes
			first := strings.Index(l, "\"")
			if first >= 0 {
				rest := l[first+1:]
				if end := strings.Index(rest, "\""); end > 0 {
					add(rest[:end])
					continue
				}
			}
		}
		// fallback: if sentence contains label is/are, take last word(s)
		low := strings.ToLower(l)
		if strings.Contains(low, "label is") || strings.Contains(low, "labels are") {
			l = strings.TrimPrefix(low, "label is:")
			l = strings.TrimPrefix(low, "labels are:")
			add(l)
		}
	}
	// limit to 3
	if len(out) > 3 {
		return out[:3]
	}
	return out
}
