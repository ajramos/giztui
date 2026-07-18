package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// draftClient is the subset of *gmail.Client used for draft CRUD. Declaring it
// as an interface keeps the draft API unit-testable.
type draftClient interface {
	ListDrafts(maxResults int64) ([]*gmail_v1.Draft, error)
	CreateDraft(to, subject, body string, cc []string) (string, error)
	UpdateDraft(draftID, to, subject, body string, cc []string) error
	DeleteDraft(draftID string) error
	UntrashMessage(messageID string) error
	GetMessageRaw(messageID string) ([]byte, error)
}

// ListDrafts returns the account's drafts as lightweight summaries.
func (a *API) ListDrafts(ctx context.Context) ([]DraftSummary, error) {
	if a.draft == nil {
		return []DraftSummary{}, nil
	}
	drafts, err := a.draft.ListDrafts(50)
	if err != nil {
		return nil, err
	}
	out := make([]DraftSummary, 0, len(drafts))
	for _, d := range drafts {
		if d == nil {
			continue
		}
		s := DraftSummary{ID: d.Id}
		if d.Message != nil {
			s.Snippet = d.Message.Snippet
			if a.mail != nil {
				s.To = a.mail.ExtractHeader(d.Message, "To")
				s.Subject = a.mail.ExtractHeader(d.Message, "Subject")
			}
		}
		out = append(out, s)
	}
	return out, nil
}

// GetDraft loads a draft for editing, with recipients and body extracted.
func (a *API) GetDraft(ctx context.Context, draftID string) (*DraftDetail, error) {
	if a.composition == nil {
		return nil, fmt.Errorf("composition service not available")
	}
	comp, err := a.composition.LoadDraftComposition(ctx, draftID)
	if err != nil {
		return nil, err
	}
	return &DraftDetail{
		ID:      draftID,
		To:      joinRecipients(comp.To),
		Cc:      joinRecipients(comp.CC),
		Subject: comp.Subject,
		Body:    comp.Body,
	}, nil
}

// SaveDraft creates a new draft and returns its ID.
func (a *API) SaveDraft(ctx context.Context, to, subject, body string, cc []string) (string, error) {
	if a.draft == nil {
		return "", fmt.Errorf("draft service not available")
	}
	return a.draft.CreateDraft(to, subject, body, cleanAddrs(cc))
}

// UpdateDraft overwrites an existing draft.
func (a *API) UpdateDraft(ctx context.Context, draftID, to, subject, body string, cc []string) error {
	if a.draft == nil {
		return fmt.Errorf("draft service not available")
	}
	if strings.TrimSpace(draftID) == "" {
		return fmt.Errorf("draft ID is required")
	}
	return a.draft.UpdateDraft(draftID, to, subject, body, cleanAddrs(cc))
}

// DeleteDraft removes a draft.
func (a *API) DeleteDraft(ctx context.Context, draftID string) error {
	if a.draft == nil {
		return fmt.Errorf("draft service not available")
	}
	return a.draft.DeleteDraft(draftID)
}

// joinRecipients renders a []Recipient as a comma-separated address string.
func joinRecipients(rs []services.Recipient) string {
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		if r.Email != "" {
			parts = append(parts, r.Email)
		}
	}
	return strings.Join(parts, ", ")
}
