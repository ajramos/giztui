# Inbox Action Plan Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an AI-assisted "Action Plan" panel that scans the unread messages already loaded in the current view, groups them into actionable categories via the LLM, and lets the user dispatch each group with a single keystroke (archive/read/trash/label) or fall through to the command palette / prompt configurator.

**Architecture:** A new pure service `InboxAnalyzerService` receives already-in-memory message metadata (ID + Subject + Sender + Snippet) from the UI, splits it into batches, streams each batch through the existing `AIService.ApplyCustomPromptStream`, parses the LLM's JSON into categories with concrete message IDs, merges categories across batches, and reports progress via a callback. A new TUI panel `action_plan.go` collects the unread messages, mounts a side panel in the existing `labelsView` slot, renders the plan, and wires navigation, quick-actions, and escape hatches. No extra Gmail API calls are made — fast mode uses only data already loaded by `MessagePreloader`.

**Tech Stack:** Go, tview (TUI), testify (tests), the existing service layer (`AIService`, `EmailService`, `LabelService`, `PromptService`), `//go:embed` for the default prompt.

### Decisions locked before writing this plan

1. **Default keybinding = `P`** (capital P). The spec said `A`, but `A` is already bound to Attachments in `DefaultKeyBindings()`. `P` is free and mnemonic for "Plan". Fully configurable via `config.json`.
2. **Self-built fast-mode payload.** The analyzer does NOT route through `BulkPromptService` (the spec §7.4 "compose BulkPromptService" line is deliberately not followed). Reasons: that path omits Subject+Sender (the strongest triage signals), makes per-message `repository.GetMessage` calls (network, contradicting fast mode), and does not expose the message→category mapping. The analyzer instead receives in-memory metadata from the UI and calls `AIService.ApplyCustomPromptStream` directly. This is still service-first compliant (a service calling another service). Per-batch SQLite caching is dropped for v1 as a documented consequence.
3. **Component theme = `"ai"`.** Action Plan is an AI feature; `GetComponentColors("ai")` already exists, so no new theme plumbing is added.
4. **Message→category mapping** uses **per-batch local numbering** (1..N within each batch), resolved to concrete message IDs immediately after each batch is parsed. Indices never leak past the parse step; merging unions resolved ID lists.

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `internal/services/interfaces.go` | Modify | Add `InboxAnalyzerService` interface + `AnalyzerMessage`, `ActionPlanCategory`, `ActionPlan`, `InboxAnalyzerOptions` types. |
| `internal/services/inbox_analyzer_service.go` | Create | `InboxAnalyzerServiceImpl`: batching, streaming per batch, JSON parse, index→ID resolution, category merge, progress callback, repair-retry, degrade. |
| `internal/services/inbox_analyzer_prompt.txt` | Create | The built-in default analyzer prompt template (embedded). |
| `internal/services/inbox_analyzer_service_test.go` | Create | Unit tests (batching, merge, parse, repair, cancel, override). |
| `internal/config/config.go` | Modify | Add `ActionPlan` key + default `"P"`; add `InboxAnalyzerConfig` struct, default func, and field on `Config`. |
| `internal/tui/app.go` | Modify | Add `PickerActionPlan` enum + `isActionPlanActive()`; add `inboxAnalyzerService` field, init in `initServices()`, add `GetInboxAnalyzerService()`. |
| `internal/tui/action_plan.go` | Create | The Action Plan panel: collect unread, mount, render, navigate, quick-actions, escape hatches, ESC, override-prompt entry. |
| `internal/tui/action_plan_test.go` | Create | TUI-level tests (unread collection, render, navigation, action mapping). |
| `internal/tui/keys.go` | Modify | `handleConfigurableKey` case for `a.Keys.ActionPlan`; add to `isKeyConfigured`. |
| `internal/tui/commands.go` | Modify | `:action-plan`/`:plan`/`:ap` + `with-prompt`; suggestions. |
| `docs/KEYBOARD_SHORTCUTS.md` | Modify | Document the `P` shortcut and `:action-plan` command family. |

---

## Phase A — Service layer (pure, fully unit-testable with a mock AIService)

### Task 1: Analyzer types + interface

**Files:**
- Modify: `internal/services/interfaces.go` (append near the other service interfaces, e.g. after the `PromptGeneratorService` block ~line 956)

- [ ] **Step 1: Add the types and interface**

Append to `internal/services/interfaces.go`:

```go
// AnalyzerMessage is the lightweight, already-in-memory representation of an inbox
// message handed to the InboxAnalyzerService. The analyzer makes NO Gmail calls — all
// fields come from metadata the UI already loaded via MessagePreloader (fast mode).
type AnalyzerMessage struct {
	ID      string
	Subject string
	From    string
	Snippet string
}

// ActionPlanCategory is one actionable group the LLM produced.
type ActionPlanCategory struct {
	Name        string   // e.g. "Newsletters"
	Priority    string   // "high" | "medium" | "low"
	Description string   // one-line LLM rationale
	Action      string   // "archive" | "mark_read" | "trash" | "label" | "none"
	Label       string   // label name, set only when Action == "label"
	MessageIDs  []string // concrete, resolved message IDs in this category
}

// ActionPlan is the merged result across all batches. It is mutated in place as
// batches complete and handed to the progress callback after each batch.
type ActionPlan struct {
	TotalAnalyzed int                  // messages actually sent to the LLM
	BatchesTotal  int                  // total batches planned
	BatchesDone   int                  // batches completed so far
	Categories    []ActionPlanCategory // merged categories
	ReadManually  []AnalyzerMessage    // messages the LLM declined to categorize
	Degraded      bool                 // true if any batch fell back to best-effort (no actions)
}

// InboxAnalyzerOptions controls a single Analyze invocation.
type InboxAnalyzerOptions struct {
	BatchSize        int    // messages per batch (default 50)
	MaxBatches       int    // safety cap on total batches (default 10)
	CustomPromptText string // empty → use the built-in default analyzer prompt
}

// InboxAnalyzerService groups unread messages into an actionable plan via the LLM.
type InboxAnalyzerService interface {
	// Analyze splits messages into batches, streams each through the AIService, parses
	// categories, resolves them to concrete message IDs, and merges across batches.
	// onProgress (may be nil) is called with the in-progress plan after each batch.
	// Honors context cancellation between and during batches.
	Analyze(ctx context.Context, messages []AnalyzerMessage, opts InboxAnalyzerOptions, onProgress func(*ActionPlan)) (*ActionPlan, error)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/services/...`
Expected: PASS (no implementation yet, just types).

- [ ] **Step 3: Commit**

```bash
git add internal/services/interfaces.go
git commit -m "feat(analyzer): add InboxAnalyzerService interface and types"
```

---

### Task 2: Embedded default analyzer prompt

**Files:**
- Create: `internal/services/inbox_analyzer_prompt.txt`

- [ ] **Step 1: Write the prompt template**

Create `internal/services/inbox_analyzer_prompt.txt` with this exact content:

```
You are an email triage assistant. You are given a numbered list of unread emails,
each with its subject, sender, and a short snippet. Group them into a small number of
actionable categories so the user can clear their inbox quickly.

Rules:
- Use at most 6 categories. Prefer fewer, larger groups.
- Every email number must appear in exactly one category OR in "read_manually".
- Put in "read_manually" only emails that genuinely need the user to read them
  individually (personal questions, deadlines, anything ambiguous).
- For each category choose ONE suggested action from this exact set:
  "archive"   — bulk archival (marketing, newsletters, no engagement needed)
  "mark_read" — informational, no action needed (receipts, confirmations)
  "trash"     — clearly unwanted
  "label"     — needs follow-up; provide a short kebab-case "label" value
  "none"      — no safe bulk action; user should decide
- priority is one of "high", "medium", "low".

Return ONLY a JSON object, no prose, no markdown fences, in exactly this shape:

{
  "categories": [
    {
      "name": "Newsletters",
      "priority": "low",
      "description": "Marketing emails, no engagement",
      "action": "archive",
      "label": "",
      "messages": [1, 4, 7]
    }
  ],
  "read_manually": [2, 5]
}

The emails:
{{messages}}
```

