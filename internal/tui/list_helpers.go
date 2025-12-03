package tui

import (
	"fmt"

	"github.com/derailed/tview"
)

// safeRemoveCurrentSelection removes the message with the given ID from the list table
// while safely updating internal caches and adjusting the selection and content panes.
// It must be called on the UI thread via a.QueueUpdateDraw.
func (a *App) safeRemoveCurrentSelection(removedMessageID string) {

	table, ok := a.views["list"].(*tview.Table)
	if !ok {
		return
	}
	count := table.GetRowCount()
	if count == 0 {
		return
	}

	// Find the index of the message to remove by ID
	removeIndex := -1
	for i, id := range a.ids {
		if id == removedMessageID {
			removeIndex = i
			break
		}
	}

	// If message not found, don't remove anything
	if removeIndex < 0 || removeIndex >= count {
		return
	}

	// Update caches using removeIndex
	if removeIndex >= 0 && removeIndex < len(a.ids) {
		a.RemoveMessageIDAt(removeIndex)
	}
	if removeIndex >= 0 && removeIndex < len(a.messagesMeta) {
		a.messagesMeta = append(a.messagesMeta[:removeIndex], a.messagesMeta[removeIndex+1:]...)
	}

	// Visual removal
	if count == 1 {
		table.Clear()
	} else {
		// CRITICAL FIX: Use correct table row index (removeIndex + 1 for header offset)
		removeTableRow := removeIndex + 1
		if removeTableRow >= 1 && removeTableRow < table.GetRowCount() {
			if a.logger != nil {
				a.logger.Printf("SAFE REMOVE FIX: Removing message at index %d (table row %d)", removeIndex, removeTableRow)
			}
			table.RemoveRow(removeTableRow)

			// CRITICAL FIX: Ensure table row count matches a.ids length + 1 (for header)
			// Sometimes RemoveRow doesn't properly sync, so force removing extra rows
			actualTableRows := table.GetRowCount()
			expectedRows := len(a.ids) + 1 // +1 for header row
			if actualTableRows != expectedRows {
				if a.logger != nil {
					a.logger.Printf("TABLE SYNC BUG: table has %d rows but should have %d (header + %d messages), removing extra rows", actualTableRows, expectedRows, len(a.ids))
				}
				// Remove extra rows from the end of the table until it matches
				// Note: We never remove row 0 (header) or go below len(a.ids) + 1
				for table.GetRowCount() > expectedRows {
					table.RemoveRow(table.GetRowCount() - 1)
				}
			}
		}
		// After removal, select the message that now occupies the same visual position
		// In the table, row 0 is the header, so messages start at row 1
		// The removeIndex was the message array index (0-based)
		// After removal:
		// - If we deleted message at index 0, the next message (previously at index 1) is now at table row 1
		// - If we deleted message at index N-1 (last), we want to select the new last message at table row len(a.ids)
		// - If we deleted message at index i, the next message (previously at index i+1) is now at table row i+1

		newCount := table.GetRowCount()
		var desired int

		// If we deleted the last message, select the new last message
		if removeIndex >= len(a.ids) {
			desired = len(a.ids) // This is the table row for the last message (accounting for header at row 0)
		} else {
			// Otherwise, stay at the same visual position (which now shows the next message)
			desired = removeIndex + 1 // Convert message index to table row (header offset)
		}

		// Safety bounds check
		if desired >= newCount {
			desired = newCount - 1
		}
		if desired < 1 {
			desired = 1
		}

		// Only auto-select if composition panel is not active
		if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
			if desired >= 1 && desired < newCount {
				table.Select(desired, 0)
			} else if newCount > 1 {
				table.Select(1, 0) // Select first message if no other option
			}
		}
	}

	// Update title
	table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))

	// Content update is handled automatically by SetSelectionChangedFunc when table.Select() is called above
	// No need to manually call showMessageWithoutFocus here as it creates race conditions
	if a.logger != nil {
		a.logger.Printf("DELETION FIX: Content update delegated to SetSelectionChangedFunc to avoid race condition")
	}

	// Only handle the edge case where there are no messages left
	if len(a.ids) == 0 {
		if text, ok := a.views["text"].(*tview.TextView); ok {
			a.enhancedTextView.SetContent("No messages")
			text.ScrollToBeginning()
			if a.aiSummaryVisible && a.aiSummaryView != nil {
				a.aiSummaryView.SetText("")
			}
		}
	}

	// Propagate to base snapshot if in local filter
	if removedMessageID != "" {
		a.baseRemoveByID(removedMessageID)
	}
}

// removeIDsFromCurrentList removes all messages with the provided IDs from the
// current list, updates caches, adjusts the selection, and updates content.
// It must be called on the UI thread via a.QueueUpdateDraw.
func (a *App) removeIDsFromCurrentList(ids []string) {
	table, ok := a.views["list"].(*tview.Table)
	if !ok {
		return
	}
	if len(ids) == 0 {
		return
	}
	// Build a set for quick lookup
	rm := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		rm[id] = struct{}{}
	}
	// Walk ids and remove those that are in rm
	i := 0
	for i < len(a.ids) {
		if _, ok := rm[a.ids[i]]; ok {
			// Get message ID for debugging before removal
			messageID := a.ids[i]

			a.RemoveMessageIDAt(i)
			if i < len(a.messagesMeta) {
				a.messagesMeta = append(a.messagesMeta[:i], a.messagesMeta[i+1:]...)
			}

			// CRITICAL FIX: Account for header row offset when removing from table
			// Message at index i is displayed in table row i+1 (because row 0 is header)
			tableRowToRemove := i + 1
			if tableRowToRemove < table.GetRowCount() && tableRowToRemove > 0 {
				if a.logger != nil {
					a.logger.Printf("MOVE FIX: Removing message '%s' at index %d (table row %d)", messageID, i, tableRowToRemove)
				}
				table.RemoveRow(tableRowToRemove)
			}
			continue
		}
		i++
	}
	table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))

	// Adjust selection and content
	cur, _ := table.GetSelection()
	if cur >= table.GetRowCount() {
		cur = table.GetRowCount() - 1
	}
	if cur < 1 { // Never select header row (0)
		cur = 1
	}
	if cur >= 1 && cur < table.GetRowCount() {
		// Only auto-select if composition panel is not active
		if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
			table.Select(cur, 0)
		}
		// Content update is handled automatically by SetSelectionChangedFunc when table.Select() is called above
		// No need to manually call showMessageWithoutFocus here as it creates race conditions
		if a.logger != nil {
			a.logger.Printf("BULK DELETION FIX: Content update delegated to SetSelectionChangedFunc to avoid race condition")
		}
	}
	if table.GetRowCount() == 0 {
		if tv, ok := a.views["text"].(*tview.TextView); ok {
			tv.SetText("No messages")
			tv.ScrollToBeginning()
		}
		if a.aiSummaryVisible && a.aiSummaryView != nil {
			a.aiSummaryView.SetText("")
		}
	}

	// Propagate to base snapshot if in local filter
	a.baseRemoveByIDs(ids)
}
