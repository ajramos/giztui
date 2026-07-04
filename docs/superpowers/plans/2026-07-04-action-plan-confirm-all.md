# Action Plan: Confirm Whole Plan (`c`) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a two-press `c` shortcut (and `:plan apply` command) to the Inbox Action Plan panel that applies every category's suggested action to its non-excluded emails in one go.

**Architecture:** A pure, unit-testable helper (`buildPlanApply`) snapshots the applicable work from the plan; a `confirmPending` flag on `actionPlanState` implements the two-press confirmation entirely on the UI goroutine; one worker goroutine applies categories sequentially, reusing the exact bulk ops the per-category keys use (extracted into `runActionPlanBulkOp` so the two paths cannot drift).

**Tech Stack:** Go, derailed/tview, existing services (`EmailService`, `LabelService`), `ErrorHandler` status system.

**Spec:** `docs/superpowers/specs/2026-07-04-action-plan-confirm-all-design.md`
**Branch:** `feat/action-plan-confirm-all` (already created; spec committed as bdb09d6)
**Issue:** #54

**Spec errata (already verified in code — use these, not the spec's names):**
- The plan type is `services.ActionPlan` (NOT `InboxActionPlan`).
- Persistent status methods are `ShowPersistentMessage(ctx, msg, level LogLevel)` and `ClearPersistentMessage()` (`internal/tui/error_handler.go:94,109`) — there is no `SetPersistentStatus`.
- The in-app `?` help lives in `internal/tui/app.go` (~lines 2297-2298 and 2330-2331), NOT in `help.go`.

## File Map

- **Create:** `internal/tui/action_plan_apply.go` — pure helpers + whole-plan apply logic
- **Create:** `internal/tui/action_plan_apply_test.go` — unit tests for helpers + two-press state machine + command parity
- **Modify:** `internal/tui/action_plan.go` — `confirmPending` field (~:92), input capture (~:702), `executeActionPlanAction` refactor (~:854), `closeActionPlanPanel` (~:679)
- **Modify:** `internal/tui/commands.go` — `executeActionPlanCommand` (~:2195): add `apply` subcommand
- **Modify:** `internal/tui/command_completion.go` — registry entry (~:182) + new `completeActionPlanArg` (~:368)
- **Modify:** `internal/config/config.go` — `KeyBindings` field (~:401), `DefaultConfig()` (~:654), `contextSeparated` allowlist (~:880)
- **Modify:** `internal/tui/app.go` — in-app `?` help (~:2298 and ~:2331)
- **Modify:** `docs/KEYBOARD_SHORTCUTS.md` — In-Panel Keys table + Commands table (Action Plan section, ~:218-245)

**Threading rules that apply everywhere in this plan (from CLAUDE.md and existing code):**
- `ErrorHandler.Show*`, `ShowPersistentMessage`, `ClearPersistentMessage` all wrap `QueueUpdateDraw` internally. From the UI goroutine (key handlers, command handlers in `action_plan.go`) they MUST be called as `go a.GetErrorHandler().X(...)` — a synchronous call deadlocks (see comment at `action_plan.go:861-863`). From a worker goroutine they are called directly.
- Exception: `commands.go` handlers already call `a.GetErrorHandler().ShowError(...)` synchronously (existing precedent in `executeActionPlanCommand`) — keep that style there.
- ESC paths stay synchronous — never `QueueUpdateDraw` inline (a `go`-wrapped ErrorHandler call is fine; it's off-goroutine).
- `confirmPending` is touched ONLY on the UI goroutine (input capture, command handler, panel lifecycle) — no mutex, same as the rest of `actionPlanState`'s non-atomic fields.

---

### Task 1: Pure helpers — `buildPlanApply` + `statusLine` (TDD)

**Files:**
- Create: `internal/tui/action_plan_apply.go`
- Create: `internal/tui/action_plan_apply_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/tui/action_plan_apply_test.go`:

```go
package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestBuildPlanApply(t *testing.T) {
	mk := func(name, action, label string, ids ...string) services.ActionPlanCategory {
		return services.ActionPlanCategory{Name: name, Action: action, Label: label, MessageIDs: ids}
	}
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Newsletters", "archive", "", "m1", "m2", "m3"),
		mk("Receipts", "label", "Finance", "m4", "m5"),
		mk("Spammy", "trash", "", "m6"),
		mk("FYI", "mark_read", "", "m7", "m8"),
		mk("Review", "none", "", "m9"),       // skipped: no action
		mk("Digest", "summarize", "", "m10"), // skipped: not bulk-appliable
		mk("NoLabel", "label", "", "m11"),    // skipped: label action without a label name
		mk("AllOff", "archive", "", "m12"),   // skipped: every email excluded
	}}
	excluded := map[string]bool{"m2": true, "m12": true}

	s := buildPlanApply(plan, excluded)

	if len(s.items) != 4 {
		t.Fatalf("want 4 applicable items, got %d: %+v", len(s.items), s.items)
	}
	// Plan order preserved; exclusions filtered out.
	if s.items[0].catName != "Newsletters" || len(s.items[0].ids) != 2 || s.items[0].ids[0] != "m1" || s.items[0].ids[1] != "m3" {
		t.Fatalf("item 0 wrong: %+v", s.items[0])
	}
	if s.items[1].action != "label" || s.items[1].label != "Finance" || len(s.items[1].ids) != 2 {
		t.Fatalf("item 1 wrong: %+v", s.items[1])
	}
	if s.items[2].catName != "Spammy" || s.items[3].catName != "FYI" {
		t.Fatalf("order wrong: %+v", s.items)
	}
	if s.counts["archive"] != 2 || s.counts["label"] != 2 || s.counts["trash"] != 1 || s.counts["mark_read"] != 2 {
		t.Fatalf("counts wrong: %v", s.counts)
	}
	if s.total != 7 {
		t.Fatalf("want total 7, got %d", s.total)
	}
}

func TestBuildPlanApplyEmpty(t *testing.T) {
	if s := buildPlanApply(nil, nil); s.total != 0 || len(s.items) != 0 {
		t.Fatalf("nil plan should be empty, got %+v", s)
	}
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		{Name: "Review", Action: "none", MessageIDs: []string{"m1"}},
	}}
	if s := buildPlanApply(plan, nil); s.total != 0 || len(s.items) != 0 {
		t.Fatalf("none-only plan should be empty, got %+v", s)
	}
}

func TestPlanApplyStatusLine(t *testing.T) {
	// Fixed action order (archive, mark read, trash, label); only non-zero counts listed.
	s := planApplySummary{counts: map[string]int{"archive": 12, "trash": 3, "label": 5}, total: 20}
	want := "Apply plan: 12 archive, 3 trash, 5 label — press 'c' again to confirm, Esc cancels"
	if got := s.statusLine("c"); got != want {
		t.Fatalf("statusLine:\n got %q\nwant %q", got, want)
	}
	s2 := planApplySummary{counts: map[string]int{"mark_read": 4}, total: 4}
	want2 := "Apply plan: 4 mark read — press 'x' again to confirm, Esc cancels"
	if got := s2.statusLine("x"); got != want2 {
		t.Fatalf("statusLine single:\n got %q\nwant %q", got, want2)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run 'TestBuildPlanApply|TestPlanApplyStatusLine' -v`
Expected: FAIL to compile — `undefined: buildPlanApply`, `undefined: planApplySummary`.

- [ ] **Step 3: Write the implementation**

Create `internal/tui/action_plan_apply.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

// planApplyItem is one category's worth of whole-plan work: the suggested action
// applied to the category's non-excluded message IDs.
type planApplyItem struct {
	catName string
	action  string   // "archive" | "mark_read" | "trash" | "label"
	label   string   // label name when action == "label"
	ids     []string // non-excluded message IDs, plan order
}

// planApplySummary is a snapshot of everything the whole-plan apply will run.
type planApplySummary struct {
	items  []planApplyItem
	counts map[string]int // action → message count (all label categories aggregate under "label")
	total  int            // total messages across items
}

// buildPlanApply computes the applicable work for "apply the whole plan": every category
// with a bulk-appliable action, restricted to its non-excluded messages. Categories with
// action "none"/"summarize" (or anything unknown), label categories without a label name,
// and categories whose checked set is empty are skipped. Pure function — no App, no
// services — so it is unit-testable without the TUI harness.
func buildPlanApply(plan *services.ActionPlan, excluded map[string]bool) planApplySummary {
	s := planApplySummary{counts: map[string]int{}}
	if plan == nil {
		return s
	}
	for _, cat := range plan.Categories {
		switch cat.Action {
		case "archive", "mark_read", "trash":
			// bulk-appliable as-is
		case "label":
			if cat.Label == "" {
				continue // nothing to apply without a label name
			}
		default:
			continue // "none", "summarize", unknown → not bulk-appliable
		}
		ids := checkedIDs(cat.MessageIDs, excluded)
		if len(ids) == 0 {
			continue
		}
		s.items = append(s.items, planApplyItem{catName: cat.Name, action: cat.Action, label: cat.Label, ids: ids})
		s.counts[cat.Action] += len(ids)
		s.total += len(ids)
	}
	return s
}

// statusLine renders the confirmation prompt shown on the first press of the confirm key,
// e.g. "Apply plan: 12 archive, 3 trash, 5 label — press 'c' again to confirm, Esc cancels".
func (s planApplySummary) statusLine(confirmKey string) string {
	order := []string{"archive", "mark_read", "trash", "label"}
	names := map[string]string{"archive": "archive", "mark_read": "mark read", "trash": "trash", "label": "label"}
	parts := make([]string, 0, len(order))
	for _, act := range order {
		if n := s.counts[act]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, names[act]))
		}
	}
	return fmt.Sprintf("Apply plan: %s — press '%s' again to confirm, Esc cancels", strings.Join(parts, ", "), confirmKey)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestBuildPlanApply|TestPlanApplyStatusLine' -v`
Expected: PASS (3 tests).

- [ ] **Step 5: gofmt + commit**

```bash
gofmt -w internal/tui/action_plan_apply.go internal/tui/action_plan_apply_test.go
git add internal/tui/action_plan_apply.go internal/tui/action_plan_apply_test.go
git commit -m "feat(tui): add buildPlanApply summary helpers for whole-plan apply (#54)"
```

---

### Task 2: Extract `runActionPlanBulkOp` (refactor, no behavior change)

**Files:**
- Modify: `internal/tui/action_plan_apply.go` (add the method)
- Modify: `internal/tui/action_plan.go:854-901` (`executeActionPlanAction` uses it)

- [ ] **Step 1: Add the shared dispatch method**

Append to `internal/tui/action_plan_apply.go` (it needs no new imports beyond `fmt`, already imported):

```go
// runActionPlanBulkOp dispatches one category's bulk operation. Shared by the per-category
// quick-action keys and the whole-plan apply so the two paths cannot drift. Must be called
// from a worker goroutine (bulkProgress and the services block).
func (a *App) runActionPlanBulkOp(emailService services.EmailService, labelService services.LabelService, action string, ids []string, label string) error {
	switch action {
	case "archive":
		return emailService.BulkArchive(a.ctx, ids, a.bulkProgress(a.ctx, "Archiving"))
	case "mark_read":
		return emailService.BulkMarkAsRead(a.ctx, ids, a.bulkProgress(a.ctx, "Marking read"))
	case "trash":
		return emailService.BulkTrash(a.ctx, ids, a.bulkProgress(a.ctx, "Trashing"))
	case "label":
		return a.applyActionPlanLabel(labelService, ids, label)
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}
```

- [ ] **Step 2: Rewire `executeActionPlanAction`**

In `internal/tui/action_plan.go`, replace the goroutine body of `executeActionPlanAction` (currently lines 871-900). The `label == ""` warning stays here (per-category behavior unchanged: warn, don't error). Replace this exact block:

```go
	go func() {
		var err error
		switch action {
		case "archive":
			err = emailService.BulkArchive(a.ctx, ids, a.bulkProgress(a.ctx, "Archiving"))
		case "mark_read":
			err = emailService.BulkMarkAsRead(a.ctx, ids, a.bulkProgress(a.ctx, "Marking read"))
		case "trash":
			err = emailService.BulkTrash(a.ctx, ids, a.bulkProgress(a.ctx, "Trashing"))
		case "label":
			if label == "" {
				a.GetErrorHandler().ShowWarning(a.ctx, "Category has no label to apply")
				return
			}
			err = a.applyActionPlanLabel(labelService, ids, label)
		default:
			return
		}
		a.GetErrorHandler().ClearPersistentMessage()
```

with:

```go
	go func() {
		if action == "label" && label == "" {
			a.GetErrorHandler().ShowWarning(a.ctx, "Category has no label to apply")
			return
		}
		err := a.runActionPlanBulkOp(emailService, labelService, action, ids, label)
		a.GetErrorHandler().ClearPersistentMessage()
```

(The rest of the goroutine — the `if err != nil` / `ShowSuccess` / `QueueUpdateDraw` block — stays exactly as it is.)

- [ ] **Step 3: Run the existing action-plan tests (refactor safety net)**

Run: `go test ./internal/tui/ -run 'TestActionPlan|TestCheckedIDs|TestApplyActionPlan|TestBuildAnalyzer' -v && go vet ./internal/tui/`
Expected: PASS, no vet issues.

- [ ] **Step 4: gofmt + commit**

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_apply.go
git add internal/tui/action_plan.go internal/tui/action_plan_apply.go
git commit -m "refactor(tui): extract runActionPlanBulkOp shared by per-category and whole-plan apply (#54)"
```

---

### Task 3: Config key `confirm_plan` (default `c`) + collision allowlist (TDD)

**Files:**
- Modify: `internal/config/config.go` (three spots)

The default `c` collides with the global `compose: "c"` (`config.go:565`) in `ValidateKeyboardConfig`'s duplicate detection. The panel key and the list key can never fire in the same context, so the pair goes in the `contextSeparated` allowlist (`config.go:879`) — exactly like `a` = archive/rule_add. The existing test `TestValidateKeyboardConfig_ContextSeparatedNotWarned` (in `internal/config/keyboard_validation_test.go`) is the failing test for this task: it runs `ValidateKeyboardConfig(DefaultConfig().Keys)` and fails on any un-allowlisted duplicate.

- [ ] **Step 1: Add the struct field**

In `internal/config/config.go`, in the `KeyBindings` struct's "Inbox Action Plan" block (~line 399), insert after `ViewPrompt`:

```go
	ViewPrompt    string `json:"view_prompt"`        // Action plan: view the effective analyzer prompt
	ConfirmPlan   string `json:"confirm_plan"`       // Action plan: confirm & apply the whole plan (two-press)
	RuleAdd       string `json:"rule_add"`           // Analyzer rules panel: add a rule
```

- [ ] **Step 2: Add the default WITHOUT the allowlist entry, run the test, verify it fails**

In `DefaultConfig()` (~line 654), insert after `ViewPrompt: "i", ...`:

```go
		ViewPrompt:    "i", // inspect the effective analyzer prompt (avoids clash with bulk_mode "v")
		ConfirmPlan:   "c", // confirm & apply the whole plan (panel-only; context-separated from compose "c")
		RuleAdd:       "a",
```

Run: `go test ./internal/config/ -run TestValidateKeyboardConfig -v`
Expected: `TestValidateKeyboardConfig_ContextSeparatedNotWarned` FAILS with a warning like `Key 'c' is assigned to multiple functions: compose, confirm_plan`. (This proves the new key participates in validation like any other — spec item 6.)

- [ ] **Step 3: Add the allowlist entry**

In the `contextSeparated` map (~line 880), add:

```go
		"a":      {"archive": true, "rule_add": true},
		"c":      {"compose": true, "confirm_plan": true},
```

- [ ] **Step 4: Run the config test suite (validation + generic migration)**

Run: `go test ./internal/config/ -v`
Expected: ALL PASS. `migrate.go`'s `MissingDefaultKeys` diffs the user file against `DefaultConfig()` generically by dotted path, so `keys.confirm_plan` surfaces to existing users' `config.json` via `:config migrate` with no bespoke code — the passing migrate tests confirm the mechanism is intact.

- [ ] **Step 5: gofmt + commit**

```bash
gofmt -w internal/config/config.go
git add internal/config/config.go
git commit -m "feat(config): add keys.confirm_plan (default 'c') for whole-plan apply (#54)"
```

---

### Task 4: Two-press state machine + whole-plan execution (TDD)

**Files:**
- Modify: `internal/tui/action_plan.go` (state field, input capture, panel close)
- Modify: `internal/tui/action_plan_apply.go` (`startActionPlanConfirm`, `executeActionPlanApply`)
- Test: `internal/tui/action_plan_apply_test.go`

- [ ] **Step 1: Write the failing TUI test**

Append to `internal/tui/action_plan_apply_test.go`. Note the imports this file now needs at the top: add `tcell "github.com/derailed/tcell/v2"`, `"github.com/derailed/tview"`, and `gmailapi "google.golang.org/api/gmail/v1"` to the existing import block. The construction pattern (bare `&App{}` + `NewErrorHandler(nil, nil, nil, nil, nil)`) is the established one from `TestActionPlanMoveInlineSwap` and `action_plan_rules_test.go:28`; with a nil tview app, ErrorHandler skips its `QueueUpdateDraw`, so the `go`-wrapped status calls are harmless in tests.

```go
func newConfirmTestApp(t *testing.T) (*App, *actionPlanState, func(*tcell.EventKey) *tcell.EventKey) {
	t.Helper()
	a := &App{Application: tview.NewApplication()}
	a.Pages = NewPages()
	a.errorHandler = NewErrorHandler(nil, nil, nil, nil, nil)
	a.Keys.ConfirmPlan = "c"
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1", "m2"}},
		}},
		excluded: map[string]bool{},
		expanded: map[int]bool{},
		metaByID: map[string]*gmailapi.Message{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	a.actionPlanState = state
	return a, state, a.actionPlanInputCapture(state)
}

func TestActionPlanConfirmTwoPressStateMachine(t *testing.T) {
	a, state, capture := newConfirmTestApp(t)

	// First press of 'c' arms the confirmation and is consumed.
	if ev := capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone)); ev != nil {
		t.Fatal("first confirm press should be consumed")
	}
	if !state.confirmPending {
		t.Fatal("first confirm press should set confirmPending")
	}

	// Any other key clears the pending confirmation AND still does its normal job
	// (Down passes through to the TreeView).
	if ev := capture(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)); ev == nil {
		t.Fatal("navigation key should still pass through to the tree")
	}
	if state.confirmPending {
		t.Fatal("any other key must clear confirmPending")
	}

	// Esc while pending cancels the confirmation ONLY — panel stays open.
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if !state.confirmPending {
		t.Fatal("re-arming failed")
	}
	if ev := capture(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)); ev != nil {
		t.Fatal("Esc while pending should be consumed")
	}
	if state.confirmPending {
		t.Fatal("Esc while pending must clear confirmPending")
	}
	if a.actionPlanState != state {
		t.Fatal("Esc while pending must NOT close the panel")
	}
}

