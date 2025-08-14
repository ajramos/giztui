package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/ajramos/gmail-tui/internal/render"
)

// EmailServiceImpl implements EmailService
type EmailServiceImpl struct {
	repo        MessageRepository
	gmailClient *gmail.Client
	renderer    *render.EmailRenderer
}

// NewEmailService creates a new email service
func NewEmailService(repo MessageRepository, gmailClient *gmail.Client, renderer *render.EmailRenderer) *EmailServiceImpl {
	return &EmailServiceImpl{
		repo:        repo,
		gmailClient: gmailClient,
		renderer:    renderer,
	}
}

func (s *EmailServiceImpl) MarkAsRead(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	updates := MessageUpdates{
		RemoveLabels: []string{"UNREAD"},
	}

	return s.repo.UpdateMessage(ctx, messageID, updates)
}

func (s *EmailServiceImpl) MarkAsUnread(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	updates := MessageUpdates{
		AddLabels: []string{"UNREAD"},
	}

	return s.repo.UpdateMessage(ctx, messageID, updates)
}

func (s *EmailServiceImpl) ArchiveMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	updates := MessageUpdates{
		RemoveLabels: []string{"INBOX"},
	}

	return s.repo.UpdateMessage(ctx, messageID, updates)
}

func (s *EmailServiceImpl) TrashMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	return s.gmailClient.TrashMessage(messageID)
}

func (s *EmailServiceImpl) SendMessage(ctx context.Context, from, to, subject, body string) error {
	if to == "" || subject == "" || body == "" {
		return fmt.Errorf("to, subject, and body cannot be empty")
	}

	_, err := s.gmailClient.SendMessage(from, to, subject, body)
	return err
}

func (s *EmailServiceImpl) ReplyToMessage(ctx context.Context, originalID, replyBody string, send bool, cc []string) error {
	if originalID == "" || replyBody == "" {
		return fmt.Errorf("originalID and replyBody cannot be empty")
	}

	_, err := s.gmailClient.ReplyMessage(originalID, replyBody, send, cc)
	return err
}

func (s *EmailServiceImpl) BulkArchive(ctx context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return fmt.Errorf("no message IDs provided")
	}

	var errs []string
	for _, id := range messageIDs {
		if err := s.ArchiveMessage(ctx, id); err != nil {
			errs = append(errs, fmt.Sprintf("failed to archive %s: %v", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk archive errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (s *EmailServiceImpl) BulkTrash(ctx context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return fmt.Errorf("no message IDs provided")
	}

	var errs []string
	for _, id := range messageIDs {
		if err := s.TrashMessage(ctx, id); err != nil {
			errs = append(errs, fmt.Sprintf("failed to trash %s: %v", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk trash errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (s *EmailServiceImpl) SaveMessageToFile(ctx context.Context, messageID, filePath string) error {
	if messageID == "" || filePath == "" {
		return fmt.Errorf("messageID and filePath cannot be empty")
	}

	// Get message with content
	msg, err := s.gmailClient.GetMessageWithContent(messageID)
	if err != nil {
		return fmt.Errorf("failed to get message content: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Render message content
	header := fmt.Sprintf("Subject: %s\nFrom: %s\nTo: %s\nDate: %s\n\n",
		msg.Subject, msg.From, msg.To, msg.Date.Format(time.RFC822))

	content := header + msg.PlainText

	// Write to file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
