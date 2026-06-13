# Analyzer "summarize" Action Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `summarize` analyzer action; pressing the summarize key on a category generates a single combined AI digest of its emails, shown in an in-place Action Plan panel (Markdown), Esc to return.

**Architecture:** Prompt gains a `summarize` action; `actionVerbLabel`/`actionRuleVerbShort`/`actionKeyHint` learn it (mapped to the `y` key). A new `dispatchActionPlanSummarize` body-swaps the tree for a TextView, fetches bodies, calls `AIService.GenerateSummary` (non-streaming), and renders the digest via `renderPromptResult` + one `QueueUpdateDraw`.

**Tech Stack:** Go, GizTUI analyzer prompt + Action Plan TUI + AIService.

Spec: `docs/superpowers/specs/2026-06-13-analyzer-summarize-action-design.md`

---

### Task 1: Prompt action + action metadata

**Files:**
- Modify: `internal/services/inbox_analyzer_prompt.txt`
- Modify: `internal/tui/action_plan.go` (`actionVerbLabel`, `actionRuleVerbShort`, `actionKeyHint`)
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_test.go`:

```go
func TestActionVerb_Summarize(t *testing.T) {
	if got := actionVerbLabel("summarize"); got != "summarize" {
		t.Fatalf("actionVerbLabel(summarize)=%q", got)
	}
	if got := actionRuleVerbShort("summarize"); got != "digest" {
		t.Fatalf("actionRuleVerbShort(summarize)=%q", got)
	}
	a := &App{}
	a.Keys.Summarize = "y"
	if got := a.actionKeyHint("summarize"); got != "y" {
		t.Fatalf("actionKeyHint(summarize)=%q, want y", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestActionVerb_Summarize -v`
Expected: FAIL — the three funcs return defaults for "summarize".

- [ ] **Step 3: Add the action to the prompt**

In `internal/services/inbox_analyzer_prompt.txt`, in the action set list (after the `"none"` line), add:

```
  "summarize" — worth a quick AI digest; not full reading, not discarding
```

- [ ] **Step 4: Teach the three metadata functions**

In `internal/tui/action_plan.go`:

`actionVerbLabel` — add a case (before its `default`):
```go
	case "summarize":
		return "summarize"
```

`actionRuleVerbShort` — add a case (before its `default`):
```go
	case "summarize":
		return "digest"
```

`actionKeyHint` — add a case (before its `default`):
```go
	case "summarize":
		return a.Keys.Summarize
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestActionVerb_Summarize -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/inbox_analyzer_prompt.txt internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(analyzer): add 'summarize' action + footer metadata"
```

---

### Task 2: `buildSummarizeInput` pure helper

**Files:**
- Create: `internal/tui/action_plan_summarize.go`
- Test: `internal/tui/action_plan_summarize_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/action_plan_summarize_test.go`:

```go
package tui

import (
	"strings"
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestBuildSummarizeInput(t *testing.T) {
	meta := map[string]*gmailapi.Message{
		"m1": {Id: "m1", Snippet: "snip1", Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
			{Name: "Subject", Value: "Hello"}, {Name: "From", Value: "a@x.com"},
		}}},
		"m2": {Id: "m2", Snippet: "snip2", Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
			{Name: "Subject", Value: "World"}, {Name: "From", Value: "b@x.com"},
		}}},
	}
	bodies := map[string]string{"m1": "full body one"}

	out := buildSummarizeInput([]string{"m1", "m2"}, bodies, meta, 1000)

	if !strings.Contains(out, "Hello") || !strings.Contains(out, "a@x.com") {
		t.Fatalf("missing m1 subject/from:\n%s", out)
	}
	if !strings.Contains(out, "full body one") {
		t.Fatalf("m1 should use its body:\n%s", out)
	}
	if !strings.Contains(out, "snip2") {
		t.Fatalf("m2 has no body → should fall back to snippet:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestBuildSummarizeInput -v`
Expected: FAIL — `buildSummarizeInput` undefined.

- [ ] **Step 3: Implement the helper (in the new file)**

Create `internal/tui/action_plan_summarize.go`:

```go
package tui

import (
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// buildSummarizeInput renders the category's emails into a numbered blob for the AI digest:
// "N. Subject: … | From: …\n   <body up to limit, or snippet if no body>". metaByID supplies
// subject/from/snippet; bodies (id->plain text) supplies the body when available.
func buildSummarizeInput(ids []string, bodies map[string]string, metaByID map[string]*gmailapi.Message, limit int) string {
	var b strings.Builder
	for i, id := range ids {
		subject, from, snippet := "(unknown)", "", ""
		if m := metaByID[id]; m != nil {
			subject = extractHeaderValue(m, "Subject")
			from = extractHeaderValue(m, "From")
			snippet = m.Snippet
		}
		body := bodies[id]
		if strings.TrimSpace(body) == "" {
			body = snippet
		}
		fmt.Fprintf(&b, "%d. Subject: %s | From: %s\n   %s\n", i+1, subject, from, truncateForSummary(body, limit))
	}
	return b.String()
}

// truncateForSummary collapses whitespace and cuts to limit runes (limit <= 0 → no cut).
func truncateForSummary(text string, limit int) string {
	collapsed := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if limit <= 0 || len([]rune(collapsed)) <= limit {
		return collapsed
	}
	return string([]rune(collapsed)[:limit])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestBuildSummarizeInput -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan_summarize.go internal/tui/action_plan_summarize_test.go
git commit -m "feat(action-plan): buildSummarizeInput digest-content helper"
```

---

### Task 3: `dispatchActionPlanSummarize` + key wiring + sentinel

**Files:**
- Modify: `internal/tui/action_plan_summarize.go` (add the dispatch method)
- Modify: `internal/tui/action_plan.go` (`actionPlanInputCapture` — `y` case)
- Modify: `internal/tui/keys.go` (pass-through sentinel)
- Test: `internal/tui/action_plan_summarize_test.go` (swap test + AIService stub)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_summarize_test.go`:

```go
import (
	"context"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

type stubAISummary struct{}

func (stubAISummary) GenerateSummary(ctx context.Context, content string, o services.SummaryOptions) (*services.SummaryResult, error) {
	return &services.SummaryResult{Summary: "DIGEST RESULT"}, nil
}
func (stubAISummary) GenerateSummaryStream(ctx context.Context, content string, o services.SummaryOptions, onToken func(string)) (*services.SummaryResult, error) {
	return &services.SummaryResult{Summary: "DIGEST RESULT"}, nil
}
func (stubAISummary) GenerateReply(ctx context.Context, content string, o services.ReplyOptions) (string, error) {
	return "", nil
}
func (stubAISummary) SuggestLabels(ctx context.Context, content string, l []string) ([]string, error) {
	return nil, nil
}
func (stubAISummary) FormatContent(ctx context.Context, content string, o services.FormatOptions) (string, error) {
	return content, nil
}
func (stubAISummary) ApplyCustomPrompt(ctx context.Context, p string, v map[string]string) (string, error) {
	return "", nil
}
func (stubAISummary) ApplyCustomPromptStream(ctx context.Context, p string, v map[string]string, onToken func(string)) (string, error) {
	return "", nil
}

func TestActionPlanSummarizeSwap(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	a.ctx = context.Background()
	a.aiService = stubAISummary{}
	a.Config = nil // BodyCharLimit read defensively below
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "News", Action: "summarize", MessageIDs: []string{"m1"}},
		}},
		selectedCategory: 0,
		excluded:         map[string]bool{},
		expanded:         map[int]bool{},
		metaByID:         map[string]*gmailapi.Message{"m1": {Id: "m1", Snippet: "s"}},
		footer:           tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.dispatchActionPlanSummarize(state)
	if a.currentFocus != "action_plan_summary" {
		t.Fatalf("expected currentFocus=action_plan_summary, got %q", a.currentFocus)
	}
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out for the summary view")
	}
	view, ok := a.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected the summary TextView focused, got %T", a.GetFocus())
	}
	// Esc restores the tree.
	if cap := view.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.currentFocus != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.currentFocus)
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc the tree should be restored")
	}
}
```

Note: the test sets `a.Config = nil`; the dispatch must read `BodyCharLimit` defensively (see Step 2).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestActionPlanSummarizeSwap -v`
Expected: FAIL — `dispatchActionPlanSummarize` undefined.

