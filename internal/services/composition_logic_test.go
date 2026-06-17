package services

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/gmail"
	gmail_v1 "google.golang.org/api/gmail/v1"
)

// All of these methods are pure (they operate on their arguments), so a zero-value impl with nil
// dependencies is enough.

func TestCompositionService_ParseRecipients(t *testing.T) {
	s := &CompositionServiceImpl{}
	got := s.parseRecipients("Ángel <a@x.com>, b@y.com ,  ")
	if len(got) != 2 {
		t.Fatalf("expected 2 recipients, got %d: %+v", len(got), got)
	}
	if got[0].Name != "Ángel" || got[0].Email != "a@x.com" {
		t.Errorf("named recipient parsed wrong: %+v", got[0])
	}
	if got[1].Name != "" || got[1].Email != "b@y.com" {
		t.Errorf("bare recipient parsed wrong: %+v", got[1])
	}
}

func TestCompositionService_FormatRecipients(t *testing.T) {
	s := &CompositionServiceImpl{}
	out := s.formatRecipients([]Recipient{{Email: "a@x.com", Name: "Al"}, {Email: "b@y.com"}})
	if out != "Al <a@x.com>, b@y.com" {
		t.Errorf("formatRecipients = %q", out)
	}
}

func TestCompositionService_ValidateComposition(t *testing.T) {
	s := &CompositionServiceImpl{}
	if errs := s.ValidateComposition(nil); len(errs) != 1 || errs[0].Field != "composition" {
		t.Fatalf("nil composition should give one error, got %+v", errs)
	}
	fields := map[string]bool{}
	for _, e := range s.ValidateComposition(&Composition{}) {
		fields[e.Field] = true
	}
	if !fields["to"] || !fields["subject"] || !fields["body"] {
		t.Errorf("empty composition should flag to/subject/body, got %v", fields)
	}
	if bad := s.ValidateComposition(&Composition{To: []Recipient{{Email: "not-an-email"}}, Subject: "s", Body: "b"}); len(bad) == 0 {
		t.Error("invalid email format should produce an error")
	}
	if ok := s.ValidateComposition(&Composition{To: []Recipient{{Email: "a@x.com"}}, Subject: "Hi", Body: "Hello"}); len(ok) != 0 {
		t.Errorf("valid composition should have no errors, got %+v", ok)
	}
}

func TestCompositionService_CreateQuotedBody(t *testing.T) {
	s := &CompositionServiceImpl{}
	date := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	out := s.createQuotedBody(&gmail.Message{PlainText: "hello\nworld"}, Recipient{Email: "a@x.com"}, date)
	if !strings.Contains(out, "a@x.com wrote:") || !strings.Contains(out, "> hello") || !strings.Contains(out, "> world") {
		t.Errorf("quoted body wrong:\n%s", out)
	}
	// Falls back to the snippet when there is no plain-text body.
	snip := &gmail.Message{Message: &gmail_v1.Message{Snippet: "snip"}}
	if out2 := s.createQuotedBody(snip, Recipient{Email: "a@x.com"}, date); !strings.Contains(out2, "> snip") {
		t.Errorf("snippet fallback wrong:\n%s", out2)
	}
}

func TestCompositionService_CreateForwardedBody(t *testing.T) {
	s := &CompositionServiceImpl{}
	m := &gmail.Message{
		Message:   &gmail_v1.Message{Payload: &gmail_v1.MessagePart{Headers: []*gmail_v1.MessagePartHeader{{Name: "Subject", Value: "Hi there"}}}},
		PlainText: "forwarded content",
	}
	out := s.createForwardedBody(m, Recipient{Email: "a@x.com"}, time.Now())
	if !strings.Contains(out, "Forwarded message") || !strings.Contains(out, "Subject: Hi there") || !strings.Contains(out, "forwarded content") {
		t.Errorf("forwarded body wrong:\n%s", out)
	}
}

func TestCompositionService_ExtractBodyFromPayload(t *testing.T) {
	s := &CompositionServiceImpl{}
	enc := func(text string) string { return base64.URLEncoding.EncodeToString([]byte(text)) }

	plain := &gmail_v1.MessagePart{MimeType: "text/plain", Body: &gmail_v1.MessagePartBody{Data: enc("plain body")}}
	if got := s.extractPlainTextBodyFromPayload(plain); got != "plain body" {
		t.Errorf("plain extraction = %q", got)
	}
	// Nested HTML part → tags stripped, <br>/<p> become newlines.
	html := &gmail_v1.MessagePart{MimeType: "multipart/alternative", Parts: []*gmail_v1.MessagePart{
		{MimeType: "text/html", Body: &gmail_v1.MessagePartBody{Data: enc("<p>Hi</p><br>there")}},
	}}
	if got := s.extractHTMLBodyFromPayload(html); !strings.Contains(got, "Hi") || !strings.Contains(got, "there") || strings.Contains(got, "<p>") {
		t.Errorf("html extraction = %q", got)
	}
	// extractDraftBody prefers plain text and trims.
	draft := &gmail_v1.Message{Payload: plain}
	if got := s.extractDraftBody(draft); got != "plain body" {
		t.Errorf("extractDraftBody = %q", got)
	}
}

func TestCompositionService_DecodeHeaderValue(t *testing.T) {
	s := &CompositionServiceImpl{}
	if got := s.decodeHeaderValue("=?UTF-8?Q?=C3=81ngel?="); got != "Ángel" {
		t.Errorf("MIME decode = %q, want Ángel", got)
	}
	if got := s.decodeHeaderValue("Plain Subject"); got != "Plain Subject" {
		t.Errorf("plain header should pass through, got %q", got)
	}
}
