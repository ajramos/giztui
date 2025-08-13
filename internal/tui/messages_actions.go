package tui

import (
	"fmt"

	"github.com/derailed/tview"
)

// archiveSelected archives the selected message
func (a *App) archiveSelected() {
	var messageID string
	var selectedIndex int = -1
	if a.currentFocus == "list" {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "summary" {
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else {
		a.showError("❌ Unknown focus state")
		return
	}
	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	if err := a.Client.ArchiveMessage(messageID); err != nil {
		a.showError(fmt.Sprintf("❌ Error archiving message: %v", err))
		return
	}
	a.showStatusMessage(fmt.Sprintf("📥 Archived: %s", subject))

	// Safe UI removal (preselect another index before removing)
	a.QueueUpdateDraw(func() { a.safeRemoveCurrentSelection(messageID) })
}

// trashSelected moves the selected message to trash
func (a *App) trashSelected() {
	var messageID string
	var selectedIndex int = -1

	// Get the current message ID based on focus
	if a.currentFocus == "list" {
		// Get from list view (Table)
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}

		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "text" {
		// Get from text view - read selection from Table
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}

		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}

		messageID = a.ids[selectedIndex]
	} else if a.currentFocus == "summary" {
		// From AI summary: operate on the selected row in the table
		list, ok := a.views["list"].(*tview.Table)
		if !ok {
			a.showError("❌ Could not access message list")
			return
		}
		selectedIndex, _ = list.GetSelection()
		if selectedIndex < 0 || selectedIndex >= len(a.ids) {
			a.showError("❌ No message selected")
			return
		}
		messageID = a.ids[selectedIndex]
	} else {
		a.showError("❌ Unknown focus state")
		return
	}

	if messageID == "" {
		a.showError("❌ Invalid message ID")
		return
	}

	// Get the current message to show confirmation
	message, err := a.Client.GetMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error getting message: %v", err))
		return
	}

	// Extract subject for confirmation
	subject := "Unknown subject"
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				subject = header.Value
				break
			}
		}
	}

	// Move message to trash
	err = a.Client.TrashMessage(messageID)
	if err != nil {
		a.showError(fmt.Sprintf("❌ Error moving to trash: %v", err))
		return
	}

	// Show success message
	a.showStatusMessage(fmt.Sprintf("🗑️  Moved to trash: %s", subject))

	// Remove the message from the list and adjust selection (UI thread)
	if selectedIndex >= 0 && selectedIndex < len(a.ids) {
		a.QueueUpdateDraw(func() { a.safeRemoveCurrentSelection(messageID) })
	}
}
