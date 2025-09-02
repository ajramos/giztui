package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/gmail"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// LabelServiceImpl implements LabelService
type LabelServiceImpl struct {
	gmailClient *gmail.Client
	undoService UndoService // Optional - for recording undo actions
}

// NewLabelService creates a new label service
func NewLabelService(gmailClient *gmail.Client) *LabelServiceImpl {
	return &LabelServiceImpl{
		gmailClient: gmailClient,
	}
}

// SetUndoService sets the undo service for recording undo actions
// This is called after initialization to avoid circular dependencies
func (s *LabelServiceImpl) SetUndoService(undoService UndoService) {
	s.undoService = undoService
}

func (s *LabelServiceImpl) ListLabels(ctx context.Context) ([]*gmail_v1.Label, error) {
	labels, err := s.gmailClient.ListLabels()
	if err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}

	return labels, nil
}

func (s *LabelServiceImpl) CreateLabel(ctx context.Context, name string) (*gmail_v1.Label, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("label name cannot be empty")
	}

	label, err := s.gmailClient.CreateLabel(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create label: %w", err)
	}

	return label, nil
}

func (s *LabelServiceImpl) RenameLabel(ctx context.Context, labelID, newName string) (*gmail_v1.Label, error) {
	if strings.TrimSpace(labelID) == "" || strings.TrimSpace(newName) == "" {
		return nil, fmt.Errorf("labelID and newName cannot be empty")
	}

	label, err := s.gmailClient.RenameLabel(labelID, newName)
	if err != nil {
		return nil, fmt.Errorf("failed to rename label: %w", err)
	}

	return label, nil
}

func (s *LabelServiceImpl) DeleteLabel(ctx context.Context, labelID string) error {
	if strings.TrimSpace(labelID) == "" {
		return fmt.Errorf("labelID cannot be empty")
	}

	if err := s.gmailClient.DeleteLabel(labelID); err != nil {
		return fmt.Errorf("failed to delete label: %w", err)
	}

	return nil
}

func (s *LabelServiceImpl) ApplyLabel(ctx context.Context, messageID, labelID string) error {
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(labelID) == "" {
		return fmt.Errorf("messageID and labelID cannot be empty")
	}

	// Record undo action before performing the operation
	if s.undoService != nil {
		action := &UndoableAction{
			Type:        UndoActionLabelAdd,
			MessageIDs:  []string{messageID},
			Description: fmt.Sprintf("Apply label"),
			IsBulk:      false,
			ExtraData: map[string]interface{}{
				"added_labels": []string{labelID},
			},
		}
		s.undoService.RecordAction(ctx, action)
	}

	if err := s.gmailClient.ApplyLabel(messageID, labelID); err != nil {
		return fmt.Errorf("failed to apply label: %w", err)
	}

	return nil
}

func (s *LabelServiceImpl) RemoveLabel(ctx context.Context, messageID, labelID string) error {
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(labelID) == "" {
		return fmt.Errorf("messageID and labelID cannot be empty")
	}

	// Record undo action before performing the operation
	if s.undoService != nil {
		action := &UndoableAction{
			Type:        UndoActionLabelRemove,
			MessageIDs:  []string{messageID},
			Description: fmt.Sprintf("Remove label"),
			IsBulk:      false,
			ExtraData: map[string]interface{}{
				"removed_labels": []string{labelID},
			},
		}
		s.undoService.RecordAction(ctx, action)
	}

	if err := s.gmailClient.RemoveLabel(messageID, labelID); err != nil {
		return fmt.Errorf("failed to remove label: %w", err)
	}

	return nil
}

func (s *LabelServiceImpl) GetMessageLabels(ctx context.Context, messageID string) ([]string, error) {
	if strings.TrimSpace(messageID) == "" {
		return nil, fmt.Errorf("messageID cannot be empty")
	}

	message, err := s.gmailClient.GetMessage(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return s.gmailClient.ExtractLabels(message), nil
}

// BulkApplyLabel applies a label to multiple messages
func (s *LabelServiceImpl) BulkApplyLabel(ctx context.Context, messageIDs []string, labelID string) error {
	if len(messageIDs) == 0 {
		return fmt.Errorf("no message IDs provided")
	}
	if strings.TrimSpace(labelID) == "" {
		return fmt.Errorf("labelID cannot be empty")
	}

	// Record bulk undo action before performing operations
	if s.undoService != nil {
		action := &UndoableAction{
			Type:        UndoActionLabelAdd,
			MessageIDs:  messageIDs,
			Description: "Apply label to messages",
			IsBulk:      true,
			ExtraData: map[string]interface{}{
				"added_labels": []string{labelID},
			},
		}
		s.undoService.RecordAction(ctx, action)
	}

	// Apply label to all messages using Gmail client directly (to avoid double undo recording)
	var errs []string
	for _, messageID := range messageIDs {
		if err := s.gmailClient.ApplyLabel(messageID, labelID); err != nil {
			errs = append(errs, fmt.Sprintf("failed to apply label to %s: %v", messageID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk apply label errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// BulkRemoveLabel removes a label from multiple messages
func (s *LabelServiceImpl) BulkRemoveLabel(ctx context.Context, messageIDs []string, labelID string) error {
	if len(messageIDs) == 0 {
		return fmt.Errorf("no message IDs provided")
	}
	if strings.TrimSpace(labelID) == "" {
		return fmt.Errorf("labelID cannot be empty")
	}

	// Record bulk undo action before performing operations
	if s.undoService != nil {
		action := &UndoableAction{
			Type:        UndoActionLabelRemove,
			MessageIDs:  messageIDs,
			Description: "Remove label from messages",
			IsBulk:      true,
			ExtraData: map[string]interface{}{
				"removed_labels": []string{labelID},
			},
		}
		s.undoService.RecordAction(ctx, action)
	}

	// Remove label from all messages using Gmail client directly (to avoid double undo recording)
	var errs []string
	for _, messageID := range messageIDs {
		if err := s.gmailClient.RemoveLabel(messageID, labelID); err != nil {
			errs = append(errs, fmt.Sprintf("failed to remove label from %s: %v", messageID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("bulk remove label errors: %s", strings.Join(errs, "; "))
	}

	return nil
}
