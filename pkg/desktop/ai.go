package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

// AIEnabled reports whether an LLM provider is configured, so the UI can hide
// AI actions when unavailable.
func (a *API) AIEnabled() bool { return a.ai != nil }

// Summarize returns an AI-generated summary of a message's body. It reuses the
// same AIService the TUI uses. Returns a clear error when AI is not configured.
func (a *API) Summarize(ctx context.Context, id string) (string, error) {
	if a.ai == nil {
		return "", fmt.Errorf("AI is not configured; enable an LLM provider in your GizTUI config")
	}
	msg, err := a.repo.GetMessage(ctx, id)
	if err != nil {
		return "", err
	}
	content := msg.PlainText
	if strings.TrimSpace(content) == "" {
		content = msg.HTML
	}
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("message has no readable content to summarize")
	}
	res, err := a.ai.GenerateSummary(ctx, content, services.SummaryOptions{
		MessageID:    id,
		AccountEmail: a.accountEmail,
	})
	if err != nil {
		return "", err
	}
	return res.Summary, nil
}
