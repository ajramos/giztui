# Effective Analyzer-Prompt Viewer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Press `v` in the Action Plan to open an in-place, read-only view of the effective analyzer prompt (saved-rules block + base prompt + literal `{{messages}}` + a note).

**Architecture:** A pure `BuildPromptPreview` service method reuses the exact assembly `Analyze` performs (`prependUserRules` over the chosen base), leaving `{{messages}}` literal. The TUI body-swaps the Action Plan tree for a scrollable `TextView` showing that preview, mirroring the move-chooser pattern.

**Tech Stack:** Go, tview/tcell (derailed forks), GizTUI services + Action Plan body-swap pattern.

Spec: `docs/superpowers/specs/2026-06-11-analyzer-prompt-viewer-design.md`

---

### Task 1: `BuildPromptPreview` service method

**Files:**
- Modify: `internal/services/interfaces.go` (`InboxAnalyzerService` interface)
- Modify: `internal/services/inbox_analyzer_service.go`
- Test: `internal/services/inbox_analyzer_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/inbox_analyzer_service_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestBuildPromptPreview -v`
Expected: FAIL — `s.BuildPromptPreview` undefined.

- [ ] **Step 3: Add the interface method**

In `internal/services/interfaces.go`, add to the `InboxAnalyzerService` interface (after the `Analyze` method):

```go
	// BuildPromptPreview returns the assembled analyzer prompt (user-rules block + base
	// prompt) with {{messages}} left literal — the same assembly Analyze performs, minus the
	// per-batch payload. Pure: no AI call, no network.
	BuildPromptPreview(opts InboxAnalyzerOptions) string
```

- [ ] **Step 4: Implement the method**

In `internal/services/inbox_analyzer_service.go`, add (after the `Analyze` method or near `prependUserRules`):

```go
// BuildPromptPreview assembles the analyzer prompt the way Analyze does (base prompt with the
// user-rules block prepended), leaving {{messages}} as a literal placeholder.
func (s *InboxAnalyzerServiceImpl) BuildPromptPreview(opts InboxAnalyzerOptions) string {
	base := opts.CustomPromptText
	if strings.TrimSpace(base) == "" {
		base = defaultAnalyzerPrompt
	}
	return prependUserRules(base, opts.UserRules)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestBuildPromptPreview -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/services/interfaces.go internal/services/inbox_analyzer_service.go internal/services/inbox_analyzer_service_test.go
git commit -m "feat(analyzer): BuildPromptPreview assembles effective prompt (rules + base)"
```

---

### Task 2: Footer hint + keys.go sentinel

**Files:**
- Modify: `internal/tui/action_plan.go` (`actionPlanFooterText`, ~line 539)
- Modify: `internal/tui/keys.go` (pass-through sentinel line)
- Test: `internal/tui/action_plan_test.go` (`TestActionPlanFooterText`)

- [ ] **Step 1: Extend the footer test**

In `internal/tui/action_plan_test.go`, inside `TestActionPlanFooterText`, add these assertions just before the final closing brace:

```go
	if !strings.Contains(onCat, "v prompt") {
		t.Fatalf("category footer should advertise the prompt viewer: %q", onCat)
	}
	if !strings.Contains(onEmail, "v prompt") {
		t.Fatalf("email footer should advertise the prompt viewer: %q", onEmail)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestActionPlanFooterText -v`
Expected: FAIL — footer does not contain "v prompt".

- [ ] **Step 3: Add the hint to both footer branches**

In `internal/tui/action_plan.go`, in `actionPlanFooterText`, update both returns:

Replace the category branch's `parts = append(...)` line:

```go
		parts = append(parts, "Enter to expand", "Ctrl+R to remember", "Tab to inbox", "Esc to close")
```

with:

```go
		parts = append(parts, "Enter to expand", "v prompt", "Ctrl+R to remember", "Tab to inbox", "Esc to close")
```

Replace the email branch's return:

```go
	return " Space to skip  |  m to move  |  Ctrl+R to remember sender  |  Tab to inbox  |  Esc to close "
```

with:

```go
	return " Space to skip  |  m to move  |  v prompt  |  Ctrl+R to remember sender  |  Tab to inbox  |  Esc to close "
```

- [ ] **Step 4: Add the keys.go sentinel**

In `internal/tui/keys.go`, in the global pass-through line that lists `"prompt_preview"` etc., add `"action_plan_prompt"`. The line becomes:

```go
		if a.currentFocus == "prompt_preview" || a.currentFocus == "action_plan_move" ||
			a.currentFocus == "analyzer_rules" || a.currentFocus == "analyzer_rules_add" ||
			a.currentFocus == "action_plan_rule" || a.currentFocus == "action_plan_prompt" {
			return event
```

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/tui/ -run TestActionPlanFooterText -v && go build ./...`
Expected: PASS, build success.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/keys.go internal/tui/action_plan_test.go
git commit -m "feat(action-plan): footer 'v prompt' hint + key pass-through sentinel"
```

---

### Task 3: The viewer panel + `v` key wiring

**Files:**
- Create: `internal/tui/action_plan_prompt.go`
- Modify: `internal/tui/action_plan.go` (`actionPlanInputCapture`, the `m` handler region)
- Test: `internal/tui/action_plan_test.go` (append a stub analyzer + swap test)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_test.go`:

```go
type stubAnalyzerSvc struct{}

