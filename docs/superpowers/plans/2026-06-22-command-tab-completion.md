# Command Bar Tab Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Tab in the `:` command bar cycle through matching command names/aliases (Shift+Tab backward, Enter runs the visible one) and complete label-name arguments for `:label`/`:labels`/`:move`, driven by one command registry instead of the hand-maintained prefix map.

**Architecture:** A package-level `commandRegistry` (canonical name + aliases + optional arg-completer) is the single source of truth. `App.commandCandidates(text)` returns the ordered candidate list for either the command token or the last argument token. Cycle state (`candidates`, `cycleIndex`) lives on `commandState`. The command bar's `InputField` capture cycles on Tab/Shift+Tab. Label names are pre-fetched async when the bar opens (the labels API is a network call and must not run on the event-loop Tab path).

**Tech Stack:** Go, tview/tcell, standard `testing` + `sort`/`strings`.

---

## Reference facts (verified in code)

- The live command bar is built in `showCommandBarWithPrefix` (`internal/tui/commands.go` ~48-138): a `tview.InputField` whose `SetInputCapture` handles `Tab`/`Up`/`Down`/`Esc`, and `SetChangedFunc` mirrors text into `a.cmd.buffer` and updates a side `hint` TextView (`a.views["cmdHint"]`). `handleCommandInput` (the `switch` further down) is legacy/unused for the live bar — do NOT wire there.
- Today's Tab calls `generateCommandSuggestion(cur)` (a giant hardcoded `map[string][]string` prefix→suggestion, `commands.go` ~215-630) and `input.SetText(s)`. `completeCommand` (~634) uses `a.cmd.suggestion`. These are what we replace.
- The top-level commands are the `case` lines of `executeCommand` (`commands.go`, switch spans line 705→`default:` at 831). The 56 entries are listed verbatim in Task 1. (Cases at 911+ are sub-command switches inside other functions — NOT top-level commands.)
- `a.userLabelNames()` (`internal/tui/action_plan.go` ~832) returns user (non-system) label names via `labelService.ListLabels` → `gmailClient.ListLabels()`, which is a **network call, not cached** (`internal/services/label_service.go`, `internal/gmail/client.go:603`). So it must be called off the event loop.
- `commandState` (`internal/tui/command_state.go`): `mode atomic.Bool` (only cross-goroutine field), plus event-loop-only `buffer`, `suggestion`, `focusOverride`, `history`, `historyIndex`. Plain fields added here stay event-loop-only.

---

## Task 1: Command registry + command-name matcher

**Files:**
- Create: `internal/tui/command_completion.go`
- Test: `internal/tui/command_completion_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/command_completion_test.go`:

```go
package tui

import (
	"reflect"
	"testing"
)

func TestCommandCandidates_Names(t *testing.T) {
	a := &App{}

	// "arch" matches canonical names "archive" and "archived" (sorted: "archive" < "archived").
	if got := a.commandCandidates("arch"); !reflect.DeepEqual(got, []string{"archive", "archived"}) {
		t.Fatalf("arch -> %v, want [archive archived]", got)
	}

	// Alias maps to its canonical name: "a" (an alias of archive) must include "archive".
	foundArchive := false
	for _, c := range a.commandCandidates("a") {
		if c == "archive" {
			foundArchive = true
		}
	}
	if !foundArchive {
		t.Fatalf("prefix 'a' should include canonical 'archive'")
	}

	// Unique match completes fully: only "attachments" (name) / "attach" (alias) start with "atta".
	if got := a.commandCandidates("atta"); !reflect.DeepEqual(got, []string{"attachments"}) {
		t.Fatalf("atta -> %v, want [attachments]", got)
	}

	// No match -> nil.
	if got := a.commandCandidates("zzzzz"); got != nil {
		t.Fatalf("zzzzz -> %v, want nil", got)
	}

	// Blank -> nil.
	if got := a.commandCandidates("   "); got != nil {
		t.Fatalf("blank -> %v, want nil", got)
	}
}

// Drift guard: every command in the registry has a non-empty canonical name and no duplicate names.
func TestCommandRegistry_NoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range commandRegistry {
		if s.name == "" {
			t.Fatal("registry has an entry with empty name")
		}
		if seen[s.name] {
			t.Fatalf("duplicate registry name: %q", s.name)
		}
		seen[s.name] = true
	}
}
```

