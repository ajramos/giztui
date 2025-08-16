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
		a.showStatusMessage("💬 Slack panel hidden")
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

		go a.forwardEmailToSlack(messageID, options)
		
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

				go a.forwardEmailToSlack(messageID, options)
				
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
	a.slackView.AddItem(spacer, 1, 0, false)
	a.slackView.AddItem(instructions, 1, 0, false)

	// Return channelList for focus setting
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

	// Show loading message
	a.GetErrorHandler().ShowInfo(a.ctx, "Forwarding email to Slack...")

	// Forward the email
	err := slackService.ForwardEmail(a.ctx, messageID, options)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to forward email to Slack: %v", err))
		return
	}

	// Show success message
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("📧 Email forwarded to #%s", options.ChannelName))
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