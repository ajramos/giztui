# Bulk operation progress feedback

**Date:** 2026-06-12
**Status:** Approved (design)

## Problem

Selecting many messages and running a bulk action (archive / trash / mark-read / mark-unread /
label) gives no incremental feedback. For 10+ messages the app looks hung until the whole
operation finishes. The user wants a live "Archiving 3/10…" indicator.

## Goal

Show per-item progress in the status bar for every bulk operation, both from the inbox bulk
selection and from Action Plan group dispatch, and clear it cleanly when done.

## Decisions (confirmed with user)

- Scope: **all** bulk ops (archive, trash, mark-read, mark-unread, label), in **both** the inbox
  bulk path and the Action Plan dispatch.
- No throttle: update per item (batch sizes are ≤ ~50; one repaint per item, spread over the
  per-item API latency, does not flood the UI).
- Clear the persistent progress message on completion (reusing the lesson from the "Fetching…"
  bug: a persistent `ShowProgress` message must be explicitly cleared).

## Architecture

### A) Optional progress callback on the bulk service methods

Add a backward-compatible **variadic** progress callback so existing call sites keep compiling:

`internal/services/interfaces.go` + `email_service.go`:
```go
BulkArchive(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
BulkTrash(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
BulkMarkAsRead(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
BulkMarkAsUnread(ctx context.Context, messageIDs []string, onProgress ...func(done, total int)) error
```

`internal/services/interfaces.go` + `label_service.go`:
```go
BulkApplyLabel(ctx context.Context, messageIDs []string, labelID string, onProgress ...func(done, total int)) error
```

Each method, inside its existing per-id loop, calls the callback (when provided) after each
message with `(done, total)` where `done` is the count processed so far (1-based) and
`total = len(messageIDs)`. A small unexported helper `reportProgress(onProgress, done, total)`
keeps the call uniform (invokes `onProgress[0]` only if `len(onProgress) > 0`).

Existing callers — `BulkArchive(ctx, ids)` etc. — are unchanged (empty variadic).

### B) TUI progress helper — `internal/tui/bulk_progress.go` (new)

```go
// bulkProgress returns a per-item progress callback that updates the status bar. verb is the
// present participle shown to the user ("Archiving", "Trashing", …). Safe to call from a worker
// goroutine (ShowProgress marshals to the UI thread); MUST NOT be called on the UI goroutine.
func (a *App) bulkProgress(ctx context.Context, verb string) func(done, total int)
```

Returns `func(done, total int) { a.GetErrorHandler().ShowProgress(ctx, fmt.Sprintf("%s %d/%d…", verb, done, total)) }`.

A second tiny pure helper formats the verb per action for reuse/testing:
`bulkProgressVerb(action string) string` → "Archiving" / "Trashing" / "Marking read" /
"Marking unread" / "Applying label".

### C) Wiring

- **Inbox bulk handlers** (the bulk archive/trash/mark-read/mark-unread/label paths): pass
  `a.bulkProgress(a.ctx, verb)` to the corresponding `Bulk*` call. These already run off the UI
  thread; confirm each runs in a goroutine before wiring (if any runs synchronously, wrap it in
  `go` first).
- **Action Plan dispatch** (`executeActionPlanAction` in `action_plan.go`, already in a
  `go func()`): pass `a.bulkProgress(a.ctx, verb)` to `BulkArchive` / `BulkMarkAsRead` /
  `BulkTrash` and to `applyActionPlanLabel` → `BulkApplyLabel`.
- On completion, after the existing success toast, call
  `a.GetErrorHandler().ClearPersistentMessage()` (or rely on the success `Show*` replacing the
  status) so the persistent progress never lingers.

## Error handling / threading

- The callback calls `ErrorHandler.ShowProgress` → `QueueUpdateDraw`, which marshals to the UI
  thread. It is invoked from the service loop, which runs on the caller's worker goroutine — never
  the UI goroutine — so there is no deadlock. (The plan verifies every call site dispatches the
  bulk op in a goroutine.)
- On error mid-loop the methods keep their current behavior (collect errors / continue); progress
  simply stops advancing. The final success/error toast replaces the persistent progress; the
  completion path clears it regardless.

## Testing

- `TestBulkArchiveProgress` (services): a counting callback passed to `BulkArchive` over 3 IDs is
  invoked 3 times, ending with `(3, 3)`. (Reuses the existing email_service repo-mock setup.)
- `TestBulkApplyLabelProgress` (services): same shape for the label service.
- `TestBulkProgressVerb` (tui): maps each action token to its participle; unknown → a sensible
  default ("Processing").
- Existing bulk tests must still pass (variadic keeps old call sites valid).

## Out of scope

- A progress bar widget (status-bar text only).
- Throttling / coalescing (per-item is sufficient at current batch sizes; revisit only if a real
  flood appears).
- Cancel-mid-bulk (separate feature).