- [ ] **Step 2: Commit**

```bash
git add internal/services/inbox_analyzer_prompt.txt
git commit -m "feat(analyzer): add built-in default analyzer prompt template"
```

---

### Task 3: JSON extraction + response parsing (TDD)

**Files:**
- Create: `internal/services/inbox_analyzer_service.go`
- Create: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/services/inbox_analyzer_service_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run 'TestExtractJSONObject|TestParseAnalyzerResponse' -v`
Expected: FAIL — `extractJSONObject`/`parseAnalyzerResponse` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/services/inbox_analyzer_service.go`:

```go
package services

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed inbox_analyzer_prompt.txt
var defaultAnalyzerPrompt string

// analyzerRawCategory mirrors the JSON the LLM returns for one category.
type analyzerRawCategory struct {
	Name        string `json:"name"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Label       string `json:"label"`
	Messages    []int  `json:"messages"`
}

type analyzerRawResponse struct {
	Categories   []analyzerRawCategory `json:"categories"`
	ReadManually []int                 `json:"read_manually"`
}

// extractJSONObject returns the first balanced {...} object in s, stripping markdown
// fences and surrounding prose. Returns "" if no object is found.
func extractJSONObject(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// parseAnalyzerResponse parses an LLM batch result into categories with concrete message
// IDs, plus the list of per-batch local indices the LLM put in "read_manually".
// batchIDs[i] is the concrete message ID for local number i+1.
func parseAnalyzerResponse(raw string, batchIDs []string) ([]ActionPlanCategory, []int, error) {
	obj := extractJSONObject(raw)
	if obj == "" {
		return nil, nil, fmt.Errorf("no JSON object in analyzer response")
	}
	var parsed analyzerRawResponse
	if err := json.Unmarshal([]byte(obj), &parsed); err != nil {
		return nil, nil, fmt.Errorf("malformed analyzer JSON: %w", err)
	}

	resolve := func(nums []int) []string {
		ids := make([]string, 0, len(nums))
		for _, n := range nums {
			if n >= 1 && n <= len(batchIDs) {
				ids = append(ids, batchIDs[n-1])
			}
		}
		return ids
	}

	cats := make([]ActionPlanCategory, 0, len(parsed.Categories))
	for _, rc := range parsed.Categories {
		ids := resolve(rc.Messages)
		if len(ids) == 0 {
			continue // category with no resolvable messages is useless
		}
		cats = append(cats, ActionPlanCategory{
			Name:        strings.TrimSpace(rc.Name),
			Priority:    normalizePriority(rc.Priority),
			Description: strings.TrimSpace(rc.Description),
			Action:      normalizeAction(rc.Action),
			Label:       strings.TrimSpace(rc.Label),
			MessageIDs:  ids,
		})
	}
	return cats, parsed.ReadManually, nil
}

func normalizePriority(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(p))
	default:
		return "medium"
	}
}

func normalizeAction(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "archive", "mark_read", "trash", "label", "none":
		return strings.ToLower(strings.TrimSpace(a))
	default:
		return "none"
	}
}

// suppress unused import until Analyze is implemented in Task 6
var _ = context.Background
var _ = time.Now
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run 'TestExtractJSONObject|TestParseAnalyzerResponse' -v`
Expected: PASS (all four tests).

- [ ] **Step 5: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): JSON extraction and response parsing with tests"
```

---

### Task 4: Batch splitting (TDD)

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go`
- Modify: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
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
```

