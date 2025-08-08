package tui

import (
	"log"
	"os"
	"path/filepath"
)

// initLogger initializes file logger under ~/.config/gmail-tui/gmail-tui.log if possible
func (a *App) initLogger() {
	if a.logger != nil && a.logFile != nil {
		return
	}
	if home, err := os.UserHomeDir(); err == nil {
		logDir := filepath.Join(home, ".config", "gmail-tui")
		if err := os.MkdirAll(logDir, 0o755); err == nil {
			lf := filepath.Join(logDir, "gmail-tui.log")
			if f, err := os.OpenFile(lf, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
				a.logFile = f
				a.logger = log.New(f, "[gmail-tui] ", log.LstdFlags|log.Lmicroseconds)
			}
		}
	}
}

// closeLogger closes the log file if opened
func (a *App) closeLogger() {
	if a.logFile != nil {
		_ = a.logFile.Close()
		a.logFile = nil
	}
}
