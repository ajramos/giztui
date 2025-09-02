package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"regexp"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
	"github.com/google/uuid"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// CompositionServiceImpl implements CompositionService
type CompositionServiceImpl struct {
	emailService EmailService
	gmailClient  *gmail.Client
	messageRepo  MessageRepository
	logger       *log.Logger
}

// NewCompositionService creates a new composition service
func NewCompositionService(emailService EmailService, gmailClient *gmail.Client, messageRepo MessageRepository) *CompositionServiceImpl {
	return &CompositionServiceImpl{
		emailService: emailService,
		gmailClient:  gmailClient,
		messageRepo:  messageRepo,
	}
}

// SetLogger sets the logger for debug output
func (s *CompositionServiceImpl) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// CreateComposition creates a new composition of the specified type
func (s *CompositionServiceImpl) CreateComposition(ctx context.Context, compositionType CompositionType, originalMessageID string) (*Composition, error) {
	composition := &Composition{
		ID:          uuid.New().String(),
		Type:        compositionType,
		To:          []Recipient{},
		CC:          []Recipient{},
		BCC:         []Recipient{},
		Subject:     "",
		Body:        "",
		Attachments: []Attachment{},
		OriginalID:  originalMessageID,
		IsDraft:     false,
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
	}

	// Process based on composition type
	switch compositionType {
	case CompositionTypeReply, CompositionTypeReplyAll:
		if originalMessageID == "" {
			return nil, fmt.Errorf("original message ID required for reply")
		}
		replyContext, err := s.ProcessReply(ctx, originalMessageID)
		if err != nil {
			return nil, fmt.Errorf("failed to process reply context: %w", err)
		}

		composition.Subject = replyContext.Subject
		composition.Body = replyContext.QuotedBody
		composition.To = replyContext.Recipients

		if compositionType == CompositionTypeReplyAll {
			// For reply-all, we need to get all recipients from the original message
			replyAllContext, err := s.ProcessReplyAll(ctx, originalMessageID)
			if err != nil {
				return nil, fmt.Errorf("failed to process reply-all context: %w", err)
			}
			// Override recipients with reply-all recipients
			composition.To = replyAllContext.Recipients
			composition.CC = replyAllContext.CC
		}

	case CompositionTypeForward:
		if originalMessageID == "" {
			return nil, fmt.Errorf("original message ID required for forward")
		}
		forwardContext, err := s.ProcessForward(ctx, originalMessageID)
		if err != nil {
			return nil, fmt.Errorf("failed to process forward context: %w", err)
		}

		composition.Subject = forwardContext.Subject
		composition.Body = forwardContext.ForwardedBody
		// Recipients remain empty for user selection

	case CompositionTypeNew:
		// Empty composition for new message

	case CompositionTypeDraft:
		return nil, fmt.Errorf("use LoadDraftComposition for draft compositions")

	default:
		return nil, fmt.Errorf("unsupported composition type: %s", compositionType)
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Created %s composition %s", compositionType, composition.ID)
	}

	return composition, nil
}

// LoadDraftComposition loads a composition from an existing draft
func (s *CompositionServiceImpl) LoadDraftComposition(ctx context.Context, draftID string) (*Composition, error) {
	if draftID == "" {
		return nil, fmt.Errorf("draft ID cannot be empty")
	}

	// Get the specific draft directly
	targetDraft, err := s.messageRepo.GetDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to load draft: %w", err)
	}

	// Parse the draft message into a composition
	composition := &Composition{
		ID:         uuid.New().String(),
		Type:       CompositionTypeDraft,
		DraftID:    draftID,
		IsDraft:    true,
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}

	// Extract recipients, subject, and body from the draft
	if targetDraft.Message != nil {
		msg := targetDraft.Message

		// Parse To recipients
		if msg.Payload != nil && msg.Payload.Headers != nil {
			for _, header := range msg.Payload.Headers {
				switch header.Name {
				case "To":
					composition.To = s.parseRecipients(header.Value)
				case "Cc":
					composition.CC = s.parseRecipients(header.Value)
				case "Bcc":
					composition.BCC = s.parseRecipients(header.Value)
				case "Subject":
					composition.Subject = s.decodeHeaderValue(header.Value)
				}
			}
		}

		// Extract body from draft message
		composition.Body = s.extractDraftBody(targetDraft.Message)
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Loaded draft composition %s from draft %s", composition.ID, draftID)
	}

	return composition, nil
}

