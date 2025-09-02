package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/mattn/go-runewidth"
	gmailapi "google.golang.org/api/gmail/v1"
)

// Gmail system folder constants
const (
	GMAIL_INBOX = "INBOX"
	GMAIL_TRASH = "TRASH"
	GMAIL_SPAM  = "SPAM"
	GMAIL_SENT  = "SENT"
	// Archive is represented by absence of INBOX label
)

// labelItem represents a label option in the move/manage panel
type labelItem struct {
	id, name string
	applied  bool
}

// System folder configuration for move panel
type systemFolder struct {
	id        string                     // Gmail label ID or special value
	name      string                     // Display name with icon
	icon      string                     // Icon only (for logging)
	condition func(labels []string) bool // When to show this option
}

// getSystemFolders returns the available system folders based on current message labels
func (a *App) getSystemFolders(messageLabels []string) []systemFolder {
	labelSet := make(map[string]bool)
	for _, label := range messageLabels {
		labelSet[label] = true
	}

	folders := []systemFolder{}

	// Show Inbox if message is not in inbox (archived, spam, trash, etc.)
	if !labelSet[GMAIL_INBOX] {
		folders = append(folders, systemFolder{
			id:   GMAIL_INBOX,
			name: "📥 Inbox",
			icon: "📥",
			condition: func(labels []string) bool {
				for _, l := range labels {
					if l == GMAIL_INBOX {
						return false
					}
				}
				return true
			},
		})
	}

	// Show Trash if message is not in trash
	if !labelSet[GMAIL_TRASH] {
		folders = append(folders, systemFolder{
			id:   GMAIL_TRASH,
			name: "🗑️ Trash",
			icon: "🗑️",
			condition: func(labels []string) bool {
				for _, l := range labels {
					if l == GMAIL_TRASH {
						return false
					}
				}
				return true
			},
		})
	}

	// Show Archive if message is currently in inbox
	if labelSet[GMAIL_INBOX] {
		folders = append(folders, systemFolder{
			id:   "REMOVE_INBOX", // Special identifier for archive operation
			name: "📁 Archive",
			icon: "📁",
			condition: func(labels []string) bool {
				for _, l := range labels {
					if l == GMAIL_INBOX {
						return true
					}
				}
				return false
			},
		})
	}

	// Show Spam if message is not in spam
	if !labelSet[GMAIL_SPAM] {
		folders = append(folders, systemFolder{
			id:   GMAIL_SPAM,
			name: "🚫 Spam",
			icon: "🚫",
			condition: func(labels []string) bool {
				for _, l := range labels {
					if l == GMAIL_SPAM {
						return false
					}
				}
				return true
			},
		})
	}

	if a.logger != nil {
		folderNames := make([]string, len(folders))
		for i, f := range folders {
			folderNames[i] = f.icon + " " + strings.TrimPrefix(f.name, f.icon+" ")
		}
		a.logger.Printf("getSystemFolders: messageLabels=%v, returning folders=[%s]", messageLabels, strings.Join(folderNames, ", "))
	}

	return folders
}

// buildMoveOptions creates a combined list of system folders and regular labels for the move panel
func (a *App) buildMoveOptions(messageID string) ([]labelItem, error) {
	if a.logger != nil {
		a.logger.Printf("buildMoveOptions: building options for messageID=%s", messageID)
	}

	// Get message details to determine current labels
	msg, err := a.Client.GetMessage(messageID)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("buildMoveOptions: failed to get message: %v", err)
		}
		return nil, fmt.Errorf("failed to get message: %v", err)
	}

	// Get all available labels
	labels, err := a.Client.ListLabels()
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("buildMoveOptions: failed to get labels: %v", err)
		}
		return nil, fmt.Errorf("failed to get labels: %v", err)
	}

	// Create options list
	options := []labelItem{}

	// Add system folders at the top
	systemFolders := a.getSystemFolders(msg.LabelIds)
	for _, folder := range systemFolders {
		options = append(options, labelItem{
			id:      folder.id,
			name:    folder.name,
			applied: false, // System folders are never "applied" - they're destinations
		})
	}

	if a.logger != nil && len(systemFolders) > 0 {
		folderNames := make([]string, len(systemFolders))
		for i, f := range systemFolders {
			folderNames[i] = strings.TrimPrefix(f.name, f.icon+" ")
		}
		a.logger.Printf("buildMoveOptions: added %d system folders: [%s]", len(systemFolders), strings.Join(folderNames, ", "))
	}

	// Add regular labels (filtered and sorted)
	currentLabels := make(map[string]bool)
	for _, lid := range msg.LabelIds {
		currentLabels[lid] = true
	}

	filtered := a.filterAndSortLabels(labels)
	for _, label := range filtered {
		options = append(options, labelItem{
			id:      label.Id,
			name:    label.Name,
			applied: currentLabels[label.Id],
		})
	}

	if a.logger != nil {
		a.logger.Printf("buildMoveOptions: returning %d total options (%d system folders + %d regular labels)",
			len(options), len(systemFolders), len(filtered))
	}

	return options, nil
}

// executeLabelAdd adds a label to the current message
func (a *App) executeLabelAdd(args []string) {
	labelName := strings.Join(args, " ")
	if labelName == "" {
		a.showError("Label name cannot be empty")
		return
	}

	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}

	go func() {
		label, err := a.Client.CreateLabel(labelName)
		if err != nil {
			labels, err := a.Client.ListLabels()
			if err != nil {
				a.showError(fmt.Sprintf("❌ Error creating/finding label: %v", err))
				return
			}
			for _, l := range labels {
				if l.Name == labelName {
					label = l
					break
				}
			}
			if label == nil {
				a.showError(fmt.Sprintf("❌ Error creating label: %v", err))
				return
			}
		}
		// Use LabelService for undo support
		_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
		if err := labelService.ApplyLabel(a.ctx, messageID, label.Id); err != nil {
			a.showError(fmt.Sprintf("❌ Error applying label: %v", err))
			return
		}

		// Update local cache and refresh display
		a.updateCachedMessageLabels(messageID, label.Id, true)
		a.updateMessageCacheLabels(messageID, labelName, true)
		a.refreshMessageContent(messageID)

		// Refresh message list to show updated label chips immediately
		a.QueueUpdateDraw(func() {
			a.reformatListItems()
		})

		go func() {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🔖 Applied label: %s", labelName))
		}()
	}()
}

// executeLabelRemove removes a label from the current message
func (a *App) executeLabelRemove(args []string) {
	labelName := strings.Join(args, " ")
	if labelName == "" {
		a.showError("Label name cannot be empty")
		return
	}
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("No message selected")
		return
	}
	go func() {
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError(fmt.Sprintf("❌ Error loading labels: %v", err))
			return
		}
		var labelID string
		for _, l := range labels {
			if l.Name == labelName {
				labelID = l.Id
				break
			}
		}
		if labelID == "" {
			a.showError(fmt.Sprintf("❌ Label not found: %s", labelName))
			return
		}
		// Use LabelService for undo support
		_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
		if err := labelService.RemoveLabel(a.ctx, messageID, labelID); err != nil {
			a.showError(fmt.Sprintf("❌ Error removing label: %v", err))
			return
		}
		a.showStatusMessage(fmt.Sprintf("🔖  Removed label: %s", labelName))
	}()
}

// manageLabels opens the labels management view for the currently selected message
func (a *App) manageLabels() {

	// Toggle contextual panel like AI Summary
	if a.isLabelsPickerActive() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.setActivePicker(PickerNone)
		a.SetFocus(a.views["text"])
		a.currentFocus = "text"
		a.updateFocusIndicators("text")
		go func() {
			a.GetErrorHandler().ShowInfo(a.ctx, "🙈 Labels hidden")
		}()
		return
	}

	messageID := a.getCurrentMessageID()
	if messageID == "" {
		go func() {
			a.GetErrorHandler().ShowError(a.ctx, "❌ No message selected")
		}()
		return
	}

	// Ensure message content is shown without stealing focus
	a.showMessageWithoutFocus(messageID)

	// Show panel and load quick view
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerLabels)
	a.labelsExpanded = false
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")

	a.populateLabelsQuickView(messageID)
}

