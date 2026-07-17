# Contextual Key-Help Cheat-Sheet — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Pressing `?` inside the Action Plan panel shows a focused, read-only cheat-sheet of that panel's keys (body-swap, Esc returns); `?` outside a panel keeps showing global help. Reusable base, wired for the Action Plan only.

**Architecture:** A tiny reusable `KeyHint` type + a pure `formatKeyHelp` renderer (no tview). The Action Plan's advertised bindings are consolidated into the existing `actionPlanFooterKeys` struct (extended) so the footer teaser and the cheat-sheet read ONE source. The cheat-sheet is shown by body-swapping the tree for a read-only `TextView`, cloning the proven `showActionPlanPromptView` (`v` prompt viewer) pattern.

**Tech Stack:** Go, tview, existing Action Plan panel infra.

**Branch:** `feat/contextual-key-help` (off `main`). **Spec:** `docs/superpowers/specs/2026-07-17-contextual-key-help-design.md`.

**Conventions (binding):**
- No `Co-Authored-By` / Claude signature in commits.
- Scoped tests only: `go test ./internal/tui/ -count=1`. Never `go test ./...`.
- `gofmt -w` touched files before each commit; `make pre-commit-check` before declaring done.
- ESC/cleanup UI work is SYNCHRONOUS — no `QueueUpdateDraw`, no goroutine, no ErrorHandler in the swap/restore paths (this is pure event-loop UI work, exactly like `showActionPlanPromptView`).
- UI only in `internal/tui`.

**Cross-branch note (important):** This branch is off `main`, which does NOT have the read-manually feature's `g` (assist) / `.` (accept) keys — those live on the unmerged `feat/read-manually-triage`. This plan lists ONLY keys present on `main`. When read-manually merges, whoever merges adds `assist` / `accept` to the extended `actionPlanFooterKeys` and to `actionPlanKeyHints` (one-line each) — they then appear in the cheat-sheet automatically via the single source. Do NOT reference `a.Keys.AssistReadManually` / `a.Keys.AcceptSuggestion` in this plan; they don't compile here.

---

## File Structure

- **Create** `internal/tui/key_help.go` — reusable `KeyHint` type + pure `formatKeyHelp(title string, hints []KeyHint) string`.
- **Create** `internal/tui/key_help_test.go` — formatter unit tests.
- **Create** `internal/tui/action_plan_key_help.go` — `actionPlanKeyHints(keys actionPlanFooterKeys) []KeyHint` (pure) + `showActionPlanKeyHelp(state *actionPlanState)` (body-swap presenter).
- **Create** `internal/tui/action_plan_key_help_test.go` — hints-builder unit test (incl. rebind reflection).
- **Modify** `internal/tui/action_plan.go` — extend `actionPlanFooterKeys` struct + its one population site; wire `?` in the panel input capture.
- **Modify** `internal/tui/app.go` — one `:help` line noting `?`-in-panel shows the panel cheat-sheet.
- **Modify** `docs/KEYBOARD_SHORTCUTS.md` — document it.

---

## Task 1: Reusable `KeyHint` + `formatKeyHelp` (pure)

**Files:**
- Create: `internal/tui/key_help.go`
- Test: `internal/tui/key_help_test.go`

- [ ] **Step 1: Write the failing test** `internal/tui/key_help_test.go`:

