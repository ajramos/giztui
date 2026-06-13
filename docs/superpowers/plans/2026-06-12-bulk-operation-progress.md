# Bulk Operation Progress Feedback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show live "Archiving 3/10…" progress in the status bar for every bulk operation (archive / trash / mark-read / mark-unread / label), from both the inbox bulk selection and Action Plan dispatch.

**Architecture:** Add a backward-compatible variadic `onProgress ...func(done, total int)` callback to the bulk methods on `EmailService` and `LabelService`; each calls it per item in its operation loop. A TUI helper builds a status-bar progress callback; it's wired at the inbox bulk + Action Plan call sites, and the persistent progress is cleared on completion.

**Tech Stack:** Go, GizTUI services + tview status bar, ErrorHandler.

Spec: `docs/superpowers/specs/2026-06-12-bulk-operation-progress-design.md`

---

### Task 1: Progress callback on EmailService bulk methods

**Files:**
- Modify: `internal/services/interfaces.go` (`EmailService` interface)
- Modify: `internal/services/email_service.go` (4 bulk methods + a helper)
- Test: `internal/services/email_service_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/services/email_service_test.go`:

```go
func TestBulkArchiveProgress(t *testing.T) {
	repo := &MockEmailRepository{}
	repo.On("UpdateMessage", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := NewEmailService(repo, &gmail.Client{}, &render.EmailRenderer{})

	var calls [][2]int
	err := svc.BulkArchive(context.Background(), []string{"a", "b", "c"}, func(done, total int) {
		calls = append(calls, [2]int{done, total})
	})
	assert.NoError(t, err)
	if len(calls) != 3 {
		t.Fatalf("expected 3 progress calls, got %d: %v", len(calls), calls)
	}
	if calls[2] != [2]int{3, 3} {
		t.Fatalf("final progress should be {3,3}, got %v", calls[2])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestBulkArchiveProgress -v`
Expected: FAIL — `BulkArchive` does not accept a callback argument (compile error).

- [ ] **Step 3: Add a progress helper**

In `internal/services/email_service.go`, add near the top (after the imports / before the methods):

```go
// reportProgress invokes the first non-nil progress callback (if any) with (done, total).
// Used by the bulk operations to report per-item progress without forcing every caller to
// pass a callback (the variadic stays empty for callers that don't need progress).
func reportProgress(onProgress []func(done, total int), done, total int) {
	if len(onProgress) > 0 && onProgress[0] != nil {
		onProgress[0](done, total)
	}
}
```

- [ ] **Step 4: Update the four EmailService interface signatures**

In `internal/services/interfaces.go`, change the bulk methods in the `EmailService` interface:

```go
	BulkMarkAsRead(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
	BulkMarkAsUnread(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
	BulkArchive(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
	BulkTrash(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
```

- [ ] **Step 5: Update the four implementations**

In `internal/services/email_service.go`, update each method's signature to add `onProgress ...func(done, total int)` and add a progress call inside its **operation** loop (the second `for _, id := range messageIDs` loop — NOT the undo-capture loop). For each, change the operation loop from:

```go
	for _, id := range messageIDs {
		// ... the existing per-id operation (UpdateMessage / TrashMessage) ...
	}
```

to:

```go
	for i, id := range messageIDs {
		// ... the existing per-id operation (unchanged) ...
		reportProgress(onProgress, i+1, len(messageIDs))
	}
```

Apply to `BulkMarkAsRead`, `BulkMarkAsUnread`, `BulkArchive`, `BulkTrash`. Keep the existing
error collection (`errs`) exactly as-is; just add the index `i` and the `reportProgress` line at
the end of the loop body (progress advances even if a single item errored, matching the
continue-on-error behavior).

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestBulkArchiveProgress -v`
Expected: PASS.

- [ ] **Step 7: Build (confirms existing call sites still compile via empty variadic)**

Run: `go build ./...`
Expected: success — existing `BulkArchive(ctx, ids)` calls are unaffected.

- [ ] **Step 8: Commit**

```bash
git add internal/services/interfaces.go internal/services/email_service.go internal/services/email_service_test.go
git commit -m "feat(services): optional per-item progress callback on EmailService bulk ops"
```

---

### Task 2: Progress callback on LabelService.BulkApplyLabel

**Files:**
- Modify: `internal/services/interfaces.go` (`LabelService` interface)
- Modify: `internal/services/label_service.go`

No unit test (BulkApplyLabel calls the gmail client directly, not the repo mock; the mechanism is
identical to Task 1 which is tested). Build confirms compilation.

- [ ] **Step 1: Update the interface signature**

In `internal/services/interfaces.go`, change `BulkApplyLabel` in the `LabelService` interface:

```go
	BulkApplyLabel(ctx context.Context, messageIDs []string, labelID string, onProgress ...func(done, total int)) error
