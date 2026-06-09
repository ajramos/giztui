# In-Place Panels + Action Plan Correctness Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace two floating-modal overlays (prompt preview, action-plan move) with in-panel body-swaps, fix the move-picker deadlock and the action-plan tree footer desync.

**Architecture:** Each in-place panel swaps the body primitive (list/tree) inside the existing side-panel `Flex` container, keeping border/title/footer; a focus-specific pass-through in `keys.go` lets the swapped-in widget own its keys. Tree selection becomes node-derived (single source of truth).

**Tech Stack:** Go, `github.com/derailed/tview`, `github.com/derailed/tcell/v2`. Spec: `docs/superpowers/specs/2026-06-09-inplace-panels-and-action-plan-fixes-design.md`.

---

## File Structure

- `internal/tui/action_plan.go` — tree build/restore + new `syncSelectionToNode` (selection source-of-truth). Modify.
- `internal/tui/action_plan_move.go` — in-panel destination chooser (body-swap), deadlock fix. Rewrite the picker; keep pure helpers.
- `internal/tui/prompt_preview.go` — in-panel preview swap helper (`showPromptPreviewInline`); remove floating overlay. Rewrite.
- `internal/tui/prompts.go` — repoint preview trigger to the inline helper. Modify.
- `internal/tui/bulk_prompts.go` — mirror inline preview for parity. Modify.
- `internal/tui/keys.go` — focus pass-through for `"prompt_preview"`/`"action_plan_move"`; drop `actionPlanMovePage` guard. Modify.
- `internal/tui/action_plan_test.go` — add `TestSyncSelectionToNode`, `TestActionPlanMoveInlineSwap`. Modify.

---

## Task 1: Tree selection source-of-truth (`syncSelectionToNode`)

**Files:**
- Modify: `internal/tui/action_plan.go` (add helper; rewire `rebuildActionPlanTree` restore sites ~lines 444/455/466/470)
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_test.go`:

```go
func TestSyncSelectionToNode(t *testing.T) {
	a := &App{}
	a.Keys.Archive = "a"
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1"}},
		}},
		excluded: map[string]bool{},
		expanded: map[int]bool{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)

	catNode := tview.NewTreeNode("Promos")
	catNode.SetReference(0)
	emailNode := tview.NewTreeNode("m1")
	emailNode.SetReference(emailRef{catIndex: 0, msgID: "m1"})
	catNode.AddChild(emailNode)
	state.root.AddChild(catNode)

	// Landing on an email node selects that email.
	a.syncSelectionToNode(state, emailNode)
	if state.selectedMsgID != "m1" || state.selectedCategory != 0 {
		t.Fatalf("email node: got msgID=%q cat=%d", state.selectedMsgID, state.selectedCategory)
	}

	// Landing on a category node MUST clear selectedMsgID (the desync bug).
	a.syncSelectionToNode(state, catNode)
	if state.selectedMsgID != "" {
		t.Fatalf("category node must clear selectedMsgID, got %q", state.selectedMsgID)
	}
	if state.selectedCategory != 0 {
		t.Fatalf("category node: got cat=%d", state.selectedCategory)
	}
}
```

Requires `tview` import in the test file (add `"github.com/derailed/tview"` to the import block if absent).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestSyncSelectionToNode`
Expected: FAIL — `a.syncSelectionToNode undefined`.

- [ ] **Step 3: Add the helper**

In `internal/tui/action_plan.go`, immediately after `syncActionPlanNode` (around line 396) add:

```go
// syncSelectionToNode makes node the current node AND derives the selection state
// (selectedCategory/selectedMsgID) from its reference, then refreshes the footer.
// SetCurrentNode does NOT fire SetChangedFunc, so callers that relocate the cursor
// programmatically (e.g. rebuildActionPlanTree) must use this to keep state, cursor,
// and footer in lockstep — the node reference is the single source of truth.
func (a *App) syncSelectionToNode(state *actionPlanState, node *tview.TreeNode) {
	if state == nil || state.tree == nil || node == nil {
		return
	}
	state.tree.SetCurrentNode(node)
	switch ref := node.GetReference().(type) {
	case int:
		state.selectedCategory = ref
		state.selectedMsgID = ""
	case emailRef:
		state.selectedCategory = ref.catIndex
		state.selectedMsgID = ref.msgID
	}
	a.updateActionPlanFooter(state)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestSyncSelectionToNode`
Expected: PASS.

- [ ] **Step 5: Rewire the three restore sites in `rebuildActionPlanTree`**