```go
package tui

import (
	"strings"
	"testing"
)

func TestFormatKeyHelp(t *testing.T) {
	hints := []KeyHint{
		{Key: "g", Desc: "Assist"},
		{Key: "Ctrl+R", Desc: "Remember rule"},
		{Key: "Esc", Desc: "Close"},
	}
	out := formatKeyHelp("Action Plan", hints)

	// Title present on the first line.
	if !strings.HasPrefix(out, "Action Plan") {
		t.Fatalf("title missing; got:\n%s", out)
	}
	// Every key and description appears.
	for _, h := range hints {
		if !strings.Contains(out, h.Key) || !strings.Contains(out, h.Desc) {
			t.Fatalf("missing %q/%q in:\n%s", h.Key, h.Desc, out)
		}
	}
	// Keys are left-aligned to the same column: the descriptions must line up.
	// "Ctrl+R" is the widest key (6). Find each desc's byte offset on its line; all equal.
	var descCols []int
	for _, line := range strings.Split(out, "\n") {
		for _, h := range hints {
			if strings.Contains(line, h.Desc) {
				descCols = append(descCols, strings.Index(line, h.Desc))
			}
		}
	}
	for i := 1; i < len(descCols); i++ {
		if descCols[i] != descCols[0] {
			t.Fatalf("descriptions not column-aligned: %v\n%s", descCols, out)
		}
	}
}

func TestFormatKeyHelp_EmptyHints(t *testing.T) {
	out := formatKeyHelp("Empty", nil)
	if strings.TrimSpace(out) != "Empty" {
		t.Fatalf("empty hints should render just the title, got %q", out)
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (undefined `KeyHint`/`formatKeyHelp`):

Run: `go test ./internal/tui/ -run TestFormatKeyHelp -count=1`
Expected: build failure `undefined: KeyHint`.

- [ ] **Step 3: Implement** `internal/tui/key_help.go`:

```go
package tui

import "strings"

// KeyHint is one row of a context cheat-sheet: a display-formatted key and what it does.
type KeyHint struct {
	Key  string // already display-formatted, e.g. "Ctrl+R", "Esc", "g"
	Desc string
}

// formatKeyHelp renders a cheat-sheet: the title, a blank line, then one "key  description"
// row per hint with keys left-padded to a common width so descriptions align. An empty hint
// list renders just the title (defensive — for panels that declare none yet).
func formatKeyHelp(title string, hints []KeyHint) string {
	if len(hints) == 0 {
		return title
	}
	width := 0
	for _, h := range hints {
		if len(h.Key) > width {
			width = len(h.Key)
		}
	}
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, h := range hints {
		b.WriteString("  ")
		b.WriteString(h.Key)
		b.WriteString(strings.Repeat(" ", width-len(h.Key)))
		b.WriteString("   ")
		b.WriteString(h.Desc)
		b.WriteString("\n")
	}
	return b.String()
}
```

- [ ] **Step 4: Run — expect PASS**:

Run: `go test ./internal/tui/ -run TestFormatKeyHelp -count=1`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**:

```bash
gofmt -w internal/tui/key_help.go internal/tui/key_help_test.go
git add internal/tui/key_help.go internal/tui/key_help_test.go
git commit -m "feat(tui): reusable KeyHint + formatKeyHelp cheat-sheet formatter"
```

---

## Task 2: Extend `actionPlanFooterKeys` + `actionPlanKeyHints` builder (pure)

**Files:**
- Modify: `internal/tui/action_plan.go` (struct at ~line 728; population site at ~line 755)
- Create: `internal/tui/action_plan_key_help.go`
- Test: `internal/tui/action_plan_key_help_test.go`

- [ ] **Step 1: Extend the struct.** In `internal/tui/action_plan.go`, the struct is:
```go
type actionPlanFooterKeys struct {
	viewPrompt, remember, move, skip string
}
```
Extend it to carry every advertised binding (keep the existing 4, add the action + confirm keys):
```go
type actionPlanFooterKeys struct {
	viewPrompt, remember, move, skip           string
	archive, trash, label, toggleRead, confirm string
}
```

- [ ] **Step 2: Populate the new fields** at the single population site (~line 755, inside `updateActionPlanFooter`), which currently is:
```go
	state.footer.SetText(actionPlanFooterText(onCategory, key, action, count, actionPlanFooterKeys{
		viewPrompt: a.Keys.ViewPrompt,
		remember:   a.Keys.RememberRule,
		move:       a.Keys.Move,
		skip:       a.Keys.BulkSelect,
	}))
