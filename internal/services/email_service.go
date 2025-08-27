package services

import (
	"context"
	"fmt"
	"log"
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
	undoService UndoService // Optional - for recording undo actions
	logger      *log.Logger // Optional - for debug logging
}

// NewEmailService creates a new email service
func NewEmailService(repo MessageRepository, gmailClient *gmail.Client, renderer *render.EmailRenderer) *EmailServiceImpl {
	return &EmailServiceImpl{
		repo:        repo,
		gmailClient: gmailClient,
		renderer:    renderer,
	}
}

// SetUndoService sets the undo service for recording undo actions
// This is called after initialization to avoid circular dependencies
func (s *EmailServiceImpl) SetUndoService(undoService UndoService) {
	s.undoService = undoService
}

// SetLogger sets the logger for debug output
func (s *EmailServiceImpl) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// ArchiveMessageAsMove archives a message and records it as a move operation for undo
func (s *EmailServiceImpl) ArchiveMessageAsMove(ctx context.Context, messageID, labelID, labelName string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	if s.logger != nil {
		s.logger.Printf("DEBUG: ArchiveMessageAsMove starting - messageID: %s, labelID: %s, labelName: %s", messageID, labelID, labelName)
	}

	// Record move undo action before performing the operation
	if s.undoService != nil {
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			prevState, err := undoServiceImpl.CaptureMessageState(ctx, messageID)
			if err == nil {
				action := &UndoableAction{
					Type:        UndoActionMove,
					MessageIDs:  []string{messageID},
					PrevState:   map[string]ActionState{messageID: prevState},
					Description: fmt.Sprintf("Moved to %s", labelName),
					IsBulk:      false,
					ExtraData: map[string]interface{}{
						"applied_labels": []string{labelID},
						"label_name":     labelName,
					},
				}
				if s.logger != nil {
					s.logger.Printf("DEBUG: ArchiveMessageAsMove recording undo action: %+v", action)
				}
				s.undoService.RecordAction(ctx, action)
			} else {
				if s.logger != nil {
					s.logger.Printf("DEBUG: ArchiveMessageAsMove failed to capture message state: %v", err)
				}
			}
		} else {
			if s.logger != nil {
				s.logger.Printf("DEBUG: ArchiveMessageAsMove undoService is not UndoServiceImpl type")
			}
		}
	} else {
		if s.logger != nil {
			s.logger.Printf("DEBUG: ArchiveMessageAsMove undoService is nil")
		}
	}

	// Perform the archive operation
	updates := MessageUpdates{
		RemoveLabels: []string{"INBOX"},
	}

	if s.logger != nil {
		s.logger.Printf("DEBUG: ArchiveMessageAsMove performing archive operation")
	}
	return s.repo.UpdateMessage(ctx, messageID, updates)
}