(Add `"fmt"` to the test file's imports.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestSplitBatches -v`
Expected: FAIL — `splitBatches` undefined.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/services/inbox_analyzer_service.go`:

```go
// splitBatches divides messages into batches of at most size, capped at maxBatches.
// Messages beyond the cap are dropped (the caller surfaces this to the user).
func splitBatches(messages []AnalyzerMessage, size, maxBatches int) [][]AnalyzerMessage {
	if size <= 0 {
		size = 50
	}
	if maxBatches <= 0 {
		maxBatches = 10
	}
	var batches [][]AnalyzerMessage
	for i := 0; i < len(messages); i += size {
		if len(batches) >= maxBatches {
			break
		}
		end := i + size
		if end > len(messages) {
			end = len(messages)
		}
		batches = append(batches, messages[i:end])
	}
	return batches
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestSplitBatches -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): batch splitting with MaxBatches cap and tests"
```

---

### Task 5: Category merging (TDD)

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go`
- Modify: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
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
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestMergeCategories -v`
Expected: FAIL — `mergeCategories` undefined.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/services/inbox_analyzer_service.go`:

```go
// mergeCategories merges incoming categories into existing, unioning message IDs of
// categories that share a name (case-insensitive) and appending new ones.
func mergeCategories(existing, incoming []ActionPlanCategory) []ActionPlanCategory {
	indexByName := make(map[string]int, len(existing))
	for i, c := range existing {
		indexByName[strings.ToLower(c.Name)] = i
	}
	for _, inc := range incoming {
		key := strings.ToLower(inc.Name)
		if idx, ok := indexByName[key]; ok {
			existing[idx].MessageIDs = unionIDs(existing[idx].MessageIDs, inc.MessageIDs)
			continue
		}
		indexByName[key] = len(existing)
		existing = append(existing, inc)
	}
	return existing
}

// unionIDs appends b's IDs to a, skipping IDs already present, preserving order.
func unionIDs(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			a = append(a, id)
			seen[id] = true
		}
	}
	return a
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestMergeCategories -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): cross-batch category merging with tests"
```

---

### Task 6: Analyze orchestration (TDD)

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go`
- Modify: `internal/services/inbox_analyzer_service_test.go`

This task wires the constructor, the per-batch payload builder, and the `Analyze` method: split → build payload → stream via AIService → parse → resolve read-manually → merge → progress. Includes one repair-retry on malformed JSON, degrade flag, and cancellation.

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`. This reuses the existing `mockAIService` type defined in `prompt_generator_service_test.go` (same package):

```go
import (
	"context" // add if not already imported
	"github.com/stretchr/testify/mock" // add if not already imported
)

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
	// One batch of 3 → LLM returns 2 categories referencing local numbers 1,2,3.
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
	// Batch 1 (msgs m1,m2) → "Newsletters" [1,2].
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(`{"categories":[{"name":"Newsletters","priority":"low","description":"d","action":"archive","label":"","messages":[1,2]}],"read_manually":[]}`, nil).Once()
	// Batch 2 (msgs m3,m4) → "Newsletters" [1] (=m3) + read_manually [2] (=m4).
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
	// First call malformed, repair retry also malformed → batch degrades.
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("garbage", nil).Twice()

	svc := NewInboxAnalyzerService(ai)
	plan, err := svc.Analyze(context.Background(), analyzerMsgs(2),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10}, nil)

	assert.NoError(t, err) // a degraded batch is not a hard error
	assert.True(t, plan.Degraded)
	// Undcategorized messages fall into ReadManually so nothing is lost.
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
	assert.Error(t, err) // first-batch failure aborts (panel won't open)
}

func TestAnalyze_Cancellation(t *testing.T) {
	ai := &mockAIService{}
	svc := NewInboxAnalyzerService(ai)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	_, err := svc.Analyze(ctx, analyzerMsgs(2),
		InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10}, nil)
	assert.ErrorIs(t, err, context.Canceled)
	ai.AssertNotCalled(t, "ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestAnalyze -v`
Expected: FAIL — `NewInboxAnalyzerService` undefined.

- [ ] **Step 3: Write minimal implementation**

Replace the temporary `var _ = context.Background` / `var _ = time.Now` lines at the bottom of `internal/services/inbox_analyzer_service.go` with:

```go
// InboxAnalyzerServiceImpl implements InboxAnalyzerService using the AIService directly.
type InboxAnalyzerServiceImpl struct {
	aiService AIService
}

// NewInboxAnalyzerService creates an inbox analyzer backed by the given AIService.
func NewInboxAnalyzerService(aiService AIService) *InboxAnalyzerServiceImpl {
	return &InboxAnalyzerServiceImpl{aiService: aiService}
}

// buildBatchPayload renders one batch as a compact, numbered list the LLM can reference
// by number. Numbering is local to the batch (1-based).
func buildBatchPayload(batch []AnalyzerMessage) string {
	var b strings.Builder
	for i, m := range batch {
		subject := m.Subject
		if subject == "" {
			subject = "(no subject)"
		}
		fmt.Fprintf(&b, "%d. Subject: %s | From: %s | %s\n", i+1, subject, m.From, m.Snippet)
	}
	return b.String()
}

// buildBatchPrompt injects the batch payload into the chosen prompt template.
func buildBatchPrompt(promptText, payload string) string {
	if strings.Contains(promptText, "{{messages}}") {
		return strings.ReplaceAll(promptText, "{{messages}}", payload)
	}
	return promptText + "\n\n" + payload
}

func (s *InboxAnalyzerServiceImpl) Analyze(ctx context.Context, messages []AnalyzerMessage, opts InboxAnalyzerOptions, onProgress func(*ActionPlan)) (*ActionPlan, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	promptText := opts.CustomPromptText
	if strings.TrimSpace(promptText) == "" {
		promptText = defaultAnalyzerPrompt
	}

	batches := splitBatches(messages, opts.BatchSize, opts.MaxBatches)
	plan := &ActionPlan{BatchesTotal: len(batches)}

	for bi, batch := range batches {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		payload := buildBatchPayload(batch)
		batchIDs := make([]string, len(batch))
		for i, m := range batch {
			batchIDs[i] = m.ID
		}

		cats, readIdx, err := s.runBatch(ctx, promptText, payload, batchIDs)
		if err != nil {
			if bi == 0 {
				// First batch hard failure → abort so the panel never opens.
				return nil, err
			}
			// Intermediate batch failure → keep what we have, mark interrupted.
			return plan, err
		}
		if cats == nil {
			// Degraded batch: surface every message in this batch as read-manually
			// so nothing is silently lost, and flag the plan.
			plan.Degraded = true
			plan.ReadManually = append(plan.ReadManually, batch...)
		} else {
			plan.Categories = mergeCategories(plan.Categories, cats)
			plan.ReadManually = append(plan.ReadManually, resolveMessages(readIdx, batch)...)
		}

		plan.TotalAnalyzed += len(batch)
		plan.BatchesDone = bi + 1
		if onProgress != nil {
			onProgress(plan)
		}
	}

	return plan, nil
}

// runBatch streams one batch through the LLM and parses it, with a single repair retry
// on malformed JSON. Returns (nil, nil, nil) when both attempts are unparseable (caller
// treats this as a degraded batch).
func (s *InboxAnalyzerServiceImpl) runBatch(ctx context.Context, promptText, payload string, batchIDs []string) ([]ActionPlanCategory, []int, error) {
	prompt := buildBatchPrompt(promptText, payload)
	result, err := s.aiService.ApplyCustomPromptStream(ctx, prompt, nil, func(string) {})
	if err != nil {
		return nil, nil, err
	}
	cats, readIdx, perr := parseAnalyzerResponse(result, batchIDs)
	if perr == nil {
		return cats, readIdx, nil
	}

	// Repair retry: re-ask with a strict instruction.
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	repair := prompt + "\n\nIMPORTANT: Your previous answer was not valid JSON. Reply with ONLY the JSON object described above, no prose, no markdown."
	result2, err := s.aiService.ApplyCustomPromptStream(ctx, repair, nil, func(string) {})
	if err != nil {
		return nil, nil, err
	}
	cats, readIdx, perr = parseAnalyzerResponse(result2, batchIDs)
	if perr != nil {
		return nil, nil, nil // degrade
	}
	return cats, readIdx, nil
}

// resolveMessages maps per-batch local indices (1-based) to AnalyzerMessage values.
func resolveMessages(indices []int, batch []AnalyzerMessage) []AnalyzerMessage {
	out := make([]AnalyzerMessage, 0, len(indices))
	for _, n := range indices {
		if n >= 1 && n <= len(batch) {
			out = append(out, batch[n-1])
		}
	}
	return out
}
```

Now remove the now-unused `time` import if the compiler complains (it is no longer referenced). Keep `context`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestAnalyze -v`
Expected: PASS (all Analyze tests).

- [ ] **Step 5: Run the full services suite**

Run: `go test ./internal/services/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): Analyze orchestration with repair-retry, degrade, cancellation"
```

---

## Phase B — Config + wiring

### Task 7: ActionPlan keybinding

**Files:**
- Modify: `internal/config/config.go` (KeyBindings struct ~line 296; DefaultKeyBindings ~line 504)

- [ ] **Step 1: Add the struct field**

In the `KeyBindings` struct, in the "Prompt Configurator" group (right after `PromptTest string \`json:"prompt_test"\``), add:

```go
	// Inbox Action Plan
	ActionPlan string `json:"action_plan"` // Open the AI inbox Action Plan panel
```

- [ ] **Step 2: Add the default**

In `DefaultKeyBindings()`, right after `PromptTest: "ctrl+t",`, add:

```go
		// Inbox Action Plan
		ActionPlan: "P", // capital P (A is taken by Attachments)
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/config/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add ActionPlan keybinding (default P)"
```

---

### Task 8: InboxAnalyzer config block

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add the config struct**

Add near `ThreadingConfig` (a new top-level struct):

```go
// InboxAnalyzerConfig configures the AI inbox Action Plan analyzer.
type InboxAnalyzerConfig struct {
	BatchSize       int    `json:"batch_size"`        // messages per LLM batch (default 50)
	MaxBatches      int    `json:"max_batches"`       // safety cap on batches (default 10)
	DefaultPromptID string `json:"default_prompt_id"` // optional saved-prompt override (name or id)
}
```

- [ ] **Step 2: Add the field to Config**

In the `Config` struct, after the `Threading ThreadingConfig` field, add:

```go
	// Inbox Action Plan analyzer configuration
	InboxAnalyzer InboxAnalyzerConfig `json:"inbox_analyzer"`
```

- [ ] **Step 3: Add the default constructor**

Add near `DefaultThreadingConfig`:

```go
// DefaultInboxAnalyzerConfig returns default analyzer settings.
func DefaultInboxAnalyzerConfig() InboxAnalyzerConfig {
	return InboxAnalyzerConfig{
		BatchSize:       50,
		MaxBatches:      10,
		DefaultPromptID: "",
	}
}
```

- [ ] **Step 4: Wire into DefaultConfig**

In `DefaultConfig()`, where other sub-configs are set (e.g. `Threading: DefaultThreadingConfig(),`), add:

```go
		InboxAnalyzer: DefaultInboxAnalyzerConfig(),
```

- [ ] **Step 5: Verify build**

Run: `go build ./internal/config/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add inbox_analyzer settings block with defaults"
```

---

### Task 9: ActivePicker enum + helper

**Files:**
- Modify: `internal/tui/app.go` (enum ~line 38; helpers ~line 3242)

- [ ] **Step 1: Add the enum constant**

In the `ActivePicker` const block, right after `PickerPromptConfigurator ActivePicker = "prompt_configurator"`, add:

```go
	PickerActionPlan         ActivePicker = "action_plan"
```

- [ ] **Step 2: Add the helper method**

Right after the `isPromptConfiguratorActive()` method, add:

```go
// isActionPlanActive returns true if the Action Plan panel is currently active.
func (a *App) isActionPlanActive() bool {
	return a.currentActivePicker == PickerActionPlan
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: PASS (the helper may be reported as unused until Task 13 — if `go vet`/lint flags it, that is acceptable mid-plan; the build itself passes because methods are never "unused").

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): add PickerActionPlan enum and isActionPlanActive helper"
```

---

