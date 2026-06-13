package services

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExtractJSONObject(t *testing.T) {
	// Plain JSON passes through.
	assert.Equal(t, `{"a":1}`, extractJSONObject(`{"a":1}`))
	// Markdown fences are stripped.
	assert.Equal(t, `{"a":1}`, extractJSONObject("```json\n{\"a\":1}\n```"))
	// Leading/trailing prose is removed.
	assert.Equal(t, `{"a":1}`, extractJSONObject("Here you go:\n{\"a\":1}\nDone."))
	// Nested braces are balanced correctly.
	assert.Equal(t, `{"a":{"b":2}}`, extractJSONObject(`prefix {"a":{"b":2}} suffix`))
	// No object → empty string.
	assert.Equal(t, "", extractJSONObject("no json here"))
}

func TestParseAnalyzerResponse(t *testing.T) {
	// batchIDs maps per-batch local number (1-based) → concrete message ID.
	batchIDs := []string{"m1", "m2", "m3", "m4"}
	raw := `{
	  "categories": [
	    {"name":"Newsletters","priority":"low","description":"marketing","action":"archive","label":"","messages":[1,3]},
	    {"name":"Follow up","priority":"high","description":"needs reply","action":"label","label":"needs-reply","messages":[4]}
	  ],
	  "read_manually": [2]
	}`

	cats, readManually, err := parseAnalyzerResponse(raw, batchIDs)
	assert.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, []string{"m1", "m3"}, cats[0].MessageIDs)
	assert.Equal(t, "archive", cats[0].Action)
	assert.Equal(t, "needs-reply", cats[1].Label)
	assert.Equal(t, []string{"m4"}, cats[1].MessageIDs)
	assert.Equal(t, []int{2}, readManually) // local indices, resolved by caller
}

func TestParseAnalyzerResponse_OutOfRangeIndexIgnored(t *testing.T) {
	batchIDs := []string{"m1", "m2"}
	raw := `{"categories":[{"name":"X","priority":"low","description":"d","action":"archive","label":"","messages":[1,9]}],"read_manually":[]}`
	cats, _, err := parseAnalyzerResponse(raw, batchIDs)
	assert.NoError(t, err)
	// index 9 is out of range → dropped, index 1 → m1.
	assert.Equal(t, []string{"m1"}, cats[0].MessageIDs)
}

func TestParseAnalyzerResponse_Malformed(t *testing.T) {
	_, _, err := parseAnalyzerResponse("not json", []string{"m1"})
	assert.Error(t, err)
}

func TestSplitBatches(t *testing.T) {
	mk := func(n int) []AnalyzerMessage {
		out := make([]AnalyzerMessage, n)
		for i := range out {
			out[i] = AnalyzerMessage{ID: fmt.Sprintf("m%d", i)}
		}
		return out
	}

	// 120 messages, size 50, cap 10 → batches of 50,50,20.
	batches := splitBatches(mk(120), 50, 10)
	assert.Len(t, batches, 3)
	assert.Len(t, batches[0], 50)
	assert.Len(t, batches[2], 20)

	// MaxBatches caps total work: 500 msgs, size 50, cap 2 → only 2 batches (100 msgs).
	capped := splitBatches(mk(500), 50, 2)
	assert.Len(t, capped, 2)
	assert.Len(t, capped[1], 50)

	// Zero/negative size falls back to a sane default (50).
	assert.Len(t, splitBatches(mk(10), 0, 10), 1)
}

