package tui

import (
	"fmt"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// archiveSelectedBulk archives all selected messages
func (a *App) archiveSelectedBulk() {
	if len(a.selected) == 0 {
		return
	}
	// Snapshot selection
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Archiving %d message(s)…", len(ids)))
	go func() {
		failed := 0
		total := len(ids)
		for i, id := range ids {
			if err := a.Client.ArchiveMessage(id); err != nil {
				failed++
				continue
			}
			// Progress update on UI thread
			idx := i + 1
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Archiving %d/%d…", idx, total))
			// Remove from UI list on main thread after loop
		}
		a.QueueUpdateDraw(func() {
			a.removeIDsFromCurrentList(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
			}
		})
		
		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, "Archived")
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Archived with %d failure(s)", failed))
		}
	}()
}

// trashSelectedBulk moves all selected messages to trash
func (a *App) trashSelectedBulk() {
	if len(a.selected) == 0 {
		return
	}
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Trashing %d message(s)…", len(ids)))
	go func() {
		failed := 0
		total := len(ids)
		for i, id := range ids {
			if err := a.Client.TrashMessage(id); err != nil {
				failed++
			}
			// Progress update on UI thread
			idx := i + 1
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Trashing %d/%d…", idx, total))
		}
		a.QueueUpdateDraw(func() {
			a.removeIDsFromCurrentList(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
			}
		})
		
		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, "Trashed")
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Trashed with %d failure(s)", failed))
		}
	}()
}

// toggleMarkReadUnreadBulk toggles read/unread status for all selected messages
func (a *App) toggleMarkReadUnreadBulk() {
	if len(a.selected) == 0 {
		return
	}
	
	// Snapshot selection
	ids := make([]string, 0, len(a.selected))
	for id := range a.selected {
		ids = append(ids, id)
	}
	
	// Determine the action by checking the majority state of selected messages
	// If majority are unread, mark all as read. If majority are read, mark all as unread.
	unreadCount := 0
	for _, id := range ids {
		// Find this message in our cache
		for i, cacheID := range a.ids {
			if cacheID == id && i < len(a.messagesMeta) && a.messagesMeta[i] != nil {
				// Check if this message has UNREAD label
				for _, labelID := range a.messagesMeta[i].LabelIds {
					if labelID == "UNREAD" {
						unreadCount++
						break
					}
				}
				break
			}
		}
	}
	
	// Decide action: if majority are unread, mark all as read; otherwise mark all as unread
	markAsUnread := unreadCount <= len(ids)/2
	action := "read"
	if markAsUnread {
		action = "unread"
	}
	
	a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Marking %d message(s) as %s…", len(ids), action))
	
	go func() {
		failed := 0
		total := len(ids)
		
		for i, id := range ids {
			var err error
			if markAsUnread {
				err = a.Client.MarkAsUnread(id)
			} else {
				err = a.Client.MarkAsRead(id)
			}
			
			if err != nil {
				failed++
				continue
			}
			
			// Progress update
			idx := i + 1
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Marking %d/%d as %s…", idx, total, action))
		}
		
		// Update UI after all operations complete
		a.QueueUpdateDraw(func() {
			// Update cache for all processed messages
			for _, id := range ids {
				if markAsUnread {
					a.updateCachedMessageLabels(id, "UNREAD", true)
				} else {
					a.updateCachedMessageLabels(id, "UNREAD", false)
				}
			}
			
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlue))
			}
		})
		
		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()
		if failed == 0 {
			if markAsUnread {
				a.GetErrorHandler().ShowSuccess(a.ctx, "Marked as unread")
			} else {
				a.GetErrorHandler().ShowSuccess(a.ctx, "Marked as read")
			}
		} else {
			if markAsUnread {
				a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Marked as unread with %d failure(s)", failed))
			} else {
				a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Marked as read with %d failure(s)", failed))
			}
		}
	}()
}
