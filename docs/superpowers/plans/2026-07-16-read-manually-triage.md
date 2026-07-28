# "Read manually" Assisted Triage — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the Action Plan's "Read manually" bucket into a sender-grouped, on-demand AI-assisted triage surface (hint + suggested action per email, accept per email or per sender group, plus apply-one-action-to-a-group).

**Architecture:** New service method `AssistReadManually` (on-demand, no auto-run) returns one `ReadManuallySuggestion` per read-manually message. The TUI groups `plan.ReadManually` by normalized sender into a third tree level, renders AI annotations inline when present, and reuses the existing bulk executor (`runActionPlanBulkOp`), the move/action chooser, and the two-press status-bar confirmation.

**Tech Stack:** Go, tview, existing `InboxAnalyzerService`/`AIService`, mockery mocks, `net/mail` for address normalization.

**Branch:** `feat/read-manually-triage`. **Spec:** `docs/superpowers/specs/2026-07-16-action-plan-read-manually-triage-design.md`.

**Conventions (binding):**
- No `Co-Authored-By`/Claude signature in commits.
- Scoped tests only: `go test ./internal/services/ ./internal/tui/ -count=1`. Never `go test ./...`.
- `gofmt -w` touched files before every commit; run `make pre-commit-check` before declaring a task done.
- ErrorHandler from the event loop (key handlers) MUST be `go`-wrapped; inside a worker goroutine call it directly. Never `QueueUpdateDraw` in streaming callbacks or ESC paths.
- Business logic in `internal/services`; UI only in `internal/tui`.

---

## File Structure

- **Create** `internal/services/inbox_analyzer_assist.go` — `AssistReadManually` impl + `parseAssistResponse` + prompt embed. (Keeps `inbox_analyzer_service.go` from growing.)
- **Create** `internal/services/inbox_analyzer_assist_prompt.txt` — default assist prompt.
- **Create** `internal/services/inbox_analyzer_assist_test.go` — parse + strict-label + degrade tests.
- **Modify** `internal/services/interfaces.go` — `ReadManuallySuggestion` type + interface method.
- **Regen** `internal/services/mocks/inbox_analyzer_service.go` — via `make test-mocks`.
- **Create** `internal/tui/action_plan_read_manually.go` — `groupReadManuallyBySender`, `senderExpandKey`, render helpers, assist + accept handlers.
- **Create** `internal/tui/action_plan_read_manually_test.go` — grouping + accept-removal pure tests.
- **Modify** `internal/tui/action_plan.go` — build the 3-level read-manually subtree; wire the two new keys in the input capture.
- **Modify** `internal/tui/action_plan_apply.go` — add `rmSuggestions map[string]services.ReadManuallySuggestion` to `actionPlanState`.
- **Modify** `internal/config/config.go` — `AssistReadManually`, `AcceptSuggestion` key fields + defaults + migration.
- **Modify** `internal/tui/app.go` — `:help` lines for the two keys.
- **Modify** `docs/KEYBOARD_SHORTCUTS.md` — document the two keys.

---

## Task 1: Service — `ReadManuallySuggestion` type + `parseAssistResponse` (pure)

**Files:**
- Modify: `internal/services/interfaces.go`
- Create: `internal/services/inbox_analyzer_assist.go`
- Test: `internal/services/inbox_analyzer_assist_test.go`

- [ ] **Step 1: Add the type + interface method** in `internal/services/interfaces.go`. Put the struct just above the `InboxAnalyzerService` interface, and add the method inside it:

```go
// ReadManuallySuggestion is the AI's per-email assist result for a read-manually message.
type ReadManuallySuggestion struct {
	ID     string // AnalyzerMessage.ID
	Hint   string // one short line: what it is / why it's here
	Action string // "archive" | "mark_read" | "trash" | "label" | "read"  ("read" = no action)
	Label  string // set only when Action == "label"
}
```

Inside `type InboxAnalyzerService interface { ... }` add:

```go
	// AssistReadManually enriches read-manually messages on demand: one suggestion per input
	// message, in input order. Honors opts.StrictLabels/AvailableLabels (a label suggestion not
	// in AvailableLabels degrades to "read"). Never applies anything — suggestions only.
	AssistReadManually(ctx context.Context, msgs []AnalyzerMessage, opts InboxAnalyzerOptions) ([]ReadManuallySuggestion, error)
```