In `internal/tui/action_plan.go`, replace the bare cursor placements (the email-restore, category-restore, and fallback) with the helper:

Email restore (was `state.tree.SetCurrentNode(child)` then `return`):
```go
				if ref, ok := child.GetReference().(emailRef); ok && ref.msgID == state.selectedMsgID {
					a.syncSelectionToNode(state, child)
					return
				}
```
Category restore (was `state.tree.SetCurrentNode(n)` then `return`):
```go
		if ref, ok := n.GetReference().(int); ok && ref == state.selectedCategory {
			a.syncSelectionToNode(state, n)
			return
		}
```
Fallback (was `state.tree.SetCurrentNode(children[0])`):
```go
	a.syncSelectionToNode(state, children[0])
```
Leave the empty-tree `state.tree.SetCurrentNode(state.root)` (line ~444) unchanged — the root carries no selectable reference.

- [ ] **Step 6: Run the full tui test suite**

Run: `go test ./internal/tui/`
Expected: PASS (existing footer/title/move/integration tests unaffected).

- [ ] **Step 7: Commit**

```bash
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "fix(tui): derive action plan selection from current node (footer desync)"
```

---

## Task 2: keys.go focus pass-through for in-place panels

**Files:**
- Modify: `internal/tui/keys.go` (add pass-through after the composition-panel check ~line 567; drop `actionPlanMovePage` from the action-plan `HasPage` guard ~line 574)

- [ ] **Step 1: Add the pass-through guard**

In `internal/tui/keys.go`, immediately AFTER the composition-panel block (the one ending `return event` near line 567) and BEFORE the Action Plan block (`if a.isActionPlanActive()`), insert:

```go
		// In-place panels that live inside a picker body and own ALL keys via their own
		// input capture: the prompt preview (a TextView) and the action-plan move chooser
		// (a List). The global capture runs before a focused widget's capture, so without
		// this pass-through it would swallow their Esc/Ctrl+P/Enter (see prompt-preview bug).
		if a.currentFocus == "prompt_preview" || a.currentFocus == "action_plan_move" {
			return event
		}
```

- [ ] **Step 2: Drop the dead `actionPlanMovePage` guard**

In the Action Plan block, change:
```go
			if a.Pages.HasPage(actionPlanRulePage) || a.Pages.HasPage(analyzerRulesPage) || a.Pages.HasPage(actionPlanMovePage) {
```
to:
```go
			if a.Pages.HasPage(actionPlanRulePage) || a.Pages.HasPage(analyzerRulesPage) {
```
(The move chooser is no longer a Page after Task 5; `actionPlanMovePage` will be removed.)

- [ ] **Step 3: Verify it still builds**

Run: `go build ./internal/tui/`
Expected: builds (the `actionPlanMovePage` const still exists until Task 5; this step only removes its use here).

- [ ] **Step 4: Commit**

```bash
git add internal/tui/keys.go
git commit -m "feat(tui): route keys to in-place preview/move panels via focus pass-through"
```

---

## Task 3: Prompt preview in-place (single picker)

**Files:**
- Rewrite: `internal/tui/prompt_preview.go` (replace floating overlay with `showPromptPreviewInline`)
- Modify: `internal/tui/prompts.go` (remove old `previewHighlightedPrompt`; trigger inline helper from input/list captures)

- [ ] **Step 1: Replace `prompt_preview.go` with the inline helper**

Replace the entire body of `internal/tui/prompt_preview.go` (keep `promptPreviewCreateNewHint` and `promptPreviewText`; remove `promptPreviewPage`, `closePromptPreview`, `showPromptPreview`):

