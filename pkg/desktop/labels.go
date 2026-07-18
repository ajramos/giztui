package desktop

import "context"

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
