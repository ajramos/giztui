package desktop

import (
	"context"
	"fmt"
	"strings"
)

// PromptsEnabled reports whether the prompt library is available (requires both
// an LLM provider and a local database).
func (a *API) PromptsEnabled() bool { return a.prompts != nil }

// ListPrompts returns all saved AI prompt templates.
func (a *API) ListPrompts(ctx context.Context) ([]Prompt, error) {
	if a.prompts == nil {
		return []Prompt{}, nil
	}
	templates, err := a.prompts.ListPrompts(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]Prompt, 0, len(templates))
	for _, t := range templates {
		if t == nil {
			continue
		}
		out = append(out, Prompt{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
		})
	}
	return out, nil
}

// ApplyPromptStream applies a saved prompt to a message and streams the result
// through onToken, returning the full result text at the end.
func (a *API) ApplyPromptStream(ctx context.Context, messageID string, promptID int, onToken func(string)) (string, error) {
	if a.prompts == nil {
		return "", fmt.Errorf("prompts are not available; enable an LLM provider and local database")
	}
	msg, err := a.repo.GetMessage(ctx, messageID)
	if err != nil {
		return "", err
	}
	content := msg.PlainText
	if strings.TrimSpace(content) == "" {
		content = msg.HTML
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("message has no readable content")
	}
	res, err := a.prompts.ApplyPromptStream(ctx, content, promptID, nil, onToken)
	if err != nil {
		return "", err
	}
	return res.ResultText, nil
}

// ApplyBulkPromptStream applies a saved prompt across many messages, streaming
// the combined result.
func (a *API) ApplyBulkPromptStream(ctx context.Context, ids []string, promptID int, onToken func(string)) (string, error) {
	if a.prompts == nil {
		return "", fmt.Errorf("prompts are not available")
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no messages selected")
	}
	res, err := a.prompts.ApplyBulkPromptStream(ctx, a.accountEmail, ids, promptID, nil, onToken)
	if err != nil {
		return "", err
	}
	return res.Summary, nil
}
