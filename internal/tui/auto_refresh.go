package tui

import "time"

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

// performAutoRefreshTick is implemented in a later task.
func (a *App) performAutoRefreshTick() {}
