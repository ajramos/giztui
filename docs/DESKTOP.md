# 🖥️ GizTUI Desktop — Architecture Guide

GizTUI Desktop is a **visual Gmail client** that reuses GizTUI's entire service
layer and presents it through a native window built with
[Wails v2](https://wails.io): a Go backend + a React/TypeScript frontend
compiled into a single native app (macOS `.app`, Windows, Linux).

It is **not** a rewrite. Every piece of business logic — Gmail access, LLM
calls, labels, drafts, threading, the inbox analyzer, theming — is the same code
the TUI runs. The desktop layer only adapts that logic to a JSON-friendly API
and renders a GUI on top of it.

---

## 🗂️ Layering

```
internal/services/   ← business logic (unchanged, shared with the TUI)
        ▲
pkg/desktop/         ← front-end-agnostic API + DTOs (pure Go, unit-tested)
        ▲
desktop/             ← Wails glue (nested Go module) + React/TypeScript frontend
```

### `pkg/desktop` — the adapter (main module)

Lives in the **main module**, so it can import `internal/...`. It exposes a
small, JSON-serializable `API` with plain methods and DTOs — no Wails, no CGO,
no webkit. That means it builds and unit-tests on any platform:

```sh
go test ./pkg/desktop/
```

Key files:

| File | Responsibility |
|------|----------------|
| `api.go` | `API` struct, `Deps` wiring, list hydration (`ListInbox`, `Search`, `GetMessage`, `Archive`, `Trash`, `MarkRead/Unread`) |
| `session.go` | `NewSession` builds the full service stack from config + OAuth token; `buildAPI` is reused for account switching; builds the Calendar client best-effort |
| `keymap.go` | Resolves the user's keybindings from `config.Keys` (TUI defaults as fallback) into a JSON `KeyMap` |
| `dto.go` | All JSON DTOs (`MessageSummary`, `MessageDetail`, `Label`, `Prompt`, `Invite`, `ConfigInfo`, …) |
| `ai.go`, `prompts.go`, `labels.go`, `actionplan.go`, `thread.go`, `saved_queries.go`, `links.go`, `save.go`, `obsidian.go`, `slack.go`, `compose.go`, `drafts.go`, `attachments.go`, `web.go`, `theme.go`, `invite.go`, `undo.go`, `stats.go` | One file per feature area; thin methods over the matching service |

Every `API` method takes a `context.Context` and returns DTOs + a plain error.
Optional dependencies (AI, prompts, Obsidian, Slack, threading, saved queries,
analyzer, rules, calendar) are `nil` when unconfigured, and each feature has an
`*Enabled()` probe so the UI can hide what isn't available.

### `desktop/` — the Wails app (nested module)

A **separate nested Go module** with `replace github.com/ajramos/giztui => ../`.
Keeping it nested means the Wails/CGO/webkit toolchain never touches the main
module's `go build ./...`, `make build`, or `make test`. It is excluded from CI's
`./...` and carries its own `go.mod`/`go.sum`.

| File | Responsibility |
|------|----------------|
| `main.go` | Wails app bootstrap (window, assets, lifecycle) |
| `app.go` | The bound struct — **thin wrappers** that forward to `session.API` and translate streaming into Wails runtime events |
| `frontend/` | React + TypeScript (strict) + Vite single-page app |

> ⚠️ `desktop/go.mod` and `desktop/go.sum` are committed and must build from a
> clean checkout. When `wails build` regenerates them locally, do **not** commit
> the churn — `git checkout -- desktop/go.mod && rm -f desktop/go.sum` if needed.
> `frontend/wailsjs/` and `frontend/package.json.md5` are generated and ignored.

---

## ⚛️ Frontend

A single-page React app. The important pieces:

| File | Responsibility |
|------|----------------|
| `src/App.tsx` | The whole application shell: state, keyboard handling, command palette, all panels and modals |
| `src/api.ts` | Typed wrapper over the Wails-bound backend **plus a full mock backend** so the UI runs in a plain browser |
| `src/Icons.tsx` | Stroke SVG icon set + the shared `IconBtn` (one button language everywhere) |
| `src/HtmlBody.tsx` | Sandboxed iframe renderer for HTML emails |
| `src/Compose.tsx`, `LabelsPicker.tsx`, `PromptsPicker.tsx`, `PromptManager.tsx`, `LinksPicker.tsx`, `CommandBar.tsx`, `AccountSwitcher.tsx`, `Help.tsx`, `MoreMenu.tsx`, `HighlightedText.tsx` | Focused components/modals |
| `src/styles.css` | CSS custom properties (theme variables) + all component styles |

### The backend proxy + mock

`api.ts` exports a `backend` proxy. Inside the packaged app it forwards to
`window.go.main.App` (the Wails bindings). In a plain browser it forwards to a
**mock backend** with fake data, so the entire UI is explorable and testable
without Gmail:

```sh
cd desktop/frontend && npm run dev   # browser, mock backend
```

This is also how the Playwright screenshots and smoke tests run.

### Streaming (AI)

AI summaries and prompt results stream token-by-token. The backend emits Wails
runtime events (`summary:token`, `prompt:token`); `streamViaEvent` in `api.ts`
subscribes, forwards each token to the panel, and returns the final text. Against
the mock backend it chunks the resolved string so streaming looks identical.

> Streaming callbacks never use `QueueUpdateDraw` — they update the panel
> directly, matching the TUI's ESC-deadlock rules.

### Config-driven keybindings

`backend.KeyMap()` returns the user's resolved shortcuts (their `config.json`
`keys` block, TUI defaults as fallback). The frontend inverts this into a
`chord → action` map (first-binding-wins for shared keys) and drives all global
shortcuts from it. List navigation (`j/k`, arrows, `Enter`, `Esc`, `*`, `gg`,
`G`, `space`) is handled directly, like the TUI's native table.

