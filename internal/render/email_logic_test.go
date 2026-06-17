package render

import (
	"testing"

	"github.com/derailed/tcell/v2"
	gmail "google.golang.org/api/gmail/v1"
)

func rmsg(labels []string, headers map[string]string, internalDate int64) *gmail.Message {
	m := &gmail.Message{LabelIds: labels, InternalDate: internalDate}
	if headers != nil {
		m.Payload = &gmail.MessagePart{}
		for k, v := range headers {
			m.Payload.Headers = append(m.Payload.Headers, &gmail.MessagePartHeader{Name: k, Value: v})
		}
	}
	return m
}

func TestEmailRenderer_StatePredicates(t *testing.T) {
	er := NewEmailRenderer(nil)
	if !er.IsUnread(rmsg([]string{"UNREAD", "INBOX"}, nil, 0)) {
		t.Error("UNREAD label should be unread")
	}
	if er.IsUnread(rmsg([]string{"INBOX"}, nil, 0)) {
		t.Error("no UNREAD label should be read")
	}
	if !er.IsImportant(rmsg([]string{"IMPORTANT"}, nil, 0)) {
		t.Error("IMPORTANT should be important")
	}
	if !er.IsImportant(rmsg([]string{"PRIORITY"}, nil, 0)) {
		t.Error("PRIORITY should be important")
	}
	if er.IsImportant(rmsg([]string{"INBOX"}, nil, 0)) {
		t.Error("plain INBOX should not be important")
	}
	if !er.IsDraft(rmsg([]string{"DRAFT"}, nil, 0)) {
		t.Error("DRAFT should be draft")
	}
	if !er.IsSent(rmsg([]string{"SENT"}, nil, 0)) {
		t.Error("SENT should be sent")
	}
}

func TestEmailRenderer_GetHeader(t *testing.T) {
	er := NewEmailRenderer(nil)
	m := rmsg(nil, map[string]string{"Subject": "Hi", "From": "a@x.com"}, 0)
	if got := er.GetHeader(m, "subject"); got != "Hi" { // case-insensitive
		t.Errorf("GetHeader(subject) = %q, want Hi", got)
	}
	if got := er.GetHeader(m, "Missing"); got != "" {
		t.Errorf("missing header should be empty, got %q", got)
	}
}

