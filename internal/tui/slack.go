package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ajramos/gmail-tui/internal/services"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// showSlackForwardDialog shows the Slack forwarding panel (like labels)
func (a *App) showSlackForwardDialog() {
	// Toggle contextual panel like AI Summary and Labels
	if a.slackVisible {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.slackView, 0, 0)
		}
		a.slackVisible = false
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
		// Use ErrorHandler instead of showStatusMessage
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "💬 Slack panel hidden")
		}()
		return
	}

	// Check if Slack is enabled
	if !a.Config.Slack.Enabled {
		a.GetErrorHandler().ShowError(a.ctx, "Slack integration is not enabled in configuration")
		return
	}

	// Check if we have a selected message
	messageID := a.GetCurrentMessageID()
	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Ensure message content is shown without stealing focus
	a.showMessageWithoutFocus(messageID)

	// Show panel and load quick view
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.slackView, 0, 1)
	}
	a.slackVisible = true
	a.currentFocus = "slack"
	a.updateFocusIndicators("slack")
	a.populateSlackPanel(messageID)
}

// showSlackBulkForwardDialog shows the Slack forwarding panel for bulk operations
func (a *App) showSlackBulkForwardDialog() {
	// Check if Slack is enabled
	if !a.Config.Slack.Enabled {
		a.GetErrorHandler().ShowError(a.ctx, "Slack integration is not enabled in configuration")
		return
	}

	// Check if we have selected messages
	if !a.bulkMode || len(a.selected) == 0 {
		a.GetErrorHandler().ShowError(a.ctx, "No messages selected for bulk Slack forwarding")
		return
	}

	messageCount := len(a.selected)
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("Preparing to forward %d messages to Slack", messageCount))
	}()

	// Show panel and load bulk view
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.slackView, 0, 1)
	}
	a.slackVisible = true
	a.currentFocus = "slack"
	a.updateFocusIndicators("slack")
	a.populateSlackBulkPanel()
}

// populateSlackPanel populates the Slack forwarding panel
func (a *App) populateSlackPanel(messageID string) {
	// Get Slack service
	slackService := a.GetSlackService()
	if slackService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Slack service not available")
		return
	}

	// Get configured channels
	channels, err := slackService.ListConfiguredChannels(a.ctx)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load Slack channels: %v", err))
		return
	}

	if len(channels) == 0 {
		a.GetErrorHandler().ShowError(a.ctx, "No Slack channels configured")
		return
	}

	channelList := a.createSlackPanel(messageID, channels)
	
	// Set focus after panel is fully created and populated
	a.SetFocus(channelList)
}

// populateSlackBulkPanel populates the Slack forwarding panel for bulk operations
func (a *App) populateSlackBulkPanel() {
	// Get Slack service
	slackService := a.GetSlackService()
	if slackService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Slack service not available")
		return
	}

	// Get configured channels
	channels, err := slackService.ListConfiguredChannels(a.ctx)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to load Slack channels: %v", err))
		return
	}

	if len(channels) == 0 {
		a.GetErrorHandler().ShowError(a.ctx, "No Slack channels configured")
		return
	}

	messageCount := len(a.selected)
	channelList := a.createSlackBulkPanel(messageCount, channels)
	
	// Set focus after panel is fully created and populated
	a.SetFocus(channelList)
}

