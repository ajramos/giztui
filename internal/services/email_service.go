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
				}
				s.undoService.RecordAction(ctx, action)
			} else {
				if s.logger != nil {
				}
			}
		} else {
			if s.logger != nil {
			}
		}
	} else {
		if s.logger != nil {
		}
	}

	// Perform the archive operation
	updates := MessageUpdates{
		RemoveLabels: []string{"INBOX"},
	}

	if s.logger != nil {
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

func (s *EmailServiceImpl) SendMessage(ctx context.Context, from, to, subject, body string, cc, bcc []string) error {
	if to == "" || subject == "" || body == "" {
		return fmt.Errorf("to, subject, and body cannot be empty")
	}

	_, err := s.gmailClient.SendMessage(from, to, subject, body, cc, bcc)
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

// MoveToSystemFolder moves a message to a system folder (Inbox, Trash, Spam) with undo support
func (s *EmailServiceImpl) MoveToSystemFolder(ctx context.Context, messageID, systemFolderID, folderName string) error {
	if messageID == "" || systemFolderID == "" {
		return fmt.Errorf("messageID and systemFolderID cannot be empty")
	}

	if s.logger != nil {
		s.logger.Printf("Moving message %s to system folder %s (%s)", messageID, systemFolderID, folderName)
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
					Description: fmt.Sprintf("Moved to %s", folderName),
					IsBulk:      false,
					ExtraData: map[string]interface{}{
						"applied_labels": []string{systemFolderID},
						"label_name":     folderName,
						"system_folder":  true,
					},
				}
				if s.logger != nil {
					s.logger.Printf("Recording undo action for system folder move: %s -> %s", messageID, folderName)
				}
				s.undoService.RecordAction(ctx, action)
			} else {
				if s.logger != nil {
					s.logger.Printf("Failed to capture message state for undo: %v", err)
				}
			}
		}
	}

	// Perform the system folder move operation based on folder type
	switch systemFolderID {
	case "INBOX":
		// Move to Inbox: Add INBOX, conditionally remove TRASH/SPAM only if they exist
		if s.logger != nil {
			s.logger.Printf("=== INBOX MOVE DEBUG ===")
			s.logger.Printf("Moving message %s to INBOX", messageID)
		}
		
		// First, add the INBOX label
		updates := MessageUpdates{
			AddLabels: []string{"INBOX"},
		}
		
		if s.logger != nil {
			s.logger.Printf("About to call s.repo.UpdateMessage to ADD INBOX label to message %s", messageID)
			s.logger.Printf("UpdateMessage will call r.gmailClient.ApplyLabel(messageID=%s, labelID=%s)", messageID, "INBOX")
		}
		
		// Apply inbox label first
		if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
			if s.logger != nil {
				s.logger.Printf("ERROR: Failed to add INBOX label to message %s: %v", messageID, err)
				s.logger.Printf("This means the Gmail API Users.Messages.Modify call failed")
			}
			return fmt.Errorf("failed to add INBOX label: %w", err)
		}
		
		if s.logger != nil {
			s.logger.Printf("SUCCESS: Added INBOX label to message %s", messageID)
			s.logger.Printf("Gmail API Users.Messages.Modify succeeded for INBOX label")
		}
		
		// Then try to remove TRASH and SPAM labels individually, ignoring errors if they don't exist
		// This prevents the entire operation from failing if the message doesn't have these labels
		if s.logger != nil {
			s.logger.Printf("Attempting to remove TRASH label from message %s", messageID)
		}
		trashUpdates := MessageUpdates{RemoveLabels: []string{"TRASH"}}
		if err := s.repo.UpdateMessage(ctx, messageID, trashUpdates); err != nil {
			if s.logger != nil {
				s.logger.Printf("Expected error removing TRASH label (probably doesn't exist): %v", err)
			}
		} else {
			if s.logger != nil {
				s.logger.Printf("Successfully removed TRASH label from message %s", messageID)
			}
		}
		
		if s.logger != nil {
			s.logger.Printf("Attempting to remove SPAM label from message %s", messageID)
		}
		spamUpdates := MessageUpdates{RemoveLabels: []string{"SPAM"}}
		if err := s.repo.UpdateMessage(ctx, messageID, spamUpdates); err != nil {
			if s.logger != nil {
				s.logger.Printf("Expected error removing SPAM label (probably doesn't exist): %v", err)
			}
		} else {
			if s.logger != nil {
				s.logger.Printf("Successfully removed SPAM label from message %s", messageID)
			}
		}
		
		if s.logger != nil {
			s.logger.Printf("=== INBOX MOVE COMPLETED ===")
		}
		
		return nil

	case "TRASH":
		// Move to Trash: Add TRASH, remove INBOX
		updates := MessageUpdates{
			AddLabels:    []string{"TRASH"},
			RemoveLabels: []string{"INBOX"},
		}
		return s.repo.UpdateMessage(ctx, messageID, updates)

	case "SPAM":
		// Move to Spam: Add SPAM, remove INBOX
		updates := MessageUpdates{
			AddLabels:    []string{"SPAM"},
			RemoveLabels: []string{"INBOX"},
		}
		return s.repo.UpdateMessage(ctx, messageID, updates)

	default:
		return fmt.Errorf("unsupported system folder: %s", systemFolderID)
	}
}
