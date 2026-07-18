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

// SummarizeStream is like Summarize but streams tokens through onToken as they
// are generated, returning the complete summary at the end. The caller (e.g.
// the Wails layer) forwards tokens to the UI, typically via a runtime event.
// When force is true it bypasses the cache and regenerates the summary.
func (a *API) SummarizeStream(ctx context.Context, id string, force bool, onToken func(string)) (string, error) {
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
	res, err := a.ai.GenerateSummaryStream(ctx, content, services.SummaryOptions{
		MessageID:       id,
		AccountEmail:    a.accountEmail,
		StreamEnabled:   true,
		UseCache:        !force,
		ForceRegenerate: force,
	}, onToken)
	if err != nil {
		return "", err
	}
	return res.Summary, nil
}

// GenerateReply asks the AI to draft a reply to a message and returns the draft
// body text (the UI opens it in the composer, prefilled, for the user to edit
// before sending).
func (a *API) GenerateReply(ctx context.Context, id string) (string, error) {
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
		return "", fmt.Errorf("message has no readable content to reply to")
	}
	return a.ai.GenerateReply(ctx, content, services.ReplyOptions{})
}
