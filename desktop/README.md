# GizTUI Desktop (Wails)

A **visual Gmail client** for GizTUI. It reuses GizTUI's existing service layer
(Gmail, rendering, labels, actions) and presents it through a native desktop
window built with [Wails v2](https://wails.io): a Go backend + a React/TypeScript
frontend, compiled into a single native app (macOS `.app`, plus Windows/Linux).

## How it fits the codebase

The heavy lifting is **not** reimplemented. This app is a thin presentation
layer over the same business logic the TUI uses:

```
internal/services/  ← business logic (unchanged, shared)
        ▲
pkg/desktop/        ← front-end-agnostic API + DTOs (pure Go, unit-tested)
        ▲
desktop/            ← Wails glue (this nested module) + React frontend
```

- **`pkg/desktop`** lives in the main module, so it can import `internal/...`.
  It exposes a small JSON-friendly `API` (`ListInbox`, `Search`, `GetMessage`,
  `Archive`, `Trash`, `MarkRead/Unread`, `ListLabels`) built on the existing
  services. It has no Wails/CGO dependency and is covered by unit tests
  (`go test ./pkg/desktop/`).
- **`desktop/`** is a **separate nested Go module** (`replace` → `../`). Keeping
  it nested means the Wails/CGO/webkit toolchain never touches the main module's
  `go build ./...`, `make build`, or `make test`.

It reuses your existing GizTUI configuration and OAuth token
(`~/.config/giztui/`), so no extra setup is needed if the TUI already works.

## Prerequisites

- Go 1.25+
- Node.js 18+ / npm
- The Wails v2 CLI:
  ```sh
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```
- Platform toolchain (macOS: Xcode command-line tools; Linux: `libwebkit2gtk`,
  `libgtk-3`). Run `wails doctor` to verify your machine.

## Develop

Live-reload (Go + Vite HMR):

```sh
cd desktop
wails dev
```

The frontend also runs standalone in a browser against a built-in **mock
backend** (no Gmail needed) for pure UI work:

```sh
cd desktop/frontend
npm install
npm run dev
```

## Build a native app

```sh
cd desktop
wails build          # → desktop/build/bin/GizTUI Desktop.app (on macOS)
```

## Current scope

- Inbox list with sender, subject, snippet, date, unread state, and label chips
- Reading pane (plain-text body + headers)
- Pagination ("Load more" / `N`)
- Gmail search (full operator syntax: `from:`, `has:attachment`, …)
- Per-message actions: archive, trash, mark read/unread
- **Compose, reply & forward** (threaded reply, cc, forwarded quoting)
- **Labels** — apply/remove via a filterable picker
- **Attachments** — listed in the reader, one-click download to the configured
  download directory
- **Bulk / selection mode** (`v`) — select messages (`Space`, `*`) and
  archive/trash/mark/label them in one go via the service layer's `Bulk*` methods
- **AI summaries with streaming** — one-click summary of the open message,
  powered by your configured LLM through the same `AIService` the TUI uses.
  Tokens stream into the panel live via a Wails runtime event (only shown when
  an LLM provider is configured)
- **AI prompt library** (`p`) — apply any of your saved prompts to the open
  message via the same `PromptService` the TUI uses; the result streams into a
  panel (requires an LLM provider and the local database)
- **Summary caching** — summaries are cached in the per-account local database,
  same as the TUI
- **Keyboard shortcuts** mirroring the TUI defaults (press `?` for the list):
  `j/k` navigate, `Enter` open, `gg`/`G` top/bottom, `a` archive, `d` trash,
  `t` toggle read, `l` labels, `c` compose, `r` reply, `f` forward, `y`
  summarize, `p` prompt, `s`/`/` search, `R` refresh, `N` load more, `v` select
  mode, `Space`/`*` select, `Esc` back.

### Keyboard parity note

GizTUI's TUI is keyboard-first, and the project mandates keyboard↔command
parity. The desktop client keeps that muscle memory: the shortcuts above use the
same default keys as the TUI (`internal/config/config.go` `DefaultKeyBindings`).
The in-app `?` overlay is the discoverable reference.

Not yet ported from the TUI: bulk prompts, Obsidian, Slack, drafts management,
threading view, RSVP, multi-account switching, HTML body rendering. These map
cleanly onto the same service layer and can be added incrementally by extending
`pkg/desktop`.
