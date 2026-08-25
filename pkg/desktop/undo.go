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

// Untrash restores a message from the trash (undo of Trash). untrash clears the
// TRASH label but does not reliably restore INBOX, and the inbox list is fetched
// filtered by the INBOX label — so re-apply INBOX (as Unarchive does) or the
// message leaves the Trash yet still never returns to the inbox on reload.
func (a *API) Untrash(ctx context.Context, id string) error {
	if a.draft == nil {
		return fmt.Errorf("gmail client not available")
	}
	if err := a.draft.UntrashMessage(id); err != nil {
		return err
	}
	return a.labels.ApplyLabel(ctx, id, "INBOX")
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
// Like Untrash, each message is untrashed and then re-added to INBOX so it
// reappears in the inbox list.
func (a *API) BulkUntrash(ctx context.Context, ids []string) error {
	if a.draft == nil {
		return fmt.Errorf("gmail client not available")
	}
	for _, id := range ids {
		if err := a.draft.UntrashMessage(id); err != nil {
			return err
		}
		if err := a.labels.ApplyLabel(ctx, id, "INBOX"); err != nil {
			return err
		}
	}
	return nil
}
