package tui

import (
	"fmt"
	"time"

	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// isAutoRefreshRunning reports whether the ticker goroutine is active.
func (a *App) isAutoRefreshRunning() bool {
	a.autoRefreshMu.Lock()
	defer a.autoRefreshMu.Unlock()
	return a.autoRefreshRunning
}

// startAutoRefresh launches the background ticker. Idempotent.
func (a *App) startAutoRefresh() {
	a.autoRefreshMu.Lock()
	if a.autoRefreshRunning {
		a.autoRefreshMu.Unlock()
		return
	}
	stop := make(chan struct{})
	a.autoRefreshStop = stop
	a.autoRefreshRunning = true
	a.autoRefreshMu.Unlock()

	interval := time.Minute
	if a.autoRefreshService != nil {
		interval = a.autoRefreshService.Interval()
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-a.ctx.Done():
				return
			case <-ticker.C:
				// Pick up interval changes without restarting the ticker goroutine.
				if a.autoRefreshService != nil {
					if cur := a.autoRefreshService.Interval(); cur > 0 && cur != interval {
						interval = cur
						ticker.Reset(interval)
					}
				}
				go a.performAutoRefreshTick()
			}
		}
	}()
}

// stopAutoRefresh stops the ticker goroutine. Idempotent.
func (a *App) stopAutoRefresh() {
	a.autoRefreshMu.Lock()
	defer a.autoRefreshMu.Unlock()
	if !a.autoRefreshRunning {
		return
	}
	close(a.autoRefreshStop)
	a.autoRefreshStop = nil
	a.autoRefreshRunning = false
}

// refreshStatusBar repaints the status baseline so indicator changes (⟳, 📬N)
// show immediately. Must be called on the UI thread (inside QueueUpdateDraw).
func (a *App) refreshStatusBar() {
	if status, ok := a.views["status"].(*tview.TextView); ok {
		status.SetText(a.statusBaseline())
	}
}

// shouldAutoRefreshPoll reports whether the displayed view is the plain inbox,
// i.e. auto-refresh should poll at all. Off-inbox views (search/folder/threading)
// idle the ticker.
func (a *App) shouldAutoRefreshPoll() bool {
	if a.searchMode != "" {
		return false
	}
	if a.currentQuery != "" {
		return false
	}
	if a.IsThreadingEnabled() && a.GetCurrentThreadViewMode() == ThreadViewThread {
		return false
	}
	return true
}

// isAutoRefreshSafeState reports whether it is safe to prepend new rows in place
// (vs. only bumping the status counter).
func (a *App) isAutoRefreshSafeState() bool {
	if !a.shouldAutoRefreshPoll() {
		return false
	}
	if a.currentActivePicker != PickerNone {
		return false
	}
	if a.bulkMode {
		return false
	}
	if a.compositionPanel != nil && a.compositionPanel.IsVisible() {
		return false
	}
	return true
}

// performAutoRefreshTick runs one detection cycle and applies the result.
func (a *App) performAutoRefreshTick() {
	if a.autoRefreshService == nil || !a.autoRefreshService.IsEnabled() {
		return
	}
	if a.IsMessagesLoading() || !a.shouldAutoRefreshPoll() {
		return
	}

	known := a.GetMessageIDs()
	newIDs, err := a.autoRefreshService.CheckForNewMessages(a.ctx, known)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("AUTO_REFRESH: detection error: %v", err)
		}
		return
	}
	if len(newIDs) == 0 {
		return
	}

	if a.isAutoRefreshSafeState() {
		a.prependNewMessages(newIDs)
		return
	}

	// Not safe: surface a pending counter without touching the list.
	a.SetPendingNewCount(len(newIDs))
	a.QueueUpdateDraw(func() {
		a.refreshStatusBar()
	})
}

// prependIDsAndLocate returns the new id slice (newIDs prepended) and the row
// index (0-based, message-space) of selectedID in the new slice, or 0 if absent.
func prependIDsAndLocate(newIDs, existingIDs []string, selectedID string) ([]string, int) {
	merged := make([]string, 0, len(newIDs)+len(existingIDs))
	merged = append(merged, newIDs...)
	merged = append(merged, existingIDs...)
	for i, id := range merged {
		if id == selectedID {
			return merged, i
		}
	}
	return merged, 0
}

// prependNewMessages fetches metadata for newIDs and inserts them at the top of
// the list in place, preserving the user's cursor. No table.Clear(), no spinner.
func (a *App) prependNewMessages(newIDs []string) {
	// Fetch metadata for just the new arrivals (newest-first order preserved).
	metas, err := a.Client.GetMessagesMetadataParallel(newIDs, 10)
	if err != nil {
		if a.logger != nil {
			a.logger.Printf("AUTO_REFRESH: metadata fetch error: %v", err)
		}
		return
	}

	// Capture current selection by message ID.
	selectedID := a.GetCurrentMessageID()

	// Update the in-memory model under lock: prepend metas and ids.
	a.mu.Lock()
	a.messagesMeta = append(append([]*gmailapi.Message{}, metas...), a.messagesMeta...)
	a.mu.Unlock()

	mergedIDs, newSelIdx := prependIDsAndLocate(newIDs, a.GetMessageIDs(), selectedID)
	a.SetMessageIDs(mergedIDs)

	// Clear the pending counter — these are now loaded.
	a.SetPendingNewCount(0)

	count := len(newIDs)
	a.QueueUpdateDraw(func() {
		a.reformatListItems() // re-render rows from the model (no network, no clear-flash)
		if table, ok := a.views["list"].(*tview.Table); ok {
			table.Select(newSelIdx+1, 0) // +1 for the header row
		}
		a.refreshStatusBar()
	})

	go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("📬 %d new message(s)", count))
}
