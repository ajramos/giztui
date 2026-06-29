# Per-Command Help (`:help <cmd>`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** `:help <cmd>` shows focused, registry-sourced help for one command in the reader pane, rendered like the full `?` help (same overlay, Esc to return).

**Architecture:** Add an optional `help` field to `commandSpec`; a pure `generateCommandHelpText(spec)` builds the text (rich when `help` is set, an auto fallback otherwise); extract the full-help "show" logic into `showHelpScreen(content, title)` reused by `toggleHelp` and by `:help <cmd>`.

**Tech Stack:** Go, tview, standard `testing`.

---

## Task 1: Help content on the command registry

**Files:** Modify `internal/tui/command_completion.go`.

- [ ] **Step 1: Add the `cmdHelp` type + `help` field**

In `internal/tui/command_completion.go`, change the `commandSpec` struct and add `cmdHelp` above it:

```go
// cmdHelp is the optional rich help for a command (registry-sourced, shown by :help <cmd>).
type cmdHelp struct {
	summary  string   // one-line description
	syntax   string   // e.g. ":search <query>"
	examples []string // e.g. {":search from:ana has:attachment"}
}

type commandSpec struct {
	name        string
	aliases     []string
	completeArg argCompleter
	help        *cmdHelp // nil → auto fallback in generateCommandHelpText
}
```

- [ ] **Step 2: Populate `help` for the 9 argument commands**

In the `commandRegistry` literal, replace these 9 entries with the help-augmented versions (keep every other entry untouched):

```go
	{name: "search", completeArg: completeSearchArg, help: &cmdHelp{
		summary:  "Search Gmail messages (server-side).",
		syntax:   ":search <query>",
		examples: []string{":search from:ana has:attachment", ":search is:unread after:2026/01/01", ":search subject:invoice"},
	}},
	{name: "labels", aliases: []string{"l"}, completeArg: completeLabelsArg, help: &cmdHelp{
		summary:  "Manage labels on the selected message(s).",
		syntax:   ":labels [add|remove|list] <label>",
		examples: []string{":labels add Work", ":labels remove Work", ":labels list"},
	}},
	{name: "move", aliases: []string{"mv"}, help: &cmdHelp{
		summary:  "Move the next N messages to a folder/label (VIM-style range).",
		syntax:   ":move <count>",
		examples: []string{":move 5"},
	}},
	{name: "label", aliases: []string{"lbl"}, help: &cmdHelp{
		summary:  "Open the label picker for the next N messages (VIM-style range).",
		syntax:   ":label <count>",
		examples: []string{":label 3"},
	}},
	{name: "accounts", aliases: []string{"acc"}, completeArg: completeAccountsArg, help: &cmdHelp{
		summary:  "Switch the active Gmail account.",
		syntax:   ":accounts [switch <id>]",
		examples: []string{":accounts", ":accounts switch work"},
	}},
	{name: "prompt", aliases: []string{"pr", "p"}, completeArg: completePromptArg, help: &cmdHelp{
		summary:  "AI prompt library and management.",
		syntax:   ":prompt [list|create|update|export|delete|stats]",
		examples: []string{":prompt", ":prompt list"},
	}},
	{name: "theme", aliases: []string{"th"}, completeArg: completeThemeArg, help: &cmdHelp{
		summary:  "Switch or inspect the color theme.",
		syntax:   ":theme [list|set <name>|preview <name>]",
		examples: []string{":theme set gruvbox", ":theme list"},
	}},
	{name: "bookmark", aliases: []string{"query"}, completeArg: completeBookmarkArg, help: &cmdHelp{
		summary:  "Run a saved search query by name.",
		syntax:   ":bookmark <query name>",
		examples: []string{":bookmark Unread VIP"},
	}},
	{name: "slack", aliases: []string{"sl"}, help: &cmdHelp{
		summary:  "Forward a message to a configured Slack channel.",
		syntax:   ":slack [<message #>]",
		examples: []string{":slack", ":slack 3"},
	}},
```

NOTE: match the existing entry text exactly when locating each (e.g. `{name: "search", completeArg: completeSearchArg},`). Only `help:` is added; `name`/`aliases`/`completeArg` are unchanged.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
gofmt -w internal/tui/command_completion.go
git add internal/tui/command_completion.go
git commit -m "feat(tui): add cmdHelp to the command registry (9 arg commands)"
```

(No Co-Authored-By line.)

---

## Task 2: `generateCommandHelpText` + tests

**Files:** Create `internal/tui/command_help.go`, `internal/tui/command_help_test.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/command_help_test.go`:

```go
package tui

import (
	"strings"
	"testing"
)

func TestGenerateCommandHelpText_Rich(t *testing.T) {
	s := &commandSpec{
		name:    "search",
		aliases: nil,
		help: &cmdHelp{
			summary:  "Search Gmail messages.",
			syntax:   ":search <query>",
			examples: []string{":search is:unread"},
		},
	}
	got := generateCommandHelpText(s)
	for _, want := range []string{":search", "Search Gmail messages.", "Syntax:", ":search <query>", "Examples:", ":search is:unread", "Esc"} {
		if !strings.Contains(got, want) {
			t.Errorf("rich help missing %q in:\n%s", want, got)
		}
	}
}

