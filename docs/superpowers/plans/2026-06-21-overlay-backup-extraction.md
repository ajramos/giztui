# Overlay Backup Extraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse the three identical reader-backup mechanisms (help, preload status, prompt stats) into one reusable `overlayBackup` type — 9 flat App fields become 3 typed ones — with zero user-visible behavior change.

**Architecture:** New `overlayBackup{text,header,title}` (in `internal/tui/overlay_backup.go`) with `active()` and `clear()` methods. `App` composes three instances (`helpBackup`, `preloadBackup`, `promptStatsBackup`). Field reads/writes go through `a.<x>Backup.text/header/title`; the restore guard becomes `active()`; the clear block becomes `clear()`. Event-loop only, no mutex.

**Tech Stack:** Go, tview, standard `testing`.

---

## Reference facts (verified in code)

- Nine fields on `App` (app.go ~78-86): `helpBackupText/Header/Title`, `preloadBackupText/Header/Title`, `promptStatsBackupText/Header/Title` — all `string`. No struct-literal initializers (zero value `""`).
- All 33 accesses are in `app.go`, on the event loop. No goroutine touches them.
- Per overlay, the SAME three operations:
  - **Save** (before showing): three independent `if widget, ok := ...; ok { a.<x>BackupField = widget.Get... }` blocks (help ~2563-2571, preload ~2625-2631, promptStats ~2743-2749). The fragmented `if`-guards must be preserved — only the assignment target changes (`a.helpBackupText` → `a.helpBackup.text`).
  - **Restore** (on close, guarded by `if ... && a.<x>BackupText != ""`): reads the three fields to `SetContent`/`SetText`/`SetTitle` (help ~2517-2546, preload ~2693-2722, promptStats ~2814-2843).
  - **Clear** (after restore): `a.<x>BackupText = ""` / `Header = ""` / `Title = ""` (help ~2551-2553, preload ~2727-2729, promptStats ~2848-2850).

---

## Task 1: Create `overlayBackup` type + tests

**Files:**
- Create: `internal/tui/overlay_backup.go`
- Test: `internal/tui/overlay_backup_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/overlay_backup_test.go`:

```go
package tui

import "testing"

func TestOverlayBackup_ActiveAndClear(t *testing.T) {
	var b overlayBackup
	// Zero value is inactive.
	if b.active() {
		t.Fatal("zero overlayBackup must be inactive")
	}
	// A non-empty text marks it active (the restore guard).
	b.text = "body"
	b.header = "hdr"
	b.title = "ttl"
	if !b.active() {
		t.Fatal("backup with non-empty text must be active")
	}
	// clear resets all three fields and makes it inactive.
	b.clear()
	if b.text != "" || b.header != "" || b.title != "" {
		t.Fatalf("clear left state: %q %q %q", b.text, b.header, b.title)
	}
	if b.active() {
		t.Fatal("cleared backup must be inactive")
	}
	// active keys off text specifically (header/title alone do not count).
	var b2 overlayBackup
	b2.header = "hdr"
	if b2.active() {
		t.Fatal("active must key off text, not header")
	}
}
```

- [ ] **Step 2: Run it, verify FAIL (overlayBackup undefined)**

Run: `go test ./internal/tui/ -run TestOverlayBackup -v`
Expected: FAIL — `overlayBackup` undefined.

- [ ] **Step 3: Implement the type**

Create `internal/tui/overlay_backup.go`:

```go
package tui

// overlayBackup snapshots the reader pane's content (text/header/title) before a full-pane overlay
// (help, preload status, prompt stats) replaces it, so the prior content can be restored on close.
// Event-loop only — no synchronization needed. Fields are written directly (the save sites guard each
// widget independently); active()/clear() centralize the restore guard and the reset.
type overlayBackup struct {
	text   string
	header string
	title  string
}

// active reports whether a backup is currently held — keyed off text, matching the original
// `<x>BackupText != ""` restore guard.
func (b *overlayBackup) active() bool { return b.text != "" }

// clear resets the backup to empty (called after restoring).
func (b *overlayBackup) clear() {
	b.text = ""
	b.header = ""
	b.title = ""
}
```

- [ ] **Step 4: Run the test, verify PASS**

Run: `go test ./internal/tui/ -run TestOverlayBackup -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/tui/overlay_backup.go internal/tui/overlay_backup_test.go
git add internal/tui/overlay_backup.go internal/tui/overlay_backup_test.go
git commit -m "feat(tui): add overlayBackup type with active/clear + tests"
```

(Do NOT add a Co-Authored-By line — the project forbids it.)

---

## Task 2: Rewire `App` to three `overlayBackup` fields

**Files:**
- Modify: `internal/tui/app.go` (field block ~78-86; the three save/restore/clear sites)

- [ ] **Step 1: Replace the nine App fields with three**

In `internal/tui/app.go`, replace the block (~78-86):

```go
	helpBackupText          string // Backup of text content before showing help
	helpBackupHeader        string // Backup of header content before showing help
	helpBackupTitle         string // Backup of text container title before showing help
	preloadStatusVisible    bool
	preloadBackupText       string // Backup of text content before showing preload status
	preloadBackupHeader     string // Backup of header content before showing preload status
	preloadBackupTitle      string // Backup of text container title before showing preload status
	promptStatsVisible      bool
	promptStatsBackupText   string // Backup of text content before showing prompt stats
	promptStatsBackupHeader string // Backup of header content before showing prompt stats
	promptStatsBackupTitle  string // Backup of text container title before showing prompt stats
```

with (keep `preloadStatusVisible` and `promptStatsVisible` — they are NOT backup fields):

