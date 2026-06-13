# Config Self-Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Detect default config keys missing from the user's `config.json` and let them add those keys (with default values, preserving their values and `_comment` annotations, with a backup) via `:config migrate` / `--migrate-config`, plus a startup notice.

**Architecture:** A map-based deep-merge in `internal/config` adds only absent keys (recursive, never overwrites, preserves `_*` keys). A `:config migrate` command and a `--migrate-config` flag write the merged file with a `.bak`. Startup notifies when keys are missing. Runtime behavior is unchanged (LoadConfig already merges defaults in memory).

**Tech Stack:** Go (encoding/json map merge), GizTUI config + TUI command + ErrorHandler.

Spec: `docs/superpowers/specs/2026-06-13-config-self-migration-design.md`

---

### Task 1: Core map-merge + file migration

**Files:**
- Create: `internal/config/migrate.go`
- Test: `internal/config/migrate_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/config/migrate_test.go`:

```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeepMergeMissing(t *testing.T) {
	user := map[string]any{
		"_comment": "keep me",
		"llm":      map[string]any{"provider": "ollama"}, // existing, sub-key missing below
		"keep":     "user-value",
	}
	defaults := map[string]any{
		"llm":          map[string]any{"provider": "openai", "model": "x"}, // provider exists (keep user), model missing
		"keep":         "default-value",                                    // exists → must NOT overwrite
		"auto_refresh": map[string]any{"enabled": false, "interval": "5m"}, // wholly missing
	}

	added := deepMergeMissing(user, defaults, "")

	if user["keep"] != "user-value" {
		t.Fatalf("must not overwrite existing value, got %v", user["keep"])
	}
	if user["_comment"] != "keep me" {
		t.Fatal("must preserve _comment key")
	}
	llm := user["llm"].(map[string]any)
	if llm["provider"] != "ollama" {
		t.Fatalf("must keep user's nested value, got %v", llm["provider"])
	}
	if llm["model"] != "x" {
		t.Fatal("must add missing nested key")
	}
	if _, ok := user["auto_refresh"]; !ok {
		t.Fatal("must add wholly-missing top-level key")
	}
	// added paths include the dotted nested path and the top-level key, not "keep".
	got := map[string]bool{}
	for _, p := range added {
		got[p] = true
	}
	if !got["llm.model"] || !got["auto_refresh"] {
		t.Fatalf("added should include llm.model and auto_refresh, got %v", added)
	}
	if got["keep"] {
		t.Fatalf("added should NOT include existing key 'keep', got %v", added)
	}
}

func TestMigrateConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	original := `{
  "_comment": "my notes",
  "llm": { "provider": "ollama" }
}`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	added, backup, err := MigrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) == 0 {
		t.Fatal("expected some added keys")
	}
	if backup == "" {
		t.Fatal("expected a backup path")
	}
	// Backup holds the original bytes.
	if b, _ := os.ReadFile(backup); string(b) != original {
		t.Fatal("backup should contain the original file bytes")
	}
	// Merged file parses, keeps user data + comment, and gained auto_refresh.
	merged := map[string]any{}
	b, _ := os.ReadFile(path)
	if err := json.Unmarshal(b, &merged); err != nil {
		t.Fatalf("merged file invalid: %v", err)
	}
	if merged["_comment"] != "my notes" {
		t.Fatal("merged lost the _comment")
	}
	if _, ok := merged["auto_refresh"]; !ok {
		t.Fatal("merged should contain auto_refresh defaults")
	}
}

func TestMigrateConfigFile_NoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// A full default config → nothing to add.
	full, _ := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err := os.WriteFile(path, full, 0600); err != nil {
		t.Fatal(err)
	}
	added, backup, err := MigrateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 0 || backup != "" {
		t.Fatalf("expected no-op, got added=%v backup=%q", added, backup)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("no .bak should be written on a no-op")
	}
}

func TestMigrateConfigFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{ not json "), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := MigrateConfigFile(path); err == nil {
		t.Fatal("expected an error on invalid JSON")
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatal("must not write a backup when the source is invalid")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run 'TestDeepMergeMissing|TestMigrateConfigFile' -v`
Expected: FAIL — `deepMergeMissing` / `MigrateConfigFile` undefined.

- [ ] **Step 3: Implement `migrate.go`**

Create `internal/config/migrate.go`:

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// deepMergeMissing adds into user every key present in defaults but absent from user, recursing
// into nested objects. It never overwrites an existing user value. Returns the dotted paths added.
func deepMergeMissing(user, defaults map[string]any, prefix string) []string {
	var added []string
	for k, dv := range defaults {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		uv, ok := user[k]
		if !ok {
			user[k] = dv
			added = append(added, path)
			continue
		}
		if dvMap, dok := dv.(map[string]any); dok {
			if uvMap, uok := uv.(map[string]any); uok {
				added = append(added, deepMergeMissing(uvMap, dvMap, path)...)
			}
		}
		// present scalar / type mismatch → keep the user's value.
	}
	return added
}