- [ ] **Step 2: Write the failing test** `internal/services/inbox_analyzer_assist_test.go`. `parseAssistResponse(raw string, batchIDs []string, available map[string]string, strict bool) []ReadManuallySuggestion` — `available` maps lowercased label name → canonical name; missing IDs and invalid actions reconcile to `{Action:"read"}`; a `label` action whose name is not in `available` (when `strict`) degrades to `"read"`.

```go
package services

import (
	"reflect"
	"testing"
)

func TestParseAssistResponse(t *testing.T) {
	batch := []string{"a", "b", "c", "d"}
	available := map[string]string{"work": "Work"}
	raw := `[
	  {"id":"a","hint":"promo","action":"archive"},
	  {"id":"b","hint":"HR notice","action":"label","label":"work"},
	  {"id":"c","hint":"unknown label","action":"label","label":"Nope"},
	  {"id":"d","hint":"just read","action":"weird"}
	]`
	got := parseAssistResponse(raw, batch, available, true)
	want := []ReadManuallySuggestion{
		{ID: "a", Hint: "promo", Action: "archive"},
		{ID: "b", Hint: "HR notice", Action: "label", Label: "Work"},
		{ID: "c", Hint: "unknown label", Action: "read"}, // strict: unknown label -> read
		{ID: "d", Hint: "just read", Action: "read"},      // invalid action -> read
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseAssistResponse_MissingIDReconciled(t *testing.T) {
	batch := []string{"a", "b"}
	raw := `[{"id":"a","hint":"x","action":"trash"}]` // b omitted by the model
	got := parseAssistResponse(raw, batch, map[string]string{}, false)
	if len(got) != 2 || got[1].ID != "b" || got[1].Action != "read" {
		t.Fatalf("missing id not reconciled to read: %+v", got)
	}
}

func TestParseAssistResponse_NonStrictKeepsUnknownLabelAsRead(t *testing.T) {
	// Even when not strict, we cannot apply a label we can't resolve to an ID, so it stays "read".
	got := parseAssistResponse(`[{"id":"a","action":"label","label":"Ghost"}]`, []string{"a"}, map[string]string{}, false)
	if got[0].Action != "read" {
		t.Fatalf("unknown label should degrade to read, got %q", got[0].Action)
	}
}
```

- [ ] **Step 3: Run the test — expect FAIL** (undefined `parseAssistResponse`):

Run: `go test ./internal/services/ -run TestParseAssistResponse -count=1`
Expected: build failure `undefined: parseAssistResponse`.

- [ ] **Step 4: Implement `parseAssistResponse`** in `internal/services/inbox_analyzer_assist.go` (also add the file header + imports; the `AssistReadManually` method comes in Task 2):

```go
package services

import (
	"context"
	"encoding/json"
	"strings"

	_ "embed"
)

//go:embed inbox_analyzer_assist_prompt.txt
var assistReadManuallyPrompt string

type assistItem struct {
	ID     string `json:"id"`
	Hint   string `json:"hint"`
	Action string `json:"action"`
	Label  string `json:"label"`
}

var validAssistActions = map[string]bool{
	"archive": true, "mark_read": true, "trash": true, "label": true, "read": true,
}

// parseAssistResponse turns the model's JSON array into one suggestion per batch ID, in
// batch order. Unknown IDs from the model are ignored; batch IDs the model omitted are
// reconciled to {Action:"read"}. An invalid action, or a "label" whose name can't be
// resolved to an existing label (always required — we need the canonical name to apply it),
// degrades to "read".
func parseAssistResponse(raw string, batchIDs []string, available map[string]string, strict bool) []ReadManuallySuggestion {
	byID := map[string]assistItem{}
	var items []assistItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(extractJSONArray(raw))), &items); err == nil {
		for _, it := range items {
			byID[it.ID] = it
		}
	}
	out := make([]ReadManuallySuggestion, 0, len(batchIDs))
	for _, id := range batchIDs {
		s := ReadManuallySuggestion{ID: id, Action: "read"}
		if it, ok := byID[id]; ok {
			s.Hint = strings.TrimSpace(it.Hint)
			act := strings.ToLower(strings.TrimSpace(it.Action))
			switch {
			case act == "label":
				if canon, ok := available[strings.ToLower(strings.TrimSpace(it.Label))]; ok {
					s.Action, s.Label = "label", canon
				} // else stays "read" (strict param reserved for future non-strict create path)
			case validAssistActions[act] && act != "read":
				s.Action = act
			}
		}
		out = append(out, s)
	}
	return out
}
```