```

- [ ] **Step 2: Update the implementation**

In `internal/services/label_service.go`, update `BulkApplyLabel`'s signature to add
`onProgress ...func(done, total int)` and change its operation loop from:

```go
	for _, messageID := range messageIDs {
		if err := s.gmailClient.ApplyLabel(messageID, labelID); err != nil {
			errs = append(errs, fmt.Sprintf("failed to apply label to %s: %v", messageID, err))
		}
	}
```

to:

```go
	for i, messageID := range messageIDs {
		if err := s.gmailClient.ApplyLabel(messageID, labelID); err != nil {
			errs = append(errs, fmt.Sprintf("failed to apply label to %s: %v", messageID, err))
		}
		reportProgress(onProgress, i+1, len(messageIDs))
	}
```

(`reportProgress` is defined in Task 1, same `services` package.)

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add internal/services/interfaces.go internal/services/label_service.go
git commit -m "feat(services): optional per-item progress callback on BulkApplyLabel"
```

---

### Task 3: TUI progress helpers

**Files:**
- Create: `internal/tui/bulk_progress.go`
- Test: `internal/tui/bulk_progress_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/bulk_progress_test.go`:

```go
package tui

import "testing"

func TestBulkProgressVerb(t *testing.T) {
	cases := map[string]string{
		"archive":     "Archiving",
		"trash":       "Trashing",
		"mark_read":   "Marking read",
		"mark_unread": "Marking unread",
		"label":       "Applying label",
		"something":   "Processing",
	}
	for action, want := range cases {
		if got := bulkProgressVerb(action); got != want {
			t.Errorf("bulkProgressVerb(%q) = %q, want %q", action, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestBulkProgressVerb -v`
Expected: FAIL — `bulkProgressVerb` undefined.

- [ ] **Step 3: Create the helpers**

Create `internal/tui/bulk_progress.go`:

```go
package tui

import (
	"context"
	"fmt"
)

// bulkProgressVerb maps an action token to its present-participle label for progress messages.
func bulkProgressVerb(action string) string {
	switch action {
	case "archive":
		return "Archiving"
	case "trash":
		return "Trashing"
	case "mark_read":
		return "Marking read"
	case "mark_unread":
		return "Marking unread"
	case "label":
		return "Applying label"
	default:
		return "Processing"
	}
}

// bulkProgress returns a per-item progress callback that updates the status bar with
// "<verb> done/total…". Safe to call from a worker goroutine (ShowProgress marshals to the UI
// thread); MUST NOT be called on the UI goroutine. Pass it to the EmailService/LabelService
// Bulk* methods. Clear the status afterwards with ErrorHandler.ClearPersistentMessage().
func (a *App) bulkProgress(ctx context.Context, verb string) func(done, total int) {
	return func(done, total int) {
		a.GetErrorHandler().ShowProgress(ctx, fmt.Sprintf("%s %d/%d…", verb, done, total))
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestBulkProgressVerb -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/bulk_progress.go internal/tui/bulk_progress_test.go
git commit -m "feat(tui): bulk progress status-bar helpers"
```

---

### Task 4: Wire progress at the bulk call sites

**Files:**
- Modify: `internal/tui/messages_bulk.go` (inbox bulk archive/trash/mark-read/unread)
- Modify: `internal/tui/keys.go` (bulk archive/trash paths, ~2217 / ~2258)
- Modify: `internal/tui/labels.go` (~2029, bulk apply label)
- Modify: `internal/tui/action_plan.go` (~783-787 dispatch, ~817 label)

Each call site already runs inside a goroutine (these are async bulk handlers). Pass the progress
callback and clear the persistent status after the call returns.

- [ ] **Step 1: Wire `messages_bulk.go`**

In `internal/tui/messages_bulk.go`, update the four calls. For the archive call (~line 24):

```go
		err := emailService.BulkArchive(a.ctx, ids)
```
→
```go
		err := emailService.BulkArchive(a.ctx, ids, a.bulkProgress(a.ctx, "Archiving"))
		a.GetErrorHandler().ClearPersistentMessage()
```

For trash (~73):
```go
		err := emailService.BulkTrash(a.ctx, ids, a.bulkProgress(a.ctx, "Trashing"))
		a.GetErrorHandler().ClearPersistentMessage()
```

