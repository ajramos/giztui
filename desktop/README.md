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

## MVP scope (current)

Reading-focused first cut:

- Inbox list with sender, subject, snippet, date, unread state, and label chips
- Reading pane (plain-text body + headers)
- Gmail search (full operator syntax: `from:`, `has:attachment`, …)
- Per-message actions: archive, trash, mark read/unread

Not yet ported from the TUI: compose/reply, AI summaries & prompts, Obsidian,
Slack, attachments, drafts, threading, RSVP, multi-account switching. These map
cleanly onto the same service layer and can be added incrementally by extending
`pkg/desktop`.
