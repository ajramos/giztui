# Command Typo Suggestion ("Did you mean?") — Design

**Date:** 2026-06-28
**Status:** Approved (brainstorming) — pending plan
**Branch:** `feat/command-typo-suggestion`
**Issue:** #30

## Goal

When a typed `:` command is unknown but close to a real one, enrich the "Unknown command" status
message with the nearest match: `Unknown command: 'archvie'. Did you mean ':archive'?`. Informational
only — the user retypes (or uses Tab completion). Builds on the `commandRegistry`/`lookupCommand`
already added for tab completion.

## Behavior (chosen)

- On Enter, `executeCommand` parses the first token as `command`. Its `default:` case (after the
  numeric-shortcut check) shows `Unknown command: %s`. We replace that with a suggestion-aware message.
- If a close match exists → `Unknown command: '<typed>'. Did you mean ':<canonical>'?`
- If no close match → the existing `Unknown command: <typed>` (unchanged).
- No re-opening the bar, no auto-run (explicitly rejected — safe + simple, the issue's core).

## Architecture

In `internal/tui/command_completion.go` (next to the registry):

```go
// closestCommand returns the canonical command name nearest to typed (case-insensitive Levenshtein
// over every registry name and alias), with ok=true only when it is a confident suggestion.
func (a *App) closestCommand(typed string) (string, bool)

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int
```

`closestCommand` logic:
- Lowercase `typed`. If `len(typed) < 3` → return ("", false) (avoid noisy suggestions for 1–2 char
  typos that collide with single-letter aliases like `a`/`s`/`i`).
- Walk `commandRegistry`; for each spec, measure `levenshtein(typed, name)` and `levenshtein(typed, alias)`
  but only against candidates with `len >= 3` (suggest real words, not single letters). Track the spec
  with the smallest distance.
- Return `(spec.name, true)` only when the best distance is `> 0` and `<= 2`. (Distance 0 can't happen
  here — an exact match would have hit a real case, not `default:`.)
- Ties: the registry's order is deterministic; first-found wins (stable). Good enough.

`levenshtein`: standard two-row DP, O(len(a)·len(b)), tiny inputs.

In `executeCommand` `default:` case (`internal/tui/commands.go` ~431-436):
```go
default:
    if matched := a.executeNumericShortcut(command); !matched {
        if suggestion, ok := a.closestCommand(command); ok {
            a.showError(fmt.Sprintf("Unknown command: '%s'. Did you mean ':%s'?", command, suggestion))
        } else {
            a.showError(fmt.Sprintf("Unknown command: %s", command))
        }
    }
```

## Data flow

`executeCommand(cmd)` → split → `command` (first token) → switch → `default` → `closestCommand(command)`
→ `showError(...)`. Pure in-memory string work; no I/O, event-loop only.

## Error handling

- Empty/short input → no suggestion (handled by the `< 3` guard); the existing message shows.
- No candidate within distance 2 → existing message (no suggestion).
- `closestCommand` never panics (bounds-safe DP, empty registry → no match).

## Testing

`command_completion_test.go` (append):
- `levenshtein`: `("kitten","sitting")==3`, `("archive","archvie")==2`? (transposition = 2 edits in
  plain Levenshtein), `("a","a")==0`, `("","abc")==3`.
- `closestCommand`: `"archvie"`→`("archive",true)`; `"lables"`→`("labels",true)`; `"summarize"`-ish far
  typo with distance >2 → `("",false)`; `"xy"` (len<3) → `("",false)`; a real-but-unknown long token
  far from everything → `("",false)`.

Harness smoke: open `:`, type `archvie`, Enter → status bar contains `Did you mean ':archive'?`; verify
via the log (the status message is logged) — no panic.

## Out of scope (YAGNI)

- Subcommand typo suggestions (`:prompt lst`).
- Auto-run or reopen-the-bar affordances.
- Configurable aliases (#28) — when that lands, the registry already feeds this for free.

## Definition of Done

- [ ] `closestCommand` + `levenshtein` + unit tests.
- [ ] `executeCommand` default case shows the suggestion when one exists.
- [ ] Harness smoke confirms the message; gate green.
- [ ] No behavior change when there is no close match.
