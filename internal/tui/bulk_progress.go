package tui

import (
	"context"
	"fmt"
)

// bulkProgressVerb maps an action token to its present-participle label for progress messages.
func bulkProgressVerb(action string) string {
	switch action {
	case "archive":
		return "Archiving"
	case "trash":
		return "Trashing"
	case "mark_read":
		return "Marking read"
	case "mark_unread":
		return "Marking unread"
	case "label":
		return "Applying label"
	default:
		return "Processing"
	}
}

// bulkProgress returns a per-item progress callback that updates the status bar with
// "<verb> done/total…". Safe to call from a worker goroutine (ShowProgress marshals to the UI
// thread); MUST NOT be called on the UI goroutine. Pass it to the EmailService/LabelService
// Bulk* methods, and clear the status afterwards with ErrorHandler.ClearPersistentMessage().
func (a *App) bulkProgress(ctx context.Context, verb string) func(done, total int) {
	return func(done, total int) {
		a.GetErrorHandler().ShowProgress(ctx, fmt.Sprintf("%s %d/%d…", verb, done, total))
	}
}