func TestMergeCategories(t *testing.T) {
	existing := []ActionPlanCategory{
		{Name: "Newsletters", Action: "archive", MessageIDs: []string{"m1", "m2"}},
	}
	incoming := []ActionPlanCategory{
		// Same name (case-insensitive) → union IDs, dedup.
		{Name: "newsletters", Action: "archive", MessageIDs: []string{"m2", "m3"}},
		// New name → appended.
		{Name: "Follow up", Action: "label", Label: "needs-reply", MessageIDs: []string{"m4"}},
	}

	merged := mergeCategories(existing, incoming)
	assert.Len(t, merged, 2)
	assert.Equal(t, []string{"m1", "m2", "m3"}, merged[0].MessageIDs)
	assert.Equal(t, "Follow up", merged[1].Name)
	assert.Equal(t, []string{"m4"}, merged[1].MessageIDs)

	// Fix 1: merging must not mutate the caller's original inner slice.
	assert.Equal(t, []string{"m1", "m2"}, existing[0].MessageIDs)
}

func analyzerMsgs(n int) []AnalyzerMessage {
	out := make([]AnalyzerMessage, n)
	for i := range out {
		out[i] = AnalyzerMessage{
			ID:      fmt.Sprintf("m%d", i+1),
			Subject: fmt.Sprintf("Subject %d", i+1),
			From:    "sender@example.com",
			Snippet: "snippet",
		}
	}
	return out
}

func TestAnalyze_HappyPath_SingleBatch(t *testing.T) {
	ai := &mockAIService{}
	resp := `{"categories":[
	  {"name":"Newsletters","priority":"low","description":"d","action":"archive","label":"","messages":[1,2]},
	  {"name":"Follow up","priority":"high","description":"d","action":"label","label":"needs-reply","messages":[3]}
	],"read_manually":[]}`
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(resp, nil).Once()

	svc := NewInboxAnalyzerService(ai)
	var progressCalls int
	plan, err := svc.Analyze(context.Background(), analyzerMsgs(3),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10},
		func(p *ActionPlan) { progressCalls++ })

	assert.NoError(t, err)
	assert.Equal(t, 3, plan.TotalAnalyzed)
	assert.Equal(t, 1, plan.BatchesTotal)
	assert.Len(t, plan.Categories, 2)
	assert.Equal(t, []string{"m1", "m2"}, plan.Categories[0].MessageIDs)
	assert.Equal(t, []string{"m3"}, plan.Categories[1].MessageIDs)
	assert.Equal(t, 1, progressCalls)
	ai.AssertExpectations(t)
}

func TestAnalyze_MergesAcrossBatches(t *testing.T) {
	ai := &mockAIService{}
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(`{"categories":[{"name":"Newsletters","priority":"low","description":"d","action":"archive","label":"","messages":[1,2]}],"read_manually":[]}`, nil).Once()
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(`{"categories":[{"name":"Newsletters","priority":"low","description":"d","action":"archive","label":"","messages":[1]}],"read_manually":[2]}`, nil).Once()

	svc := NewInboxAnalyzerService(ai)
	plan, err := svc.Analyze(context.Background(), analyzerMsgs(4),
		InboxAnalyzerOptions{BatchSize: 2, MaxBatches: 10}, nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, plan.BatchesTotal)
	assert.Len(t, plan.Categories, 1)
	assert.Equal(t, []string{"m1", "m2", "m3"}, plan.Categories[0].MessageIDs)
	assert.Len(t, plan.ReadManually, 1)
	assert.Equal(t, "m4", plan.ReadManually[0].ID)
}

func TestAnalyze_RepairRetryThenDegrade(t *testing.T) {
	ai := &mockAIService{}
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("garbage", nil).Twice()

	svc := NewInboxAnalyzerService(ai)
	plan, err := svc.Analyze(context.Background(), analyzerMsgs(2),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10}, nil)

	assert.NoError(t, err)
	assert.True(t, plan.Degraded)
	assert.Len(t, plan.ReadManually, 2)
	ai.AssertExpectations(t)
}