```
Change it to build the struct once and reuse it (so it can also feed the cheat-sheet). Replace with:
```go
	fk := actionPlanFooterKeys{
		viewPrompt: a.Keys.ViewPrompt,
		remember:   a.Keys.RememberRule,
		move:       a.Keys.Move,
		skip:       a.Keys.BulkSelect,
		archive:    a.Keys.Archive,
		trash:      a.Keys.Trash,
		label:      a.Keys.ManageLabels,
		toggleRead: a.Keys.ToggleRead,
		confirm:    a.Keys.ConfirmPlan,
	}
	state.footer.SetText(actionPlanFooterText(onCategory, key, action, count, fk))
```
(`actionPlanFooterText` ignores the new fields — no change needed there.)

- [ ] **Step 3: Write the failing test** `internal/tui/action_plan_key_help_test.go`:

```go
package tui

import "testing"

func TestActionPlanKeyHints(t *testing.T) {
	keys := actionPlanFooterKeys{
		viewPrompt: "i", remember: "ctrl+r", move: "m", skip: "space",
		archive: "a", trash: "d", label: "l", toggleRead: "t", confirm: "c",
	}
	hints := actionPlanKeyHints(keys)

	// Spot-check that configured keys are rendered (via prettyKeyLabel) with their descriptions.
	want := map[string]string{
		"Ctrl+R": "", // remember
		"m":      "", // move
		"i":      "", // view prompt
		"c":      "", // confirm
		"Esc":    "", // fixed key
	}
	seen := map[string]bool{}
	for _, h := range hints {
		seen[h.Key] = true
	}
	for k := range want {
		if !seen[k] {
			t.Fatalf("expected key %q in hints, got %+v", k, hints)
		}
	}
}

func TestActionPlanKeyHints_ReflectsRebind(t *testing.T) {
	// Rebinding remember to a different key must change the cheat-sheet (single source of truth).
	base := actionPlanFooterKeys{viewPrompt: "i", remember: "ctrl+r", move: "m", skip: "space",
		archive: "a", trash: "d", label: "l", toggleRead: "t", confirm: "c"}
	rebound := base
	rebound.remember = "x"

	hasKey := func(hs []KeyHint, k string) bool {
		for _, h := range hs {
			if h.Key == k {
				return true
			}
		}
		return false
	}
	if !hasKey(actionPlanKeyHints(base), "Ctrl+R") {
		t.Fatal("base should advertise Ctrl+R for remember")
	}
	if !hasKey(actionPlanKeyHints(rebound), "x") || hasKey(actionPlanKeyHints(rebound), "Ctrl+R") {
		t.Fatal("rebind not reflected in hints")
	}
}
```

- [ ] **Step 4: Run — expect FAIL** (undefined `actionPlanKeyHints`):

Run: `go test ./internal/tui/ -run TestActionPlanKeyHints -count=1`
Expected: build failure.

- [ ] **Step 5: Implement** `actionPlanKeyHints` in `internal/tui/action_plan_key_help.go` (the `showActionPlanKeyHelp` presenter is added in Task 3 — this file starts with the pure builder):

```go
package tui

