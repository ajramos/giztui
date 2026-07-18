package desktop

import "context"

// Bulk operations forward to the service layer's existing Bulk* methods, which
// already parallelize and (optionally) report progress. Progress reporting is
// omitted here for now; the desktop UI updates optimistically.

// BulkArchive archives every message in ids.
func (a *API) BulkArchive(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return a.email.BulkArchive(ctx, ids)
}

// BulkTrash trashes every message in ids.
func (a *API) BulkTrash(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return a.email.BulkTrash(ctx, ids)
}

// BulkMarkRead marks every message in ids as read.
func (a *API) BulkMarkRead(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return a.email.BulkMarkAsRead(ctx, ids)
}

// BulkMarkUnread marks every message in ids as unread.
func (a *API) BulkMarkUnread(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return a.email.BulkMarkAsUnread(ctx, ids)
}

// BulkApplyLabel applies a label to every message in ids.
func (a *API) BulkApplyLabel(ctx context.Context, ids []string, labelID string) error {
	if len(ids) == 0 {
		return nil
	}
	return a.labels.BulkApplyLabel(ctx, ids, labelID)
}

// BulkRemoveLabel removes a label from every message in ids.
func (a *API) BulkRemoveLabel(ctx context.Context, ids []string, labelID string) error {
	if len(ids) == 0 {
		return nil
	}
	return a.labels.BulkRemoveLabel(ctx, ids, labelID)
}