```go
package tui

import (
	"fmt"
	"strings"

	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// promptPreviewCreateNewHint is shown when the highlighted row is the
// "✨ Create new with AI…" entry rather than a real prompt.
const promptPreviewCreateNewHint = "Opens the AI prompt configurator to create a new prompt."

// promptPreviewText builds the preview body: a Description block followed by the
// full Template. Empty fields become explicit placeholders so it is never blank.
func promptPreviewText(description, promptText string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "(no description)"
	}
	tmpl := strings.TrimSpace(promptText)
	if tmpl == "" {
		tmpl = "(empty template)"
	}
	return fmt.Sprintf("Description:\n%s\n\nTemplate:\n%s", desc, tmpl)
}

// showPromptPreviewInline swaps the picker's list for a scrollable preview inside the
// SAME container (search input stays on top), so the preview reads as deeper navigation
// rather than a floating modal. Enter runs onApply (apply prompt / open configurator);
// Esc or Ctrl+P restores the list. While shown, currentFocus is "prompt_preview" so
// keys.go passes keys straight to the preview's own capture.
func (a *App) showPromptPreviewInline(container *tview.Flex, list *tview.List, footer *tview.TextView, footerNormal, name, body string, onApply func()) {
	colors := a.GetComponentColors("prompts")

	tv := tview.NewTextView().SetDynamicColors(false).SetWrap(true).SetText(body)
	tv.SetScrollable(true)
	tv.SetBackgroundColor(colors.Background.Color())
	tv.SetTextColor(colors.Text.Color())

	prevTitle := container.GetTitle()

	restore := func() {
		container.RemoveItem(tv)
		container.RemoveItem(footer)
		container.AddItem(list, 0, 1, true)
		container.AddItem(footer, 1, 0, false)
		container.SetTitle(prevTitle)
		footer.SetText(footerNormal)
		a.currentFocus = "prompts"
		a.SetFocus(list)
	}

	tv.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch {
		case e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyCtrlP:
			restore()
			return nil
		case e.Key() == tcell.KeyEnter:
			onApply() // applies the prompt or opens the configurator; both navigate away
			return nil
		}
		return e
	})

	container.RemoveItem(list)
	container.RemoveItem(footer)
	container.AddItem(tv, 0, 1, true)
	container.AddItem(footer, 1, 0, false)
	container.SetTitle(fmt.Sprintf(" 👁 Preview: %s ", name))
	footer.SetText(" Enter to apply  |  Esc/Ctrl+P to go back ")
	a.currentFocus = "prompt_preview"
	a.SetFocus(tv)
}
```

- [ ] **Step 2: Remove the old standalone preview trigger in `prompts.go`**

Delete the `previewHighlightedPrompt := func(...) {...}` block (lines ~76-85). The `visible`/`all` declarations above it stay.

- [ ] **Step 3: Define the inline trigger after the container is built**

In `prompts.go`, inside the `QueueUpdateDraw` block, AFTER the footer is created and added (~line 240, right before the `list.SetInputCapture` at line 243), add:

```go
			triggerPreview := func() {
				isCreateNew, vi := promptPickerSelection(list.GetCurrentItem(), len(visible))
				var name, body string
				var onApply func()
				if isCreateNew {
					name, body = "Create new with AI", promptPreviewCreateNewHint
					onApply = func() {
						a.closePromptPicker()
						a.openPromptConfigurator(promptConfiguratorContext{mode: "single", messageID: messageID})
					}
				} else {
					v := visible[vi]
					name, body = v.name, promptPreviewText(v.description, v.promptText)
					vid, vname := v.id, v.name
					onApply = func() { go a.applyPromptToMessage(messageID, vid, vname, message) }
				}
				a.showPromptPreviewInline(container, list, footer, " Enter to apply | Esc to cancel ", name, body, onApply)
			}
```

- [ ] **Step 4: Repoint the two capture call sites**

In `prompts.go`, in BOTH the `input.SetInputCapture` (line ~180) and `list.SetInputCapture` (line ~244) handlers, replace:
```go
					previewHighlightedPrompt(list, visible)
```
with:
```go
					triggerPreview()
```

- [ ] **Step 5: Build and run tui tests**

Run: `go build ./internal/tui/ && go test ./internal/tui/`
Expected: builds, tests PASS. (No `showPromptPreview`/`promptPreviewPage` references remain — grep to confirm: `grep -rn "promptPreviewPage\|showPromptPreview\b\|closePromptPreview" internal/tui/` returns nothing.)

- [ ] **Step 6: Commit**

```bash
git add internal/tui/prompt_preview.go internal/tui/prompts.go
git commit -m "feat(tui): in-place prompt preview (Enter applies, Esc/Ctrl+P returns)"
```

---

## Task 4: Prompt preview in-place (bulk picker)

**Files:**
- Modify: `internal/tui/bulk_prompts.go` (mirror Task 3: remove `previewHighlightedBulkPrompt`, trigger inline helper)

- [ ] **Step 1: Remove the old bulk preview trigger**

Delete the `previewHighlightedBulkPrompt := func(...) {...}` block (lines ~74-83).

- [ ] **Step 2: Define the inline trigger after the bulk container/footer are built**

In `bulk_prompts.go`, AFTER `container.AddItem(footer, 1, 0, false)` (~line 277), add:

