# Config self-migration

**Date:** 2026-06-13
**Status:** Approved (design)

## Problem

New config keys are added to the code's `DefaultConfig()` over time, but a user's existing
`config.json` on another machine lacks them. At runtime this is harmless (`LoadConfig` already
starts from `DefaultConfig()` and unmarshals the user file on top, so absent keys keep their
defaults). The real gap is **discoverability/editability**: the user can't see or change new
options that aren't in their file — e.g. they couldn't turn on `auto_refresh` because the block
wasn't present to edit. They want new default keys written into their `config.json` so they can
tweak them.

## Goal

On startup, detect default keys missing from the user's `config.json` and notify them; an
explicit `:config migrate` command (and a `--migrate-config` flag) adds those keys with their
default values, preserving the user's existing values and `_comment` annotations, with a backup.

## Decisions (confirmed with user)

- **Trigger:** detect-on-startup + notify; apply via explicit `:config migrate` command (never
  write the file automatically). Also a `--migrate-config` flag for headless/scripts.
- **Merge:** map-based (not struct) so `_comment`/`_*` keys are preserved; recursive into nested
  objects; never overwrites user values — only adds absent keys.
- **Backup:** write `<config>.bak` before rewriting.
- **Reordering accepted:** the rewrite re-sorts keys alphabetically (Go marshals maps sorted);
  the `.bak` preserves the original order. (Preserving exact order was explicitly de-scoped.)

## Architecture

### A) Core merge — `internal/config/migrate.go` (new)

```go
// deepMergeMissing adds into user every key present in defaults but ABSENT from user (recursing
// into nested map[string]any), never overwriting an existing user value. Returns the dotted
// paths added (e.g. "auto_refresh", "inbox_analyzer.body_char_limit"). Pure; mutates+returns user.
func deepMergeMissing(user, defaults map[string]any, prefix string) (added []string)

// defaultConfigMap returns DefaultConfig() as a generic map (Marshal → Unmarshal).
func defaultConfigMap() (map[string]any, error)

// readConfigMap reads a config file as a generic map (preserves _comment keys). Missing file → {}.
func readConfigMap(path string) (map[string]any, error)

// MissingDefaultKeys returns the dotted paths of default keys absent from the user's config
// (read-only; for the startup notice).
func MissingDefaultKeys(path string) ([]string, error)

// MigrateConfigFile merges missing default keys into the user's config file, writing a .bak
// first. Returns the added paths and the backup path. No-op (added=nil, backup="") if nothing
// is missing. The file is written with json.MarshalIndent (2-space), keys alphabetically sorted.
func MigrateConfigFile(path string) (added []string, backupPath string, err error)
```

`deepMergeMissing` rules per key in `defaults`:
- absent in `user` → deep-copy it in, record the dotted path.
- present in both AND both values are `map[string]any` → recurse (record nested additions).
- present in `user` (any non-map, or type mismatch) → leave the user's value untouched.

Keys present only in `user` (including `_comment`/`_*`) are always kept.

### B) Startup detection + notice — `cmd/giztui/main.go` / app

After config load, compute `MissingDefaultKeys(configPath)`. If non-empty, once the TUI is up
show `ErrorHandler.ShowInfo`: `ℹ N new config option(s) available — run :config migrate to add them`.
(Log the list to the logger as well.) Never writes the file.

### C) Command + flag

- `:config migrate` — extend the existing `:config` command handler (`commands.go`) with a
  `migrate` subcommand: call `config.MigrateConfigFile(a.Config's path)`, then
  `ShowSuccess`: `✓ Added N option(s) to config.json (backup: <path>). Edit them and restart to apply.`
  If nothing missing: `ShowInfo("Config is already up to date")`. (Resolve the active config path
  the same way main.go does — flag/env/default.)
- `--migrate-config` flag (`main.go`, alongside `--setup`): run `MigrateConfigFile`, print the
  result to stdout, exit. For scripts / headless.

### D) Definition-of-Done follow-through (new CLAUDE.md steps)

- Update the in-app `:help` / `?` command list to include `:config migrate`.
- Update `docs/CONFIGURATION.md` (document `:config migrate` / `--migrate-config`).

## Error handling

- Unreadable/!exist config file → treat as empty map (migration would add all defaults). Invalid
  JSON in the user file → return an error, do NOT write (don't clobber a malformed-but-precious file).
- Backup write failure → abort before touching the original.
- All user-facing messages via `ErrorHandler` (`go`-dispatched from the command handler).

## Testing

- `TestDeepMergeMissing` (table): top-level key absent → added + path; nested sub-key absent →
  added + dotted path; existing scalar value → NOT overwritten; existing `_comment` key →
  preserved; key only in user, not in defaults → preserved; type mismatch (user scalar where
  default is object) → user value kept, no panic.
- `TestMigrateConfigFile` (temp file): write a user config missing `auto_refresh` and with a
  `_comment` key → migrate → file now contains `auto_refresh` (defaults), the `_comment` and the
  user's other values are intact, `.bak` exists with the original bytes, returned `added` lists
  `auto_refresh`.
- `TestMigrateConfigFile_NoOp`: a complete config → `added` empty, no `.bak`, file unchanged.
- `TestMigrateConfigFile_InvalidJSON`: malformed file → error, original untouched, no `.bak`.

## Out of scope

- Preserving the original key order (de-scoped; alphabetical sort accepted, backup retained).
- Removing keys that no longer exist in defaults (migration only ADDS; stale keys are left alone).
- Generating `_comment` annotations for newly added keys (added bare; documented in CONFIGURATION.md).
