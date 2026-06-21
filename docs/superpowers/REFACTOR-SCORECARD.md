# God-Object Refactor — Decision Scorecard (#49)

**Date:** 2026-06-21
**Purpose:** Stop picking the next `App` decomposition slice by gut feel. This is the explicit
decision framework: the drivers, their weights, the measured candidates, and the resulting
priority ranking. Re-measure and re-score before each new slice.

## Context

`App` (internal/tui/app.go) is a god object: started at 129 fields, now **113** after extracting
`vimState` (v1.17.0), `commandState`, `searchState`. Each slice = a self-contained type App composes,
unit-tested, with any latent race fixed. We decide the NEXT slice with the scorecard below.

## Decision drivers (weights)

| Driver | Weight | What it measures | Why it matters |
|--------|-------:|------------------|----------------|
| **Benefit** | 30% | Fields removed + duplication killed | The whole point — shrink the god object. |
| **Safety (low risk)** | 35% | Inverse of (files touched + goroutine accesses) | This TUI has a deadlock/race history. A risky slice that breaks the app costs more than a clean one saves. Highest weight. |
| **Low effort** | 20% | Inverse of total field accesses to rewire | More call sites = more hands-on edits = more chance of a slip. |
| **Value (testability/de-dup)** | 15% | New unit-testable logic or duplication removed | Some slices add real coverage; some just move fields. |

Each candidate is scored 1–5 per driver (5 = best), then weighted.

## Measured candidates (data, not opinion)

Measured 2026-06-21 (`grep` over `internal/tui/*.go`, non-test). "goroutine touches" = field accesses
inside a `go func()` (proxy for threading risk).

| Candidate | Fields | Accesses | Files | Goroutine touches |
|-----------|-------:|---------:|------:|------------------:|
| Overlay backups (help/preload/promptStats) | 9 | 33 | 1 | 0 |
| AI summary pane | 7 | 214 | 12 | 47 |
| Picker state | 6 | 320 | 21 | 116 |
| Caches (message/render/invite/markdown) | 4 | 43 | 5 | 7 |
| UI lifecycle flags | 4 | 16 | 4 | 5 |
| Layout state | 3 | 9 | 3 | 0 |
| Bulk selection | 2 | 273 | 12 | 107 |
| Draft mode | 2 | 3 | 3 | 1 |
| Markdown toggles | 2 | 0 | 0 | 0 (already behind accessors) |

## Scorecard

| Candidate | Benefit (30%) | Safety (35%) | Effort (20%) | Value (15%) | **Weighted** | Rank |
|-----------|:---:|:---:|:---:|:---:|:---:|:---:|
| **Overlay backups** | 5 | 5 | 4 | 5 | **4.80** | 1 |
| **Layout state** | 2 | 5 | 5 | 2 | **3.65** | 2 |
| **UI lifecycle flags** | 3 | 4 | 4 | 2 | **3.40** | 3 |
| **Caches** | 3 | 3 | 3 | 3 | **3.00** | 4= |
| **Draft mode** | 1 | 4 | 5 | 2 | **3.00** | 4= |
| **AI summary pane** | 4 | 2 | 1 | 4 | **2.70** | 6 |
| **Picker state** | 4 | 1 | 1 | 3 | **2.20** | 7 |
| **Bulk selection** | 1 | 1 | 1 | 2 | **1.15** | 8 |
| Markdown toggles | — | — | — | — | done | — |

## Recommended roadmap (in priority order)

1. **Overlay backups** (4.80) — 9 fields → 3 typed, kills a triplicated save/restore pattern, 1 file,
   no threading. Best benefit-to-risk on the board. ← spec already drafted (2026-06-21).
2. **Layout state** (3.65) — tiny, dead-safe (3 files, 0 goroutines). Quick win.
3. **UI lifecycle flags** (3.40) — small, low risk.
4. **Caches** / **Draft mode** (3.00) — medium / trivial; do when convenient.
5. **AI summary pane** (2.70) — high benefit but 12 files + 47 goroutine touches + 214 accesses.
   Only when there's a robust live-test setup (the test-Gmail plan) to catch streaming/focus regressions.
6. **Picker state** (2.20) / **Bulk selection** (1.15) — worst ratio (2–6 fields but 273–320 accesses
   across 12–21 files, 107–116 goroutine touches). Defer; consider service-layer extraction instead of
   a state struct, or leave them.

## How to re-score (each new slice)

1. Re-run the measurement greps (fields/accesses/files/goroutine-touches).
2. Score 1–5 per driver; weight.
3. Pick the top unstarted candidate — unless the user wants to re-weight the drivers (e.g. value
   speed of god-object shrink over safety → raise Benefit, lower Safety).

## Notes

- Markdown toggles already route through accessors (0 direct accesses) — nothing to extract.
- Picker/bulk are so cross-cutting they may be better served by moving logic into the service layer
  than by a state struct; that's a separate design question, not a field-grouping refactor.
- Stop rule: when the remaining candidates are all low-score (big, risky, cross-cutting), the
  god-object refactor has hit diminishing returns — switch to features or service-layer work.
