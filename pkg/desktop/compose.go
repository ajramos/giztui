package desktop

import (
	"context"
	"fmt"
	"strings"
)

// SendMail composes and sends a new message from the active account. cc and bcc
// may be nil/empty.
func (a *API) SendMail(ctx context.Context, to, subject, body string, cc, bcc []string) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("a recipient (to) is required")
	}
	return a.email.SendMessage(ctx, a.accountEmail, to, subject, body, cleanAddrs(cc), cleanAddrs(bcc))
}

// Reply sends a reply to an existing message, preserving threading. cc may be
// nil/empty.
func (a *API) Reply(ctx context.Context, originalID, body string, cc []string) error {
	if strings.TrimSpace(originalID) == "" {
		return fmt.Errorf("original message id is required")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("reply body cannot be empty")
	}
	return a.email.ReplyToMessage(ctx, originalID, body, true, cleanAddrs(cc))
}

// cleanAddrs trims and drops empty entries so the UI can pass raw arrays.
func cleanAddrs(addrs []string) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if s := strings.TrimSpace(a); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