- [ ] **Step 2: Run it, verify FAIL (undefined: commandRegistry / commandCandidates)**

Run: `go test ./internal/tui/ -run 'TestCommandCandidates_Names|TestCommandRegistry_NoDuplicateNames' -v`
Expected: FAIL — `commandRegistry`/`commandCandidates` undefined.

- [ ] **Step 3: Create the registry + matcher**

Create `internal/tui/command_completion.go`:

```go
package tui

import (
	"sort"
	"strings"
)

// argCompleter returns candidate completions for the current (last) argument token of a command.
// prefix is the partial token the user has typed (may be ""). Implementations must NOT block the
// event loop (no network) — read already-loaded state only.
type argCompleter func(a *App, prefix string) []string

// commandSpec is one entry in the command registry: a canonical command name, its aliases, and an
// optional argument completer. The registry mirrors the executeCommand switch and is the single
// source of truth for Tab completion.
type commandSpec struct {
	name        string
	aliases     []string
	completeArg argCompleter
}

// commandRegistry lists every top-level `:` command. Keep in sync with the executeCommand switch in
// commands.go. Adding a command here is all that's needed for it to autocomplete.
var commandRegistry = []commandSpec{
	{name: "labels", aliases: []string{"l"}, completeArg: completeLabelArg},
	{name: "links", aliases: []string{"link"}},
	{name: "attachments", aliases: []string{"attach"}},
	{name: "gmail", aliases: []string{"web", "open-web", "o"}},
	{name: "search"},
	{name: "slack", aliases: []string{"sl"}},
	{name: "s"},
	{name: "summary"},
	{name: "rsvp"},
	{name: "inbox", aliases: []string{"i"}},
	{name: "compose", aliases: []string{"c"}},
	{name: "headers", aliases: []string{"toggle-headers"}},
	{name: "threads", aliases: []string{"thr"}},
	{name: "flatten", aliases: []string{"flat"}},
	{name: "thread-summary", aliases: []string{"th-sum"}},
	{name: "expand-all", aliases: []string{"expand"}},
	{name: "collapse-all", aliases: []string{"collapse"}},
	{name: "help", aliases: []string{"h"}},
	{name: "numbers", aliases: []string{"n"}},
	{name: "quit", aliases: []string{"q"}},
	{name: "cache"},
	{name: "preload", aliases: []string{"pl"}},
	{name: "stats", aliases: []string{"usage"}},
	{name: "g"},
	{name: "archive", aliases: []string{"a"}},
	{name: "trash", aliases: []string{"d"}},
	{name: "read", aliases: []string{"toggle-read", "t"}},
	{name: "new"},
	{name: "reply", aliases: []string{"r"}},
	{name: "reply-all", aliases: []string{"ra"}},
	{name: "forward", aliases: []string{"f"}},
	{name: "drafts", aliases: []string{"dr"}},
	{name: "refresh"},
	{name: "autorefresh", aliases: []string{"arr"}},
	{name: "config", aliases: []string{"cfg"}},
	{name: "load", aliases: []string{"more", "next"}},
	{name: "unread", aliases: []string{"u"}},
	{name: "undo"},
	{name: "archived", aliases: []string{"arch-search", "b"}},
	{name: "select", aliases: []string{"sel"}},
	{name: "move", aliases: []string{"mv"}, completeArg: completeLabelArg},
	{name: "label", aliases: []string{"lbl"}, completeArg: completeLabelArg},
	{name: "obsidian", aliases: []string{"obs"}},
	{name: "accounts", aliases: []string{"acc"}},
	{name: "prompt", aliases: []string{"pr", "p"}},
	{name: "prompt-new", aliases: []string{"pn"}},
	{name: "prompt-refine", aliases: []string{"prf"}},
	{name: "prompt-save", aliases: []string{"ps"}},
	{name: "action-plan", aliases: []string{"plan", "ap"}},
	{name: "markdown", aliases: []string{"md"}},
	{name: "touch-up", aliases: []string{"touchup"}},
	{name: "theme", aliases: []string{"th"}},
	{name: "save-query", aliases: []string{"save", "sq"}},
	{name: "bookmarks", aliases: []string{"queries", "bm", "qb"}},
	{name: "bookmark", aliases: []string{"query"}},
}

// lookupCommand resolves a command token (name or alias, case-insensitive) to its spec, or nil.
func lookupCommand(token string) *commandSpec {
	token = strings.ToLower(token)
	for i := range commandRegistry {
		s := &commandRegistry[i]
		if strings.ToLower(s.name) == token {
			return s
		}
		for _, al := range s.aliases {
			if strings.ToLower(al) == token {
				return s
			}
		}
	}
	return nil
}

// matchesPrefix reports whether the spec's name or any alias starts with lowerPrefix.
func matchesPrefix(s *commandSpec, lowerPrefix string) bool {
	if strings.HasPrefix(strings.ToLower(s.name), lowerPrefix) {
		return true
	}
	for _, al := range s.aliases {
		if strings.HasPrefix(strings.ToLower(al), lowerPrefix) {
			return true
		}
	}
	return false
}

// commandCandidates returns the ordered Tab candidates for the given command-bar text. With no
// space yet it completes the command token (returns matching canonical names, sorted, de-duped).
// With a "command <args>" shape it delegates to the command's argument completer for the last token.
// Returns nil when nothing matches. The input is NOT trimmed of a trailing space (a trailing space
// means "complete the next, empty, argument").
func (a *App) commandCandidates(text string) []string {
	text = strings.TrimLeft(text, " ")
	if text == "" {
		return nil
	}

	// Argument completion: "command<space>...".
	if i := strings.IndexByte(text, ' '); i >= 0 {
		spec := lookupCommand(text[:i])
		if spec == nil || spec.completeArg == nil {
			return nil
		}
		rest := text[i+1:] // everything after "command "
		head := ""         // already-typed arg tokens, including the trailing space
		argPrefix := rest
		if ls := strings.LastIndexByte(rest, ' '); ls >= 0 {
			head = rest[:ls+1]
			argPrefix = rest[ls+1:]
		}
		cands := spec.completeArg(a, argPrefix)
		if len(cands) == 0 {
			return nil
		}
		linePrefix := text[:i] + " " + head
		out := make([]string, 0, len(cands))
		for _, c := range cands {
			out = append(out, linePrefix+c)
		}
		return out
	}

	// Command-token completion.
	lower := strings.ToLower(text)
	seen := map[string]bool{}
	var out []string
	for i := range commandRegistry {
		s := &commandRegistry[i]
		if matchesPrefix(s, lower) && !seen[s.name] {
			seen[s.name] = true
			out = append(out, s.name)
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}
```