### Task 10: Service field + initialization + getter

**Files:**
- Modify: `internal/tui/app.go` (struct field ~line 195; initServices ~line 742; getter ~line 1413)

- [ ] **Step 1: Add the App struct field**

After the `promptGeneratorService services.PromptGeneratorService` field (locate it near the other service fields), add:

```go
	inboxAnalyzerService    services.InboxAnalyzerService
```

- [ ] **Step 2: Initialize it in initServices()**

In `initServices()`, right after the `promptGeneratorService` is created (the `if a.aiService != nil { a.promptGeneratorService = ... }` block), add:

```go
	if a.aiService != nil {
		a.inboxAnalyzerService = services.NewInboxAnalyzerService(a.aiService)
		if a.logger != nil {
			a.logger.Printf("initServices: inbox analyzer service initialized: %v", a.inboxAnalyzerService != nil)
		}
	}
```

- [ ] **Step 3: Add the getter**

After `GetPromptGeneratorService()`, add:

```go
// GetInboxAnalyzerService returns the inbox analyzer service or nil if not initialized.
func (a *App) GetInboxAnalyzerService() services.InboxAnalyzerService {
	return a.inboxAnalyzerService
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./internal/tui/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): wire InboxAnalyzerService into App (field, init, getter)"
```

---

## Phase C — UI panel

### Task 11: Unread collection helper (TDD)

**Files:**
- Create: `internal/tui/action_plan.go`
- Create: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/action_plan_test.go`:

```go
package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
	"github.com/stretchr/testify/assert"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestBuildAnalyzerMessages(t *testing.T) {
	mk := func(id, subj, from, snippet string, unread bool) *gmailapi.Message {
		labels := []string{}
		if unread {
			labels = append(labels, "UNREAD")
		}
		return &gmailapi.Message{
			Id:       id,
			Snippet:  snippet,
			LabelIds: labels,
			Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
				{Name: "Subject", Value: subj},
				{Name: "From", Value: from},
			}},
		}
	}

	metas := []*gmailapi.Message{
		mk("m1", "Hello", "a@x.com", "snip1", true),
		mk("m2", "Read one", "b@x.com", "snip2", false), // read → excluded
		mk("m3", "World", "c@x.com", "snip3", true),
		nil, // defensive: nil entries are skipped
	}

	got := buildAnalyzerMessages(metas)
	assert.Len(t, got, 2)
	assert.Equal(t, services.AnalyzerMessage{ID: "m1", Subject: "Hello", From: "a@x.com", Snippet: "snip1"}, got[0])
	assert.Equal(t, "m3", got[1].ID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestBuildAnalyzerMessages -v`
Expected: FAIL — `buildAnalyzerMessages` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `internal/tui/action_plan.go`:

```go
package tui

import (
	"github.com/ajramos/giztui/internal/services"
	gmailapi "google.golang.org/api/gmail/v1"
)

// buildAnalyzerMessages converts already-loaded message metadata into the lightweight
// AnalyzerMessage list the InboxAnalyzerService consumes. Only UNREAD messages are
// included. No Gmail calls are made — everything comes from in-memory metadata.
func buildAnalyzerMessages(metas []*gmailapi.Message) []services.AnalyzerMessage {
	out := make([]services.AnalyzerMessage, 0, len(metas))
	for _, m := range metas {
		if m == nil {
			continue
		}
		if !isUnreadMeta(m) {
			continue
		}
		out = append(out, services.AnalyzerMessage{
			ID:      m.Id,
			Subject: extractHeaderValue(m, "Subject"),
			From:    extractHeaderValue(m, "From"),
			Snippet: m.Snippet,
		})
	}
	return out
}

// isUnreadMeta reports whether a raw message metadata carries the UNREAD label.
func isUnreadMeta(m *gmailapi.Message) bool {
	for _, l := range m.LabelIds {
		if l == "UNREAD" {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestBuildAnalyzerMessages -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): unread metadata collection for the analyzer with tests"
```

---

### Task 12: Panel state + render (TDD for render)

**Files:**
- Modify: `internal/tui/action_plan.go`
- Modify: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_test.go`:

```go
import "github.com/ajramos/giztui/internal/services" // already imported above

func TestRenderActionPlanText(t *testing.T) {
	plan := &services.ActionPlan{
		TotalAnalyzed: 30,
		BatchesTotal:  1,
		BatchesDone:   1,
		Categories: []services.ActionPlanCategory{
			{Name: "Newsletters", Priority: "low", Description: "marketing", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Follow up", Priority: "high", Description: "needs reply", Action: "label", Label: "needs-reply", MessageIDs: []string{"m3"}},
		},
		ReadManually: []services.AnalyzerMessage{{ID: "m4", Subject: "Budget", From: "cfo@x.com"}},
	}

	// selected index 1 → second category gets the ▸ marker.
	out := renderActionPlanText(plan, 1)

	assert.Contains(t, out, "Newsletters")
	assert.Contains(t, out, "Archive 2")         // action verb + count
	assert.Contains(t, out, "[a]")               // archive key hint
	assert.Contains(t, out, "needs-reply")       // label name shown
	assert.Contains(t, out, "Read manually (1)") // bucket header with count
	assert.Contains(t, out, "Budget")            // read-manually subject
	// The selected (second) category carries the marker; the first does not.
	assert.Contains(t, out, "▸")
}

func TestActionKeyHint(t *testing.T) {
	a := &App{}
	a.Keys.Archive = "a"
	a.Keys.ToggleRead = "t"
	a.Keys.Trash = "d"
	a.Keys.ManageLabels = "l"
	assert.Equal(t, "a", a.actionKeyHint("archive"))
	assert.Equal(t, "t", a.actionKeyHint("mark_read"))
	assert.Equal(t, "d", a.actionKeyHint("trash"))
	assert.Equal(t, "l", a.actionKeyHint("label"))
	assert.Equal(t, "", a.actionKeyHint("none"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'TestRenderActionPlanText|TestActionKeyHint' -v`
Expected: FAIL — `renderActionPlanText`/`actionKeyHint` undefined.

- [ ] **Step 3: Write minimal implementation**

Add to `internal/tui/action_plan.go` (extend imports with `context`, `fmt`, `strings`, `github.com/rivo/tview`):

```go
// actionPlanState holds the mutable state of the Action Plan panel.
type actionPlanState struct {
	plan             *services.ActionPlan
	selectedCategory int
	analyzing        bool // true while batches are still streaming; blocks quick-actions

	customPromptText string // override prompt text, "" = default

	header          *tview.TextView
	body            *tview.TextView
	footer          *tview.TextView
	container       *tview.Flex
	streamingCancel context.CancelFunc
}

// actionVerbLabel maps an action token to a human verb for the category header.
func actionVerbLabel(action string) string {
	switch action {
	case "archive":
		return "Archive"
	case "mark_read":
		return "Mark read"
	case "trash":
		return "Trash"
	case "label":
		return "Label"
	default:
		return "Review"
	}
}

// actionKeyHint returns the configured key for the action's quick-action, or "" if none.
func (a *App) actionKeyHint(action string) string {
	switch action {
	case "archive":
		return a.Keys.Archive
	case "mark_read":
		return a.Keys.ToggleRead
	case "trash":
		return a.Keys.Trash
	case "label":
		return a.Keys.ManageLabels
	default:
		return ""
	}
}

// renderActionPlanText formats a plan into the panel body. selected is the index of the
// currently-highlighted category (or -1). Uses tview dynamic-color tags.
func renderActionPlanText(plan *services.ActionPlan, selected int) string {
	if plan == nil {
		return "Analyzing…"
	}
	var b strings.Builder
	for i, c := range plan.Categories {
		marker := " "
		if i == selected {
			marker = "[::b]▸[::-]"
		}
		key := actionKeyHintForAction(c.Action)
		keyHint := ""
		if key != "" {
			keyHint = fmt.Sprintf("[%s] ", key)
		}
		verb := actionVerbLabel(c.Action)
		fmt.Fprintf(&b, "%s%s%s %d %s   ◀ %s\n", marker, keyHint, verb, len(c.MessageIDs), c.Name, strings.ToUpper(c.Priority))
		if c.Action == "label" && c.Label != "" {
			fmt.Fprintf(&b, "     → label: %s\n", c.Label)
		}
		if c.Description != "" {
			fmt.Fprintf(&b, "     %s\n", c.Description)
		}
		b.WriteString("\n")
	}
	if len(plan.ReadManually) > 0 {
		fmt.Fprintf(&b, "─── Read manually (%d) ───\n", len(plan.ReadManually))
		for i, m := range plan.ReadManually {
			if i >= 10 {
				fmt.Fprintf(&b, "   …and %d more\n", len(plan.ReadManually)-10)
				break
			}
			fmt.Fprintf(&b, "   • %s — %s\n", m.Subject, m.From)
		}
	}
	return b.String()
}

// actionKeyHintForAction is the package-level default-key mapping used by the renderer
// (mirrors App.actionKeyHint defaults; the live panel uses the configured keys directly).
func actionKeyHintForAction(action string) string {
	switch action {
	case "archive":
		return "a"
	case "mark_read":
		return "t"
	case "trash":
		return "d"
	case "label":
		return "l"
	default:
		return ""
	}
}
```

> Note for the implementer: the render test asserts the default hints (`[a]`, etc.). The live panel (Task 13) builds the body via `renderActionPlanText` plus a footer legend built from `a.actionKeyHint(...)` so user-remapped keys appear in the footer. Keeping `actionKeyHintForAction` package-level keeps `renderActionPlanText` a pure function (easy to test).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run 'TestRenderActionPlanText|TestActionKeyHint' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): Action Plan panel state and pure render function with tests"
```

---

### Task 13: Open / mount / analyze the panel

**Files:**
- Modify: `internal/tui/action_plan.go`

This task adds the panel-open path: guard, collect unread, mount in the `labelsView` slot (mirroring `openPromptConfigurator`), launch the analysis goroutine, render progressively, and handle empty/error states. No new test (it is UI plumbing exercised by integration tests in Phase E); follow the exact mount/close pattern from `prompt_configurator.go`.

- [ ] **Step 1: Add the open + close + analyze functions**

Add to `internal/tui/action_plan.go`:

```go
// openActionPlanPanel opens the Action Plan panel using the built-in default prompt.
func (a *App) openActionPlanPanel() {
	a.openActionPlanWithText("")
}

// openActionPlanWithText opens the panel; customPromptText=="" uses the default prompt.
func (a *App) openActionPlanWithText(customPromptText string) {
	if a.GetInboxAnalyzerService() == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Inbox analyzer not available — check LLM configuration")
		return
	}
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}

	// Collect unread messages already in memory (fast mode, zero Gmail calls).
	a.mu.RLock()
	metas := make([]*gmailapi.Message, len(a.messagesMeta))
	copy(metas, a.messagesMeta)
	a.mu.RUnlock()
	messages := buildAnalyzerMessages(metas)
	if len(messages) == 0 {
		a.GetErrorHandler().ShowInfo(a.ctx, "No unread messages in current view. Try :search is:unread or change filter.")
		return
	}

	colors := a.GetComponentColors("ai")
	bg := colors.Background.Color()

	state := &actionPlanState{analyzing: true, selectedCategory: 0, customPromptText: customPromptText}

	state.header = tview.NewTextView().SetDynamicColors(true)
	state.header.SetBackgroundColor(bg)
	state.header.SetTextColor(colors.Text.Color())

	state.body = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	state.body.SetBackgroundColor(bg)
	state.body.SetTextColor(colors.Text.Color())
	state.body.SetText("Analyzing…")

	state.footer = tview.NewTextView().SetDynamicColors(true)
	state.footer.SetBackgroundColor(bg)
	state.footer.SetTextColor(colors.Text.Color())
	state.footer.SetText("[↑↓] navigate  [Enter] action  [:] palette  [p] configurator  [Esc] close")

	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.SetBackgroundColor(bg)
	state.container.SetBorder(true)
	state.container.SetTitle("📋 Action Plan")
	state.container.SetTitleColor(colors.Title.Color())
	state.container.SetBorderColor(colors.Border.Color())
	state.container.AddItem(state.header, 2, 0, false)
	state.container.AddItem(state.body, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)

	a.actionPlanState = state

	// Mount in the shared right-panel slot (same pattern as openPromptConfigurator).
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = state.container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	state.body.SetInputCapture(a.actionPlanInputCapture(state))
	a.SetFocus(state.body)
	a.currentFocus = "action_plan"
	a.updateFocusIndicators("action_plan")
	a.setActivePicker(PickerActionPlan)

	// Launch analysis in the background. ctx cancel is registered both on the state
	// (for closeActionPlanPanel) and on the App (for the global ESC handler in keys.go).
	ctx, cancel := context.WithCancel(a.ctx)
	state.streamingCancel = cancel
	a.streamingCancel = cancel

	batchSize := a.Config.InboxAnalyzer.BatchSize
	maxBatches := a.Config.InboxAnalyzer.MaxBatches

	go func() {
		defer func() {
			cancel()
			if state.streamingCancel != nil {
				state.streamingCancel = nil
			}
			a.streamingCancel = nil
		}()

		_, err := a.GetInboxAnalyzerService().Analyze(ctx, messages,
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText},
			func(p *services.ActionPlan) {
				// Progress callback runs in this goroutine. Direct UI update (NEVER
				// QueueUpdateDraw inside streaming/progress callbacks — CLAUDE.md). Guard
				// with the captured state to avoid writing into a reopened panel.
				if ctx.Err() != nil || a.actionPlanState != state {
					return
				}
				state.plan = p
				a.renderActionPlanPanel(state)
			})

		if ctx.Err() != nil || a.actionPlanState != state {
			return // cancelled or panel replaced; nothing to report
		}
		state.analyzing = false
		if err != nil {
			if state.plan == nil {
				// First-batch failure: no categories at all.
				a.GetErrorHandler().ShowError(a.ctx, "⚠ LLM unavailable. Try again later.")
				return
			}
			a.GetErrorHandler().ShowWarning(a.ctx, "Analysis interrupted — showing partial plan.")
		}
		a.renderActionPlanPanel(state)
		if state.plan != nil && state.plan.Degraded {
			a.GetErrorHandler().ShowInfo(a.ctx, "ℹ Plan rendered with limited actions — some LLM output was malformed.")
		}
	}()
}

