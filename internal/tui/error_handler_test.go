package tui

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
)

// Test ErrorHandler constructor
func TestNewErrorHandler(t *testing.T) {
	app := tview.NewApplication()
	statusView := tview.NewTextView()
	flashView := tview.NewTextView()
	logger := log.New(os.Stdout, "", log.LstdFlags)

	eh := NewErrorHandler(app, nil, statusView, flashView, logger)

	assert.NotNil(t, eh)
	assert.Equal(t, app, eh.app)
	assert.Nil(t, eh.appRef)
	assert.Equal(t, statusView, eh.statusView)
	assert.Equal(t, flashView, eh.flashView)
	assert.Equal(t, logger, eh.logger)
	assert.Empty(t, eh.currentStatus)
	assert.Empty(t, eh.persistentStatus)
}

func TestNewErrorHandler_NilInputs(t *testing.T) {
	eh := NewErrorHandler(nil, nil, nil, nil, nil)

	assert.NotNil(t, eh)
	assert.Nil(t, eh.app)
	assert.Nil(t, eh.appRef)
	assert.Nil(t, eh.statusView)
	assert.Nil(t, eh.flashView)
	assert.Nil(t, eh.logger)
}

// Test error handling
func TestErrorHandler_HandleError_NilError(t *testing.T) {
	eh := &ErrorHandler{}

	// Should not panic or do anything with nil error
	assert.NotPanics(t, func() {
		eh.HandleError(context.Background(), nil, "test message")
	})
}

func TestErrorHandler_HandleError_WithError(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	testError := errors.New("test error")

	// Should not panic
	assert.NotPanics(t, func() {
		eh.HandleError(context.Background(), testError, "Custom error message")
	})
}

func TestErrorHandler_HandleError_EmptyUserMessage(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	testError := errors.New("test error")

	// Should use default message when userMsg is empty
	assert.NotPanics(t, func() {
		eh.HandleError(context.Background(), testError, "")
	})
}

// Test message formatting
func TestErrorHandler_formatMessage(t *testing.T) {
	eh := &ErrorHandler{}

	testCases := []struct {
		message  string
		level    LogLevel
		wantIcon string
	}{
		{"Test info", LogLevelInfo, "ℹ️"},
		{"Test warning", LogLevelWarning, "⚠️"},
		{"Test error", LogLevelError, "❌"},
		{"Test success", LogLevelSuccess, "✅"},
		{"Test unknown", LogLevel(99), "•"},
	}

	for _, tc := range testCases {
		result := eh.formatMessage(tc.message, tc.level)
		assert.Contains(t, result, tc.wantIcon)
		assert.Contains(t, result, tc.message)
	}
}

func TestErrorHandler_formatMessage_EmptyMessage(t *testing.T) {
	eh := &ErrorHandler{}

	result := eh.formatMessage("", LogLevelInfo)
	assert.Contains(t, result, "ℹ️")
	// Should still contain the icon even with empty message
}