// defaultConfigMap returns DefaultConfig() as a generic map.
func defaultConfigMap() (map[string]any, error) {
	data, err := json.Marshal(DefaultConfig())
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// readConfigMap reads a config file as a generic map (preserving _comment keys). A missing or
// empty file yields an empty map; invalid JSON is an error.
func readConfigMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("config is not valid JSON: %w", err)
	}
	return m, nil
}

// MissingDefaultKeys returns the dotted paths of default keys absent from the user's config file.
// Read-only (used for the startup notice).
func MissingDefaultKeys(path string) ([]string, error) {
	defaults, err := defaultConfigMap()
	if err != nil {
		return nil, err
	}
	user, err := readConfigMap(path)
	if err != nil {
		return nil, err
	}
	return deepMergeMissing(user, defaults, ""), nil // mutates the local user map; discarded
}

// MigrateConfigFile adds missing default keys to the user's config file, writing a .bak first.
// Returns the added dotted paths and the backup path. No-op (nil, "", nil) when nothing is missing.
// Output is json.MarshalIndent (2-space), keys alphabetically sorted.
func MigrateConfigFile(path string) ([]string, string, error) {
	defaults, err := defaultConfigMap()
	if err != nil {
		return nil, "", err
	}
	user, err := readConfigMap(path)
	if err != nil {
		return nil, "", err
	}
	added := deepMergeMissing(user, defaults, "")
	if len(added) == 0 {
		return nil, "", nil
	}
	out, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return nil, "", err
	}
	backupPath := ""
	if orig, rerr := os.ReadFile(path); rerr == nil {
		backupPath = path + ".bak"
		if werr := os.WriteFile(backupPath, orig, 0600); werr != nil {
			return nil, "", fmt.Errorf("could not write backup %s: %w", backupPath, werr)
		}
	}
	if err := os.WriteFile(path, out, 0600); err != nil {
		return nil, "", err
	}
	return added, backupPath, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run 'TestDeepMergeMissing|TestMigrateConfigFile' -v`
Expected: PASS (all four).

- [ ] **Step 5: Commit**

```bash
git add internal/config/migrate.go internal/config/migrate_test.go
git commit -m "feat(config): map-based self-migration (add missing default keys, preserve comments)"
```

---

### Task 2: `:config migrate` command

**Files:**
- Modify: `internal/tui/commands.go` (`executeCommand` switch + new handler + suggestion)

No unit test (thin command wrapper over Task 1, which is tested; build covers it).

- [ ] **Step 1: Add the command routing**

In `internal/tui/commands.go`, in the `executeCommand` switch, add (near the other cases, e.g. after `"autorefresh"`):

```go
	case "config", "cfg":
		a.executeConfigCommand(args)
```

- [ ] **Step 2: Add the handler**

In `internal/tui/commands.go`, add the function (anywhere among the other `execute*Command` funcs):

```go
// executeConfigCommand handles :config [migrate]. Without a subcommand it prints usage.
func (a *App) executeConfigCommand(args []string) {
	if len(args) > 0 && strings.ToLower(args[0]) == "migrate" {
		go func() {
			path := config.DefaultConfigPath()
			added, backup, err := config.MigrateConfigFile(path)
			if err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Config migrate failed: %v", err))
				return
			}
			if len(added) == 0 {
				a.GetErrorHandler().ShowInfo(a.ctx, "Config is already up to date")
				return
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("✓ Added %d config option(s) (backup: %s). Edit them and restart to apply.", len(added), backup))
		}()
		return
	}
	go a.GetErrorHandler().ShowInfo(a.ctx, "Usage: :config migrate")
}
```

Ensure `internal/tui/commands.go` imports `"github.com/ajramos/giztui/internal/config"` and
`"strings"` / `"fmt"` (add `config` if missing; `strings`/`fmt` are almost certainly present).

- [ ] **Step 3: Add a suggestion entry**

In `generateCommandSuggestion`'s command map (same file), add entries so it autocompletes:

```go
		"config":         {"config"},
		"cfg":            {"config"},
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat(tui): :config migrate command to add missing config options"
```

---

### Task 3: Startup notice

**Files:**
- Modify: `internal/tui/app.go` (`initServices`, after `initErrorHandler`)

- [ ] **Step 1: Add the notice**

In `internal/tui/app.go`, in `initServices()`, immediately after `a.initErrorHandler()` (and before/after the auto-refresh start block), add:

```go
	// Notify (do not modify the file) when the user's config is missing options this version
	// knows about, so they can pull them in with :config migrate.
	if missing, err := config.MissingDefaultKeys(config.DefaultConfigPath()); err == nil && len(missing) > 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, fmt.Sprintf("ℹ %d new config option(s) available — run :config migrate to add them", len(missing)))
	}
```

Confirm `internal/tui/app.go` imports `"github.com/ajramos/giztui/internal/config"` and `"fmt"`
(both already used in app.go).

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): notify on startup when config is missing new default options"
```

---

### Task 4: `--migrate-config` flag

**Files:**
- Modify: `cmd/giztui/main.go` (flag definition + early handling, alongside `--setup`)

