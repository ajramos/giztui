package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	gmailapi "google.golang.org/api/gmail/v1"
)

// slackStubAI is a configurable AIService stub for slack_service_test.go.
// We cannot import internal/services/mocks (import cycle: mocks → services),
// and stubAIService already exists in prompt_service_test.go without configurable
// return values, so we define a dedicated variant here.
type slackStubAI struct {
	result string
	err    error
}

func (s *slackStubAI) ApplyCustomPrompt(_ context.Context, _ string, _ map[string]string) (string, error) {
	return s.result, s.err
}
func (s *slackStubAI) ApplyCustomPromptStream(_ context.Context, _ string, _ map[string]string, _ func(string)) (string, error) {
	return "", nil
}
func (s *slackStubAI) FormatContent(_ context.Context, _ string, _ FormatOptions) (string, error) {
	return "", nil
}
func (s *slackStubAI) GenerateReply(_ context.Context, _ string, _ ReplyOptions) (string, error) {
	return "", nil
}
func (s *slackStubAI) GenerateSummary(_ context.Context, _ string, _ SummaryOptions) (*SummaryResult, error) {
	return nil, nil
}
func (s *slackStubAI) GenerateSummaryStream(_ context.Context, _ string, _ SummaryOptions, _ func(string)) (*SummaryResult, error) {
	return nil, nil
}
func (s *slackStubAI) SuggestLabels(_ context.Context, _ string, _ []string) ([]string, error) {
	return nil, nil
}

func TestSummarizeForDigest(t *testing.T) {
	cfg := &config.Config{}
	cfg.Slack = config.DefaultSlackConfig() // provides GetSummaryPrompt with {{body}}/{{max_words}}

	// Success path: returns trimmed AI output.
	ai := &slackStubAI{result: "  Recap here.  ", err: nil}
	s := &SlackServiceImpl{config: cfg, aiService: ai}
	if got := s.summarizeForDigest(context.Background(), "long body text"); got != "Recap here." {
		t.Errorf("summary = %q, want %q", got, "Recap here.")
	}

	// AI error → "" (caller keeps the plain line).
	aiErr := &slackStubAI{result: "", err: errors.New("boom")}
	sErr := &SlackServiceImpl{config: cfg, aiService: aiErr}
	if got := sErr.summarizeForDigest(context.Background(), "body"); got != "" {
		t.Errorf("on AI error want \"\", got %q", got)
	}

	// nil aiService or empty body → "".
	sNil := &SlackServiceImpl{config: cfg, aiService: nil}
	if got := sNil.summarizeForDigest(context.Background(), "body"); got != "" {
		t.Errorf("nil aiService want \"\", got %q", got)
	}
	if got := s.summarizeForDigest(context.Background(), "   "); got != "" {
		t.Errorf("empty body want \"\", got %q", got)
	}
}

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

type slackStubGmail struct {
	meta, full       []*gmailapi.Message
	metaErr, fullErr error
}

func (s *slackStubGmail) GetMessage(id string) (*gmailapi.Message, error) { return nil, nil }
func (s *slackStubGmail) GetMessagesMetadataParallel(_ []string, _ int) ([]*gmailapi.Message, error) {
	return s.meta, s.metaErr
}
func (s *slackStubGmail) GetMessagesParallel(_ []string, _ int) ([]*gmailapi.Message, error) {
	return s.full, s.fullErr
}

func TestSendNewMailDigest_Orchestration(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m SlackMessage
		_ = json.Unmarshal(b, &m)
		captured = m.Text
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	cfg.Slack = config.DefaultSlackConfig()
	cfg.Slack.Enabled = true
	cfg.Slack.Channels = []config.SlackChannel{{Name: "gen", WebhookURL: srv.URL, Default: true}}

	meta := func(id, subj, from string) *gmailapi.Message {
		return &gmailapi.Message{Id: id, Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
			{Name: "Subject", Value: subj}, {Name: "From", Value: from},
		}}}
	}
	body := func(id, text string) *gmailapi.Message {
		return &gmailapi.Message{Id: id, Payload: &gmailapi.MessagePart{
			MimeType: "text/plain",
			Body:     &gmailapi.MessagePartBody{Data: base64.URLEncoding.EncodeToString([]byte(text))},
		}}
	}

	ids := []string{"m1", "m2"}
	gc := &slackStubGmail{
		meta: []*gmailapi.Message{meta("m1", "Hello", "a@x.com"), meta("m2", "World", "b@x.com")},
		full: []*gmailapi.Message{body("m1", "some content")},
	}

	// Case A: summaries on, AI returns a summary → summary line present.
	captured = ""
	sA := &SlackServiceImpl{client: gc, config: cfg, aiService: &slackStubAI{result: "AI recap"}, httpClient: srv.Client()}
	if err := sA.SendNewMailDigest(context.Background(), ids, NewMailDigestOptions{Summaries: true, SummaryLimit: 1}); err != nil {
		t.Fatalf("case A unexpected error: %v", err)
	}
	if !strings.Contains(captured, "Hello") || !strings.Contains(captured, "_AI recap_") {
		t.Errorf("case A expected summary line, got: %q", captured)
	}

	// Case B: summaries off → no AI summary line, plain rows only.
	captured = ""
	sB := &SlackServiceImpl{client: gc, config: cfg, aiService: &slackStubAI{result: "AI recap"}, httpClient: srv.Client()}
	if err := sB.SendNewMailDigest(context.Background(), ids, NewMailDigestOptions{Summaries: false}); err != nil {
		t.Fatalf("case B unexpected error: %v", err)
	}
	if strings.Contains(captured, "_AI recap_") {
		t.Errorf("case B summaries disabled but got summary: %q", captured)
	}
	if !strings.Contains(captured, "Hello") || !strings.Contains(captured, "World") {
		t.Errorf("case B expected plain rows: %q", captured)
	}

	// Case C: AI error → graceful, no summary line.
	captured = ""
	sC := &SlackServiceImpl{client: gc, config: cfg, aiService: &slackStubAI{err: errors.New("boom")}, httpClient: srv.Client()}
	if err := sC.SendNewMailDigest(context.Background(), ids, NewMailDigestOptions{Summaries: true, SummaryLimit: 1}); err != nil {
		t.Fatalf("case C unexpected error: %v", err)
	}
	if strings.Contains(captured, "_") {
		t.Errorf("case C AI failed, should have no summary line: %q", captured)
	}

	// Case D: nil client guard.
	sD := &SlackServiceImpl{client: nil, config: cfg, aiService: &slackStubAI{}, httpClient: srv.Client()}
	if err := sD.SendNewMailDigest(context.Background(), ids, NewMailDigestOptions{}); err == nil {
		t.Error("case D expected error with nil client")
	}

	// Empty IDs → nil.
	if err := sA.SendNewMailDigest(context.Background(), nil, NewMailDigestOptions{}); err != nil {
		t.Errorf("empty IDs should return nil, got %v", err)
	}
}
