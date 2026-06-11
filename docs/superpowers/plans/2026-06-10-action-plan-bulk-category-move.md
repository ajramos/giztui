# Action Plan Bulk Category Move — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pressing `m` on a category header (or the "Read manually" group) in the Action Plan moves *all* of that group's emails to a chosen destination, preserving each email's excluded flag.

**Architecture:** Add a pure `applyActionPlanBulkMove` that loops the existing index-safe `applyActionPlanMove` over a pre-collected ID list. Extract the body-swap destination chooser from `showActionPlanMoveInline` into a shared `showActionPlanMoveChooser` so single + bulk paths share focus/restore logic. Branch the `m` key handler on the tree node's reference type.

**Tech Stack:** Go, tview/tcell (derailed forks), existing GizTUI Action Plan TUI.

Spec: `docs/superpowers/specs/2026-06-10-action-plan-bulk-category-move-design.md`

---

### Task 1: `applyActionPlanBulkMove` pure function

**Files:**
- Modify: `internal/tui/action_plan_move.go`
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/action_plan_test.go`:

```go
func TestApplyActionPlanBulkMove(t *testing.T) {
	mk := func(name, action string, ids ...string) services.ActionPlanCategory {
		return services.ActionPlanCategory{Name: name, Action: action, MessageIDs: ids}
	}

	// Category → another category: all IDs move, empty source pruned.
	plan := &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Promos", "archive", "m1", "m2", "m3"),
		mk("Notifs", "mark_read", "m4"),
	}}
	n := applyActionPlanBulkMove(plan, nil, 0, moveTarget{kind: "category", catName: "Notifs"})
	if n != 3 {
		t.Fatalf("want 3 moved, got %d", n)
	}
	if categoryIndexByName(plan, "Promos") != -1 {
		t.Fatal("empty source Promos should be pruned")
	}
	if ni := categoryIndexByName(plan, "Notifs"); ni < 0 || len(plan.Categories[ni].MessageIDs) != 4 {
		t.Fatalf("Notifs should hold 4, got %+v", plan.Categories)
	}

	// Category → action (trash), no existing trash group → new category gets all IDs.
	plan = &services.ActionPlan{Categories: []services.ActionPlanCategory{
		mk("Promos", "archive", "a1", "a2"),
	}}
	n = applyActionPlanBulkMove(plan, nil, 0, moveTarget{kind: "action", action: "trash"})
	if n != 2 {
		t.Fatalf("want 2 moved, got %d", n)
	}
	idx := firstCategoryWithAction(plan, "trash")
	if idx < 0 || len(plan.Categories[idx].MessageIDs) != 2 {
		t.Fatalf("trash group should hold 2, got %+v", plan.Categories)
	}

	// Read manually (-1) → a category: all ReadManually IDs move, ReadManually emptied.
	plan = &services.ActionPlan{
		Categories:   []services.ActionPlanCategory{mk("Notifs", "mark_read", "k1")},
		ReadManually: []services.AnalyzerMessage{{ID: "r1"}, {ID: "r2"}},
	}
	n = applyActionPlanBulkMove(plan, nil, -1, moveTarget{kind: "category", catName: "Notifs"})
	if n != 2 {
		t.Fatalf("want 2 moved from ReadManually, got %d", n)
	}
	if len(plan.ReadManually) != 0 {
		t.Fatalf("ReadManually should be empty, got %+v", plan.ReadManually)
	}
	if ni := categoryIndexByName(plan, "Notifs"); ni < 0 || len(plan.Categories[ni].MessageIDs) != 3 {
		t.Fatalf("Notifs should hold 3, got %+v", plan.Categories)
	}

	// Out-of-range source: no-op, returns 0.
	plan = &services.ActionPlan{Categories: []services.ActionPlanCategory{mk("Notifs", "mark_read", "x1")}}
	if got := applyActionPlanBulkMove(plan, nil, 9, moveTarget{kind: "action", action: "trash"}); got != 0 {
		t.Fatalf("out-of-range source should move 0, got %d", got)
	}
	if len(plan.Categories) != 1 || len(plan.Categories[0].MessageIDs) != 1 {
		t.Fatalf("plan should be unchanged, got %+v", plan.Categories)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestApplyActionPlanBulkMove -v`
Expected: FAIL — `undefined: applyActionPlanBulkMove`.

- [ ] **Step 3: Write the implementation**

Append to `internal/tui/action_plan_move.go` (after `applyActionPlanMove`):

```go
// applyActionPlanBulkMove reassigns every message in the source group (a category by index,
// or ReadManually when srcCatIdx == -1) to target, returning the number moved. It loops the
// index-safe applyActionPlanMove over a pre-collected ID list, so mid-loop pruning/re-indexing
// is harmless (targets resolve by name/action). It does NOT touch any excluded map: msgIDs are
// unchanged, so the caller's state.excluded keys still apply to the moved emails.
func applyActionPlanBulkMove(plan *services.ActionPlan, metaByID map[string]*gmailapi.Message, srcCatIdx int, target moveTarget) int {
	var ids []string
	switch {
	case srcCatIdx == -1:
		for _, m := range plan.ReadManually {
			ids = append(ids, m.ID)
		}
	case srcCatIdx >= 0 && srcCatIdx < len(plan.Categories):
		ids = append(ids, plan.Categories[srcCatIdx].MessageIDs...)
	}
	for _, id := range ids {
		applyActionPlanMove(plan, metaByID, id, target)
	}
	return len(ids)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestApplyActionPlanBulkMove -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/action_plan_move.go internal/tui/action_plan_test.go
git commit -m "feat(action-plan): applyActionPlanBulkMove reassigns a whole group"
```

---

### Task 2: Extract shared `showActionPlanMoveChooser` helper

**Files:**
- Modify: `internal/tui/action_plan_move.go:145-217` (the current `showActionPlanMoveInline`)
- Test: `internal/tui/action_plan_test.go` (existing `TestActionPlanMoveInlineSwap` is the safety net)

This is a behavior-preserving refactor: the existing `TestActionPlanMoveInlineSwap` must stay green. `showActionPlanMoveInline` keeps its exact signature `(state, srcCatIdx, msgID)`.

- [ ] **Step 1: Replace `showActionPlanMoveInline` with helper + thin caller**

In `internal/tui/action_plan_move.go`, replace the entire `showActionPlanMoveInline` function (currently lines 145-217) with these two functions:

```go
// showActionPlanMoveChooser swaps the panel's tree for a destination List inside the SAME
// container (body-swap), shared by single-email and whole-group move. srcCatName is excluded
// from the destination list; title is the panel title while choosing; onChosen runs the actual
// reassignment + feedback. Esc returns to the tree. Staying inside the panel's focus avoids the
// global-capture key swallow (see currentFocus="action_plan_move" pass-through in keys.go).
func (a *App) showActionPlanMoveChooser(state *actionPlanState, srcCatName, title string, onChosen func(target moveTarget)) {
	colors := a.GetComponentColors("ai")
	targets := actionPlanMoveTargets(state.plan, srcCatName)

	list := tview.NewList().ShowSecondaryText(false)
	list.SetBackgroundColor(colors.Background.Color())
	list.SetMainTextColor(colors.Text.Color())
	for _, tg := range targets {
		list.AddItem(tg.label, "", 0, nil)
	}

	restore := func() {
		state.container.RemoveItem(list)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state) // restores title, footer and selection from the tree
	}

	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx < 0 || idx >= len(targets) {
			restore()
			return
		}
		if state.plan == nil || a.actionPlanState != state {
			restore()
			return
		}
		onChosen(targets[idx])
		state.selectedMsgID = "" // land the cursor on a category header after the move
		restore()
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
	state.container.SetTitle(title)
	state.footer.SetText(" Enter to move  |  Esc to go back ")
	a.currentFocus = "action_plan_move"
	a.SetFocus(list)
}

// showActionPlanMoveInline opens the destination chooser for a SINGLE email. srcCatIdx is the
// email's current category (-1 for read-manually). Enter reassigns (the action runs at dispatch).
func (a *App) showActionPlanMoveInline(state *actionPlanState, srcCatIdx int, msgID string) {
	srcCatName := ""
	if srcCatIdx >= 0 && srcCatIdx < len(state.plan.Categories) {
		srcCatName = state.plan.Categories[srcCatIdx].Name
	}

	subj := "message"
	if m := state.metaByID[msgID]; m != nil {
		if s := extractHeaderValue(m, "Subject"); s != "" {
			subj = s
		}
	}
	if utf8.RuneCountInString(subj) > 40 {
		r := []rune(subj)
		subj = string(r[:40]) + "…"
	}

	a.showActionPlanMoveChooser(state, srcCatName, fmt.Sprintf(" ➫ Move %q to ", subj), func(target moveTarget) {
		applyActionPlanMove(state.plan, state.metaByID, msgID, target)
		delete(state.excluded, msgID) // a deliberately-placed message starts checked
		// ErrorHandler.ShowSuccess wraps QueueUpdateDraw; calling it synchronously on the UI
		// goroutine would deadlock — dispatch off-thread.
		go a.GetErrorHandler().ShowSuccess(a.ctx,
			fmt.Sprintf("Moved to %s — applies when you dispatch that group", target.label))
	})
}
```

- [ ] **Step 2: Run the existing swap test + the bulk test + build**

Run: `go build ./... && go test ./internal/tui/ -run 'TestActionPlanMoveInlineSwap|TestApplyActionPlanMove' -v`
Expected: PASS (behavior unchanged; `TestActionPlanMoveInlineSwap` still swaps and Esc-restores).

- [ ] **Step 3: Commit**

```bash
git add internal/tui/action_plan_move.go
git commit -m "refactor(action-plan): extract shared showActionPlanMoveChooser"
```

---

### Task 3: `showActionPlanBulkMoveInline`

**Files:**
- Modify: `internal/tui/action_plan_move.go`

- [ ] **Step 1: Add the bulk inline opener**

Append to `internal/tui/action_plan_move.go` (after `showActionPlanMoveInline`):

```go
// showActionPlanBulkMoveInline opens the destination chooser for a WHOLE group: a category by
// index, or the read-manually pile when srcCatIdx == -1. Excluded flags are preserved across the
// move (applyActionPlanBulkMove leaves state.excluded untouched). Empty groups are a no-op.
func (a *App) showActionPlanBulkMoveInline(state *actionPlanState, srcCatIdx int) {
	srcName, count := "Read manually", 0
	if srcCatIdx == -1 {
		count = len(state.plan.ReadManually)
	} else if srcCatIdx >= 0 && srcCatIdx < len(state.plan.Categories) {
		srcName = state.plan.Categories[srcCatIdx].Name
		count = len(state.plan.Categories[srcCatIdx].MessageIDs)
	}
	if count == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "This group is empty — nothing to move")
		return
	}

	// Exclude a real category from its own destination list; read-manually (-1) excludes nothing.
	srcCatName := ""
	if srcCatIdx >= 0 {
		srcCatName = srcName
	}

	a.showActionPlanMoveChooser(state, srcCatName, fmt.Sprintf(" ➫ Move %q (%d) to ", srcName, count), func(target moveTarget) {
		n := applyActionPlanBulkMove(state.plan, state.metaByID, srcCatIdx, target)
		// Excluded flags are intentionally preserved (msgIDs unchanged).
		go a.GetErrorHandler().ShowSuccess(a.ctx,
			fmt.Sprintf("Moved %d emails to %s — applies when you dispatch that group", n, target.label))
	})
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success (function is unused until Task 4 — Go allows unused methods, unlike unused locals).