- [ ] **Step 1: Define the flag**

In `cmd/giztui/main.go`, next to the other `flag.*` definitions (~line 24-27):

```go
	migrateConfigFlag := flag.Bool("migrate-config", false, "Add missing default options to the config file and exit")
```

- [ ] **Step 2: Handle it before launching the TUI**

In `cmd/giztui/main.go`, after flags are parsed and after `configPath := getConfigPath(*configPathFlag)` is available (mirror how `--setup`/`--version` are handled — early return), add:

```go
	if *migrateConfigFlag {
		added, backup, err := config.MigrateConfigFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Config migrate failed: %v\n", err)
			os.Exit(1)
		}
		if len(added) == 0 {
			fmt.Println("Config is already up to date.")
			return
		}
		fmt.Printf("Added %d option(s) to %s (backup: %s):\n", len(added), configPath, backup)
		for _, k := range added {
			fmt.Printf("  - %s\n", k)
		}
		return
	}
```

Place this after `configPath` is computed (line ~66) and after the `--version`/`--setup` early
handlers so it runs before the TUI launches. `config`, `fmt`, `os` are already imported in main.go.

- [ ] **Step 3: Build + manual check**

Run: `go build ./... && ./build/giztui --help 2>&1 | grep -A1 migrate-config || true`
Expected: build success; `--migrate-config` appears (rebuild with `make build` if needed to refresh the binary).

- [ ] **Step 4: Commit**

```bash
git add cmd/giztui/main.go
git commit -m "feat(cli): --migrate-config flag to add missing config options headlessly"
```

---

### Task 5: Docs — `:help` + CONFIGURATION.md (Definition of Done)

**Files:**
- Modify: `internal/tui/app.go` (`generateHelpText` — the in-app `?` help command list)
- Modify: `docs/CONFIGURATION.md`
- Modify: `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Add `:config migrate` to the in-app help**

In `internal/tui/app.go`, find `generateHelpText()` and the section that lists commands (search for an existing command line such as `":refresh"` or `":undo"`). Add a line in the same format:

```
:config migrate — add new config options to your config.json
```

Match the surrounding `fmt.Fprintf`/string style used in `generateHelpText`.

- [ ] **Step 2: Document in CONFIGURATION.md**

In `docs/CONFIGURATION.md`, add a short subsection (near the top or in a "Maintenance" area):

```markdown
### Keeping your config up to date

New releases may add config options. Your existing `config.json` keeps working (missing keys use
their defaults), but to see and customize new options run:

- In-app: `:config migrate`
- CLI: `giztui --migrate-config`

This adds any missing default keys to your `config.json` (writing a `<config>.bak` backup first)
without touching your existing values or comments. On startup the app also tells you when new
options are available. (Note: the file is re-sorted alphabetically on migrate; the `.bak` keeps
the original.)
```

- [ ] **Step 3: Add to KEYBOARD_SHORTCUTS.md command table**

In `docs/KEYBOARD_SHORTCUTS.md`, add a row near the other `:` commands:

```markdown
| `:config migrate` | Add missing default options to your config.json (backup written) |
```

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go docs/CONFIGURATION.md docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs: document :config migrate (help screen + config + shortcuts)"
```

---

### Task 6: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass (essential tests include `./internal/config`).

- [ ] **Step 2: Config package tests explicitly**

Run: `go test ./internal/config/ -v 2>&1 | tail -15`
Expected: all `ok`, including the four new migration tests.

- [ ] **Step 3: Build + flag smoke test**

Run: `make build && ./build/giztui --migrate-config --config /tmp/giz-migrate-test.json`
Expected: since `/tmp/giz-migrate-test.json` doesn't exist, `readConfigMap` returns an empty map →
all defaults are "added" → it writes the file and prints the added list (no backup, since there
was no original). Inspect `/tmp/giz-migrate-test.json` — it should be a full default config. Then
clean up: `rm -f /tmp/giz-migrate-test.json`.

---

## Self-review notes

- **Spec coverage:** map merge preserving comments + never-overwrite (Task 1); `:config migrate`
  (Task 2); startup notice (Task 3); `--migrate-config` flag (Task 4); `:help` + docs per the new
  CLAUDE.md DoD (Task 5); tests for merge/migrate/no-op/invalid-JSON (Task 1) + verification (Task 6).
  All spec sections mapped. Path resolution: command uses `config.DefaultConfigPath()` (consistent
  with existing `SaveConfig` usage); the flag uses the fully-resolved `configPath` (flag/env/default).
- **Type consistency:** `deepMergeMissing(user, defaults map[string]any, prefix string) []string`,
  `MissingDefaultKeys(path) ([]string, error)`, `MigrateConfigFile(path) ([]string, string, error)`,
  `defaultConfigMap()`, `readConfigMap(path)` — names/signatures match across tasks and call sites.
- **No placeholders:** every code step shows full code; commands have expected output.
- **Threading:** the command handler runs the migration in a goroutine (file IO + ErrorHandler off
  the UI thread); the startup notice is `go`-dispatched.