```go
	// Reader-content backups for full-pane overlays (type in overlay_backup.go)
	helpBackup           overlayBackup
	preloadStatusVisible bool
	preloadBackup        overlayBackup
	promptStatsVisible   bool
	promptStatsBackup    overlayBackup
```

IMPORTANT: only the 9 `*Backup*` fields are replaced. `preloadStatusVisible` and `promptStatsVisible`
are interleaved in that block and MUST be preserved as plain bool fields.

- [ ] **Step 2: Rewire the field reads/writes (mechanical rename across app.go)**

Apply these exact renames everywhere in `app.go`:
- `a.helpBackupText` → `a.helpBackup.text`, `a.helpBackupHeader` → `a.helpBackup.header`, `a.helpBackupTitle` → `a.helpBackup.title`
- `a.preloadBackupText` → `a.preloadBackup.text`, `a.preloadBackupHeader` → `a.preloadBackup.header`, `a.preloadBackupTitle` → `a.preloadBackup.title`
- `a.promptStatsBackupText` → `a.promptStatsBackup.text`, `a.promptStatsBackupHeader` → `a.promptStatsBackup.header`, `a.promptStatsBackupTitle` → `a.promptStatsBackup.title`

This covers the save sites (e.g. `a.helpBackup.text = text.GetText(false)`) and the restore reads
(e.g. `a.enhancedTextView.SetContent(a.helpBackup.text)`), preserving the fragmented `if widget, ok`
save guards exactly.

- [ ] **Step 3: Use `active()` for the three restore guards**

Replace each restore guard:
- help (~2517) `if a.enhancedTextView != nil && a.helpBackupText != "" {` → `if a.enhancedTextView != nil && a.helpBackup.active() {`
- preload (~2693) `if a.enhancedTextView != nil && a.preloadBackupText != "" {` → `if a.enhancedTextView != nil && a.preloadBackup.active() {`
- promptStats (~2814) `if a.enhancedTextView != nil && a.promptStatsBackupText != "" {` → `if a.enhancedTextView != nil && a.promptStatsBackup.active() {`

(If any of these reads `a.helpBackup.text != ""` after the Step-2 rename, change it to `a.helpBackup.active()`.)

- [ ] **Step 4: Use `clear()` for the three clear blocks**

Replace each 3-line clear block:
- help (~2551-2553):
```go
		a.helpBackup.text = ""
		a.helpBackup.header = ""
		a.helpBackup.title = ""
```
→ `a.helpBackup.clear()`
- preload (~2727-2729): the three `a.preloadBackup.* = ""` → `a.preloadBackup.clear()`
- promptStats (~2848-2850): the three `a.promptStatsBackup.* = ""` → `a.promptStatsBackup.clear()`

- [ ] **Step 5: Build and check for stragglers**

Run: `go build ./...`
Expected: success.

Run: `grep -nE 'a\.(help|preload|promptStats)Backup(Text|Header|Title)\b' internal/tui/*.go | grep -v _test`
Expected: no output. (`a.helpBackup.text` etc. do not match this pattern.)

- [ ] **Step 6: Tests**

Run: `go test ./internal/tui/ 2>&1 | tail -3` → `ok`.
Run `gofmt -w internal/tui/app.go`.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go
git commit -m "refactor(tui): route overlay backups through overlayBackup (9 App fields → 3)"
```

(Do NOT add a Co-Authored-By line.)

---

## Task 3: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Canonical pre-commit check**

Run: `make pre-commit-check`
Expected: `All pre-commit checks passed!`

- [ ] **Step 2: Full test suite**

Run: `make test 2>&1 | grep -E "^(ok|FAIL)" | grep -v "no test files"`
Expected: all `ok`, no `FAIL`.

- [ ] **Step 3: Race detector on the TUI package**

Run: `go test -race ./internal/tui/ 2>&1 | tail -3`
Expected: `ok`, no race warnings.

- [ ] **Step 4: Build the binary**

Run: `make build`
Expected: `Built build/giztui ...`.

- [ ] **Step 5: Finish the branch**

Use the superpowers:finishing-a-development-branch skill. Manual smoke test on the user's Mac before
merge: open and close `?` help — the reader content (body, header, title) returns; same for the
prompt-stats and preload-status overlays. Do NOT push/merge without explicit user confirmation
(project rule: "commit" ≠ "publish"). Do NOT add a Co-Authored-By line.

---

## Self-Review

**Spec coverage:**
- `overlayBackup` type + `active`/`clear` → Task 1. ✓
- App composes 3 `overlayBackup` fields; 9 flat fields removed (visible-flag fields preserved) → Task 2 Step 1. ✓
- Save/restore field access rewired (fragmented if-guards preserved) → Task 2 Step 2. ✓
- Restore guards use `active()`; clear blocks use `clear()` → Task 2 Steps 3-4. ✓
- `overlay_backup_test.go` covering active/clear → Task 1. ✓
- No stray `a.<x>Backup(Text|Header|Title)` refs → Task 2 Step 5 grep. ✓
- `make pre-commit-check` + `-race` green → Task 3. ✓
- No user-visible behavior change → save sites keep their per-widget guards; `active()` is exactly `text != ""`; `clear()` is exactly the 3 resets; manual smoke test Task 3 Step 5. ✓

**Type/signature consistency:** `overlayBackup{text,header,title}` + `active() bool` + `clear()` defined in Task 1 and used identically in Task 2. `helpBackup`/`preloadBackup`/`promptStatsBackup` named consistently. ✓

**Placeholder scan:** no TBD/TODO; every code step shows before/after. The Step-2 mechanical rename is a complete, enumerated mapping backed by the Step-5 straggler grep — not a placeholder. ✓
