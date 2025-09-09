package tui

// closeLogger closes the log file if opened
func (a *App) closeLogger() {
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
}
