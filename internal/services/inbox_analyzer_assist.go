package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	_ "embed"
)

//go:embed inbox_analyzer_assist_prompt.txt
var assistReadManuallyPrompt string

type assistItem struct {
	ID     string `json:"id"`
	Hint   string `json:"hint"`
	Action string `json:"action"`
	Label  string `json:"label"`
}

var validAssistActions = map[string]bool{
	"archive": true, "mark_read": true, "trash": true, "label": true, "read": true,
}

// parseAssistResponse turns the model's JSON array into one suggestion per batch ID, in
// batch order. Unknown model IDs are ignored; omitted batch IDs reconcile to {Action:"read"}.
// An invalid action, or a "label" whose name can't be resolved to an existing label, -> "read".
func parseAssistResponse(raw string, batchIDs []string, available map[string]string, strict bool) []ReadManuallySuggestion {
	byID := map[string]assistItem{}
	var items []assistItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(extractJSONArrayAssist(raw))), &items); err == nil {
		for _, it := range items {
			byID[it.ID] = it
		}
	}
	out := make([]ReadManuallySuggestion, 0, len(batchIDs))
	for _, id := range batchIDs {
		s := ReadManuallySuggestion{ID: id, Action: "read"}
		if it, ok := byID[id]; ok {
			s.Hint = strings.TrimSpace(it.Hint)
			act := strings.ToLower(strings.TrimSpace(it.Action))
			switch {
			case act == "label":
				if canon, ok := available[strings.ToLower(strings.TrimSpace(it.Label))]; ok {
					s.Action, s.Label = "label", canon
				}
			case validAssistActions[act] && act != "read":
				s.Action = act
			}
		}
		out = append(out, s)
	}
	return out
}

// extractJSONArrayAssist returns the substring from the first '[' to the last ']' inclusive.
func extractJSONArrayAssist(raw string) string {
	i, j := strings.Index(raw, "["), strings.LastIndex(raw, "]")
	if i >= 0 && j > i {
		return raw[i : j+1]
	}
	return raw
}

// AssistReadManually is implemented in the next task; stub keeps the package compiling.
func (s *InboxAnalyzerServiceImpl) AssistReadManually(ctx context.Context, msgs []AnalyzerMessage, opts InboxAnalyzerOptions) ([]ReadManuallySuggestion, error) {
	return nil, fmt.Errorf("not implemented")
}