// SaveDraft saves a composition as a draft
func (s *CompositionServiceImpl) SaveDraft(ctx context.Context, composition *Composition) (string, error) {
	if composition == nil {
		return "", fmt.Errorf("composition cannot be nil")
	}

	// Validate composition before saving
	if errors := s.ValidateComposition(composition); len(errors) > 0 {
		return "", fmt.Errorf("composition validation failed: %v", errors)
	}

	// Convert composition to email format for Gmail API
	to := s.formatRecipients(composition.To)
	cc := make([]string, len(composition.CC))
	for i, recipient := range composition.CC {
		cc[i] = recipient.Email
	}

	var draftID string
	var err error

	// Check if this composition already has a draft ID (update existing) or create new
	if composition.DraftID != "" {
		// Update existing draft
		err = s.gmailClient.UpdateDraft(composition.DraftID, to, composition.Subject, composition.Body, cc)
		if err != nil {
			return "", fmt.Errorf("failed to update existing draft via Gmail API: %w", err)
		}
		draftID = composition.DraftID

		if s.logger != nil {
			s.logger.Printf("CompositionService: Updated existing Gmail draft %s for composition %s", draftID, composition.ID)
		}
	} else {
		// Create new draft
		draftID, err = s.gmailClient.CreateDraft(to, composition.Subject, composition.Body, cc)
		if err != nil {
			return "", fmt.Errorf("failed to create new draft via Gmail API: %w", err)
		}

		if s.logger != nil {
			s.logger.Printf("CompositionService: Created new Gmail draft %s for composition %s", draftID, composition.ID)
		}
	}

	composition.DraftID = draftID
	composition.IsDraft = true
	composition.ModifiedAt = time.Now()

	return draftID, nil
}

// DeleteComposition deletes a draft composition
func (s *CompositionServiceImpl) DeleteComposition(ctx context.Context, compositionID string) error {
	if compositionID == "" {
		return fmt.Errorf("composition ID cannot be empty")
	}

	// Delete the draft from Gmail
	if err := s.gmailClient.DeleteDraft(compositionID); err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Deleted composition %s", compositionID)
	}

	return nil
}

// SendComposition sends the composition as an email
func (s *CompositionServiceImpl) SendComposition(ctx context.Context, composition *Composition) error {
	if composition == nil {
		return fmt.Errorf("composition cannot be nil")
	}

	// Validate composition before sending
	if errors := s.ValidateComposition(composition); len(errors) > 0 {
		return fmt.Errorf("composition validation failed: %v", errors)
	}

	// Convert composition to email parameters
	to := s.formatRecipients(composition.To)
	if to == "" {
		return fmt.Errorf("at least one recipient is required")
	}

	// Handle different composition types
	switch composition.Type {
	case CompositionTypeNew, CompositionTypeForward, CompositionTypeDraft:
		// Convert CC and BCC recipients to string slices
		cc := make([]string, len(composition.CC))
		for i, recipient := range composition.CC {
			cc[i] = recipient.Email
		}
		bcc := make([]string, len(composition.BCC))
		for i, recipient := range composition.BCC {
			bcc[i] = recipient.Email
		}

		// Send as new message
		err := s.emailService.SendMessage(ctx, "", to, composition.Subject, composition.Body, cc, bcc)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// If this was a draft, delete it from Gmail after successful send
		if composition.DraftID != "" {
			if deleteErr := s.gmailClient.DeleteDraft(composition.DraftID); deleteErr != nil {
				// Log the error but don't fail the send operation
				if s.logger != nil {
					s.logger.Printf("CompositionService: Failed to delete draft %s after sending: %v", composition.DraftID, deleteErr)
				}
			} else {
				if s.logger != nil {
					s.logger.Printf("CompositionService: Successfully deleted draft %s after sending", composition.DraftID)
				}
			}
		}

	case CompositionTypeReply, CompositionTypeReplyAll:
		// Send as reply
		if composition.OriginalID == "" {
			return fmt.Errorf("original message ID required for reply")
		}

		// Extract CC recipients for reply
		var ccList []string
		for _, recipient := range composition.CC {
			ccList = append(ccList, recipient.Email)
		}

		err := s.emailService.ReplyToMessage(ctx, composition.OriginalID, composition.Body, true, ccList)
		if err != nil {
			return fmt.Errorf("failed to send reply: %w", err)
		}

	default:
		return fmt.Errorf("unsupported composition type for sending: %s", composition.Type)
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Sent composition %s (type: %s)", composition.ID, composition.Type)
	}

	return nil
}

