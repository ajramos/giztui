package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/gmail"
	gmailapi "google.golang.org/api/gmail/v1"
)

// SlackServiceImpl implements the SlackService interface
type SlackServiceImpl struct {
	client     *gmail.Client
	config     *config.Config
	aiService  AIService
	httpClient *http.Client
}

// NewSlackService creates a new SlackService implementation
func NewSlackService(client *gmail.Client, config *config.Config, aiService AIService) *SlackServiceImpl {
	return &SlackServiceImpl{
		client:     client,
		config:     config,
		aiService:  aiService,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ForwardEmail forwards a Gmail message to Slack
func (s *SlackServiceImpl) ForwardEmail(ctx context.Context, messageID string, options SlackForwardOptions) error {
	// Get the email message from Gmail API
	gmailMessage, err := s.client.Service.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return fmt.Errorf("failed to get email message: %w", err)
	}

	// Format the message for Slack
	slackMessage, err := s.formatEmailForSlack(ctx, gmailMessage, options)
	if err != nil {
		return fmt.Errorf("failed to format email for Slack: %w", err)
	}

	// Send to Slack
	err = s.sendToSlack(ctx, slackMessage, options.WebhookURL)
	if err != nil {
		return fmt.Errorf("failed to send to Slack: %w", err)
	}

	return nil
}

// ValidateWebhook validates a Slack webhook URL by sending a test message
func (s *SlackServiceImpl) ValidateWebhook(ctx context.Context, webhookURL string) error {
	testMessage := SlackMessage{
		Text: "📧 GizTUI - Webhook validation test",
	}

	return s.sendToSlack(ctx, testMessage, webhookURL)
}

// ListConfiguredChannels returns the list of configured Slack channels
func (s *SlackServiceImpl) ListConfiguredChannels(ctx context.Context) ([]SlackChannel, error) {
	if !s.config.Slack.Enabled {
		return []SlackChannel{}, nil
	}

	// Convert config SlackChannels to service SlackChannels
	channels := make([]SlackChannel, len(s.config.Slack.Channels))
	for i, ch := range s.config.Slack.Channels {
		channels[i] = SlackChannel{
			ID:          ch.ID,
			Name:        ch.Name,
			WebhookURL:  ch.WebhookURL,
			Default:     ch.Default,
			Description: ch.Description,
		}
	}

	return channels, nil
}

// formatEmailForSlack formats an email message for Slack posting
func (s *SlackServiceImpl) formatEmailForSlack(ctx context.Context, message *gmailapi.Message, options SlackForwardOptions) (SlackMessage, error) {
	var slackMessage SlackMessage

	// Extract email metadata and body
	headers := s.extractEmailMetadata(message)
	body := s.extractEmailBody(message)

	// Build the message based on format style
	switch options.FormatStyle {
	case "summary":
		content, err := s.formatSummaryMessage(ctx, headers, body, options)
		if err != nil {
			return slackMessage, err
		}
		slackMessage.Text = content
	case "compact":
		slackMessage.Text = s.formatCompactMessage(headers, body, options)
	case "full":
		slackMessage.Text = s.formatFullMessage(headers, options)
	case "raw":
		slackMessage.Text = s.formatRawMessage(headers, body, options)
	default:
		slackMessage.Text = s.formatCompactMessage(headers, body, options)
	}

	return slackMessage, nil
}

// formatSummaryMessage creates a summary-formatted message using AI
func (s *SlackServiceImpl) formatSummaryMessage(ctx context.Context, headers map[string]string, body string, options SlackForwardOptions) (string, error) {
	var parts []string

	// Add user message if provided
	if options.UserMessage != "" {
		parts = append(parts, fmt.Sprintf("💬 %s\n\n", options.UserMessage))
	}

	// Generate AI summary if available
	if s.aiService != nil {
		// Prepare variables for the prompt (all available headers + body)
		variables := map[string]string{
			"body":        body,
			"subject":     headers["subject"],
			"from":        headers["from"],
			"to":          headers["to"],
			"cc":          headers["cc"],
			"bcc":         headers["bcc"],
			"date":        headers["date"],
			"reply-to":    headers["reply-to"],
			"message-id":  headers["message-id"],
			"in-reply-to": headers["in-reply-to"],
			"references":  headers["references"],
			"max_words":   "50",                // Keep summaries concise for Slack
			"comment":     options.UserMessage, // User's pre-message for context
		}

		// Replace variables in the prompt (like PromptService does)
		promptWithVars := s.config.Slack.GetSummaryPrompt()
		for key, value := range variables {
			placeholder := fmt.Sprintf("{{%s}}", key)
			promptWithVars = strings.ReplaceAll(promptWithVars, placeholder, value)
		}

		summary, err := s.aiService.ApplyCustomPrompt(ctx, promptWithVars, variables)
		if err != nil {
			// Fallback to first few lines if AI fails
			summary = s.truncateText(body, 200)
		}

		parts = append(parts, fmt.Sprintf("*Summary:* %s\n", summary))
	} else {
		// Fallback to truncated body
		truncated := s.truncateText(body, 200)
		parts = append(parts, fmt.Sprintf("*Preview:* %s\n", truncated))
	}

	// Metadata is now available as variables in the prompt template
	// Users can include {{subject}}, {{from}}, {{date}} in their custom prompts

	return strings.Join(parts, ""), nil
}

// formatCompactMessage creates a compact-formatted message
func (s *SlackServiceImpl) formatCompactMessage(headers map[string]string, body string, options SlackForwardOptions) string {
	var parts []string

	// Add user message if provided
	if options.UserMessage != "" {
		parts = append(parts, fmt.Sprintf("💬 %s\n", options.UserMessage))
	}

	// Compact format
	parts = append(parts, fmt.Sprintf("*From:* %s • *Subject:* %s", headers["from"], headers["subject"]))

	// Add body preview
	preview := s.truncateText(body, 200) // Default reasonable length
	if preview != "" {
		parts = append(parts, fmt.Sprintf("> %s", preview))
	}

	return strings.Join(parts, "\n")
}

// formatFullMessage creates a full-formatted message using TUI-processed content
func (s *SlackServiceImpl) formatFullMessage(headers map[string]string, options SlackForwardOptions) string {
	var parts []string

	// Add user message if provided
	if options.UserMessage != "" {
		parts = append(parts, fmt.Sprintf("💬 %s\n", options.UserMessage))
	} else {
		parts = append(parts, "📧 *Email Forward*")
	}

	// Full headers (show main ones, others available as variables)
	parts = append(parts, fmt.Sprintf("*From:* %s", headers["from"]))
	parts = append(parts, fmt.Sprintf("*Subject:* %s", headers["subject"]))
	if headers["date"] != "" {
		parts = append(parts, fmt.Sprintf("*Date:* %s", headers["date"]))
	}
	if headers["to"] != "" {
		parts = append(parts, fmt.Sprintf("*To:* %s", headers["to"]))
	}
	if headers["cc"] != "" {
		parts = append(parts, fmt.Sprintf("*CC:* %s", headers["cc"]))
	}

	// Use TUI-processed content if available, otherwise fallback to basic processing
	var content string
	if options.ProcessedContent != "" {
		content = s.truncateText(options.ProcessedContent, 2000) // Larger limit for processed content
	} else {
		content = "⚠️ Processed content not available"
	}

	parts = append(parts, "\n*Message:*")
	parts = append(parts, content)

	return strings.Join(parts, "\n")
}

// formatRawMessage creates a raw-formatted message with minimal processing
func (s *SlackServiceImpl) formatRawMessage(headers map[string]string, body string, options SlackForwardOptions) string {
	var parts []string

	// Add user message if provided
	if options.UserMessage != "" {
		parts = append(parts, fmt.Sprintf("💬 %s\n", options.UserMessage))
	} else {
		parts = append(parts, "📧 *Email Forward (Raw)*")
	}

	// Minimal headers
	parts = append(parts, fmt.Sprintf("*From:* %s", headers["from"]))
	parts = append(parts, fmt.Sprintf("*Subject:* %s", headers["subject"]))
	if headers["date"] != "" {
		parts = append(parts, fmt.Sprintf("*Date:* %s", headers["date"]))
	}

	// Raw body with minimal processing (just basic cleanup)
	rawBody := s.truncateText(body, 1500) // Reasonable limit for raw content
	parts = append(parts, "\n*Raw Message:*")
	parts = append(parts, rawBody)

	return strings.Join(parts, "\n")
}

// extractEmailMetadata extracts all headers from email
func (s *SlackServiceImpl) extractEmailMetadata(message *gmailapi.Message) map[string]string {
	headers := map[string]string{
		"subject":     "",
		"from":        "",
		"to":          "",
		"cc":          "",
		"bcc":         "",
		"date":        "",
		"reply-to":    "",
		"message-id":  "",
		"in-reply-to": "",
		"references":  "",
	}

	if message.Payload == nil || message.Payload.Headers == nil {
		// Set defaults for essential headers
		headers["subject"] = "(No Subject)"
		headers["from"] = "Unknown Sender"
		return headers
	}

	for _, header := range message.Payload.Headers {
		switch strings.ToLower(header.Name) {
		case "subject":
			headers["subject"] = header.Value
		case "from":
			headers["from"] = header.Value
		case "to":
			headers["to"] = header.Value
		case "cc":
			headers["cc"] = header.Value
		case "bcc":
			headers["bcc"] = header.Value
		case "date":
			headers["date"] = header.Value
		case "reply-to":
			headers["reply-to"] = header.Value
		case "message-id":
			headers["message-id"] = header.Value
		case "in-reply-to":
			headers["in-reply-to"] = header.Value
		case "references":
			headers["references"] = header.Value
		}
	}

	// Set defaults for essential headers if empty
	if headers["subject"] == "" {
		headers["subject"] = "(No Subject)"
	}
	if headers["from"] == "" {
		headers["from"] = "Unknown Sender"
	}

	return headers
}

// extractEmailBody extracts the email body text
func (s *SlackServiceImpl) extractEmailBody(message *gmailapi.Message) string {
	if message.Payload == nil {
		return ""
	}

	// Try to get plain text body
	body := s.extractPlainTextBody(message.Payload)
	if body == "" {
		// Fallback to HTML body converted to text (simplified)
		body = s.extractHTMLBody(message.Payload)
	}

	return strings.TrimSpace(body)
}

// extractPlainTextBody extracts plain text from email payload
func (s *SlackServiceImpl) extractPlainTextBody(payload *gmailapi.MessagePart) string {
	if payload.MimeType == "text/plain" && payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			return string(decoded)
		}
	}

	// Check parts recursively
	for _, part := range payload.Parts {
		if body := s.extractPlainTextBody(part); body != "" {
			return body
		}
	}

	return ""
}