Reuse the analyzer's existing JSON-extraction helper. Check `inbox_analyzer_service.go` for the exact name (it strips code fences / prose around the array). If it is named differently than `extractJSONArray`, use that name here. If no such helper exists, add a tiny one in this file:

```go
// extractJSONArray returns the substring from the first '[' to the last ']' inclusive, or raw.
func extractJSONArray(raw string) string {
	i, j := strings.Index(raw, "["), strings.LastIndex(raw, "]")
	if i >= 0 && j > i {
		return raw[i : j+1]
	}
	return raw
}
```

- [ ] **Step 5: Run the tests — expect PASS**:

Run: `go test ./internal/services/ -run TestParseAssistResponse -count=1`
Expected: PASS (3 tests).

- [ ] **Step 6: Commit**:

```bash
gofmt -w internal/services/interfaces.go internal/services/inbox_analyzer_assist.go internal/services/inbox_analyzer_assist_test.go
git add internal/services/interfaces.go internal/services/inbox_analyzer_assist.go internal/services/inbox_analyzer_assist_test.go
git commit -m "feat(services): ReadManuallySuggestion + parseAssistResponse"
```

---

## Task 2: Service — `AssistReadManually` method + default prompt + mock

**Files:**
- Create: `internal/services/inbox_analyzer_assist_prompt.txt`
- Modify: `internal/services/inbox_analyzer_assist.go`
- Test: `internal/services/inbox_analyzer_assist_test.go`
- Regen: `internal/services/mocks/inbox_analyzer_service.go`

- [ ] **Step 1: Create the default prompt** `internal/services/inbox_analyzer_assist_prompt.txt`. It must instruct a JSON-array reply, one object per email, fields `id,hint,action,label`; `action` ∈ archive|mark_read|trash|label|read; `hint` ≤ ~8 words; use only labels from the provided list; `read` when unsure. Mirror the tone/structure of `inbox_analyzer_prompt.txt` (read it first). Include the `{{messages}}` marker and a `{{labels}}` marker for the available-labels list.

- [ ] **Step 2: Write the failing test** for `AssistReadManually` using a hand-rolled fake `AIService`. First read `internal/services/inbox_analyzer_service_test.go` to copy the exact fake-AIService shape used there (it stubs `ApplyCustomPromptStream`). Then add:

```go
func TestAssistReadManually_EndToEnd(t *testing.T) {
	fake := &fakeAIService{ // reuse the test's existing fake type; set its canned response
		customResp: `[{"id":"m1","hint":"promo","action":"archive"},{"id":"m2","hint":"read me","action":"read"}]`,
	}
	svc := NewInboxAnalyzerService(fake)
	msgs := []AnalyzerMessage{{ID: "m1", From: "a@x.com", Subject: "S1"}, {ID: "m2", From: "b@x.com", Subject: "S2"}}
	got, err := svc.AssistReadManually(context.Background(), msgs, InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10})
	if err != nil {
		t.Fatalf("assist: %v", err)
	}
	if len(got) != 2 || got[0].Action != "archive" || got[1].Action != "read" {
		t.Fatalf("unexpected suggestions: %+v", got)
	}
}

func TestAssistReadManually_NoAIService(t *testing.T) {
	svc := NewInboxAnalyzerService(nil)
	if _, err := svc.AssistReadManually(context.Background(), []AnalyzerMessage{{ID: "x"}}, InboxAnalyzerOptions{}); err == nil {
		t.Fatal("expected error when AI service is nil")
	}
}
```

(If the existing fake type/field names differ, adapt these two lines — do not invent a new fake.)

- [ ] **Step 3: Run — expect FAIL** (`undefined: (*InboxAnalyzerServiceImpl).AssistReadManually`):

Run: `go test ./internal/services/ -run TestAssistReadManually -count=1`
Expected: build failure.

- [ ] **Step 4: Implement `AssistReadManually`** in `internal/services/inbox_analyzer_assist.go`. Mirror `Analyze`'s batching + `ApplyCustomPromptStream` call (read `inbox_analyzer_service.go:432-513` for the exact call shape and defaults). Skeleton:

