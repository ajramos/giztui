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
(`~/.config/giztui/`), so no extra setup is needed if the TUI already works. On a
fresh machine (no token yet), first launch opens Google sign-in in your **system
browser** and shows a sign-in screen with an "Open sign-in in browser" button;
once you approve, the app continues automatically.

## Prerequisites

- Go 1.25.13+
- Node.js 20.19+ or 22.12+ / npm
- The pinned Wails v2 CLI:
  ```sh
  make -C desktop deps
  ```
- Platform toolchain (macOS: Xcode command-line tools; Linux:
  `build-essential`, `libwebkit2gtk-4.0-dev`, and `libgtk-3-dev`). Run
  `wails doctor` to verify your machine.

## Develop

Live-reload (Go + Vite HMR):

```sh
make -C desktop dev
```

The frontend also runs standalone in a browser against a built-in **mock
backend** (no Gmail needed) for pure UI work:

```sh
cd desktop/frontend
npm ci
npm run dev
```

## Build a native app

```sh
make -C desktop build
open "desktop/build/bin/GizTUI Desktop.app" # macOS
```

> **Architecture:** see [docs/DESKTOP.md](../docs/DESKTOP.md).
> **Test plan:** see [docs/DESKTOP_TEST_PLAN.md](../docs/DESKTOP_TEST_PLAN.md).

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
- **Multi-account** — switch between configured accounts from the header; the
  whole service stack (Gmail client, database, AI, prompts) rebuilds for the
  selected account via the existing `AccountService`
- **HTML email rendering** — rich HTML bodies render in an isolated Shadow DOM,
  sanitized with DOMPurify (no scripts/handlers). Remote images/trackers are
  blocked until you load them (`:images`, fetched via the backend); inline
  `cid:` images load automatically. Toggle HTML/plain-text with the `M` key
- **Drafts** (`D`) — list, open/edit, save, update, and delete drafts; sending a
  draft removes it. Compose can save any message as a draft
- **Open in Gmail web** (`O`) — open the current message in Gmail in the system
  browser
- **Bulk prompts** — apply a prompt across a bulk selection (streamed result)
- **Obsidian** & **Slack** — send the open message to your vault / a channel
- **Threading** — full conversation view with per-message and all expand/collapse
  and a streamed thread summary
- **Saved searches** (`Q`) and **save current search** (`Z`)
- **Inbox action plan** (`P`) — AI triage (with a deterministic-rules first pass)
  into categories you can expand, apply per-bucket or all at once, and
  **recategorize** (`m` moves an email or a whole bucket to another category);
  plus a rules manager and an effective-prompt viewer
- **Move to folder** (`m`), **quick searches** (`F`/`T`/`S`), **content search**
  (`/`), **toggle headers** (`h`)
- **Draft reply (AI)**, **touch-up (AI)**, **regenerate summary**, **prompt
  manager** (create/edit/refine/delete)
- **Advanced search** builder and a **local filter** toggle
- **Undo** (`U`) for archive/trash/read/unread (single & bulk)
- **Auto-refresh** (background inbox polling), **save raw `.eml`**
- **Calendar RSVP** — Accept / Tentative / Decline on invites via a keyboard
  picker (`V`); needs the `calendar.events` scope, requested at sign-in (older
  tokens must re-authorize once)
- **Themes** — live theme switching (`H`) mapping your YAML themes to the UI
- **Stats / config / cache** — AI usage panel, a read-only config view, and cache
  clearing
- **Optional, unified toolbar** — the reader toolbar can be hidden
  (keyboard-first); the topbar, reader, and bulk bars share one button style
- **Keyboard shortcuts** mirroring the TUI defaults (press `?` for the full
  list): `j/k` navigate, `Enter` open, `gg`/`G` top/bottom, `a` archive, `d`
  trash, `t` toggle read, `U` undo, `l` labels, `m` move, `c` compose, `r`
  reply, `E` reply-all, `f` forward, `g`* draft-reply, `y` summarize, `p`
  prompt, `o` suggest labels, `P` action plan, `D` drafts, `O` open in Gmail,
  `s`/`F`/`T`/`S` search, `/` find-in-message, `R` refresh, `N` load more, `M`
  HTML/text, `h` headers, `L` links, `w` save, `Q` saved searches, `H` theme,
  `v` select mode, `Space`/`*` select, `Esc` back. Every shortcut has a `:`
  command equivalent.

\* `g` draft-reply is on the `⋯` menu / `:draft` (the `gg` goto-top sequence
claims a bare `g`).

### Keyboard parity note

GizTUI's TUI is keyboard-first, and the project mandates keyboard↔command
parity. The desktop client keeps that muscle memory: the shortcuts above use the
same default keys as the TUI (`internal/config/config.go` `DefaultKeyBindings`).
The in-app `?` overlay is the discoverable reference.

Two deliberate deviations (see `pkg/desktop/keymap.go`): `T` is search-to
(matching the TUI, where `toggle_threading` is unbound by default — the
conversation toggle is a toolbar button + `:threads`), and `g` draft-reply is
menu/command-only because `gg` goto-top claims a bare `g`.

### Not ported (by design)

`TTS` (read-aloud), vim ranges (`d3d`), and thread-grouped list navigation are
intentionally left out — they don't map well to a GUI. See the "Known non-goals"
section of [docs/DESKTOP_TEST_PLAN.md](../docs/DESKTOP_TEST_PLAN.md).
