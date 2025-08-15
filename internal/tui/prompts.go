package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// openPromptPicker shows a picker similar to labels for selecting prompts
func (a *App) openPromptPicker() {
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Get message content for prompt processing
	message, err := a.Client.GetMessageWithContent(messageID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Failed to load message content")
		return
	}

	// Get prompt service
	_, _, _, _, _, promptService := a.GetServices()
	if promptService == nil {
		if a.logger != nil {
			a.logger.Printf("openPromptPicker: prompt service is nil")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available - check LLM and cache configuration")
		return
	}

	// Create picker UI similar to labels
	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30)
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)

	type promptItem struct {
		id          int
		name        string
		description string
		category    string
	}

	var all []promptItem
	var visible []promptItem

	// Reload function for filtering
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) &&
				!strings.Contains(strings.ToLower(item.description), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			// Category icon
			icon := "📄"
			switch item.category {
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			default:
				icon = "📝"
			}

			display := fmt.Sprintf("%s %s", icon, item.name)

			// Capture variables for closure
			promptID := item.id
			promptName := item.name

			list.AddItem(display, "Enter: apply", 0, func() {
				if a.logger != nil {
					a.logger.Printf("prompt picker: selected promptID=%d name=%s", promptID, promptName)
				}
				// Apply prompt (it will handle closing picker and setting focus)
				go a.applyPromptToMessage(messageID, promptID, promptName, message)
			})
		}
	}

	// Load prompts in background
	go func() {
		if a.logger != nil {
			a.logger.Printf("openPromptPicker: loading prompts...")
		}
		prompts, err := promptService.ListPrompts(a.ctx, "")
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("openPromptPicker: failed to load prompts: %v", err)
			}
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load prompts: %v", err))
			return
		}
		if a.logger != nil {
			a.logger.Printf("openPromptPicker: loaded %d prompts", len(prompts))
		}

		// Convert to promptItem, excluding bulk_analysis prompts
		all = make([]promptItem, 0, len(prompts))
		for _, p := range prompts {
			// Skip bulk_analysis prompts for single message picker
			if p.Category == "bulk_analysis" {
				continue
			}
			all = append(all, promptItem{
				id:          p.ID,
				name:        p.Name,
				description: p.Description,
				category:    p.Category,
			})
		}

		a.QueueUpdateDraw(func() {
			// Set up input field
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })

			// Allow navigation from input to list
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp || e.Key() == tcell.KeyPgDn || e.Key() == tcell.KeyPgUp {
					a.SetFocus(list)
					return e
				}
				return e
			})

			// Handle enter in input field (select first match)
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.closePromptPicker()
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						v := visible[0]
						if a.logger != nil {
							a.logger.Printf("prompt picker: pick via search promptID=%d name=%s", v.id, v.name)
						}
						// Apply prompt (it will handle closing picker and setting focus)
						go a.applyPromptToMessage(messageID, v.id, v.name, message)
					}
				}
			})

			// Create container
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
			container.SetBorder(true)
			container.SetTitle(" 🤖 Prompt Library ")
			container.SetTitleColor(tcell.ColorYellow)
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)

			// Footer
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter to apply | Esc to cancel ")
			footer.SetTextColor(tcell.ColorGray)
			container.AddItem(footer, 1, 0, false)

			// Handle navigation between input and list
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
					a.SetFocus(input)
					return nil
				}
				if e.Key() == tcell.KeyEscape {
					a.closePromptPicker()
					return nil
				}
				return e
			})

			// Add to content split like labels
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}
			a.SetFocus(input)
			a.currentFocus = "prompts"
			a.updateFocusIndicators("prompts")
			a.labelsVisible = true // Needed for proper visual state

			// Initial load
			reload("")
		})
	}()
}

// closePromptPicker closes the prompt picker and restores focus
func (a *App) closePromptPicker() {
	// Cancel any active streaming operations
	if a.streamingCancel != nil {
		a.streamingCancel()
		a.streamingCancel = nil
	}
	
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 0)
	}
	a.labelsVisible = false
	// Restore focus to text view
	if text, ok := a.views["text"].(*tview.TextView); ok {
		a.SetFocus(text)
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	}
}