func TestGenerateCommandHelpText_Fallback(t *testing.T) {
	s := &commandSpec{name: "archive", aliases: []string{"a"}}
	got := generateCommandHelpText(s)
	if !strings.Contains(got, ":archive") || !strings.Contains(got, "a") {
		t.Errorf("fallback must name the command + aliases:\n%s", got)
	}
	if !strings.Contains(got, "No detailed help") {
		t.Errorf("fallback must state no detailed help:\n%s", got)
	}
	if strings.Contains(got, "Syntax:") {
		t.Errorf("fallback must NOT have a Syntax block:\n%s", got)
	}
}
```

- [ ] **Step 2: Run, verify FAIL**

Run: `go test ./internal/tui/ -run TestGenerateCommandHelpText -v`
Expected: FAIL — `generateCommandHelpText` undefined.

- [ ] **Step 3: Implement**

Create `internal/tui/command_help.go`:

```go
package tui

import (
	"fmt"
	"strings"
)

// generateCommandHelpText renders focused help for a single command, shown by :help <cmd> in the
// reader pane (same overlay as the full help). Rich when spec.help is set, otherwise an auto fallback
// derived from the registry (name + aliases). tview dynamic-color tags style the title.
func generateCommandHelpText(s *commandSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[::b] :%s [::-]\n\n", s.name)

	if s.help == nil {
		b.WriteString("No detailed help for this command.\n\n")
		writeAliases(&b, s)
		b.WriteString("\nPress Esc to return. Press ? for the full command/shortcut list.\n")
		return b.String()
	}

	b.WriteString(s.help.summary + "\n\n")
	if s.help.syntax != "" {
		fmt.Fprintf(&b, "Syntax:\n    %s\n\n", s.help.syntax)
	}
	if len(s.help.examples) > 0 {
		b.WriteString("Examples:\n")
		for _, ex := range s.help.examples {
			fmt.Fprintf(&b, "    %s\n", ex)
		}
		b.WriteString("\n")
	}
	writeAliases(&b, s)
	b.WriteString("\nPress Esc to return.\n")
	return b.String()
}

func writeAliases(b *strings.Builder, s *commandSpec) {
	if len(s.aliases) == 0 {
		b.WriteString("Aliases: (none)\n")
		return
	}
	fmt.Fprintf(b, "Aliases: %s\n", strings.Join(s.aliases, ", "))
}
```

- [ ] **Step 4: Run, verify PASS**

Run: `go test ./internal/tui/ -run TestGenerateCommandHelpText -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/command_help.go internal/tui/command_help_test.go
git add internal/tui/command_help.go internal/tui/command_help_test.go
git commit -m "feat(tui): generateCommandHelpText (rich + fallback) with tests"
```

(No Co-Authored-By line.)

---

## Task 3: Reuse the help overlay + wire `:help <cmd>` + verify

**Files:** Modify `internal/tui/app.go` (`toggleHelp` show branch → `showHelpScreen`), `internal/tui/commands.go` (`executeHelpCommand`), and the in-app `?` help text.

- [ ] **Step 1: Extract `showHelpScreen` from `toggleHelp`**

In `internal/tui/app.go`, the `toggleHelp()` `else { ... }` block (the "show" branch, ~2494-2552) currently inlines the backup + header-hide + title + content + focus. Replace that ENTIRE `else` block body with a single call:

```go
	} else {
		a.showHelpScreen(a.generateHelpText(), " 📚 Help & Shortcuts ")
	}