```go
func (s *InboxAnalyzerServiceImpl) AssistReadManually(ctx context.Context, msgs []AnalyzerMessage, opts InboxAnalyzerOptions) ([]ReadManuallySuggestion, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if len(msgs) == 0 {
		return nil, nil
	}
	available := map[string]string{}
	for _, name := range opts.AvailableLabels {
		available[strings.ToLower(strings.TrimSpace(name))] = name
	}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}
	out := make([]ReadManuallySuggestion, 0, len(msgs))
	for start := 0; start < len(msgs); start += batchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := start + batchSize
		if end > len(msgs) {
			end = len(msgs)
		}
		batch := msgs[start:end]
		batchIDs := make([]string, len(batch))
		for i, m := range batch {
			batchIDs[i] = m.ID
		}
		prompt := buildAssistPrompt(batch, opts) // fills {{messages}} + {{labels}}; see below
		raw, err := s.aiService.ApplyCustomPromptStream(ctx, prompt, nil, nil)
		if err != nil {
			// Degrade this batch to "read" rather than failing the whole pass.
			for _, id := range batchIDs {
				out = append(out, ReadManuallySuggestion{ID: id, Action: "read"})
			}
			continue
		}
		out = append(out, parseAssistResponse(raw, batchIDs, available, opts.StrictLabels)...)
	}
	return out, nil
}
```

Add `buildAssistPrompt(batch []AnalyzerMessage, opts InboxAnalyzerOptions) string` in the same file: start from `assistReadManuallyPrompt` (or `opts.CustomPromptText` if you decide to honor it — keep parity with Analyze), replace `{{labels}}` with the comma-joined `opts.AvailableLabels`, and `{{messages}}` with the rendered batch. Reuse the analyzer's batch-rendering helper if one exists (grep `inbox_analyzer_service.go` for how it renders id/subject/from/body under `{{messages}}`); otherwise render `- id: <id> | from: <From> | subject: <Subject> | <Snippet-or-Body-trimmed>` per line. Add the imports (`fmt`, `strings`, `context` already there).

- [ ] **Step 5: Run — expect PASS**:

Run: `go test ./internal/services/ -run TestAssistReadManually -count=1`
Expected: PASS.

- [ ] **Step 6: Regenerate mocks and build**:

Run: `make test-mocks && go build ./...`
Expected: `internal/services/mocks/inbox_analyzer_service.go` now has `AssistReadManually`; build OK.

- [ ] **Step 7: Commit**:

```bash
gofmt -w internal/services/inbox_analyzer_assist.go internal/services/inbox_analyzer_assist_test.go
git add internal/services/inbox_analyzer_assist.go internal/services/inbox_analyzer_assist_prompt.txt internal/services/inbox_analyzer_assist_test.go internal/services/mocks/inbox_analyzer_service.go
git commit -m "feat(services): AssistReadManually on-demand read-manually enrichment"
```

---

## Task 3: TUI — sender grouping (pure)

**Files:**
- Create: `internal/tui/action_plan_read_manually.go`
- Test: `internal/tui/action_plan_read_manually_test.go`

- [ ] **Step 1: Write the failing test** `internal/tui/action_plan_read_manually_test.go`:

```go
package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestGroupReadManuallyBySender(t *testing.T) {
	msgs := []services.AnalyzerMessage{
		{ID: "1", From: "Ana García <ana@x.com>"},
		{ID: "2", From: "news@acme.com"},
		{ID: "3", From: "ana@x.com"},           // same address as #1, different display
		{ID: "4", From: "news@acme.com"},
		{ID: "5", From: "news@acme.com"},
	}
	groups := groupReadManuallyBySender(msgs)
	// Largest group first: acme (3), then ana (2).
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(groups))
	}
	if groups[0].senderKey != "news@acme.com" || len(groups[0].msgs) != 3 {
		t.Fatalf("group0 = %+v", groups[0])
	}
	if groups[1].senderKey != "ana@x.com" || len(groups[1].msgs) != 2 {
		t.Fatalf("group1 = %+v", groups[1])
	}
	// Within a group, plan order is preserved.
	if groups[1].msgs[0].ID != "1" || groups[1].msgs[1].ID != "3" {
		t.Fatalf("within-group order not preserved: %+v", groups[1].msgs)
	}
}

func TestSenderExpandKey(t *testing.T) {
	if got := senderExpandKey("news@acme.com"); got != "\x00read-manually:news@acme.com" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (undefined functions):

Run: `go test ./internal/tui/ -run 'TestGroupReadManuallyBySender|TestSenderExpandKey' -count=1`
Expected: build failure.

- [ ] **Step 3: Implement** in `internal/tui/action_plan_read_manually.go`:

```go
package tui

