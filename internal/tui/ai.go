package tui

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/ajramos/gmail-tui/internal/db"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// toggleAISummary shows/hides the AI summary pane and triggers generation if needed
func (a *App) toggleAISummary() {
	if a.debug {
		a.logger.Printf("toggleAISummary: called, aiSummaryVisible=%v, currentFocus=%s", a.aiSummaryVisible, a.currentFocus)
	}

	// Safety check: ensure application is ready and views are initialized
	if a.views == nil {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - views map is nil, application not ready")
		}
		a.showError("❌ Application not ready, please wait for messages to load")
		return
	}

	// Safety check: ensure list view exists and is accessible
	if _, ok := a.views["list"]; !ok {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - list view not found")
		}
		a.showError("❌ Message list not ready, please wait")
		return
	}

	// Safety check: ensure messages are not currently loading
	if a.IsMessagesLoading() {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - messages are currently loading")
		}
		a.showError("❌ Messages are loading, please wait")
		return
	}

	// Safety check: ensure messages are loaded
	if len(a.ids) == 0 {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - no messages loaded yet")
		}
		a.showError("❌ No messages loaded yet, please wait")
		return
	}

	// Safety check: ensure AI summary view is available
	if a.aiSummaryView == nil {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - AI summary view not initialized")
		}
		a.showError("❌ AI summary view not ready, please wait")
		return
	}

	if a.aiSummaryVisible && a.currentFocus == "summary" {
		// If AI summary is visible and focused, close it
		if a.debug {
			a.logger.Printf("toggleAISummary: AI summary visible and focused, closing panel")
		}
		a.closeAISummary()
		return
	}

	// Safety check: ensure views map exists
	if a.views == nil {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - views map is nil")
		}
		a.showError("❌ Application views not initialized")
		return
	}

	if a.debug {
		a.logger.Printf("toggleAISummary: views map exists, checking current message ID")
	}

	mid := a.GetCurrentMessageID()
	if a.debug {
		a.logger.Printf("toggleAISummary: current message ID='%s', len(ids)=%d", mid, len(a.ids))
	}

	if mid == "" && len(a.ids) > 0 {
		if a.debug {
			a.logger.Printf("toggleAISummary: trying to get message ID from table selection")
		}
		// Try to get from current table selection
		if table, ok := a.views["list"]; ok && table != nil {
			if tableView, ok := table.(*tview.Table); ok {
				row, _ := tableView.GetSelection()
				if a.debug {
					a.logger.Printf("toggleAISummary: table row selection=%d", row)
				}
				if row >= 0 && row < len(a.ids) {
					mid = a.ids[row]
					a.SetCurrentMessageID(mid)
					if a.debug {
						a.logger.Printf("toggleAISummary: set message ID to '%s' from table selection", mid)
					}
					go a.showMessage(mid)
				}
			}
		}
	}
	if mid == "" {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - no message ID available")
		}
		a.showError("❌ No message selected")
		return
	}

	if mid != "" {
		a.showMessageWithoutFocus(mid)
	}

	// Safety check: ensure contentSplit exists and is accessible
	if split, ok := a.views["contentSplit"]; ok && split != nil {
		if contentSplit, ok := split.(*tview.Flex); ok {
			contentSplit.ResizeItem(a.aiSummaryView, 0, 1)
		}
	}

	// Safety check: ensure aiSummaryView is initialized
	if a.aiSummaryView == nil {
		if a.debug {
			a.logger.Printf("toggleAISummary: ERROR - aiSummaryView is nil")
		}
		a.showError("❌ AI Summary view not initialized")
		return
	}

	if a.debug {
		a.logger.Printf("toggleAISummary: aiSummaryView is initialized, proceeding with summary generation")
	}

	a.aiSummaryVisible = true
	a.aiPanelInPromptMode = false // Reset prompt mode flag

	// Safety check: ensure aiSummaryView is accessible before setting focus
	if a.aiSummaryView != nil {
		a.SetFocus(a.aiSummaryView)
		a.currentFocus = "summary"
		a.aiSummaryView.SetBorderColor(tcell.ColorYellow)
		a.aiSummaryView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		// Reset title to AI Summary when switching from prompt mode
		a.aiSummaryView.SetTitle(" 🧠 AI Summary ")
		a.updateFocusIndicators("summary")
	} else {
		a.showError("❌ AI Summary view not accessible")
		return
	}

	if a.debug {
		a.logger.Printf("toggleAISummary: showing AI summary panel and generating summary for message '%s'", mid)
	}

	// Generate summary immediately
	go a.generateOrShowSummary(mid)
}