func (stubAnalyzerSvc) Analyze(ctx context.Context, messages []services.AnalyzerMessage, opts services.InboxAnalyzerOptions, onProgress func(*services.ActionPlan)) (*services.ActionPlan, error) {
	return nil, nil
}
func (stubAnalyzerSvc) BuildPromptPreview(opts services.InboxAnalyzerOptions) string {
	return "PREVIEW-BODY {{messages}}"
}

func TestActionPlanPromptViewSwap(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	a.Config = config.DefaultConfig()
	a.inboxAnalyzerService = stubAnalyzerSvc{}
	state := &actionPlanState{
		customPromptText: "",
		excluded:         map[string]bool{},
		expanded:         map[int]bool{},
		footer:           tview.NewTextView(),
		plan:             &services.ActionPlan{},
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.showActionPlanPromptView(state)
	if a.currentFocus != "action_plan_prompt" {
		t.Fatalf("expected currentFocus=action_plan_prompt, got %q", a.currentFocus)
	}
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out while the prompt view is shown")
	}
	view, ok := a.GetFocus().(*tview.TextView)
	if !ok {
		t.Fatalf("expected the prompt TextView focused, got %T", a.GetFocus())
	}
	if !strings.Contains(view.GetText(true), "PREVIEW-BODY") {
		t.Fatalf("view should show the assembled prompt, got %q", view.GetText(true))
	}

	// Esc restores the tree.
	if cap := view.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.currentFocus != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.currentFocus)
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc, the tree should be restored as the container body")
	}
}
```

Add `"context"` and `"github.com/ajramos/giztui/internal/config"` to the imports of
`internal/tui/action_plan_test.go` if not already present.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestActionPlanPromptViewSwap -v`
Expected: FAIL — `a.showActionPlanPromptView` undefined.

- [ ] **Step 3: Create the viewer panel**

Create `internal/tui/action_plan_prompt.go`:

```go
package tui

import (
	"fmt"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// showActionPlanPromptView body-swaps the Action Plan tree for a read-only view of the
// effective analyzer prompt (rules + base + literal {{messages}}), so the user can see exactly
// what is assembled before the per-batch payload is injected. Esc returns to the tree.
func (a *App) showActionPlanPromptView(state *actionPlanState) {
	svc := a.GetInboxAnalyzerService()
	if svc == nil || state == nil {
		return
	}

	var userRules []string
	if rsvc := a.GetAnalyzerRulesService(); rsvc != nil {
		if rs, err := rsvc.ListRules(a.ctx); err == nil {
			for _, r := range rs {
				userRules = append(userRules, r.RuleText)
			}
		}
	}
	opts := services.InboxAnalyzerOptions{
		CustomPromptText: state.customPromptText,
		UserRules:        userRules,
		BodyCharLimit:    a.Config.InboxAnalyzer.BodyCharLimit,
	}
	prompt := svc.BuildPromptPreview(opts)

	note := fmt.Sprintf("{{messages}} is replaced per batch with each email's subject / from / body (up to %d chars).", a.Config.InboxAnalyzer.BodyCharLimit)
	if !a.Config.InboxAnalyzer.IncludeBody {
		note = "{{messages}} is replaced per batch with each email's subject / from / snippet."
	}

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(true)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(tview.Escape(note + "\n\n" + prompt))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
	}
	view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev // arrows scroll the TextView
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(view, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(" 🔎 Effective analyzer prompt ")
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.currentFocus = "action_plan_prompt"
	a.SetFocus(view)
}
```

- [ ] **Step 4: Wire the `v` key**

In `internal/tui/action_plan.go`, in `actionPlanInputCapture`, immediately after the `m` handler block (the `if ev.Rune() == 'm' && cur != nil { … }` switch) and before the `switch key { … }` action block, add:

```go
		// 'v' opens the effective-prompt viewer (blocked during analysis, like quick-actions).
		if ev.Rune() == 'v' {
			a.showActionPlanPromptView(state)
			return nil
		}
```

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/tui/ -run TestActionPlanPromptViewSwap -v && go build ./...`
Expected: PASS, build success.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/action_plan_prompt.go internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(action-plan): v opens the effective analyzer-prompt viewer in place"
```

---

### Task 4: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass. Fix any issue and re-run.

- [ ] **Step 2: Full tui + services tests**

Run: `go test ./internal/tui/ ./internal/services/ 2>&1 | tail -4`
Expected: all `ok`.

- [ ] **Step 3: Build**

Run: `make build`
Expected: success; binary reports the current version.

(Live E2E — open the Action Plan, press `v`, confirm the assembled prompt shows with the rules block and `{{messages}}` placeholder, arrows scroll, Esc returns to the tree — is deferred to the user's end-of-queue E2E sweep.)

---

## Self-review notes

- **Spec coverage:** `BuildPromptPreview` reusing the Analyze assembly (Task 1), `v` trigger + footer hint + keys sentinel (Tasks 2–3), in-place body-swap TextView with the per-batch note (Task 3), tests for the service method + the swap (Tasks 1, 3), verification (Task 4). All spec sections mapped.
- **Type consistency:** `BuildPromptPreview(opts InboxAnalyzerOptions) string` matches between the interface (Task 1), the impl (Task 1), the stub (Task 3), and the call site (Task 3). `currentFocus == "action_plan_prompt"` matches between keys.go (Task 2), the panel (Task 3), and the test (Task 3). `showActionPlanPromptView(state *actionPlanState)` matches the `v` call site.
- **No placeholders:** every code step shows full code; commands have expected output.
- **Threading:** `ListRules` is a fast read; body-swap and Esc restore are synchronous (no `QueueUpdateDraw`), matching the move chooser; no `ErrorHandler` calls on this path.
