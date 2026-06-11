# Analyzer Rules In-Place UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two floating analyzer-rules modals (the `:plan rules` manager and the `Ctrl+R` quick-remember input) with in-place panels so they behave like the rest of the pickers and can no longer hang.

**Architecture:** The rules manager becomes a side-panel picker mounted as `a.labelsView` in `contentSplit` (the same slot the Action Plan uses), with an embedded body-swap input for "add". The `Ctrl+R` quick-remember becomes a body-swap of the Action Plan tree for an input field. `keys.go` gains `currentFocus` sentinels so the global capture lets these panels own their keys; the dead `Pages` overlay code is removed.

**Tech Stack:** Go, tview/tcell (derailed forks), GizTUI TUI picker + body-swap patterns.

Spec: `docs/superpowers/specs/2026-06-11-analyzer-rules-inplace-design.md`

---

### Task 1: Picker constant + keys.go pass-through sentinels

**Files:**
- Modify: `internal/tui/app.go` (ActivePicker const block, ~line 41)
- Modify: `internal/tui/keys.go:579` (global pass-through sentinel line)

No standalone test (wiring; covered by Tasks 2–3 swap tests). Build only.

- [ ] **Step 1: Add the picker constant**

In `internal/tui/app.go`, in the `const ( ... )` ActivePicker block, after `PickerActionPlan`:

```go
	PickerAnalyzerRules      ActivePicker = "analyzer_rules"
```

- [ ] **Step 2: Add the currentFocus sentinels to the global pass-through**

In `internal/tui/keys.go`, replace line 579:

```go
		if a.currentFocus == "prompt_preview" || a.currentFocus == "action_plan_move" {
```

with:

```go
		if a.currentFocus == "prompt_preview" || a.currentFocus == "action_plan_move" ||
			a.currentFocus == "analyzer_rules" || a.currentFocus == "analyzer_rules_add" ||
			a.currentFocus == "action_plan_rule" {
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go internal/tui/keys.go
git commit -m "feat(tui): add analyzer-rules picker constant + key pass-through sentinels"
```

---

### Task 2: Rules manager → side-panel picker (with embedded add)

**Files:**
- Modify: `internal/tui/action_plan_rules.go` (rewrite `openAnalyzerRulesManager`)
- Test: `internal/tui/action_plan_rules_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `internal/tui/action_plan_rules_test.go`:

```go
package tui

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// stubRulesService implements services.AnalyzerRulesService for picker swap/focus tests.
type stubRulesService struct{}

func (stubRulesService) SaveRule(ctx context.Context, ruleText string) error { return nil }
func (stubRulesService) ListRules(ctx context.Context) ([]services.AnalyzerRuleInfo, error) {
	return nil, nil
}
func (stubRulesService) DeleteRule(ctx context.Context, id int64) error { return nil }
func (stubRulesService) SuggestRuleFromContext(from, action string, negate bool) string {
	return ""
}

func newRulesTestApp() *App {
	a := &App{Application: tview.NewApplication()}
	a.ctx = context.Background()
	a.Pages = NewPages()
	a.errorHandler = NewErrorHandler(nil, nil, nil, nil, nil)
	a.analyzerRulesService = stubRulesService{}
	a.views = map[string]tview.Primitive{
		"contentSplit": tview.NewFlex(),
		"list":         tview.NewTable(),
	}
	return a
}

