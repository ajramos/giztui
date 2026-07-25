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

// GetPrompt returns a single prompt template including its editable text.
func (a *API) GetPrompt(ctx context.Context, id int) (*PromptDetail, error) {
	if a.prompts == nil {
		return nil, fmt.Errorf("prompts are not available")
	}
	t, err := a.prompts.GetPrompt(ctx, id)
	if err != nil {
		return nil, err
	}
	return &PromptDetail{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Category:    t.Category,
		Text:        t.PromptText,
	}, nil
}

// CreatePrompt saves a new prompt template and returns its id.
func (a *API) CreatePrompt(ctx context.Context, name, description, text, category string) (int, error) {
	if a.prompts == nil {
		return 0, fmt.Errorf("prompts are not available")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("name and prompt text are required")
	}
	return a.prompts.CreatePrompt(ctx, name, description, text, category)
}

// UpdatePrompt edits an existing prompt template.
func (a *API) UpdatePrompt(ctx context.Context, id int, name, description, text, category string) error {
	if a.prompts == nil {
		return fmt.Errorf("prompts are not available")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(text) == "" {
		return fmt.Errorf("name and prompt text are required")
	}
	return a.prompts.UpdatePrompt(ctx, id, name, description, text, category)
}

// DeletePrompt removes a prompt template.
func (a *API) DeletePrompt(ctx context.Context, id int) error {
	if a.prompts == nil {
		return fmt.Errorf("prompts are not available")
	}
	return a.prompts.DeletePrompt(ctx, id)
}

// RefinePromptText asks the AI to improve a prompt template's text, returning
// the refined version (mirrors the TUI's prompt-refine action).
func (a *API) RefinePromptText(ctx context.Context, text string) (string, error) {
	if a.ai == nil {
		return "", fmt.Errorf("AI is not configured")
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("prompt text is required")
	}
	meta := "You are a prompt engineer. Improve the following email-assistant prompt " +
		"so it is clearer and more effective. Keep any {{body}} or {{messages}} " +
		"placeholders intact. Return ONLY the improved prompt text, with no " +
		"commentary.\n\nPROMPT:\n{{body}}"
	return a.ai.ApplyCustomPrompt(ctx, meta, map[string]string{"body": text})
}

// ApplyPromptStream applies a saved prompt to a message and streams the result
// through onToken, returning the full result text at the end.
func (a *API) ApplyPromptStream(ctx context.Context, messageID string, promptID int, onToken func(string)) (string, error) {
	if a.prompts == nil {
		return "", fmt.Errorf("prompts are not available; enable an LLM provider and local database")
	}
	// Reuse a persisted result for this (account, message, prompt) so re-running a
	// prompt — even in a later session — never re-hits the LLM. Stream the cached
	// text through onToken so the UI renders it just like a fresh run.
	if cached, err := a.prompts.GetCachedResult(ctx, a.accountEmail, messageID, promptID); err == nil && cached != nil && strings.TrimSpace(cached.ResultText) != "" {
		if onToken != nil {
			onToken(cached.ResultText)
		}
		return cached.ResultText, nil
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
	// Persist so it survives the session (best-effort; a save failure shouldn't
	// fail the prompt the user just successfully ran).
	if a.accountEmail != "" {
		_ = a.prompts.SaveResult(ctx, a.accountEmail, messageID, promptID, res.ResultText)
	}
	return res.ResultText, nil
}

// CachedPromptResult is a persisted single-message prompt result, returned so the
// frontend can restore the reader's AI panels across sessions.
type CachedPromptResult struct {
	PromptID int    `json:"promptId"`
	Name     string `json:"name"`
	Text     string `json:"text"`
}

// CachedPrompts returns the persisted prompt results for a message (latest per
// prompt, most recent first). Empty when prompts/DB aren't available.
func (a *API) CachedPrompts(ctx context.Context, messageID string) ([]CachedPromptResult, error) {
	if a.prompts == nil || a.accountEmail == "" {
		return nil, nil
	}
	results, err := a.prompts.GetCachedResultsForMessage(ctx, a.accountEmail, messageID)
	if err != nil || len(results) == 0 {
		return nil, err
	}
	names := make(map[int]string)
	if list, err := a.prompts.ListPrompts(ctx, ""); err == nil {
		for _, p := range list {
			if p != nil {
				names[p.ID] = p.Name
			}
		}
	}
	out := make([]CachedPromptResult, 0, len(results))
	for _, r := range results {
		if r == nil {
			continue
		}
		out = append(out, CachedPromptResult{PromptID: r.PromptID, Name: names[r.PromptID], Text: r.ResultText})
	}
	return out, nil
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
