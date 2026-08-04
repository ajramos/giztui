# 🖥️ GizTUI Desktop — Conventions for Claude Code

Guidance for the Wails desktop client (`pkg/desktop/` = reusable API, `desktop/`
= Wails glue + React frontend in `desktop/frontend/`). The goal is **feature
parity with the TUI** and a **keyboard-first** experience. Read this before
touching desktop UI, especially pickers/modals.

## 🧭 Pickers & modals — MANDATORY premises

Every picker/modal (labels, links, themes, saved-searches, move, RSVP, rules,
action-plan, …) MUST follow these so they behave consistently:

1. **Keyboard-first, focus-independent.** macOS WKWebView will **not reliably
   focus a bare `<div>`**, so never rely on a focused modal div to receive keys.
   - Pickers with **no text input**: drive keys from a **window-level** listener.
     Use `useListNav(items, { onEscape, windowKeys: true })` (see
     `useListNav.ts`), or a single `window` `keydown` listener with refs to fresh
     state (see `LinksPicker.tsx` / `RSVPPicker.tsx` for the 1-9 pattern).
   - Pickers with a **filter input**: `autoFocus` the input (inputs *do* focus
     reliably) and still use `windowKeys: true` for arrows/Enter/Escape — do NOT
     also put `onKeyDown={nav.onKeyDown}` on the input (double-fires).
2. **Highlight** the active row with the `nav-active` class; `onMouseEnter` sets
   the active index so mouse and keyboard agree — but use `nav.setActiveHover(i)`,
   NOT `nav.setActive(i)`. Keyboard nav disarms hover until the pointer genuinely
   moves; otherwise scrolling a long list slides a row under the stationary cursor,
   fires `mouseenter`, and snaps the selection back (the cursor gets "trapped").
3. **Escape closes**, from anywhere. The App's global handler closes the topmost
   modal at the window level (see the `anyModal` block in `App.tsx`). A picker
   opened via the command bar must not be instantly actioned by the Enter that
   opened it — the `CommandBar` calls `e.stopPropagation()` for that reason.
4. **Footers show keyboard hints**, not `Done`/`Cancel` buttons. Use
   `<span className="foot-hint">↑↓ move · Enter … · Esc close</span>`. Keep only
   genuinely additive footer buttons (e.g. "Save current search").
5. **No emojis.** Use the shared **SVG icons** from `Icons.tsx` (`Icon.archive`,
   `Icon.label`, `Icon.cloud`, …). Add a new Feather-style icon there rather than
   reaching for an emoji. The one allowed glyph is `✕` for the close button
   (consistent across all modals). This matches the rest of the interface, which
   is SVG-icon based.
6. **`1-9` quick-select** where the TUI has it (links, RSVP options).
7. **Reflect mutations in place** (labels added/removed, rows moved) instead of
   refetching, so the list updates instantly and nothing gets marked read.
8. When you add a picker, wire it into the App's **`anyModal` guard** and the
   **Escape chain**, and register any command in `COMMANDS` + `executeCommand`.
9. **Editable entities → `usePickerCrud`.** If a picker's rows are entities the
   user can edit or delete (saved searches, prompts, rules, jobs, …), drive
   edit/delete through the shared **`usePickerCrud(items, active, {onEdit, onDelete})`**
   hook — never hand-roll a `keydown` listener for it. It gives one convention
   everywhere: `e`/`d` (and `Delete`/`Backspace`) when the list holds focus, and
   `Shift+E`/`Shift+Delete`/`Shift+Backspace` when a filter or edit form is focused.
   Keep a ✎/🗑 button per row for the mouse, and open edits in a small modal that
   stacks over the picker (`stopPropagation` on its Escape so only the modal
   closes — see `EditQueryModal`/`PromptEditModal`). Pure action/selection pickers
   (move, links, RSVP, theme, attachments, suggest, Slack, labels-on-a-message)
   have no entity to edit — they don't use the hook.

## 🍏 WKWebView gotchas (hard-won)

- **HTML email is rendered in a Shadow DOM** (`HtmlBody.tsx`), NOT an iframe:
  WKWebView never delivers click/keydown events from inside a sandboxed iframe to
  our listeners, which killed links and shortcuts. Shadow DOM keeps events
  working; the HTML is sanitized with **DOMPurify** first (no scripts/handlers/
  `javascript:`), and remote images are **proxied through the Go backend**
  (`FetchImage` → data URI) because WKWebView won't load external subresources
  from the app's custom-scheme origin.