// renderActionPlanPanel refreshes header + body from the current state. Safe to call
// from the analysis goroutine (direct SetText, no QueueUpdateDraw).
func (a *App) renderActionPlanPanel(state *actionPlanState) {
	if state == nil || state.plan == nil {
		return
	}
	p := state.plan
	status := "analyzing"
	if !state.analyzing {
		status = "done"
	}
	state.header.SetText(fmt.Sprintf("[::b]%d msgs • batch %d/%d • %s[::-]", p.TotalAnalyzed, p.BatchesDone, p.BatchesTotal, status))
	if state.selectedCategory >= len(p.Categories) {
		state.selectedCategory = len(p.Categories) - 1
	}
	if state.selectedCategory < 0 {
		state.selectedCategory = 0
	}
	state.body.SetText(renderActionPlanText(p, state.selectedCategory))
}

// closeActionPlanPanel closes the panel and restores the list view. Synchronous — no
// QueueUpdateDraw (CLAUDE.md ESC rule).
func (a *App) closeActionPlanPanel() {
	if a.actionPlanState != nil && a.actionPlanState.streamingCancel != nil {
		a.actionPlanState.streamingCancel()
		a.actionPlanState.streamingCancel = nil
	}
	a.streamingCancel = nil

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}

	a.setActivePicker(PickerNone)
	a.actionPlanState = nil

	if list, ok := a.views["list"].(*tview.Table); ok {
		a.SetFocus(list)
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}
```

> Implementer note: confirm `a.Config` is the field name holding `*config.Config` on the App struct (grep `a.Config.` in `internal/tui/`). If the field is named differently (e.g. `a.config`), use that. Also confirm `a.views["list"]` is a `*tview.Table` (it is — see `getCurrentSelectedMessageID` in `columns.go`).

- [ ] **Step 2: Add the App struct field for the panel state**

In `internal/tui/app.go`, right after the `promptConfiguratorState *promptConfiguratorState` field, add:

```go
	actionPlanState         *actionPlanState