// ValidateComposition validates a composition and returns any errors
func (s *CompositionServiceImpl) ValidateComposition(composition *Composition) []ValidationError {
	var errors []ValidationError

	if composition == nil {
		errors = append(errors, ValidationError{Field: "composition", Message: "Composition cannot be nil"})
		return errors
	}

	// Validate recipients
	if len(composition.To) == 0 {
		errors = append(errors, ValidationError{Field: "to", Message: "At least one recipient is required"})
	}

	// Validate email formats
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	for i, recipient := range composition.To {
		if !emailRegex.MatchString(recipient.Email) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("to[%d]", i),
				Message: fmt.Sprintf("Invalid email format: %s", recipient.Email),
			})
		}
	}

	for i, recipient := range composition.CC {
		if !emailRegex.MatchString(recipient.Email) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("cc[%d]", i),
				Message: fmt.Sprintf("Invalid email format: %s", recipient.Email),
			})
		}
	}

	for i, recipient := range composition.BCC {
		if !emailRegex.MatchString(recipient.Email) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("bcc[%d]", i),
				Message: fmt.Sprintf("Invalid email format: %s", recipient.Email),
			})
		}
	}

	// Validate subject
	if composition.Subject == "" {
		errors = append(errors, ValidationError{Field: "subject", Message: "Subject is required"})
	}

	// Validate body
	if composition.Body == "" {
		errors = append(errors, ValidationError{Field: "body", Message: "Message body is required"})
	}

	return errors
}

// ProcessReply processes an original message to create reply context
func (s *CompositionServiceImpl) ProcessReply(ctx context.Context, originalMessageID string) (*ReplyContext, error) {
	if originalMessageID == "" {
		return nil, fmt.Errorf("original message ID cannot be empty")
	}

	// Get the original message
	message, err := s.messageRepo.GetMessage(ctx, originalMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	// Extract original sender and date
	var originalSender Recipient
	var originalDate time.Time
	var originalSubject string

	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			switch header.Name {
			case "From":
				recipients := s.parseRecipients(header.Value)
				if len(recipients) > 0 {
					originalSender = recipients[0]
				}
			case "Date":
				if parsedDate, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
					originalDate = parsedDate
				}
			case "Subject":
				originalSubject = s.decodeHeaderValue(header.Value)
			}
		}
	}

	// Create reply subject
	replySubject := originalSubject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}

	// Create quoted body
	quotedBody := s.createQuotedBody(message, originalSender, originalDate)

	replyContext := &ReplyContext{
		OriginalMessage: message,
		Recipients:      []Recipient{originalSender},
		Subject:         replySubject,
		QuotedBody:      quotedBody,
		ThreadID:        message.ThreadId,
		OriginalSender:  originalSender,
		OriginalDate:    originalDate,
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Processed reply context for message %s", originalMessageID)
	}

	return replyContext, nil
}