// Test level conversion functions
func TestErrorHandler_levelToString(t *testing.T) {
	eh := &ErrorHandler{}

	testCases := []struct {
		level LogLevel
		want  string
	}{
		{LogLevelInfo, "INFO"},
		{LogLevelWarning, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelSuccess, "SUCCESS"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tc := range testCases {
		result := eh.levelToString(tc.level)
		assert.Equal(t, tc.want, result)
	}
}

func TestErrorHandler_levelToColor_NilAppRef(t *testing.T) {
	eh := &ErrorHandler{appRef: nil}

	// Should not panic with nil appRef
	assert.NotPanics(t, func() {
		eh.levelToColor(LogLevelInfo)
		eh.levelToColor(LogLevelWarning)
		eh.levelToColor(LogLevelError)
		eh.levelToColor(LogLevelSuccess)
		eh.levelToColor(LogLevel(99))
	})
}

// Test baseline status
func TestErrorHandler_getBaselineStatus_WithoutAppRef(t *testing.T) {
	eh := &ErrorHandler{appRef: nil}

	result := eh.getBaselineStatus()
	assert.Equal(t, "GizTUI • Press ? for help • : for commands", result)
}

// Test persistent message functionality
func TestErrorHandler_ShowPersistentMessage(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	assert.NotPanics(t, func() {
		eh.ShowPersistentMessage(context.Background(), "Test persistent message", LogLevelInfo)
	})

	// Check that persistent status was set
	eh.mu.RLock()
	assert.Contains(t, eh.persistentStatus, "Test persistent message")
	eh.mu.RUnlock()
}

func TestErrorHandler_ClearPersistentMessage(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView:       statusView,
		persistentStatus: "Test persistent",
	}

	assert.NotPanics(t, func() {
		eh.ClearPersistentMessage()
	})

	// Check that persistent status was cleared
	eh.mu.RLock()
	assert.Empty(t, eh.persistentStatus)
	eh.mu.RUnlock()
}

// Test show message without app (no UI updates)
func TestErrorHandler_ShowMessage_NoApp(t *testing.T) {
	eh := &ErrorHandler{app: nil}

	assert.NotPanics(t, func() {
		eh.ShowMessage(context.Background(), "Test message", LogLevelInfo)
	})
}

func TestErrorHandler_ShowMessage_EmptyMessage(t *testing.T) {
	eh := &ErrorHandler{}

	// Should return early without doing anything
	assert.NotPanics(t, func() {
		eh.ShowMessage(context.Background(), "", LogLevelInfo)
		eh.ShowMessage(context.Background(), "   ", LogLevelInfo)
		eh.ShowMessage(context.Background(), "\t\n", LogLevelInfo)
	})
}

// Test flash message functionality
func TestErrorHandler_ShowFlashMessage_NoFlashView(t *testing.T) {
	eh := &ErrorHandler{flashView: nil}

	// Should fallback to regular message when no flash view
	assert.NotPanics(t, func() {
		eh.ShowFlashMessage(context.Background(), "Flash test", LogLevelInfo, 100)
	})
}

func TestErrorHandler_ShowFlashMessage_WithFlashView(t *testing.T) {
	flashView := tview.NewTextView()
	eh := &ErrorHandler{
		flashView: flashView,
	}

	assert.NotPanics(t, func() {
		eh.ShowFlashMessage(context.Background(), "Flash test", LogLevelInfo, 100)
	})
}

// Test convenience methods
func TestErrorHandler_ConvenienceMethods(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	ctx := context.Background()

	// Test all convenience methods
	assert.NotPanics(t, func() {
		eh.ShowInfo(ctx, "Info message")
		eh.ShowWarning(ctx, "Warning message")
		eh.ShowError(ctx, "Error message")
		eh.ShowSuccess(ctx, "Success message")
		eh.ShowProgress(ctx, "Progress message")
		eh.ClearProgress()
	})
}

func TestErrorHandler_ShowLLMError(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	testError := errors.New("LLM connection failed")

	assert.NotPanics(t, func() {
		eh.ShowLLMError(context.Background(), "summarization", testError)
	})
}

func TestErrorHandler_ShowGmailError(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	testError := errors.New("Gmail API quota exceeded")

	assert.NotPanics(t, func() {
		eh.ShowGmailError(context.Background(), "message retrieval", testError)
	})
}

// Test status display logic
func TestErrorHandler_refreshStatusDisplay_Priority(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	// Test priority: current > persistent > baseline
	eh.mu.Lock()
	eh.currentStatus = "Current message"
	eh.persistentStatus = "Persistent message"
	eh.mu.Unlock()

	eh.refreshStatusDisplay()

	// Should show current status (highest priority)
	text := strings.TrimSpace(statusView.GetText(false))
	assert.Equal(t, "Current message", text)

	// Clear current, should show persistent
	eh.mu.Lock()
	eh.currentStatus = ""
	eh.mu.Unlock()

	eh.refreshStatusDisplay()
	text = strings.TrimSpace(statusView.GetText(false))
	assert.Equal(t, "Persistent message", text)

	// Clear persistent, should show baseline
	eh.mu.Lock()
	eh.persistentStatus = ""
	eh.mu.Unlock()

	eh.refreshStatusDisplay()
	text = strings.TrimSpace(statusView.GetText(false))
	assert.Equal(t, "GizTUI • Press ? for help • : for commands", text)
}

func TestErrorHandler_refreshStatusDisplay_NoStatusView(t *testing.T) {
	eh := &ErrorHandler{statusView: nil}

	// Should not panic with nil statusView
	assert.NotPanics(t, func() {
		eh.refreshStatusDisplay()
	})
}

// Test thread safety
func TestErrorHandler_ThreadSafety(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	ctx := context.Background()

	// Test concurrent access to status methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			eh.ShowInfo(ctx, "Concurrent message")
			eh.ShowProgress(ctx, "Concurrent progress")
			eh.ClearProgress()
			eh.ShowPersistentMessage(ctx, "Concurrent persistent", LogLevelInfo)
			eh.ClearPersistentMessage()
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not have panicked
	assert.True(t, true)
}

// Test edge cases
func TestErrorHandler_EdgeCases(t *testing.T) {
	t.Run("unicode_messages", func(t *testing.T) {
		eh := &ErrorHandler{}

		unicodeMsg := "测试消息 🔥 emoji"
		result := eh.formatMessage(unicodeMsg, LogLevelInfo)

		assert.Contains(t, result, unicodeMsg)
		assert.Contains(t, result, "ℹ️")
	})

	t.Run("very_long_message", func(t *testing.T) {
		eh := &ErrorHandler{}

		longMsg := strings.Repeat("A", 1000)
		result := eh.formatMessage(longMsg, LogLevelError)

		assert.Contains(t, result, longMsg)
		assert.Contains(t, result, "❌")
	})

	t.Run("newlines_in_message", func(t *testing.T) {
		eh := &ErrorHandler{}

		multilineMsg := "Line 1\nLine 2\nLine 3"
		result := eh.formatMessage(multilineMsg, LogLevelWarning)

		assert.Contains(t, result, multilineMsg)
		assert.Contains(t, result, "⚠️")
	})

	t.Run("nil_context", func(t *testing.T) {
		eh := &ErrorHandler{}

		assert.NotPanics(t, func() {
			eh.ShowMessage(nil, "Test with nil context", LogLevelInfo)
			eh.ShowPersistentMessage(nil, "Persistent with nil context", LogLevelInfo)
			eh.HandleError(nil, errors.New("test"), "Error with nil context")
		})
	})
}

// Test status message auto-clearing behavior
func TestErrorHandler_StatusAutoClearing(t *testing.T) {
	statusView := tview.NewTextView()
	eh := &ErrorHandler{
		statusView: statusView,
	}

	// Test that error messages should auto-clear
	eh.updateStatusMessage("Test error", LogLevelError)

	// Should have a timer set for auto-clearing
	eh.mu.RLock()
	hasTimer := eh.statusTimer != nil
	eh.mu.RUnlock()

	assert.True(t, hasTimer, "Error messages should have auto-clear timer")

	// Test that info messages with no persistent status should auto-clear
	eh.updateStatusMessage("Test info", LogLevelInfo)

	eh.mu.RLock()
	hasTimer = eh.statusTimer != nil
	eh.mu.RUnlock()

	assert.True(t, hasTimer, "Info messages should auto-clear when no persistent status")
}
