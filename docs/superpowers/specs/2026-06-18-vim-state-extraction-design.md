# VIM State Extraction — Design (Pilot for App god-object refactor)

**Date:** 2026-06-18
**Status:** Approved (brainstorming) — pending implementation plan
**Branch:** `refactor/vim-state`
**Issue:** #49 (graphify — `App` god object, 158 edges, cohesion 0.03)

## Goal

Extract the VIM key-sequence state out of the `App` god object into a self-contained
`vimState` type, as the **pilot** for the incremental App-decomposition refactor. Improve
maintainability, fix a latent locking inconsistency, and add unit tests where there are none —
**with zero change to user-visible behavior**.

## Why this subsystem (pilot selection)

The App god object has 129 fields and 492 methods across 43 files. A full decomposition does not
fit one spec, so the agreed strategy is **incremental**: extract one cohesive subsystem, validate
the pattern and the ripple, then repeat in separate specs.

VIM navigation was chosen because it is the lowest-risk, highest-validation candidate:

| Candidate | Field accesses | Files touched | Nature |
|-----------|---------------:|--------------:|--------|
| **VIM nav** | 45 | **1** (`keys.go`) | sequence parsing = pure logic |
| Command system | 76 | 4 | history/suggestions |
| Search/Filter | 81 | 4 (incl. app.go) | coupled to ids/pagination |
| Markdown | 0 direct | — | already behind accessors |

VIM's ripple is contained to a single file and its core logic (parse `gg`, `G`, `s5s` → operation +
count) is deterministic, so it validates the target pattern (cohesive state + behavior, extractable
to its own type with unit tests) at minimal risk.

## The 5 fields being extracted (today on `App`)

```go
vimSequence          string    // accumulated key sequence ("g", "s", "s5", ...)
vimTimeout           time.Time // when the current sequence expires
vimOperationCount    int       // numeric count being built ("5" in "s5s")
vimOperationType     string    // pending range-op key ("s", archive key, ...)
vimOriginalMessageID string    // message under cursor when the sequence started
```

These are used only in `internal/tui/keys.go`, across six methods:
`handleVimSequence`, `handleVimNavigation`, `handleVimRangeOperation` (state machine) and
`executeVimRangeOperation`, `executeVimSingleOperation`, `executeVimSingleOperationWithID`
(effects — unchanged by this refactor).

## Latent bug this refactor fixes

The five fields are accessed from **two goroutines**:
1. the tview event-loop (key handlers), and
2. a timeout goroutine (`keys.go:1939`) that sleeps `VimRangeTimeoutMs`, then under `a.mu` checks
   whether the sequence is still pending and fires the single operation.

The locking is **inconsistent**: the timeout goroutine reads/writes the fields under `a.mu`
(lines 1949–1968), but the event-loop writes them **without** `a.mu` (lines 1882, 1917–1919,
2000–2004). That is a latent data race; it "works" today only because the timeout window is wide
(~2s) and the path has zero test coverage. Encapsulating the fields in `vimState` with its **own
mutex**, where every access goes through a synchronized method, removes the inconsistency at the
root.

## Architecture

New file `internal/tui/vim_navigator.go` defining a package-private `vimState`:

```go
type vimState struct {
    mu             sync.Mutex
    sequence       string
    operationType  string
    operationCount int
    timeout        time.Time
    originalMsgID  string
}
```

`App` composes one field `vim vimState` in place of the five flat fields.

**Separation of concerns:** `vimState` owns the **pure, synchronized state machine** and returns
*decisions*. `keys.go` keeps orchestrating the **effects** (`executeVimRangeOperation`,
`enhancedTextView.GotoTop/GotoBottom`, `executeGoToFirst`, `ShowProgress`, `logger`). Effects always
run **outside** the lock (CLAUDE.md rule: never run UI actions under a mutex).

**Time injection:** methods receive `now time.Time` and already-resolved `time.Duration` values
(keys.go reads `a.Keys.VimRangeTimeoutMs` / `VimNavigationTimeoutMs` with their `<=0` fallbacks and
passes them in). This makes the state machine deterministic and unit-testable without real sleeps.

### Method surface (decisions in, effects out)

```go
func (v *vimState) clearIfExpired(now time.Time) bool                          // reset if timeout passed; reset happened?
func (v *vimState) pendingG() bool                                             // is "g" half-typed?
func (v *vimState) startG(now time.Time, d time.Duration)                      // begin "g", set timeout
func (v *vimState) clearSequence()                                             // reset navigation state
func (v *vimState) appendDigit(digit int, now time.Time, d time.Duration) (count int, ok bool)
func (v *vimState) startOperation(key string, now time.Time, d time.Duration, msgID string)
func (v *vimState) completeOperation(key string) (op string, count int, ok bool) // "s5s" → op+count, reset
func (v *vimState) takePendingSingle(key string) (msgID string, ok bool)         // timeout goroutine: atomic check-and-take
```

`takePendingSingle` makes the timeout goroutine's three-step "check pending → capture msgID →
reset" a single atomic, locked operation — closing the race window (compare-and-swap on the state
machine).

## Behavior preservation (non-negotiable)

User-visible VIM behavior must be **identical**: `gg` (top), `G` (bottom), `s5s` (operate on 5),
single-op-after-timeout, focus-aware `gg`/`G` (text vs list), conflict keys (`p`/`o`) still working
normally outside a sequence, configurable timeouts. This is internal reorganization, not a feature.

## Testing

New `internal/tui/vim_navigator_test.go` covering the state machine (today: 0 coverage):
- `appendDigit` accumulates `count*10+digit`; returns ok only when an operation is pending.
- `startOperation` then `completeOperation(sameKey)` returns the op and count (count 0 → effective
  caller default of 1 stays in keys.go), and resets state.
- `clearIfExpired`: with injected `now` before vs after the timeout, resets only when expired.
- `pendingG`/`startG`/`clearSequence` model the `gg` two-step.
- `takePendingSingle`: returns the captured msgID + ok exactly once for a still-pending sequence;
  returns ok=false if the sequence was already completed/cleared (the CAS semantics).

Existing behavior verified by `make test` (TUI suite) + manual VIM smoke test on the user's Mac.

## Follow-up (deferred, not this pilot)

- **VIM expiry is effectively 2× the configured timeout.** The original handler sets
  `timeout = start + d` and then expires when `now.Sub(timeout) > d`, i.e. at `start + 2d`. This
  pilot **preserves** that behavior bit-for-bit (`vimState.clearIfExpired` uses `now.Sub > d`). The
  observable impact is minimal because the main timeout (single-op execution after pressing an op
  key) is driven by the timeout goroutine's `time.Sleep(d)`, not by `clearIfExpired`. Normalizing
  the cleanup window to a true `d` is a separate, explicit behavior change — do it in its own commit
  if desired, not folded into the refactor.

## Out of scope (YAGNI)

- The effect methods (`executeVim*`) — only their state access changes from `a.vimX` to `a.vim.X()`.
- Other App subsystems (command system, search/filter, …) — future pilots, separate specs.
- Any change to VIM keybindings, timeouts defaults, or behavior.

## Definition of Done

- [ ] `vimState` type + methods in `internal/tui/vim_navigator.go`
- [ ] `App` composes `vim vimState`; five flat fields removed
- [ ] `keys.go` rewired to `a.vim.*`; timeout goroutine no longer takes `a.mu` for VIM fields
- [ ] `vim_navigator_test.go` covering the state machine
- [ ] `make pre-commit-check` green; `go test -race ./internal/tui/...` green
- [ ] No user-visible behavior change (manual VIM smoke test)