// ProcessReplyAll processes an original message to create reply-all context
func (s *CompositionServiceImpl) ProcessReplyAll(ctx context.Context, originalMessageID string) (*ReplyAllContext, error) {
	if originalMessageID == "" {
		return nil, fmt.Errorf("original message ID cannot be empty")
	}

	// Get the original message
	message, err := s.messageRepo.GetMessage(ctx, originalMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	// Extract recipients from headers
	var toRecipients []Recipient
	var ccRecipients []Recipient
	var originalSender Recipient
	var originalDate time.Time
	var originalSubject string

	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			switch header.Name {
			case "From":
				recipients := s.parseRecipients(header.Value)
				if len(recipients) > 0 {
					originalSender = recipients[0]
				}
			case "To":
				toRecipients = s.parseRecipients(header.Value)
			case "Cc":
				ccRecipients = s.parseRecipients(header.Value)
			case "Date":
				if parsedDate, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
					originalDate = parsedDate
				}
			case "Subject":
				originalSubject = s.decodeHeaderValue(header.Value)
			}
		}
	}

	// Get current user's email to exclude from recipients
	currentUserEmail := ""
	if s.gmailClient != nil {
		if email, err := s.gmailClient.ActiveAccountEmail(ctx); err == nil {
			currentUserEmail = email
		}
	}

	// Combine all recipients: original sender + To + CC, excluding current user
	allRecipients := []Recipient{}

	// Add original sender first
	if originalSender.Email != "" && originalSender.Email != currentUserEmail {
		allRecipients = append(allRecipients, originalSender)
	}

	// Add all To recipients
	for _, recipient := range toRecipients {
		if recipient.Email != currentUserEmail {
			allRecipients = append(allRecipients, recipient)
		}
	}

	// Keep CC recipients separate for CC field
	finalCCRecipients := []Recipient{}
	for _, recipient := range ccRecipients {
		if recipient.Email != currentUserEmail {
			finalCCRecipients = append(finalCCRecipients, recipient)
		}
	}

	// Create reply subject
	replySubject := originalSubject
	if !strings.HasPrefix(strings.ToLower(replySubject), "re:") {
		replySubject = "Re: " + replySubject
	}

	// Create quoted body
	quotedBody := s.createQuotedBody(message, originalSender, originalDate)

	replyAllContext := &ReplyAllContext{
		OriginalMessage: message,
		Recipients:      allRecipients,
		CC:              finalCCRecipients,
		Subject:         replySubject,
		QuotedBody:      quotedBody,
		ThreadID:        message.ThreadId,
		OriginalSender:  originalSender,
		OriginalDate:    originalDate,
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Processed reply-all context for message %s with %d recipients and %d CC",
			originalMessageID, len(allRecipients), len(finalCCRecipients))
	}

	return replyAllContext, nil
}

// ProcessForward processes an original message to create forward context
func (s *CompositionServiceImpl) ProcessForward(ctx context.Context, originalMessageID string) (*ForwardContext, error) {
	if originalMessageID == "" {
		return nil, fmt.Errorf("original message ID cannot be empty")
	}

	// Get the original message
	message, err := s.messageRepo.GetMessage(ctx, originalMessageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	// Extract original sender and date
	var originalSender Recipient
	var originalDate time.Time
	var originalSubject string

	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			switch header.Name {
			case "From":
				recipients := s.parseRecipients(header.Value)
				if len(recipients) > 0 {
					originalSender = recipients[0]
				}
			case "Date":
				if parsedDate, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
					originalDate = parsedDate
				}
			case "Subject":
				originalSubject = s.decodeHeaderValue(header.Value)
			}
		}
	}

	// Create forward subject
	forwardSubject := originalSubject
	if !strings.HasPrefix(strings.ToLower(forwardSubject), "fwd:") {
		forwardSubject = "Fwd: " + forwardSubject
	}

	// Create forwarded body
	forwardedBody := s.createForwardedBody(message, originalSender, originalDate)

	forwardContext := &ForwardContext{
		OriginalMessage: message,
		Subject:         forwardSubject,
		ForwardedBody:   forwardedBody,
		OriginalSender:  originalSender,
		OriginalDate:    originalDate,
	}

	if s.logger != nil {
		s.logger.Printf("CompositionService: Processed forward context for message %s", originalMessageID)
	}

	return forwardContext, nil
}

