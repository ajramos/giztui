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

// buildAssistPrompt renders the assist prompt for one batch: the existing labels are
// injected at {{labels}} and the batch is rendered at {{messages}} (one line per email,
// keyed by concrete message ID so the model echoes it back).
func buildAssistPrompt(batch []AnalyzerMessage, opts InboxAnalyzerOptions) string {
	prompt := assistReadManuallyPrompt
	prompt = strings.ReplaceAll(prompt, "{{labels}}", strings.Join(opts.AvailableLabels, ", "))

	var b strings.Builder
	for _, m := range batch {
		subject := strings.TrimSpace(strings.ReplaceAll(m.Subject, "\n", " "))
		if subject == "" {
			subject = "(no subject)"
		}
		body := m.Body
		if strings.TrimSpace(body) == "" {
			body = m.Snippet
		}
		fmt.Fprintf(&b, "- id: %s | from: %s | subject: %s | %s\n",
			m.ID, m.From, subject, truncateForAnalyzer(body, opts.BodyCharLimit))
	}
	payload := b.String()

	if strings.Contains(prompt, "{{messages}}") {
		return strings.ReplaceAll(prompt, "{{messages}}", payload)
	}
	return prompt + "\n\n" + payload
}

// AssistReadManually enriches read-manually messages on demand: one suggestion per input
// message, in input order. It streams each batch through the AIService and reconciles the
// reply against the batch IDs. A batch whose LLM call fails degrades to {Action:"read"} for
// each of its messages rather than failing the whole pass.
func (s *InboxAnalyzerServiceImpl) AssistReadManually(ctx context.Context, msgs []AnalyzerMessage, opts InboxAnalyzerOptions) ([]ReadManuallySuggestion, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if len(msgs) == 0 {
		return nil, nil
	}

	available := make(map[string]string, len(opts.AvailableLabels))
	for _, name := range opts.AvailableLabels {
		available[strings.ToLower(strings.TrimSpace(name))] = name
	}

	batches := splitBatches(msgs, opts.BatchSize, opts.MaxBatches)
	out := make([]ReadManuallySuggestion, 0, len(msgs))

	for _, batch := range batches {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		batchIDs := make([]string, len(batch))
		for i, m := range batch {
			batchIDs[i] = m.ID
		}

		prompt := buildAssistPrompt(batch, opts)
		raw, err := s.aiService.ApplyCustomPromptStream(ctx, prompt, nil, nil)
		if err != nil {
			// Degrade this batch: surface every message as read-manually so nothing is lost.
			for _, id := range batchIDs {
				out = append(out, ReadManuallySuggestion{ID: id, Action: "read"})
			}
			continue
		}
		out = append(out, parseAssistResponse(raw, batchIDs, available, opts.StrictLabels)...)
	}

	return out, nil
}