// showMessageLabelsView displays labels for a specific message
func (a *App) showMessageLabelsView(labels []*gmailapi.Label, message *gmailapi.Message) {
	// Backwards-compat modal; keep for command bar or future use
	// Create labels list view
	labelsList := tview.NewList()
	labelsList.SetBorder(true)
	labelsList.SetTitle(" 🔖  Message Labels ")

	// Apply component-specific selection colors
	labelColors := a.GetComponentColors("labels")
	labelsList.SetMainTextColor(labelColors.Text.Color())
	labelsList.SetSelectedTextColor(labelColors.Background.Color())   // Use background for selected text (inverse)
	labelsList.SetSelectedBackgroundColor(labelColors.Accent.Color()) // Use accent for selection highlight

	// Get current message labels
	currentLabels := make(map[string]bool)
	if message.LabelIds != nil {
		for _, labelID := range message.LabelIds {
			currentLabels[labelID] = true
		}
	}

	// Extract subject for display
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Partition applied vs not-applied and sort each group; applied first
	applied, notApplied := a.partitionAndSortLabels(labels, currentLabels)
	for _, label := range append(applied, notApplied...) {
		// Store label info for the callback (avoid capturing loop vars directly)
		labelID := label.Id
		labelName := label.Name

		// Determine current applied state
		isApplied := currentLabels[labelID]

		// Create display text with label info and status (no secondary text)
		var displayText string
		if isApplied {
			displayText = fmt.Sprintf("✅ %s", labelName)
		} else {
			displayText = fmt.Sprintf("○ %s", labelName)
		}

		labelsList.AddItem(displayText, "", 0, func() {
			// Capture current index and state at click time
			index := labelsList.GetCurrentItem()
			currentlyApplied := currentLabels[labelID]
			// Async toggle
			a.toggleLabelForMessage(message.Id, labelID, labelName, currentlyApplied, func(newApplied bool, err error) {
				if err != nil {
					return
				}
				// Update local state map
				currentLabels[labelID] = newApplied
				// Update UI immediately
				newText := fmt.Sprintf("○ %s", labelName)
				if newApplied {
					newText = fmt.Sprintf("✅ %s", labelName)
				}
				a.QueueUpdateDraw(func() {
					labelsList.SetItemText(index, newText, "")
				})
				// Update cached meta for main list (for UNREAD/star, etc.)
				a.updateCachedMessageLabels(message.Id, labelID, newApplied)
			})
		})
	}

	// Add "Create new label" option at the end
	labelsList.AddItem("➕ Create new label", "Press Enter to create", 0, func() {
		a.createNewLabelFromView()
	})

	// Ensure first item is selected to enable immediate arrow navigation
	if labelsList.GetItemCount() > 0 {
		labelsList.SetCurrentItem(0)
	}

	// Set up key bindings for the labels view
	labelsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Return to main view
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			// Refresh currently displayed message content to reflect new labels
			go a.refreshMessageContent(message.Id)
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'n' {
				// Create new label
				a.createNewLabelFromView()
				return nil
			}
			if event.Rune() == 'r' {
				// Refresh labels view
				go a.manageLabels()
				return nil
			}
		}
		return event
	})

	// Create the labels view page
	labelsView := tview.NewFlex().SetDirection(tview.FlexRow)

	// Title with message subject
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText(fmt.Sprintf("🔖  Labels for: %s", subject))
	title.SetTextColor(a.GetComponentColors("labels").Title.Color())
	title.SetBorder(true)

	// Instructions
	instructions := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	instructions.SetText("Enter: Toggle label | n: Create new label | r: Refresh | ESC: Back")
	instructions.SetTextColor(a.GetComponentColors("general").Text.Color())

	labelsView.AddItem(title, 3, 0, false)
	labelsView.AddItem(labelsList, 0, 1, true)
	labelsView.AddItem(instructions, 2, 0, false)

	// Add labels view to pages
	a.Pages.AddPage("messageLabels", labelsView, true, true)
	a.Pages.SwitchToPage("messageLabels")
	a.SetFocus(labelsList)
}

// populateLabelsQuickView renders current labels + quick actions in the side panel
func (a *App) populateLabelsQuickView(messageID string) {
	if a.logger != nil {
		a.logger.Printf("populateLabelsQuickView: starting for messageID=%s, bulkMode=%v, selectedCount=%d", messageID, a.bulkMode, len(a.selected))
	}
	go func() {
		if a.logger != nil {
			a.logger.Printf("populateLabelsQuickView: fetching message details for messageID=%s", messageID)
		}
		msg, err := a.Client.GetMessage(messageID)
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("populateLabelsQuickView: FAILED to get message: %v", err)
			}
			a.showError("❌ Error loading message")
			return
		}
		if a.logger != nil {
			a.logger.Printf("populateLabelsQuickView: fetching labels list")
		}
		labels, err := a.Client.ListLabels()
		if err != nil {
			if a.logger != nil {
				a.logger.Printf("populateLabelsQuickView: FAILED to get labels: %v", err)
			}
			a.showError("❌ Error loading labels")
			return
		}
		if a.logger != nil {
			a.logger.Printf("populateLabelsQuickView: got %d labels, building UI", len(labels))
		}
		// Build quick view UI off-thread then apply
		current := make(map[string]bool)
		for _, lid := range msg.LabelIds {
			current[lid] = true
		}
		applied, notApplied := a.partitionAndSortLabels(labels, current)

		body := tview.NewList().ShowSecondaryText(false)
		body.SetBorder(false)

		// Apply component-specific selection colors
		labelColors := a.GetComponentColors("labels")
		body.SetMainTextColor(labelColors.Text.Color())
		body.SetSelectedTextColor(labelColors.Background.Color())   // Use background for selected text (inverse)
		body.SetSelectedBackgroundColor(labelColors.Accent.Color()) // Use accent for selection highlight
		// Helper to pad emoji to width 2 for alignment across fonts
		padIcon := func(icon string) string {
			if runewidth.StringWidth(icon) < 2 {
				return icon + " "
			}
			return icon
		}
		// Current labels first (checked)
		for _, l := range applied {
			name := l.Name
			lid := l.Id
			body.AddItem("✅ "+name, "Enter: toggle off", 0, func() {
				// Check if we need to apply to bulk selection
				if a.bulkMode && len(a.selected) > 0 {
					// Apply label to all selected messages (remove since currently applied)
					go a.applyLabelToBulkSelection(lid, name, true)
				} else {
					// Single message label toggle
					a.toggleLabelForMessage(messageID, lid, name, true, func(newApplied bool, err error) {
						if err == nil {
							a.updateCachedMessageLabels(messageID, lid, newApplied)
							a.updateMessageCacheLabels(messageID, name, newApplied)
							a.populateLabelsQuickView(messageID)
							a.refreshMessageContent(messageID)
							// Refresh message list to show updated label chips
							a.QueueUpdateDraw(func() {
								a.reformatListItems()
							})
						}
					})
				}
			})
		}
		// Quick actions: first N from notApplied
		maxQuick := 6
		for i, l := range notApplied {
			if i >= maxQuick {
				break
			}
			name := l.Name
			lid := l.Id
			body.AddItem("○ "+name, "Enter: apply", 0, func() {
				// Check if we need to apply to bulk selection
				if a.bulkMode && len(a.selected) > 0 {
					// Apply label to all selected messages (add since not currently applied)
					go a.applyLabelToBulkSelection(lid, name, false)
				} else {
					// Single message label toggle
					a.toggleLabelForMessage(messageID, lid, name, false, func(newApplied bool, err error) {
						if err == nil {
							a.updateCachedMessageLabels(messageID, lid, newApplied)
							a.updateMessageCacheLabels(messageID, name, newApplied)
							a.populateLabelsQuickView(messageID)
							a.refreshMessageContent(messageID)
							// Refresh message list to show updated label chips
							a.QueueUpdateDraw(func() {
								a.reformatListItems()
							})
						}
					})
				}
			})
		}
		// Actions
		body.AddItem(padIcon("🔍")+" Browse all labels…", "Enter to apply 1st match | Esc to back", 0, func() {
			a.expandLabelsBrowse(messageID)
		})
		body.AddItem(padIcon("➕")+" Add custom label…", "Create or apply", 0, func() {
			a.labelsExpanded = true // prevent quick view from repainting over input
			go a.addCustomLabelInline(messageID)
		})
		body.AddItem(padIcon("📝")+" Edit existing label…", "Rename a label", 0, func() {
			a.browseLabelForEdit(messageID)
		})
		body.AddItem(padIcon("🗑")+" Remove existing label…", "Delete a label", 0, func() {
			a.browseLabelForRemove(messageID)
		})

		// Capture ESC in quick view to close panel (hint shown in footer of subpanels)
		body.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
			if e.Key() == tcell.KeyEscape {
				if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
					split.ResizeItem(a.labelsView, 0, 0)
				}
				a.setActivePicker(PickerNone)
				// Also exit bulk mode if it was active
				if a.bulkMode {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.refreshTableDisplay()
					// CRITICAL: Clear progress asynchronously to avoid ESC deadlock
					go func() {
						a.GetErrorHandler().ClearProgress()
					}()
					if tbl, ok := a.views["list"].(*tview.Table); ok {
						tbl.SetSelectedStyle(a.getSelectionStyle())
					}
				}
				// Restore list navigation
				if l, ok := a.views["list"].(*tview.Table); ok {
					l.SetInputCapture(nil)
				}
				a.SetFocus(a.views["text"])
				a.currentFocus = "text"
				a.updateFocusIndicators("text")
				return nil
			}
			// Keep focus anchored in labels list when using Up/Down
			if e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown {
				a.currentFocus = "labels"
				a.updateFocusIndicators("labels")
				return e
			}
			return e
		})

		container := tview.NewFlex().SetDirection(tview.FlexRow)
		bgColor := a.GetComponentColors("labels").Background.Color()
		container.SetBorder(true)
		container.SetTitle(" 🔖  Message Labels ")
		container.SetTitleColor(a.GetComponentColors("labels").Title.Color())
		container.SetBackgroundColor(bgColor)

		// Set background on child components as well
		body.SetBackgroundColor(bgColor)

		container.AddItem(body, 0, 1, true)
		// Footer hint: quick view uses ESC to close panel
		footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
		footer.SetText(" Esc to back ")
		footer.SetTextColor(a.GetComponentColors("general").Text.Color())
		footer.SetBackgroundColor(bgColor)
		container.AddItem(footer, 1, 0, false)

		if a.logger != nil {
			a.logger.Printf("populateLabelsQuickView: about to call QueueUpdateDraw to update UI")
		}
		a.QueueUpdateDraw(func() {
			if a.logger != nil {
				a.logger.Printf("populateLabelsQuickView: inside QueueUpdateDraw callback")
			}
			// If user navigated to an expanded subpanel (browse/create), do not overwrite it
			if a.labelsExpanded {
				return
			}
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				// replace labelsView item with new container
				split.RemoveItem(a.labelsView)
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, false)
			}
			// While labels son visibles, solo tragamos flechas en la lista
			// when the current focus is in labels. If the user changes with
			// Tab a la lista, las flechas deben funcionar normalmente.
			if l, ok := a.views["list"].(*tview.Table); ok {
				l.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
					if a.isLabelsPickerActive() && a.currentFocus == "labels" {
						switch ev.Key() {
						case tcell.KeyUp, tcell.KeyDown, tcell.KeyPgUp, tcell.KeyPgDn, tcell.KeyHome, tcell.KeyEnd:
							return nil
						}
					}
					return ev
				})
			}
			// Solo forzar foco si ya estamos en labels (toggle inicial)
			if a.currentFocus == "labels" {
				a.SetFocus(body)
				a.updateFocusIndicators("labels")
				// Preselect first label item for proper arrow navigation
				if body.GetItemCount() > 0 {
					body.SetCurrentItem(0)
				}
			}
		})
	}()
}

