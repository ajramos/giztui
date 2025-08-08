package gmail

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// Client wraps the gmail.Service and provides convenience methods
type Client struct {
	Service *gmail.Service
}

// NewClient creates a new Gmail client
func NewClient(service *gmail.Service) *Client {
	return &Client{Service: service}
}

// Message represents a Gmail message with extracted content
type Message struct {
	*gmail.Message
	PlainText string
	HTML      string
	Subject   string
	From      string
	To        string
	Date      time.Time
	Labels    []string
}

// ListMessages returns first page of inbox messages (backward-compatible)
func (c *Client) ListMessages(maxResults int64) ([]*gmail.Message, error) {
	msgs, _, err := c.ListMessagesPage(maxResults, "")
	return msgs, err
}

// ListMessagesPage returns a page of inbox messages and the nextPageToken
func (c *Client) ListMessagesPage(maxResults int64, pageToken string) ([]*gmail.Message, string, error) {
	user := "me"
	// Align with Gmail Web Inbox: only INBOX and exclude sent, drafts, chat, spam, trash
	call := c.Service.Users.Messages.List(user).
		LabelIds("INBOX").
		Q("-in:sent -in:draft -in:chat -in:spam -in:trash")
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	res, err := call.Do()
	if err != nil {
		return nil, "", fmt.Errorf("no se pudieron listar los mensajes: %w", err)
	}

	return res.Messages, res.NextPageToken, nil
}

// GetMessage retrieves a specific message by ID
func (c *Client) GetMessage(id string) (*gmail.Message, error) {
	user := "me"
	msg, err := c.Service.Users.Messages.Get(user, id).Do()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el mensaje: %w", err)
	}

	return msg, nil
}

// GetMessageWithContent retrieves a message and extracts its content
func (c *Client) GetMessageWithContent(id string) (*Message, error) {
	msg, err := c.GetMessage(id)
	if err != nil {
		return nil, err
	}

	message := &Message{Message: msg}
	message.PlainText = ExtractPlainText(msg)
	message.HTML = ExtractHTML(msg)
	message.Subject = extractHeader(msg, "Subject")
	message.From = extractHeader(msg, "From")
	message.To = extractHeader(msg, "To")
	message.Date = extractDate(msg)
	// Map label IDs to human-friendly names and filter system labels to align with labels UI
	message.Labels = c.humanReadableLabels(extractLabels(msg))

	return message, nil
}

// humanReadableLabels converts label IDs to names and filters out non-actionable system labels
func (c *Client) humanReadableLabels(labelIDs []string) []string {
	if len(labelIDs) == 0 {
		return []string{}
	}

	// Build ID->Name map once per call (fast enough and simple)
	labels, err := c.ListLabels()
	if err != nil {
		// If we cannot load labels, return the raw IDs as a fallback
		return labelIDs
	}
	idToName := make(map[string]string, len(labels))
	for _, l := range labels {
		idToName[l.Id] = l.Name
	}

	var out []string
	for _, id := range labelIDs {
		// Filter out non-actionable/system labels not shown in labels UI
		if strings.HasPrefix(id, "CATEGORY_") || id == "INBOX" || id == "CHAT" || id == "SENT" || id == "TRASH" || id == "SPAM" {
			continue
		}
		// Keep UNREAD, IMPORTANT, STARRED. Exclude colored star variants
		if (strings.HasSuffix(id, "_STAR") || strings.HasSuffix(id, "_STARRED")) && id != "STARRED" {
			continue
		}

		if name, ok := idToName[id]; ok && name != "" {
			out = append(out, name)
		} else {
			out = append(out, id)
		}
	}
	return out
}

// SearchMessages searches for messages using Gmail query syntax
func (c *Client) SearchMessages(query string, maxResults int64) ([]*gmail.Message, error) {
	user := "me"
	call := c.Service.Users.Messages.List(user).Q(query)
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("no se pudieron buscar los mensajes: %w", err)
	}

	return res.Messages, nil
}

