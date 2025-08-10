# Prompt: Implement an Improved Welcome Screen in Gmail TUI

This prompt is intended to be handed to the assistant to implement an improved, actionable Welcome screen in the `tui` package while respecting the established architecture and module boundaries of the project.

## Context
- Today, the welcome message is written directly into the `text` view from `app.go` during startup.
- Goal: Move welcome rendering into layout code and make the screen more informative, actionable, and consistent with the app’s k9s-like UX.
- Keep the `tui` architecture as documented in `docs` and the project rules: split responsibilities across `layout.go`, `status.go`, `messages.go`, etc.

Relevant snippet (current behavior):

```
// app.go (Run)
// Show welcome message and load messages
if text, ok := a.views["text"].(*tview.TextView); ok {
    text.SetText("👋 Welcome to GizTUI!\n\n" +
        "Your terminal for Gmail\n\n" +
        "Press '?' for help or 'q' to quit")
}
// Load messages in background
go a.reloadMessages()
```

## Objectives
- Informative: show title/logo, short description, account email (if available), and current loading/setup state.
- Actionable: provide quick actions and shortcuts (k9s-style) to discover the app quickly.
- Consistent: follow theme colors and emojis used across the TUI.
- Resilient: distinct states for “loading”, “no credentials / first run”, and “error”.

## Scope
- Replace the existing inlined welcome text with a structured welcome screen component.
- Integrate it into `layout.go` (or a dedicated `welcome.go` inside `tui/` if preferred), keeping `app.go` limited to orchestration.
- Do not change external packages (`internal/gmail`, `internal/render`, `internal/config`) beyond what is required by the TUI wiring.

## Non-Goals
- No changes to Gmail fetching logic beyond invoking existing methods from the welcome flow.
- No new network operations in input handlers.

## Architecture & Module Boundaries (must follow)
- UI updates must happen within `a.QueueUpdateDraw(...)`.
- No network calls in input handlers; perform work in goroutines and apply results via `QueueUpdateDraw`.
- Respect `tui` file boundaries:
  - `layout.go`: creation of views/layout and focus highlighting.
  - `status.go`: status/flash.
  - `messages.go`: message loading/refreshing.
  - `keys.go`: input routing only (no feature logic).
  - `logging.go`: logger.
  - `commands.go`: command bar logic.
- Import only what is needed; no circular dependencies.
- Documentation and code in English.

## Proposed API Additions
- In `layout.go` (or `welcome.go`):
  - `func (a *App) createWelcomeView() tview.Primitive`
  - `func (a *App) showWelcomeScreen(loading bool, accountEmail string)`

Notes:
- `createWelcomeView` builds a composite component (TextView/Flex) displaying the welcome content and quick actions.
- `showWelcomeScreen` sets the welcome content into the `text` container (or returns a dedicated widget added to the main layout) and switches state between loading/setup.

## Behavior & States
1. Client present (normal startup):
   - Show: Title/logo, short description, account email if available, quick actions, and a loading line (e.g., “⏳ Loading inbox…” with a simple spinner or animated dots).
   - Immediately kick off `reloadMessages()` in a goroutine. When the first page arrives, normal list/content flow proceeds (no blocking).

2. Client missing (first run or credentials not found):
   - Show: Title/logo, short description, and a compact setup guide with steps:
     1. Download OAuth credentials.
     2. Place them at `~/.config/gmail-tui/credentials.json` (or provide the path from config if different).
     3. Restart the app.
   - Provide link to `README.md` and show key `?` for Help / `q` for Quit.

3. Error state (optional):
   - If an initialization error is known, show a brief message and suggest retrying.

## Visual & Content Guidelines
- Title: “📨 GizTUI — Your terminal for Gmail” (consistent with theme).
- Subtitle/description line below the title.
- Account line when available (e.g., `Account: user@example.com`).
- Quick actions as text “chips”, e.g.: `[? Help] [s Search] [u Unread] [: Commands]`.
- Loading line: “⏳ Loading inbox…” with a simple progress hint.
- Use colors from the applied theme (`applyTheme()`) for titles and borders.

## Integration Points
- `app.go (Run)`: replace the direct `text.SetText(...)` with:
  - `a.showWelcomeScreen(true, a.Client.ActiveAccountEmail())` when client is present (if no accessor exists, pass empty string or implement a safe getter).
  - `a.showWelcomeScreen(false, "")` (setup state) when client is nil.
- Continue to call `go a.reloadMessages()` on startup when client is present.

## Acceptance Criteria
- The welcome UI is not defined in `app.go`; `app.go` only orchestrates calling the new method(s).
- The screen shows quick actions, status, and uses theme colors and emojis.
- With client present, it shows a loading state and does not block input; list loads in background and the welcome content is replaced by real content once available.
- Without client, it shows a concise setup guide and does not crash.
- All UI updates are done via `QueueUpdateDraw`; no direct goroutine updates on tview components.
- Build (`go build ./...`) succeeds without linter errors in changed files.

## Implementation Notes
- Use `tview.Flex` and `tview.TextView` to compose the welcome block inside the existing `textContainer` area.
- Keep UI mutations in the main thread using `QueueUpdateDraw`.
- Reuse `status.go` for transient messages if needed; do not overuse the content pane for transient status once messages are loaded.
- Keep imports minimal in each file.
- If the function grows large, split into small private helpers in the same file.

## Testing Plan
- Manual flows:
  1. Launch with valid credentials: verify welcome shown with loading line, then messages appear.
  2. Launch without credentials: verify setup instructions are shown; app remains responsive.
  3. Press `?`, `s`, `u`, and `:` at welcome: ensure handlers work and no network call happens inside the handler (network only in the background loaders).
- Run `go build ./...` to confirm compile.

## Risks & Mitigations
- Risk: UI modified from background goroutine. Mitigation: wrap all UI changes in `QueueUpdateDraw`.
- Risk: Import cycles if adding a new file. Mitigation: keep functions within `tui` and only import `tview/tcell` as needed.
- Risk: Over-coupling welcome with messages. Mitigation: keep welcome logic in layout, only call into `messages.go` via existing public methods.

## Rollback Plan
- If issues arise, revert to the previous behavior by restoring the direct welcome `SetText` in `app.go`.

## Deliverables Checklist
- [ ] New functions added: `createWelcomeView`, `showWelcomeScreen` (or equivalent), with tests if applicable.
- [ ] `app.go` updated to use the new welcome functions; no feature logic added to `app.go`.
- [ ] Themed appearance verified (borders/titles/emoji consistent).
- [ ] Successful build and manual interaction test on supported platforms.