func TestActionPlanConfirmBlockedWhileAnalyzing(t *testing.T) {
	_, state, capture := newConfirmTestApp(t)
	state.analyzing.Store(true)
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if state.confirmPending {
		t.Fatal("confirm must be blocked while analysis is running")
	}
}

func TestStartActionPlanConfirmNothingToApply(t *testing.T) {
	_, state, capture := newConfirmTestApp(t)
	state.excluded["m1"] = true
	state.excluded["m2"] = true // everything excluded → nothing applicable
	capture(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if state.confirmPending {
		t.Fatal("empty apply set must not arm the confirmation")
	}
}
```

(Deliberately NOT simulated here: the second `c` press, because `executeActionPlanApply` would call real bulk services, which are nil in this harness. The execution body is a thin sequential loop over `runActionPlanBulkOp` — dispatch already covered by Task 2's suite — and gets exercised in the live smoke test.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run 'TestActionPlanConfirm|TestStartActionPlanConfirm' -v`
Expected: FAIL to compile — `a.Keys.ConfirmPlan` exists (Task 3) but `state.confirmPending` is undefined.

- [ ] **Step 3: Add the state field**

In `internal/tui/action_plan.go`, `actionPlanState` struct (~line 91), add after `streamingCancel`:

```go
	streamingCancel context.CancelFunc

	confirmPending bool // whole-plan apply armed (first press of keys.confirm_plan); UI goroutine only
```

- [ ] **Step 4: Add the confirmation entry points**

Append to `internal/tui/action_plan_apply.go`:

```go
// startActionPlanConfirm handles the FIRST press of the confirm-plan key (and :plan apply):
// compute the apply summary and arm the two-press confirmation. UI goroutine only.
func (a *App) startActionPlanConfirm(state *actionPlanState) {
	summary := buildPlanApply(state.plan, state.excluded)
	if summary.total == 0 {
		// go: Show* wrap QueueUpdateDraw; a synchronous call from the UI goroutine deadlocks.
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing to apply — all emails are excluded or categories have no action")
		return
	}
	state.confirmPending = true
	go a.GetErrorHandler().ShowPersistentMessage(a.ctx, summary.statusLine(a.Keys.ConfirmPlan), LogLevelInfo)
}

// executeActionPlanApply runs the whole plan: every applicable category, sequentially, in one
// worker goroutine. Failures are reported and skipped (the rest of the plan still runs);
// applied categories disappear from the tree as they complete, same as per-category apply.
// The summary snapshot is computed on the UI goroutine BEFORE the worker starts.
func (a *App) executeActionPlanApply(state *actionPlanState) {
	summary := buildPlanApply(state.plan, state.excluded)
	if summary.total == 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing to apply — all emails are excluded or categories have no action")
		return
	}
	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()

	go func() {
		applied, appliedMsgs, failed := 0, 0, 0
		for i, item := range summary.items {
			a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Applying plan (%d/%d): %s…", i+1, len(summary.items), item.catName))
			if err := a.runActionPlanBulkOp(emailService, labelService, item.action, item.ids, item.label); err != nil {
				failed++
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Action failed on %q: %v", item.catName, err))
				continue // a failing category (e.g. missing label in strict mode) must not abort the rest
			}
			applied++
			appliedMsgs += len(item.ids)
			catName := item.catName
			a.QueueUpdateDraw(func() {
				if a.actionPlanState == state { // panel may have been closed mid-run
					a.removeActionPlanCategory(state, catName)
				}
			})
		}
		a.GetErrorHandler().ClearPersistentMessage()
		if failed == 0 {
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("✓ Plan applied: %d categories, %d messages", applied, appliedMsgs))
		} else {
			a.GetErrorHandler().ShowWarning(a.ctx, fmt.Sprintf("Plan applied: %d categories, %d messages (%d failed)", applied, appliedMsgs, failed))
		}
	}()
}
```

- [ ] **Step 5: Wire the input capture**

In `internal/tui/action_plan.go`, `actionPlanInputCapture` (~line 702), two edits.

**Edit A** — at the very top of the returned closure, BEFORE the existing `if ev.Key() == tcell.KeyEscape` branch, insert the pending-state dispatcher:

```go
	return func(ev *tcell.EventKey) *tcell.EventKey {
		// Two-press confirm state: while armed, the confirm key executes, Esc cancels the
		// confirmation ONLY (panel stays open; next Esc closes as usual), and any other key
		// disarms then does its normal job. Status calls are go-wrapped (QueueUpdateDraw inside).
		if state.confirmPending {
			switch {
			case a.matchesConfiguredKey(ev, a.Keys.ConfirmPlan):
				state.confirmPending = false
				go a.GetErrorHandler().ClearPersistentMessage()
				a.executeActionPlanApply(state)
				return nil
			case ev.Key() == tcell.KeyEscape:
				state.confirmPending = false
				go a.GetErrorHandler().ClearPersistentMessage()
				return nil
			default:
				state.confirmPending = false
				go a.GetErrorHandler().ClearPersistentMessage()
				// fall through: the key still performs its normal action below
			}
		}

		// ESC: synchronous close (no QueueUpdateDraw).
		if ev.Key() == tcell.KeyEscape {
```

**Edit B** — the first-press handler goes AFTER the analyzing guard (blocked during analysis, like the other quick-actions). Insert between the guard and the `Move` handler:

```go
		// Quick-actions are blocked until analysis finishes (avoids racing the plan).
		if state.analyzing.Load() {
			return nil
		}
		// Confirm & apply the whole plan (two-press; see startActionPlanConfirm).
		if a.matchesConfiguredKey(ev, a.Keys.ConfirmPlan) {
			a.startActionPlanConfirm(state)
			return nil
		}
		// 'm' moves: on an email node, that one email; on a category or read-manually header,
```

- [ ] **Step 6: Clear a dangling confirmation status on panel close**

In `internal/tui/action_plan.go`, `closeActionPlanPanel` (~line 679), insert right after the `streamingCancel` block (spec item: the panel can close through paths other than Esc-while-pending):

```go
	if a.actionPlanState != nil && a.actionPlanState.confirmPending {
		go a.GetErrorHandler().ClearPersistentMessage()
	}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestActionPlanConfirm|TestStartActionPlanConfirm' -v && go test ./internal/tui/ -run 'TestActionPlan' -v`
Expected: ALL PASS (new + pre-existing action-plan tests).

- [ ] **Step 8: gofmt + commit**

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_apply.go internal/tui/action_plan_apply_test.go
git add internal/tui/action_plan.go internal/tui/action_plan_apply.go internal/tui/action_plan_apply_test.go
git commit -m "feat(tui): two-press 'c' confirm applies the whole action plan (#54)"
```

---

### Task 5: Command parity — `:plan apply` (TDD)

**Files:**
- Modify: `internal/tui/action_plan_apply.go` (`applyActionPlanFromCommand`)
- Modify: `internal/tui/commands.go:2195` (`executeActionPlanCommand`)
- Modify: `internal/tui/command_completion.go` (~:182 registry entry, ~:368 completer)
- Test: `internal/tui/action_plan_apply_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_apply_test.go`:

```go
func TestExecuteActionPlanApplyCommand(t *testing.T) {
	a, state, _ := newConfirmTestApp(t)

	// Analysis still running → refused, not armed.
	state.analyzing.Store(true)
	a.executeActionPlanCommand([]string{"apply"})
	if state.confirmPending {
		t.Fatal(":plan apply must be refused while analysis is running")
	}
	state.analyzing.Store(false)

	// Panel open + finished → same first-press behavior as the key, and focusOverride
	// is set so hideCommandBar's teardown doesn't steal focus from the panel.
	a.executeActionPlanCommand([]string{"apply"})
	if !state.confirmPending {
		t.Fatal(":plan apply should arm the two-press confirmation")
	}
	if a.cmd.focusOverride != "keep" {
		t.Fatalf("expected cmd.focusOverride=keep, got %q", a.cmd.focusOverride)
	}

	// No panel open → error, no panic.
	a.actionPlanState = nil
	a.executeActionPlanCommand([]string{"apply"})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestExecuteActionPlanApplyCommand -v`
Expected: FAIL — `:plan apply` currently hits the `Unknown action-plan option` branch, so `confirmPending` stays false.

- [ ] **Step 3: Implement the command path**

Append to `internal/tui/action_plan_apply.go`:

```go
// applyActionPlanFromCommand backs ':action-plan apply' / ':plan apply' / ':ap apply'.
// Runs on the UI goroutine (command dispatch). ShowError is called synchronously here,
// matching the existing style of executeActionPlanCommand.
func (a *App) applyActionPlanFromCommand() {
	state := a.actionPlanState
	if state == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Open the action plan first (:plan)")
		return
	}
	if state.analyzing.Load() {
		a.GetErrorHandler().ShowError(a.ctx, "Analysis still running — wait for it to finish")
		return
	}
	// Keep focus on the panel: hideCommandBar's restoreFocusAfterModal would otherwise
	// force focus back to the list (see :plan rules, action_plan_rules.go:220).
	a.cmd.focusOverride = "keep"
	a.startActionPlanConfirm(state)
}
```

In `internal/tui/commands.go`, `executeActionPlanCommand` (~line 2195), insert a branch before the `rules` one (and update the doc comment above the function to `// executeActionPlanCommand handles :action-plan / :plan / :ap [apply|rules|with-prompt <name-or-id>].`):

```go
	if strings.ToLower(args[0]) == "apply" {
		a.applyActionPlanFromCommand()
		return
	}
	if strings.ToLower(args[0]) == "rules" {
```

- [ ] **Step 4: Registry help card + argument completion**

In `internal/tui/command_completion.go`, replace the registry entry (~line 182):

```go
	{name: "action-plan", aliases: []string{"plan", "ap"}, completeArg: completeActionPlanArg, help: &cmdHelp{
		summary:  "Run the AI Inbox Action Plan over the selected messages.",
		syntax:   ":action-plan [apply|rules|with-prompt <name-or-id>]",
		examples: []string{":plan", ":plan apply", ":plan rules", ":plan with-prompt triage"},
	}},
```

And add the completer next to `completePromptArg` (~line 368), following the same shape:

```go
// completeActionPlanArg: ':action-plan <subcommand>'. First token → apply/rules/with-prompt.
func completeActionPlanArg(a *App, rest string) []string {
	head, prefix := splitLastToken(rest)
	if head != "" {
		return nil
	}
	return withHead("", filterByPrefix([]string{"apply", "rules", "with-prompt"}, prefix))
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestExecuteActionPlanApplyCommand|TestCommandCompletion|TestComplete' -v`
Expected: PASS (the completion test patterns are generic; if a registry-snapshot test exists it now includes the new syntax line).

- [ ] **Step 6: gofmt + commit**

```bash
gofmt -w internal/tui/commands.go internal/tui/command_completion.go internal/tui/action_plan_apply.go internal/tui/action_plan_apply_test.go
git add internal/tui/commands.go internal/tui/command_completion.go internal/tui/action_plan_apply.go internal/tui/action_plan_apply_test.go
git commit -m "feat(tui): :plan apply command parity for whole-plan confirm (#54)"
```

---

### Task 6: In-app `?` help + docs

**Files:**
- Modify: `internal/tui/app.go:2298` and `:2330-2331`
- Modify: `docs/KEYBOARD_SHORTCUTS.md` (Action Plan section)

- [ ] **Step 1: In-app help — panel line**

In `internal/tui/app.go` (~line 2298), replace:

```go
	fmt.Fprintf(&help, "      └ in panel: Enter open in reader · %s remember rule · %s view prompt · %s move · %s exclude\n\n", a.Keys.RememberRule, a.Keys.ViewPrompt, a.Keys.Move, a.Keys.BulkSelect)
```

with:

```go
	fmt.Fprintf(&help, "      └ in panel: Enter open in reader · %s remember rule · %s view prompt · %s move · %s exclude · %s confirm whole plan (press twice)\n\n", a.Keys.RememberRule, a.Keys.ViewPrompt, a.Keys.Move, a.Keys.BulkSelect, a.Keys.ConfirmPlan)
```

- [ ] **Step 2: In-app help — command equivalents**

In `internal/tui/app.go` (~line 2331), after the `:plan rules` line, add:

```go
	fmt.Fprintf(&help, "    %-18s 🧠  Apply the whole plan (same as '%s' in the panel; press twice to confirm)\n", ":plan apply", a.Keys.ConfirmPlan)
```

- [ ] **Step 3: KEYBOARD_SHORTCUTS.md — In-Panel Keys table**

In `docs/KEYBOARD_SHORTCUTS.md`, Action Plan "In-Panel Keys" table (~line 218), add a row after the `Ctrl+R` (Remember rule) row:

```markdown
| `c` | Confirm whole plan | Press once to see a summary of everything the plan will do (e.g. `Apply plan: 12 archive, 3 trash, 5 label`), press `c` again to apply every category's suggested action to its checked emails; `Esc` cancels the confirmation without closing the panel (`keys.confirm_plan`) |
```

- [ ] **Step 4: KEYBOARD_SHORTCUTS.md — Commands table**

In the Action Plan "Commands" table (~line 245), add after the `:action-plan rules` row:

```markdown
| `:action-plan apply` | `:plan apply`, `:ap apply` | Apply the whole plan — same two-step confirmation as `c` in the panel (requires the panel to be open with analysis finished) |
```

- [ ] **Step 5: Build + commit**

Run: `go build ./... && go vet ./internal/tui/`
Expected: clean.

```bash
gofmt -w internal/tui/app.go
git add internal/tui/app.go docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs: document whole-plan confirm key and :plan apply (#54)"
```

---

### Task 7: Final quality gate

- [ ] **Step 1: Full pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + golangci-lint + essential tests ALL green. Fix anything it flags (gofmt runs inline, no permission needed).

- [ ] **Step 2: Full test suite (separate step, leak detector)**

Run: `make test`
Expected: ALL green, including the `test/helpers` leak detector. (Watch for goroutine leaks: the only new goroutines are the `go ErrorHandler.X` wrappers and the apply worker — none run in `initServices` or tests with a live app, so no leak is expected.)

- [ ] **Step 3: Commit any gate fixes (if needed), otherwise done**

```bash
git status   # confirm clean
git log --oneline main..HEAD
```

Expected: ~6 commits on `feat/action-plan-confirm-all`.

**After this task (per project workflow, NOT part of the plan's commits):** push the branch for the user's Mac smoke test — do NOT open a PR or merge; merge to main + close #54 only after the user confirms "funciona".

---

## Self-Review Notes

- Spec coverage: trigger key + placement after analyzing guard (T4), two-press flow incl. Esc-cancels-confirmation-only and any-other-key-disarms (T4), nothing-to-apply ShowInfo (T4 via `startActionPlanConfirm`), sequential continue-on-failure execution with per-category tree removal guarded by `a.actionPlanState == state` (T4), `runActionPlanBulkOp` extraction (T2), pure `buildPlanApply`/`statusLine` + unit tests (T1), command parity with focusOverride + registry card + completer (T5), config field/default/allowlist/generic migration (T3), in-app help + docs (T6), quality gates (T7). Undo of the whole plan: out of scope per spec — no task, correct.
- Spec deviation (intentional, documented): categories with action `summarize` are skipped by `buildPlanApply` — summarize is an AI streaming op, not a bulk-appliable action; the spec's skip rule ("none") is extended to anything not in {archive, mark_read, trash, label}.
- The second-press execution path is not simulated in the TUI harness (nil services would panic in the worker); it is covered by the pure-helper tests + `runActionPlanBulkOp` reuse + the live smoke test on the TEST Gmail account.