// ListDrafts returns draft messages
func (c *Client) ListDrafts(maxResults int64) ([]*gmail.Draft, error) {
	user := "me"
	call := c.Service.Users.Drafts.List(user)
	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los borradores: %w", err)
	}

	return res.Drafts, nil
}

// CreateDraft creates a new draft message
func (c *Client) CreateDraft(to, subject, body string, cc []string) (string, error) {
	msg := &mail.Message{
		Header: mail.Header{
			"From":    []string{"me"},
			"To":      []string{to},
			"Subject": []string{subject},
		},
		Body: strings.NewReader(body),
	}

	if len(cc) > 0 {
		msg.Header["Cc"] = cc
	}

	var sb strings.Builder
	for k, v := range msg.Header {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(sb.String()))

	draft := &gmail.Draft{
		Message: &gmail.Message{
			Raw: raw,
		},
	}

	user := "me"
	createdDraft, err := c.Service.Users.Drafts.Create(user, draft).Do()
	if err != nil {
		return "", fmt.Errorf("no se pudo crear el borrador: %w", err)
	}

	return createdDraft.Id, nil
}

// SendMessage sends a message
func (c *Client) SendMessage(from, to, subject, body string) (string, error) {
	msg := &mail.Message{
		Header: mail.Header{
			"From":    []string{from},
			"To":      []string{to},
			"Subject": []string{subject},
		},
		Body: strings.NewReader(body),
	}

	var sb strings.Builder
	for k, v := range msg.Header {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(sb.String()))

	message := &gmail.Message{
		Raw: raw,
	}

	user := "me"
	sentMsg, err := c.Service.Users.Messages.Send(user, message).Do()
	if err != nil {
		return "", fmt.Errorf("no se pudo enviar el mensaje: %w", err)
	}

	return sentMsg.Id, nil
}

// ReplyMessage creates a reply to an existing message
func (c *Client) ReplyMessage(originalMsgID, replyBody string, send bool, cc []string) (string, error) {
	originalMsg, err := c.GetMessage(originalMsgID)
	if err != nil {
		return "", err
	}

	// Extract original message details
	subject := extractHeader(originalMsg, "Subject")
	if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	from := extractHeader(originalMsg, "From")

	if send {
		return c.SendMessage("me", from, subject, replyBody)
	} else {
		return c.CreateDraft(from, subject, replyBody, cc)
	}
}

// MarkAsRead marks a message as read
func (c *Client) MarkAsRead(messageID string) error {
	user := "me"
	modifyRequest := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err := c.Service.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("no se pudo marcar como leído: %w", err)
	}

	return nil
}

// MarkAsUnread marks a message as unread
func (c *Client) MarkAsUnread(messageID string) error {
	user := "me"
	modifyRequest := &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"UNREAD"},
	}

	_, err := c.Service.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("no se pudo marcar como no leído: %w", err)
	}

	return nil
}

// TrashMessage moves a message to trash
func (c *Client) TrashMessage(messageID string) error {
	user := "me"
	_, err := c.Service.Users.Messages.Trash(user, messageID).Do()
	if err != nil {
		return fmt.Errorf("no se pudo mover a papelera: %w", err)
	}

	return nil
}

// ArchiveMessage archives a message
func (c *Client) ArchiveMessage(messageID string) error {
	user := "me"
	modifyRequest := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}

	_, err := c.Service.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("no se pudo archivar: %w", err)
	}

	return nil
}

// ListLabels returns all labels
func (c *Client) ListLabels() ([]*gmail.Label, error) {
	user := "me"
	res, err := c.Service.Users.Labels.List(user).Do()
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar las etiquetas: %w", err)
	}

	return res.Labels, nil
}

// CreateLabel creates a new label
func (c *Client) CreateLabel(name string) (*gmail.Label, error) {
	user := "me"
	label := &gmail.Label{
		Name: name,
	}

	createdLabel, err := c.Service.Users.Labels.Create(user, label).Do()
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear la etiqueta: %w", err)
	}

	return createdLabel, nil
}