// actionPlanKeyHints builds the full ordered cheat-sheet for the Action Plan panel from the
// same actionPlanFooterKeys the footer uses, so footer teaser and cheat-sheet never drift.
// Fixed tview keys (arrows/Enter/Tab/Esc) are literals; configured keys use prettyKeyLabel.
func actionPlanKeyHints(keys actionPlanFooterKeys) []KeyHint {
	return []KeyHint{
		{Key: "↑/↓", Desc: "Move between nodes"},
		{Key: "Enter/→", Desc: "Expand category / open email"},
		{Key: "←", Desc: "Collapse category"},
		{Key: prettyKeyLabel(keys.skip), Desc: "Exclude / include email"},
		{Key: prettyKeyLabel(keys.archive), Desc: "Archive the category's checked emails"},
		{Key: prettyKeyLabel(keys.trash), Desc: "Trash the category's checked emails"},
		{Key: prettyKeyLabel(keys.label), Desc: "Apply the category's label"},
		{Key: prettyKeyLabel(keys.toggleRead), Desc: "Mark the category's checked emails read"},
		{Key: prettyKeyLabel(keys.move), Desc: "Move email / category to another label"},
		{Key: prettyKeyLabel(keys.viewPrompt), Desc: "View the effective analyzer prompt"},
		{Key: prettyKeyLabel(keys.remember), Desc: "Remember a rule / interest"},
		{Key: prettyKeyLabel(keys.confirm), Desc: "Confirm & apply the whole plan (two-press)"},
		{Key: "Tab", Desc: "Move focus to the inbox"},
		{Key: "Esc", Desc: "Close the panel"},
	}
}
```

- [ ] **Step 6: Run — expect PASS**, and build:

Run: `go test ./internal/tui/ -run TestActionPlanKeyHints -count=1 && go build ./internal/tui/...`
Expected: PASS (2 tests), BUILD OK.

- [ ] **Step 7: Commit**:

```bash
gofmt -w internal/tui/action_plan.go internal/tui/action_plan_key_help.go internal/tui/action_plan_key_help_test.go
git add internal/tui/action_plan.go internal/tui/action_plan_key_help.go internal/tui/action_plan_key_help_test.go
git commit -m "feat(tui): single-source action-plan key hints builder"
```

---

## Task 3: Cheat-sheet presenter (body-swap) + wire `?` + docs

**Files:**
- Modify: `internal/tui/action_plan_key_help.go` (add the presenter)
- Modify: `internal/tui/action_plan.go` (input capture — add the `?` case)
- Modify: `internal/tui/app.go` (`:help` line)
- Modify: `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Add the presenter** `showActionPlanKeyHelp` in `internal/tui/action_plan_key_help.go`. It is a near-clone of `showActionPlanPromptView` (in `internal/tui/action_plan_prompt.go`) — read that function first and mirror its swap/restore exactly. Add the tview/tcell imports to the file:

```go
package tui

import (
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// showActionPlanKeyHelp body-swaps the Action Plan tree for a read-only cheat-sheet of the
// panel's keys. Esc returns to the tree. Mirrors showActionPlanPromptView (the `v` viewer):
// synchronous event-loop UI work only — no goroutine / QueueUpdateDraw / ErrorHandler.
func (a *App) showActionPlanKeyHelp(state *actionPlanState) {
	if state == nil {
		return
	}
	fk := actionPlanFooterKeys{
		viewPrompt: a.Keys.ViewPrompt,
		remember:   a.Keys.RememberRule,
		move:       a.Keys.Move,
		skip:       a.Keys.BulkSelect,
		archive:    a.Keys.Archive,
		trash:      a.Keys.Trash,
		label:      a.Keys.ManageLabels,
		toggleRead: a.Keys.ToggleRead,
		confirm:    a.Keys.ConfirmPlan,
	}
	text := formatKeyHelp("Action Plan — keys", actionPlanKeyHints(fk))

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(false)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(tview.Escape(text))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.focus.set("action_plan")
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state)
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
	state.container.SetTitle(" ⌨️  Action Plan keys ")
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.focus.set("action_plan_key_help")
	a.SetFocus(view)
}
```

Confirm against `showActionPlanPromptView` that `state.container`, `state.tree`, `state.footer`, `a.focus.set`, `a.renderActionPlanPanel(state)`, and `a.GetComponentColors("ai")` are the real names/signatures (they are, per that file). If `renderActionPlanPanel` has a different name, use whatever that function uses to restore title/footer/selection.