- **UI zoom** is CSS `zoom` on the document root (Cmd/Ctrl +/-/0); the app shell
  uses `height: 100%` (not `100vh`) so zoom doesn't overflow.
- **Account-scoped services** (query, analyzer rules, deterministic rules) need
  `SetAccountEmail` or they error with "account email not set".

## 🔌 Adding a backend-backed feature

1. Reuse the TUI service in `pkg/desktop/` (main module — can import `internal/`).
   Add DTOs + methods on `*API`; scope searches like the TUI (`scopeSearch`).
2. Bind a thin wrapper on `*App` in `desktop/app.go` (uses `a.api()` / `a.enabled`).
3. Add the method + a mock to `desktop/frontend/src/api.ts` (mock keeps browser
   dev working).
4. Build the UI following the picker premises above.

## ✅ Verify before claiming done

- `cd desktop/frontend && npm test` (vitest unit tests — the frontend now has them)
- `cd desktop/frontend && npm run test:e2e` (Playwright integration suite in `e2e/`
  — drives the real app against the api.ts mock; the regression net for the App.tsx
  decomposition. Extend it when you add/refactor a coupled flow.)
- `cd desktop/frontend && npx tsc --noEmit && npm run build`
- `go build ./pkg/desktop/ && (cd desktop && go build ./...) && go test ./pkg/desktop/`
- Drive the change in a browser against the mock (Playwright + the pre-installed
  Chromium) — pickers especially: open via command, arrow-navigate, Enter, Escape.

## 🧱 `App.tsx` decomposition — DONE (READ if touching App.tsx)

The decomposition is **complete**: every code file in the desktop module (Go,
TS/TSX, CSS) is now **under 500 lines** (`App.tsx` is ~490). Keep it that way —
**use/extend the existing modules, don't re-inline logic into `App.tsx`.**

`App.tsx` is now a thin **orchestrator**: it owns cross-subsystem state and wires
the hooks + two presentational surfaces. Where things live:
- **Render** — `AppInbox.tsx` (top bar + banners + list/reader) and `AppModals.tsx`
  (the modal/picker stack, a `ComponentProps` forwarder over `ModalsPrimary` +
  `ModalsSecondary`). App builds one dense props object per surface.
- **Global input** — `useAppWiring.ts` owns the single window `keydown` listener
  and the command runner; it takes the merged `KeydownCtx & CommandCtx` through a
  ref (registered once, always fresh) and returns a stable `executeCommand`.
- **Subsystem hooks own their state** — `useAiActions` (AI panels + the
  `openIdRef`/`aiCache`/mirror-ref landmines), `useActionPlan` (plan/analyze/rules),
  `useMessages`, `useReader`, `useMailActions`, `useKeymap`, `useMiscActions`,
  `useDrafts`, `useAttachments`, `useRsvp`, `useThreading`, `useBootstrap`,
  `useTheme`, `useZoom`, `useAutoRefresh`, `useIntegrations`, `useUndo`.
- **Pure logic modules (unit-tested)** — `format.ts`, `composeBuilders.ts`
  (reply/forward builders), `commands.ts` (`COMMANDS` + palette), `advancedSearch.ts`,
  `planNodes.ts`, `messageListModel.ts`, `aiPanels.ts`. Add tests next to new pure logic.
- **Self-contained modals** — `StatsModal`, `ConfigModal`, `PromptPreviewModal`,
  `SaveQueryModal`, `AnalyzerRulesModal`, `AdvancedSearchModal`, `ActionPlanModal`,
  `StartupScreens`. Presentational render lives in `TopBar`/`MessageList`/`Reader`.

> **Filenames:** the pure-logic siblings are named to avoid case-only collisions
> with their components (`composeBuilders.ts` vs `Compose.tsx`, `messageListModel.ts`
> vs `MessageList.tsx`) — macOS is case-insensitive and `tsc` picks `.ts` over
> `.tsx`, so never reintroduce a `compose.ts`/`messageList.ts` pair.

**Before touching anything stateful, re-read the coupling landmines** (`openIdRef`,
`summaryForId`/`promptForId`, `loadMessage` reset order, mirror refs,
first-binding-wins chords, the `anyModal` Escape order) documented in
[`docs/DESKTOP_REFACTOR_PLAN.md`](../docs/DESKTOP_REFACTOR_PLAN.md) §3. The
Playwright suite in `e2e/` (39 specs) + 62 vitest unit tests are the regression
net — extend them when you refactor a coupled flow (AI panels are the riskiest).