```go
	triggerPreview := func() {
		isCreateNew, vi := promptPickerSelection(list.GetCurrentItem(), len(visible))
		var name, body string
		var onApply func()
		if isCreateNew {
			name, body = "Create new with AI", promptPreviewCreateNewHint
			onApply = func() {
				messageIDs := make([]string, 0, len(a.selected))
				for id := range a.selected {
					messageIDs = append(messageIDs, id)
				}
				a.closeBulkPromptPicker()
				a.openPromptConfigurator(promptConfiguratorContext{mode: "bulk", messageIDs: messageIDs})
			}
		} else {
			v := visible[vi]
			name, body = v.name, promptPreviewText(v.description, v.promptText)
			vid, vname := v.id, v.name
			onApply = func() { go a.applyBulkPrompt(vid, vname) }
		}
		a.showPromptPreviewInline(container, list, footer, " Enter to apply | Esc to cancel ", name, body, onApply)
	}
```

(Names verified against `bulk_prompts.go`: `applyBulkPrompt(id, name)`, `closeBulkPromptPicker()`, configurator ctx `{mode: "bulk", messageIDs: …}`.)

- [ ] **Step 3: Repoint the two bulk capture call sites**

In `bulk_prompts.go`, replace both `previewHighlightedBulkPrompt(list, visible)` calls (lines ~197 and ~214) with `triggerPreview()`.

- [ ] **Step 4: Build and run tui tests**

