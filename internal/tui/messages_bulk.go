package tui

import (
	"fmt"
	"time"

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
		// Use bulk service method for proper undo recording
		emailService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
		err := emailService.BulkArchive(a.ctx, ids)

		failed := 0
		if err != nil {
			// Count failures (this is approximate since BulkArchive doesn't return detailed failure info)
			failed = 1 // Mark as partial failure
		}
		a.QueueUpdateDraw(func() {
			a.removeIDsFromCurrentList(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(a.getSelectionStyle())
			}
		})

		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()

		// Small delay to ensure progress message is fully cleared before showing final status
		go func() {
			time.Sleep(100 * time.Millisecond)
			if failed == 0 {
				a.GetErrorHandler().ShowSuccess(a.ctx, "Archived")
			} else {
				a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Archived with %d failure(s)", failed))
			}
		}()
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
		// Use bulk service method for proper undo recording
		emailService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
		err := emailService.BulkTrash(a.ctx, ids)

		failed := 0
		if err != nil {
			// Count failures (this is approximate since BulkTrash doesn't return detailed failure info)
			failed = 1 // Mark as partial failure
		}
		a.QueueUpdateDraw(func() {
			a.removeIDsFromCurrentList(ids)
			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(a.getSelectionStyle())
			}
		})

		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()

		// Small delay to ensure progress message is fully cleared before showing final status
		go func() {
			time.Sleep(100 * time.Millisecond)
			if failed == 0 {
				a.GetErrorHandler().ShowSuccess(a.ctx, "Trashed")
			} else {
				a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Trashed with %d failure(s)", failed))
			}
		}()
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
		// Get EmailService to ensure undo actions are recorded
		emailService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()

		var err error
		if markAsUnread {
			err = emailService.BulkMarkAsUnread(a.ctx, ids)
		} else {
			err = emailService.BulkMarkAsRead(a.ctx, ids)
		}

		failed := 0
		if err != nil {
			failed = len(ids) // If bulk operation fails, consider all as failed
		}

		// Update UI after all operations complete
		a.QueueUpdateDraw(func() {
			// NOTE: Removed manual cache updates - let undo system handle cache updates to avoid conflicts
			// The bulk service methods record proper undo actions, and undo will handle cache updates

			// Exit bulk mode and restore normal rendering/styles
			a.selected = make(map[string]bool)
			a.bulkMode = false
			a.reformatListItems()
			if list, ok := a.views["list"].(*tview.Table); ok {
				list.SetSelectedStyle(a.getSelectionStyle())
			}
		})

		// ErrorHandler calls outside QueueUpdateDraw to avoid deadlock
		a.GetErrorHandler().ClearProgress()

		// Small delay to ensure progress message is fully cleared before showing final status
		go func() {
			time.Sleep(100 * time.Millisecond)
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
	}()
}