// createSlackPanel creates the Slack forwarding contextual panel and returns the channel list for focus setting
func (a *App) createSlackPanel(messageID string, channels []services.SlackChannel) *tview.List {
	// Clear existing slack view
	a.slackView.Clear()

	// No title needed - removing "🔗 Available channels" as requested

	// Channel selection list
	channelList := tview.NewList()
	channelList.ShowSecondaryText(false)
	channelList.SetBorder(false)

	// Find default channel
	defaultIndex := 0
	for i, channel := range channels {
		displayName := fmt.Sprintf("📺 %s", channel.Name)
		channelList.AddItem(displayName, "", 0, nil)
		if channel.Default {
			defaultIndex = i
		}
	}
	channelList.SetCurrentItem(defaultIndex)

	// Pre-message input in same row as label
	userMessageInput := tview.NewInputField()
	userMessageInput.SetLabel("📝 Pre-message: ")
	userMessageInput.SetLabelColor(tcell.ColorYellow)
	userMessageInput.SetBorder(false)
	userMessageInput.SetPlaceholder("Hey guys, heads up with this email...")

	// Add spacing between optional message and instructions
	spacer := tview.NewTextView()
	spacer.SetText("\n")
	
	// Instructions
	instructions := tview.NewTextView()
	instructions.SetText("Enter to Send | Esc to Close")
	instructions.SetTextAlign(tview.AlignRight)
	instructions.SetTextColor(tcell.ColorGray)

	// Set up Enter key handler for sending
	channelList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selectedChannel := channels[index]
		userMessage := strings.TrimSpace(userMessageInput.GetText())
		
		options := services.SlackForwardOptions{
			ChannelID:   selectedChannel.ID,
			WebhookURL:  selectedChannel.WebhookURL,
			ChannelName: selectedChannel.Name,
			UserMessage: userMessage,
			FormatStyle: a.Config.Slack.Defaults.FormatStyle,
		}

		a.forwardEmailToSlack(messageID, options)
		
		// Hide the Slack panel
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.slackView, 0, 0)
		}
		a.slackVisible = false
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
	})

	// Handle input field enter key for sending
	userMessageInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// Trigger the same send logic
			index := channelList.GetCurrentItem()
			if index >= 0 && index < len(channels) {
				selectedChannel := channels[index]
				userMessage := strings.TrimSpace(userMessageInput.GetText())
				
				options := services.SlackForwardOptions{
					ChannelID:   selectedChannel.ID,
					WebhookURL:  selectedChannel.WebhookURL,
					ChannelName: selectedChannel.Name,
					UserMessage: userMessage,
					FormatStyle: a.Config.Slack.Defaults.FormatStyle,
				}

				a.forwardEmailToSlack(messageID, options)
				
				// Hide the Slack panel
				a.hideSlackPanel()
			}
		} else if key == tcell.KeyEscape {
			// ESC closes the Slack panel
			a.hideSlackPanel()
		} else if key == tcell.KeyTab {
			// Tab moves back to channel list
			a.SetFocus(channelList)
		}
	})

	// Handle tab navigation from channel list to input field
	channelList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.SetFocus(userMessageInput)
			return nil
		} else if event.Key() == tcell.KeyEscape {
			// ESC closes the Slack panel
			a.hideSlackPanel()
			return nil
		}
		return event
	})

	// Layout the panel
	a.slackView.AddItem(channelList, 0, 1, true)
	a.slackView.AddItem(userMessageInput, 1, 0, false)
	a.slackView.AddItem(instructions, 1, 0, false)

	// Return channelList for focus setting
	return channelList
}