// expandLabelsBrowse shows full list with search inside the side panel
func (a *App) expandLabelsBrowse(messageID string) {
	a.expandLabelsBrowseWithMode(messageID, false)
}

// expandLabelsBrowseWithMode shows full list with search inside the side panel.
// If moveMode is true, selecting a label will move the message (apply + archive)
// and then close the panel.
func (a *App) expandLabelsBrowseWithMode(messageID string, moveMode bool) {
	a.labelsExpanded = true
	// Get theme colors for labels component
	labelColors := a.GetComponentColors("labels")

	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(labelColors.Title.Color()).
		SetFieldBackgroundColor(labelColors.Background.Color()).
		SetFieldTextColor(labelColors.Text.Color())
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	// Apply theme colors to list component
	list.SetMainTextColor(labelColors.Text.Color())
	list.SetSelectedTextColor(labelColors.Background.Color())   // Use background for selected text (inverse)
	list.SetSelectedBackgroundColor(labelColors.Accent.Color()) // Use accent for selection highlight

	// Loader
	var all []labelItem
	var visible []labelItem
	var reload func(filter string)
	reload = func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, it := range all {
			if filter != "" && !strings.Contains(strings.ToLower(it.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, it)

			// Handle display differently for system folders vs regular labels
			var display string
			if moveMode && (strings.HasPrefix(it.name, "📥") || strings.HasPrefix(it.name, "🗑️") ||
				strings.HasPrefix(it.name, "📁") || strings.HasPrefix(it.name, "🚫")) {
				// System folders: show icon and name directly (no toggle indicators)
				display = it.name
			} else {
				// Regular labels: show toggle indicators
				display = "○ " + it.name
				if it.applied {
					display = "✅ " + it.name
				}
			}
			id := it.id
			name := it.name
			applied := it.applied

			// Set appropriate secondary text based on mode and item type
			var secondaryText string
			if moveMode {
				if strings.HasPrefix(it.name, "📥") || strings.HasPrefix(it.name, "🗑️") ||
					strings.HasPrefix(it.name, "📁") || strings.HasPrefix(it.name, "🚫") {
					secondaryText = "Enter: move here"
				} else {
					secondaryText = "Enter: move to label"
				}
			} else {
				secondaryText = "Enter: toggle"
			}

			list.AddItem(display, secondaryText, 0, func() {
				if a.logger != nil {
					a.logger.Printf("📋 LIST ITEM CALLBACK TRIGGERED: id='%s', name='%s', moveMode=%v", id, name, moveMode)
				}
				if !moveMode {
					// Check if we need to apply to bulk selection
					if a.bulkMode && len(a.selected) > 0 {
						// Apply label to all selected messages
						go a.applyLabelToBulkSelection(id, name, applied)
					} else {
						// Single message label toggle
						a.toggleLabelForMessage(messageID, id, name, applied, func(newApplied bool, err error) {
							if err == nil {
								// Update local model then rerender
								for i := range all {
									if all[i].id == id {
										all[i].applied = newApplied
										break
									}
								}
								a.updateCachedMessageLabels(messageID, id, newApplied)
								a.updateMessageCacheLabels(messageID, name, newApplied)
								reload(strings.TrimSpace(input.GetText()))
								a.refreshMessageContent(messageID)
							}
						})
					}
					return
				}
				// Move mode: handle system folders and regular labels differently
				go func() {
					if a.logger != nil {
						a.logger.Printf("🚀 MOVE OPERATION GOROUTINE STARTED: id='%s', name='%s'", id, name)
					}
					// Construir conjunto de mensajes a mover
					idsToMove := []string{messageID}
					if a.bulkMode && len(a.selected) > 0 {
						idsToMove = idsToMove[:0]
						for sid := range a.selected {
							idsToMove = append(idsToMove, sid)
						}
					}

					failed := 0
					var operationName string

					// Get services for undo support - use proper move function
					emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

					// Handle system folders vs regular labels
					if a.logger != nil {
						a.logger.Printf("🔍 SWITCH DEBUG: About to switch on id='%s', name='%s'", id, name)
						a.logger.Printf("🔍 Comparing with GMAIL_INBOX='%s', GMAIL_TRASH='%s', GMAIL_SPAM='%s'", GMAIL_INBOX, GMAIL_TRASH, GMAIL_SPAM)
					}
					switch id {
					case GMAIL_INBOX:
						// Move to Inbox: Use undo-aware system folder move
						operationName = "Inbox"
						if a.logger != nil {
							a.logger.Printf("🔥 INBOX MOVE OPERATION STARTED - User triggered move to inbox with %d messages", len(idsToMove))
						}
						for _, mid := range idsToMove {
							if a.logger != nil {
								a.logger.Printf("=== INBOX MOVE TEST LOG ===")
								a.logger.Printf("[UI DEBUG] About to call emailService.MoveToSystemFolder for message %s to INBOX", mid)
								a.logger.Printf("Logger is working! File path should be ~/.config/giztui/giztui.log")
							} else {
								// Fallback logging if logger is nil
								fmt.Printf("WARNING: a.logger is nil! Cannot log to file.\n")
							}
							if err := emailService.MoveToSystemFolder(a.ctx, mid, GMAIL_INBOX, "Inbox"); err != nil {
								failed++
								if a.logger != nil {
									a.logger.Printf("[UI ERROR] Failed to move message %s to Inbox: %v", mid, err)
								}
							} else {
								if a.logger != nil {
									a.logger.Printf("[UI SUCCESS] Successfully moved message %s to Inbox", mid)
								}
							}
						}

					case GMAIL_TRASH:
						// Move to Trash: Use undo-aware system folder move
						operationName = "Trash"
						for _, mid := range idsToMove {
							if err := emailService.MoveToSystemFolder(a.ctx, mid, GMAIL_TRASH, "Trash"); err != nil {
								failed++
								if a.logger != nil {
									a.logger.Printf("Failed to move message %s to Trash: %v", mid, err)
								}
							}
						}

					case GMAIL_SPAM:
						// Move to Spam: Use undo-aware system folder move
						operationName = "Spam"
						for _, mid := range idsToMove {
							if err := emailService.MoveToSystemFolder(a.ctx, mid, GMAIL_SPAM, "Spam"); err != nil {
								failed++
								if a.logger != nil {
									a.logger.Printf("Failed to move message %s to Spam: %v", mid, err)
								}
							}
						}

					case "REMOVE_INBOX":
						// Archive: Remove INBOX label (ArchiveMessage handles both label removal and undo)
						operationName = "Archive"
						for _, mid := range idsToMove {
							// Use ArchiveMessage which removes INBOX label and records undo action
							if err := emailService.ArchiveMessage(a.ctx, mid); err != nil {
								failed++
								if a.logger != nil {
									a.logger.Printf("Failed to archive message %s: %v", mid, err)
								}
							}
						}

					default:
						// Regular label: Apply label and archive (original behavior)
						operationName = name
						for _, mid := range idsToMove {
							if err := labelService.ApplyLabel(a.ctx, mid, id); err != nil {
								failed++
							}
							// Use ArchiveMessageAsMove to record proper move undo action
							if err := emailService.ArchiveMessageAsMove(a.ctx, mid, id, name); err != nil {
								failed++
							}
						}
					}

					if a.logger != nil {
						a.logger.Printf("Move operation completed: %s, failed=%d/%d", operationName, failed, len(idsToMove))
					}
					// Simplified UI update to avoid complex operations that might hang
					a.QueueUpdateDraw(func() {
						// Close the panel first
						if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
							split.ResizeItem(a.labelsView, 0, 0)
						}
						a.setActivePicker(PickerNone)
						a.labelsExpanded = false

						// Exit bulk mode
						a.selected = make(map[string]bool)
						a.bulkMode = false
						a.refreshTableDisplay()

						// Restore focus
						a.SetFocus(a.views["list"])
						a.currentFocus = "list"
						a.updateFocusIndicators("list")
					})

					// Do complex operations outside QueueUpdateDraw to avoid hanging
					go func() {
						time.Sleep(100 * time.Millisecond) // Let UI update complete first

						// CRITICAL FIX: Perform all data structure updates within a single UI operation
						// to prevent race conditions between a.ids and a.messagesMeta
						a.QueueUpdateDraw(func() {
							// Remove messages from internal data structures
							rm := make(map[string]bool, len(idsToMove))
							for _, mid := range idsToMove {
								rm[mid] = true
							}

							// Use the existing thread-safe helper that removes from both a.ids and table
							a.removeIDsFromCurrentList(idsToMove)

							// The removeIDsFromCurrentList function handles:
							// - Removing from a.ids
							// - Removing from a.messagesMeta
							// - Updating the table view
							// - Adjusting selection and content
							// - Updating title count
							// All operations are synchronized within this UI thread
						})
					}()

					// Simple status update without complex threading
					if len(idsToMove) <= 1 && failed == 0 {
						a.showStatusMessage("📦 Moved to: " + operationName)
					} else {
						successCount := len(idsToMove) - failed
						if failed == 0 {
							a.showStatusMessage(fmt.Sprintf("📦 Moved %d message(s) to %s", successCount, operationName))
						} else {
							a.showStatusMessage(fmt.Sprintf("📦 Moved %d/%d message(s) to %s", successCount, len(idsToMove), operationName))
						}
					}
				}()
			})
		}
	}

	go func() {
		if moveMode {
			// Use the new system folders + labels builder for move mode
			options, err := a.buildMoveOptions(messageID)
			if err != nil {
				a.showError(fmt.Sprintf("❌ Error loading move options: %v", err))
				return
			}
			all = options
		} else {
			// Original logic for non-move mode (manage labels)
			msg, err := a.Client.GetMessage(messageID)
			if err != nil {
				a.showError("❌ Error loading message")
				return
			}
			labels, err := a.Client.ListLabels()
			if err != nil {
				a.showError("❌ Error loading labels")
				return
			}
			current := make(map[string]bool)
			for _, lid := range msg.LabelIds {
				current[lid] = true
			}
			filtered := a.filterAndSortLabels(labels)
			all = make([]labelItem, 0, len(filtered))
			for _, l := range filtered {
				all = append(all, labelItem{l.Id, l.Name, current[l.Id]})
			}
		}

		// CRITICAL FIX: Set up ESC handler OUTSIDE QueueUpdateDraw to prevent deadlock
		escHandler := func() {
			if moveMode {
				// Close the panel completely - all synchronous operations
				a.labelsExpanded = false
				if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
					split.ResizeItem(a.labelsView, 0, 0)
				}
				a.setActivePicker(PickerNone)
				if a.bulkMode {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.refreshTableDisplay()
					// Use synchronous operation for list style reset
					if list, ok := a.views["list"].(*tview.Table); ok {
						list.SetSelectedStyle(a.getSelectionStyle())
					}
				}
				a.SetFocus(a.views["list"])
				a.currentFocus = "list"
				a.updateFocusIndicators("list")
				// Clear progress asynchronously to avoid deadlock
				go func() {
					a.GetErrorHandler().ClearProgress()
				}()
			} else {
				// back to quick view - synchronous operations
				a.labelsExpanded = false
				if a.bulkMode {
					a.bulkMode = false
					a.selected = make(map[string]bool)
					a.refreshTableDisplay()
					// Use synchronous operation for list style reset
					if list, ok := a.views["list"].(*tview.Table); ok {
						list.SetSelectedStyle(a.getSelectionStyle())
					}
					// Clear progress asynchronously to avoid deadlock
					go func() {
						a.GetErrorHandler().ClearProgress()
					}()
				}
				a.populateLabelsQuickView(messageID)
			}
		}

		a.QueueUpdateDraw(func() {
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					// Call the synchronous ESC handler
					escHandler()
					return
				}
				if key == tcell.KeyEnter {
					// UX shortcut: if there is at least one visible result, apply/move the first one
					if len(visible) >= 1 {
						v := visible[0]
						if !moveMode {
							// Check if we need to apply to bulk selection
							if a.bulkMode && len(a.selected) > 0 {
								// Apply label to all selected messages
								go a.applyLabelToBulkSelection(v.id, v.name, v.applied)
							} else {
								// Single message label toggle
								if !v.applied {
									a.toggleLabelForMessage(messageID, v.id, v.name, false, func(newApplied bool, err error) {
										if err == nil {
											for i := range all {
												if all[i].id == v.id {
													all[i].applied = newApplied
													break
												}
											}
											a.updateCachedMessageLabels(messageID, v.id, newApplied)
											a.updateMessageCacheLabels(messageID, v.name, newApplied)
											reload(strings.TrimSpace(input.GetText()))
											a.refreshMessageContent(messageID)
											// Refresh message list to show updated label chips
											a.QueueUpdateDraw(func() {
												a.reformatListItems()
											})
										}
									})
								} else {
									a.showStatusMessage("✔️ Label already applied: " + v.name)
								}
							}
						} else {
							// Move mode: reuse the same logic as the list callback
							go func(id, name string) {
								idsToMove := []string{messageID}
								if a.bulkMode && len(a.selected) > 0 {
									idsToMove = idsToMove[:0]
									for sid := range a.selected {
										idsToMove = append(idsToMove, sid)
									}
								}
								failed := 0

								// Debug logging for move operation
								// Starting move operation for bulk messages

								// Process messages using proper system folder handling
								if a.logger != nil {
									a.logger.Printf("🔥 SEARCH INPUT ENTER: Executing move for id='%s', name='%s'", id, name)
								}

								emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

								// Handle system folders vs regular labels (same logic as list handlers)
								switch id {
								case GMAIL_INBOX:
									for _, mid := range idsToMove {
										if err := emailService.MoveToSystemFolder(a.ctx, mid, GMAIL_INBOX, "Inbox"); err != nil {
											failed++
											if a.logger != nil {
												a.logger.Printf("Failed to move message %s to Inbox: %v", mid, err)
											}
										}
									}
								case GMAIL_TRASH:
									for _, mid := range idsToMove {
										if err := emailService.MoveToSystemFolder(a.ctx, mid, GMAIL_TRASH, "Trash"); err != nil {
											failed++
										}
									}
								case GMAIL_SPAM:
									for _, mid := range idsToMove {
										if err := emailService.MoveToSystemFolder(a.ctx, mid, GMAIL_SPAM, "Spam"); err != nil {
											failed++
										}
									}
								case "REMOVE_INBOX":
									for _, mid := range idsToMove {
										if err := emailService.ArchiveMessage(a.ctx, mid); err != nil {
											failed++
										}
									}
								default:
									// Regular label: Apply label and archive (original behavior)
									for _, mid := range idsToMove {
										if err := labelService.ApplyLabel(a.ctx, mid, id); err != nil {
											failed++
										}
										if err := emailService.ArchiveMessage(a.ctx, mid); err != nil {
											failed++
										}
									}
								}
								// Simplified UI update to avoid complex operations that might hang
								a.QueueUpdateDraw(func() {
									// Close the panel first
									if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
										split.ResizeItem(a.labelsView, 0, 0)
									}
									a.setActivePicker(PickerNone)
									a.labelsExpanded = false

									// Exit bulk mode
									a.selected = make(map[string]bool)
									a.bulkMode = false
									a.refreshTableDisplay()

									// Restore focus
									a.SetFocus(a.views["list"])
									a.currentFocus = "list"
									a.updateFocusIndicators("list")
								})

								// Do complex operations outside QueueUpdateDraw to avoid hanging
								go func() {
									time.Sleep(100 * time.Millisecond) // Let UI update complete first

									// CRITICAL FIX: Perform all data structure updates within a single UI operation
									// to prevent race conditions between a.ids and a.messagesMeta
									a.QueueUpdateDraw(func() {
										// Remove messages from internal data structures
										rm := make(map[string]bool, len(idsToMove))
										for _, mid := range idsToMove {
											rm[mid] = true
										}

										// Use the existing thread-safe helper that removes from both a.ids and table
										a.removeIDsFromCurrentList(idsToMove)

										// The removeIDsFromCurrentList function handles:
										// - Removing from a.ids
										// - Removing from a.messagesMeta
										// - Updating the table view
										// - Adjusting selection and content
										// - Updating title count
										// All operations are synchronized within this UI thread
									})
								}()

								// Simple status update without complex threading
								if len(idsToMove) <= 1 && failed == 0 {
									a.showStatusMessage("📦 Moved to: " + name)
								} else {
									a.showStatusMessage(fmt.Sprintf("📦 Moved %d message(s) to %s", len(idsToMove), name))
								}
							}(v.id, v.name)
						}
					}
					return
				}
			})
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp || e.Key() == tcell.KeyDown {
					// Redirect arrow keys to the list when in the search field
					a.SetFocus(list)
					a.currentFocus = "labels"
					a.updateFocusIndicators("labels")
					return e
				}
				return e
			})

			container := tview.NewFlex().SetDirection(tview.FlexRow)
			bgColor := labelColors.Background.Color()
			container.SetBackgroundColor(bgColor)
			container.SetBorderColor(labelColors.Border.Color())
			container.SetBorder(true)

			// Set background on child components as well
			input.SetBackgroundColor(bgColor)
			list.SetBackgroundColor(bgColor)
			titleText := " 🔖 › 🔎 Browse all labels… "
			if moveMode {
				count := 1
				if a.bulkMode && len(a.selected) > 0 {
					count = len(a.selected)
				}
				if count == 1 {
					titleText = " 📦 Move message to… "
				} else {
					titleText = fmt.Sprintf(" 📦 Move %d messages to… ", count)
				}
			}
			container.SetTitle(titleText)
			container.SetTitleColor(labelColors.Title.Color())
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)
			// Footer hint (bottom-right)
			footer := tview.NewTextView().
				SetDynamicColors(true).
				SetTextAlign(tview.AlignRight)
			if moveMode {
				footer.SetText(" Enter to move 1st match  |  Esc to cancel ")
			} else {
				footer.SetText(" Enter to apply 1st match  |  Esc to back ")
			}
			footer.SetTextColor(labelColors.Text.Color())
			footer.SetBackgroundColor(labelColors.Background.Color())
			container.AddItem(footer, 1, 0, false)

			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.RemoveItem(a.labelsView)
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
			}
			// ESC handling and Up on first item: back to search
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if a.logger != nil {
					a.logger.Printf("🎹 LIST INPUT CAPTURE: key=%d, rune=%c, currentFocus=%s", int(e.Key()), e.Rune(), a.currentFocus)
				}
				// Remove custom Enter handling to let tview's built-in mechanism handle it
				// This allows both Enter and Space to work identically via AddItem selectedFunc
				if e.Key() == tcell.KeyEscape {
					// CRITICAL FIX: Make ESC operations synchronous to prevent deadlock
					a.labelsExpanded = false
					if moveMode {
						if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
							split.ResizeItem(a.labelsView, 0, 0)
						}
						a.setActivePicker(PickerNone)
						if a.bulkMode {
							a.bulkMode = false
							a.selected = make(map[string]bool)
							a.refreshTableDisplay()
							// Use synchronous operation for list style reset
							if list, ok := a.views["list"].(*tview.Table); ok {
								list.SetSelectedStyle(a.getSelectionStyle())
							}
							// Clear progress asynchronously to avoid deadlock
							go func() {
								a.GetErrorHandler().ClearProgress()
							}()
						}
						a.SetFocus(a.views["list"])
						a.currentFocus = "list"
						a.updateFocusIndicators("list")
					} else {
						if a.bulkMode {
							a.bulkMode = false
							a.selected = make(map[string]bool)
							a.refreshTableDisplay()
							// Use synchronous operation for list style reset
							if list, ok := a.views["list"].(*tview.Table); ok {
								list.SetSelectedStyle(a.getSelectionStyle())
							}
							// Clear progress asynchronously to avoid deadlock
							go func() {
								a.GetErrorHandler().ClearProgress()
							}()
						}
						a.populateLabelsQuickView(messageID)
					}
					return nil
				}
				if e.Key() == tcell.KeyUp {
					idx := list.GetCurrentItem()
					if idx <= 0 {
						a.SetFocus(input)
						a.currentFocus = "labels"
						a.updateFocusIndicators("labels")
						return nil
					}
				}
				if a.logger != nil {
					a.logger.Printf("🎹 LIST INPUT CAPTURE: Returning event key=%d to system", int(e.Key()))
				}
				return e
			})
			reload("")
			a.SetFocus(input)
			a.currentFocus = "labels"
			a.updateFocusIndicators("labels")
		})
	}()
}

