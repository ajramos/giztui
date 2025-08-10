# Status Bar and Flash System — Refactor & Implementation Prompt

You are a senior Go/TUI engineer working on a `tview`-based Gmail TUI. Refactor and enhance the status/flash messaging system with strong UI thread-safety, consistent UX, and clear APIs. Follow the project’s architecture and module boundaries.

## Context (Current State)
- Status bar is created in `tui/layout.go` via `createStatusBar()` and stored in `a.views["status"]`.
- Helpers in `tui/status.go`:
  - `showStatusMessage` (transient), `setStatusPersistent` (persistent), `showError`, `showInfo`.
- Calls spread across `tui/messages.go` and `tui/keys.go` for user feedback.
- There is a top `Flash` primitive (`a.flash.textView`) already added to the layout but currently unused for long-running/critical notices.

## Problems Observed
- UI thread safety: status helpers write to `tview.TextView` directly from goroutines without `a.QueueUpdateDraw(...)`.
- Race conditions: transient reset after a fixed delay may override a newer message (no cancellation/tokening).
- Inconsistency: baseline text differs — `createStatusBar` uses “GizTUI …” while reset uses “Gmail TUI …”.
- UX: no severity-based styling; no structured baseline context (mode, selection, query, etc.).
- Flash bar is underutilized, missing a role distinction from the bottom status bar.

## Objectives
- Provide a unified, thread-safe, ergonomic API for both transient and persistent messages.
- Enforce consistent baseline and visual language (icons/colors) for Info/Success/Warning/Error.
- Avoid message races with cancellation tokens for transients.
- Clarify responsibilities: bottom status bar for short feedback and context; top Flash for long-running operations and critical alerts.
- Keep changes aligned with `tui` module boundaries and concurrency rules.

## Non-Goals
- No new networking behavior; no change to business logic of messages/labels/AI.
- No redesign of the entire layout beyond status/flash behavior.

## Constraints & Rules
- All UI mutations must happen via `a.QueueUpdateDraw(...)`.
- No network calls inside input handlers; do them in goroutines and surface results via `QueueUpdateDraw`.
- Keep feature logic out of `keys.go`; route to functions in the right files.
- Maintain `tui/` file responsibilities:
  - `status.go`: status/flash helpers only.
  - `layout.go`: create views and wire them once.
- Documentation and code in English; user-facing strings can include emoji but keep clarity.

## Target UX
- Bottom status bar:
  - Shows brief feedback like “✅ Archived”, “❌ Error …”, “ℹ️ Info …”.
  - Displays a baseline like: `Gmail TUI | <context> | Press ? for help | Press q to quit`.
  - Context may include: mode (Bulk), selection `row/total`, active search/label.
- Top Flash bar:
  - Used for long-running actions and critical alerts (e.g., “Archiving 134 messages…”), optionally with a spinner.
  - Dismissed automatically on completion or explicitly via `Hide()`.

## API Design (status.go)
Introduce a small, typed API with severity and cancellation for transients.

```go
// Severity classification for styling
type Severity int
const (
    SevInfo Severity = iota
    SevSuccess
    SevWarn
    SevError
)

// Baseline management
func (a *App) SetStatusBaseline(text string)

// Persistent stateful message (no auto-clear)
func (a *App) SetStatusPersistent(text string, sev Severity)

// Transient message with duration and race-free cancellation
func (a *App) SetStatusTransient(text string, d time.Duration, sev Severity)

// Convenience helpers
func (a *App) ShowInfo(msg string)
func (a *App) ShowSuccess(msg string)
func (a *App) ShowWarn(msg string)
func (a *App) ShowError(msg string)
```

Implementation details:
- Wrap all `TextView` mutations in `a.QueueUpdateDraw`.
- Keep a `statusTransientToken` (e.g., `uint64` or UUID). On each transient set, store a new token; the reset goroutine checks that its token matches before applying the baseline.
- Use `tview` color tags for severities: red for error, yellow for warn, green for success, blue/neutral for info.
- Unify baseline text: `Gmail TUI | Press ? for help | Press q to quit`.
- Provide internal helpers to compose a baseline with optional context segments (bulk mode, selection, query).

## API Design (Flash — status.go or dedicated flash.go)
```go
// High-visibility flash area (top of layout)
func (a *App) FlashShow(text string, sev Severity)
func (a *App) FlashShowWithSpinner(text string, sev Severity)
func (a *App) FlashHide()
```

Implementation details:
- Ensure `Flash` shows/hides via `QueueUpdateDraw`.
- Optional: simple spinner using a goroutine that updates text periodically while a `flashActive` flag is true.
- Severity styles consistent with status bar.

## Step-by-Step Plan
1) Infrastructure
   - Unify baseline in `layout.go` and `status.go` (“Gmail TUI …”).
   - Implement `SetStatusBaseline`, `SetStatusPersistent`, `SetStatusTransient` with token-based cancellation and UI-thread safety.
   - Implement severity helpers and styling.
2) Replace call sites (minimal risk first)
   - In `keys.go` and `messages.go`, replace `showStatusMessage/setStatusPersistent/showError/showInfo` with new helpers.
   - Ensure selection-change updates either call `SetStatusPersistent` or are debounced if too frequent.
3) Flash adoption
   - Implement `FlashShow/FlashHide` and route long-running bulk operations (`archive`, `trash`, `load more`) to use Flash for progress and keep status bar for short confirmations.
4) Context enrichment (optional, time-boxed)
   - Compose baseline with live context: mode (Bulk), selected count, current query/label.
   - Keep baseline stable to avoid flicker; update only when relevant state changes.

## Acceptance Criteria
- No direct `TextView` mutations from background goroutines; all through `QueueUpdateDraw`.
- Transient messages never override newer content after they expire (tokenized cancellation works).
- Baseline text is consistent and correct (“Gmail TUI …”).
- Severity-based coloring is applied and readable on dark and light themes.
- Flash appears for long-running operations and hides correctly; status bar remains responsive.
- Build passes and app runs; smoke-test key flows: load inbox, paginate, archive/trash single and bulk, toggle bulk mode, search.

## Test Ideas
- Rapid-fire multiple transients and verify only the latest baseline remains after all timers.
- Trigger errors from network calls and confirm red-styled error; then verify a success transient can replace it.
- Enter/exit bulk mode and confirm baseline context updates without flicker.
- Long-running bulk operation shows Flash until completion; status bar displays a success transient afterward.

## Risks & Mitigations
- Risk: Baseline flicker due to frequent updates (selection change).
  - Mitigation: Debounce selection-driven updates (e.g., 50–100ms) or only update when row index changes.
- Risk: Theme readability.
  - Mitigation: Validate both bundled themes; avoid low-contrast color pairs.
- Risk: Overuse of Flash causing noise.
  - Mitigation: Reserve Flash for long operations and critical notices only.

## References
- `tui/layout.go`: `createStatusBar`, placement of `a.flash.textView`.
- `tui/status.go`: existing helpers to be refactored/extended.
- `tui/keys.go`, `tui/messages.go`: primary call sites to update.
- Architecture rules: Gmail TUI Architecture & Module Boundaries (QueueUpdateDraw, feature placement, import hygiene).

---

When you start implementation, proceed with Phase 1 (infrastructure) and submit a small PR for review before changing all call sites. Keep diffs focused and follow the file responsibility guidelines.