func TestAnalyzerRulesPickerSwap(t *testing.T) {
	a := newRulesTestApp()

	a.openAnalyzerRulesManager()
	if a.currentFocus != "analyzer_rules" {
		t.Fatalf("expected currentFocus=analyzer_rules, got %q", a.currentFocus)
	}
	if a.currentActivePicker != PickerAnalyzerRules {
		t.Fatalf("expected active picker=analyzer_rules, got %q", a.currentActivePicker)
	}
	// The picker list must be focused.
	list, ok := a.GetFocus().(*tview.List)
	if !ok {
		t.Fatalf("expected the rules list focused, got %T", a.GetFocus())
	}

	// Esc closes the picker and returns focus to the inbox.
	if cap := list.GetInputCapture(); cap != nil {
		cap(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.currentFocus != "list" {
		t.Fatalf("after Esc, currentFocus should be list, got %q", a.currentFocus)
	}
	if a.currentActivePicker != PickerNone {
		t.Fatalf("after Esc, active picker should be none, got %q", a.currentActivePicker)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestAnalyzerRulesPickerSwap -v`
Expected: FAIL — current `openAnalyzerRulesManager` uses a floating page, so `currentActivePicker`/`currentFocus` are not set as asserted.

- [ ] **Step 3: Rewrite `openAnalyzerRulesManager`**

In `internal/tui/action_plan_rules.go`, replace the entire `openAnalyzerRulesManager` function with:

```go
// openAnalyzerRulesManager shows the analyzer rules as an in-place side-panel picker
// (mounted as a.labelsView in contentSplit, like the Action Plan). 'a' adds a rule via an
// embedded input (body-swap), 'd' deletes the highlighted rule, Esc closes. No floating modal.
func (a *App) openAnalyzerRulesManager() {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	// The rules picker and the Action Plan both occupy a.labelsView; close the plan first.
	if a.actionPlanState != nil {
		a.closeActionPlanPanel()
	}
	colors := a.GetComponentColors("ai")

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBackgroundColor(colors.Background.Color())
	container.SetBorder(true)
	container.SetTitle(" 🧠 Analyzer rules ")
	container.SetTitleColor(colors.Title.Color())
	container.SetBorderColor(colors.Border.Color())

	footer := tview.NewTextView().SetTextAlign(tview.AlignRight)
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	footer.SetText(" a add · d delete · Esc close ")

	var rules []services.AnalyzerRuleInfo
	reload := func() {
		list.Clear()
		rs, err := svc.ListRules(a.ctx)
		if err != nil {
			go a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("List rules failed: %v", err))
			return
		}
		rules = rs
		if len(rs) == 0 {
			list.AddItem("(no rules yet — press 'a' to add)", "", 0, nil)
			return
		}
		for _, r := range rs {
			list.AddItem(r.RuleText, "", 0, nil)
		}
	}
	reload()

	container.AddItem(list, 0, 1, true)
	container.AddItem(footer, 1, 0, false)

	closePicker := func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.ResizeItem(a.labelsView, 0, 0)
			}
		}
		a.setActivePicker(PickerNone)
		if l, ok := a.views["list"].(*tview.Table); ok {
			a.SetFocus(l)
		}
		a.currentFocus = "list"
		a.updateFocusIndicators("list")
	}

	// showAddInput body-swaps the list for an input field inside the same container.
	showAddInput := func() {
		input := tview.NewInputField().SetLabel(" New rule: ").SetFieldWidth(0)
		input.SetBackgroundColor(colors.Background.Color())
		input.SetFieldBackgroundColor(colors.Background.Color())
		input.SetFieldTextColor(colors.Text.Color())

		restore := func() {
			container.RemoveItem(input)
			container.RemoveItem(footer)
			container.AddItem(list, 0, 1, true)
			container.AddItem(footer, 1, 0, false)
			footer.SetText(" a add · d delete · Esc close ")
			a.currentFocus = "analyzer_rules"
			a.SetFocus(list)
		}
		input.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				text := input.GetText()
				restore()
				go func() {
					if err := svc.SaveRule(a.ctx, text); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
						return
					}
					a.QueueUpdateDraw(reload)
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved — applies on next analysis")
				}()
				return
			}
			restore() // Esc / Tab
		})

		container.RemoveItem(list)
		container.RemoveItem(footer)
		container.AddItem(input, 1, 0, true)
		container.AddItem(footer, 1, 0, false)
		footer.SetText(" Enter save · Esc cancel ")
		a.currentFocus = "analyzer_rules_add"
		a.SetFocus(input)
	}

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch {
		case ev.Key() == tcell.KeyEscape:
			closePicker()
			return nil
		case ev.Rune() == 'a':
			showAddInput()
			return nil
		case ev.Rune() == 'd':
			idx := list.GetCurrentItem()
			if idx >= 0 && idx < len(rules) {
				id := rules[idx].ID
				go func() {
					if err := svc.DeleteRule(a.ctx, id); err != nil {
						a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Delete failed: %v", err))
						return
					}
					a.QueueUpdateDraw(reload)
					a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule deleted")
				}()
			}
			return nil
		}
		return ev
	})

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.setActivePicker(PickerAnalyzerRules)
	a.currentFocus = "analyzer_rules"
	a.SetFocus(list)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestAnalyzerRulesPickerSwap -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan_rules.go internal/tui/action_plan_rules_test.go