NOTE: `completeLabelArg` is referenced here but defined in Task 3. The package will not compile until Task 3 adds it — so this task's test is run with a temporary stub. Add this stub at the bottom of `command_completion.go` now and DELETE it in Task 3:

```go
// TEMP stub (replaced in Task 3). Allows Task 1 to compile/test in isolation.
func completeLabelArg(a *App, prefix string) []string { return nil }
```

- [ ] **Step 4: Run the tests, verify PASS**

Run: `go test ./internal/tui/ -run 'TestCommandCandidates_Names|TestCommandRegistry_NoDuplicateNames' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/command_completion.go internal/tui/command_completion_test.go
git add internal/tui/command_completion.go internal/tui/command_completion_test.go
git commit -m "feat(tui): command registry + command-name completion matcher"
```

(No Co-Authored-By line.)

---

## Task 2: Cycle state on commandState

**Files:**
- Modify: `internal/tui/command_state.go`
- Test: `internal/tui/command_state_test.go` (create if absent; otherwise append)

- [ ] **Step 1: Write the failing test**

Create or append `internal/tui/command_state_test.go`:

```go
package tui

import "testing"

func TestCommandState_Cycle(t *testing.T) {
	var c commandState
	// No candidates yet.
	if _, ok := c.nextCandidate(true); ok {
		t.Fatal("nextCandidate on empty must return ok=false")
	}

	c.startCycle([]string{"search", "slack"})
	// Forward: -1 -> 0 -> 1 -> wrap 0.
	if v, ok := c.nextCandidate(true); !ok || v != "search" {
		t.Fatalf("first forward = %q,%v want search,true", v, ok)
	}
	if v, _ := c.nextCandidate(true); v != "slack" {
		t.Fatalf("second forward = %q want slack", v)
	}
	if v, _ := c.nextCandidate(true); v != "search" {
		t.Fatalf("third forward (wrap) = %q want search", v)
	}
	// Backward from index 0 wraps to last.
	if v, _ := c.nextCandidate(false); v != "slack" {
		t.Fatalf("backward = %q want slack", v)
	}

	// clearCycle empties the candidate list.
	c.clearCycle()
	if _, ok := c.nextCandidate(true); ok {
		t.Fatal("nextCandidate after clearCycle must return ok=false")
	}
}
```