// browseLabelForEdit opens a browse-all picker to select a label to rename
func (a *App) browseLabelForEdit(messageID string) {
	a.labelsExpanded = true
	a.expandLabelsBrowseGeneric(messageID, " 📝 Select label to edit ", func(id, name string) {
		a.editLabelInline(id, name)
	})
}

// browseLabelForRemove opens a browse-all picker to select a label to delete
func (a *App) browseLabelForRemove(messageID string) {
	a.labelsExpanded = true
	a.expandLabelsBrowseGeneric(messageID, " 🗑 Select label to remove ", func(id, name string) {
		a.confirmDeleteLabel(id, name)
	})
}

// expandLabelsBrowseGeneric clones the browse-all list but calls onPick when the user confirms a label
func (a *App) expandLabelsBrowseGeneric(messageID, title string, onPick func(id, name string)) {
	// Get theme colors for labels component
	labelColors := a.GetComponentColors("labels")
	if a.logger != nil {
		a.logger.Printf("DEBUG expandLabelsBrowseGeneric: background color = %s", labelColors.Background.Color())
	}

	input := tview.NewInputField().
		SetLabel("🔍 Search: ").
		SetFieldWidth(30).
		SetLabelColor(labelColors.Title.Color()).
		SetFieldBackgroundColor(labelColors.Background.Color()).
		SetFieldTextColor(labelColors.Text.Color())
	// Set background on the input field component itself
	input.SetBackgroundColor(labelColors.Background.Color())

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(false)
	// Set background on list to prevent transparency
	list.SetBackgroundColor(labelColors.Background.Color())
	// Set selection colors like working suggested labels
	list.SetSelectedTextColor(labelColors.Background.Color())
	list.SetSelectedBackgroundColor(labelColors.Accent.Color())

	type labelItem struct{ id, name string }
	var all []labelItem
	var visible []labelItem
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]
		for _, it := range all {
			if filter != "" && !strings.Contains(strings.ToLower(it.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, it)
			id := it.id
			name := it.name
			list.AddItem(name, "Enter: pick", 0, func() {
				if a.logger != nil {
					a.logger.Printf("browseGeneric: pick via list id=%s name=%s", id, name)
				}
				go onPick(id, name)
			})
		}
	}

	go func() {
		labels, err := a.Client.ListLabels()
		if err != nil {
			a.showError("❌ Error loading labels")
			return
		}
		filtered := a.filterAndSortLabels(labels)
		all = make([]labelItem, 0, len(filtered))
		for _, l := range filtered {
			all = append(all, labelItem{l.Id, l.Name})
		}
		a.QueueUpdateDraw(func() {
			input.SetChangedFunc(func(text string) { reload(strings.TrimSpace(text)) })
			// Permitir volver al listado con flechas desde el buscador
			input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp || e.Key() == tcell.KeyPgDn || e.Key() == tcell.KeyPgUp {
					a.SetFocus(list)
					a.currentFocus = "labels"
					a.updateFocusIndicators("labels")
					return e
				}
				return e
			})
			input.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEscape {
					a.labelsExpanded = false
					a.populateLabelsQuickView(messageID)
					return
				}
				if key == tcell.KeyEnter {
					if len(visible) > 0 {
						v := visible[0]
						if a.logger != nil {
							a.logger.Printf("browseGeneric: pick via search id=%s name=%s", v.id, v.name)
						}
						go onPick(v.id, v.name)
					}
				}
			})
			container := tview.NewFlex().SetDirection(tview.FlexRow)
			container.SetBackgroundColor(labelColors.Background.Color())
			container.SetBorderColor(labelColors.Border.Color())
			container.SetBorder(true)
			container.SetTitle(title)
			container.SetTitleColor(labelColors.Title.Color())
			container.AddItem(input, 3, 0, true)
			container.AddItem(list, 0, 1, true)
			footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
			footer.SetText(" Enter to pick 1st match  |  Esc to back ")
			footer.SetTextColor(labelColors.Text.Color())
			footer.SetBackgroundColor(labelColors.Background.Color())
			container.AddItem(footer, 1, 0, false)
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				if a.labelsView != nil {
					split.RemoveItem(a.labelsView)
				}
				a.labelsView = container
				split.AddItem(a.labelsView, 0, 1, true)
				split.ResizeItem(a.labelsView, 0, 1)
			}
			// Capturar flechas en la lista: si estamos en la primera y pulsamos Arriba, volver al buscador
			list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
				if e.Key() == tcell.KeyUp {
					idx := list.GetCurrentItem()
					if idx <= 0 {
						a.SetFocus(input)
						a.currentFocus = "labels"
						a.updateFocusIndicators("labels")
						return nil
					}
				}
				return e
			})
			a.setActivePicker(PickerLabels)
			a.currentFocus = "labels"
			a.updateFocusIndicators("labels")
			a.SetFocus(input)
			reload("")
		})
	}()
}