// extractHTMLBody extracts and simplifies HTML body (basic conversion)
func (s *SlackServiceImpl) extractHTMLBody(payload *gmailapi.MessagePart) string {
	if payload.MimeType == "text/html" && payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			// Basic HTML to text conversion (remove tags)
			text := string(decoded)
			text = strings.ReplaceAll(text, "<br>", "\n")
			text = strings.ReplaceAll(text, "<br/>", "\n")
			text = strings.ReplaceAll(text, "<p>", "\n")
			text = strings.ReplaceAll(text, "</p>", "\n")
			// Remove all other HTML tags (basic)
			for strings.Contains(text, "<") && strings.Contains(text, ">") {
				start := strings.Index(text, "<")
				end := strings.Index(text[start:], ">")
				if end != -1 {
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
		if body := s.extractHTMLBody(part); body != "" {
			return body
		}
	}

	return ""
}

// truncateText truncates text to a maximum length with ellipsis
func (s *SlackServiceImpl) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	truncated := text[:maxLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLength/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// digestMaxList caps how many emails are listed in the new-mail Slack digest.
const digestMaxList = 10

// summaryCount returns how many of `total` new emails should be AI-summarized,
// applying the opt-in flag, the <=0 → 5 clamp, the digestMaxList cap, and min(total).
func summaryCount(summaries bool, limit, total int) int {
	if !summaries {
		return 0
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > digestMaxList {
		limit = digestMaxList
	}
	if limit > total {
		return total
	}
	return limit
}

// defaultSlackWebhook returns the webhook of the default channel (the one with Default==true,
// else the first configured channel). Errors when no channel is configured.
func defaultSlackWebhook(cfg *config.Config) (string, error) {
	if cfg == nil || len(cfg.Slack.Channels) == 0 {
		return "", fmt.Errorf("no Slack channel configured")
	}
	for _, ch := range cfg.Slack.Channels {
		if ch.Default && strings.TrimSpace(ch.WebhookURL) != "" {
			return ch.WebhookURL, nil
		}
	}
	if wh := strings.TrimSpace(cfg.Slack.Channels[0].WebhookURL); wh != "" {
		return wh, nil
	}
	return "", fmt.Errorf("no Slack webhook configured")
}

// SendNotification posts a plain-text message to the default Slack channel.
func (s *SlackServiceImpl) SendNotification(ctx context.Context, text string) error {
	webhook, err := defaultSlackWebhook(s.config)
	if err != nil {
		return err
	}
	return s.sendToSlack(ctx, SlackMessage{Text: text}, webhook)
}

// sendToSlack sends a message to Slack via webhook
func (s *SlackServiceImpl) sendToSlack(ctx context.Context, message SlackMessage, webhookURL string) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // Error not actionable in defer

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// summarizeForDigest returns a short AI summary of body, or "" when unavailable
// (nil AI service, empty body, or AI error) so the caller falls back to the plain row.
func (s *SlackServiceImpl) summarizeForDigest(ctx context.Context, body string) string {
	if s.aiService == nil || strings.TrimSpace(body) == "" {
		return ""
	}
	const maxWords = "50"
	prompt := s.config.Slack.GetSummaryPrompt()
	prompt = strings.ReplaceAll(prompt, "{{body}}", body)
	prompt = strings.ReplaceAll(prompt, "{{max_words}}", maxWords)
	variables := map[string]string{"body": body, "max_words": maxWords}
	out, err := s.aiService.ApplyCustomPrompt(ctx, prompt, variables)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// digestItem is one new-mail row for the Slack notification.
type digestItem struct {
	Subject string
	From    string
	Link    string // pre-resolved Gmail hyperlink URL, "" if none
	Summary string // AI summary, "" when not summarized
}

// buildNewMailDigest formats the new-mail Slack notification, capping listed rows at digestMaxList.
// A non-empty Summary renders as an indented italic line under the row.
func buildNewMailDigest(items []digestItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "📬 %d new email(s):", len(items))
	for i, it := range items {
		if i >= digestMaxList {
			fmt.Fprintf(&b, "\n…and %d more", len(items)-digestMaxList)
			break
		}
		if it.Link != "" {
			fmt.Fprintf(&b, "\n• <%s|%s> — %s", it.Link, it.Subject, it.From)
		} else {
			fmt.Fprintf(&b, "\n• %s — %s", it.Subject, it.From)
		}
		if it.Summary != "" {
			fmt.Fprintf(&b, "\n   _%s_", it.Summary)
		}
	}
	return b.String()
}

// SlackMessage represents a message to be sent to Slack
type SlackMessage struct {
	Text string `json:"text"`
}