// createSlackBulkPanel creates the Slack forwarding panel for bulk operations
func (a *App) createSlackBulkPanel(messageCount int, channels []services.SlackChannel) *tview.List {
	// Clear existing slack view
	a.slackView.Clear()

	// Channel selection list
	channelList := tview.NewList()
	channelList.ShowSecondaryText(false)
	channelList.SetBorder(false)

	// Find default channel
	defaultIndex := 0
	for i, channel := range channels {
		displayName := fmt.Sprintf("📺 %s", channel.Name)
		channelList.AddItem(displayName, "", 0, nil)
		if channel.Default {
			defaultIndex = i
		}
	}
	channelList.SetCurrentItem(defaultIndex)

	// Comment input field for bulk operation (like Obsidian)
	commentLabel := tview.NewTextView().SetText(fmt.Sprintf("💬 Bulk comment (%d emails):", messageCount))
	commentLabel.SetTextColor(tcell.ColorYellow)

	userMessageInput := tview.NewInputField()
	userMessageInput.SetLabel("")
	userMessageInput.SetText("")
	userMessageInput.SetPlaceholder("Add a message that will be included with all forwarded emails...")
	userMessageInput.SetFieldWidth(50)
	userMessageInput.SetBorder(false)
	userMessageInput.SetFieldBackgroundColor(tcell.ColorBlue)
	userMessageInput.SetFieldTextColor(tcell.ColorDarkGreen)

	// Instructions
	instructions := tview.NewTextView()
	instructions.SetText(fmt.Sprintf("Enter to Send %d emails | Esc to Close | Tab to navigate", messageCount))
	instructions.SetTextAlign(tview.AlignCenter)
	instructions.SetTextColor(tcell.ColorGray)

	// Create a horizontal flex for label and input alignment (like Obsidian)
	commentRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	commentRow.AddItem(commentLabel, 0, 1, false)
	commentRow.AddItem(userMessageInput, 0, 1, false)

	// Set up Enter key handler for sending bulk messages
	channelList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selectedChannel := channels[index]
		userMessage := strings.TrimSpace(userMessageInput.GetText())
		
		options := services.SlackForwardOptions{
			ChannelID:   selectedChannel.ID,
			WebhookURL:  selectedChannel.WebhookURL,
			ChannelName: selectedChannel.Name,
			UserMessage: userMessage,
			FormatStyle: a.Config.Slack.Defaults.FormatStyle,
		}

		a.forwardBulkEmailsToSlack(options)
		
		// Hide the Slack panel
		a.hideSlackPanel()
	})

	// Handle input field enter key for sending
	userMessageInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// Trigger the same send logic
			index := channelList.GetCurrentItem()
			if index >= 0 && index < len(channels) {
				selectedChannel := channels[index]
				userMessage := strings.TrimSpace(userMessageInput.GetText())
				
				options := services.SlackForwardOptions{
					ChannelID:   selectedChannel.ID,
					WebhookURL:  selectedChannel.WebhookURL,
					ChannelName: selectedChannel.Name,
					UserMessage: userMessage,
					FormatStyle: a.Config.Slack.Defaults.FormatStyle,
				}

				a.forwardBulkEmailsToSlack(options)
				
				// Hide the Slack panel
				a.hideSlackPanel()
			}
		} else if key == tcell.KeyEscape {
			// ESC closes the Slack panel
			a.hideSlackPanel()
		} else if key == tcell.KeyTab {
			// Tab moves back to channel list
			a.SetFocus(channelList)
		}
	})

	// Handle tab navigation from channel list to input field
	channelList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			a.SetFocus(userMessageInput)
			return nil
		} else if event.Key() == tcell.KeyEscape {
			// ESC closes the Slack panel
			a.hideSlackPanel()
			return nil
		}
		return event
	})

	// Layout the panel (bulk version)
	a.slackView.AddItem(channelList, 0, 1, true)
	a.slackView.AddItem(commentRow, 2, 0, false)
	a.slackView.AddItem(instructions, 1, 0, false)

	// Return channelList for focus setting, but can navigate to comment input with Tab
	return channelList
}

// hideSlackPanel hides the Slack panel (synchronous operation like hideAIPanel)
func (a *App) hideSlackPanel() {
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.slackView, 0, 0)
	}
	a.slackVisible = false
	a.SetFocus(a.views["text"])
	a.currentFocus = "text"
	a.updateFocusIndicators("text")
}