- [ ] **Step 2: Run it, verify FAIL**

Run: `go test ./internal/tui/ -run TestCommandState_Cycle -v`
Expected: FAIL — `startCycle`/`nextCandidate`/`clearCycle` undefined.

- [ ] **Step 3: Add cycle state + methods**

In `internal/tui/command_state.go`, add three fields to the `commandState` struct (right after `historyIndex int`):

```go
	// Tab-completion cycle (event-loop only): candidate list and current index (-1 = before first).
	candidates []string
	cycleIndex int
	cycling    bool // true while we programmatically set the input during a cycle (see SetChangedFunc)
	// labelNames caches the user's label names for argument completion, pre-fetched off the event
	// loop when the bar opens (the labels API is a blocking network call).
	labelNames []string
```

Then add these methods at the end of the file:

```go
// startCycle begins a fresh Tab cycle over cands (index parked before the first element).
func (c *commandState) startCycle(cands []string) {
	c.candidates = cands
	c.cycleIndex = -1
}

// nextCandidate advances the cycle (forward or backward, wrapping) and returns the candidate.
// ok is false when there are no candidates.
func (c *commandState) nextCandidate(forward bool) (string, bool) {
	n := len(c.candidates)
	if n == 0 {
		return "", false
	}
	if forward {
		c.cycleIndex = (c.cycleIndex + 1) % n
	} else {
		c.cycleIndex = (c.cycleIndex - 1 + n) % n
	}
	return c.candidates[c.cycleIndex], true
}

// clearCycle drops the candidate cache (called when the user edits the buffer).
func (c *commandState) clearCycle() {
	c.candidates = nil
	c.cycleIndex = -1
}
```

- [ ] **Step 4: Run the test, verify PASS**

Run: `go test ./internal/tui/ -run TestCommandState_Cycle -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/command_state.go internal/tui/command_state_test.go
git add internal/tui/command_state.go internal/tui/command_state_test.go
git commit -m "feat(tui): Tab cycle state on commandState"
```

---

## Task 3: Label argument completer + async preload

**Files:**
- Modify: `internal/tui/command_completion.go` (replace the temp stub)
- Test: `internal/tui/command_completion_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/command_completion_test.go`:

```go
func TestCompleteLabelArg(t *testing.T) {
	a := &App{}
	a.cmd.labelNames = []string{"Work", "Personal", "Worklog", "travel"}

	// Case-insensitive prefix, sorted.
	got := completeLabelArg(a, "wor")
	want := []string{"Work", "Worklog"}
	if len(got) != 2 || got[0] != "Work" || got[1] != "Worklog" {
		t.Fatalf("wor -> %v, want %v", got, want)
	}

	// Empty prefix returns all (sorted, case-insensitive).
	all := completeLabelArg(a, "")
	if len(all) != 4 {
		t.Fatalf("empty prefix -> %d candidates, want 4", len(all))
	}

	// No match -> nil.
	if got := completeLabelArg(a, "zzz"); got != nil {
		t.Fatalf("zzz -> %v, want nil", got)
	}
}

func TestCommandCandidates_LabelArg(t *testing.T) {
	a := &App{}
	a.cmd.labelNames = []string{"Work", "Personal"}

	// "labels add wor" -> "labels add Work".
	got := a.commandCandidates("labels add wor")
	if len(got) != 1 || got[0] != "labels add Work" {
		t.Fatalf("labels add wor -> %v, want [labels add Work]", got)
	}

	// Command without an arg completer yields nil in arg position.
	if got := a.commandCandidates("archive x"); got != nil {
		t.Fatalf("archive x -> %v, want nil", got)
	}
}
```

- [ ] **Step 2: Run it, verify FAIL**

Run: `go test ./internal/tui/ -run 'TestCompleteLabelArg|TestCommandCandidates_LabelArg' -v`
Expected: FAIL — the temp `completeLabelArg` stub returns nil, so assertions fail.

- [ ] **Step 3: Replace the stub with the real completer**

In `internal/tui/command_completion.go`, DELETE the temp stub:

```go
// TEMP stub (replaced in Task 3). Allows Task 1 to compile/test in isolation.
func completeLabelArg(a *App, prefix string) []string { return nil }
```

and add the real implementation (it reads the pre-fetched `a.cmd.labelNames`; it does NOT call the network):

```go
// completeLabelArg completes a label-name argument from the pre-fetched label list (a.cmd.labelNames,
// populated off the event loop when the command bar opens). Case-insensitive prefix, sorted.
func completeLabelArg(a *App, prefix string) []string {
	lower := strings.ToLower(prefix)
	var out []string
	for _, name := range a.cmd.labelNames {
		if strings.HasPrefix(strings.ToLower(name), lower) {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}
```

- [ ] **Step 4: Run the tests, verify PASS**

Run: `go test ./internal/tui/ -run 'TestCompleteLabelArg|TestCommandCandidates_LabelArg|TestCommandCandidates_Names' -v`
Expected: PASS (all).

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/command_completion.go internal/tui/command_completion_test.go
git add internal/tui/command_completion.go internal/tui/command_completion_test.go
git commit -m "feat(tui): label-name argument completer for command bar"
```

---

## Task 4: Wire cycling into the command bar; remove the old suggestion map

**Files:**
- Modify: `internal/tui/commands.go` (`showCommandBarWithPrefix` capture + changed func; `hideCommandBar`; remove `generateCommandSuggestion` + `completeCommand`)

- [ ] **Step 1: Pre-fetch label names when the bar opens**

In `showCommandBarWithPrefix` (`internal/tui/commands.go`), immediately after `a.cmd.suggestion = ""` (~line 53), add an async label pre-fetch (the labels API blocks; never call it on the event loop):

```go
	a.cmd.suggestion = ""
	a.cmd.clearCycle()
	a.cmd.labelNames = nil
	// Pre-fetch label names off the event loop so the Tab path can complete them without blocking.
	go func() {
		names := a.userLabelNames()
		a.QueueUpdateDraw(func() { a.cmd.labelNames = names })
	}()
```

- [ ] **Step 2: Replace the Tab handler with cycle logic; add Shift+Tab (Backtab)**

In the same function, replace the existing `case tcell.KeyTab:` block in `input.SetInputCapture`:

```go
		case tcell.KeyTab:
			// Complete using context-aware suggestion; may return full-line replacement
			cur := strings.TrimSpace(input.GetText())
			s := a.generateCommandSuggestion(cur)
			if s != "" && s != cur {
				input.SetText(s)
			}
			return nil