- [ ] **Step 3: Implement `dispatchActionPlanSummarize`**

Append to `internal/tui/action_plan_summarize.go` (and add the needed imports: `services`,
`tcell`, `tview`, plus keep `fmt`/`strings`/`gmailapi`):

```go
// dispatchActionPlanSummarize body-swaps the tree for an in-place panel showing a combined AI
// digest of the focused category's checked emails. Esc returns to the tree.
func (a *App) dispatchActionPlanSummarize(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	ids := checkedIDs(cat.MessageIDs, state.excluded)
	if len(ids) == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "All emails in this category are excluded — nothing to summarize")
		return
	}

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(true)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(fmt.Sprintf("⏳ Summarizing %d email(s)…", len(ids)))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state)
	}
	view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(view, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(fmt.Sprintf(" 📝 Digest of %q ", cat.Name))
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.currentFocus = "action_plan_summary"
	a.SetFocus(view)

	limit := 1000
	if a.Config != nil {
		limit = a.Config.InboxAnalyzer.BodyCharLimit
	}
	emailService, aiService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	go func() {
		var bodies map[string]string
		if emailService != nil {
			bodies, _ = emailService.GetMessagePlainTexts(a.ctx, ids, 0)
		}
		blob := buildSummarizeInput(ids, bodies, state.metaByID, limit)
		if aiService == nil {
			a.QueueUpdateDraw(func() {
				if a.actionPlanState == state {
					view.SetText("⚠️ AI service not available")
				}
			})
			return
		}
		res, err := aiService.GenerateSummary(a.ctx, blob, services.SummaryOptions{})
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state || a.ctx.Err() != nil {
				return
			}
			if err != nil {
				view.SetText(fmt.Sprintf("⚠️ Could not summarize: %v", err))
				return
			}
			view.SetText(a.renderPromptResult(res.Summary))
		})
	}()
}
```