```

- [ ] **Step 3: Verify build (input capture stub needed)**

`actionPlanInputCapture` is defined in Task 14. To keep this task compiling on its own, temporarily add a stub at the end of `action_plan.go`:

```go
// actionPlanInputCapture is implemented in Task 14.
func (a *App) actionPlanInputCapture(state *actionPlanState) func(*tcell.EventKey) *tcell.EventKey {
	return nil
}
```

(Add `"github.com/gdamore/tcell/v2"` to imports.)

Run: `go build ./internal/tui/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/app.go
git commit -m "feat(tui): mount Action Plan panel and run analysis progressively"
```

---

### Task 14: Panel navigation, quick-actions, escape hatches

**Files:**
- Modify: `internal/tui/action_plan.go` (replace the `actionPlanInputCapture` stub)

- [ ] **Step 1: Replace the stub with the real input capture**

Replace the stub `actionPlanInputCapture` with:

```go
// actionPlanInputCapture handles all key input while the Action Plan panel is focused.
func (a *App) actionPlanInputCapture(state *actionPlanState) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		// ESC: synchronous close (no QueueUpdateDraw).
		if ev.Key() == tcell.KeyEscape {
			a.closeActionPlanPanel()
			return nil
		}

		// Navigation works during and after analysis.
		switch ev.Key() {
		case tcell.KeyUp:
			a.moveActionPlanSelection(state, -1)
			return nil
		case tcell.KeyDown:
			a.moveActionPlanSelection(state, +1)
			return nil
		}

		key := string(ev.Rune())

		// Escape hatches (available any time).
		if key == a.Keys.CommandMode { // ':'
			a.actionPlanOpenPalette(state)
			return nil
		}
		if key == a.Keys.Prompt { // 'p'
			a.actionPlanOpenConfigurator(state)
			return nil
		}

		// Quick-actions are blocked until analysis finishes (avoids racing the plan).
		if state.analyzing {
			return nil
		}
		switch key {
		case a.Keys.Archive:
			a.executeActionPlanAction(state, "archive")
			return nil
		case a.Keys.ToggleRead:
			a.executeActionPlanAction(state, "mark_read")
			return nil
		case a.Keys.Trash:
			a.executeActionPlanAction(state, "trash")
			return nil
		case a.Keys.ManageLabels:
			a.executeActionPlanAction(state, "label")
			return nil
		}
		return ev
	}
}

// moveActionPlanSelection moves the highlight and re-renders.
func (a *App) moveActionPlanSelection(state *actionPlanState, delta int) {
	if state.plan == nil || len(state.plan.Categories) == 0 {
		return
	}
	state.selectedCategory += delta
	if state.selectedCategory < 0 {
		state.selectedCategory = 0
	}
	if state.selectedCategory >= len(state.plan.Categories) {
		state.selectedCategory = len(state.plan.Categories) - 1
	}
	a.renderActionPlanPanel(state)
}

// currentCategory returns the selected category or nil.
func (a *App) currentActionPlanCategory(state *actionPlanState) *services.ActionPlanCategory {
	if state.plan == nil || state.selectedCategory < 0 || state.selectedCategory >= len(state.plan.Categories) {
		return nil
	}
	return &state.plan.Categories[state.selectedCategory]
}

// executeActionPlanAction runs a bulk action on the selected category's messages.
func (a *App) executeActionPlanAction(state *actionPlanState, action string) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	ids := make([]string, len(cat.MessageIDs))
	copy(ids, cat.MessageIDs)
	catName := cat.Name
	label := cat.Label

	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

	go func() {
		var err error
		switch action {
		case "archive":
			err = emailService.BulkArchive(a.ctx, ids)
		case "mark_read":
			err = emailService.BulkMarkAsRead(a.ctx, ids)
		case "trash":
			err = emailService.BulkTrash(a.ctx, ids)
		case "label":
			if label == "" {
				a.GetErrorHandler().ShowWarning(a.ctx, "Category has no label to apply")
				return
			}
			err = a.applyActionPlanLabel(labelService, ids, label)
		default:
			return
		}
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Action failed on %q: %v", catName, err))
			return
		}
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("✓ %s applied to %d messages (%s)", actionVerbLabel(action), len(ids), catName))
		a.removeActionPlanCategory(state, catName)
	}()
}

// removeActionPlanCategory drops a completed category and re-renders.
func (a *App) removeActionPlanCategory(state *actionPlanState, name string) {
	if state.plan == nil {
		return
	}
	kept := state.plan.Categories[:0]
	for _, c := range state.plan.Categories {
		if c.Name != name {
			kept = append(kept, c)
		}
	}
	state.plan.Categories = kept
	a.renderActionPlanPanel(state)
}

// actionPlanOpenPalette sets a virtual bulk selection over the category's messages and
// opens the command palette (the ':' escape hatch).
func (a *App) actionPlanOpenPalette(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	a.setVirtualBulkSelection(cat.MessageIDs)
	a.closeActionPlanPanel()
	a.showCommandBar()
}

// actionPlanOpenConfigurator opens the bulk prompt picker scoped to the category (the
// 'p' escape hatch).
func (a *App) actionPlanOpenConfigurator(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	a.setVirtualBulkSelection(cat.MessageIDs)
	a.closeActionPlanPanel()
	go a.openBulkPromptPicker()
}

// setVirtualBulkSelection marks the given IDs as selected and enables bulk mode so the
// existing command palette / bulk picker operate on exactly these messages.
func (a *App) setVirtualBulkSelection(ids []string) {
	a.mu.Lock()
	if a.selected == nil {
		a.selected = make(map[string]bool)
	} else {
		for k := range a.selected {
			delete(a.selected, k)
		}
	}
	for _, id := range ids {
		a.selected[id] = true
	}
	a.bulkMode = true
	a.mu.Unlock()
}
```

- [ ] **Step 2: Add the label-apply helper**

The `LabelService.BulkApplyLabel` takes a label **ID**, but categories carry a label **name**. Add a resolver that finds-or-creates the label by name. Check whether `LabelService` exposes a find/create method (grep `internal/services/interfaces.go` for `Label` methods such as `GetOrCreateLabel`, `FindLabelByName`, `CreateLabel`). Implement:

```go
// applyActionPlanLabel resolves a label name to an ID (creating it if needed) and applies
// it to the messages in bulk.
func (a *App) applyActionPlanLabel(labelService services.LabelService, ids []string, labelName string) error {
	labelID, err := a.resolveOrCreateLabelID(a.ctx, labelService, labelName)
	if err != nil {
		return err
	}
	return labelService.BulkApplyLabel(a.ctx, ids, labelID)
}
```

> Implementer note: implement `resolveOrCreateLabelID` using whatever `LabelService` already offers. Search `internal/tui/labels.go` and `internal/services/label_service.go` for an existing "ensure label exists by name" helper (the manual label flow already does this when the user types a new label name) and reuse it. If a suitable helper exists on the App (e.g. used by `:label <name>`), call that instead of adding a new one. Do NOT invent a new Gmail call path — reuse the existing label create/apply flow. Write a focused unit test for `resolveOrCreateLabelID` against a mocked `LabelService` if you add new logic.

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: PASS.

- [ ] **Step 4: Run TUI tests**

Run: `go test ./internal/tui/ -run 'TestBuildAnalyzerMessages|TestRenderActionPlanText|TestActionKeyHint' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan.go
git commit -m "feat(tui): Action Plan navigation, quick-actions, and escape hatches"
```

---

### Task 15: Override-prompt resolution

**Files:**
- Modify: `internal/tui/action_plan.go`

- [ ] **Step 1: Add the override entry point**

Add to `internal/tui/action_plan.go`:

```go
// openActionPlanWithPrompt opens the panel using a saved prompt (by name or numeric id)
// as the analyzer override. Falls back to the default prompt if the prompt is not found.
func (a *App) openActionPlanWithPrompt(nameOrID string) {
	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Prompt library unavailable — using default analyzer prompt")
		a.openActionPlanWithText("")
		return
	}

	var tmpl *services.PromptTemplate
	var err error
	if id, convErr := strconv.Atoi(nameOrID); convErr == nil {
		tmpl, err = promptService.GetPrompt(a.ctx, id)
	} else {
		tmpl, err = promptService.FindPromptByName(a.ctx, nameOrID)
	}
	if err != nil || tmpl == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("⚠ Prompt %q not found. Using default analyzer prompt.", nameOrID))
		a.openActionPlanWithText("")
		return
	}
	a.openActionPlanWithText(tmpl.PromptText)
}
```

(Add `"strconv"` to imports. `PromptTemplate` is the type returned by `PromptService.GetPrompt`; confirm its package — it is `services.PromptTemplate` per `interfaces.go`, aliasing `prompts.PromptTemplate`. If `GetPrompt` returns `*prompts.PromptTemplate`, import that package and use it.)

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/action_plan.go
git commit -m "feat(tui): action plan override-prompt resolution with fallback"
```

