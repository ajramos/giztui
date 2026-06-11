package tui

import (
	"time"

	"github.com/derailed/tview"
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

// prependNewMessages is implemented in a later task.
func (a *App) prependNewMessages(newIDs []string) {}