In the test, `a.GetServices()` returns the stub `aiService` only if `a.aiService` is set — which
the test does. `emailService` will be nil in the test (fine; `bodies` stays nil → snippet used).

- [ ] **Step 4: Wire the `y` key in `actionPlanInputCapture`**

In `internal/tui/action_plan.go`, in the `switch key { ... }` action block (with the
`a.Keys.Archive` etc. cases), add:

```go
		case a.Keys.Summarize:
			a.dispatchActionPlanSummarize(state)
			return nil
```

- [ ] **Step 5: Add the keys.go sentinel**

In `internal/tui/keys.go`, add `"action_plan_summary"` to the global pass-through line (next to
`"action_plan_prompt"`):

```go
		if a.currentFocus == "prompt_preview" || a.currentFocus == "action_plan_move" ||
			a.currentFocus == "analyzer_rules" || a.currentFocus == "analyzer_rules_add" ||
			a.currentFocus == "action_plan_rule" || a.currentFocus == "action_plan_prompt" ||
			a.currentFocus == "action_plan_summary" {
			return event
		}
```

- [ ] **Step 6: Run the test + build**

Run: `go test ./internal/tui/ -run TestActionPlanSummarizeSwap -v && go build ./...`
Expected: PASS, build success.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/action_plan_summarize.go internal/tui/action_plan.go internal/tui/keys.go internal/tui/action_plan_summarize_test.go
git commit -m "feat(action-plan): 'y' dispatches a combined AI digest of a category (in-place)"
```

---

### Task 4: Docs + full verification

**Files:**
- Modify: `internal/tui/app.go` (`generateHelpText` footer note — optional), `docs/CONFIGURATION.md`

- [ ] **Step 1: Document the summarize action**

In `docs/CONFIGURATION.md` (analyzer section), add a sentence:

```markdown
The analyzer may also assign a **`summarize`** action to a category; press your summarize key
(`keys.summarize`, default `y`) on that category to get a combined AI digest of its emails in an
in-place panel (Esc to return).
```

- [ ] **Step 2: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass.

- [ ] **Step 3: Suite + leak check**

Run: `go test ./internal/services/ ./internal/tui/ ./test/helpers/ 2>&1 | tail -5`
Expected: all `ok` (the single final `QueueUpdateDraw` is the standard safe pattern — no leak).

- [ ] **Step 4: Build**

Run: `make build`
Expected: success.

- [ ] **Step 5: Commit docs**

```bash
git add internal/tui/app.go docs/CONFIGURATION.md
git commit -m "docs: document the analyzer summarize action"
```

(Live E2E — a `summarize` category, press `y`, see the ⏳ then the rendered digest; Esc returns —
deferred to the user's E2E sweep.)

---

## Self-review notes

- **Spec coverage:** prompt action + metadata (Task 1), `buildSummarizeInput` (Task 2),
  `dispatchActionPlanSummarize` body-swap + non-streaming `GenerateSummary` + final QueueUpdateDraw
  render + `y` wiring + sentinel (Task 3), docs + verification incl. leak check (Task 4). All spec
  sections mapped.
- **Type consistency:** `buildSummarizeInput(ids []string, bodies map[string]string, metaByID map[string]*gmailapi.Message, limit int) string`,
  `dispatchActionPlanSummarize(state *actionPlanState)`, `"action_plan_summary"` sentinel,
  `GenerateSummary(ctx, content, SummaryOptions) (*SummaryResult, error)` (Summary field) — match
  across tasks. `GetServices()` order: EmailService(1), AIService(2).
- **No placeholders:** every code step shows full code.
- **Leak-safety:** non-streaming; a single final `QueueUpdateDraw` guarded on
  `a.actionPlanState == state` — the standard worker→UI pattern, not a per-token callback.
