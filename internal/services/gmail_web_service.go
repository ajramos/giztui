package services

import (
	"context"
	"fmt"
	"strings"
)

// GmailWebServiceImpl implements GmailWebService
type GmailWebServiceImpl struct {
	linkService LinkService
}

// NewGmailWebService creates a new Gmail web service
func NewGmailWebService(linkService LinkService) *GmailWebServiceImpl {
	return &GmailWebServiceImpl{
		linkService: linkService,
	}
}

// OpenMessageInWeb opens a Gmail message in the web interface
func (s *GmailWebServiceImpl) OpenMessageInWeb(ctx context.Context, messageID string) error {
	if err := s.ValidateMessageID(messageID); err != nil {
		return fmt.Errorf("invalid message ID: %w", err)
	}

	url := s.GenerateGmailWebURL(messageID)

	if s.linkService == nil {
		return fmt.Errorf("link service not available")
	}

	if err := s.linkService.OpenLink(ctx, url); err != nil {
		return fmt.Errorf("failed to open Gmail URL: %w", err)
	}

	return nil
}

// ValidateMessageID validates a Gmail message ID
func (s *GmailWebServiceImpl) ValidateMessageID(messageID string) error {
	if strings.TrimSpace(messageID) == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	// Gmail message IDs are typically 16 characters long and alphanumeric
	// but can contain some special characters like underscores and hyphens
	messageID = strings.TrimSpace(messageID)
	if len(messageID) < 10 {
		return fmt.Errorf("message ID too short: %s", messageID)
	}

	// Check for obviously invalid characters
	for _, char := range messageID {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' && char != '-' {
			return fmt.Errorf("message ID contains invalid characters: %s", messageID)
		}
	}

	return nil
}

// GenerateGmailWebURL generates the Gmail web interface URL for a message
func (s *GmailWebServiceImpl) GenerateGmailWebURL(messageID string) string {
	// Gmail web interface URL pattern
	// Using u/0 assumes primary Gmail account, which is the most common case
	return fmt.Sprintf("https://mail.google.com/mail/u/0/#inbox/%s", strings.TrimSpace(messageID))
}
