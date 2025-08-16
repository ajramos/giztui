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
			a.GetErrorHandler().ShowSuccess(a.ctx, "✅ Archived")
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("✅ Archived with %d failure(s)", failed))
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
			a.GetErrorHandler().ShowSuccess(a.ctx, "✅ Trashed")
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("✅ Trashed with %d failure(s)", failed))
		}
	}()
}