- [ ] **Step 3: Commit**

```bash
git add internal/tui/action_plan_move.go
git commit -m "feat(action-plan): showActionPlanBulkMoveInline group move chooser"
```

---

### Task 4: Wire the `m` key to branch single vs bulk

**Files:**
- Modify: `internal/tui/action_plan.go:688-696`

- [ ] **Step 1: Replace the `m` handler with a reference type switch**

In `internal/tui/action_plan.go`, replace this block (currently lines 688-696):

```go
		// 'm' on an email node opens the move/recategorize picker.
		if ev.Rune() == 'm' {
			if cur != nil {
				if ref, ok := cur.GetReference().(emailRef); ok {
					a.showActionPlanMoveInline(state, ref.catIndex, ref.msgID)
					return nil
				}
			}
		}
```

with:

```go
		// 'm' moves: on an email node, that one email; on a category or read-manually header,
		// the whole group.
		if ev.Rune() == 'm' && cur != nil {
			switch ref := cur.GetReference().(type) {
			case emailRef:
				a.showActionPlanMoveInline(state, ref.catIndex, ref.msgID)
				return nil
			case int:
				a.showActionPlanBulkMoveInline(state, ref)
				return nil
			}
		}
```

- [ ] **Step 2: Build + full tui test run**

Run: `go build ./... && go test ./internal/tui/ -run 'ActionPlan' -v`
Expected: PASS (all Action Plan tests green).