func (s *EmailServiceImpl) MarkAsRead(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	// Record undo action before performing the operation
	if s.undoService != nil {
		// Capture current state for undo
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			prevState, err := undoServiceImpl.CaptureMessageState(ctx, messageID)
			if err == nil {
				action := &UndoableAction{
					Type:        UndoActionMarkRead,
					MessageIDs:  []string{messageID},
					PrevState:   map[string]ActionState{messageID: prevState},
					Description: "Mark as read",
					IsBulk:      false,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
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

	// Record undo action before performing the operation
	if s.undoService != nil {
		// Capture current state for undo
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			prevState, err := undoServiceImpl.CaptureMessageState(ctx, messageID)
			if err == nil {
				action := &UndoableAction{
					Type:        UndoActionMarkUnread,
					MessageIDs:  []string{messageID},
					PrevState:   map[string]ActionState{messageID: prevState},
					Description: "Mark as unread",
					IsBulk:      false,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
	}

	updates := MessageUpdates{
		AddLabels: []string{"UNREAD"},
	}

	return s.repo.UpdateMessage(ctx, messageID, updates)
}

// BulkMarkAsRead marks multiple messages as read
func (s *EmailServiceImpl) BulkMarkAsRead(ctx context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return fmt.Errorf("no message IDs provided")
	}

	// Record bulk undo action before performing operations
	if s.undoService != nil {
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			// Capture state for all messages
			prevStates := make(map[string]ActionState)
			for _, id := range messageIDs {
				if prevState, err := undoServiceImpl.CaptureMessageState(ctx, id); err == nil {
					prevStates[id] = prevState
				}
			}

			// Record single bulk undo action
			if len(prevStates) > 0 {
				action := &UndoableAction{
					Type:        UndoActionMarkRead,
					MessageIDs:  messageIDs,
					PrevState:   prevStates,
					Description: "Mark messages as read",
					IsBulk:      true,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
	}

	// Perform the actual operations using repository directly (to avoid double undo recording)
	var errs []string
	for _, id := range messageIDs {
		updates := MessageUpdates{
			RemoveLabels: []string{"UNREAD"},
		}
		if err := s.repo.UpdateMessage(ctx, id, updates); err != nil {
			errs = append(errs, fmt.Sprintf("failed to mark as read %s: %v", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk mark as read errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// BulkMarkAsUnread marks multiple messages as unread
func (s *EmailServiceImpl) BulkMarkAsUnread(ctx context.Context, messageIDs []string) error {
	if len(messageIDs) == 0 {
		return fmt.Errorf("no message IDs provided")
	}

	// Record bulk undo action before performing operations
	if s.undoService != nil {
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			// Capture state for all messages
			prevStates := make(map[string]ActionState)
			for _, id := range messageIDs {
				if prevState, err := undoServiceImpl.CaptureMessageState(ctx, id); err == nil {
					prevStates[id] = prevState
				}
			}

			// Record single bulk undo action
			if len(prevStates) > 0 {
				action := &UndoableAction{
					Type:        UndoActionMarkUnread,
					MessageIDs:  messageIDs,
					PrevState:   prevStates,
					Description: "Mark messages as unread",
					IsBulk:      true,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
	}

	// Perform the actual operations using repository directly (to avoid double undo recording)
	var errs []string
	for _, id := range messageIDs {
		updates := MessageUpdates{
			AddLabels: []string{"UNREAD"},
		}
		if err := s.repo.UpdateMessage(ctx, id, updates); err != nil {
			errs = append(errs, fmt.Sprintf("failed to mark as unread %s: %v", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk mark as unread errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

func (s *EmailServiceImpl) ArchiveMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	// Record undo action before performing the operation
	if s.undoService != nil {
		// Capture current state for undo
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			prevState, err := undoServiceImpl.CaptureMessageState(ctx, messageID)
			if err == nil {
				action := &UndoableAction{
					Type:        UndoActionArchive,
					MessageIDs:  []string{messageID},
					PrevState:   map[string]ActionState{messageID: prevState},
					Description: "Archive message",
					IsBulk:      false,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
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

	// Record undo action before performing the operation
	if s.undoService != nil {
		// Capture current state for undo
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			prevState, err := undoServiceImpl.CaptureMessageState(ctx, messageID)
			if err == nil {
				action := &UndoableAction{
					Type:        UndoActionTrash,
					MessageIDs:  []string{messageID},
					PrevState:   map[string]ActionState{messageID: prevState},
					Description: "Trash message",
					IsBulk:      false,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
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

	// Record bulk undo action before performing operations
	if s.undoService != nil {
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			// Capture state for all messages
			prevStates := make(map[string]ActionState)
			for _, id := range messageIDs {
				if prevState, err := undoServiceImpl.CaptureMessageState(ctx, id); err == nil {
					prevStates[id] = prevState
				}
			}

			// Record single bulk undo action
			if len(prevStates) > 0 {
				action := &UndoableAction{
					Type:        UndoActionArchive,
					MessageIDs:  messageIDs,
					PrevState:   prevStates,
					Description: "Archive messages",
					IsBulk:      true,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
	}

	// Perform the actual archiving using repository directly (to avoid double undo recording)
	var errs []string
	for _, id := range messageIDs {
		updates := MessageUpdates{
			RemoveLabels: []string{"INBOX"},
		}
		if err := s.repo.UpdateMessage(ctx, id, updates); err != nil {
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

	// Record bulk undo action before performing operations
	if s.undoService != nil {
		if undoServiceImpl, ok := s.undoService.(*UndoServiceImpl); ok {
			// Capture state for all messages
			prevStates := make(map[string]ActionState)
			for _, id := range messageIDs {
				if prevState, err := undoServiceImpl.CaptureMessageState(ctx, id); err == nil {
					prevStates[id] = prevState
				}
			}

			// Record single bulk undo action
			if len(prevStates) > 0 {
				action := &UndoableAction{
					Type:        UndoActionTrash,
					MessageIDs:  messageIDs,
					PrevState:   prevStates,
					Description: "Trash messages",
					IsBulk:      true,
				}
				s.undoService.RecordAction(ctx, action)
			}
		}
	}

	// Perform the actual trashing using Gmail client directly (to avoid double undo recording)
	var errs []string
	for _, id := range messageIDs {
		if err := s.gmailClient.TrashMessage(id); err != nil {
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