import (
	"net/mail"
	"sort"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

type readManuallyGroup struct {
	senderKey  string // normalized address, lowercased
	senderDisp string // first-seen raw From for display
	msgs       []services.AnalyzerMessage
}

// normalizeSender extracts a lowercased email address from a From header, falling back to
// the trimmed/lowercased raw value when it doesn't parse.
func normalizeSender(from string) string {
	if addr, err := mail.ParseAddress(strings.TrimSpace(from)); err == nil {
		return strings.ToLower(strings.TrimSpace(addr.Address))
	}
	return strings.ToLower(strings.TrimSpace(from))
}

// groupReadManuallyBySender groups messages by normalized sender, ordered by descending
// group size then senderKey; within a group, input order is preserved.
func groupReadManuallyBySender(msgs []services.AnalyzerMessage) []readManuallyGroup {
	idx := map[string]int{}
	var groups []readManuallyGroup
	for _, m := range msgs {
		key := normalizeSender(m.From)
		if i, ok := idx[key]; ok {
			groups[i].msgs = append(groups[i].msgs, m)
			continue
		}
		idx[key] = len(groups)
		groups = append(groups, readManuallyGroup{senderKey: key, senderDisp: strings.TrimSpace(m.From), msgs: []services.AnalyzerMessage{m}})
	}
	sort.SliceStable(groups, func(a, b int) bool {
		if len(groups[a].msgs) != len(groups[b].msgs) {
			return len(groups[a].msgs) > len(groups[b].msgs)
		}
		return groups[a].senderKey < groups[b].senderKey
	})
	return groups
}

// senderExpandKey is the state.expanded map key for a sender group under read-manually.
func senderExpandKey(senderKey string) string {
	return "\x00read-manually:" + senderKey
}
```

- [ ] **Step 4: Run — expect PASS**:

Run: `go test ./internal/tui/ -run 'TestGroupReadManuallyBySender|TestSenderExpandKey' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**:

```bash
gofmt -w internal/tui/action_plan_read_manually.go internal/tui/action_plan_read_manually_test.go
git add internal/tui/action_plan_read_manually.go internal/tui/action_plan_read_manually_test.go
git commit -m "feat(tui): group read-manually messages by sender"
```

---

## Task 4: TUI — suggestions state + render

**Files:**
- Modify: `internal/tui/action_plan_apply.go` (add field to `actionPlanState`)
- Modify: `internal/tui/action_plan.go` (render the 3-level subtree)
- Modify: `internal/tui/action_plan_read_manually.go` (render helpers)

- [ ] **Step 1: Add state field.** Read the `actionPlanState` struct definition (grep `type actionPlanState struct` in `internal/tui/action_plan_apply.go`) and add:

```go
	rmSuggestions map[string]services.ReadManuallySuggestion // read-manually assist results, keyed by msg ID; nil until requested
```

- [ ] **Step 2: Add a render helper** in `internal/tui/action_plan_read_manually.go`:

```go
// readManuallyLeafLabel renders one email leaf, appending the AI hint/suggestion when present.
func readManuallyLeafLabel(m services.AnalyzerMessage, sug services.ReadManuallySuggestion, hasSug bool) string {
	subject := strings.TrimSpace(m.Subject)
	if subject == "" {
		subject = "(no subject)"
	}
	if !hasSug || sug.Hint == "" && sug.Action == "read" {
		return subject
	}
	if sug.Action == "read" {
		return subject + " — 💡 " + sug.Hint
	}
	verb := actionVerbLabel(sug.Action) // existing helper in action_plan.go; for "label" append the name
	if sug.Action == "label" && sug.Label != "" {
		verb = verb + " " + sug.Label
	}
	return subject + " — 💡 " + sug.Hint + " · sugiere: " + verb
}
```

Confirm `actionVerbLabel` exists and its signature (grep in `internal/tui/action_plan.go`); if it takes different args, adapt.

- [ ] **Step 3: Rebuild the read-manually subtree** in `internal/tui/action_plan.go`. Find where the read-manually pseudo-node's children are added (around the flat loop over `state.plan.ReadManually`, ~lines 627-660). Replace the flat children with two levels: for each `g := range groupReadManuallyBySender(state.plan.ReadManually)`, add a sender node labeled `chevron + " " + g.senderDisp + " · " + len(g.msgs)` whose expand state is `state.expanded[senderExpandKey(g.senderKey)]`; when expanded, add each message as a leaf using `readManuallyLeafLabel(m, state.rmSuggestions[m.ID], state.rmSuggestions != nil)`. Give the sender node tview reference `senderExpandKey(g.senderKey)` and each leaf reference `"\x00rm-msg:" + m.ID` so the input handler can classify selection. Keep categories untouched.

- [ ] **Step 4: Build + smoke-compile**:

Run: `go build ./... && go test ./internal/tui/ -count=1`
Expected: BUILD OK, existing tests pass (no behavior asserted yet for the new tree).

- [ ] **Step 5: Commit**:

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_apply.go internal/tui/action_plan_read_manually.go
git add internal/tui/action_plan.go internal/tui/action_plan_apply.go internal/tui/action_plan_read_manually.go
git commit -m "feat(tui): render read-manually as sender groups with optional AI annotations"
```

---

## Task 5: TUI — assist key (on-demand AI pass)

**Files:**
- Modify: `internal/config/config.go` (key field + default + migration)
- Modify: `internal/tui/action_plan.go` (input capture)
- Modify: `internal/tui/action_plan_read_manually.go` (handler)

- [ ] **Step 1: Add the key binding.** In `internal/config/config.go` add to the `KeyBindings` struct: `AssistReadManually string ` + json tag `assist_read_manually`. Read how a recent key (e.g. `RememberRule`) is added in the struct, in `DefaultConfig()`, and in the migration list, and mirror all three. Default value: `"g"` (verify it's free in the action-plan panel context — the plan panel currently uses a, space, l, m, y, t, v, Enter, Ctrl+R, ConfirmPlan; `g` is free). Add a `_comment` if the struct uses them.

- [ ] **Step 2: Implement the handler** in `internal/tui/action_plan_read_manually.go`:

```go
// assistReadManually runs the on-demand AI pass over the current read-manually bucket and
// re-renders with hints/suggested actions. Called from the event loop; does its work in a
// worker goroutine.
func (a *App) assistReadManually(state *actionPlanState) {
	if state == nil || state.plan == nil || len(state.plan.ReadManually) == 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing in Read manually to assist")
		return
	}
	msgs := append([]services.AnalyzerMessage(nil), state.plan.ReadManually...)
	opts := a.analyzerOptions() // reuse the same options builder Analyze uses (AvailableLabels, StrictLabels…)
	analyzer := a.GetInboxAnalyzerService()
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Assisting %d email(s)…", len(msgs)))
		sug, err := analyzer.AssistReadManually(a.ctx, msgs, opts)
		a.GetErrorHandler().ClearProgress()
		if err != nil {
			a.GetErrorHandler().ShowWarning(a.ctx, "Could not get AI suggestions — showing the list only")
			return
		}
		m := make(map[string]services.ReadManuallySuggestion, len(sug))
		for _, s := range sug {
			m[s.ID] = s
		}
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state {
				return // panel changed/closed
			}
			state.rmSuggestions = m
			a.rebuildActionPlanTree(state)
		})
		a.GetErrorHandler().ShowSuccess(a.ctx, "AI suggestions ready")
	}()
}
```

Grep for the exact names of `a.analyzerOptions()` / the options builder used by the existing analyze flow and `a.GetInboxAnalyzerService()`; if they differ, use the real ones. Confirm `rebuildActionPlanTree(state)` is the tree refresh entry point (seen at `action_plan.go:523`).

- [ ] **Step 3: Wire the key** in the action-plan input capture in `internal/tui/action_plan.go` (find the `SetInputCapture`/key switch for the plan panel). Add, alongside the other `a.matchesConfiguredKey(...)` cases:

```go
		case a.matchesConfiguredKey(ev, a.Keys.AssistReadManually):
			a.assistReadManually(a.actionPlanState)
			return nil