git commit -m "feat(tui): analyzer rules manager as in-place side-panel picker"
```

---

### Task 3: Ctrl+R quick-remember → embedded input in the Action Plan

**Files:**
- Modify: `internal/tui/action_plan_rules.go` (rewrite `showRememberRuleModal`)
- Test: `internal/tui/action_plan_rules_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_rules_test.go`:

```go
func TestRememberRuleInlineSwap(t *testing.T) {
	a := newRulesTestApp()

	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1"}},
		}},
		excluded: map[string]bool{},
		expanded: map[int]bool{},
		metaByID: nil,
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	a.showRememberRuleModal("always archive promos")
	if a.currentFocus != "action_plan_rule" {
		t.Fatalf("expected currentFocus=action_plan_rule, got %q", a.currentFocus)
	}
	// Tree swapped out for the input.
	if state.container.ItemAt(0) == state.tree {
		t.Fatal("tree should be swapped out while the remember-rule input is shown")
	}
	input, ok := a.GetFocus().(*tview.InputField)
	if !ok {
		t.Fatalf("expected the remember-rule input focused, got %T", a.GetFocus())
	}
	if input.GetText() != "always archive promos" {
		t.Fatalf("input should be pre-seeded with the suggestion, got %q", input.GetText())
	}

	// Esc restores the tree.
	if done := input.GetInputCapture(); done != nil {
		done(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	}
	if a.currentFocus != "action_plan" {
		t.Fatalf("after Esc, currentFocus should be action_plan, got %q", a.currentFocus)
	}
	if a.actionPlanState.container.ItemAt(0) != state.tree {
		t.Fatal("after Esc, the tree should be restored as the container body")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRememberRuleInlineSwap -v`
Expected: FAIL — current `showRememberRuleModal` uses a floating page; `currentFocus`/container swap are not as asserted.

- [ ] **Step 3: Rewrite `showRememberRuleModal`**

In `internal/tui/action_plan_rules.go`, replace the entire `showRememberRuleModal` function with:

```go
// showRememberRuleModal body-swaps the Action Plan tree for a pre-seeded input field inside
// the panel container (the showActionPlanMoveChooser pattern), so saving a preference reads as
// deeper navigation rather than a floating modal. Enter saves; Esc returns to the tree.
func (a *App) showRememberRuleModal(suggestion string) {
	svc := a.GetAnalyzerRulesService()
	if svc == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "Rules unavailable — check account/DB")
		return
	}
	state := a.actionPlanState
	if state == nil {
		return
	}
	colors := a.GetComponentColors("ai")

	input := tview.NewInputField().SetLabel(" Rule: ").SetText(suggestion).SetFieldWidth(0)
	input.SetBackgroundColor(colors.Background.Color())
	input.SetFieldBackgroundColor(colors.Background.Color())
	input.SetFieldTextColor(colors.Text.Color())

	restore := func() {
		state.container.RemoveItem(input)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := input.GetText()
			restore()
			go func() {
				if err := svc.SaveRule(a.ctx, text); err != nil {
					a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Could not save rule: %v", err))
					return
				}
				a.GetErrorHandler().ShowSuccess(a.ctx, "✓ Rule saved — applies on next analysis")
			}()
			return
		}
		restore() // Esc / Tab
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(input, 1, 0, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(" 🧠 Remember preference ")
	state.footer.SetText(" Enter save · Esc cancel ")
	a.currentFocus = "action_plan_rule"
	a.SetFocus(input)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestRememberRuleInlineSwap -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan_rules.go internal/tui/action_plan_rules_test.go
git commit -m "feat(tui): Ctrl+R remember-rule as in-place input in the Action Plan"
```

---

### Task 4: Remove dead floating-modal code

**Files:**
- Modify: `internal/tui/keys.go` (remove the `HasPage` guard)
- Modify: `internal/tui/action_plan.go` (remove `RemovePage` calls)
- Modify: `internal/tui/action_plan_rules.go` (remove unused page constants)

- [ ] **Step 1: Remove the dead keys.go guard**

In `internal/tui/keys.go`, find and delete this block (inside the `if a.isActionPlanActive()` section):

```go
			// A rule/move overlay open on top owns all keys (its own Esc closes it).
			if a.Pages.HasPage(actionPlanRulePage) || a.Pages.HasPage(analyzerRulesPage) {
				return event
			}
```

(The new `currentFocus` sentinels added in Task 1 now cover these panels.)

- [ ] **Step 2: Remove the RemovePage calls in closeActionPlanPanel**

In `internal/tui/action_plan.go`, delete these lines (and their preceding comment) in `closeActionPlanPanel`:

```go
	// Defensively tear down any rule overlay still on the Pages stack, so closing the
	// panel can never leave an orphaned modal that re-captures focus. RemovePage is a
	// no-op when the page is absent.
	a.Pages.RemovePage(actionPlanRulePage)
	a.Pages.RemovePage(analyzerRulesPage)
```

- [ ] **Step 3: Remove the now-unused page constants**

In `internal/tui/action_plan_rules.go`, delete:

```go
const actionPlanRulePage = "actionPlanRule"
const analyzerRulesPage = "analyzerRules"
```

- [ ] **Step 4: Build (catches any remaining reference)**

Run: `go build ./...`
Expected: success. If the build reports `actionPlanRulePage`/`analyzerRulesPage` still used somewhere, grep for it (`grep -rn "actionPlanRulePage\|analyzerRulesPage" internal/tui/`) and remove that reference too.

- [ ] **Step 5: Run the full tui test package**

Run: `go test ./internal/tui/ 2>&1 | tail -3`
Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/keys.go internal/tui/action_plan.go internal/tui/action_plan_rules.go
git commit -m "refactor(tui): drop dead floating-modal page code for analyzer rules"
```

---

### Task 5: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass. Fix any issue and re-run.

- [ ] **Step 2: Race detector on the new panels**

Run: `go test -race ./internal/tui/ -run 'AnalyzerRules|RememberRule' -timeout 120s`
Expected: `ok` (no data races in the save/delete goroutines + QueueUpdateDraw).

- [ ] **Step 3: Build the binary**

Run: `make build`
Expected: success; binary reports the current version.

(Live tmux E2E — `:plan rules` opens/closes in place; `a` adds, `d` deletes, Esc closes; `Ctrl+R` in the Action Plan swaps in an input and Esc/Enter restores the tree — is deferred to the user's end-of-queue E2E sweep.)

---

## Self-review notes

- **Spec coverage:** PickerAnalyzerRules + keys sentinels (Task 1), manager as side-panel picker + embedded add (Task 2), Ctrl+R in-place body-swap (Task 3), dead-code removal of the guard/RemovePage/constants (Task 4), verification + race (Task 5). All spec sections mapped.
- **Type consistency:** `openAnalyzerRulesManager()` and `showRememberRuleModal(suggestion string)` keep their existing signatures (callers in `commands.go` / `action_plan.go` unchanged). `currentFocus` values `"analyzer_rules"`, `"analyzer_rules_add"`, `"action_plan_rule"` match between Task 1 (keys.go) and Tasks 2–3. `PickerAnalyzerRules` matches between Task 1 and Task 2.
- **No placeholders:** every code step shows full code; commands have expected output.
- **Threading:** all `ErrorHandler.Show*` calls are `go`-dispatched; reloads after save/delete use `QueueUpdateDraw` from the worker goroutine; Esc/close paths are synchronous swaps (no `QueueUpdateDraw`), matching the move chooser.
