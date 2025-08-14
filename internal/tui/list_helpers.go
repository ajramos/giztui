package tui

import (
	"fmt"

	"github.com/derailed/tview"
)

// safeRemoveCurrentSelection removes the currently selected row from the list table
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

	// Determine index to remove from current selection
	removeIndex, _ := table.GetSelection()
	if removeIndex < 0 || removeIndex >= count {
		removeIndex = 0
	}

	// Preselect a different index to avoid glitches when removing the selected row
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
		table.Select(pre, 0)
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
		if removeIndex >= 0 && removeIndex < table.GetRowCount() {
			table.RemoveRow(removeIndex)
		}
		// Keep the same visual position when possible
		desired := removeIndex
		newCount := table.GetRowCount()
		if desired >= newCount {
			desired = newCount - 1
		}
		if desired >= 0 && desired < newCount {
			table.Select(desired, 0)
		}
	}

	// Update title and content
	table.SetTitle(fmt.Sprintf(" 📧 Messages (%d) ", len(a.ids)))
	if text, ok := a.views["text"].(*tview.TextView); ok {
		cur, _ := table.GetSelection()
		if cur >= 0 && cur < len(a.ids) {
			go a.showMessageWithoutFocus(a.ids[cur])
			if a.aiSummaryVisible {
				go a.generateOrShowSummary(a.ids[cur])
			}
		} else {
			text.SetText("No messages")
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
			a.RemoveMessageIDAt(i)
			if i < len(a.messagesMeta) {
				a.messagesMeta = append(a.messagesMeta[:i], a.messagesMeta[i+1:]...)
			}
			if i < table.GetRowCount() {
				table.RemoveRow(i)
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
	if cur >= 0 {
		table.Select(cur, 0)
		if cur < len(a.ids) {
			go a.showMessageWithoutFocus(a.ids[cur])
			if a.aiSummaryVisible {
				go a.generateOrShowSummary(a.ids[cur])
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
