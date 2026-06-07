package tui

import (
	"github.com/ajramos/giztui/internal/services"
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