// editLabelInline opens an inline form to rename a label
func (a *App) editLabelInline(labelID, name string) {
	// Get theme colors for labels component
	labelColors := a.GetComponentColors("labels")

	input := tview.NewInputField().
		SetLabel("New name: ").
		SetText(name).
		SetFieldWidth(30).
		SetFieldBackgroundColor(labelColors.Background.Color()).
		SetFieldTextColor(labelColors.Text.Color()).
		SetLabelColor(labelColors.Title.Color())
	// Set background on the input field component itself
	input.SetBackgroundColor(labelColors.Background.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter to rename  |  Esc to back ")
	footer.SetTextColor(labelColors.Text.Color())
	footer.SetBackgroundColor(labelColors.Background.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(labelColors.Background.Color())
	container.SetBorderColor(labelColors.Border.Color())
	container.SetBorder(true)
	container.SetTitle(" 📝 Edit label ")
	container.SetTitleColor(labelColors.Title.Color())
	container.AddItem(input, 3, 0, true)
	container.AddItem(footer, 1, 0, false)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.browseLabelForEdit(a.getCurrentMessageID())
			return
		}
		if key == tcell.KeyEnter {
			newName := strings.TrimSpace(input.GetText())
			if newName == "" || strings.EqualFold(newName, name) {
				a.browseLabelForEdit(a.getCurrentMessageID())
				return
			}
			go func(oldName, newName string) {
				if _, err := a.Client.RenameLabel(labelID, newName); err != nil {
					a.showError("❌ Error renaming label")
					return
				}
				a.QueueUpdateDraw(func() {
					a.showStatusMessage("✏️ Renamed: " + oldName + " → " + newName)
					a.labelsExpanded = false
					mid := a.getCurrentMessageID()
					a.renameLabelInMessageCache(mid, oldName, newName)
					a.populateLabelsQuickView(mid)
					a.refreshMessageContent(mid)
				})
			}(name, newName)
		}
	})
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.SetFocus(input)
		a.currentFocus = "labels"
		a.updateFocusIndicators("labels")
	})
}