```

- [ ] **Step 4: Build + test**:

Run: `go build ./... && go test ./internal/tui/ ./internal/config/ -count=1`
Expected: BUILD OK, tests pass.

- [ ] **Step 5: Commit**:

```bash
gofmt -w internal/config/config.go internal/tui/action_plan.go internal/tui/action_plan_read_manually.go
git add internal/config/config.go internal/tui/action_plan.go internal/tui/action_plan_read_manually.go
git commit -m "feat(tui): on-demand AI assist key for read-manually"
```

---

## Task 6: TUI — accept key (context-sensitive) + group action reuse

**Files:**
- Modify: `internal/config/config.go` (second key)
- Modify: `internal/tui/action_plan.go` (input capture)
- Modify: `internal/tui/action_plan_read_manually.go` (handlers)
- Test: `internal/tui/action_plan_read_manually_test.go`

- [ ] **Step 1: Add the second key** `AcceptSuggestion string` json `accept_suggestion`, default `"."` (free in the panel), same three touchpoints as Task 5 Step 1.

- [ ] **Step 2: Write the failing pure test** for the apply-plan-side effect helper. Add to `action_plan_read_manually_test.go`:

```go
func TestApplyRMSuggestionsRemovesFromBucket(t *testing.T) {
	plan := &services.ActionPlan{ReadManually: []services.AnalyzerMessage{
		{ID: "1", From: "a@x"}, {ID: "2", From: "a@x"}, {ID: "3", From: "b@y"},
	}}
	// Applying ids {1,3} must leave only id 2 in the bucket.
	dropReadManually(plan, []string{"1", "3"})
	if len(plan.ReadManually) != 1 || plan.ReadManually[0].ID != "2" {
		t.Fatalf("bucket after drop = %+v", plan.ReadManually)
	}
}
```

- [ ] **Step 3: Run — expect FAIL** (`undefined: dropReadManually`).

Run: `go test ./internal/tui/ -run TestApplyRMSuggestionsRemovesFromBucket -count=1`
Expected: build failure.

- [ ] **Step 4: Implement** the pure drop helper + the accept handlers in `action_plan_read_manually.go`:

```go
// dropReadManually removes every id in ids from plan.ReadManually (reuses removeReadManuallyByID).
func dropReadManually(plan *services.ActionPlan, ids []string) {
	for _, id := range ids {
		plan.ReadManually = removeReadManuallyByID(plan.ReadManually, id)
	}
}

