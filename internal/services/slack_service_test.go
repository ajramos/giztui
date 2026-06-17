package services

import (
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
)

func TestDefaultSlackWebhook(t *testing.T) {
	cfg := &config.Config{Slack: config.SlackConfig{Channels: []config.SlackChannel{
		{Name: "first", WebhookURL: "https://hooks/first"},
		{Name: "main", WebhookURL: "https://hooks/main", Default: true},
	}}}
	if got, err := defaultSlackWebhook(cfg); err != nil || got != "https://hooks/main" {
		t.Fatalf("default channel webhook = %q, err=%v", got, err)
	}

	cfg2 := &config.Config{Slack: config.SlackConfig{Channels: []config.SlackChannel{
		{Name: "only", WebhookURL: "https://hooks/only"},
	}}}
	if got, err := defaultSlackWebhook(cfg2); err != nil || got != "https://hooks/only" {
		t.Fatalf("no default → first; got %q err=%v", got, err)
	}

	cfg3 := &config.Config{Slack: config.SlackConfig{}}
	if _, err := defaultSlackWebhook(cfg3); err == nil {
		t.Fatal("no channels should error")
	}
}

func TestSummaryCount(t *testing.T) {
	cases := []struct {
		name               string
		summaries          bool
		limit, total, want int
	}{
		{"disabled", false, 5, 10, 0},
		{"normal", true, 5, 10, 5},
		{"limit zero clamps to 5", true, 0, 10, 5},
		{"negative clamps to 5", true, -3, 10, 5},
		{"capped at 10", true, 20, 50, 10},
		{"fewer emails than limit", true, 5, 2, 2},
	}
	for _, c := range cases {
		if got := summaryCount(c.summaries, c.limit, c.total); got != c.want {
			t.Errorf("%s: summaryCount(%v,%d,%d) = %d, want %d", c.name, c.summaries, c.limit, c.total, got, c.want)
		}
	}
}

func TestBuildNewMailDigest(t *testing.T) {
	// Two items, no links, no summaries.
	got := buildNewMailDigest([]digestItem{
		{Subject: "Hello", From: "a@x.com"},
		{Subject: "World", From: "b@x.com"},
	})
	if !strings.Contains(got, "📬 2 new email(s):") {
		t.Errorf("missing header: %q", got)
	}
	if !strings.Contains(got, "• Hello — a@x.com") || !strings.Contains(got, "• World — b@x.com") {
		t.Errorf("missing plain lines: %q", got)
	}

	// Link + summary rendering.
	linked := buildNewMailDigest([]digestItem{
		{Subject: "Hi", From: "a@x.com", Link: "https://mail.google.com/x", Summary: "Short recap."},
	})
	if !strings.Contains(linked, "• <https://mail.google.com/x|Hi> — a@x.com") {
		t.Errorf("missing hyperlink line: %q", linked)
	}
	if !strings.Contains(linked, "\n   _Short recap._") {
		t.Errorf("missing italic summary line: %q", linked)
	}

	// Summary line absent when Summary == "".
	noSum := buildNewMailDigest([]digestItem{{Subject: "Hi", From: "a@x.com"}})
	if strings.Contains(noSum, "_") {
		t.Errorf("should not render summary line when empty: %q", noSum)
	}

	// Cap at 10 with "…and N more".
	many := make([]digestItem, 12)
	for i := range many {
		many[i] = digestItem{Subject: "S", From: "f@x.com"}
	}
	capped := buildNewMailDigest(many)
	if !strings.Contains(capped, "…and 2 more") {
		t.Errorf("missing overflow line: %q", capped)
	}

	// Empty input.
	if got := buildNewMailDigest(nil); !strings.Contains(got, "📬 0 new email(s):") {
		t.Errorf("empty digest wrong: %q", got)
	}
}