// confirmDeleteLabel shows a lightweight confirmation and deletes on Enter
func (a *App) confirmDeleteLabel(labelID, name string) {
	// Get theme colors for labels component
	labelColors := a.GetComponentColors("labels")

	text := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	text.SetText("Delete label '" + name + "'? This cannot be undone.")
	text.SetBackgroundColor(labelColors.Background.Color())
	text.SetTextColor(labelColors.Text.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter to confirm  |  Esc to back ")
	footer.SetTextColor(labelColors.Text.Color())
	footer.SetBackgroundColor(labelColors.Background.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(labelColors.Background.Color())
	container.SetBorderColor(labelColors.Border.Color())
	container.SetBorder(true)
	container.SetTitle(" 🗑 Remove label ")
	container.SetTitleColor(labelColors.Title.Color())
	container.AddItem(text, 0, 1, true)
	container.AddItem(footer, 1, 0, false)
	container.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.browseLabelForRemove(a.getCurrentMessageID())
			return nil
		}
		if e.Key() == tcell.KeyEnter {
			go func(delName string) {
				if err := a.Client.DeleteLabel(labelID); err != nil {
					a.showError("❌ Error deleting label")
					return
				}
				a.QueueUpdateDraw(func() {
					a.showStatusMessage("🗑️ Deleted: " + delName)
					a.labelsExpanded = false
					mid := a.getCurrentMessageID()
					a.removeLabelNameFromMessageCache(mid, delName)
					a.populateLabelsQuickView(mid)
					a.refreshMessageContent(mid)
				})
			}(name)
			return nil
		}
		return e
	})
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.SetFocus(container)
		a.currentFocus = "labels"
		a.updateFocusIndicators("labels")
	})
}

// openMovePanel opens the side panel directly in browse-all mode to move the message
func (a *App) openMovePanel() {
	messageID := a.getCurrentMessageID()
	if messageID == "" {
		a.showError("❌ No message selected")
		return
	}
	// Ensure panel is visible
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerLabels)
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	// Open browse in move mode
	a.expandLabelsBrowseWithMode(messageID, true)
}

// openMovePanelBulk opens the move panel in bulk mode with pluralized title
func (a *App) openMovePanelBulk() {
	// If nothing selected fallback to single
	if len(a.selected) == 0 {
		a.openMovePanel()
		return
	}
	// Ensure panel visible
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerLabels)
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")
	// Use any selected message to populate current labels; choose the current focus message if selected, else any
	mid := a.getCurrentMessageID()
	if mid == "" || !a.selected[mid] {
		for id := range a.selected {
			mid = id
			break
		}
	}
	// Reuse browse with moveMode; title inside will be adjusted by len(a.selected) later if needed
	a.expandLabelsBrowseWithMode(mid, true)
}

// manageLabelsBulk opens labels management for all selected messages
func (a *App) manageLabelsBulk() {
	// If nothing selected fallback to single
	if len(a.selected) == 0 {
		a.manageLabels()
		return
	}

	// Ensure panel visible
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerLabels)
	a.currentFocus = "labels"
	a.updateFocusIndicators("labels")

	// Use any selected message to populate current labels; choose the current focus message if selected, else any
	mid := a.getCurrentMessageID()
	if mid == "" || !a.selected[mid] {
		for id := range a.selected {
			mid = id
			break
		}
	}

	// Use browse mode (not move mode) for bulk label management
	a.expandLabelsBrowseWithMode(mid, false)
}

// addCustomLabelInline prompts for a name and applies/creates it
func (a *App) addCustomLabelInline(messageID string) {
	a.labelsExpanded = true
	// Small hint so the user sees immediate feedback
	a.showStatusMessage("➕ New label…")
	if a.logger != nil {
		a.logger.Printf("addCustomLabelInline: open mid=%s", messageID)
	}
	// Get theme colors for labels component
	labelColors := a.GetComponentColors("labels")
	if a.logger != nil {
		a.logger.Printf("DEBUG addCustomLabelInline: background color = %s", labelColors.Background.Color())
	}

	// Inline input inside labels side panel (no modal)
	input := tview.NewInputField().
		SetLabel("Label name: ").
		SetFieldWidth(30).
		SetLabelColor(labelColors.Title.Color()).
		SetFieldBackgroundColor(labelColors.Background.Color()).
		SetFieldTextColor(labelColors.Text.Color())
	// Set background on the input field component itself
	input.SetBackgroundColor(labelColors.Background.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetText(" Enter to apply  |  Esc to back ")
	footer.SetTextColor(labelColors.Text.Color())
	footer.SetBackgroundColor(labelColors.Background.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(labelColors.Background.Color())
	container.SetBorderColor(labelColors.Border.Color())
	container.SetBorder(true)
	container.SetTitle(" ➕ Add custom label ")
	container.SetTitleColor(labelColors.Title.Color())
	container.AddItem(input, 3, 0, true)
	container.AddItem(footer, 1, 0, false)

	input.SetDoneFunc(func(key tcell.Key) {
		if a.logger != nil {
			a.logger.Printf("addCustomLabelInline: key=%v", key)
		}
		if key == tcell.KeyEscape {
			a.labelsExpanded = false
			a.populateLabelsQuickView(messageID)
			return
		}
		if key == tcell.KeyEnter {
			name := strings.TrimSpace(input.GetText())
			if name == "" {
				return
			}
			// Run non-blocking; update status from inside the worker to avoid blocking handler
			go func() {
				if a.logger != nil {
					a.logger.Printf("addCustomLabelInline: worker start")
				}
				a.GetErrorHandler().ShowProgress(a.ctx, "⏳ Creating/applying label…")
				if a.logger != nil {
					a.logger.Printf("addCustomLabelInline: ListLabels start")
				}
				labels, err := a.Client.ListLabels()
				if err != nil {
					if a.logger != nil {
						a.logger.Printf("addCustomLabelInline: ListLabels error: %v", err)
					}
					a.QueueUpdateDraw(func() { a.showError("❌ Error loading labels") })
					return
				}
				if a.logger != nil {
					a.logger.Printf("addCustomLabelInline: ListLabels ok (%d)", len(labels))
				}
				nameToID := make(map[string]string)
				for _, l := range labels {
					nameToID[l.Name] = l.Id
				}
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
					if a.logger != nil {
						a.logger.Printf("addCustomLabelInline: CreateLabel %q", name)
					}
					created, err := a.Client.CreateLabel(name)
					if err != nil {
						if a.logger != nil {
							a.logger.Printf("addCustomLabelInline: CreateLabel error: %v", err)
						}
						a.QueueUpdateDraw(func() { a.showError("❌ Error creating label") })
						return
					}
					id = created.Id
				}
				if a.logger != nil {
					a.logger.Printf("addCustomLabelInline: ApplyLabel mid=%s id=%s", messageID, id)
				}
				// Use LabelService for undo support
				_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
				if err := labelService.ApplyLabel(a.ctx, messageID, id); err != nil {
					if a.logger != nil {
						a.logger.Printf("addCustomLabelInline: ApplyLabel error: %v", err)
					}
					a.QueueUpdateDraw(func() { a.showError("❌ Error applying label") })
					return
				}
				a.updateCachedMessageLabels(messageID, id, true)
				// Also update full message cache labels to reflect immediately
				a.updateMessageCacheLabels(messageID, name, true)
				// Refresh message content to show updated labels
				a.refreshMessageContent(messageID)
				// CRITICAL: Separate synchronous UI updates from complex operations
				a.QueueUpdateDraw(func() {
					a.labelsExpanded = false
					// Refresh message list to show updated label chips
					a.reformatListItems()
				})

				// CRITICAL: Do complex operations outside QueueUpdateDraw to avoid deadlock
				go func() {
					a.GetErrorHandler().ShowSuccess(a.ctx, "Applied: "+name)
				}()
				go func() {
					a.GetErrorHandler().ClearProgress()
				}()

				// Refresh views asynchronously
				go func() {
					time.Sleep(50 * time.Millisecond)
					if a.logger != nil {
						a.logger.Printf("addCustomLabelInline: refreshing views asynchronously")
					}
					a.populateLabelsQuickView(messageID)
					go a.refreshMessageContent(messageID)
				}()
			}()
		}
	})

	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			// Replace a.labelsView item by index to avoid losing layout
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.setActivePicker(PickerLabels)
		a.currentFocus = "labels"
		a.updateFocusIndicators("labels")
		a.SetFocus(input)
	})
}

// toggleLabelForMessage toggles a label asynchronously and invokes onDone when finished
func (a *App) toggleLabelForMessage(messageID, labelID, labelName string, isCurrentlyApplied bool, onDone func(newApplied bool, err error)) {
	go func() {
		// Use LabelService for undo support
		_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

		if isCurrentlyApplied {
			if err := labelService.RemoveLabel(a.ctx, messageID, labelID); err != nil {
				a.showError(fmt.Sprintf("❌ Error removing label %s: %v", labelName, err))
				onDone(isCurrentlyApplied, err)
				return
			}
			go func() {
				a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🔖 Removed label: %s", labelName))
			}()
			onDone(false, nil)
			return
		}
		if err := labelService.ApplyLabel(a.ctx, messageID, labelID); err != nil {
			a.showError(fmt.Sprintf("❌ Error applying label %s: %v", labelName, err))
			onDone(isCurrentlyApplied, err)
			return
		}
		go func() {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("🔖 Applied label: %s", labelName))
		}()
		onDone(true, nil)
	}()
}

// showMessagesWithLabel shows messages that have a specific label
func (a *App) showMessagesWithLabel(labelID, labelName string) {
	// Search for messages with this label
	query := fmt.Sprintf("label:%s", labelName)
	messages, err := a.Client.SearchMessages(query, 50)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error searching messages with label %s: %v", labelName, err))
		return
	}

	// Create messages view for this label
	a.showMessagesForLabel(messages, labelName)
}