// closeAISummary closes the AI summary panel
func (a *App) closeAISummary() {
	if a.debug {
		a.logger.Printf("closeAISummary: closing AI summary panel")
	}

	if split, ok := a.views["contentSplit"]; ok && split != nil {
		if contentSplit, ok := split.(*tview.Flex); ok {
			contentSplit.ResizeItem(a.aiSummaryView, 0, 0)
		}
	}
	a.aiSummaryVisible = false
	a.aiPanelInPromptMode = false // Reset prompt mode flag when hiding panel
	
	// Cancel any active streaming operations when hiding panel
	if a.streamingCancel != nil {
		a.streamingCancel()
		a.streamingCancel = nil
	}

	// Safety check: ensure text view exists before setting focus
	if textView, ok := a.views["text"]; ok && textView != nil {
		a.SetFocus(textView)
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	} else {
		// Fallback to list view if text view is not available
		if listView, ok := a.views["list"]; ok && listView != nil {
			a.SetFocus(listView)
			a.currentFocus = "list"
			a.updateFocusIndicators("list")
		}
	}
	a.showStatusMessage("🙈 AI summary hidden")
}

// generateOrShowSummary shows cached summary or triggers generation if missing
func (a *App) generateOrShowSummary(messageID string) {
	if a.debug {
		a.logger.Printf("generateOrShowSummary: called for message '%s'", messageID)
	}

	if a.aiSummaryView == nil {
		if a.debug {
			a.logger.Printf("generateOrShowSummary: ERROR - aiSummaryView is nil")
		}
		return
	}

	// Check cache first
	if sum, ok := a.aiSummaryCache[messageID]; ok && sum != "" {
		if a.debug {
			a.logger.Printf("generateOrShowSummary: found cached summary for '%s'", messageID)
		}
		a.aiSummaryView.SetText(sanitizeForTerminal(sum))
		a.aiSummaryView.ScrollToBeginning()
		return
	}

	// Check if already processing
	if a.aiInFlight[messageID] {
		if a.debug {
			a.logger.Printf("generateOrShowSummary: already processing message '%s'", messageID)
		}
		a.aiSummaryView.SetText("🧠 Already summarizing…")
		a.aiSummaryView.ScrollToBeginning()
		return
	}

	// Check if LLM is available
	if a.LLM == nil {
		if a.debug {
			a.logger.Printf("generateOrShowSummary: ERROR - LLM is nil")
		}
		a.aiSummaryView.SetText("⚠️ LLM not available\n\nPlease check your LLM configuration.")
		a.aiSummaryView.ScrollToBeginning()
		return
	}

	// Show loading message
	a.aiSummaryView.SetText("🧠 Summarizing…")
	a.aiSummaryView.ScrollToBeginning()

	// Mark as in flight
	a.aiInFlight[messageID] = true

	// Generate summary in background following the working pattern
	go func(id string) {
		defer func() {
			// Always clean up in-flight status
			delete(a.aiInFlight, id)
		}()

		// Get message content
		m, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			if a.debug {
				a.logger.Printf("generateOrShowSummary: GetMessageWithContent error: %v", err)
			}
			a.QueueUpdateDraw(func() {
				a.aiSummaryView.SetText("⚠️ Error loading message\n\n" + err.Error())
				a.aiSummaryView.ScrollToBeginning()
			})
			return
		}

		// Prepare content for summary
		body := m.PlainText
		if len([]rune(body)) > 8000 {
			body = string([]rune(body)[:8000])
		}

		// Build prompt from configuration template, with a sensible fallback
		template := strings.TrimSpace(a.Config.LLM.SummarizePrompt)
		if template == "" {
			template = "Briefly summarize the following email. Keep it concise and factual.\n\n{{body}}"
		}
		prompt := strings.ReplaceAll(template, "{{body}}", body)

		if a.debug {
			a.logger.Printf("generateOrShowSummary: generating summary for message '%s', prompt size=%d", id, len(prompt))
		}

		// Try streaming first if supported, fallback to regular generation
		var finalResult string

		// Check if LLM supports streaming
		if streamer, ok := a.LLM.(interface {
			GenerateStream(context.Context, string, func(string)) error
		}); ok {
			if a.debug {
				a.logger.Printf("generateOrShowSummary: using streaming for message '%s'", id)
			}

			var resultBuilder strings.Builder
			ctx, cancel := context.WithCancel(a.ctx)
			a.streamingCancel = cancel // Store cancel function for Esc handler
			defer func() {
				cancel()
				a.streamingCancel = nil // Clear when done
			}()
			err = streamer.GenerateStream(ctx, prompt, func(token string) {
				resultBuilder.WriteString(token)
				// Update UI with each token for real-time streaming
				a.QueueUpdateDraw(func() {
					currentText := a.aiSummaryView.GetText(true)
					if currentText == "🧠 Summarizing…" {
						// First token, start building
						a.aiSummaryView.SetText("🧠 " + token)
					} else {
						// Append token to existing content
						a.aiSummaryView.SetText(currentText + token)
					}
					a.aiSummaryView.ScrollToEnd()
				})
			})

			if err == nil {
				finalResult = resultBuilder.String()
			}
		} else {
			if a.debug {
				a.logger.Printf("generateOrShowSummary: streaming not supported, using regular generation for message '%s'", id)
			}

			// Fallback to regular generation
			finalResult, err = a.LLM.Generate(prompt)
		}

		if err != nil {
			if a.debug {
				a.logger.Printf("generateOrShowSummary: LLM error: %v", err)
			}
			a.QueueUpdateDraw(func() {
				a.aiSummaryView.SetText("⚠️ Error generating summary\n\n" + err.Error())
				a.aiSummaryView.ScrollToBeginning()
			})
			return
		}

		// Cache the final result
		a.aiSummaryCache[id] = finalResult

		// If we used streaming, the UI is already updated. If not, show the final result
		if finalResult != "" && !strings.Contains(a.aiSummaryView.GetText(true), finalResult) {
			a.QueueUpdateDraw(func() {
				a.aiSummaryView.SetText(sanitizeForTerminal(finalResult))
				a.aiSummaryView.ScrollToBeginning()
			})
		}

		if a.debug {
			a.logger.Printf("generateOrShowSummary: completed successfully for message '%s'", id)
		}
	}(messageID)
}