```

with:

```go
		case tcell.KeyTab, tcell.KeyBacktab:
			forward := ev.Key() == tcell.KeyTab
			if len(a.cmd.candidates) == 0 {
				a.cmd.startCycle(a.commandCandidates(input.GetText()))
			}
			if cand, ok := a.cmd.nextCandidate(forward); ok {
				a.cmd.cycling = true
				input.SetText(cand)
				a.cmd.cycling = false
			}
			return nil
```

- [ ] **Step 3: Update SetChangedFunc — clear cycle on user edits, drive hint from candidates**

Replace the existing `input.SetChangedFunc(...)` block:

```go
	input.SetChangedFunc(func(text string) {
		a.cmd.buffer = text
		// Update live hint based on current buffer
		cur := strings.TrimSpace(text)
		s := a.generateCommandSuggestion(cur)
		if s != "" && s != cur {
			hint.SetText("[" + s + "]")
		} else {
			hint.SetText("")
		}
	})
```

with:

```go
	input.SetChangedFunc(func(text string) {
		a.cmd.buffer = text
		// A user edit (not a programmatic cycle step) invalidates the current Tab cycle.
		if !a.cmd.cycling {
			a.cmd.clearCycle()
		}
		// Ghost hint = first candidate, when it adds something beyond what's typed.
		cands := a.commandCandidates(text)
		if len(cands) > 0 && cands[0] != strings.TrimSpace(text) {
			hint.SetText("[" + cands[0] + "]")
		} else {
			hint.SetText("")
		}
	})
```

- [ ] **Step 4: Clear cycle/label cache on close**

In `hideCommandBar` (`commands.go`), after `a.cmd.buffer = ""` (~line 151), add:

```go
	a.cmd.suggestion = ""
	a.cmd.clearCycle()
	a.cmd.labelNames = nil
