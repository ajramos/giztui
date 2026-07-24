package desktop

import (
	"context"
	"fmt"
	"strings"
)

// SuggestLabels asks the AI to suggest labels for a message, given the account's
// existing labels as candidates.
func (a *API) SuggestLabels(ctx context.Context, id string) ([]string, error) {
	if a.ai == nil {
		return nil, fmt.Errorf("AI is not configured")
	}
	msg, err := a.repo.GetMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	content := msg.PlainText
	if strings.TrimSpace(content) == "" {
		content = msg.HTML
	}
	var available []string
	if ls, err := a.labels.ListLabels(ctx); err == nil {
		for _, l := range ls {
			if l == nil {
				continue
			}
			if _, sys := systemLabels[l.Id]; sys || strings.HasPrefix(l.Id, "CATEGORY_") {
				continue
			}
			available = append(available, l.Name)
		}
	}
	return a.ai.SuggestLabels(ctx, content, available)
}

// ApplyLabelByName applies a label by its display name, creating it if needed.
func (a *API) ApplyLabelByName(ctx context.Context, messageID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("label name is required")
	}
	ls, err := a.labels.ListLabels(ctx)
	if err != nil {
		return err
	}
	for _, l := range ls {
		if l != nil && strings.EqualFold(l.Name, name) {
			return a.labels.ApplyLabel(ctx, messageID, l.Id)
		}
	}
	created, err := a.labels.CreateLabel(ctx, name)
	if err != nil {
		return err
	}
	return a.labels.ApplyLabel(ctx, messageID, created.Id)
}

// MoveToLabel moves a message to a label: it applies the label (creating it if
// needed) and archives the message (removes it from the inbox), mirroring the
// TUI's ":move" behavior.
func (a *API) MoveToLabel(ctx context.Context, messageID, name string) error {
	if err := a.ApplyLabelByName(ctx, messageID, name); err != nil {
		return err
	}
	return a.email.ArchiveMessage(ctx, messageID)
}

// MessageLabelIDs returns the raw Gmail label IDs currently applied to a
// message, so the UI can show which labels are checked in the label picker.
func (a *API) MessageLabelIDs(ctx context.Context, id string) ([]string, error) {
	return a.labels.GetMessageLabels(ctx, id)
}

// ApplyLabel adds a label to a message.
func (a *API) ApplyLabel(ctx context.Context, messageID, labelID string) error {
	return a.labels.ApplyLabel(ctx, messageID, labelID)
}

// RemoveLabel removes a label from a message.
func (a *API) RemoveLabel(ctx context.Context, messageID, labelID string) error {
	return a.labels.RemoveLabel(ctx, messageID, labelID)
}
