package tui

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajramos/giztui/internal/config"
)

// initLogger initializes file logger under ~/.config/giztui/giztui.log if possible
func (a *App) initLogger() {
	if a.logger != nil && a.logFile != nil {
		return
	}
	// Prefer config.LogFile if provided
	if a.Config != nil && a.Config.LogFile != "" {
		if f, err := os.OpenFile(a.Config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
			a.logFile = f
			a.logger = log.New(f, "[giztui] ", log.LstdFlags|log.Lmicroseconds)
			return
		}
		// if it fails, fall back to default path
	}
	logDir := config.DefaultLogDir()
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0o750); err == nil {
			lf := filepath.Join(logDir, "giztui.log")
			// Validate path to prevent directory traversal
			cleanPath := filepath.Clean(lf)
			if strings.Contains(cleanPath, "..") {
				return // Skip logging if invalid path
			}
			if f, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err == nil {
				a.logFile = f
				a.logger = log.New(f, "[giztui] ", log.LstdFlags|log.Lmicroseconds)
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