```

(The `a.cmd.suggestion = ""` line may already exist there — if so, just add the two new lines.)

- [ ] **Step 5: Remove the dead suggestion map and completer**

Delete the entire `generateCommandSuggestion` function (`commands.go` ~214-631) and the `completeCommand` function (~633-640). Then grep for stragglers:

Run: `grep -rn 'generateCommandSuggestion\|completeCommand\b\|a\.cmd\.suggestion' internal/tui/*.go | grep -v _test`
Expected: no references except the `suggestion` field declaration in `command_state.go` (leave the field; it is harmless and still zeroed). If `handleCommandInput` (legacy) references `completeCommand`, replace that one call with `a.cmd.startCycle(a.commandCandidates(a.cmd.buffer)); if c, ok := a.cmd.nextCandidate(true); ok { a.cmd.buffer = c }` OR, simpler, delete the `case tcell.KeyTab:` arm in `handleCommandInput` if that function is unused (verify with `grep -rn 'handleCommandInput' internal/tui/*.go` — if only its own definition matches, it is dead; leaving its Tab arm referencing a deleted function will not compile, so remove that arm).

- [ ] **Step 6: Build + straggler check**

Run: `go build ./...`
Expected: success.

Run: `grep -rn 'generateCommandSuggestion\|func (a \*App) completeCommand' internal/tui/*.go`
Expected: no output.

- [ ] **Step 7: Tests + format**

Run: `go test ./internal/tui/ 2>&1 | tail -3` → `ok`.
Run: `gofmt -w internal/tui/commands.go`.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat(tui): cycle command/arg completions on Tab; remove hardcoded suggestion map"
```

---

## Task 5: Docs + final verification

**Files:**
- Modify: in-app help (`internal/tui/app.go`, the `:` commands help section) and `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Document Tab completion in the in-app help**

In `internal/tui/app.go`, find the command-bar help section (search for `":commands"` or the `Fprintf` block that lists `:` usage — grep `grep -n "command bar\|: to enter\|Commands:" internal/tui/app.go`). Add a line in that section:

```go
	fmt.Fprintf(&help, "    %-18s ⭾  Tab cycles matching commands (Shift+Tab back); after a space, Tab completes label names for :label/:labels/:move\n", "Tab")
```

(Place it adjacent to the existing command-bar help lines; match the surrounding `Fprintf` style/width.)

- [ ] **Step 2: Document in KEYBOARD_SHORTCUTS.md**

In `docs/KEYBOARD_SHORTCUTS.md`, under the command-bar / commands section, add:

```markdown
| `Tab` / `Shift+Tab` | Autocomplete commands | In the `:` bar, cycle through commands that match what you've typed (Shift+Tab reverses). After `:label`/`:labels`/`:move ` + space, Tab completes your label names. |
```

- [ ] **Step 3: Canonical pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!`

- [ ] **Step 4: Race + full suite**

Run: `go test -race ./internal/tui/ 2>&1 | tail -3` → `ok`, no races.
Run: `make test 2>&1 | grep -E '^FAIL' || echo NO_FAILURES` → `NO_FAILURES`.

- [ ] **Step 5: Build the binary**

Run: `make build`
Expected: `Built build/giztui ...`.

- [ ] **Step 6: Live smoke test (pty harness)**

Run (the harness from the smoke-test setup; types `:`, then a partial command, Tab, then quits):

```bash
log=~/.config/giztui/giztui.log; : > "$log"
timeout 30 python3 /tmp/giztui_smoke.py ./build/giztui "8:" "1::" "1:arch" "1:\t" "2:\x1b" "1:q" 2>&1 | tail -40
```

Expected: the command bar shows `archive` (or `archived` after a second Tab) — verify the captured frame contains `archive`. Confirm no panic in the log (`grep -aic panic "$log"` → 0).

- [ ] **Step 7: Commit + finish**

```bash
git add internal/tui/app.go docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs(tui): document Tab command/arg completion in help and shortcuts"
```

Then use the superpowers:finishing-a-development-branch skill. Manual smoke on the user's Mac: open `:`, type `lab`+Tab → `:labels`; type `s`+Tab/Tab → cycles `:search`/`:slack`; type `:labels add ` + Tab → cycles label names. Do NOT push/merge without explicit user confirmation. No Co-Authored-By.

---

## Self-Review

**Spec coverage:**
- Cycle command names/aliases with Tab/Shift+Tab, Enter runs visible → Task 2 (state) + Task 4 (wiring). Enter is unchanged (`input.SetDoneFunc`/`executeCommand`). ✓
- Single match completes fully; no match no-op → `commandCandidates` returns 1 or nil; cycle of length 1 sets that one; nil → `nextCandidate` ok=false → no-op. Task 1/2/4. ✓
- Command registry as single source of truth, replaces hardcoded map → Task 1 (registry) + Task 4 Step 5 (deletion). ✓
- Label arg completion for `:label`/`:labels`/`:move`, extensible `argCompleter` → Task 1 (interface + wiring on 3 specs) + Task 3 (impl). ✓
- No network on Tab path → labels pre-fetched async on open (Task 4 Step 1), completer reads cache (Task 3). ✓
- Cycle state on commandState → Task 2. ✓
- Tests (matcher, cycling, alias→canonical, single-complete, arg split, label filter, drift guard) → Tasks 1-3. ✓
- Ghost hint kept, fed from candidates[0] (the spec's open question, default = keep) → Task 4 Step 3. ✓
- Docs / :help → Task 5. ✓

**Placeholder scan:** No TBD/TODO. The Task 1 temp `completeLabelArg` stub is explicitly introduced and explicitly deleted in Task 3 (named, not vague). Registry is fully enumerated (56 entries). ✓

**Type consistency:** `commandSpec{name, aliases, completeArg}`, `argCompleter func(*App,string)[]string`, `commandCandidates(string)[]string`, `lookupCommand`, `matchesPrefix`, `completeLabelArg`, and `commandState.{candidates, cycleIndex, cycling, labelNames}` + `startCycle/nextCandidate/clearCycle` are named identically across Tasks 1-4. ✓