// GetTemplates returns available email templates
func (s *CompositionServiceImpl) GetTemplates(ctx context.Context, category string) ([]*EmailTemplate, error) {
	// For now, return some basic templates
	// Real implementation would load from database or file system
	templates := []*EmailTemplate{
		{
			ID:         "thanks",
			Name:       "Thank You",
			Category:   "response",
			Subject:    "Thank you",
			Body:       "Thank you for your message. I appreciate you taking the time to reach out.",
			Variables:  []string{},
			Metadata:   map[string]string{"type": "polite"},
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		},
		{
			ID:         "meeting",
			Name:       "Meeting Request",
			Category:   "business",
			Subject:    "Meeting Request - {{topic}}",
			Body:       "Hi {{name}},\n\nI'd like to schedule a meeting to discuss {{topic}}. Are you available {{date}}?\n\nBest regards",
			Variables:  []string{"name", "topic", "date"},
			Metadata:   map[string]string{"type": "meeting"},
			CreatedAt:  time.Now(),
			ModifiedAt: time.Now(),
		},
	}

	// Filter by category if specified
	if category != "" {
		var filtered []*EmailTemplate
		for _, template := range templates {
			if template.Category == category {
				filtered = append(filtered, template)
			}
		}
		return filtered, nil
	}

	return templates, nil
}

// ApplyTemplate applies a template to a composition
func (s *CompositionServiceImpl) ApplyTemplate(ctx context.Context, composition *Composition, templateID string) error {
	if composition == nil {
		return fmt.Errorf("composition cannot be nil")
	}

	templates, err := s.GetTemplates(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	var template *EmailTemplate
	for _, t := range templates {
		if t.ID == templateID {
			template = t
			break
		}
	}

	if template == nil {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// Apply template to composition
	composition.Subject = template.Subject
	composition.Body = template.Body
	composition.ModifiedAt = time.Now()

	if s.logger != nil {
		s.logger.Printf("CompositionService: Applied template %s to composition %s", templateID, composition.ID)
	}

	return nil
}

// GetRecipientSuggestions returns recipient suggestions based on query
func (s *CompositionServiceImpl) GetRecipientSuggestions(ctx context.Context, query string) ([]Recipient, error) {
	// For now, return empty suggestions
	// Real implementation would search contacts, email history, etc.
	return []Recipient{}, nil
}

// Helper methods

// parseRecipients parses a recipient header string into Recipient structs
func (s *CompositionServiceImpl) parseRecipients(headerValue string) []Recipient {
	var recipients []Recipient

	// Simple parsing - real implementation would handle complex formats
	addresses := strings.Split(headerValue, ",")
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		// Extract name and email if in format "Name <email@domain.com>"
		if strings.Contains(addr, "<") && strings.Contains(addr, ">") {
			parts := strings.Split(addr, "<")
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				email := strings.TrimSpace(strings.TrimSuffix(parts[1], ">"))
				recipients = append(recipients, Recipient{Email: email, Name: name})
			}
		} else {
			// Just email address
			recipients = append(recipients, Recipient{Email: addr})
		}
	}

	return recipients
}

// formatRecipients formats recipients into a comma-separated string
func (s *CompositionServiceImpl) formatRecipients(recipients []Recipient) string {
	var parts []string
	for _, recipient := range recipients {
		if recipient.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", recipient.Name, recipient.Email))
		} else {
			parts = append(parts, recipient.Email)
		}
	}
	return strings.Join(parts, ", ")
}

