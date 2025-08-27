package services
import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"github.com/ajramos/gmail-tui/internal/gmail"
	"github.com/google/uuid"
)
// UndoServiceImpl implements UndoService
type UndoServiceImpl struct {
	repo         MessageRepository
	labelService LabelService
	gmailClient  *gmail.Client
	lastAction   *UndoableAction
	mu           sync.RWMutex
	logger       *log.Logger // Optional - for debug logging
}
// NewUndoService creates a new undo service
func NewUndoService(repo MessageRepository, labelService LabelService, gmailClient *gmail.Client) *UndoServiceImpl {
	return &UndoServiceImpl{
		repo:         repo,
		labelService: labelService,
		gmailClient:  gmailClient,
	}
}
// SetLogger sets the logger for debug output
func (s *UndoServiceImpl) SetLogger(logger *log.Logger) {
	s.logger = logger
}
// RecordAction records an action for potential undo
func (s *UndoServiceImpl) RecordAction(ctx context.Context, action *UndoableAction) error {
	if action == nil {
		return fmt.Errorf("action cannot be nil")
	}
	// Generate unique ID if not provided
	if action.ID == "" {
		action.ID = uuid.New().String()
	}
	// Set timestamp if not provided
	if action.Timestamp.IsZero() {
		action.Timestamp = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Store the action (single-level undo for MVP)
	s.lastAction = action
	return nil
}
// UndoLastAction undoes the last recorded action
func (s *UndoServiceImpl) UndoLastAction(ctx context.Context) (*UndoResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastAction == nil {
		return &UndoResult{
			Success:     false,
			Description: "No action to undo",
			Errors:      []string{"No undoable action available"},
		}, nil
	}
	action := s.lastAction
	result := &UndoResult{
		Success:      true,
		MessageCount: len(action.MessageIDs),
		Errors:       []string{},
		ActionType:   action.Type,
		MessageIDs:   action.MessageIDs,
		ExtraData:    action.ExtraData,
	}
	// Perform undo based on action type
	switch action.Type {
	case UndoActionArchive:
		result.Description = s.formatUndoDescription("Unarchived", action)
		err := s.undoArchive(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	case UndoActionTrash:
		result.Description = s.formatUndoDescription("Restored from trash", action)
		err := s.undoTrash(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	case UndoActionMarkRead:
		result.Description = s.formatUndoDescription("Marked as unread", action)
		err := s.undoMarkRead(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	case UndoActionMarkUnread:
		result.Description = s.formatUndoDescription("Marked as read", action)
		err := s.undoMarkUnread(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	case UndoActionLabelAdd:
		result.Description = s.formatUndoDescription("Removed labels", action)
		err := s.undoLabelAdd(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	case UndoActionLabelRemove:
		result.Description = s.formatUndoDescription("Re-added labels", action)
		err := s.undoLabelRemove(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	case UndoActionMove:
		// Use proper move undo that removes applied labels
		result.Description = s.formatUndoDescription("Undid move", action)
		err := s.undoMove(ctx, action)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, err.Error())
		}
	default:
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Unknown action type: %s", action.Type))
	}
	// Clear the undo history after performing undo (single-level undo)
	if result.Success {
		s.lastAction = nil
	}
	return result, nil
}
// HasUndoableAction checks if there's an action that can be undone
func (s *UndoServiceImpl) HasUndoableAction() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAction != nil
}
// GetUndoDescription returns a description of what will be undone
func (s *UndoServiceImpl) GetUndoDescription() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastAction == nil {
		return "No action to undo"
	}
	return s.lastAction.Description
}
// ClearUndoHistory clears the undo history
func (s *UndoServiceImpl) ClearUndoHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAction = nil
	return nil
}
// Helper methods for specific undo operations
func (s *UndoServiceImpl) undoArchive(ctx context.Context, action *UndoableAction) error {
	// To undo archive, we need to restore messages to their previous state
	for _, messageID := range action.MessageIDs {
		prevState, exists := action.PrevState[messageID]
		if !exists {
			continue
		}
		// To undo archive: add back INBOX label (archive removes INBOX label)
		updates := MessageUpdates{
			AddLabels: []string{},
		}
		// Add back to inbox if it was there before
		if prevState.IsInInbox {
			updates.AddLabels = append(updates.AddLabels, "INBOX")
		}
		if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
			return fmt.Errorf("failed to undo archive for message %s: %v", messageID, err)
		}
	}
	return nil
}
func (s *UndoServiceImpl) undoTrash(ctx context.Context, action *UndoableAction) error {
	// To undo trash, restore messages to their previous labels
	for _, messageID := range action.MessageIDs {
		prevState, exists := action.PrevState[messageID]
		if !exists {
			continue
		}
		updates := MessageUpdates{
			RemoveLabels: []string{"TRASH"},
			AddLabels:    prevState.Labels,
		}
		if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
			return fmt.Errorf("failed to undo trash for message %s: %v", messageID, err)
		}
	}
	return nil
}
func (s *UndoServiceImpl) undoMarkRead(ctx context.Context, action *UndoableAction) error {
	// Check if this is a toggle operation that needs to restore to previous state
	if operationType, exists := action.ExtraData["operation_type"]; exists && operationType == "toggle_read" {
		// Handle toggle operations by restoring each message to its previous state
		for _, messageID := range action.MessageIDs {
			// Get the previous state for this message
			prevState, exists := action.PrevState[messageID]
			if !exists {
				continue
			}
			// Restore to previous read state
			var updates MessageUpdates
			if prevState.IsRead {
				// Message was read before, restore by removing UNREAD label
				updates = MessageUpdates{
					RemoveLabels: []string{"UNREAD"},
				}
			} else {
				// Message was unread before, restore by adding UNREAD label
				updates = MessageUpdates{
					AddLabels: []string{"UNREAD"},
				}
			}
			if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
				return fmt.Errorf("failed to undo toggle read for message %s: %v", messageID, err)
			}
		}
		return nil
	}
	// Standard mark as read undo: mark as unread
	for _, messageID := range action.MessageIDs {
		updates := MessageUpdates{
			AddLabels: []string{"UNREAD"},
		}
		if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
			return fmt.Errorf("failed to undo mark read for message %s: %v", messageID, err)
		}
	}
	return nil
}
func (s *UndoServiceImpl) undoMarkUnread(ctx context.Context, action *UndoableAction) error {
	// To undo mark as unread, mark as read
	for _, messageID := range action.MessageIDs {
		updates := MessageUpdates{
			RemoveLabels: []string{"UNREAD"},
		}
		if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
			return fmt.Errorf("failed to undo mark unread for message %s: %v", messageID, err)
		}
	}
	return nil
}
func (s *UndoServiceImpl) undoLabelAdd(ctx context.Context, action *UndoableAction) error {
	// To undo label add, remove the labels that were added
	// Use Gmail client directly to avoid circular undo recording
	if labelsToRemove, exists := action.ExtraData["added_labels"].([]string); exists {
		for _, messageID := range action.MessageIDs {
			for _, labelID := range labelsToRemove {
				if err := s.gmailClient.RemoveLabel(messageID, labelID); err != nil {
					return fmt.Errorf("failed to remove label %s from message %s: %v", labelID, messageID, err)
				}
			}
		}
	}
	return nil
}
func (s *UndoServiceImpl) undoLabelRemove(ctx context.Context, action *UndoableAction) error {
	// To undo label remove, re-add the labels that were removed
	// Use Gmail client directly to avoid circular undo recording
	if labelsToAdd, exists := action.ExtraData["removed_labels"].([]string); exists {
		for _, messageID := range action.MessageIDs {
			for _, labelID := range labelsToAdd {
				if err := s.gmailClient.ApplyLabel(messageID, labelID); err != nil {
					return fmt.Errorf("failed to re-add label %s to message %s: %v", labelID, messageID, err)
				}
			}
		}
	}
	return nil
}
func (s *UndoServiceImpl) undoMove(ctx context.Context, action *UndoableAction) error {
	// Move operation consists of: apply label + archive
	// To undo: restore from archive (add INBOX) + remove applied label
	for _, messageID := range action.MessageIDs {
		// First, restore from archive (add INBOX label)
		updates := MessageUpdates{
			AddLabels: []string{"INBOX"},
		}
		if err := s.repo.UpdateMessage(ctx, messageID, updates); err != nil {
			return fmt.Errorf("failed to unarchive moved message %s: %v", messageID, err)
		}
		// Then, remove the applied label using Gmail client directly
		if appliedLabels, exists := action.ExtraData["applied_labels"].([]string); exists {
			for _, labelID := range appliedLabels {
				if err := s.gmailClient.RemoveLabel(messageID, labelID); err != nil {
					return fmt.Errorf("failed to remove applied label %s from message %s: %v", labelID, messageID, err)
				}
			}
		} else {
		}
	}
	return nil
}
// formatUndoDescription creates a human-readable description for undo result
func (s *UndoServiceImpl) formatUndoDescription(actionVerb string, action *UndoableAction) string {
	count := len(action.MessageIDs)
	if count == 1 {
		return fmt.Sprintf("%s message", actionVerb)
	}
	return fmt.Sprintf("%s %d messages", actionVerb, count)
}
// Helper function to capture message state for undo operations
func (s *UndoServiceImpl) CaptureMessageState(ctx context.Context, messageID string) (ActionState, error) {
	// Get current message labels
	labels, err := s.labelService.GetMessageLabels(ctx, messageID)
	if err != nil {
		return ActionState{}, fmt.Errorf("failed to get message labels: %v", err)
	}
	// Check if message is read (doesn't have UNREAD label)
	isRead := true
	isInInbox := false
	for _, label := range labels {
		if label == "UNREAD" {
			isRead = false
		}
		if label == "INBOX" {
			isInInbox = true
		}
	}
	return ActionState{
		Labels:    labels,
		IsRead:    isRead,
		IsInInbox: isInInbox,
	}, nil
}
