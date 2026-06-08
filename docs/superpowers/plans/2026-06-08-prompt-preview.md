# Prompt Preview (On-Demand) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Ctrl+P on-demand preview to both prompt pickers that opens a modal showing the highlighted prompt's description and full template text.

**Architecture:** Presentation-only. A new `internal/tui/prompt_preview.go` holds a pure body formatter (`promptPreviewText`) and a modal shower (`showPromptPreview`) that adds a centered overlay via `a.Pages` and restores focus on close. Both pickers gain a `promptText` field on their local `promptItem` and a Ctrl+P branch in their input/list input-captures, reusing the existing `promptPickerSelection` helper to resolve the highlighted item.

**Tech Stack:** Go, derailed/tview (TextView/Flex/Pages), tcell (KeyCtrlP), existing TUI patterns.

**Spec:** `docs/superpowers/specs/2026-06-08-prompt-preview-design.md`

---

## File Structure

- `internal/tui/prompt_preview.go` — NEW: `promptPreviewText()` (pure), `promptPreviewCreateNewHint` const, `showPromptPreview()` (modal overlay).
- `internal/tui/prompt_preview_test.go` — NEW: unit tests for `promptPreviewText` and modal page creation.
- `internal/tui/bulk_prompts.go` — add `promptText` to local `promptItem` (line 63), populate it, add Ctrl+P branch to `input.SetInputCapture` (line 182) and `list.SetInputCapture` (line 195).
- `internal/tui/prompts.go` — same for `openPromptPicker`: `promptItem` (line 65), `input.SetInputCapture` (line 166), `list.SetInputCapture` (line 226). (NOT `openPromptPickerForManagement` at line 589 — that is the CRUD picker, out of scope.)
- `docs/KEYBOARD_SHORTCUTS.md` — document Ctrl+P preview in the prompt pickers.

**Verified facts (use exactly):**
- `PromptTemplate` (internal/prompts/types.go) has `Name`, `Description`, `PromptText`, `Category`. The pickers already receive `[]*PromptTemplate` from `promptService.ListPrompts`.
- Modal precedent: `a.Pages.AddPage("name", view, true, true)` + `a.Pages.RemovePage("name")` (messages.go:2022, composition.go:883). `ForceFilledBorderFlex(f *tview.Flex)` exists in layout.go:13.
- `promptPickerSelection(currentItem, visibleCount int) (isCreateNew bool, visibleIndex int)` already exists (internal/tui/prompt_picker_selection.go): index 0 → create-new; else `visible[index-1]`.
- tview import alias in this package is plain `tview "github.com/derailed/tview"` (see any picker file). `tcell` is `github.com/derailed/tcell/v2` (see bulk_prompts.go).
- `(*tview.Application).GetFocus()` and `SetFocus()` are available (App embeds `*tview.Application`).

---

## Task 1: Pure preview-body formatter

**Files:**
- Create: `internal/tui/prompt_preview.go`
- Test: `internal/tui/prompt_preview_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/prompt_preview_test.go`:

```go
package tui

import (
	"strings"
	"testing"
)

func TestPromptPreviewText(t *testing.T) {
	out := promptPreviewText("Short factual summary", "Summarize {{body}} in {{max_words}} words")
	if !strings.Contains(out, "Description:") || !strings.Contains(out, "Short factual summary") {
		t.Errorf("description block missing: %q", out)
	}
	if !strings.Contains(out, "Template:") || !strings.Contains(out, "{{body}}") {
		t.Errorf("template block missing: %q", out)
	}

	// Empty fields render explicit placeholders, never blank.
	out2 := promptPreviewText("  ", "")
	if !strings.Contains(out2, "(no description)") {
		t.Errorf("empty description placeholder missing: %q", out2)
	}
	if !strings.Contains(out2, "(empty template)") {
		t.Errorf("empty template placeholder missing: %q", out2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestPromptPreviewText -v`
Expected: FAIL — `promptPreviewText` undefined.

- [ ] **Step 3: Implement the formatter**