// createQuotedBody creates a quoted body for replies
func (s *CompositionServiceImpl) createQuotedBody(message *gmail.Message, sender Recipient, date time.Time) string {
	var body strings.Builder

	body.WriteString("\n\n")
	body.WriteString(fmt.Sprintf("On %s, %s wrote:\n", date.Format("Jan 2, 2006 at 3:04 PM"), sender.Email))

	// Get full message content
	content := message.PlainText
	if content == "" && message.Snippet != "" {
		// Fallback to snippet if body extraction fails
		content = message.Snippet
	}

	// Quote the content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		body.WriteString("> " + line + "\n")
	}

	return body.String()
}

// createForwardedBody creates a forwarded body
func (s *CompositionServiceImpl) createForwardedBody(message *gmail.Message, sender Recipient, date time.Time) string {
	var body strings.Builder

	body.WriteString("\n\n---------- Forwarded message ---------\n")
	body.WriteString(fmt.Sprintf("From: %s\n", sender.Email))
	body.WriteString(fmt.Sprintf("Date: %s\n", date.Format("Mon, Jan 2, 2006 at 3:04 PM")))

	// Add subject
	if message.Payload != nil && message.Payload.Headers != nil {
		for _, header := range message.Payload.Headers {
			if header.Name == "Subject" {
				body.WriteString(fmt.Sprintf("Subject: %s\n", s.decodeHeaderValue(header.Value)))
				break
			}
		}
	}

	body.WriteString("\n")

	// Get full message content
	content := message.PlainText
	if content == "" && message.Snippet != "" {
		// Fallback to snippet if body extraction fails
		content = message.Snippet
	}
	body.WriteString(content)

	return body.String()
}

// extractDraftBody extracts the full body text from a draft message, preserving line breaks
func (s *CompositionServiceImpl) extractDraftBody(message *gmail_v1.Message) string {
	if message.Payload == nil {
		return ""
	}

	// Try to get plain text body first
	body := s.extractPlainTextBodyFromPayload(message.Payload)
	if body == "" {
		// Fallback to HTML body converted to text
		body = s.extractHTMLBodyFromPayload(message.Payload)
	}

	// If still no body, fallback to snippet
	if body == "" && message.Snippet != "" {
		body = message.Snippet
	}

	return strings.TrimSpace(body)
}

// extractPlainTextBodyFromPayload extracts plain text from email payload recursively
func (s *CompositionServiceImpl) extractPlainTextBodyFromPayload(payload *gmail_v1.MessagePart) string {
	if payload.MimeType == "text/plain" && payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(decoded)
		}
	}

	// Check parts recursively
	for _, part := range payload.Parts {
		if body := s.extractPlainTextBodyFromPayload(part); body != "" {
			return body
		}
	}

	return ""
}

// extractHTMLBodyFromPayload extracts and converts HTML body to plain text
func (s *CompositionServiceImpl) extractHTMLBodyFromPayload(payload *gmail_v1.MessagePart) string {
	if payload.MimeType == "text/html" && payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			// Basic HTML to text conversion
			text := string(decoded)
			text = strings.ReplaceAll(text, "<br>", "\n")
			text = strings.ReplaceAll(text, "<br/>", "\n")
			text = strings.ReplaceAll(text, "<p>", "\n")
			text = strings.ReplaceAll(text, "</p>", "\n")

			// Remove HTML tags (basic)
			for strings.Contains(text, "<") && strings.Contains(text, ">") {
				start := strings.Index(text, "<")
				end := strings.Index(text[start:], ">")
				if end > 0 {
					text = text[:start] + text[start+end+1:]
				} else {
					break
				}
			}

			return text
		}
	}

	// Check parts recursively
	for _, part := range payload.Parts {
		if body := s.extractHTMLBodyFromPayload(part); body != "" {
			return body
		}
	}

	return ""
}

// decodeHeaderValue decodes MIME-encoded header values (e.g., =?UTF-8?Q?...?=)
func (s *CompositionServiceImpl) decodeHeaderValue(headerValue string) string {
	// Use mime.WordDecoder to decode MIME headers
	decoder := mime.WordDecoder{}
	decoded, err := decoder.DecodeHeader(headerValue)
	if err != nil {
		// If decoding fails, return the original value
		return headerValue
	}
	return decoded
}