// ApplyLabel applies a label to a message
func (c *Client) ApplyLabel(messageID, labelID string) error {
	user := "me"
	modifyRequest := &gmail.ModifyMessageRequest{
		AddLabelIds: []string{labelID},
	}

	_, err := c.Service.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("no se pudo aplicar la etiqueta: %w", err)
	}

	return nil
}

// RemoveLabel removes a label from a message
func (c *Client) RemoveLabel(messageID, labelID string) error {
	user := "me"
	modifyRequest := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{labelID},
	}

	_, err := c.Service.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("no se pudo quitar la etiqueta: %w", err)
	}

	return nil
}

// GetAttachment downloads an attachment
func (c *Client) GetAttachment(messageID, attachmentID string) ([]byte, string, error) {
	user := "me"
	att, err := c.Service.Users.Messages.Attachments.Get(user, messageID, attachmentID).Do()
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo obtener el adjunto: %w", err)
	}

	data, err := base64.URLEncoding.DecodeString(att.Data)
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo decodificar el adjunto: %w", err)
	}

	// Try to get filename
	filename := "attachment"
	msg, err := c.GetMessage(messageID)
	if err == nil && msg.Payload != nil {
		var find func(part *gmail.MessagePart)
		find = func(part *gmail.MessagePart) {
			if part.Body != nil && part.Body.AttachmentId == attachmentID && part.Filename != "" {
				filename = part.Filename
				return
			}
			for _, p := range part.Parts {
				find(p)
			}
		}
		find(msg.Payload)
	}

	return data, filename, nil
}

// Helper functions
func extractHeader(msg *gmail.Message, name string) string {
	if msg.Payload == nil || msg.Payload.Headers == nil {
		return ""
	}

	for _, header := range msg.Payload.Headers {
		if header.Name == name {
			return header.Value
		}
	}

	return ""
}

func extractDate(msg *gmail.Message) time.Time {
	dateStr := extractHeader(msg, "Date")
	if dateStr == "" {
		return time.Now()
	}

	// Try to parse the date
	if t, err := time.Parse(time.RFC1123Z, dateStr); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC1123, dateStr); err == nil {
		return t
	}

	return time.Now()
}

func extractLabels(msg *gmail.Message) []string {
	if msg.LabelIds == nil {
		return []string{}
	}
	return msg.LabelIds
}

// ExtractPlainText extracts plain text content from a Gmail message
func ExtractPlainText(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}

	return extractTextFromPart(msg.Payload)
}

func extractTextFromPart(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// If this part has text content
	if part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err != nil {
			return ""
		}

		// Check if it's quoted-printable encoded
		if part.MimeType == "text/plain" {
			// Try to decode quoted-printable
			decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(string(data))))
			if err == nil {
				return string(decoded)
			}
			return string(data)
		}
	}

	// Recursively check parts
	for _, p := range part.Parts {
		if text := extractTextFromPart(p); text != "" {
			return text
		}
	}

	return ""
}

// ExtractHTML extracts HTML content from a Gmail message
func ExtractHTML(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}
	return extractHTMLFromPart(msg.Payload)
}

func extractHTMLFromPart(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// If this part has html content
	if part.Body != nil && part.Body.Data != "" && strings.EqualFold(part.MimeType, "text/html") {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err != nil {
			return ""
		}
		// Try to decode quoted-printable just in case
		decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(string(data))))
		if err == nil {
			return string(decoded)
		}
		return string(data)
	}

	// Recursively check parts
	for _, p := range part.Parts {
		if html := extractHTMLFromPart(p); html != "" {
			return html
		}
	}

	return ""
}

// ExtractHeader extracts a header value from a message
func (c *Client) ExtractHeader(msg *gmail.Message, name string) string {
	return extractHeader(msg, name)
}

// ExtractDate extracts the date from a message
func (c *Client) ExtractDate(msg *gmail.Message) time.Time {
	return extractDate(msg)
}

// ExtractLabels extracts labels from a message
func (c *Client) ExtractLabels(msg *gmail.Message) []string {
	return extractLabels(msg)
}