func TestAnalyze_CustomPromptOverride(t *testing.T) {
	ai := &mockAIService{}
	var capturedPrompt string
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.MatchedBy(func(p string) bool {
		capturedPrompt = p
		return true
	}), mock.Anything, mock.Anything).
		Return(`{"categories":[{"name":"X","priority":"low","description":"d","action":"none","label":"","messages":[1]}],"read_manually":[]}`, nil).Once()

	svc := NewInboxAnalyzerService(ai)
	_, err := svc.Analyze(context.Background(), analyzerMsgs(1),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10, CustomPromptText: "CUSTOM {{messages}}"}, nil)

	assert.NoError(t, err)
	assert.Contains(t, capturedPrompt, "CUSTOM ")
	assert.Contains(t, capturedPrompt, "Subject 1")
}

func TestAnalyze_FirstBatchHardError(t *testing.T) {
	ai := &mockAIService{}
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", fmt.Errorf("llm down")).Once()

	svc := NewInboxAnalyzerService(ai)
	_, err := svc.Analyze(context.Background(), analyzerMsgs(2),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10}, nil)
	assert.Error(t, err)
}

func TestAnalyze_Cancellation(t *testing.T) {
	ai := &mockAIService{}
	svc := NewInboxAnalyzerService(ai)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Analyze(ctx, analyzerMsgs(2),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10}, nil)
	assert.ErrorIs(t, err, context.Canceled)
	ai.AssertNotCalled(t, "ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestParseAnalyzerResponse_CrossCategoryDuplicateFirstWins(t *testing.T) {
	batchIDs := []string{"m1", "m2", "m3"}
	// m2 appears in both categories; only the first category should keep it.
	raw := `{"categories":[
	  {"name":"A","priority":"low","description":"d","action":"archive","label":"","messages":[1,2]},
	  {"name":"B","priority":"low","description":"d","action":"trash","label":"","messages":[2,3]}
	],"read_manually":[]}`
	cats, _, err := parseAnalyzerResponse(raw, batchIDs)
	assert.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, []string{"m1", "m2"}, cats[0].MessageIDs)
	assert.Equal(t, []string{"m3"}, cats[1].MessageIDs) // m2 dropped from B
}

func TestExtractJSONObject_BraceInsideString(t *testing.T) {
	in := `{"subject":"Re: {project} deadline","n":1}`
	assert.Equal(t, in, extractJSONObject(in))
}

func TestAnalyze_IntermediateBatchErrorReturnsPartialPlan(t *testing.T) {
	ai := &mockAIService{}
	// Batch 1 (m1,m2) succeeds; batch 2 (m3,m4) hard-fails.
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(`{"categories":[{"name":"A","priority":"low","description":"d","action":"archive","label":"","messages":[1,2]}],"read_manually":[]}`, nil).Once()
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", fmt.Errorf("llm down")).Once()

	svc := NewInboxAnalyzerService(ai)
	plan, err := svc.Analyze(context.Background(), analyzerMsgs(4),
		InboxAnalyzerOptions{BatchSize: 2, MaxBatches: 10}, nil)

	assert.Error(t, err)   // error propagates
	assert.NotNil(t, plan) // partial plan preserved
	assert.Len(t, plan.Categories, 1)
	assert.Equal(t, []string{"m1", "m2"}, plan.Categories[0].MessageIDs)
}

func TestPrependUserRules(t *testing.T) {
	base := "Categorize these messages."
	got := prependUserRules(base, []string{
		"Never trash emails from tldr.tech",
		"Archive newsletters automatically",
	})
	if !strings.Contains(got, "## User preferences") {
		t.Fatalf("missing header in: %q", got)
	}
	if !strings.Contains(got, "- Never trash emails from tldr.tech") {
		t.Fatalf("missing rule 1 in: %q", got)
	}
	if !strings.Contains(got, "- Archive newsletters automatically") {
		t.Fatalf("missing rule 2 in: %q", got)
	}
	if !strings.HasSuffix(got, base) {
		t.Fatalf("base prompt must follow the rules block, got: %q", got)
	}

	// Empty rules → unchanged.
	if prependUserRules(base, nil) != base {
		t.Fatal("nil rules must return the base prompt unchanged")
	}
}

