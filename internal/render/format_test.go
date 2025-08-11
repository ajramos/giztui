package render

import (
	"context"
	"strings"
	"testing"

	gmailwrap "github.com/ajramos/gmail-tui/internal/gmail"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestDetectPlainTextLinks(t *testing.T) {
	body := "Check this https://example.com/page?x=1#sec and this http://foo.bar"
	links, replaced := detectPlainTextLinks(body)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if !strings.Contains(replaced, "[1]") || !strings.Contains(replaced, "[2]") {
		t.Fatalf("replaced body should contain [1] and [2], got: %s", replaced)
	}
}

func TestSanitizeBodyPreservingCode(t *testing.T) {
	in := "Line … with – unicode\n```\nkeep 🚀 emoji inside code\n```\nBack • outside"
	out := sanitizeBodyPreservingCode(in)
	// Emoji should be preserved inside code block
	if !strings.Contains(out, "🚀") {
		t.Fatalf("emoji inside code should be preserved, got: %s", out)
	}
	// Bullet outside should be normalized to '- '
	if strings.Contains(out, "•") {
		t.Fatalf("bullet outside code should be normalized, got: %s", out)
	}
	// Ellipsis and en-dash normalized
	if !strings.Contains(out, "Line ... with - unicode") {
		t.Fatalf("unicode punctuation should be normalized, got: %s", out)
	}
}

func TestRenderHTMLTableGridCollapse(t *testing.T) {
	html := `<table><tbody><tr><td>a</td><td>b</td><td>c</td><td>d</td><td>e</td><td>f</td></tr></tbody></table>`
	txt, _, _, err := renderHTMLToText(html)
	if err != nil {
		t.Fatalf("renderHTMLToText error: %v", err)
	}
	// Should not contain a lot of pipes; should contain concatenated letters
	if strings.Count(txt, "|") > 2 {
		t.Fatalf("expected collapsed grid, got: %q", txt)
	}
	if !strings.Contains(txt, "abcdef") {
		t.Fatalf("expected concatenated cells, got: %q", txt)
	}
}

func TestFormatEmailForTerminal_NoTofuAndLinks(t *testing.T) {
	msg := &gmailwrap.Message{
		Message:   &gmailapi.Message{},
		PlainText: "Hello • world https://example.com",
		HTML:      "",
	}
	out, err := FormatEmailForTerminal(context.Background(), msg, FormatOptions{WrapWidth: 80}, nil)
	if err != nil {
		t.Fatalf("FormatEmailForTerminal error: %v", err)
	}
	if strings.Contains(out, "�") {
		t.Fatalf("output contains tofu glyph: %q", out)
	}
	if !strings.Contains(out, "[LINKS]") || !strings.Contains(out, "(1) https://example.com") {
		t.Fatalf("expected links section, got: %q", out)
	}
}