func TestEmailRenderer_ExtractSenderName(t *testing.T) {
	er := NewEmailRenderer(nil)
	cases := map[string]string{
		"Ángel Ramos <a@x.com>": "Ángel Ramos",
		"a@x.com":               "a@x.com",
		"":                      "",
	}
	for in, want := range cases {
		if got := er.ExtractSenderName(in); got != want {
			t.Errorf("ExtractSenderName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmailRenderer_GetDate(t *testing.T) {
	er := NewEmailRenderer(nil)
	// InternalDate (epoch millis) takes precedence.
	if got := er.GetDate(rmsg(nil, nil, 1700000000000)); got.UnixMilli() != 1700000000000 {
		t.Errorf("InternalDate not used: %v", got)
	}
	// Falls back to parsing the Date header.
	if got := er.GetDate(rmsg(nil, map[string]string{"Date": "Mon, 02 Jan 2006 15:04:05 -0700"}, 0)); got.Year() != 2006 {
		t.Errorf("Date header not parsed: %v", got)
	}
}

func TestEmailRenderer_GetMessageColor_Precedence(t *testing.T) {
	er := NewEmailRenderer(nil)
	// Distinct colors so precedence is actually observable.
	unread, read, important, sent, draft, def :=
		tcell.Color(1), tcell.Color(2), tcell.Color(3), tcell.Color(4), tcell.Color(5), tcell.Color(6)
	er.UpdateColorer(unread, read, important, sent, draft, def)

	if got := er.GetMessageColor(rmsg([]string{"IMPORTANT", "UNREAD"}, nil, 0)); got != important {
		t.Errorf("important must win over unread, got %v", got)
	}
	if got := er.GetMessageColor(rmsg([]string{"DRAFT", "UNREAD"}, nil, 0)); got != draft {
		t.Errorf("draft must win over unread, got %v", got)
	}
	if got := er.GetMessageColor(rmsg([]string{"SENT"}, nil, 0)); got != sent {
		t.Errorf("sent color, got %v", got)
	}
	if got := er.GetMessageColor(rmsg([]string{"UNREAD"}, nil, 0)); got != unread {
		t.Errorf("unread color, got %v", got)
	}
	if got := er.GetMessageColor(rmsg([]string{"INBOX"}, nil, 0)); got != read {
		t.Errorf("plain read color, got %v", got)
	}
}

func TestToTitleCase(t *testing.T) {
	cases := map[string]string{
		"hello_world":   "Hello World",
		"GCP-deep-dive": "Gcp Deep Dive",
		"foo.bar":       "Foo Bar",
		"":              "",
		"  spaced  ":    "Spaced",
	}
	for in, want := range cases {
		if got := toTitleCase(in); got != want {
			t.Errorf("toTitleCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func partWith(filename, mime, attachmentID string, children ...*gmail.MessagePart) *gmail.MessagePart {
	p := &gmail.MessagePart{Filename: filename, MimeType: mime, Parts: children}
	if attachmentID != "" {
		p.Body = &gmail.MessagePartBody{AttachmentId: attachmentID}
	}
	return p
}

func TestEmailRenderer_ExtractAttachmentIcon(t *testing.T) {
	er := NewEmailRenderer(nil)
	none := &gmail.Message{Payload: partWith("", "text/plain", "")}
	if got := er.ExtractAttachmentIcon(none); got != "  " {
		t.Errorf("no attachment → 2 spaces, got %q", got)
	}
	// Nested part with a filename.
	withFile := &gmail.Message{Payload: partWith("", "multipart/mixed", "", partWith("doc.pdf", "application/pdf", ""))}
	if got := er.ExtractAttachmentIcon(withFile); got != "📎" {
		t.Errorf("attachment by filename → 📎, got %q", got)
	}
	// Part flagged by AttachmentId.
	withID := &gmail.Message{Payload: partWith("", "application/octet-stream", "att123")}
	if got := er.ExtractAttachmentIcon(withID); got != "📎" {
		t.Errorf("attachment by id → 📎, got %q", got)
	}
	if got := er.ExtractAttachmentIcon(nil); got != "  " {
		t.Errorf("nil message → 2 spaces, got %q", got)
	}
}

func TestEmailRenderer_ExtractCalendarIcon(t *testing.T) {
	er := NewEmailRenderer(nil)
	if got := er.ExtractCalendarIcon(&gmail.Message{Payload: partWith("invite.ics", "application/octet-stream", "")}); got != "📅" {
		t.Errorf(".ics filename → 📅, got %q", got)
	}
	if got := er.ExtractCalendarIcon(&gmail.Message{Payload: partWith("", "text/calendar; method=REQUEST", "")}); got != "📅" {
		t.Errorf("text/calendar mime → 📅, got %q", got)
	}
	if got := er.ExtractCalendarIcon(&gmail.Message{Payload: partWith("note.txt", "text/plain", "")}); got != "  " {
		t.Errorf("no calendar → 2 spaces, got %q", got)
	}
}

func TestEmailRenderer_FormatLabelsForColumn(t *testing.T) {
	er := NewEmailRenderer(nil)
	if got := er.FormatLabelsForColumn(nil, 40); got != "" {
		t.Errorf("nil message → empty, got %q", got)
	}
	if got := er.FormatLabelsForColumn(rmsg([]string{"Label_1"}, nil, 0), 0); got != "" {
		t.Errorf("maxWidth<=0 → empty, got %q", got)
	}
	// System + state labels are filtered out (shown via colors), so nothing remains.
	if got := er.FormatLabelsForColumn(rmsg([]string{"INBOX", "UNREAD", "IMPORTANT"}, nil, 0), 40); got != "" {
		t.Errorf("only system/state labels → empty, got %q", got)
	}
	// A user label (mapped to a friendly name) is rendered in a wide column.
	er.SetLabelMap(map[string]string{"Label_1": "Work"})
	if got := er.FormatLabelsForColumn(rmsg([]string{"Label_1"}, nil, 0), 40); !contains(got, "Work") {
		t.Errorf("user label should appear, got %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