// forwardEmailToSlack forwards the email using the Slack service
func (a *App) forwardEmailToSlack(messageID string, options services.SlackForwardOptions) {
	slackService := a.GetSlackService()
	if slackService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Slack service not available")
		return
	}

	// For "full" format, get the TUI-processed content
	if options.FormatStyle == "full" {
		// Get processed message content (same as what's displayed in TUI)
		if cached, ok := a.messageCache[messageID]; ok {
			// Use the cached processed message
			rendered, _ := a.renderMessageContent(cached)
			// Clean up the content for Slack (remove tview markup and ANSI)
			cleanContent := a.cleanContentForSlack(rendered)
			options.ProcessedContent = cleanContent
		} else {
			// If not cached, load the message
			message, err := a.Client.GetMessageWithContent(messageID)
			if err == nil {
				a.messageCache[messageID] = message
				rendered, _ := a.renderMessageContent(message)
				cleanContent := a.cleanContentForSlack(rendered)
				options.ProcessedContent = cleanContent
			}
		}
	}

	// Show loading message asynchronously
	go func() {
		a.GetErrorHandler().ShowInfo(a.ctx, "Forwarding email to Slack...")
	}()

	// Forward the email in a goroutine to avoid blocking UI
	go func() {
		err := slackService.ForwardEmail(a.ctx, messageID, options)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to forward email to Slack: %v", err))
			return
		}
		
		// Show success message
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("📧 Email forwarded to #%s", options.ChannelName))
	}()
}

// forwardBulkEmailsToSlack forwards multiple selected emails to Slack
func (a *App) forwardBulkEmailsToSlack(options services.SlackForwardOptions) {
	if !a.bulkMode || len(a.selected) == 0 {
		a.GetErrorHandler().ShowError(a.ctx, "No messages selected for bulk Slack forwarding")
		return
	}

	slackService := a.GetSlackService()
	if slackService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Slack service not available")
		return
	}

	// Snapshot selection to avoid race conditions
	messageIDs := make([]string, 0, len(a.selected))
	for id := range a.selected {
		messageIDs = append(messageIDs, id)
	}

	messageCount := len(messageIDs)

	// Show initial progress
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Forwarding %d emails to #%s...", messageCount, options.ChannelName))
	}()

	// Process bulk forwarding in background
	go func() {
		failed := 0
		
		for i, messageID := range messageIDs {
			// Update progress
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Forwarding %d/%d to #%s...", i+1, messageCount, options.ChannelName))
			
			// Create a copy of options for this specific message
			messageOptions := options
			
			// For "full" format, get the TUI-processed content for this specific message
			if options.FormatStyle == "full" {
				if cached, ok := a.messageCache[messageID]; ok {
					// Use the cached processed message
					rendered, _ := a.renderMessageContent(cached)
					cleanContent := a.cleanContentForSlack(rendered)
					messageOptions.ProcessedContent = cleanContent
				} else {
					// If not cached, load the message
					message, err := a.Client.GetMessageWithContent(messageID)
					if err == nil {
						a.messageCache[messageID] = message
						rendered, _ := a.renderMessageContent(message)
						cleanContent := a.cleanContentForSlack(rendered)
						messageOptions.ProcessedContent = cleanContent
					}
				}
			}
			
			err := slackService.ForwardEmail(a.ctx, messageID, messageOptions)
			if err != nil {
				failed++
				// Continue with other messages even if one fails
				continue
			}
		}

		// Final status
		a.GetErrorHandler().ClearProgress()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Forwarded %d emails to #%s", messageCount, options.ChannelName))
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Forwarded %d emails to #%s with %d failure(s)", messageCount-failed, options.ChannelName, failed))
		}

		// Exit bulk mode after successful operation (following other bulk operations pattern)
		a.QueueUpdateDraw(func() {
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
			}
		})
	}()
}

// cleanContentForSlack removes tview markup and ANSI codes from content for Slack
func (a *App) cleanContentForSlack(content string) string {
	// Remove tview color markup like [red], [yellow], etc.
	// Simple regex-based cleanup for common tview markup patterns
	cleaned := content
	
	// Remove tview color tags [color] and [color:background]
	colorRegex := regexp.MustCompile(`\[[a-zA-Z0-9#:]*\]`)
	cleaned = colorRegex.ReplaceAllString(cleaned, "")
	
	// Remove ANSI escape sequences if any
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	cleaned = ansiRegex.ReplaceAllString(cleaned, "")
	
	return strings.TrimSpace(cleaned)
}