package desktop

import (
	"context"
	"fmt"
)

// Unarchive puts a message back in the inbox (undo of Archive) by re-applying
// the INBOX label.
func (a *API) Unarchive(ctx context.Context, id string) error {
	return a.labels.ApplyLabel(ctx, id, "INBOX")
}

// Untrash restores a message from the trash (undo of Trash).
func (a *API) Untrash(ctx context.Context, id string) error {
	if a.draft == nil {
		return fmt.Errorf("gmail client not available")
	}
	return a.draft.UntrashMessage(id)
}

// BulkUnarchive re-applies INBOX to many messages (undo of a bulk archive).
func (a *API) BulkUnarchive(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := a.labels.ApplyLabel(ctx, id, "INBOX"); err != nil {
			return err
		}
	}
	return nil
}

// BulkUntrash restores many messages from the trash (undo of a bulk trash).
func (a *API) BulkUntrash(ctx context.Context, ids []string) error {
	if a.draft == nil {
		return fmt.Errorf("gmail client not available")
	}
	for _, id := range ids {
		if err := a.draft.UntrashMessage(id); err != nil {
			return err
		}
	}
	return nil
}
