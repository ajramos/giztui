package desktop

import "time"

// MessageSummary is a lightweight message representation for list views.
// It is JSON-serializable and safe to expose to the frontend.
type MessageSummary struct {
	ID       string    `json:"id"`
	ThreadID string    `json:"threadId"`
	Subject  string    `json:"subject"`
	From     string    `json:"from"`
	Snippet  string    `json:"snippet"`
	Date     time.Time `json:"date"`
	Unread   bool      `json:"unread"`
	Labels   []string  `json:"labels"`
}

// MessageDetail is the full message representation for the reading pane.
type MessageDetail struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"threadId"`
	Subject   string    `json:"subject"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Cc        string    `json:"cc"`
	Date      time.Time `json:"date"`
	Unread    bool      `json:"unread"`
	Labels    []string  `json:"labels"`
	PlainText string    `json:"plainText"`
	HTML      string    `json:"html"`
}

// MessageList is a page of message summaries plus a pagination cursor.
type MessageList struct {
	Messages      []MessageSummary `json:"messages"`
	NextPageToken string           `json:"nextPageToken"`
}

// Label is a JSON-serializable Gmail label.
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DraftSummary is a lightweight draft for the drafts list.
type DraftSummary struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Snippet string `json:"snippet"`
}

// DraftDetail is a draft loaded for editing.
type DraftDetail struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Cc      string `json:"cc"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// AccountInfo is a JSON-serializable account for the account switcher.
type AccountInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Active      bool   `json:"active"`
}

// Link is a JSON-serializable link extracted from a message body.
type Link struct {
	Index int    `json:"index"`
	URL   string `json:"url"`
	Text  string `json:"text"`
	Type  string `json:"type"`
}

// Prompt is a JSON-serializable AI prompt template.
type Prompt struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// Attachment is a JSON-serializable message attachment. It mirrors the fields of
// services.AttachmentInfo but lives in this public package so the nested Wails
// module (which cannot import internal/...) can consume it.
type Attachment struct {
	AttachmentID string `json:"attachmentId"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	Type         string `json:"type"`
	Inline       bool   `json:"inline"`
}