Run: `go build ./internal/tui/ && go test ./internal/tui/`
Expected: builds, tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/bulk_prompts.go
git commit -m "feat(tui): in-place bulk prompt preview (parity with single picker)"
```

---

## Task 5: Action plan move in-place (body-swap) + deadlock fix

**Files:**
- Modify: `internal/tui/action_plan_move.go` (replace `openActionPlanMovePicker` Pages overlay with in-panel swap; remove `actionPlanMovePage`; wrap `ShowSuccess` in `go func()`)
- Modify: `internal/tui/action_plan.go` (`closeActionPlanPanel`: drop `RemovePage(actionPlanMovePage)`)
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing swap test**

Append to `internal/tui/action_plan_test.go`:

```go
func TestActionPlanMoveInlineSwap(t *testing.T) {
	a := &App{}
	a.Keys.Archive = "a"
	state := &actionPlanState{
		plan: &services.ActionPlan{Categories: []services.ActionPlanCategory{
			{Name: "Promos", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Notifs", Action: "mark_read", MessageIDs: []string{"m3"}},
		}},
		excluded: map[string]bool{},
		expanded: map[int]bool{0: true},
		metaByID: map[string]*gmailapi.Message{},
		footer:   tview.NewTextView(),
	}
	state.root = tview.NewTreeNode("")
	state.tree = tview.NewTreeView().SetRoot(state.root)
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.AddItem(state.tree, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	a.actionPlanState = state

	// Show the chooser: tree is swapped out, currentFocus flips, container still 2 items.
	a.showActionPlanMoveInline(state, 0, "m2")
	if a.currentFocus != "action_plan_move" {
		t.Fatalf("expected currentFocus=action_plan_move, got %q", a.currentFocus)
	}
	if state.container.GetItemCount() != 2 {
		t.Fatalf("container should hold [chooser, footer], got %d items", state.container.GetItemCount())
	}
	if state.container.GetItem(0) == state.tree {
		t.Fatal("tree should be swapped out while the move chooser is shown")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/tui/ -run TestActionPlanMoveInlineSwap`
Expected: FAIL — `a.showActionPlanMoveInline undefined`.

- [ ] **Step 3: Rewrite the move picker as an in-panel swap**

In `internal/tui/action_plan_move.go`, remove `const actionPlanMovePage = "actionPlanMove"` and replace the whole `openActionPlanMovePicker` function with:

```go
// showActionPlanMoveInline swaps the panel's tree for a destination chooser inside the
// SAME container (body-swap), so recategorizing reads as deeper navigation rather than a
// floating modal — and stays inside the panel's focus, avoiding the global-capture key
// swallow. srcCatIdx is the email's current category (-1 for read-manually). Enter moves
// (reassignment only; the action runs at dispatch); Esc returns to the tree.
func (a *App) showActionPlanMoveInline(state *actionPlanState, srcCatIdx int, msgID string) {
	colors := a.GetComponentColors("ai")

	srcCatName := ""
	if srcCatIdx >= 0 && srcCatIdx < len(state.plan.Categories) {
		srcCatName = state.plan.Categories[srcCatIdx].Name
	}
	targets := actionPlanMoveTargets(state.plan, srcCatName)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())
	for _, tg := range targets {
		list.AddItem(tg.label, "", 0, nil)
	}

	prevTitle := state.container.GetTitle()
	subj := "message"
	if m := state.metaByID[msgID]; m != nil {
		if s := extractHeaderValue(m, "Subject"); s != "" {
			subj = s
		}
	}

	restore := func() {
		state.container.RemoveItem(list)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		state.container.SetTitle(prevTitle)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx < 0 || idx >= len(targets) {
			restore()
			return
		}
		target := targets[idx]
		if state.plan == nil || a.actionPlanState != state {
			restore()
			return
		}
		applyActionPlanMove(state.plan, state.metaByID, msgID, target)
		delete(state.excluded, msgID) // a deliberately-placed message starts checked
		state.selectedMsgID = ""      // land the cursor on a category header after the move
		restore()
		// ErrorHandler.ShowSuccess wraps QueueUpdateDraw; calling it synchronously here (on
		// the UI goroutine) would deadlock — dispatch it off-thread (links.go pattern).
		go a.GetErrorHandler().ShowSuccess(a.ctx,
			fmt.Sprintf("Moved to %s — applies when you dispatch that group", target.label))
	})
	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(list, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(fmt.Sprintf(" ➫ Move %q to ", subj))
	state.footer.SetText(" Enter to move  |  Esc to go back ")
	a.currentFocus = "action_plan_move"
	a.SetFocus(list)
}
```

Confirm the `tview`/`tcell`/`fmt` imports remain used (they are). The pure helpers (`removeID`, `applyActionPlanMove`, `actionPlanMoveTargets`, etc.) are unchanged.

- [ ] **Step 4: Repoint the `m` key handler**

In `internal/tui/action_plan.go` (~line 662), change:
```go
					a.openActionPlanMovePicker(state, ref.catIndex, ref.msgID)
```
to:
```go
					a.showActionPlanMoveInline(state, ref.catIndex, ref.msgID)
```

- [ ] **Step 5: Drop the dead `RemovePage` in `closeActionPlanPanel`**

In `internal/tui/action_plan.go` `closeActionPlanPanel` (~line 536), delete the line:
```go
	a.Pages.RemovePage(actionPlanMovePage)
```

- [ ] **Step 6: Run the swap test + full suite**

Run: `go test ./internal/tui/ -run TestActionPlanMoveInlineSwap` → PASS.
Run: `go test ./internal/tui/` → PASS (existing `TestApplyActionPlanMove` etc. still green).
Confirm no dangling refs: `grep -rn "actionPlanMovePage\|openActionPlanMovePicker" internal/tui/` → nothing.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/action_plan_move.go internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): in-place action plan move chooser + fix ShowSuccess deadlock"
```

---

## Task 6: Final verification gate

**Files:** none (verification only)

- [ ] **Step 1: Full gate**

Run: `go build ./... && go test ./internal/tui/ ./internal/render/ && go vet ./internal/tui/... && golangci-lint run ./internal/tui/...`
Expected: build OK, tests PASS, vet clean, lint `0 issues.`

- [ ] **Step 2: Pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!`

- [ ] **Step 3: Rebuild the binary**

Run: `make build`
Expected: `Built build/giztui …`

- [ ] **Step 4: Hand off for manual E2E**

Report to the user to verify in the running app:
- Prompt picker: `p` → `Ctrl+P` shows preview in-panel; `Enter` applies (and "Create new" opens the configurator); `Esc`/`Ctrl+P` returns to the list. No floating modal; nothing hangs.
- Action plan: `m` on an email shows the destination chooser in-panel; `Enter` moves without hanging and shows the toast; `Esc` returns to the tree.
- Action plan tree: footer always matches the highlighted node (category vs email), during and after analysis, including after a move/space-toggle rebuild.

---

## Self-Review Notes

- **Spec coverage:** A (prompt preview) → Tasks 3+4; B (move + deadlock) → Task 5; C (tree desync) → Task 1; keys.go routing → Task 2; verification → Task 6. All spec sections covered.
- **Type consistency:** `syncSelectionToNode(state, node)`, `showPromptPreviewInline(container, list, footer, footerNormal, name, body, onApply)`, `showActionPlanMoveInline(state, srcCatIdx, msgID)` used consistently across tasks. `currentFocus` sentinels `"prompt_preview"`/`"action_plan_move"` set in Tasks 3/5 and matched in Task 2.
- **Bulk names verified:** `applyBulkPrompt(id, name)`, `closeBulkPromptPicker()`, `promptConfiguratorContext{mode: "bulk", messageIDs: …}` confirmed against `bulk_prompts.go`; `messageIDs` is rebuilt from `a.selected` inside `triggerPreview` (the original is scoped to `reload`).
```