func TestTruncateForAnalyzer(t *testing.T) {
	// Collapses runs of whitespace/newlines to single spaces.
	if got := truncateForAnalyzer("a\n\n  b\tc", 100); got != "a b c" {
		t.Fatalf("whitespace collapse: got %q", got)
	}
	// Cuts to limit on a rune boundary (no panic, no partial multi-byte rune).
	if got := truncateForAnalyzer("áéíóú", 3); got != "áéí" {
		t.Fatalf("rune-boundary cut: got %q", got)
	}
	// limit <= 0 returns collapsed-but-untrimmed text.
	if got := truncateForAnalyzer("a  b", 0); got != "a b" {
		t.Fatalf("limit<=0: got %q", got)
	}
	// Empty/whitespace-only input.
	if got := truncateForAnalyzer("   ", 10); got != "" {
		t.Fatalf("empty/whitespace-only: got %q", got)
	}
}

func TestBuildBatchPayload_BodyVsSnippet(t *testing.T) {
	batch := []AnalyzerMessage{
		{Subject: "Hello", From: "a@x.com", Snippet: "snip-a", Body: "this is the full body of email A"},
		{Subject: "World", From: "b@x.com", Snippet: "snip-b"}, // no body → snippet
	}
	out := buildBatchPayload(batch, 1000)

	if !strings.Contains(out, "full body of email A") {
		t.Fatalf("expected body for msg 1, got:\n%s", out)
	}
	if strings.Contains(out, "snip-a") {
		t.Fatalf("msg 1 should not fall back to snippet, got:\n%s", out)
	}
	if !strings.Contains(out, "snip-b") {
		t.Fatalf("expected snippet for msg 2, got:\n%s", out)
	}

	long := []AnalyzerMessage{{Subject: "L", From: "c@mail.com", Body: strings.Repeat("x", 50)}}
	if out := buildBatchPayload(long, 10); strings.Count(out, "x") != 10 {
		t.Fatalf("body should be truncated to 10 x's, got %d", strings.Count(out, "x"))
	}
}

func TestBuildPromptPreview(t *testing.T) {
	s := NewInboxAnalyzerService(nil)

	// Default base + rules.
	out := s.BuildPromptPreview(InboxAnalyzerOptions{UserRules: []string{"keep boss emails"}})
	if !strings.Contains(out, "## User preferences") || !strings.Contains(out, "keep boss emails") {
		t.Fatalf("rules block missing: %q", out)
	}
	if !strings.Contains(out, "{{messages}}") {
		t.Fatalf("should keep {{messages}} placeholder: %q", out)
	}
	if !strings.Contains(out, "email triage assistant") { // default prompt marker
		t.Fatalf("should include default base: %q", out)
	}

	// Custom base, no rules.
	out = s.BuildPromptPreview(InboxAnalyzerOptions{CustomPromptText: "MY CUSTOM {{messages}}"})
	if !strings.Contains(out, "MY CUSTOM") || strings.Contains(out, "email triage assistant") {
		t.Fatalf("should use custom base, not default: %q", out)
	}
	if strings.Contains(out, "## User preferences") {
		t.Fatalf("no rules → no preferences block: %q", out)
	}
}

func TestPrependUserRules_Interests(t *testing.T) {
	got := prependUserRules("BASEPROMPT", []string{"interested in AI"})
	if !strings.Contains(got, "## User preferences and interests") {
		t.Fatalf("expected reframed heading, got:\n%s", got)
	}
	if !strings.Contains(got, "interest/relevance") {
		t.Fatalf("expected the relevance instruction, got:\n%s", got)
	}
	if !strings.Contains(got, "interested in AI") || !strings.Contains(got, "BASEPROMPT") {
		t.Fatalf("expected rule text + base prompt, got:\n%s", got)
	}
	if prependUserRules("BASEPROMPT", nil) != "BASEPROMPT" {
		t.Fatal("empty rules must return the base prompt unchanged")
	}
}