- [ ] **Step 3: Commit**

```bash
git add internal/tui/action_plan.go
git commit -m "feat(action-plan): m on a category header bulk-moves the whole group"
```

---

### Task 5: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Run the project's pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt clean, vet clean, golangci-lint clean, essential tests pass.

- [ ] **Step 2: If anything fails, fix it, re-run, then commit the fix**

```bash
git add -A
git commit -m "fix(action-plan): address pre-commit findings for bulk move"
```

(Skip the commit if Step 1 passed with no changes.)

---

## Self-review notes

- **Spec coverage:** trigger (Task 4), all-emails-preserving-excluded (Task 1 + Task 3 leaving `state.excluded` untouched), Read-manually `-1` (Task 1 + Task 3), shared chooser extraction (Task 2), empty-source warning (Task 3), count in title/toast (Task 3), tests (Task 1). All spec sections mapped.
- **Type consistency:** `applyActionPlanBulkMove(plan, metaByID, srcCatIdx, target) int`, `showActionPlanMoveChooser(state, srcCatName, title, onChosen)`, `showActionPlanBulkMoveInline(state, srcCatIdx)`, `showActionPlanMoveInline(state, srcCatIdx, msgID)` — names match across all tasks and the `m` handler.
- **No placeholders:** every code step shows full code; commands have expected output.