// showMessagesForLabel displays messages that have a specific label
func (a *App) showMessagesForLabel(messages []*gmailapi.Message, labelName string) {
	// Create messages list for this label
	messagesList := tview.NewList()
	messagesList.SetBorder(true)
	messagesList.SetTitle(fmt.Sprintf(" 📧 Messages with label: %s ", labelName))

	// Clear current IDs and populate with new messages
	a.ClearMessageIDs()

	for i, msg := range messages {
		a.AppendMessageID(msg.Id)

		// Get message details
		message, err := a.Client.GetMessageWithContent(msg.Id)
		if err != nil {
			messagesList.AddItem(fmt.Sprintf("⚠️  Error loading message %d", i+1), "Failed to load", 0, nil)
			continue
		}

		subject := message.Subject
		if subject == "" {
			subject = "(No subject)"
		}

		// Format the display text
		displayText := fmt.Sprintf("%s", subject)
		secondaryText := fmt.Sprintf("From: %s | %s", message.From, formatRelativeTime(message.Date))

		messagesList.AddItem(displayText, secondaryText, 0, func() {
			// Show the selected message
			if len(a.ids) > 0 {
				go a.showMessage(a.ids[0])
			}
		})
	}

	if len(messages) == 0 {
		messagesList.AddItem("No messages with this label", "", 0, nil)
	}

	// Set up key bindings
	messagesList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Return to labels view
			a.Pages.SwitchToPage("labels")
			return nil
		case 't':
			// Toggle read/unread for selected message
			go a.toggleMarkReadUnread()
			return nil
		case 'd':
			// Trash selected message
			go a.trashSelected()
			return nil
		}
		return event
	})

	// Create the messages view page
	pageName := fmt.Sprintf("messages_%s", labelName)
	a.Pages.AddPage(pageName, messagesList, true, true)
	a.Pages.SwitchToPage(pageName)
	a.SetFocus(messagesList)
}

// createNewLabelFromView creates a new label from the labels view
func (a *App) createNewLabelFromView() {
	// Create input field for new label name
	inputField := tview.NewInputField().
		SetLabel("Label name: ").
		SetFieldWidth(30).
		SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
			return len(textToCheck) > 0 && len(textToCheck) < 50
		})

	// Create modal for new label
	modal := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetText("🔖  Create New Label")
	title.SetTextColor(a.GetComponentColors("labels").Title.Color())
	title.SetBorder(true)

	instructions := tview.NewTextView().SetTextAlign(tview.AlignRight)
	instructions.SetText(" Enter to apply  |  Esc to back ")
	instructions.SetTextColor(a.GetComponentColors("general").Text.Color())

	modal.AddItem(title, 3, 0, false)
	modal.AddItem(inputField, 3, 0, true)
	modal.AddItem(instructions, 2, 0, false)

	// Handle input
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			labelName := strings.TrimSpace(inputField.GetText())
			if labelName != "" {
				go func() {
					_, err := a.Client.CreateLabel(labelName)
					if err != nil {
						// Use ErrorHandler instead of deprecated showError
						go func() {
							a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("❌ Error creating label: %v", err))
						}()
						return
					}

					// Show success message
					go func() {
						a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Created label: %s", labelName))
					}()

					// CRITICAL: Switch pages first (synchronous), then refresh (async)
					a.QueueUpdateDraw(func() {
						a.Pages.SwitchToPage("labels")
					})

					// CRITICAL: Call manageLabels outside QueueUpdateDraw to avoid deadlock
					go func() {
						// Small delay to ensure page switch completes
						time.Sleep(50 * time.Millisecond)
						a.manageLabels()
					}()
				}()
			}
		case tcell.KeyEscape:
			a.Pages.SwitchToPage("labels")
		}
	})

	// Add modal to pages
	a.Pages.AddPage("createLabel", modal, true, true)
	a.Pages.SwitchToPage("createLabel")
	a.SetFocus(inputField)
}

// deleteSelectedLabel deletes the selected label (placeholder for now)
func (a *App) deleteSelectedLabel(labelsList *tview.List) {
	a.showInfo("Delete label functionality not yet implemented")
}

// updateCachedMessageLabels updates the cached labels for a message ID
func (a *App) updateCachedMessageLabels(messageID, labelID string, applied bool) {
	// Find index
	var idx = -1
	for i, id := range a.ids {
		if id == messageID {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(a.messagesMeta) || a.messagesMeta[idx] == nil {
		return
	}
	msg := a.messagesMeta[idx]
	if applied {
		// add if not exists
		exists := false
		for _, l := range msg.LabelIds {
			if l == labelID {
				exists = true
				break
			}
		}
		if !exists {
			msg.LabelIds = append(msg.LabelIds, labelID)
		}
		// Mirror to base snapshot if in local filter
		a.updateBaseCachedMessageLabels(messageID, labelID, applied)
	} else {
		// remove
		out := msg.LabelIds[:0]
		for _, l := range msg.LabelIds {
			if l != labelID {
				out = append(out, l)
			}
		}
		msg.LabelIds = out
	}
}

// updateMessageCacheLabels updates the cached full message labels (names) so the
// rendered header reflects changes without requiring a refetch.
func (a *App) updateMessageCacheLabels(messageID, labelName string, applied bool) {
	if m, ok := a.messageCache[messageID]; ok && m != nil {
		if applied {
			// Add if missing (case-insensitive)
			exists := false
			for _, ln := range m.Labels {
				if strings.EqualFold(ln, labelName) {
					exists = true
					break
				}
			}
			if !exists {
				m.Labels = append(m.Labels, labelName)
			}
		} else {
			// Remove if present
			out := m.Labels[:0]
			for _, ln := range m.Labels {
				if !strings.EqualFold(ln, labelName) {
					out = append(out, ln)
				}
			}
			m.Labels = out
		}
	}
}

// renameLabelInMessageCache updates the cached full message label names when a label
// entity has been renamed. This avoids a refetch and ensures the header reflects
// the new name immediately for the current message.
func (a *App) renameLabelInMessageCache(messageID, oldName, newName string) {
	if m, ok := a.messageCache[messageID]; ok && m != nil {
		for i, ln := range m.Labels {
			if strings.EqualFold(ln, oldName) {
				m.Labels[i] = newName
			}
		}
	}
}

// removeLabelNameFromMessageCache removes a label name from the cached full message.
// Useful after deleting a label entity so the header updates immediately.
func (a *App) removeLabelNameFromMessageCache(messageID, name string) {
	if m, ok := a.messageCache[messageID]; ok && m != nil {
		out := m.Labels[:0]
		for _, ln := range m.Labels {
			if !strings.EqualFold(ln, name) {
				out = append(out, ln)
			}
		}
		m.Labels = out
	}
}

// Also reflect label updates into base snapshot message cache when in local filter
// (header rendering relies on names; base snapshot keeps only meta IDs, so we
// update via updateBaseCachedMessageLabels which operates on LabelIds).

// moveSelected opens the labels picker to choose a destination label, applies it, then archives the message
func (a *App) moveSelected() {
	// Get the current message ID from cached state (for undo functionality)
	messageID := a.GetCurrentMessageID()

	// Ensure cache is synchronized with cursor position
	if a.logger != nil {
		cursorID := a.getCurrentSelectedMessageID()
		// If they don't match, sync the cached state and use cursor-based ID
		if messageID != cursorID && cursorID != "" {
			messageID = cursorID
			a.SetCurrentMessageID(messageID)
		}
	}

	if messageID == "" {
		a.GetErrorHandler().ShowError(a.ctx, "No message selected")
		return
	}

	// Load available labels and message metadata
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Error loading labels: %v", err))
		return
	}
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Error getting message: %v", err))
		return
	}

	a.showMoveLabelsView(labels, message)
}

// showMoveLabelsView lets user choose a label to apply and then archives the message (move semantics)
func (a *App) showMoveLabelsView(labels []*gmailapi.Label, message *gmailapi.Message) {
	picker := tview.NewList().ShowSecondaryText(false)
	picker.SetBorder(true)
	picker.SetTitle(" 📦 Move to label ")

	// Build list of candidate labels with applied first
	curr := make(map[string]bool)
	for _, l := range message.LabelIds {
		curr[l] = true
	}
	applied, notApplied := a.partitionAndSortLabels(labels, curr)
	for _, label := range append(applied, notApplied...) {
		// Store values for closure
		labelID := label.Id
		labelName := label.Name
		picker.AddItem(labelName, "", 0, func() {
			go func() {
				// Apply label if not already present
				has := false
				for _, l := range message.LabelIds {
					if l == labelID {
						has = true
						break
					}
				}
				if !has {
					// Update cache for immediate UI feedback
					a.updateCachedMessageLabels(message.Id, labelID, true)
				}

				// Apply label and archive using dedicated move function for undo support
				emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
				if !has {
					if err := labelService.ApplyLabel(a.ctx, message.Id, labelID); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Error applying label: %v", err))
						return
					}
					a.updateCachedMessageLabels(message.Id, labelID, true)
				}
				// Use ArchiveMessageAsMove to record proper move undo action
				if err := emailService.ArchiveMessageAsMove(a.ctx, message.Id, labelID, labelName); err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Error archiving: %v", err))
					return
				}

				go func() {
					a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("📦 Moved to: %s", labelName))
				}()

				// Remove from current list (safe removal pattern) since we show INBOX only
				a.QueueUpdateDraw(func() {
					list, ok := a.views["list"].(*tview.List)
					if !ok {
						a.Pages.SwitchToPage("main")
						a.restoreFocusAfterModal()
						return
					}
					count := list.GetItemCount()
					if count == 0 {
						a.Pages.SwitchToPage("main")
						a.restoreFocusAfterModal()
						return
					}
					// Determine index of the moved message
					removeIndex := -1
					for i, id := range a.ids {
						if id == message.Id {
							removeIndex = i
							break
						}
					}
					if removeIndex < 0 || removeIndex >= count {
						removeIndex = list.GetCurrentItem()
						if removeIndex < 0 || removeIndex >= count {
							removeIndex = 0
						}
					}
					// Preselect a different index to avoid tview internal -1 during removal
					var next = -1
					if count > 1 {
						pre := removeIndex - 1
						if removeIndex == 0 {
							pre = 1
						}
						if pre < 0 {
							pre = 0
						}
						if pre >= count {
							pre = count - 1
						}
						list.SetCurrentItem(pre)
						next = pre
					}
					// Update caches using removeIndex
					if removeIndex >= 0 && removeIndex < len(a.GetMessageIDs()) {
						a.RemoveMessageIDAt(removeIndex)
					}
					if removeIndex >= 0 && removeIndex < len(a.messagesMeta) {
						a.messagesMeta = append(a.messagesMeta[:removeIndex], a.messagesMeta[removeIndex+1:]...)
					}
					// Propagate to base snapshot if in local filter
					a.baseRemoveByID(message.Id)
					// Visual removal
					if count == 1 {
						list.Clear()
						next = -1
					} else {
						if removeIndex >= 0 && removeIndex < list.GetItemCount() {
							list.RemoveItem(removeIndex)
						}
						if next >= 0 && next < list.GetItemCount() {
							list.SetCurrentItem(next)
						}
					}
					list.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
					// Update message content pane similar to trash/archive
					if text, ok := a.views["text"].(*tview.TextView); ok {
						if next >= 0 && next < len(a.ids) {
							go a.showMessageWithoutFocus(a.ids[next])
						} else {
							a.enhancedTextView.SetContent("No messages")
							text.ScrollToBeginning()
						}
					}
					// Return to main
					a.Pages.SwitchToPage("main")
					a.restoreFocusAfterModal()
				})
			}()
		})
	}

	// Basic keys
	picker.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("main")
			a.restoreFocusAfterModal()
			return nil
		}
		return event
	})

	// Container view
	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select destination label and press Enter. ESC to cancel")
	v.AddItem(title, 3, 0, false)
	v.AddItem(picker, 0, 1, true)

	a.Pages.AddPage("moveLabels", v, true, true)
	a.Pages.SwitchToPage("moveLabels")
	if picker.GetItemCount() > 0 {
		picker.SetCurrentItem(0)
	}
	a.SetFocus(picker)
}