For the mark read/unread block (~155-157), replace:
```go
			err = emailService.BulkMarkAsUnread(a.ctx, ids)
		} else {
			err = emailService.BulkMarkAsRead(a.ctx, ids)
```
with:
```go
			err = emailService.BulkMarkAsUnread(a.ctx, ids, a.bulkProgress(a.ctx, "Marking unread"))
		} else {
			err = emailService.BulkMarkAsRead(a.ctx, ids, a.bulkProgress(a.ctx, "Marking read"))
```
and add `a.GetErrorHandler().ClearPersistentMessage()` immediately after that `if/else` block (before the existing result handling).

- [ ] **Step 2: Wire `keys.go` bulk archive/trash**

In `internal/tui/keys.go` at ~2217 and ~2258, mirror Step 1:
```go
		err := emailService.BulkArchive(a.ctx, messageIDs, a.bulkProgress(a.ctx, "Archiving"))
		a.GetErrorHandler().ClearPersistentMessage()
```
```go
		err := emailService.BulkTrash(a.ctx, messageIDs, a.bulkProgress(a.ctx, "Trashing"))
		a.GetErrorHandler().ClearPersistentMessage()
```

- [ ] **Step 3: Wire `labels.go` bulk apply label (~2029)**

```go
			err = labelService.BulkApplyLabel(a.ctx, messageIDs, labelID)
```
→
```go
			err = labelService.BulkApplyLabel(a.ctx, messageIDs, labelID, a.bulkProgress(a.ctx, "Applying label"))
			a.GetErrorHandler().ClearPersistentMessage()
```

- [ ] **Step 4: Wire `action_plan.go` dispatch + label**

In `executeActionPlanAction` (~783-787), pass the verb per action:
```go
		case "archive":
			err = emailService.BulkArchive(a.ctx, ids, a.bulkProgress(a.ctx, "Archiving"))
		case "mark_read":
			err = emailService.BulkMarkAsRead(a.ctx, ids, a.bulkProgress(a.ctx, "Marking read"))
		case "trash":
			err = emailService.BulkTrash(a.ctx, ids, a.bulkProgress(a.ctx, "Trashing"))
```
and in `applyActionPlanLabel` (~817):
```go
	return labelService.BulkApplyLabel(a.ctx, ids, labelID, a.bulkProgress(a.ctx, "Applying label"))
```
After the bulk action completes in `executeActionPlanAction` (in its result-handling goroutine, after the `switch`), add `a.GetErrorHandler().ClearPersistentMessage()` before showing the final success toast.

- [ ] **Step 5: Build + full tui tests**

Run: `go build ./... && go test ./internal/tui/ ./internal/services/ 2>&1 | tail -4`
Expected: build success; all `ok`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/messages_bulk.go internal/tui/keys.go internal/tui/labels.go internal/tui/action_plan.go
git commit -m "feat(tui): show per-item progress for inbox + action-plan bulk operations"
```

---

### Task 5: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass.

- [ ] **Step 2: Race detector on the bulk progress path**

Run: `go test -race ./internal/services/ -run 'BulkArchiveProgress' -timeout 120s`
Expected: `ok` (the callback runs in the caller's goroutine; the service loop has no shared state).

- [ ] **Step 3: Build the binary**

Run: `make build`
Expected: success.

(Live E2E — select 10+ messages, archive/trash, watch the status bar count "Archiving 3/10…" then
clear; same for Action Plan dispatch — is deferred to the user's E2E sweep.)

---

## Self-review notes

- **Spec coverage:** variadic callback on the 4 EmailService bulk methods (Task 1) + BulkApplyLabel (Task 2); `reportProgress` helper (Task 1); `bulkProgress`/`bulkProgressVerb` TUI helpers (Task 3); wiring inbox bulk + action-plan dispatch + label, with persistent-clear on completion (Task 4); tests for the repo-based bulk progress + verb mapping (Tasks 1, 3); verification (Task 5). All spec sections mapped. (Label/trash unit tests are out — they use the gmail client directly, not repo-mockable; mechanism identical to the tested archive path.)
- **Type consistency:** `onProgress ...func(done, total int)` identical across all 5 methods, the interfaces, and `reportProgress([]func(done,total int), done, total)`. `bulkProgress(ctx, verb) func(done,total int)` and `bulkProgressVerb(action) string` match between Task 3 and the Task 4 call sites.
- **Threading:** callbacks run in the existing bulk goroutines (off the UI thread) → `ShowProgress`→`QueueUpdateDraw` is safe; `ClearPersistentMessage` after each bulk call prevents the lingering-status bug (same class as the v1.8.1 "Fetching" fix).
- **No placeholders:** every code step shows full code; commands have expected output.