---

## Phase D — Keys + commands parity

### Task 16: Keyboard shortcut

**Files:**
- Modify: `internal/tui/keys.go` (`handleConfigurableKey` ~line 312; `isKeyConfigured` ~line 509)

- [ ] **Step 1: Add the key handler case**

In `handleConfigurableKey`, alongside the other feature cases (e.g. near the Slack/Prompt cases), add:

```go
	case a.Keys.ActionPlan:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> action_plan", key)
		}
		go a.openActionPlanPanel()
		return true
```

- [ ] **Step 2: Register the key as configured**

In `isKeyConfigured`, add to the boolean chain:

```go
		keyStr == a.Keys.ActionPlan ||
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: PASS.

- [ ] **Step 4: Manual sanity (deferred to Phase E E2E)** — no unit test for raw key plumbing; covered by the tmux smoke test.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/keys.go
git commit -m "feat(tui): bind P to open the Action Plan panel"
```

---

### Task 17: Command parity

**Files:**
- Modify: `internal/tui/commands.go` (`executeCommand` switch ~line 800; `generateCommandSuggestion` ~line 239)

- [ ] **Step 1: Add the dispatcher case**

In `executeCommand`'s `switch command {`, add:

```go
	case "action-plan", "plan", "ap":
		a.executeActionPlanCommand(args)
```

- [ ] **Step 2: Add the command handler**

Add near the other `execute*Command` functions:

```go
// executeActionPlanCommand handles :action-plan / :plan / :ap [with-prompt <name-or-id>].
func (a *App) executeActionPlanCommand(args []string) {
	if len(args) == 0 {
		go a.openActionPlanPanel()
		return
	}
	if strings.ToLower(args[0]) == "with-prompt" {
		if len(args) < 2 {
			a.GetErrorHandler().ShowError(a.ctx, "Usage: :action-plan with-prompt <name-or-id>")
			return
		}
		nameOrID := strings.Join(args[1:], " ")
		go a.openActionPlanWithPrompt(nameOrID)
		return
	}
	a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Unknown action-plan option: %s", args[0]))
}
```

- [ ] **Step 3: Add tab-completion suggestions**

In `generateCommandSuggestion`, add `"action-plan"` (and the family) to the suggestion source the same way existing commands are registered. Locate the command-name list/map used for prefix completion and add `action-plan`, `plan`, `ap`. Do NOT reuse `pl` (already maps to `preload`).

- [ ] **Step 4: Verify build**

Run: `go build ./internal/tui/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat(tui): add :action-plan command family (parity with P shortcut)"
```

---

## Phase E — Integration, docs, gate

### Task 18: Integration tests (happy path + override + degrade)

**Files:**
- Create or extend: `internal/tui/action_plan_integration_test.go`

These tests exercise the service end-to-end with a mocked AIService (no real LLM, no Gmail). They construct an `App` with the analyzer service wired to a `mockAIService` and a populated `messagesMeta`, then drive the panel logic directly (calling the exported-within-package functions). Where constructing a full `App` is impractical, test the service + panel-state functions together.

- [ ] **Step 1: Write the integration test**

Create `internal/tui/action_plan_integration_test.go`:

```go
package tui

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	"github.com/stretchr/testify/assert"
)

// fakeAnalyzer lets us drive the panel render/action logic without a real LLM.
type fakeAnalyzer struct {
	plan *services.ActionPlan
	err  error
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, msgs []services.AnalyzerMessage, opts services.InboxAnalyzerOptions, onProgress func(*services.ActionPlan)) (*services.ActionPlan, error) {
	if f.err != nil {
		return nil, f.err
	}
	if onProgress != nil {
		onProgress(f.plan)
	}
	return f.plan, f.err
}

func TestActionPlan_RenderAndRemoveCategory(t *testing.T) {
	plan := &services.ActionPlan{
		TotalAnalyzed: 3, BatchesTotal: 1, BatchesDone: 1,
		Categories: []services.ActionPlanCategory{
			{Name: "Newsletters", Priority: "low", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Follow up", Priority: "high", Action: "label", Label: "needs-reply", MessageIDs: []string{"m3"}},
		},
	}
	a := &App{}
	a.Keys.Archive, a.Keys.ToggleRead, a.Keys.Trash, a.Keys.ManageLabels = "a", "t", "d", "l"
	state := &actionPlanState{plan: plan, selectedCategory: 0}

	// Navigation clamps within range.
	a.moveActionPlanSelection(state, +5)
	assert.Equal(t, 1, state.selectedCategory)
	a.moveActionPlanSelection(state, -5)
	assert.Equal(t, 0, state.selectedCategory)

	// Removing a completed category drops it.
	a.removeActionPlanCategory(state, "Newsletters")
	assert.Len(t, state.plan.Categories, 1)
	assert.Equal(t, "Follow up", state.plan.Categories[0].Name)
}

func TestActionPlan_VirtualBulkSelection(t *testing.T) {
	a := &App{selected: map[string]bool{"old": true}}
	a.setVirtualBulkSelection([]string{"m1", "m2"})
	assert.True(t, a.bulkMode)
	assert.True(t, a.selected["m1"])
	assert.True(t, a.selected["m2"])
	assert.False(t, a.selected["old"]) // previous selection cleared
}
```