// filterAndSortLabels filters out system labels and returns a name-sorted slice
func (a *App) filterAndSortLabels(labels []*gmailapi.Label) []*gmailapi.Label {
	filtered := make([]*gmailapi.Label, 0, len(labels))
	for _, l := range labels {
		if strings.HasPrefix(l.Id, "CATEGORY_") || l.Id == "INBOX" || l.Id == "SENT" || l.Id == "DRAFT" ||
			l.Id == "SPAM" || l.Id == "TRASH" || l.Id == "CHAT" || (strings.HasSuffix(l.Id, "_STARRED") && l.Id != "STARRED") {
			continue
		}
		filtered = append(filtered, l)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})
	return filtered
}

// partitionAndSortLabels returns two sorted slices: labels applied to current and the rest
func (a *App) partitionAndSortLabels(labels []*gmailapi.Label, current map[string]bool) ([]*gmailapi.Label, []*gmailapi.Label) {
	filtered := a.filterAndSortLabels(labels)
	applied := make([]*gmailapi.Label, 0)
	notApplied := make([]*gmailapi.Label, 0)
	for _, l := range filtered {
		if current[l.Id] {
			applied = append(applied, l)
		} else {
			notApplied = append(notApplied, l)
		}
	}
	// Already sorted by name from filterAndSortLabels; preserve order
	return applied, notApplied
}

// showAllLabelsPicker shows a list of all actionable labels to apply one to the message
func (a *App) showAllLabelsPicker(messageID string) {
	labels, err := a.Client.ListLabels()
	if err != nil {
		a.showError("❌ Error loading labels")
		return
	}
	// Get current message labels to mark applied ones
	msg, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError("❌ Error loading message")
		return
	}
	current := make(map[string]bool, len(msg.LabelIds))
	for _, lid := range msg.LabelIds {
		current[lid] = true
	}
	// Build sorted actionable labels with applied first
	applied, notApplied := a.partitionAndSortLabels(labels, current)
	all := append(applied, notApplied...)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true)
	list.SetTitle(" 🗂️  All Labels ")

	// Map name -> id
	nameToID := make(map[string]string, len(all))
	for _, l := range all {
		nameToID[l.Name] = l.Id
	}

	for _, l := range all {
		lbl := l.Name
		icon := "○ "
		if current[l.Id] {
			icon = "✅ "
		}
		display := icon + lbl
		list.AddItem(display, "", 0, func() {
			if id, ok := nameToID[lbl]; ok {
				a.applyLabelAndRefresh(messageID, id, lbl)
				go func() {
					a.GetErrorHandler().ShowSuccess(a.ctx, "✅ Applied: "+lbl)
				}()
				a.Pages.SwitchToPage("main")
				a.restoreFocusAfterModal()
			}
		})
	}

	list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape {
			a.Pages.SwitchToPage("aiLabelSuggestions")
			return nil
		}
		return e
	})

	v := tview.NewFlex().SetDirection(tview.FlexRow)
	title := tview.NewTextView().SetTextAlign(tview.AlignCenter)
	title.SetBorder(true)
	title.SetText("Select a label to apply | Enter=apply, ESC=back")
	v.AddItem(title, 3, 0, false)
	v.AddItem(list, 0, 1, true)
	a.Pages.AddPage("aiAllLabels", v, true, true)
	a.Pages.SwitchToPage("aiAllLabels")
	if list.GetItemCount() > 0 {
		list.SetCurrentItem(0)
	}
	a.SetFocus(list)
}

// applyLabelAndRefresh applies a label to a message and refreshes its content
func (a *App) applyLabelAndRefresh(messageID, labelID, labelName string) {
	// We assume that we want to apply (not toggle off), so pass isCurrentlyApplied=false
	a.toggleLabelForMessage(messageID, labelID, labelName, false, func(newApplied bool, err error) {
		if err != nil {
			return
		}
		if newApplied {
			// Keep meta cache consistent
			a.updateCachedMessageLabels(messageID, labelID, true)
			a.updateMessageCacheLabels(messageID, labelName, true)
			// Refresh message content to show updated labels
			a.refreshMessageContent(messageID)
			// Refresh message list to show updated label chips immediately (synchronous like selection change)
			a.reformatListItems()
		}
	})
}

// applyLabelToBulkSelection applies a label to all selected messages WITHOUT archiving them
func (a *App) applyLabelToBulkSelection(labelID, labelName string, currentlyApplied bool) {
	if !a.bulkMode || len(a.selected) == 0 {
		return
	}

	// Get all selected message IDs
	messageIDs := make([]string, 0, len(a.selected))
	for id := range a.selected {
		messageIDs = append(messageIDs, id)
	}

	// Debug logging
	if a.logger != nil {
		a.logger.Printf("applyLabelToBulkSelection: processing %d messages, labelID=%s, action=%s",
			len(messageIDs), labelID, func() string {
				if currentlyApplied {
					return "remove"
				} else {
					return "add"
				}
			}())
		for i, id := range messageIDs {
			a.logger.Printf("applyLabelToBulkSelection: messageIDs[%d] = %s", i, id)
		}
	}

	// Determine the action - if ANY message has the label, we remove it from all
	// If NO messages have the label, we add it to all
	action := "add"
	if currentlyApplied {
		action = "remove"
	}

	// Close the labels panel immediately (like the working move operations)
	a.QueueUpdateDraw(func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			split.ResizeItem(a.labelsView, 0, 0)
		}
		a.setActivePicker(PickerNone)
		a.labelsExpanded = false

		// Stay in bulk mode (don't exit like move operations do)
		a.SetFocus(a.views["list"])
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	})

	// Do the actual labeling work in a separate goroutine (like move operations)
	go func() {
		failed := 0
		total := len(messageIDs)

		// Use bulk label service methods for proper undo recording
		_, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
		var err error
		if action == "add" {
			if a.logger != nil {
				a.logger.Printf("applyLabelToBulkSelection: calling BulkApplyLabel for %d messages", len(messageIDs))
			}
			err = labelService.BulkApplyLabel(a.ctx, messageIDs, labelID)
		} else {
			if a.logger != nil {
				a.logger.Printf("applyLabelToBulkSelection: calling BulkRemoveLabel for %d messages", len(messageIDs))
			}
			err = labelService.BulkRemoveLabel(a.ctx, messageIDs, labelID)
		}

		if err != nil {
			if a.logger != nil {
				a.logger.Printf("applyLabelToBulkSelection: bulk operation FAILED: %v", err)
			}
			failed = len(messageIDs) // If bulk operation fails, consider all as failed
		} else {
			if a.logger != nil {
				a.logger.Printf("applyLabelToBulkSelection: bulk operation SUCCESS for all %d messages", len(messageIDs))
			}
			// Update local cache for all messages
			for _, messageID := range messageIDs {
				a.updateCachedMessageLabels(messageID, labelID, action == "add")
			}
		}

		// Update UI after all operations complete
		a.QueueUpdateDraw(func() {
			// Update the visual list to reflect label changes
			a.refreshTableDisplay()
		})

		// Show completion status using ErrorHandler (async to avoid deadlock)
		successful := total - failed
		go func() {
			if failed == 0 {
				if action == "add" {
					a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied '%s' to %d messages", labelName, total))
				} else {
					a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Removed '%s' from %d messages", labelName, total))
				}
			} else {
				if action == "add" {
					a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Applied '%s' to %d/%d messages (%d failed)", labelName, successful, total, failed))
				} else {
					a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Removed '%s' from %d/%d messages (%d failed)", labelName, successful, total, failed))
				}
			}
		}()
	}()
}