Create `internal/tui/prompt_preview.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// promptPreviewCreateNewHint is shown when the highlighted row is the
// "✨ Create new with AI…" entry rather than a real prompt.
const promptPreviewCreateNewHint = "Opens the AI prompt configurator to create a new prompt."

// promptPreviewText builds the modal body for a prompt preview: a Description
// block followed by the full Template (PromptText). Empty fields become explicit
// placeholders so the preview is never blank.
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
```

(The `tcell`/`tview` imports are used by `showPromptPreview` in Task 2; if implementing strictly task-by-task and the build complains about unused imports, add them in Task 2 instead.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestPromptPreviewText -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/prompt_preview.go internal/tui/prompt_preview_test.go
git commit -m "feat(tui): add promptPreviewText formatter for prompt preview"
```

---

## Task 2: Modal overlay shower

**Files:**
- Modify: `internal/tui/prompt_preview.go`
- Test: `internal/tui/prompt_preview_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/prompt_preview_test.go`:

```go
import (
	// add to the existing import block:
	"github.com/derailed/tview"
)

func TestShowPromptPreviewAddsAndRemovesPage(t *testing.T) {
	app := &App{
		Application: tview.NewApplication(),
		Config:      &config.Config{},
	}
	app.Pages = tview.NewPages()

	app.showPromptPreview("Quick Summary", "Description:\nx\n\nTemplate:\ny")
	if !app.Pages.HasPage("promptPreview") {
		t.Fatal("expected promptPreview page to be added")
	}

	// closePromptPreview removes the page (the Esc/Ctrl+P handler calls this).
	app.closePromptPreview()
	if app.Pages.HasPage("promptPreview") {
		t.Fatal("expected promptPreview page to be removed")
	}
}
```

Add `"github.com/ajramos/giztui/internal/config"` to the test file's imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestShowPromptPreviewAddsAndRemovesPage -v`
Expected: FAIL — `showPromptPreview` / `closePromptPreview` undefined.

- [ ] **Step 3: Implement the modal**

Append to `internal/tui/prompt_preview.go`:

```go
// promptPreviewPage is the Pages name for the preview overlay.
const promptPreviewPage = "promptPreview"

// closePromptPreview removes the preview overlay and restores focus to whatever
// the picker had focused before it opened. Synchronous (no QueueUpdateDraw) so it
// is safe from Esc handlers.
func (a *App) closePromptPreview() {
	a.Pages.RemovePage(promptPreviewPage)
	if a.promptPreviewPrevFocus != nil {
		a.SetFocus(a.promptPreviewPrevFocus)
		a.promptPreviewPrevFocus = nil
	}
}

// showPromptPreview opens a centered, scrollable modal showing a prompt preview.
// Esc or Ctrl+P closes it (via closePromptPreview); other keys scroll the body.
func (a *App) showPromptPreview(name, body string) {
	colors := a.GetComponentColors("prompts")

	tv := tview.NewTextView().
		SetDynamicColors(false).
		SetWrap(true).
		SetText(body)
	tv.SetScrollable(true)
	tv.SetBackgroundColor(colors.Background.Color())
	tv.SetTextColor(colors.Text.Color())

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.SetBorder(true).
		SetTitle(fmt.Sprintf(" 👁 Preview: %s ", name)).
		SetTitleColor(colors.Title.Color()).
		SetBorderColor(colors.Border.Color()).
		SetBackgroundColor(colors.Background.Color())
	ForceFilledBorderFlex(box)
	box.SetTitleColor(colors.Title.Color())
	box.AddItem(tv, 0, 1, true)

	footer := tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("Esc / Ctrl+P to close")
	footer.SetBackgroundColor(colors.Background.Color())
	footer.SetTextColor(colors.Text.Color())
	box.AddItem(footer, 1, 0, false)

	tv.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyEscape || e.Key() == tcell.KeyCtrlP {
			a.closePromptPreview()
			return nil
		}
		return e // arrows / PgUp / PgDn scroll the TextView
	})

	// Center the box at roughly 4/6 of the screen in each axis using spacers.
	centered := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(box, 0, 4, true).
			AddItem(nil, 0, 1, false), 0, 4, true).
		AddItem(nil, 0, 1, false)

	a.promptPreviewPrevFocus = a.GetFocus()
	a.Pages.AddPage(promptPreviewPage, centered, true, true)
	a.SetFocus(tv)
}
```

- [ ] **Step 4: Add the focus-restore field to App**

In `internal/tui/app.go`, add a field to the `App` struct (near other transient UI state fields, e.g. after `currentActivePicker`):

```go
	promptPreviewPrevFocus tview.Primitive // focus to restore when the prompt preview modal closes
```

Confirm `internal/tui/app.go` imports `tview "github.com/derailed/tview"` (it does).

- [ ] **Step 5: Run test + build**

Run: `go test ./internal/tui/ -run TestShowPromptPreviewAddsAndRemovesPage -v`
Expected: PASS.
Run: `go build ./internal/tui/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/prompt_preview.go internal/tui/prompt_preview_test.go internal/tui/app.go
git commit -m "feat(tui): add showPromptPreview centered modal overlay"
```

---

## Task 3: Wire Ctrl+P into the bulk prompt picker

**Files:**
- Modify: `internal/tui/bulk_prompts.go` (promptItem ~line 63; item build ~line 154; input capture ~line 182; list capture ~line 195)

- [ ] **Step 1: Add `promptText` to the local promptItem and populate it**

In `openBulkPromptPicker`, the local struct (around line 63) is:

```go
	type promptItem struct {
		id          int
		name        string
		description string
		category    string
	}
```

Change it to add the template:

```go
	type promptItem struct {
		id          int
		name        string
		description string
		promptText  string
		category    string
	}
```

Then where items are built from the loaded prompts (around line 153-159, inside the `for _, p := range prompts` loop):

```go
				all = append(all, promptItem{
					id:          p.ID,
					name:        p.Name,
					description: p.Description,
					category:    p.Category,
				})
```

add the field:

```go
				all = append(all, promptItem{
					id:          p.ID,
					name:        p.Name,
					description: p.Description,
					promptText:  p.PromptText,
					category:    p.Category,
				})
```

- [ ] **Step 2: Add Ctrl+P to the input capture**

The bulk picker's `input.SetInputCapture` (around line 182) currently starts:

```go
				input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					if event.Key() == tcell.KeyEscape {
						a.closeBulkPromptPicker()
						return nil
					}
```

Insert a Ctrl+P branch as the first check inside it:

```go
				input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					if event.Key() == tcell.KeyCtrlP {
						a.previewHighlightedBulkPrompt(list, visible)
						return nil
					}
					if event.Key() == tcell.KeyEscape {
						a.closeBulkPromptPicker()
						return nil
					}
```

- [ ] **Step 3: Add Ctrl+P to the list capture**

The bulk picker's `list.SetInputCapture` (around line 195) currently:

```go
				list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
					if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
						a.SetFocus(input)
						return nil
					}
```

Insert a Ctrl+P branch as the first check:

```go
				list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
					if e.Key() == tcell.KeyCtrlP {
						a.previewHighlightedBulkPrompt(list, visible)
						return nil
					}
					if e.Key() == tcell.KeyUp && list.GetCurrentItem() == 0 {
						a.SetFocus(input)
						return nil
					}
```

- [ ] **Step 4: Add the bulk preview helper**

`visible` is a `[]promptItem` whose element type is local to `openBulkPromptPicker`, so the helper must be a closure defined inside that function (it cannot be a package-level method taking `[]promptItem`). Define it just before the container is assembled (after the `reload`/load goroutine, near line 220, but it must close over nothing that changes — define it inside `openBulkPromptPicker` so it can reference the local `promptItem` type). Add:

```go
	// previewHighlightedBulkPrompt opens the preview modal for the highlighted row.
	previewHighlightedBulkPrompt := func(list *tview.List, visible []promptItem) {
		isCreateNew, vi := promptPickerSelection(list.GetCurrentItem(), len(visible))
		if isCreateNew {
			a.showPromptPreview("Create new with AI", promptPreviewCreateNewHint)
			return
		}
		v := visible[vi]
		a.showPromptPreview(v.name, promptPreviewText(v.description, v.promptText))
	}
```

Then change the two call sites in Steps 2 and 3 from `a.previewHighlightedBulkPrompt(list, visible)` to `previewHighlightedBulkPrompt(list, visible)` (local closure, no `a.` receiver).

NOTE: `previewHighlightedBulkPrompt` must be declared lexically BEFORE the `input.SetInputCapture`/`list.SetInputCapture` calls that reference it (those are set inside the load goroutine at ~line 182/195). Declare it near the top of `openBulkPromptPicker`, right after `visible` is declared (~line 71), so it is in scope for the goroutine closures.

- [ ] **Step 5: Build**

Run: `go build ./internal/tui/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/bulk_prompts.go
git commit -m "feat(tui): Ctrl+P prompt preview in the bulk prompt picker"
```

---

## Task 4: Wire Ctrl+P into the single-message prompt picker

**Files:**
- Modify: `internal/tui/prompts.go` (promptItem ~line 65; item build ~line 144; input capture ~line 166; list capture ~line 226)

- [ ] **Step 1: Add `promptText` to the local promptItem and populate it**

In `openPromptPicker`, change the local struct (around line 65) from:

```go
	type promptItem struct {
		id          int
		name        string
		description string
		category    string
	}
```

to:

```go
	type promptItem struct {
		id          int
		name        string
		description string
		promptText  string
		category    string
	}
```

Where items are built (around line 144-150, `for _, p := range prompts`):

```go
		all = append(all, promptItem{
			id:          p.ID,
			name:        p.Name,
			description: p.Description,
			category:    p.Category,
		})
```

add the field:

```go
		all = append(all, promptItem{
			id:          p.ID,
			name:        p.Name,
			description: p.Description,
			promptText:  p.PromptText,
			category:    p.Category,
		})
```

- [ ] **Step 2: Add the single-picker preview closure**

Right after `visible` is declared (around line 73, `var visible []promptItem`), add the closure so it is in scope for the input-capture set inside the load goroutine:

```go
	// previewHighlightedPrompt opens the preview modal for the highlighted row.
	previewHighlightedPrompt := func(list *tview.List, visible []promptItem) {
		isCreateNew, vi := promptPickerSelection(list.GetCurrentItem(), len(visible))
		if isCreateNew {
			a.showPromptPreview("Create new with AI", promptPreviewCreateNewHint)
			return
		}
		v := visible[vi]
		a.showPromptPreview(v.name, promptPreviewText(v.description, v.promptText))
	}
```

- [ ] **Step 3: Add Ctrl+P to the input capture**

The single picker's `input.SetInputCapture` (around line 166):

```go
				input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
					if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp || e.Key() == tcell.KeyPgDn || e.Key() == tcell.KeyPgUp {
						a.SetFocus(list)
						return e
					}
					return e
				})
```

Add a Ctrl+P branch first:

```go
				input.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
					if e.Key() == tcell.KeyCtrlP {
						previewHighlightedPrompt(list, visible)
						return nil
					}
					if e.Key() == tcell.KeyDown || e.Key() == tcell.KeyUp || e.Key() == tcell.KeyPgDn || e.Key() == tcell.KeyPgUp {
						a.SetFocus(list)
						return e
					}
					return e
				})
```

- [ ] **Step 4: Add Ctrl+P to the list capture**

The single picker's `list.SetInputCapture` (around line 226). Add a Ctrl+P branch as the first check, mirroring the existing handler shape (it currently handles `KeyUp` at item 0 / `KeyEscape`):

```go
				list.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
					if e.Key() == tcell.KeyCtrlP {
						previewHighlightedPrompt(list, visible)
						return nil
					}
```

(Leave the rest of the existing list capture body unchanged.)

- [ ] **Step 5: Build**

Run: `go build ./internal/tui/`
Expected: success.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/prompts.go
git commit -m "feat(tui): Ctrl+P prompt preview in the single-message prompt picker"
```

---

## Task 5: Documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Document the preview key**

Find the prompt picker / Prompt Library section (search for "Prompt Library" or the `p` / Prompt key). Add a note:

> **In the prompt picker (single or bulk):** press **Ctrl+P** with a prompt highlighted to preview its description and full template in a popup. Press **Esc** or **Ctrl+P** again to close and return to the picker. Works whether focus is on the search field or the list.

- [ ] **Step 2: Commit**

```bash
git add docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs: document Ctrl+P prompt preview in the pickers"
```

---

## Task 6: Full verification + real-app E2E

**Files:** none (verification only)

- [ ] **Step 1: Run the pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + golangci-lint + essential tests all pass. Fix any issues.

- [ ] **Step 2: Build the binary**

Run: `make build`
Expected: `build/giztui` produced.

- [ ] **Step 3: Real-app E2E via tmux (drive the app — use /usr/bin/tmux directly)**

1. Start the app: `/usr/bin/tmux new-session -d -s gtz -x 200 -y 50 -c $(pwd); /usr/bin/tmux send-keys -t gtz './build/giztui' Enter; sleep 12`.
2. **Single picker:** focus the list, press `p` to open the Prompt Library. Capture the pane and confirm it is open.
3. Press **Ctrl+P** (`/usr/bin/tmux send-keys -t gtz C-p`). Capture and confirm a `👁 Preview:` modal shows the highlighted prompt's Description and Template.
4. Press an arrow / PgDn to confirm a long template scrolls; press **Esc** and confirm the modal closes and the picker is back with the same highlighted row.
5. Press Ctrl+P again, then Ctrl+P to confirm it also closes the modal.
6. **Bulk picker:** select 2 messages (`v`, `Space`, `j`, `Space`), press `p`, then Ctrl+P; confirm the same modal behavior.
7. Highlight the `✨ Create new with AI…` row and press Ctrl+P; confirm the modal shows the create-new hint, not a template.
8. Kill the session: `/usr/bin/tmux kill-session -t gtz`.

If anything misbehaves, debug with superpowers:systematic-debugging before claiming done.

- [ ] **Step 4: Final commit (only if E2E required fixes)**

```bash
git add -A
git commit -m "fix(tui): prompt preview E2E corrections"
```

---

## Self-Review Notes

- **Spec coverage:** on-demand (Tasks 3/4 Ctrl+P), modal popup via `a.Pages` (Task 2), Ctrl+P from input AND list focus (Tasks 3/4 both captures), content = description + full template (Task 1), scrollable (Task 2 `SetScrollable` + pass-through keys), both pickers (Tasks 3/4), create-new hint (Task 1 const + Tasks 3/4 branch), focus restore on close (Task 2 `promptPreviewPrevFocus`), no service changes (only local `promptItem` gains a field), theming `GetComponentColors("prompts")` (Task 2), no command parity needed (spec). All covered.
- **Type consistency:** `promptPreviewText(description, promptText string) string`, `showPromptPreview(name, body string)`, `closePromptPreview()`, `promptPreviewPrevFocus tview.Primitive`, `promptPreviewPage`/`promptPreviewCreateNewHint` consts, and `promptPickerSelection(currentItem, visibleCount int) (bool, int)` are used identically across tasks.
- **Closure-not-method note:** because `promptItem` is a function-local type in each picker, the preview trigger is a local closure (`previewHighlightedBulkPrompt` / `previewHighlightedPrompt`) declared before the input-capture goroutine, NOT a package method. Flagged in Tasks 3/4 so the implementer declares it in the right scope.
- **Verify-before-use:** the exact line numbers are approximate ("~line N"); the implementer should locate the anchor strings shown. The `tcell` import path (`github.com/derailed/tcell/v2`) should be confirmed from `bulk_prompts.go`'s import block.
