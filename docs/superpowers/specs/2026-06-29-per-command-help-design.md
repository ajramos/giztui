# Per-Command Help (`:help <cmd>`) — Design

**Date:** 2026-06-29
**Status:** Approved (brainstorming) — pending plan
**Branch:** `feat/per-command-help`
**Issue:** #29

## Goal

`:help <cmd>` shows focused help for a single command (description, syntax, examples, aliases) in the
reader pane, rendered exactly like the full `?` help screen (same overlay, same Esc-to-return), so the
UX is consistent. Help text lives in the command registry, not in handlers.

## Scope (decided)

- Rich help (summary + syntax + examples) only for the commands that take arguments/operators:
  `search`, `labels`, `label`, `move`, `theme`, `prompt`, `bookmark`, `slack`, `accounts`. Everything
  else gets an **auto-generated fallback** (name + aliases + "no detailed help; press ? for the full
  list"). No need to write 56 cards.
- Only the `:help <cmd>` trigger. The `:<cmd> ?` form is intentionally OUT (scope creep + would touch
  the executeCommand hot path).

## Current mechanism (verified)

- The full help (`?` / `:help`) renders via `toggleHelp()` (app.go ~2446). Its "show" branch backs up
  the reader (text/header/title → `helpBackup`), hides the header, sets the textContainer title to
  ` 📚 Help & Shortcuts `, puts `generateHelpText()` into the reader, sets `showHelp = true`, focuses
  the reader. Its "restore" branch (on the next toggle / Esc) restores from `helpBackup` and clears
  `showHelp`. Esc while `showHelp` calls `toggleHelp()`.
- `:help` is `executeHelpCommand(args)` → currently just `a.toggleHelp()`.
- `commandSpec` (command_completion.go): `{name, aliases, completeArg}`. `lookupCommand(token)` resolves
  a name/alias to its spec.

## Architecture

**1. Help content on the registry** (`internal/tui/command_completion.go`):
```go
type cmdHelp struct {
	summary  string   // one-line description
	syntax   string   // e.g. ":search <query>"
	examples []string // e.g. {":search from:ana has:attachment", ":search is:unread after:2026/01/01"}
}

type commandSpec struct {
	name        string
	aliases     []string
	completeArg argCompleter
	help        *cmdHelp // nil → auto fallback
}
```
Populate `help` for the 9 rich commands (concrete text in the plan).

**2. Render text** (`command_help.go`, new): `func (a *App) generateCommandHelpText(s *commandSpec) string`
builds the focused help — a title line ` :<name> `, the summary, a `Syntax:` block, an `Examples:`
block (only when `s.help` is non-nil), an `Aliases:` line, and a trailing "Press Esc to return." When
`s.help == nil`, it produces the fallback: name + aliases + "No detailed help for this command — press
? for the full command/shortcut list." Uses the same color-tag style as `generateHelpText`.

**3. Reuse the help overlay** — extract the "show" branch of `toggleHelp` into:
```go
// showHelpScreen backs up the reader (only if not already showing), then renders content in the
// reader pane with the given title and marks showHelp. Esc restores via toggleHelp's existing path.
func (a *App) showHelpScreen(content, title string)
```
`toggleHelp()`'s show branch becomes `a.showHelpScreen(a.generateHelpText(), " 📚 Help & Shortcuts ")`.
The backup is taken only when `!a.showHelp` so re-rendering (e.g. `:help search` while help is open)
doesn't clobber the original reader backup. The restore branch of `toggleHelp` is unchanged and serves
both full and per-command help (Esc → toggleHelp → restore).

**4. Wire `:help <cmd>`** — `executeHelpCommand(args)`:
```go
func (a *App) executeHelpCommand(args []string) {
	if len(args) > 0 {
		if s := lookupCommand(args[0]); s != nil {
			a.showHelpScreen(a.generateCommandHelpText(s), " 📚 :"+s.name+" ")
			return
		}
	}
	a.toggleHelp() // no arg, or unknown command → full help screen (toggle)
}
```

## Data flow

`:help search` → `executeCommand` → `executeHelpCommand(["search"])` → `lookupCommand("search")` → spec
→ `generateCommandHelpText(spec)` → `showHelpScreen(text, " 📚 :search ")` → reader shows it, `showHelp=true`.
Esc → `toggleHelp()` restore branch → reader restored. Pure event-loop; no I/O.

## Error handling

- `:help` (no arg) → full help (existing toggle).
- `:help <unknown>` → full help (the user still gets help). No error noise.
- `showHelpScreen` guards the backup on `!a.showHelp`, so it is safe to call when help is already open.

## Testing

- `command_help_test.go`: `generateCommandHelpText` for a RICH command (contains summary + a `Syntax:`
  line + an example string + alias) and for a FALLBACK command (contains the name and the "No detailed
  help" sentence, no `Syntax:` block). Build the `*commandSpec` directly in the test (no App for the
  text-builder if it's a pure helper; if it needs `a` for colors, construct a minimal App with the
  theme — match how `help_text_test.go` constructs App).
- Harness smoke: `:help search` → reader shows the search help (grep the captured frame for "Operators"
  or "Examples"); Esc → reader restored to the message; no panic. Also `:help archive` → fallback text.

## Out of scope (YAGNI)

- `:<cmd> ?` trigger; subcommand help (`:prompt list ?`); configurable aliases (#28).
- Rewriting the full `?` help.

## Definition of Done

- [ ] `cmdHelp` + `help` field; rich help for the 9 arg commands.
- [ ] `generateCommandHelpText` (rich + fallback) + tests.
- [ ] `showHelpScreen` extracted; `toggleHelp` uses it; `:help <cmd>` wired.
- [ ] Harness: `:help search` shows it + Esc restores; gate green.
- [ ] In-app `?` help: add a one-line note that `:help <cmd>` exists (Definition of Done per CLAUDE.md).
- [ ] No behavior change to the full `?` help.