// acceptReadManuallySuggestions applies each message's suggested action (skipping "read"),
// grouped per action for one bulk call each, then drops the applied ids from the bucket.
// ids is the set to consider (one email, or a whole sender group).
func (a *App) acceptReadManuallySuggestions(state *actionPlanState, ids []string) {
	if state.rmSuggestions == nil {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Press the assist key first to get suggestions")
		return
	}
	// Bucket ids by action (and label) so each action is one bulk op.
	type key struct{ action, label string }
	buckets := map[key][]string{}
	for _, id := range ids {
		s, ok := state.rmSuggestions[id]
		if !ok || s.Action == "read" {
			continue
		}
		k := key{s.Action, s.Label}
		buckets[k] = append(buckets[k], id)
	}
	if len(buckets) == 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, "No actionable suggestions here")
		return
	}
	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
	go func() {
		var applied []string
		for k, kids := range buckets {
			if err := a.runActionPlanBulkOp(emailService, labelService, k.action, kids, k.label); err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("%s failed: %v", k.action, err))
				continue
			}
			applied = append(applied, kids...)
		}
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state {
				return
			}
			dropReadManually(state.plan, applied)
			a.rebuildActionPlanTree(state)
		})
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied %d suggestion(s)", len(applied)))
	}()
}
```

- [ ] **Step 5: Wire the accept key context-sensitively** in `action_plan.go`. In the input-capture handler, when the accept key matches, inspect the currently-selected tree node's reference:
  - reference `== "\x00rm-msg:"+id` (a read-manually leaf) → `a.acceptReadManuallySuggestions(state, []string{id})`.
  - reference `== senderExpandKey(key)` (a sender header) → collect that group's ids from `groupReadManuallyBySender(state.plan.ReadManually)` and call `acceptReadManuallySuggestions(state, groupIDs)`. Wrap group-level (multi-id) accepts in the two-press confirmation (mirror `startActionPlanConfirm`/`confirmPending`): first press arms + persistent status "Accept N suggestion(s) from <sender>? press again…", second press applies. Single-leaf accept is direct.
  - otherwise (a category/other node) → `go a.GetErrorHandler().ShowInfo(a.ctx, "Accept works on Read manually items")`.

  Read `getSelectedNodeRef`/how the panel reads the current tree selection (grep `GetReference` in `action_plan.go`) to get the reference string.

- [ ] **Step 6: "Apply one action to the group" — reuse the move/action chooser.** Find the handler for `a.Keys.Move` in the plan input capture. Today on a category/read-manually header it moves the whole group. Confirm that when invoked on a sender header it targets that sender's ids (build ids via `groupReadManuallyBySender`); if the existing chooser only understands categories and the flat read-manually node, extend its id-collection to accept a sender group's ids. The chooser already offers archive/label/trash/keep and runs through `runActionPlanBulkOp`, so no new action code — only the id set changes. Drop applied ids via `dropReadManually` on completion.

- [ ] **Step 7: Run all + build**:

Run: `go build ./... && go test ./internal/tui/ ./internal/config/ -count=1`
Expected: BUILD OK, tests pass.

- [ ] **Step 8: Commit**:

```bash
gofmt -w internal/config/config.go internal/tui/action_plan.go internal/tui/action_plan_read_manually.go internal/tui/action_plan_read_manually_test.go
git add internal/config/config.go internal/tui/action_plan.go internal/tui/action_plan_read_manually.go internal/tui/action_plan_read_manually_test.go
git commit -m "feat(tui): accept suggestions (email/group) + group action for read-manually"
```

---

## Task 7: Config migration + `:help` + docs

**Files:**
- Modify: `internal/config/config.go` (ensure migration surfaces both keys)
- Modify: `internal/tui/app.go` (`:help` lines)
- Modify: `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Verify migration.** Confirm both `assist_read_manually` and `accept_suggestion` appear in whatever list `MigrateConfigFile`/`deepMergeMissing` uses so existing users' `config.json` gains them (grep the migration path used for prior key additions in `config.go`). Add them if the migration is key-list-driven.

