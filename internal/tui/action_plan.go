package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// buildAnalyzerMessages converts already-loaded message metadata into the lightweight
// AnalyzerMessage list the InboxAnalyzerService consumes. Only UNREAD messages are
// included. No Gmail calls are made — everything comes from in-memory metadata.
func buildAnalyzerMessages(metas []*gmailapi.Message) []services.AnalyzerMessage {
	out := make([]services.AnalyzerMessage, 0, len(metas))
	for _, m := range metas {
		if m == nil {
			continue
		}
		if !isUnreadMeta(m) {
			continue
		}
		out = append(out, services.AnalyzerMessage{
			ID:      m.Id,
			Subject: extractHeaderValue(m, "Subject"),
			From:    extractHeaderValue(m, "From"),
			Snippet: m.Snippet,
		})
	}
	return out
}

// isUnreadMeta reports whether a raw message metadata carries the UNREAD label.
func isUnreadMeta(m *gmailapi.Message) bool {
	for _, l := range m.LabelIds {
		if l == "UNREAD" {
			return true
		}
	}
	return false
}

// actionPlanState holds the mutable state of the Action Plan panel.
type actionPlanState struct {
	plan             *services.ActionPlan
	selectedCategory int
	analyzing        bool // true while batches are still streaming; blocks quick-actions

	customPromptText string // override prompt text, "" = default

	header          *tview.TextView
	body            *tview.TextView
	footer          *tview.TextView
	container       *tview.Flex
	streamingCancel context.CancelFunc
}

// actionVerbLabel maps an action token to a human verb for the category header.
func actionVerbLabel(action string) string {
	switch action {
	case "archive":
		return "Archive"
	case "mark_read":
		return "Mark read"
	case "trash":
		return "Trash"
	case "label":
		return "Label"
	default:
		return "Review"
	}
}

// actionKeyHint returns the configured key for the action's quick-action, or "" if none.
func (a *App) actionKeyHint(action string) string {
	switch action {
	case "archive":
		return a.Keys.Archive
	case "mark_read":
		return a.Keys.ToggleRead
	case "trash":
		return a.Keys.Trash
	case "label":
		return a.Keys.ManageLabels
	default:
		return ""
	}
}

// actionKeyHintForAction is the package-level default-key mapping used by the renderer
// (mirrors App.actionKeyHint defaults; the live panel footer uses the configured keys).
func actionKeyHintForAction(action string) string {
	switch action {
	case "archive":
		return "a"
	case "mark_read":
		return "t"
	case "trash":
		return "d"
	case "label":
		return "l"
	default:
		return ""
	}
}

// renderActionPlanText formats a plan into the panel body. selected is the index of the
// currently-highlighted category (or -1). Uses tview dynamic-color tags.
func renderActionPlanText(plan *services.ActionPlan, selected int) string {
	if plan == nil {
		return "Analyzing…"
	}
	var b strings.Builder
	for i, c := range plan.Categories {
		marker := " "
		if i == selected {
			marker = "[::b]▸[::-]"
		}
		key := actionKeyHintForAction(c.Action)
		keyHint := ""
		if key != "" {
			keyHint = fmt.Sprintf("[%s] ", key)
		}
		verb := actionVerbLabel(c.Action)
		fmt.Fprintf(&b, "%s%s%s %d %s   ◀ %s\n", marker, keyHint, verb, len(c.MessageIDs), c.Name, strings.ToUpper(c.Priority))
		if c.Action == "label" && c.Label != "" {
			fmt.Fprintf(&b, "     → label: %s\n", c.Label)
		}
		if c.Description != "" {
			fmt.Fprintf(&b, "     %s\n", c.Description)
		}
		b.WriteString("\n")
	}
	if len(plan.ReadManually) > 0 {
		fmt.Fprintf(&b, "─── Read manually (%d) ───\n", len(plan.ReadManually))
		for i, m := range plan.ReadManually {
			if i >= 10 {
				fmt.Fprintf(&b, "   …and %d more\n", len(plan.ReadManually)-10)
				break
			}
			fmt.Fprintf(&b, "   • %s — %s\n", m.Subject, m.From)
		}
	}
	return b.String()
}