> Implementer note: `App` has many fields; if `&App{}` is insufficient (nil maps/mutex panics), use the existing TUI test helper that builds a minimal App (search `internal/tui/*_test.go` for a `newTestApp`/`setupTestApp` helper and reuse it). The mutex `a.mu` is a zero-value `sync.RWMutex` which is valid; `a.ctx` may need setting to `context.Background()`.

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/tui/ -run TestActionPlan -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/action_plan_integration_test.go
git commit -m "test(tui): action plan render, navigation, and bulk-selection integration"
```

---

### Task 19: Manual end-to-end smoke test (real LLM via tmux)

**The hard-won lesson from the Prompt Configurator: unit tests + code review did NOT catch the real bugs — only manual E2E with a real LLM did.** Drive the app yourself; do not ask the user to beta-test. Use only non-destructive keys.

- [ ] **Step 1: Build**

Run: `make build`
Expected: binary at `./build/giztui`, no errors.

- [ ] **Step 2: Launch in a detached tmux session**

```bash
/usr/bin/tmux kill-session -t giztest 2>/dev/null; /usr/bin/tmux new-session -d -s giztest -x 140 -y 40 "./build/giztui"
sleep 8
/usr/bin/tmux capture-pane -t giztest -p | tail -40
```

Expected: inbox list renders with unread messages.

- [ ] **Step 3: Open the Action Plan panel**

```bash
/usr/bin/tmux send-keys -t giztest 'P'
sleep 20
/usr/bin/tmux capture-pane -t giztest -p | tail -40
```

Expected: panel opens; header shows `N msgs • batch x/y`; categories stream in with priority + counts + `[a]/[t]/[d]/[l]` hints; a "Read manually" bucket appears. Verify category counts look sane vs. the inbox.

- [ ] **Step 4: Navigate (non-destructive)**

```bash
/usr/bin/tmux send-keys -t giztest Down; sleep 1; /usr/bin/tmux send-keys -t giztest Down; sleep 1
/usr/bin/tmux capture-pane -t giztest -p | tail -40
```

Expected: the ▸ marker moves between categories; no flicker/crash.

- [ ] **Step 5: Test the `:` escape hatch (do NOT execute a destructive command)**

```bash
/usr/bin/tmux send-keys -t giztest ':'
sleep 1
/usr/bin/tmux capture-pane -t giztest -p | tail -10
/usr/bin/tmux send-keys -t giztest Escape
sleep 1
```

Expected: command bar opens (🐶> prompt) with the category's messages held as bulk selection; ESC dismisses it cleanly.

- [ ] **Step 6: Test ESC during analysis (cancellation)**

Reopen the panel and immediately press ESC while it is still analyzing:

```bash
/usr/bin/tmux send-keys -t giztest 'P'; sleep 2; /usr/bin/tmux send-keys -t giztest Escape; sleep 2
/usr/bin/tmux capture-pane -t giztest -p | tail -20
```

Expected: panel closes immediately, focus returns to the list, no deadlock/hang. Check `~/.config/giztui/giztui.log` for clean cancellation.

- [ ] **Step 7: Verify the `:action-plan` command works**

```bash
/usr/bin/tmux send-keys -t giztest ':action-plan' Enter; sleep 20
/usr/bin/tmux capture-pane -t giztest -p | tail -40
```

Expected: identical panel to the `P` shortcut.

- [ ] **Step 8: (Optional, destructive — only on a throwaway/safe category) Execute one quick action**

Only if you have a clearly-safe category (e.g. archive newsletters) and the user consents. Otherwise skip and note it as user-verifiable. Capture the success status and confirm the category disappears.

- [ ] **Step 9: Tear down**

```bash
/usr/bin/tmux send-keys -t giztest 'q'; sleep 1; /usr/bin/tmux kill-session -t giztest 2>/dev/null
```

- [ ] **Step 10: Record findings** — note any bug (placeholder substitution, focus, parse failures, header drift, off-thread close) and fix before proceeding. Re-run the relevant unit tests after any fix.

---

### Task 20: Documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md`
- Modify: example config (search for the canonical config example, e.g. `config.example.json` or the keys section in docs)

- [ ] **Step 1: Document the shortcut and commands**

In `docs/KEYBOARD_SHORTCUTS.md`, add a row/section: `P` → "Open AI Action Plan (triage unread inbox)"; and document the command family `:action-plan` / `:plan` / `:ap`, plus `:action-plan with-prompt <name-or-id>`.

- [ ] **Step 2: Document config**

Document the `inbox_analyzer` config block (`batch_size`, `max_batches`, `default_prompt_id`) and the `keys.action_plan` override wherever AI/config settings are documented.

- [ ] **Step 3: Commit**

```bash
git add docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs: document Action Plan shortcut, commands, and config"
```

---

### Task 21: Final gate

- [ ] **Step 1: Run the full pre-commit check**

Run: `make pre-commit-check`
Expected: PASS (fmt + vet + golangci-lint + essential tests). Fix anything it flags.

> Known caveat: golangci-lint in file-mode can emit false `typecheck` errors for `internal/tui` due to cross-file symbol resolution. If you see those, confirm with `go build ./...` and `go vet ./internal/tui/...` that the code is actually clean.

- [ ] **Step 2: Run the scoped test suite**

Run: `make test`
Expected: PASS.

- [ ] **Step 3: Final commit if any fixups were needed**

```bash
git add -A internal/ docs/    # stage specific dirs, NOT bare git add -A
git commit -m "chore(analyzer): pre-commit fixups"
```

> Reminder from the prior session: avoid bare `git add -A` from repo root (it once committed a stray runtime lock file). Stage specific paths.

- [ ] **Step 4: Finish the branch**

Use the `superpowers:finishing-a-development-branch` skill to choose merge/PR/cleanup.

---

## Self-Review (completed by plan author)

**Spec coverage** (Feature 1 sections of `2026-06-06-inbox-analysis-prompt-configurator-design.md`):

- §5.1 batching 50–100 configurable → Task 4 + Task 8 (`InboxAnalyzerConfig.BatchSize`). ✓
- §5.2 categorized tabular report with quick-actions → Task 12 render + Task 14 actions. ✓
- §5.3 `:` palette + `p` configurator escape hatches → Task 14. ✓
- §5.4 default fixed buckets + custom-prompt override → default prompt Task 2; override Task 15 + Task 17. ✓
- §7.3 panel layout (header, categories, read-manually, legend) → Task 12 + Task 13. ✓
- §7.4 analyzer engine (batch, stream, parse, merge, progress) → Tasks 3–6. ✓
- §8.1 fast mode, zero extra Gmail calls → Task 11 (in-memory metadata) + decision #2. ✓
- §8.2 quick action via existing bulk methods → Task 14 (`BulkArchive`/`BulkTrash`/`BulkMarkAsRead`/`BulkApplyLabel`). ✓
- §8.3 `:` virtual bulk selection → Task 14 `setVirtualBulkSelection` + `showCommandBar`. ✓
- §8.4 `p` configurator scoped → Task 14 `openBulkPromptPicker`. ✓
- §8.6 override with saved prompt + fallback when not found → Task 15. ✓
- §8.8 ESC behavior (cancel batch / close, preserve partial) → Task 13 close + Task 14 ESC. ✓
- §9 error handling (LLM unavailable, unparseable→repair→degrade, empty context) → Task 6 (repair/degrade) + Task 13 (empty/first-batch/interrupted). ✓
- §10 configurable keys (default + config) → Task 7 + Task 16. ✓
- §11 settings block → Task 8. ✓
- §12 command parity → Task 17. ✓
- §13 testing (service ≥85%, UI hot paths, integration scenarios) → Tasks 3–6, 11, 12, 18; manual E2E Task 19. ✓
- §14.4 default prompt in a dedicated file → Task 2 (embedded `inbox_analyzer_prompt.txt`). ✓

**Deviations from spec (intentional, see Decisions):** key `P` not `A` (#1); analyzer calls AIService directly, not BulkPromptService (#2); per-batch SQLite caching dropped for v1 (#2); theme component `"ai"` reused (#3).

**Placeholder scan:** Three implementer notes intentionally defer to existing code rather than invent APIs: (a) the exact `App` config field name (`a.Config` vs `a.config`); (b) the `resolveOrCreateLabelID` label find/create helper — must reuse the existing manual-label flow rather than add a new Gmail path; (c) the `generateCommandSuggestion` registration site. These are "use the existing pattern" notes, not missing logic — each names exactly what to grep for and what to reuse. All net-new logic has complete code.

**Type consistency:** `ActionPlan`, `ActionPlanCategory`, `AnalyzerMessage`, `InboxAnalyzerOptions`, `InboxAnalyzerService` are defined once (Task 1) and used consistently. `Analyze(ctx, []AnalyzerMessage, InboxAnalyzerOptions, func(*ActionPlan))` signature matches across service impl (Task 6), App goroutine (Task 13), and tests (Task 6/18). Action tokens (`archive`/`mark_read`/`trash`/`label`/`none`) are consistent across the prompt (Task 2), `normalizeAction` (Task 3), `actionVerbLabel`/`actionKeyHint` (Task 12), and the input capture (Task 14).
