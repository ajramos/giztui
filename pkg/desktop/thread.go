package desktop

import (
	"context"
	"fmt"

	"github.com/ajramos/giztui/internal/services"
)

// ThreadingEnabled reports whether thread/conversation features are available.
func (a *API) ThreadingEnabled() bool { return a.thread != nil }

// GetThread returns all messages in a thread (oldest first) as full details,
// for the conversation view.
func (a *API) GetThread(ctx context.Context, threadID string) ([]MessageDetail, error) {
	if a.thread == nil {
		return nil, fmt.Errorf("threading is not available")
	}
	raws, err := a.thread.GetThreadMessages(ctx, threadID, services.MessageQueryOptions{
		Format:    "full",
		SortOrder: "asc",
	})
	if err != nil {
		return nil, err
	}
	out := make([]MessageDetail, 0, len(raws))
	for _, r := range raws {
		if r == nil {
			continue
		}
		d, err := a.GetMessage(ctx, r.Id)
		if err != nil || d == nil {
			continue
		}
		out = append(out, *d)
	}
	return out, nil
}

// ThreadSummaryStream streams an AI summary of a whole conversation.
func (a *API) ThreadSummaryStream(ctx context.Context, threadID string, onToken func(string)) (string, error) {
	if a.thread == nil {
		return "", fmt.Errorf("threading is not available")
	}
	res, err := a.thread.GenerateThreadSummaryStream(ctx, threadID, services.ThreadSummaryOptions{
		StreamEnabled: true,
		AccountEmail:  a.accountEmail,
		SummaryType:   "conversation",
	}, onToken)
	if err != nil {
		return "", err
	}
	return res.Summary, nil
}