```

Then add the new method immediately AFTER the closing `}` of `toggleHelp`:

```go
// showHelpScreen renders content in the reader pane with the given title, using the same overlay as
// the full help (Esc restores via toggleHelp's restore branch). The reader is backed up and the
// header hidden only on first show, so re-rendering (e.g. :help <cmd> while help is open) keeps the
// original backup. Shared by the full ? help and :help <cmd>.
func (a *App) showHelpScreen(content, title string) {
	if !a.showHelp {
		if text, ok := a.views["text"].(*tview.TextView); ok {
			a.helpBackup.text = text.GetText(false)
		}
		if header, ok := a.views["header"].(*tview.TextView); ok {
			a.helpBackup.header = header.GetText(false)
		}
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			a.helpBackup.title = textContainer.GetTitle()
		}
		if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
			if header, ok := a.views["header"].(*tview.TextView); ok {
				a.originalHeaderHeight = a.calculateHeaderHeight(header.GetText(false))
				header.SetDynamicColors(true)
				header.SetText("")
				textContainer.ResizeItem(header, 0, 0)
			}
		}
	}
	a.showHelp = true

	if textContainer, ok := a.views["textContainer"].(*tview.Flex); ok {
		textContainer.SetTitle(title)
		textContainer.SetTitleColor(a.GetComponentColors("general").Title.Color())
	}

	if a.enhancedTextView != nil {
		a.enhancedTextView.SetContent(content)
		a.enhancedTextView.SetDynamicColors(true)
		a.enhancedTextView.ScrollToBeginning()
	} else if text, ok := a.views["text"].(*tview.TextView); ok {
		text.SetDynamicColors(true)
		text.Clear()
		text.SetText(content)
		text.ScrollToBeginning()
	}

	if a.compositionPanel == nil || !a.compositionPanel.IsVisible() {
		a.focus.set("text")
		a.SetFocus(a.views["text"])
		a.updateFocusIndicators("text")
	}
}
```

- [ ] **Step 2: Build (full-help path unchanged)**

Run: `go build ./...` → success.
Run: `go test ./internal/tui/ -run TestGenerateHelpText 2>&1 | tail -2` → the existing help-text test still `ok` (the full `?` help is unchanged).

- [ ] **Step 3: Wire `:help <cmd>`**

In `internal/tui/commands.go`, replace:

```go
func (a *App) executeHelpCommand(args []string) {
	a.toggleHelp()
}
```

with:

```go
func (a *App) executeHelpCommand(args []string) {
	if len(args) > 0 {
		if s := lookupCommand(args[0]); s != nil {
			a.showHelpScreen(generateCommandHelpText(s), " 📚 :"+s.name+" ")
			return
		}
	}
	a.toggleHelp() // no arg, or unknown command → full help screen (toggle)
}
```

- [ ] **Step 4: Add a line to the in-app `?` help**

In `internal/tui/app.go`'s `generateHelpText()`, find the block that documents the `:help` command (grep `grep -n '"help"\|:help\|Help & Shortcuts\|help command' internal/tui/app.go`, or the COMMANDS section listing `:` commands). Add a line near the help/commands documentation:

```go
	fmt.Fprintf(&help, "    %-18s 📖  Focused help for one command (e.g. :help search)\n", ":help <cmd>")
```

Match the surrounding `Fprintf` width/style; place it adjacent to where `:help`/commands are listed. If you cannot find a clear commands block, add it right after the line that documents the full `:help`/`?` entry. Report where you put it.

- [ ] **Step 5: Build + gofmt + tests**

Run: `go build ./...` → success.
Run: `gofmt -w internal/tui/app.go internal/tui/commands.go`.
Run: `go test ./internal/tui/ 2>&1 | tail -3` → `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app.go internal/tui/commands.go
git commit -m "feat(tui): :help <cmd> shows focused per-command help (#29)"
```

(No Co-Authored-By line.)

- [ ] **Step 7: Canonical gate + harness smoke**

Run: `make pre-commit-check` → "All pre-commit checks passed!".
Run: `make test 2>&1 | grep -E '^FAIL' || echo NO_FAILURES` → `NO_FAILURES`.
Run: `make build` → built.

Harness smoke (pty driver `/tmp/giztui_smoke.py`): the per-command help renders in the reader pane.
```bash
log=~/.config/giztui/giztui.log; : > "$log"
timeout 28 python3 /tmp/giztui_smoke.py ./build/giztui "11:" "1::help search\r" "3:" "1:q" 2>&1 | tr '│' '\n' | grep -aiE "Operators|Examples|Search Gmail|Syntax" | head
grep -aic panic "$log"  # expect 0
```
Expected: the captured frame contains the search help text (e.g. "Search Gmail", "Syntax:", "Examples:"); 0 panics. If the status/screen capture is noisy, fall back to a temporary `a.logger.Printf` at the top of the `len(args) > 0` branch in `executeHelpCommand` to confirm the path runs with `s.name == "search"`, then REVERT it (do not commit the debug log).

- [ ] **Step 8: Finish**

Use superpowers:finishing-a-development-branch. Manual Mac smoke: `:help search` shows the focused help in the reader; Esc returns to the message; `:help archive` shows the fallback; `:help` (no arg) still toggles the full help. Do NOT push/merge without explicit user confirmation. No Co-Authored-By line.

---

## Self-Review

**Spec coverage:** `cmdHelp` + `help` field + rich help for the 9 arg commands (Task 1); `generateCommandHelpText` rich + fallback + tests (Task 2); `showHelpScreen` extracted, `toggleHelp` reuses it, `:help <cmd>` wired, unknown/no-arg → full help, `?`-help note (Task 3); harness + gate (Task 3 Steps 5,7). ✓

**Placeholder scan:** none — full code in every step; the only judgment step is locating the `?`-help insertion point (Task 3 Step 4), bounded by the grep and "report where you put it." ✓

**Type consistency:** `cmdHelp{summary, syntax, examples}`, `commandSpec.help *cmdHelp`, `generateCommandHelpText(s *commandSpec) string` (pure function — deviates from the spec's `(a *App)` receiver for testability; the help text needs no theme/App state), `showHelpScreen(content, title string)`, `writeAliases(b, s)` — named/used identically across tasks. `lookupCommand` is the existing registry resolver. ✓
