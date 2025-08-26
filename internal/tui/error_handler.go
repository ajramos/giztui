package tui

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// LogLevel represents the severity of a message
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarning
	LogLevelError
	LogLevelSuccess
)

// ErrorHandler provides consistent error handling and user feedback
type ErrorHandler struct {
	mu         sync.RWMutex
	app        *tview.Application
	appRef     *App // Reference to main App for baseline status
	statusView *tview.TextView
	flashView  *tview.TextView
	logger     *log.Logger

	// Status message state
	currentStatus    string
	persistentStatus string
	statusTimer      *time.Timer
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(app *tview.Application, appRef *App, statusView *tview.TextView, flashView *tview.TextView, logger *log.Logger) *ErrorHandler {
	return &ErrorHandler{
		app:        app,
		appRef:     appRef,
		statusView: statusView,
		flashView:  flashView,
		logger:     logger,
	}
}

// HandleError handles an error and shows appropriate user feedback
func (eh *ErrorHandler) HandleError(ctx context.Context, err error, userMsg string) {
	if err == nil {
		return
	}

	// Log the technical error
	if eh.logger != nil {
		eh.logger.Printf("ERROR: %v", err)
	}

	// Show user-friendly message
	if userMsg == "" {
		userMsg = "An error occurred"
	}

	eh.ShowMessage(ctx, userMsg, LogLevelError)
}

// ShowMessage displays a message to the user
func (eh *ErrorHandler) ShowMessage(ctx context.Context, msg string, level LogLevel) {
	if strings.TrimSpace(msg) == "" {
		return
	}

	// Format message with appropriate color and icon
	formattedMsg := eh.formatMessage(msg, level)

	// Log the message
	if eh.logger != nil {
		levelStr := eh.levelToString(level)
		eh.logger.Printf("%s: %s", levelStr, msg)
	}

	// Update UI in the main thread
	if eh.app != nil {
		eh.app.QueueUpdateDraw(func() {
			eh.updateStatusMessage(formattedMsg, level)
		})
	}
}

// ShowPersistentMessage shows a persistent status message
func (eh *ErrorHandler) ShowPersistentMessage(ctx context.Context, msg string, level LogLevel) {
	formattedMsg := eh.formatMessage(msg, level)

	eh.mu.Lock()
	eh.persistentStatus = formattedMsg
	eh.mu.Unlock()

	if eh.app != nil {
		eh.app.QueueUpdateDraw(func() {
			eh.updatePersistentStatus(formattedMsg)
		})
	}
}

// ClearPersistentMessage clears the persistent status message
func (eh *ErrorHandler) ClearPersistentMessage() {
	eh.mu.Lock()
	eh.persistentStatus = ""
	eh.mu.Unlock()

	if eh.app != nil {
		eh.app.QueueUpdateDraw(func() {
			eh.updatePersistentStatus("")
		})
	}
}

// ShowFlashMessage shows a temporary flash message
func (eh *ErrorHandler) ShowFlashMessage(ctx context.Context, msg string, level LogLevel, duration time.Duration) {
	if eh.flashView == nil {
		// Fallback to status message if no flash view
		eh.ShowMessage(ctx, msg, level)
		return
	}

	formattedMsg := eh.formatMessage(msg, level)

	if eh.app != nil {
		eh.app.QueueUpdateDraw(func() {
			eh.flashView.SetText(formattedMsg)
			eh.flashView.SetTextColor(eh.levelToColor(level))

			// Hide flash message after duration
			time.AfterFunc(duration, func() {
				eh.app.QueueUpdateDraw(func() {
					eh.flashView.SetText("")
				})
			})
		})
	}
}

// formatMessage formats a message with appropriate icon and styling
func (eh *ErrorHandler) formatMessage(msg string, level LogLevel) string {
	var icon string

	switch level {
	case LogLevelInfo:
		icon = "ℹ️"
	case LogLevelWarning:
		icon = "⚠️"
	case LogLevelError:
		icon = "❌"
	case LogLevelSuccess:
		icon = "✅"
	default:
		icon = "•"
	}

	return fmt.Sprintf("%s %s", icon, msg)
}

// levelToString converts LogLevel to string
func (eh *ErrorHandler) levelToString(level LogLevel) string {
	switch level {
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarning:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelSuccess:
		return "SUCCESS"
	default:
		return "UNKNOWN"
	}
}

// levelToColor converts LogLevel to theme-aware tcell.Color
func (eh *ErrorHandler) levelToColor(level LogLevel) tcell.Color {
	switch level {
	case LogLevelInfo:
		return eh.appRef.getStatusColor("info")
	case LogLevelWarning:
		return eh.appRef.getStatusColor("warning")
	case LogLevelError:
		return eh.appRef.getStatusColor("error")
	case LogLevelSuccess:
		return eh.appRef.getStatusColor("success")
	default:
		return eh.appRef.getStatusColor("info") // Default to info color
	}
}

// updateStatusMessage updates the status message with auto-clear
func (eh *ErrorHandler) updateStatusMessage(msg string, level LogLevel) {
	if eh.statusView == nil {
		return
	}

	eh.mu.Lock()
	defer eh.mu.Unlock()

	// Cancel existing timer
	if eh.statusTimer != nil {
		eh.statusTimer.Stop()
	}

	// Update status
	eh.currentStatus = msg
	eh.refreshStatusDisplay()

	// Auto-clear temporary messages after 5 seconds
	if level != LogLevelInfo || eh.persistentStatus == "" {
		// Store current message to check against later for race condition prevention
		currentMsg := msg
		eh.statusTimer = time.AfterFunc(5*time.Second, func() {
			// Avoid nested QueueUpdateDraw - use direct method call instead
			eh.clearCurrentStatusSafely(currentMsg)
		})
	}
}

// clearCurrentStatusSafely clears the current status message without race conditions
func (eh *ErrorHandler) clearCurrentStatusSafely(expectedMsg string) {
	if eh.app != nil {
		eh.app.QueueUpdateDraw(func() {
			eh.mu.Lock()
			defer eh.mu.Unlock()
			
			// Only clear if the current message matches what we expect to clear
			// This prevents clearing a newer message that was set after the timer started
			if eh.currentStatus == expectedMsg {
				eh.currentStatus = ""
				eh.refreshStatusDisplay()
			}
			// If currentStatus != expectedMsg, a newer message was set, so don't clear it
		})
	}
}

// updatePersistentStatus updates the persistent status
func (eh *ErrorHandler) updatePersistentStatus(msg string) {
	eh.mu.Lock()
	defer eh.mu.Unlock()

	eh.persistentStatus = msg
	eh.refreshStatusDisplay()
}

// refreshStatusDisplay refreshes the status display
func (eh *ErrorHandler) refreshStatusDisplay() {
	if eh.statusView == nil {
		return
	}

	var displayText string

	if eh.currentStatus != "" {
		displayText = eh.currentStatus
	} else if eh.persistentStatus != "" {
		displayText = eh.persistentStatus
	} else {
		displayText = eh.getBaselineStatus()
	}

	eh.statusView.SetText(displayText)
}

// getBaselineStatus returns the baseline status text
func (eh *ErrorHandler) getBaselineStatus() string {
	if eh.appRef != nil {
		return eh.appRef.statusBaseline()
	}
	return "Gmail TUI • Press ? for help • : for commands"
}

// Convenience methods for common operations

// ShowInfo shows an info message
func (eh *ErrorHandler) ShowInfo(ctx context.Context, msg string) {
	eh.ShowMessage(ctx, msg, LogLevelInfo)
}

// ShowWarning shows a warning message
func (eh *ErrorHandler) ShowWarning(ctx context.Context, msg string) {
	eh.ShowMessage(ctx, msg, LogLevelWarning)
}

// ShowError shows an error message
func (eh *ErrorHandler) ShowError(ctx context.Context, msg string) {
	eh.ShowMessage(ctx, msg, LogLevelError)
}

// ShowSuccess shows a success message
func (eh *ErrorHandler) ShowSuccess(ctx context.Context, msg string) {
	eh.ShowMessage(ctx, msg, LogLevelSuccess)
}

// ShowLLMError shows an LLM-specific error with context
func (eh *ErrorHandler) ShowLLMError(ctx context.Context, operation string, err error) {
	userMsg := fmt.Sprintf("AI %s failed", operation)
	eh.HandleError(ctx, err, userMsg)
}

// ShowGmailError shows a Gmail API error with context
func (eh *ErrorHandler) ShowGmailError(ctx context.Context, operation string, err error) {
	userMsg := fmt.Sprintf("Gmail %s failed", operation)
	eh.HandleError(ctx, err, userMsg)
}

// ShowProgress shows a progress message
func (eh *ErrorHandler) ShowProgress(ctx context.Context, msg string) {
	eh.ShowPersistentMessage(ctx, msg, LogLevelInfo)
}

// ClearProgress clears any progress message
func (eh *ErrorHandler) ClearProgress() {
	eh.ClearPersistentMessage()
}
