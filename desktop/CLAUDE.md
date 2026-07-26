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
- `cd desktop/frontend && npx tsc --noEmit && npm run build`
- `go build ./pkg/desktop/ && (cd desktop && go build ./...) && go test ./pkg/desktop/`
- Drive the change in a browser against the mock (Playwright + the pre-installed
  Chromium) — pickers especially: open via command, arrow-navigate, Enter, Escape.

## 🧱 Ongoing: `App.tsx` decomposition (READ if touching App.tsx)

`App.tsx` is a large god component being broken up incrementally. Pure logic
already lives in small, unit-tested modules — **use/extend these, don't re-inline**:
`format.ts` (formatting/parsing), `compose.ts` (reply/forward builders),
`commands.ts` (`COMMANDS` + palette resolution). Add tests next to new pure logic.

**The plan, progress tracker, and — critically — the coupling landmines that keep
causing bugs (`openIdRef`, `summaryForId`/`promptForId`, `loadMessage` as
`useCallback([])`, mirror refs, first-binding-wins chords, the `anyModal` Escape
order) live in [`docs/DESKTOP_REFACTOR_PLAN.md`](../docs/DESKTOP_REFACTOR_PLAN.md).
Read it before extracting anything stateful.** Next up is F3 (subsystem hooks);
stand up a Playwright integration suite FIRST — it's the only net for the coupled
behavior (AI panels are the riskiest, do them last).
