# Analyzer Body Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Feed the inbox Action Plan analyzer the plain-text email body (truncated, opt-in) in addition to subject/sender/snippet, so classification quality improves.

**Architecture:** A config flag (`include_body`, default on) + char limit (`body_char_limit`, default 1000) gate body inclusion. `AnalyzerMessage` gains a `Body` field. `EmailService.GetMessagePlainTexts` fetches bodies concurrently (reusing `GetMessagesParallel` + `ExtractPlainText`). The TUI analyze flow caps the message list to `BatchSize × MaxBatches`, fetches bodies with progress, and populates `Body`. `buildBatchPayload` renders the truncated body when present, else the snippet. The analyzer service stays Gmail-free.

**Tech Stack:** Go, Gmail API (`google.golang.org/api/gmail/v1`), GizTUI services + tview TUI.

Spec: `docs/superpowers/specs/2026-06-11-analyzer-body-context-design.md`

---

### Task 1: Config fields + defaults

**Files:**
- Modify: `internal/config/config.go:224-228` (struct), `internal/config/config.go:598-603` (defaults)
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_test.go`:

```go
func TestDefaultInboxAnalyzerConfig_BodyContext(t *testing.T) {
	c := DefaultInboxAnalyzerConfig()
	if !c.IncludeBody {
		t.Fatal("IncludeBody should default to true")
	}
	if c.BodyCharLimit != 1000 {
		t.Fatalf("BodyCharLimit should default to 1000, got %d", c.BodyCharLimit)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDefaultInboxAnalyzerConfig_BodyContext -v`
Expected: FAIL — `c.IncludeBody undefined` (build error).

- [ ] **Step 3: Add struct fields**

In `internal/config/config.go`, replace the `InboxAnalyzerConfig` struct (lines 224-228):

```go
type InboxAnalyzerConfig struct {
	BatchSize       int    `json:"batch_size"`        // messages per LLM batch (default 50)
	MaxBatches      int    `json:"max_batches"`       // safety cap on batches (default 10)
	DefaultPromptID string `json:"default_prompt_id"` // optional saved-prompt override (name or id)
	IncludeBody     bool   `json:"include_body"`      // include plain-text body in analyzer context (default true)
	BodyCharLimit   int    `json:"body_char_limit"`   // max body chars per email (default 1000)
}
```

- [ ] **Step 4: Add defaults**

In `internal/config/config.go`, replace `DefaultInboxAnalyzerConfig` (lines 598-603):

```go
func DefaultInboxAnalyzerConfig() InboxAnalyzerConfig {
	return InboxAnalyzerConfig{
		BatchSize:       50,
		MaxBatches:      10,
		DefaultPromptID: "",
		IncludeBody:     true,
		BodyCharLimit:   1000,
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestDefaultInboxAnalyzerConfig_BodyContext -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): analyzer include_body + body_char_limit (default on, 1000)"
```

---

### Task 2: Data model fields (`Body`, `BodyCharLimit`)

**Files:**
- Modify: `internal/services/interfaces.go` (`AnalyzerMessage`, `InboxAnalyzerOptions`)

No test (compile-only struct fields; behavior tested in Tasks 3-4).

- [ ] **Step 1: Add `Body` to `AnalyzerMessage`**

In `internal/services/interfaces.go`, replace the `AnalyzerMessage` struct:

```go
type AnalyzerMessage struct {
	ID      string
	Subject string
	From    string
	Snippet string
	Body    string // plain-text body (truncated upstream); empty → fall back to Snippet
}
```

- [ ] **Step 2: Add `BodyCharLimit` to `InboxAnalyzerOptions`**

In `internal/services/interfaces.go`, replace the `InboxAnalyzerOptions` struct:

```go
type InboxAnalyzerOptions struct {
	BatchSize        int      // messages per batch (default 50)
	MaxBatches       int      // safety cap on total batches (default 10)
	CustomPromptText string   // empty → use the built-in default analyzer prompt
	UserRules        []string // free-text preference rules prepended to the prompt; empty → none
	BodyCharLimit    int      // max body chars rendered per email; <= 0 → no extra trim
}
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add internal/services/interfaces.go
git commit -m "feat(services): AnalyzerMessage.Body + InboxAnalyzerOptions.BodyCharLimit"
```

---

### Task 3: `truncateForAnalyzer` pure helper

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go`
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
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
	// Empty input.
	if got := truncateForAnalyzer("   ", 10); got != "" {
		t.Fatalf("empty/whitespace-only: got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestTruncateForAnalyzer -v`
Expected: FAIL — `undefined: truncateForAnalyzer`.

- [ ] **Step 3: Write the implementation**

In `internal/services/inbox_analyzer_service.go`, add (after `buildBatchPayload`):

```go
// truncateForAnalyzer collapses runs of whitespace (incl. newlines) to single spaces, trims
// the ends, and cuts to at most limit runes on a rune boundary. limit <= 0 skips the cut.
func truncateForAnalyzer(text string, limit int) string {
	collapsed := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if limit <= 0 {
		return collapsed
	}
	r := []rune(collapsed)
	if len(r) <= limit {
		return collapsed
	}
	return string(r[:limit])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestTruncateForAnalyzer -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): truncateForAnalyzer whitespace-collapse + rune-safe cut"
```

---

### Task 4: `buildBatchPayload` renders body when present

**Files:**
- Modify: `internal/services/inbox_analyzer_service.go:214-225` (`buildBatchPayload`), `:273` (call site in `Analyze`)
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
func TestBuildBatchPayload_BodyVsSnippet(t *testing.T) {
	batch := []AnalyzerMessage{
		{Subject: "Hello", From: "a@x.com", Snippet: "snip-a", Body: "this is the full body of email A"},
		{Subject: "World", From: "b@x.com", Snippet: "snip-b"}, // no body → snippet
	}
	out := buildBatchPayload(batch, 1000)

	// Message 1 uses its body, not the snippet.
	if !strings.Contains(out, "full body of email A") {
		t.Fatalf("expected body for msg 1, got:\n%s", out)
	}
	if strings.Contains(out, "snip-a") {
		t.Fatalf("msg 1 should not fall back to snippet, got:\n%s", out)
	}
	// Message 2 (no body) falls back to its snippet.
	if !strings.Contains(out, "snip-b") {
		t.Fatalf("expected snippet for msg 2, got:\n%s", out)
	}

	// Body is truncated to the limit.
	long := []AnalyzerMessage{{Subject: "L", From: "c@x.com", Body: strings.Repeat("x", 50)}}
	if out := buildBatchPayload(long, 10); strings.Count(out, "x") != 10 {
		t.Fatalf("body should be truncated to 10 x's, got %d", strings.Count(out, "x"))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestBuildBatchPayload_BodyVsSnippet -v`
Expected: FAIL — too many arguments to `buildBatchPayload` (build error).

- [ ] **Step 3: Update `buildBatchPayload`**

In `internal/services/inbox_analyzer_service.go`, replace `buildBatchPayload` (lines 214-225):

```go
// buildBatchPayload renders one batch as a compact, numbered list the LLM can reference
// by number. Numbering is local to the batch (1-based). When a message has a Body it is
// rendered (truncated to bodyCharLimit) on its own line; otherwise the Snippet is used inline.
func buildBatchPayload(batch []AnalyzerMessage, bodyCharLimit int) string {
	var b strings.Builder
	for i, m := range batch {
		subject := strings.ReplaceAll(m.Subject, "\n", " ")
		if strings.TrimSpace(subject) == "" {
			subject = "(no subject)"
		}
		if strings.TrimSpace(m.Body) != "" {
			fmt.Fprintf(&b, "%d. Subject: %s | From: %s\n   %s\n", i+1, subject, m.From, truncateForAnalyzer(m.Body, bodyCharLimit))
			continue
		}
		snippet := strings.ReplaceAll(m.Snippet, "\n", " ")
		fmt.Fprintf(&b, "%d. Subject: %s | From: %s | %s\n", i+1, subject, m.From, snippet)
	}
	return b.String()
}
```

- [ ] **Step 4: Update the call site in `Analyze`**

In `internal/services/inbox_analyzer_service.go`, change line 273 from:

```go
		payload := buildBatchPayload(batch)
```

to:

```go
		payload := buildBatchPayload(batch, opts.BodyCharLimit)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/services/ -run 'TestBuildBatchPayload|TestTruncateForAnalyzer' -v`
Expected: PASS (and no other analyzer test breaks).

- [ ] **Step 6: Commit**

```bash
git add internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): render truncated body in batch payload, snippet fallback"
```

---

### Task 5: `EmailService.GetMessagePlainTexts` + `plainTextsByID` helper

**Files:**
- Modify: `internal/services/interfaces.go` (`EmailService` interface), `internal/services/email_service.go`
- Test: `internal/services/email_service_test.go`

- [ ] **Step 1: Write the failing test for the pure helper**

Append to `internal/services/email_service_test.go`:

```go
func TestPlainTextsByID(t *testing.T) {
	mk := func(id, text string) *gmail_v1.Message {
		return &gmail_v1.Message{
			Id: id,
			Payload: &gmail_v1.MessagePart{
				Body: &gmail_v1.MessagePartBody{Data: base64.URLEncoding.EncodeToString([]byte(text))},
			},
		}
	}
	msgs := []*gmail_v1.Message{mk("a", "body-a"), nil, mk("b", "body-b")}
	got := plainTextsByID(msgs)

	if got["a"] != "body-a" || got["b"] != "body-b" {
		t.Fatalf("unexpected map: %#v", got)
	}
	if len(got) != 2 {
		t.Fatalf("nil messages must be skipped, got %d entries: %#v", len(got), got)
	}
}
```

Add `"encoding/base64"` to the test file's imports if not present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestPlainTextsByID -v`
Expected: FAIL — `undefined: plainTextsByID`.

- [ ] **Step 3: Add the interface method**

In `internal/services/interfaces.go`, add to the `EmailService` interface (after `MoveToSystemFolder`, line 38):

```go
	GetMessagePlainTexts(ctx context.Context, ids []string, maxWorkers int) (map[string]string, error)
```

- [ ] **Step 4: Implement helper + method**

In `internal/services/email_service.go`, add (e.g. after `SaveMessageToFile`):

```go
// plainTextsByID maps each non-nil message's ID to its extracted plain text.
func plainTextsByID(msgs []*gmail_v1.Message) map[string]string {
	out := make(map[string]string, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		out[m.Id] = gmail.ExtractPlainText(m)
	}
	return out
}

// GetMessagePlainTexts fetches plain-text bodies for the given message IDs concurrently.
// Returns id -> plain text; IDs that fail to fetch are simply absent from the map.
func (s *EmailServiceImpl) GetMessagePlainTexts(ctx context.Context, ids []string, maxWorkers int) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	if s.gmailClient == nil {
		return nil, fmt.Errorf("gmail client not available")
	}
	msgs, err := s.gmailClient.GetMessagesParallel(ids, maxWorkers)
	if err != nil {
		return nil, err
	}
	return plainTextsByID(msgs), nil
}
```

Ensure `email_service.go` imports `gmail_v1 "google.golang.org/api/gmail/v1"` (add it to the import block if absent; the file already imports `internal/gmail` as `gmail`).

- [ ] **Step 5: Run tests + build**

Run: `go build ./... && go test ./internal/services/ -run TestPlainTextsByID -v`
Expected: build success, test PASS. (If a generated `MockEmailService` exists and fails to build, regenerate with `make test-mocks`.)

- [ ] **Step 6: Commit**

```bash
git add internal/services/interfaces.go internal/services/email_service.go internal/services/email_service_test.go
git commit -m "feat(services): EmailService.GetMessagePlainTexts bulk body fetch"
```

---

### Task 6: TUI wiring — cap, fetch bodies, populate `Body`

**Files:**
- Modify: `internal/tui/action_plan.go` (around lines 274-300, inside `openActionPlanWithText`)

No unit test (TUI orchestration over a live Gmail client; covered by the service/helper tests + manual E2E).

- [ ] **Step 1: Add body fetch before analysis**

In `internal/tui/action_plan.go`, the block currently reads (lines ~274-284):

```go
	batchSize := a.Config.InboxAnalyzer.BatchSize
	maxBatches := a.Config.InboxAnalyzer.MaxBatches

	var userRules []string
	if svc := a.GetAnalyzerRulesService(); svc != nil {
		if rs, err := svc.ListRules(a.ctx); err == nil {
			for _, r := range rs {
				userRules = append(userRules, r.RuleText)
			}
		}
	}
```

Replace it with:

```go
	batchSize := a.Config.InboxAnalyzer.BatchSize
	maxBatches := a.Config.InboxAnalyzer.MaxBatches
	bodyCharLimit := a.Config.InboxAnalyzer.BodyCharLimit

	var userRules []string
	if svc := a.GetAnalyzerRulesService(); svc != nil {
		if rs, err := svc.ListRules(a.ctx); err == nil {
			for _, r := range rs {
				userRules = append(userRules, r.RuleText)
			}
		}
	}

	// Enrich the analyzer context with each email's plain-text body (opt-in). Cap to what is
	// actually analyzed (BatchSize x MaxBatches) so we never fetch bodies for messages the
	// analyzer would drop. Failures degrade gracefully to the snippet (Body left empty).
	if a.Config.InboxAnalyzer.IncludeBody {
		bs, mb := batchSize, maxBatches
		if bs <= 0 {
			bs = 50
		}
		if mb <= 0 {
			mb = 10
		}
		if cap := bs * mb; len(messages) > cap {
			messages = messages[:cap]
		}
		ids := make([]string, len(messages))
		for i := range messages {
			ids[i] = messages[i].ID
		}
		emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
		if emailService != nil {
			go a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Fetching email bodies for %d messages…", len(ids)))
			if bodies, err := emailService.GetMessagePlainTexts(a.ctx, ids, 0); err == nil {
				for i := range messages {
					if body := bodies[messages[i].ID]; body != "" {
						messages[i].Body = body
					}
				}
			}
		}
	}
```

- [ ] **Step 2: Pass `BodyCharLimit` in the analyzer options**

In `internal/tui/action_plan.go`, change the `Analyze` options (line ~300) from:

```go
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules},
```

to:

```go
			services.InboxAnalyzerOptions{BatchSize: batchSize, MaxBatches: maxBatches, CustomPromptText: customPromptText, UserRules: userRules, BodyCharLimit: bodyCharLimit},
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success. (`fmt` is already imported in `action_plan.go`.)

- [ ] **Step 4: Commit**

```bash
git add internal/tui/action_plan.go
git commit -m "feat(action-plan): fetch email bodies to enrich analyzer context (opt-in)"
```

---

### Task 7: Docs + full verification

**Files:**
- Modify: any analyzer/config doc that lists `inbox_analyzer` keys (e.g. `docs/` config reference, example `config.json`)

- [ ] **Step 1: Document the new config keys**

Find where `inbox_analyzer` config is documented:

Run: `grep -rln "batch_size\|inbox_analyzer\|max_batches" docs/ config/ *.json 2>/dev/null`

In each hit that lists the analyzer keys, add `include_body` (default `true`) and `body_char_limit` (default `1000`) with a one-line description: "include the first N chars of each email's plain-text body in the analyzer context; set `include_body` to false for slow local models or very large inboxes." If no such doc exists, skip this step (the JSON tags + defaults are self-documenting).

- [ ] **Step 2: Run the project pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt clean, vet clean, golangci-lint clean, essential tests pass.

- [ ] **Step 3: Run the analyzer + services + config tests explicitly**

Run: `go test ./internal/services/ ./internal/config/ ./internal/tui/ 2>&1 | tail -20`
Expected: all `ok`.

- [ ] **Step 4: Commit any doc changes**

```bash
git add -A
git commit -m "docs: document analyzer include_body + body_char_limit"
```

(Skip if no doc changes were needed.)

---

## Self-review notes

- **Spec coverage:** config opt-in + limit (Task 1), `AnalyzerMessage.Body` (Task 2), `truncateForAnalyzer` (Task 3), payload body/snippet + limit threading (Task 4), `GetMessagePlainTexts` bulk fetch (Task 5), TUI cap+fetch+populate+progress+graceful fallback (Task 6), docs (Task 7), tests (Tasks 1,3,4,5). All spec sections mapped.
- **Type consistency:** `truncateForAnalyzer(text string, limit int) string`, `buildBatchPayload(batch []AnalyzerMessage, bodyCharLimit int) string`, `plainTextsByID([]*gmail_v1.Message) map[string]string`, `GetMessagePlainTexts(ctx, ids []string, maxWorkers int) (map[string]string, error)`, config `IncludeBody`/`BodyCharLimit`, options `BodyCharLimit` — names match across tasks and call sites.
- **GetServices order:** the 12-value tuple returns `EmailService` first (per CLAUDE.md), so `emailService, _, _, _, _, _, _, _, _, _, _, _ := a.GetServices()` is correct.
- **No placeholders:** every code step shows full code; commands have expected output.
