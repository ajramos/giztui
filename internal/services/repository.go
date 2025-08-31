package services

import (
	"context"
	"fmt"

	"github.com/ajramos/gmail-tui/internal/gmail"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// MessageRepositoryImpl implements MessageRepository
type MessageRepositoryImpl struct {
	gmailClient *gmail.Client
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(gmailClient *gmail.Client) *MessageRepositoryImpl {
	return &MessageRepositoryImpl{
		gmailClient: gmailClient,
	}
}

func (r *MessageRepositoryImpl) GetMessages(ctx context.Context, opts QueryOptions) (*MessagePage, error) {
	messages, nextToken, err := r.gmailClient.ListMessagesPage(opts.MaxResults, opts.PageToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return &MessagePage{
		Messages:      messages,
		NextPageToken: nextToken,
		TotalCount:    len(messages),
	}, nil
}

func (r *MessageRepositoryImpl) GetMessage(ctx context.Context, id string) (*gmail.Message, error) {
	if id == "" {
		return nil, fmt.Errorf("message ID cannot be empty")
	}

	msg, err := r.gmailClient.GetMessageWithContent(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get message %s: %w", id, err)
	}

	return msg, nil
}

func (r *MessageRepositoryImpl) SearchMessages(ctx context.Context, query string, opts QueryOptions) (*MessagePage, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	messages, nextToken, err := r.gmailClient.SearchMessagesPage(query, opts.MaxResults, opts.PageToken)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}

	return &MessagePage{
		Messages:      messages,
		NextPageToken: nextToken,
		TotalCount:    len(messages),
	}, nil
}

func (r *MessageRepositoryImpl) UpdateMessage(ctx context.Context, id string, updates MessageUpdates) error {
	if id == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	// Apply label additions
	for _, labelID := range updates.AddLabels {
		if err := r.gmailClient.ApplyLabel(id, labelID); err != nil {
			return fmt.Errorf("failed to add label %s to message %s: %w", labelID, id, err)
		}
	}

	// Apply label removals
	for _, labelID := range updates.RemoveLabels {
		if err := r.gmailClient.RemoveLabel(id, labelID); err != nil {
			return fmt.Errorf("failed to remove label %s from message %s: %w", labelID, id, err)
		}
	}

	// Handle read/unread status
	if updates.MarkAsRead != nil {
		if *updates.MarkAsRead {
			if err := r.gmailClient.MarkAsRead(id); err != nil {
				return fmt.Errorf("failed to mark message as read: %w", err)
			}
		} else {
			if err := r.gmailClient.MarkAsUnread(id); err != nil {
				return fmt.Errorf("failed to mark message as unread: %w", err)
			}
		}
	}

	return nil
}

func (r *MessageRepositoryImpl) GetDrafts(ctx context.Context, maxResults int64) ([]*gmail_v1.Draft, error) {
	drafts, err := r.gmailClient.ListDrafts(maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to get drafts: %w", err)
	}

	return drafts, nil
}

func (r *MessageRepositoryImpl) GetDraft(ctx context.Context, draftID string) (*gmail_v1.Draft, error) {
	draft, err := r.gmailClient.GetDraft(draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft %s: %w", draftID, err)
	}

	return draft, nil
}