// applyPromptToMessage applies the selected prompt to the message and shows result in AI panel
func (a *App) applyPromptToMessage(messageID string, promptID int, promptName string, message *gmail.Message) {
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: starting messageID=%s promptID=%d name=%s", messageID, promptID, promptName)
	}

	// Close picker first
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.labelsVisible = false
	})

	// Get services
	_, _, _, _, _, promptService := a.GetServices()
	if promptService == nil {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: prompt service not available")
		}
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: got services successfully")
	}

	// Extract message content using same pattern as AI summary
	content := message.PlainText
	if len([]rune(content)) > 8000 {
		content = string([]rune(content)[:8000])
	}

	if content == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No content found in message")
		return
	}

	// Show AI panel immediately with loading message and set focus
	a.QueueUpdateDraw(func() {
		// Show AI panel manually to avoid potential issues with toggleAISummary
		if !a.aiSummaryVisible {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiSummaryVisible = true
		}

		if a.aiSummaryView != nil {
			// Mark panel as being in prompt mode
			a.aiPanelInPromptMode = true

			// Update title to show prompt name
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 %s ", promptName))
			// Show loading message
			a.aiSummaryView.SetText("🤖 Applying prompt...")
			a.aiSummaryView.ScrollToBeginning()

			// Set focus to AI panel
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")

			if a.logger != nil {
				a.logger.Printf("applyPromptToMessage: initial panel setup - title: %s, text: 🤖 Applying prompt...", promptName)
			}
		}
	})

	// Get account email for caching
	accountEmail := a.getActiveAccountEmail()

	// Check if we have a cached result first
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: checking cache for accountEmail=%s messageID=%s promptID=%d", accountEmail, messageID, promptID)
	}
	if cachedResult, err := promptService.GetCachedResult(a.ctx, accountEmail, messageID, promptID); err == nil && cachedResult != nil {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: found cached result")
		}
		// Show progress message for cached result
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Loading cached result: %s...", promptName))

		// Update AI panel with cached result
		a.QueueUpdateDraw(func() {
			if a.aiSummaryView != nil {
				// Set the cached result text
				a.aiSummaryView.SetText(cachedResult.ResultText)
				a.aiSummaryView.ScrollToBeginning()
			}
		})

		// Clear progress and show success
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("%s (cached)", promptName))
		return
	}

	// Show progress in status
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: no cached result, applying prompt")
	}
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Applying prompt: %s...", promptName))

	// Get the prompt template to build the full prompt text
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: getting prompt template for promptID=%d", promptID)
	}
	template, err := promptService.GetPrompt(a.ctx, promptID)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("applyPromptToMessage: failed to get prompt template: %v", err)
		}
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to get prompt template: %v", err))
		return
	}
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: got template name=%s", template.Name)
	}

	// Build the full prompt text with variable substitution
	promptText := template.PromptText
	variables := map[string]string{
		"from":    a.extractHeader(message, "From"),
		"subject": a.extractHeader(message, "Subject"),
		"date":    a.extractHeader(message, "Date"),
		"body":    content,
	}

	// Replace all variables in the prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		promptText = strings.ReplaceAll(promptText, placeholder, value)
	}

	// Try streaming if enabled and supported
	if a.Config != nil && a.Config.LLMStreamEnabled {
		if prov, ok := a.LLM.(interface{ Name() string }); ok && prov.Name() == "ollama" {
			if streamer, ok2 := a.LLM.(interface {
				GenerateStream(context.Context, string, func(string)) error
			}); ok2 {
				if a.logger != nil {
					a.logger.Printf("applyPromptToMessage: using streaming")
				}

				// Show streaming progress in status bar
				a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Streaming prompt: %s...", promptName))

				var b strings.Builder
				a.QueueUpdateDraw(func() {
					if a.aiSummaryView != nil {
						// Show loading message before starting streaming
						a.aiSummaryView.SetText("🤖 Processing prompt...")
					}
				})

				ctx, cancel := context.WithCancel(a.ctx)
				a.streamingCancel = cancel // Store cancel function for Esc handler
				defer func() {
					cancel()
					a.streamingCancel = nil // Clear when done
				}()
				if a.logger != nil {
					a.logger.Printf("applyPromptToMessage: starting GenerateStream")
				}
				err := streamer.GenerateStream(ctx, promptText, func(tok string) {
					b.WriteString(tok)
					if a.logger != nil && len(b.String())%100 == 0 { // Log every 100 chars
						a.logger.Printf("applyPromptToMessage: streaming progress: %d chars", len(b.String()))
					}
					currentText := sanitizeForTerminal(b.String())
					a.QueueUpdateDraw(func() {
						if a.aiSummaryView != nil {
							a.aiSummaryView.SetText(currentText)
							if a.logger != nil && len(currentText)%200 == 0 { // Log UI updates
								a.logger.Printf("applyPromptToMessage: UI updated with %d chars", len(currentText))
							}
						}
					})
				})
				if a.logger != nil {
					a.logger.Printf("applyPromptToMessage: GenerateStream completed, result length: %d", len(b.String()))
				}

				if err != nil {
					a.GetErrorHandler().ClearProgress()
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply prompt: %v", err))
					return
				}

				result := b.String()

				// Final UI update to ensure result is shown
				a.QueueUpdateDraw(func() {
					if a.aiSummaryView != nil {
						finalText := sanitizeForTerminal(result)
						a.aiSummaryView.SetText(finalText)
						a.aiSummaryView.ScrollToBeginning()
						if a.logger != nil {
							a.logger.Printf("applyPromptToMessage: final UI update, text length: %d", len(finalText))
						}
					}
				})

				// Save result for history
				_ = promptService.SaveResult(a.ctx, accountEmail, messageID, promptID, result)

				// Increment usage count
				_ = promptService.IncrementUsage(a.ctx, promptID)

				// Clear progress and show success
				a.GetErrorHandler().ClearProgress()
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", promptName))
				return
			}
		}
	}

	// Fallback to non-streaming
	if a.logger != nil {
		a.logger.Printf("applyPromptToMessage: using non-streaming fallback")
	}

	result, err := promptService.ApplyPrompt(a.ctx, content, promptID, map[string]string{
		"from":    a.extractHeader(message, "From"),
		"subject": a.extractHeader(message, "Subject"),
		"date":    a.extractHeader(message, "Date"),
	})

	if err != nil {
		a.GetErrorHandler().ClearProgress()
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply prompt: %v", err))
		return
	}

	// Save result for history
	_ = promptService.SaveResult(a.ctx, accountEmail, messageID, promptID, result.ResultText)

	// Update AI panel with result
	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			// Set the result text
			a.aiSummaryView.SetText(result.ResultText)
			a.aiSummaryView.ScrollToBeginning()
		}
	})

	// Clear progress and show success
	a.GetErrorHandler().ClearProgress()
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", promptName))
}

// extractHeader extracts a header value from a message
func (a *App) extractHeader(message *gmail.Message, headerName string) string {
	if message.Payload == nil || message.Payload.Headers == nil {
		return ""
	}

	for _, header := range message.Payload.Headers {
		if header.Name == headerName {
			return header.Value
		}
	}
	return ""
}
