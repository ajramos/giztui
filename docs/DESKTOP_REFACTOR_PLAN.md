# 🧭 Desktop `App.tsx` Decomposition & Test-Coverage Plan

> **Why this file exists.** `App.tsx` grew into a 5.6k-line god component during
> the desktop build-out, with **zero** frontend test coverage. Breaking it up is
> risky *because* that's where the subtle, coupled bugs live. This plan is the
> **durable source of truth** for that work — written to the repo (not to any
> assistant's ephemeral context) so a fresh session, or a human, can pick it up
> mid-flight. Update the **Progress tracker** as phases land.

**Status:** APPROVED — executing F0 → F1 → F2 this batch (F3 hooks deferred).
**Owner:** _tbd_ · **Last updated:** 2026-07-25

> **Sequencing note (refinement, approved):** Playwright integration tests protect
> the *coupled* refactors (F3). F1/F2 are pure/near-pure and are protected by unit
> tests, so this batch only stands up the **vitest** harness now; the Playwright
> suite is built at the **start of F3** (when it actually guards a coupled change).

---

## 1. Current state (measured, not guessed)

`desktop/frontend/src/App.tsx`:

| Metric | Count |
| --- | --- |
| Lines | ~5,618 |
| `useState` | 121 |
| `useRef` | 28 |
| `useCallback` | 81 |
| `useEffect` | 14 |
| Inline `backend.*` API calls | 88 |
| Modals/pickers rendered in one JSX tree | ~21 |
| Setters touching the Prompt/AI subsystem | **68** (the biggest single tangle) |

Frontend test coverage: **0%** — no vitest/jest, no `test` script. Only manual +
ad-hoc Playwright-against-the-mock has ever exercised it.

Go coverage for reference: `internal/services` 32%, `pkg/desktop` 24%,
`internal/tui` 9.7% (measured 2026-07-25).

## 2. Goals & non-goals

**Goals, in priority order:**
1. **Regression safety.** This app keeps surfacing coupling bugs; the #1 job is a
   net that catches them, then structure that makes them less likely.
2. **Testability.** Pull logic out of the render component so it can be unit-tested.
3. **Maintainability.** Smaller, single-responsibility modules.

**Non-goals (explicitly out of scope for now):**
- Rewriting behavior or "improving" UX while refactoring. **Zero behavior change.**
- A state-management library (Redux/Zustand/etc.). Not yet — decide later.
- Touching the TUI Go monoliths (`messages.go`, `app.go`). Separate effort.
- Big-bang. Never.

## 3. Landmines — the coupling that caused our bugs (READ THIS FIRST)

Do **not** extract anything below without preserving these exact contracts:

- **`openIdRef`** is read inside streaming callbacks so a run started on message A
  never paints into the panel after you've navigated to B. Any hook that streams
  must keep reading the *live* ref, not a captured value.
- **`summaryForId` / `promptForId` / `runningLabelRef`** gate which message a run
  belongs to (title/body must not bleed across messages). 
- **`loadMessage` is `useCallback([])` on purpose** for identity stability; it is
  called from many places and effects. Changing its deps has ripple effects.
- **Mirror refs** (`summaryRef`, `promptResultRef`, `promptLabelRef`, …) exist to
  dodge stale closures in window-level handlers. Extractions must keep the
  ref-freshness pattern, not "clean it up" into plain deps.
- **`chordAction` is first-binding-wins** (mirrors the TUI's first-match), e.g.
  a remapped `Y` resolves to load-more, shadowing regenerate. Order matters.
- **`anyModal` Escape chain** closes the *topmost* modal in a specific order;
  WKWebView won't focus bare divs, so keys come from **window listeners**.
- **The order of state resets in `loadMessage`** is load-bearing (we fixed the
  "blank prompt title on return" bug by *not* clobbering live streaming state).

If a change can't preserve these, it doesn't ship.

## 4. Strategy

**Options considered:**
- **A. Just add tests, no structural change.** Lowest risk, but leaves the
  god-file; coverage of a 5.6k component is shallow.
- **B. Characterization tests first, then refactor under their protection.** ✅
  Correct for a risky, coupled file with no net.
- **C. Big-bang split into hooks/components.** Fast on paper, high blast radius,
  exactly what caused the bugs. Rejected.
- **D. Strangler / incremental extraction by risk**, each step shippable. ✅

**Chosen: B + D.** Build the net first, then peel off cohesive pieces
lowest-risk → highest-risk, each an independent PR that must stay behavior-identical.

## 5. Testing strategy (two layers — be honest about what each covers)

| Layer | Tooling | Covers | Does NOT cover |
| --- | --- | --- | --- |
| **Unit** | vitest + jsdom | Pure & near-pure logic: `format.ts`, command parse/match, compose builders, any cache logic extracted as pure functions | Coupled runtime behavior (WKWebView, window listeners, focus, streaming order) — jsdom can't reproduce it |
| **Integration** | Playwright + the api.ts **mock** | The coupled flows where bugs live: open→summarize→apply prompt→switch→return→dismiss; search scoping; delete/prune; bulk; picker keyboard nav | Fine-grained internal logic (do that in unit) |

**Key insight:** the integration layer is the real protection for this refactor —
it's where every bug this session actually lived. We've been running these flows
by hand; **formalizing them into a committed, repeatable suite is the highest-value
first step.** Do not over-invest in jsdom component tests of `App` itself.

## 6. Phased plan

Each phase = one small PR. **Gate for every PR:** `tsc --noEmit` clean, `npm run
build` clean, `go build ./pkg/desktop/ && (cd desktop && go build ./...)` clean,
unit tests green, **Playwright smoke green**, and a manual read of the diff
confirming zero behavior change.

### F0 — Harness (do this first; it makes everything else safe)
- Add **vitest + jsdom** (devDeps), a `test` script, minimal config.
- Formalize 3–5 **Playwright** smokes we already run by hand into a committed
  suite (open message, summarize, apply-prompt-then-switch-then-return, search
  scope, delete). A tiny runner + the pre-installed Chromium.
- **DoD:** `npm test` runs unit; a documented command runs the Playwright suite;
  both green in CI-equivalent local run.

### F1 — Pure helpers (near-zero risk)
- `format.ts`: `displayName`, `emailAddr`, `cleanSubject`, `countMatches`,
  `formatDate`, `formatFull`, `formatSize`, `matchesCombo`, `formatICSDate`,
  `mixHex`, `labelForAction` (already scoped once; ~11 fns).
- Compose builders (`replyInit`, `replyAllInit`, `forwardInit`) + quick-search
  query building, if verified pure.
- **Unit tests for each.** −~200–250 lines from `App.tsx`.

### F2 — Command layer (medium risk, high coverage payoff)
- `commands.ts`: the `COMMANDS` table + a dispatcher parameterized by an
  **actions object** (so it's testable without React). `executeCommand` becomes
  a thin adapter that passes real handlers in.
- **Unit-test** command resolution / aliasing / arg parsing.

### F3 — Subsystem hooks (highest risk — separate, careful sessions)
Extract **as behavior-preserving moves**, safest → riskiest, Playwright in front:
1. `useZoom`, `useTheme` (small, isolated).
2. `useThreading`, `useAttachments`, `useRsvp`.
3. `useMessages` (list/load/pagination/pendingNew/prune) — touches `openIdRef`.
4. **`useAiPanels`** LAST (summarize/prompt/touchUp/cache/regenerate/dismiss +
   the ForId/running/mirror refs) — the 68-setter tangle. This is where the bugs
   were; treat with maximum care and full Playwright coverage.

### F4 — JSX component splits (mechanical, verbose)
- `<MessageList>`, `<Reader>` (+ toolbar), and lift the ~21 modals into their own
  files where they aren't already. Prop-drilling or a small context; decide then.

## 7. Progress tracker

- [x] **F0** Harness — vitest + jsdom, `test` script (Playwright suite deferred to F3 start)
- [x] **F1** Pure helpers + unit tests — `format.ts` (11 fns, 23 tests) + `compose.ts` (reply/replyAll/forward, 6 tests) ✅
- [x] **F2** Command layer — `commands.ts` (COMMANDS + pure `parseCommand`/`filterCommands`/`resolveEnter`), CommandBar/App rewired, 9 unit tests + integrity check ✅. Scope kept safe: the big `executeCommand` switch stays as the handler adapter; **data-driven dispatch (actions object) is deferred** — low marginal value, higher risk.
- [ ] **F3.1** useZoom / useTheme
- [ ] **F3.2** useThreading / useAttachments / useRsvp
- [ ] **F3.3** useMessages
- [ ] **F3.4** useAiPanels  ⚠️ highest risk
- [ ] **F4** JSX component splits

## 8. Safety rules (non-negotiable)

1. One cohesive change per PR; small diffs.
2. The full gate (§6) green before merge. No exceptions for "it's obviously fine."
3. **Zero behavior change** in a refactor PR. Behavior changes are separate PRs.
4. Stop before F3 unless F0 is done and Playwright is protecting the flow.
5. If a step can't preserve a §3 landmine contract, abandon that step and rethink.

## 9. Open decisions (needed before executing)

- [ ] Approve **vitest + jsdom** as the unit harness.
- [ ] Confirm order: **F0 → F1 → F2 this batch; F3 (hooks) in dedicated sessions.**
- [ ] Where do Playwright specs live (`desktop/frontend/e2e/`?) and do we wire them
  into CI or keep them local-run for now?