- [ ] **Step 2: Wire `?` in the panel input capture.** In `internal/tui/action_plan.go`, find the tree input-capture block where `a.Keys.ViewPrompt` is handled (~line 919-921):
```go
		if a.matchesConfiguredKey(ev, a.Keys.ViewPrompt) {
			a.showActionPlanPromptView(state)
			...
			return nil
		}
```
Add a sibling case for the help key just before/after it (place it so it doesn't shadow another handled key; the help key is `a.Keys.Help`, default `?`):
```go
		if a.matchesConfiguredKey(ev, a.Keys.Help) {
			a.showActionPlanKeyHelp(state)
			return nil
		}
```
This consumes `?` inside the panel so it never reaches the global help toggle while the panel is focused. Outside the panel, `?` is untouched.

- [ ] **Step 3: Build + test**:

Run: `go build ./... && go test ./internal/tui/ -count=1`
Expected: BUILD OK, all tui tests pass.

- [ ] **Step 4: Update `:help`.** In `internal/tui/app.go`, find the Action Plan help block (grep for the action-plan `:help` lines, near where the assist/footer lines are `fmt.Fprintf(&help, ...)`). Add one line, referencing `a.Keys.Help` (not a hardcoded `?`):
```go
	fmt.Fprintf(&help, "    %-18s ⚡  Inside a panel, '%s' shows that panel's key cheat-sheet\n", "", a.Keys.Help)
```
Place it in the Action Plan section. If `help_text_test.go` asserts exact line counts/content, update it to match.

- [ ] **Step 5: Document** in `docs/KEYBOARD_SHORTCUTS.md`. In the Action Plan In-Panel Keys table, add a row:
```
| `?` | Key cheat-sheet | Show all of this panel's keys in a read-only overlay; `Esc` returns (`keys.help`) |
```
And a one-line general note that `?` inside a panel shows the panel's cheat-sheet while `?` on the main view shows global help.

- [ ] **Step 6: Build, format, commit**:

```bash
gofmt -w internal/tui/action_plan_key_help.go internal/tui/action_plan.go internal/tui/app.go
go build ./... && go test ./internal/tui/ -count=1
git add internal/tui/action_plan_key_help.go internal/tui/action_plan.go internal/tui/app.go docs/KEYBOARD_SHORTCUTS.md
git commit -m "feat(tui): '?' shows the Action Plan key cheat-sheet in-panel"
```

---

## Task 4: Verification

- [ ] **Step 1:** `gofmt -l internal/` shows nothing; `make pre-commit-check` — Expected: all green.
- [ ] **Step 2:** Scoped tests: `go test ./internal/tui/ ./internal/config/ -count=1` — Expected: PASS.
- [ ] **Step 3:** Live smoke (dev laptop): build, open the Action Plan (`P`), press `?` → cheat-sheet appears listing the panel keys; `↑/↓` scrolls; `Esc` returns to the tree with selection intact. Separately, from the main list press `?` → global help still opens (no regression). Use the timing/log trick from the smoke harness ([[smoke-test-harness]]) to confirm no freeze.
- [ ] **Step 4:** `graphify update .`

---

## Self-Review Notes

- **Spec coverage:** reusable `KeyHint` + formatter (Task 1); single-source extended footer keys + hints builder (Task 2); body-swap presenter cloning the `v` viewer + `?` routing + `:help` + docs (Task 3); tests at each layer incl. rebind-reflection (Tasks 1-2) and no-regression smoke (Task 4). Out-of-scope items (other panels, `??`/`:help all`, provider interface) intentionally excluded.
- **Type consistency:** `KeyHint{Key,Desc}`, `formatKeyHelp(title, hints)`, `actionPlanKeyHints(keys actionPlanFooterKeys)`, `showActionPlanKeyHelp(state)`, extended `actionPlanFooterKeys{...,archive,trash,label,toggleRead,confirm}` — consistent across tasks.
- **Cross-branch:** g/. deliberately omitted (not on main); note in the header explains the trivial add-on at read-manually merge time.
- **Threading:** presenter is synchronous event-loop UI (clone of `showActionPlanPromptView`) — no goroutine/QueueUpdateDraw/ErrorHandler, satisfying the ESC-safety rule.
