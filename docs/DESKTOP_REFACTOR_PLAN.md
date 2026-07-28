# 🧭 Desktop `App.tsx` Decomposition & Test-Coverage Plan

> **Why this file exists.** `App.tsx` grew into a 5.6k-line god component during
> the desktop build-out, with **zero** frontend test coverage. Breaking it up is
> risky *because* that's where the subtle, coupled bugs live. This plan is the
> **durable source of truth** for that work — written to the repo (not to any
> assistant's ephemeral context) so a fresh session, or a human, can pick it up
> mid-flight. Update the **Progress tracker** as phases land.

**Status:** F0–F2 DONE & merged (PR #61). Playwright integration net DONE (this batch). Approach approved.
**Owner:** _tbd_ · **Last updated:** 2026-07-26

> **▶ RESUME HERE (next session).** F0–F3 are merged, and **F4.1–F4.5 are done**:
> all self-contained/presentational modals are now their own files
> (`StatsModal`, `ConfigModal`, `PromptPreviewModal`, `SaveQueryModal`,
> `AnalyzerRulesModal`, `AdvancedSearchModal`, `ActionPlanModal`), each with a pure
> logic module where it earned one (`advancedSearch.ts`, `planNodes.ts`) and an e2e
> spec guarding it. **61 unit tests + 33 e2e**, all green; App.tsx **5,747 → 4,794**
> lines. The safety recipe every time: unit-test the pure derivation, e2e-guard the
> flow against CURRENT code first, extract as a behavior-preserving move (state +
> nav stay in App), re-verify `npm test` + `npm run test:e2e` + `tsc` + `build`.
> **Next (and last of F4): the prop-heavy `<MessageList>` / `<Reader>` splits** —
> highest coupling (reader touches the AI panels, links, attachments, RSVP,
> threading, `openIdRef`); do these with dedicated care, re-read §3 (landmines)
> first, and lean on the full e2e net. The branch
> `claude/giztui-visual-client-iadjgl` was reset off `main` after each merge —
> reset it to latest `main` again before starting.

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
- [x] **F3.0** Playwright integration net — `@playwright/test` + `playwright.config.ts` (pre-installed Chromium, vite `webServer` on :5199) + `desktop/frontend/e2e/` (13 specs across inbox/reader, search scope, AI summary+prompt caching/landmines, pickers). `npm run test:e2e`. This is the safety net that guards every F3.x hook extraction ✅.
- [x] **F3.1** `useZoom` (`src/useZoom.ts`) + `useTheme` (`src/useTheme.ts`) — behavior-preserving moves out of App.tsx (−65 net lines). No §3 landmines touched. Guarded by new e2e specs (`e2e/zoom.spec.ts`, `e2e/theme.spec.ts`; suite now 18 specs). Diff was a pure relocation (verified) ✅.
- [x] **F3.2** `useAttachments` + `useRsvp` + `useThreading` (`src/useAttachments.ts`, `src/useRsvp.ts`, `src/useThreading.ts`) — behavior-preserving moves. The per-message fetch/reset lines stay in `loadMessage` and call the hooks' setters/refs under the same names, so the reset ORDER + `openIdRef` gating are byte-identical. **`summarizeThread` deliberately stays in App.tsx** (it writes the AI-summary panel state → belongs to F3.4). Guarded by `e2e/attachments.spec.ts`, `e2e/rsvp.spec.ts`, `e2e/threading.spec.ts`; suite now 25 ✅.
- [x] **F3.3** Message-list logic — scoped safely: the list *state* (`messages`/
  `fullMessagesRef`) is a cross-cutting spine used by ~20 call sites (every
  bulk/doAction/prune), and `loadMessage` reaches into AI restore (F3.4 territory),
  so a full `useMessages` hook would be high-churn prop-threading with little real
  decoupling. Instead extracted the **pure list logic** to `messageList.ts`
  (`freshPrefix` = contiguous-unknown-prefix new-mail detection — the anti-scramble
  logic; `dedupeNew`) + 8 unit tests including the "old message shifted onto page 1
  after a delete is NOT new" case. No React state moved; no §3 landmines touched.
  Guarded by the e2e inbox specs ✅.
- [x] **F3.4** AI panels — scoped to the safe slice, full state-relocation **assessed & declined**.
  A `useAiPanels` state hook is high-risk / low-reward here: `openIdRef` is shared
  (attachments/invite/loader/AI), so it can't be AI-owned; `loadMessage`'s AI-restore
  is interleaved with the reset ORDER (a §3 landmine), so a hook would relocate the
  coupling, not remove it; and the render is already extracted (`AiPanel`). Instead
  extracted the **pure regenerate-decision** (`activeAiPanel` in `src/aiPanels.ts`) —
  the branching that's easy to get subtly wrong — with 5 unit tests. Stateful
  streaming (summarize/runPrompt/touchUp) + refs stay in App **by design**. Guarded
  by the e2e AI specs ✅. _If a full relocation is still wanted, it's a dedicated,
  high-care pass — but the ROI is line-moving, not decoupling._
- [~] **F4** JSX component splits (incremental; each self-contained modal → its own file + e2e)
  - [x] F4.1 `StatsModal` + `ConfigModal` (display modals; −96 lines; +2 e2e specs)
  - [x] F4.2 `PromptPreviewModal` (self-contained; the keyboard-scroll ref+effect moved in; −41 lines; +1 e2e)
  - [x] F4.3 `SaveQueryModal` (presentational pure move) + `AnalyzerRulesModal` (owns its useListNav + delete-key effect); −108 lines; +2 e2e
  - [x] F4.4 `AdvancedSearchModal` + pure `buildAdvancedQuery` in `advancedSearch.ts` (5 unit tests); −114 lines; +1 e2e; unit 56 / e2e 31
  - [x] F4.5 `ActionPlanModal` (presentational, ~23 props; state + planNav stay in App) + pure `buildPlanNodes` in `planNodes.ts` (5 unit tests); −249 lines; +2 e2e (`actionplan.spec.ts`); unit 61 / e2e 33; App.tsx 5,043 → 4,794
  - [x] F4.6 render extraction: `MessageList` (260), `Reader` (248) → `ReaderToolbar` (233) + `ReaderBody` (338), `TopBar` (213). State + landmine refs stay in App, passed as values + plain onX handlers. App.tsx 4,794 → 4,109; every new file < 400; e2e 33 green.
  - [ ] optional: `ModalStack` — low ROI (relocates ~150 props; the file would itself exceed 400 and need re-splitting). Skip unless it buys real clarity.

## 7bis. The hard target: **no file > 400 lines** (owner's goal)

Measured: only **two** files exceed 400 — `App.tsx` (4,109) and `api.ts` (1,150).

**`App.tsx` breakdown (why render extraction alone won't get there):** render is now
~653 lines (of which the modal stack is ~330); the other **~3,456 lines are LOGIC** —
84 `useState`/`useRef` decls, **74 `useCallback` handlers**, 10 effects, and the
keymap/command block. The render was never the reason App is huge. To reach < 400,
the state+handlers must move into **subsystem hooks** (the risky decoupling — re-read
§3 landmines first: `openIdRef`, `aiCache`, mirror refs, `loadMessage` reset order).

Proposed hook sequence (each its own PR, e2e-guarded, safest → riskiest):
- `useUndo`, `useAutoRefresh`, `useIntegrations` (obsidian/slack) — small, isolated
- `useDrafts`, `useCompose` — self-contained
- `useAccounts` (switch/list) — isolated
- `useBulk` (bulkAction/toggleSelect/exitBulk/select-all)
- `useSearch` (query/localFilter/load/loadMore/pending/prune) — touches `fullMessagesRef`
- `useCommands` + `useKeymap` (the ~600-line palette/keymap block → mostly data + `commands.ts`)
- `useReaderActions` (open/doAction/quickSearch/save/links) — touches `openIdRef`
- `useAiActions` (summarize/runPrompt/touchUp/generateReply/suggest + dismiss) — **LAST, riskiest**: `aiCache`, `summaryForId`/`promptForId`, mirror refs, streaming callbacks
- `api.ts` split: types → `apiTypes.ts`, mock → `apiMock.ts`, keep the thin real-binding wrapper

## 8. Safety rules (non-negotiable)

1. One cohesive change per PR; small diffs.
2. The full gate (§6) green before merge. No exceptions for "it's obviously fine."
3. **Zero behavior change** in a refactor PR. Behavior changes are separate PRs.
4. Stop before F3 unless F0 is done and Playwright is protecting the flow.
5. If a step can't preserve a §3 landmine contract, abandon that step and rethink.

## 9. Open decisions

- [x] vitest + jsdom approved as the unit harness.
- [x] Order approved: F0 → F1 → F2 this batch; F3 (hooks) in dedicated sessions.
- [x] Playwright specs live in **`desktop/frontend/e2e/`**, run via **`npm run test:e2e`**
  (config `desktop/frontend/playwright.config.ts`; a vite `webServer` boots the app
  against the api.ts mock; Chromium is the pre-installed browser, never downloaded).
  Kept **local-run for now** (not wired into CI) — revisit wiring it into CI once the
  F3 hook extractions have proven the suite stays stable. Artifacts
  (`test-results/`, `playwright-report/`) are git-ignored.

## 10. Related outstanding work (NOT this plan — separate backlog)

- [ ] Add repo secret **`HOMEBREW_TAP_TOKEN`** (contents:write on
  `ajramos/homebrew-giztui`) so `release-desktop.yml`'s cask auto-bump runs.
- [ ] macOS **notarization** + Windows signing (phase 2 in `DESKTOP_DISTRIBUTION.md`)
  — the biggest user-facing friction (Gatekeeper/SmartScreen on unsigned builds).
- [ ] Product decision: keep or remove **`:touch-up`** (user finds it unclear).
- [ ] Minor TUI↔desktop command-parity gaps (`:numbers`, `:preload`, bookmarks).
- [ ] Coverage baseline (2026-07-25): Go `internal/services` 32%, `pkg/desktop` 24%,
  `internal/tui` 9.7%; **frontend bootstrapped 0 → 38 unit tests**.
