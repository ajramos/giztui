package tui

import (
	"net/mail"
	"sort"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

type readManuallyGroup struct {
	senderKey  string // normalized address, lowercased
	senderDisp string // first-seen raw From, for display
	msgs       []services.AnalyzerMessage
}

// normalizeSender extracts a lowercased email address from a From header, falling back to the
// trimmed/lowercased raw value when it doesn't parse.
func normalizeSender(from string) string {
	if addr, err := mail.ParseAddress(strings.TrimSpace(from)); err == nil {
		return strings.ToLower(strings.TrimSpace(addr.Address))
	}
	return strings.ToLower(strings.TrimSpace(from))
}

// groupReadManuallyBySender groups messages by normalized sender, ordered by descending group
// size then senderKey; within a group, input order is preserved.
func groupReadManuallyBySender(msgs []services.AnalyzerMessage) []readManuallyGroup {
	idx := map[string]int{}
	var groups []readManuallyGroup
	for _, m := range msgs {
		key := normalizeSender(m.From)
		if i, ok := idx[key]; ok {
			groups[i].msgs = append(groups[i].msgs, m)
			continue
		}
		idx[key] = len(groups)
		groups = append(groups, readManuallyGroup{senderKey: key, senderDisp: strings.TrimSpace(m.From), msgs: []services.AnalyzerMessage{m}})
	}
	sort.SliceStable(groups, func(a, b int) bool {
		if len(groups[a].msgs) != len(groups[b].msgs) {
			return len(groups[a].msgs) > len(groups[b].msgs)
		}
		return groups[a].senderKey < groups[b].senderKey
	})
	return groups
}

// senderExpandKey is the state.expanded map key for a sender group under read-manually.
func senderExpandKey(senderKey string) string {
	return "\x00read-manually:" + senderKey
}

// readManuallyLeafLabel renders one email leaf, appending the AI hint/suggestion when present.
func readManuallyLeafLabel(m services.AnalyzerMessage, sug services.ReadManuallySuggestion, hasSug bool) string {
	subject := strings.TrimSpace(m.Subject)
	if subject == "" {
		subject = "(no subject)"
	}
	if !hasSug || (sug.Hint == "" && sug.Action == "read") {
		return subject
	}
	if sug.Action == "read" {
		return subject + " — 💡 " + sug.Hint
	}
	verb := actionVerbLabel(sug.Action)
	if sug.Action == "label" && sug.Label != "" {
		verb = verb + " " + sug.Label
	}
	out := subject
	if sug.Hint != "" {
		out += " — 💡 " + sug.Hint
	}
	return out + " · suggests: " + verb
}
