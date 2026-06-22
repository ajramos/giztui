# Command Bar Tab Completion — Design

**Date:** 2026-06-22
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `feat/command-tab-completion`

## Goal

Make Tab in the `:` command bar a real autocomplete: cycle through matching command names/aliases, and complete arguments for a starter set of commands (labels). Replace today's hand-maintained prefix→suggestion map (the source of incompleteness and drift) with a single command registry that both name completion and future arg completers derive from.

## Why now / current state

- `internal/tui/commands.go` already wires `Tab → completeCommand()`, but completion is driven by `generateCommandSuggestion`, a hardcoded `map[string][]string` of every prefix → suggestion (hundreds of entries, manually enumerated). New commands don't autocomplete until someone adds all their prefixes; it drifts from the real command set.
- It yields a single guess (`commandState.suggestion`) with no way to see or cycle alternatives (`:s` could be `search` or `slack`).
- The real command set lives in the `executeCommand` switch (`internal/tui/commands.go`, ~95 `case "name", "alias":` lines). There is no shared list to derive completions from.

## Behavior (UX)

- **Command token (first word after `:`):** Tab cycles through commands whose canonical name or alias starts with the typed text, in a stable order (canonical names alphabetical; a command appears once even if multiple aliases match). Shift+Tab cycles backward. Enter executes whatever command text is currently shown. Typing any character refines the buffer and resets the cycle.
  - Exactly one match → Tab completes it fully (`:arch` → `:archive`).
  - No match → Tab is a no-op (no change, no crash).
- **Argument (after `command` + space):** Tab delegates to that command's argument completer, cycling its candidates the same way. v1 ships ONE completer: label names, for `:label`, `:labels`, and `:move`. Completion applies to the last whitespace-separated token (`:labels add wor` → `:labels add Work`); earlier tokens are preserved.
  - Commands without a completer → arg Tab is a no-op.

Completion is prefix-based and case-insensitive. No fuzzy matching, no dropdown menu (out of scope).

## Architecture

New file `internal/tui/command_completion.go` (TUI layer; completion is presentation/input logic, not business logic):

- **Command registry** — a package-level slice, the single source of truth:
  ```go
  type argCompleter func(a *App, prefix string) []string // returns candidates for the current arg token

  type commandSpec struct {
      name      string       // canonical command (e.g. "archive")
      aliases   []string     // e.g. {"a"}
      completeArg argCompleter // nil if the command takes no completable args
  }

  var commandRegistry = []commandSpec{
      {name: "archive", aliases: []string{"a"}},
      {name: "labels", aliases: []string{"l", "label"}, completeArg: completeLabelArg},
      {name: "move", aliases: []string{"m"}, completeArg: completeLabelArg},
      // ... one entry per executeCommand case
  }
  ```
  Entries mirror the `executeCommand` switch cases. (executeCommand itself is unchanged in v1; keeping the registry as the completion source is enough. A later cleanup can have executeCommand consult the registry, but that is out of scope.)

- **Completion engine** — `func (a *App) commandCandidates(buffer string) []string`:
  - If the buffer has no space yet → it's the command token. Match `buffer` (case-insensitive prefix) against every spec's name and aliases; return the set of **canonical names** that matched, de-duplicated, alphabetical. (Cycling shows canonical names, never aliases, so Enter always runs a real command.)
  - If the buffer has a command followed by a space → resolve the command (by name or alias) to its spec; if it has a `completeArg`, split the remainder into tokens, take the last token as the arg prefix, call `completeArg(a, prefix)`, and return each candidate joined back as the full buffer (`command<space>...<space>candidate`).
  - Returns nil when nothing matches.

- **Argument completer for labels** — `func completeLabelArg(a *App, prefix string) []string`: read the user's label names (the label list already available to the App / LabelService), case-insensitive prefix filter, alphabetical. Returns label display names.

- **Cycle state** — lives on `commandState` (`internal/tui/command_state.go`, already holds the buffer): a `candidates []string` slice and `cycleIndex int`. The Tab handler:
  1. If `candidates` is empty or stale (buffer changed since last computed) → recompute via `commandCandidates`, reset index.
  2. Advance index (Tab forward, Shift+Tab back, wrapping), set `buffer` to `candidates[index]`, redraw.
  Any non-Tab key edit clears the cycle so the next Tab recomputes from the new buffer.

## Data flow

`handleCommandInput` (commands.go) `Tab`/`Shift+Tab` → `commandState.cycle(forward)` → uses cached `candidates` or asks `App.commandCandidates(buffer)` → sets `buffer` → redraw. `Enter` → `executeCommand(buffer)` unchanged. The hardcoded `generateCommandSuggestion` map and the single-`suggestion` field are removed (or the inline ghost-suggestion can be re-derived from `candidates[0]` if we keep that affordance — see Open question).

## Error handling

- No candidates → no-op (buffer unchanged).
- Label service / label list unavailable → `completeLabelArg` returns nil → arg Tab is a no-op (never blocks the command bar).
- All matching is pure string work on in-memory data; no network calls in the Tab path (label names come from already-loaded state).

## Testing

- Unit tests (`command_completion_test.go`): prefix match returns canonical names; alias match maps to canonical name; multi-match order is stable/alphabetical; single match completes fully; no match → nil; arg path splits tokens and preserves earlier tokens; `completeLabelArg` filters case-insensitively. Registry-vs-switch coverage test: every `executeCommand` case name has a registry entry (guards against drift) — implemented as a test listing expected names.
- Cycle-state unit test on `commandState`: forward/back wrapping, reset on buffer edit.
- Smoke test via the pty harness ([[smoke-test-harness]]): open `:`, type `arch`, Tab → shows `:archive`; type `s`, Tab/Tab cycles `:search`/`:slack`.

## Out of scope (YAGNI for v1)

- Argument completers other than labels (search operators, Slack channels, prompts) — the `argCompleter` interface makes these drop-in later.
- Fuzzy matching, ranking by usage, dropdown menu UI.
- Reworking `executeCommand` to consume the registry.

## Definition of Done

- [ ] `command_completion.go` with registry + engine + label arg completer.
- [ ] `commandState` gains cycle state + `cycle(forward)`; Tab/Shift+Tab wired in `handleCommandInput`.
- [ ] Hardcoded `generateCommandSuggestion` prefix map removed; completion derives from the registry.
- [ ] Registry covers every `executeCommand` case (drift-guard test).
- [ ] Unit tests + harness smoke test green.
- [ ] In-app `:help` / `docs/KEYBOARD_SHORTCUTS.md` note that Tab cycles command/arg completions (Definition of Done per CLAUDE.md).

## Open question (decide during planning)

Keep the inline ghost-suggestion (greyed `candidates[0]` shown as you type, before pressing Tab)? It's a small nicety the current UI already approximates. Default: keep it, fed from `candidates[0]`, so behavior is strictly better than today.
