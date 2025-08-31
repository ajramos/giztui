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

	// Preselect a different index to avoid glitches when removing the selected row
	if count > 1 {
		// Convert removeIndex (message index) to table row index (+1 for header)
		removeTableRow := removeIndex + 1

		// Calculate pre-selection table row
		pre := removeTableRow - 1
		if removeIndex == 0 {
			pre = 2 // Second message (table row 2) when removing first message
		}
		if pre < 1 { // Never select header row (0)
			pre = 1
		}
		if pre >= count {
			pre = count - 1
		}
		// Only auto-select if composition panel is not active
		if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
			table.Select(pre, 0)
		}
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

			// CRITICAL FIX: Ensure table row count matches a.ids length
			// Sometimes RemoveRow doesn't properly sync, so force a refresh
			actualTableRows := table.GetRowCount()
			expectedRows := len(a.ids)
			if actualTableRows != expectedRows {
				if a.logger != nil {
					a.logger.Printf("TABLE SYNC BUG: table has %d rows but a.ids has %d entries, forcing table rebuild", actualTableRows, expectedRows)
				}
				// Force rebuild the table to sync with a.ids
				table.Clear()
				for i := range a.ids {
					if i >= len(a.messagesMeta) || a.messagesMeta[i] == nil {
						table.SetCell(i, 0, tview.NewTableCell(fmt.Sprintf("Loading message %d...", i+1)))
						continue
					}
					msg := a.messagesMeta[i]
					text, _ := a.emailRenderer.FormatEmailList(msg, a.getFormatWidth())
					// Create cell with proper styling
					cell := tview.NewTableCell(text).SetExpansion(1)
					table.SetCell(i, 0, cell)
				}
			}
		}
		// Keep the same visual position when possible
		// Convert removeIndex (message index) to table row index (+1 for header)
		desired := removeIndex + 1
		newCount := table.GetRowCount()
		if desired >= newCount {
			desired = newCount - 1
		}
		// Only auto-select if composition panel is not active
		if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
			if desired >= 1 && desired < newCount { // Never select header row (0)
				table.Select(desired, 0)
			} else if newCount > 1 {
				table.Select(1, 0) // Select first message if no other option
			}
		}
	}

	// Update title and content
	table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
	if text, ok := a.views["text"].(*tview.TextView); ok {
		messageIndex := a.getCurrentSelectedMessageIndex()
		if messageIndex >= 0 {
			go a.showMessageWithoutFocus(a.ids[messageIndex])
			if a.aiSummaryVisible {
				go a.generateOrShowSummary(a.ids[messageIndex])
			}
		} else {
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
		// Convert table row index to message index (-1 for header)
		messageIndex := cur - 1
		if messageIndex >= 0 && messageIndex < len(a.ids) {
			go a.showMessageWithoutFocus(a.ids[messageIndex])
			if a.aiSummaryVisible {
				go a.generateOrShowSummary(a.ids[messageIndex])
			}
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
