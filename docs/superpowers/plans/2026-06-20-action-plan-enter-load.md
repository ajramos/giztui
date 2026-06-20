# Action Plan Enter-to-Load Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pressing Enter on an email node in the Action Plan selects that email in the inbox list and loads it into the reader (focus stays on the Action Plan, so Tab reaches the reader).

**Architecture:** TUI-only. Reuse `(*tview.Table).Select` (which fires the list's existing `SetSelectionChangedFunc` → `showMessageWithoutFocus`) plus a direct `showMessageWithoutFocus(id)` safety-net. A pure `messageRowInList` helper maps a message ID to its table row. No new service/fetch logic.

**Tech Stack:** Go, derailed/tview (Table + TreeView), existing `App.showMessageWithoutFocus`.

---

## Reference facts (verified in code)

- `internal/tui/action_plan.go:68` — `type emailRef struct { catIndex int; msgID string }`. Tree nodes carry either an `int` (category/read-manually) or an `emailRef` (email) as their reference.
- `internal/tui/action_plan.go:673` — the `KeyEnter, KeyRight` case in `actionPlanInputCapture`. Today it only handles the `int` (category) case (toggle expand/collapse); an `emailRef` falls through and the key is consumed (no-op).
- `internal/tui/keys.go:1465` — the inbox list's `SetSelectionChangedFunc` calls `go a.showMessageWithoutFocus(id)` and sets a `Message N/M` status when the table selection changes. Table row = message index + 1 (row 0 is the header).
- `internal/tui/messages.go:2416` — `func (a *App) showMessageWithoutFocus(id string)` loads content in a background goroutine via cache/preloader, **without** changing focus and without setting `currentMessageID`. It guards against stale updates by captured ID.
- `internal/tui/action_plan.go:588` — the email-node footer string (`onCategory=false`). `actionPlanFooterText(onCategory, key, action, count, keys)` + `actionPlanFooterKeys{viewPrompt, remember, move, skip}`. Tested by `TestActionPlanFooterText` (`action_plan_test.go:89`).
- Tests construct `&App{}` directly (e.g. `action_plan_test.go:77`). `a.ids` is a plain `[]string` field read directly in the event-loop handlers (e.g. `keys.go:1461`).
- `a.views["list"]` is a `*tview.Table`.

---

## Task 1: Pure `messageRowInList` helper

**Files:**
- Modify: `internal/tui/action_plan.go`
- Test: `internal/tui/action_plan_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/action_plan_test.go`:

```go
func TestMessageRowInList(t *testing.T) {
	ids := []string{"a", "b", "c"}
	if row, ok := messageRowInList(ids, "a"); !ok || row != 1 {
		t.Errorf("'a' → row=%d ok=%v, want 1/true (header is row 0)", row, ok)
	}
	if row, ok := messageRowInList(ids, "c"); !ok || row != 3 {
		t.Errorf("'c' → row=%d ok=%v, want 3/true", row, ok)
	}
	if _, ok := messageRowInList(ids, "z"); ok {
		t.Error("absent id should return ok=false")
	}
	if _, ok := messageRowInList(nil, "a"); ok {
		t.Error("empty list should return ok=false")
	}
	if _, ok := messageRowInList(ids, ""); ok {
		t.Error("empty msgID should return ok=false")
	}
}
```

- [ ] **Step 2: Run the test, verify it FAILS to compile**

Run: `go test ./internal/tui/ -run TestMessageRowInList -v`
Expected: FAIL — `messageRowInList` undefined.

- [ ] **Step 3: Implement the helper**

Add to `internal/tui/action_plan.go` (near the other small helpers like `checkedIDs`, ~line 94):

```go
// messageRowInList returns the inbox table row for a message ID (table row = index + 1 because
// row 0 is the header). ok is false when the id is empty or not in the list.
func messageRowInList(ids []string, msgID string) (row int, ok bool) {
	if msgID == "" {
		return 0, false
	}
	for i, id := range ids {
		if id == msgID {
			return i + 1, true
		}
	}
	return 0, false
}
```

- [ ] **Step 4: Run the test, verify it PASSES**

Run: `go test ./internal/tui/ -run TestMessageRowInList -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_test.go
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): add messageRowInList helper for action-plan enter-to-load"
```

---

## Task 2: Wire Enter on an email node

**Files:**
- Modify: `internal/tui/action_plan.go` (the `KeyEnter, KeyRight` case ~line 673; add `openActionPlanEmail` method)

- [ ] **Step 1: Add the `openActionPlanEmail` method**

Add to `internal/tui/action_plan.go` (near `messageRowInList` from Task 1):

```go
// openActionPlanEmail selects the email in the inbox list (when still present) and loads it into the
// reader, WITHOUT moving focus — the user presses Tab to read it. If the email is no longer in the
// list (e.g. archived/moved from the Action Plan), it is loaded directly by id. showMessageWithoutFocus
// is idempotent (captured-ID guard + cache), so calling it after Select() is a harmless safety net
// against tview not firing SelectionChangedFunc on a programmatic Select.
func (a *App) openActionPlanEmail(msgID string) {
	if msgID == "" {
		return
	}
	if row, ok := messageRowInList(a.ids, msgID); ok {
		if list, listOK := a.views["list"].(*tview.Table); listOK {
			list.Select(row, 0)
		}
	}
	go a.showMessageWithoutFocus(msgID)
}
```

- [ ] **Step 2: Extend the `KeyEnter, KeyRight` case to handle email nodes**

In `internal/tui/action_plan.go`, replace the current case (~line 673):

```go
		case tcell.KeyEnter, tcell.KeyRight:
			if cur != nil {
				if idx, ok := cur.GetReference().(int); ok { // category / read-manually node
					state.expanded[idx] = !state.expanded[idx]
					cur.SetExpanded(state.expanded[idx])
					a.syncActionPlanNode(state, cur, idx) // refresh chevron + footer
				}
			}
			return nil
```

with:

```go
		case tcell.KeyEnter, tcell.KeyRight:
			if cur != nil {
				switch ref := cur.GetReference().(type) {
				case int: // category / read-manually node
					state.expanded[ref] = !state.expanded[ref]
					cur.SetExpanded(state.expanded[ref])
					a.syncActionPlanNode(state, cur, ref) // refresh chevron + footer
				case emailRef: // email node → load it into the list + reader (focus stays here)
					a.openActionPlanEmail(ref.msgID)
				}
			}
			return nil
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: success.

- [ ] **Step 4: Run the action-plan tests**

Run: `go test ./internal/tui/ -run 'ActionPlan|MessageRowInList' 2>&1 | tail -5`
Expected: `ok` (no failures).

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/action_plan.go
git add internal/tui/action_plan.go
git commit -m "feat(tui): Enter on an Action Plan email loads it into the list + reader (#51)"
```

---

## Task 3: Advertise Enter in the email-node footer

**Files:**
- Modify: `internal/tui/action_plan.go` (`actionPlanFooterText` ~line 588)
- Test: `internal/tui/action_plan_test.go` (`TestActionPlanFooterText` ~line 107)

- [ ] **Step 1: Update the footer test to expect the new hint**

In `internal/tui/action_plan_test.go`, the email-node assertion (~line 107) currently is:

```go
	onEmail := actionPlanFooterText(false, "a", "archive", 7, keys)
	if !strings.Contains(onEmail, "Space to skip") || !strings.Contains(onEmail, "Ctrl+R to remember sender") {
		t.Fatalf("email footer wrong: %q", onEmail)
	}
```

Replace it with:

```go
	onEmail := actionPlanFooterText(false, "a", "archive", 7, keys)
	if !strings.Contains(onEmail, "Enter to open") || !strings.Contains(onEmail, "Space to skip") || !strings.Contains(onEmail, "Ctrl+R to remember sender") {
		t.Fatalf("email footer wrong: %q", onEmail)
	}
```

- [ ] **Step 2: Run the test, verify it FAILS**

Run: `go test ./internal/tui/ -run TestActionPlanFooterText -v`
Expected: FAIL — footer does not contain "Enter to open" yet.

- [ ] **Step 3: Add the hint to the email-node footer**

In `internal/tui/action_plan.go`, the email-node branch of `actionPlanFooterText` (~line 588) currently is:

```go
	return fmt.Sprintf(" %s to skip  |  %s to move  |  %s prompt  |  %s to remember sender  |  Tab to inbox  |  Esc to close ",
		prettyKeyLabel(keys.skip), prettyKeyLabel(keys.move), prettyKeyLabel(keys.viewPrompt), prettyKeyLabel(keys.remember))
```

Replace it with:

```go
	return fmt.Sprintf(" Enter to open  |  %s to skip  |  %s to move  |  %s prompt  |  %s to remember sender  |  Tab to inbox  |  Esc to close ",
		prettyKeyLabel(keys.skip), prettyKeyLabel(keys.move), prettyKeyLabel(keys.viewPrompt), prettyKeyLabel(keys.remember))
```

- [ ] **Step 4: Run the test, verify it PASSES**

Run: `go test ./internal/tui/ -run TestActionPlanFooterText -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_test.go
git add internal/tui/action_plan.go internal/tui/action_plan_test.go
git commit -m "feat(tui): advertise 'Enter to open' in the Action Plan email footer"
```

---

## Task 4: Update `:help` and final verification

**Files:**
- Modify: `internal/tui/app.go` (Action Plan help text, if email-node keys are listed there)

- [ ] **Step 1: Check whether `:help` lists Action Plan email-node keys**

Run: `grep -nE "Action Plan|action.plan|skip|exclude" internal/tui/app.go | grep -iE "help|plan" | head`
- If the `:help` screen documents the Action Plan's per-email keys (Space/exclude, `m` move, `i` prompt), add a matching line for Enter: in the same `fmt.Fprintf(&help, ...)` style and column alignment as the neighbouring Action Plan lines, e.g. a hint that **Enter** opens the highlighted email in the reader (Tab to read).
- If `:help` does NOT enumerate Action Plan per-email keys (they live only in the in-panel footer, already updated in Task 3), make NO change here and note that in the commit/PR — the footer is the source of truth for these contextual keys.

- [ ] **Step 2: If a help line was added, verify build**

Run: `go build ./internal/tui/...`
Expected: success. (Skip if no help change was needed.)

- [ ] **Step 3: Run the canonical pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!`

- [ ] **Step 4: Full test suite**

Run: `make test 2>&1 | grep -E "^(ok|FAIL)" | grep -v "no test files"`
Expected: all `ok`, no `FAIL`.

- [ ] **Step 5: Build the binary**

Run: `make build`
Expected: `Built build/giztui ...`.

- [ ] **Step 6: Commit (only if a help change was made) and finish the branch**

```bash
gofmt -w internal/tui/app.go
git add internal/tui/app.go
git commit -m "docs: document Action Plan Enter-to-open in :help"
```

Then use the superpowers:finishing-a-development-branch skill. Manual E2E on the user's Mac before merge: in the Action Plan, Up/Down to an email, press Enter → it should select in the inbox list and load in the reader; Tab → reader shows the body; Space/category-Enter/`m`/`i` still behave as before; pressing Enter on an email that was archived from the plan still loads it. Do NOT push/merge without explicit user confirmation (project rule: "commit" ≠ "publish").

---

## Self-Review

**Spec coverage:**
- Enter on email node selects in list + loads reader → Task 2 (`openActionPlanEmail` + case). ✓
- Email not in list still loads by id → `openActionPlanEmail` always calls `showMessageWithoutFocus(msgID)`. ✓
- Focus stays on Action Plan → no `SetFocus`; `Select` does not change app focus; `showMessageWithoutFocus` does not steal focus. ✓
- Category-Enter / Space / `m` / `i` unchanged → only the `emailRef` case is added; `int` case preserved verbatim. ✓
- `messageRowInList` unit-tested → Task 1. ✓
- Footer/`:help` parity → Task 3 (footer) + Task 4 (help, conditional). ✓
- `make pre-commit-check` green + manual E2E → Task 4. ✓

**Type/signature consistency:** `messageRowInList(ids []string, msgID string) (row int, ok bool)` (Task 1) is called identically in `openActionPlanEmail` (Task 2). `openActionPlanEmail(msgID string)` defined and called in Task 2. `emailRef.msgID` matches the struct at action_plan.go:68. `actionPlanFooterText` signature unchanged (only the email-node string body changes). ✓

**Placeholder scan:** no TBD/TODO; every code step shows full before/after. Task 4 Step 1 is a genuine conditional (depends on whether `:help` enumerates these keys) with both branches specified — not a placeholder. ✓