Two deliberate deviations from raw key defaults (documented in `keymap.go`):

- **`T` = search-to**, matching the TUI (where `toggle_threading` is unbound by
  default). The conversation toggle lives on its toolbar button and `:threads`.
- **`g` = draft-reply is menu/command-only** — the `gg` goto-top sequence
  intercepts `g` first, so draft-reply is reached via the `⋯` menu or `:draft`.

### Command parity

Every action also has a `:` command (`CommandBar`), mirroring the TUI's
keyboard↔command parity mandate. The `?` overlay (`Help.tsx`) is the discoverable
reference for both.

### Theming

`backend.GetThemeColors(name)` flattens a YAML theme (the same `ThemeService` the
TUI uses) into a palette that the frontend maps onto CSS custom properties on
startup. The `H` theme picker switches it live. Every component reads theme
variables — no hardcoded colors.

### HTML email safety

HTML bodies render in a sandboxed iframe (`HtmlBody.tsx`):

- **No scripts** — the sandbox omits `allow-scripts`, so email HTML can't run JS.
- **Remote content blocked** by a strict CSP until the user clicks "Load images"
  (no tracking pixels by default).
- `allow-same-origin` (still no scripts) lets the component intercept link clicks
  → open them in the **system browser** via `OpenURL`, and **forward keystrokes**
  back to the app so shortcuts keep working while an email has focus.

---

## 🔁 Runtime flow

1. `NewSession` loads config + OAuth token from `~/.config/giztui/` (the same
   files the TUI uses — no extra setup), builds the Gmail/LLM/DB/Calendar stack,
   and returns an `API`.
2. `app.go` binds the `API`; Wails injects it as `window.go.main.App`.
3. The frontend probes feature flags, loads the keymap + theme, and renders.
4. User input → a `backend` call → an `API` method → the shared service → Gmail /
   LLM / local DB. Results come back as DTOs and render.

Account switching rebuilds the whole stack for the selected account via the
existing `AccountService` (`session.SwitchAccount` → `buildAPI`).

---

## 🛠️ Build & run

Prerequisites: Go 1.25+, Node 18+/npm, the Wails v2 CLI
(`go install github.com/wailsapp/wails/v2/cmd/wails@latest`), and the platform
toolchain (macOS: Xcode CLT; Linux: `libwebkit2gtk`, `libgtk-3`). `wails doctor`
verifies your machine.

```sh
# Live-reload dev (Go + Vite HMR)
cd desktop && wails dev

# UI-only, mock backend, in a browser
cd desktop/frontend && npm run dev

# Native build → desktop/build/bin/GizTUI Desktop.app (macOS)
cd desktop && wails build && open "build/bin/GizTUI Desktop.app"
```

Verification without a GUI (CI-friendly):

```sh
go test ./pkg/desktop/         # adapter unit tests
(cd desktop && go build ./...) # bindings compile
(cd desktop/frontend && npx tsc --noEmit && npm run build)  # frontend
```

---

## ➕ Adding a feature

1. Add a method (+ DTO if needed) to the matching `pkg/desktop/*.go` file,
   delegating to the service. Add an `*Enabled()` probe if it's optional.
2. Wire the service into `session.go` `buildAPI` if it's new.
3. Add a thin wrapper in `desktop/app.go` (translate streaming to a runtime event
   if it streams).
4. Frontend: add the method + a mock to `api.ts`, wire UI in `App.tsx`, add a
   keybinding (via `KeyMap`) **and** a `:` command, and document it in `Help.tsx`.
5. Verify with the commands above; screenshot against the mock if it's visual.

See [DESKTOP_TEST_PLAN.md](DESKTOP_TEST_PLAN.md) for the full feature checklist.