// forceRegenerateSummary clears caches and regenerates the summary for the current message
func (a *App) forceRegenerateSummary() {
	id := a.getCurrentMessageID()
	if id == "" {
		a.showError("No message selected")
		return
	}
	// Remove from in-memory cache
	delete(a.aiSummaryCache, id)
	// Remove from SQLite cache (best-effort)
	if a.dbStore != nil && a.Config != nil && a.Config.LLM.CacheEnabled {
		if email, err := a.Client.ActiveAccountEmail(a.ctx); err == nil {
			cacheStore := db.NewCacheStore(a.dbStore)
			_ = cacheStore.DeleteAISummary(a.ctx, strings.ToLower(email), id)
		}
	}
	a.GetErrorHandler().ShowProgress(a.ctx, "Regenerating summary...")
	go a.generateOrShowSummary(id)
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
	// If search is active, do not start suggestion to avoid UI conflicts
	if a.currentFocus == "search" {
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
		template := strings.TrimSpace(a.Config.LLM.LabelPrompt)
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
		a.QueueUpdateDraw(func() {
			if a.currentFocus == "search" {
				// Clear persistent status if user moved to search meanwhile
				a.setStatusPersistent("")
				return
			}
			a.showLabelSuggestions(messageID, uniq)
			a.showStatusMessage("✅ Suggestions ready")
		})
	}()
}

// showLabelSuggestions displays a picker to apply one or all suggested labels
func (a *App) showLabelSuggestions(messageID string, suggestions []string) {
	if a.logger != nil {
		a.logger.Printf("showLabelSuggestions: start mid=%s count=%d", messageID, len(suggestions))
	}
	// Do not interrupt advanced search
	if a.currentFocus == "search" {
		if a.logger != nil {
			a.logger.Println("showLabelSuggestions: aborted (search active)")
		}
		a.showStatusMessage("🔎 Search active — suggestions deferred")
		return
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