- [ ] **Step 2: Write a config test** confirming defaults load. In `internal/config/config_test.go` (match existing test style) assert `DefaultConfig().Keys.AssistReadManually == "g"` and `AcceptSuggestion == "."`, and that loading a config missing both keeps the defaults.

- [ ] **Step 3: Run:** `go test ./internal/config/ -count=1` — Expected: PASS.

- [ ] **Step 4: Add `:help` lines** in `internal/tui/app.go` (find the Action Plan help block; grep `Fprintf.*action` / where plan keys are documented). Add two `fmt.Fprintf(&help, ...)` lines describing the assist key ("AI-assist the Read manually bucket") and accept key ("accept the suggested action (email or sender group)"). Match the surrounding format and reference `a.Keys.AssistReadManually` / `a.Keys.AcceptSuggestion` (do not hardcode the letters). Update `help_text_test.go` if it asserts specific lines.

- [ ] **Step 5: Document** in `docs/KEYBOARD_SHORTCUTS.md` under the Action Plan section: the two keys and the one-line behavior.

- [ ] **Step 6: Commit**:

```bash
gofmt -w internal/config/config.go internal/tui/app.go
git add internal/config/config.go internal/config/config_test.go internal/tui/app.go docs/KEYBOARD_SHORTCUTS.md
git commit -m "feat(config,docs): read-manually assist/accept keys, migration, help"
```

---

## Task 8: Full verification

- [ ] **Step 1:** `gofmt` check + `make pre-commit-check` — Expected: all green.
- [ ] **Step 2:** Scoped tests: `go test ./internal/services/ ./internal/tui/ ./internal/config/ -count=1` — Expected: PASS.
- [ ] **Step 3:** Full suite as the pre-merge gate (separate step): `make test` — Expected: exit 0, no leaks/races.
- [ ] **Step 4:** Live smoke on the dev laptop: build, open the Action Plan on the test account, expand "Read manually", press the assist key, confirm hints/suggestions render; accept one email and one sender group; verify the bucket shrinks. (Assist needs a working local/remote LLM — if none, confirm the graceful "showing the list only" warning + sender grouping still works.)
- [ ] **Step 5:** `graphify update .`

---

## Self-Review Notes

- **Spec coverage:** grouping-by-sender (Task 3), on-demand AI hint+action (Tasks 1-2, 5), accept per-email + per-group (Task 6), apply-one-action-to-group via existing chooser (Task 6 Step 6), degrade-without-AI (Task 5 handler warning + grouping independent of AI), strict_labels (Task 1 parse), two-press confirm for group actions (Task 6 Step 5), two configurable keys + migration + help (Tasks 5-7), own prompt (Task 2). All covered.
- **Out of scope preserved:** categories untouched (only the read-manually subtree changes); no auto-run; no "reduce how many fall here".
- **Type consistency:** `ReadManuallySuggestion{ID,Hint,Action,Label}`, `readManuallyGroup{senderKey,senderDisp,msgs}`, `senderExpandKey`, `dropReadManually`, `state.rmSuggestions` used consistently across tasks.
- **Reuse:** `runActionPlanBulkOp`, `removeReadManuallyByID`, `actionVerbLabel`, `rebuildActionPlanTree`, the move/action chooser, and the `confirmPending` two-press pattern — no duplication.
