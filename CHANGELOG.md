# Changelog

All notable changes to GizTUI (formerly Gmail TUI) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.28.0-rc.1] - 2026-08-14

Release candidate for the hardened SDLC pipeline. This is the first release built
end-to-end by the fail-closed release workflow: exact-tag validation, SBOMs,
checksums, and GitHub provenance attestations. Stable releases stay paused until
this candidate passes the Phase 3 rehearsal.

### 🚀 Features

- **Per-account LLM configuration (Phase 1).** Each Gmail account can now set its
  own model and provider. An `EffectiveLLM` resolver picks the account-scoped
  model over the global default, falling back cleanly when an account has no
  override.
- **Gmail-only filter deletion.** The rules manager (TUI and desktop) can now
  delete a filter that affects only Gmail labels, with a per-rule preview and a
  confirmation step.

### 🐛 Bug Fixes

- **Desktop: repaint band when scrolling image-heavy HTML email.** Scrolling
  image-heavy HTML messages no longer shows a gray repaint band.

### 🛠️ Refactors

- **Command dispatch is table-driven** in the TUI, and **configurable key
  shortcuts are table-driven** too. Bindings and command definitions now live in
  data structures instead of switch statements.

### 🔒 CI and Release Supply Chain

- One fail-closed local gate, `make ci`, reproduces required CI locally.
- Architecture, per-file complexity, and coverage ratchets are enforced in CI.
- Desktop CI (Vitest, Playwright, TypeScript, desktop Go) is complete.
- Releases flow through a single orchestrator: exact-tag checkout, strict version
  agreement across all sources, central checksums, CycloneDX SBOMs, artifact
  attestations, and one draft-first publisher in the protected `release`
  environment.

## [1.27.0] - 2026-08-08

The **TUI half of the keyboard-first CRUD model** shipped for the desktop in 1.26.0.
The terminal saved-searches picker and Prompt Library Manager now open in *list
mode* and share the same `n` / `e` / `d` keys as every other picker, so muscle
memory carries across clients.

### 🚀 Features

- **Saved searches: create from the picker.** Press `n` in the saved-searches picker
  (`Q` / `:bookmarks`) to add a new bookmark inline, alongside the existing `e`
  (edit) — no more dropping to `:save-query` just to create one.
- **Prompts: edit and create in place (TUI).** The Prompt Library Manager
  (`:prompts` / `:prompt list`) now edits the highlighted prompt with `e` and
  creates one with `n` via an inline form, matching the desktop. `Enter` still opens
  the read-only prompt view.
- **List-mode filter in the TUI (desktop parity).** Both pickers open with the row
  list focused, so bare `n` / `e` / `d` are commands; press `/` to focus the filter
  (`@category` still narrows saved searches) and `Esc` to leave it.

### 🐛 Bug Fixes

- **Saved searches: deletes now confirm.** Deleting a bookmark (`d`) asks for a
  second press in the status bar before removing it — `Esc` cancels — instead of
  deleting immediately with no undo.

### 🔧 Changed

- **Prompt Library Manager: export moves to `x`.** `e` now edits the selected prompt
  (was: export); export the highlighted prompt to a file with `x`. Delete (`d`) gains
  the same two-press confirmation as the other pickers.
- New remappable keys under `keys`: `saved_query_new`, `prompt_new`, `prompt_edit`,
  `prompt_delete`, `prompt_export` (defaults `n` / `n` / `e` / `d` / `x`). Existing
  `config.json` files pick up the defaults automatically on upgrade.

## [1.26.0] - 2026-08-08

A **consistent, keyboard-first CRUD model for every editable picker** across both
the TUI and the desktop client, plus **saved-search editing and categories**. Once
you learn `e` / `d` / `n` in one picker, every other picker works the same way.

### 🚀 Features

- **Saved searches: categories.** Group the saved-search picker by category, filter
  to one group by typing `@category`, and assign a category when saving with
  `:save-query <name> @work`. Category headers are tinted with the component's
  title colour so groups are easy to scan. (#18)
- **Saved searches: edit in place.** Rename or change a saved search from the picker
  without deleting and re-creating it, via `e` (or the ✎ button). (#19, #11)
- **Prompts: inline CRUD in the picker.** Create, edit and delete prompts directly
  from the prompts picker; the separate prompt-manager screen is retired, so there
  is one place to manage prompts. New prompts are created with `n`.
- **One CRUD convention everywhere.** Every picker whose rows are editable entities
  — saved searches, prompts, analyzer rules, deterministic rules, jobs — now shares
  the same keys across the TUI and the desktop: `e` edits, `d` / `Delete` deletes,
  `n` creates, `Enter` runs. Deletes ask to confirm first. (#11)
- **Desktop pickers use a list-mode filter (TUI parity).** Pickers with both a
  filter and letter shortcuts open in *list mode* so bare `e` / `d` / `n` act as
  commands; press `/` to focus the filter and type freely, `Esc` to go back.

### 🐛 Bug Fixes

- **Desktop: typing a capital letter in a picker filter no longer fires a command.**
  With the filter focused, `Shift+E` / `Shift+N` are just how you type an uppercase
  `E` / `N`, so a name like "Newsletter" used to trigger new/edit/delete. The new
  list-mode model removes the collision entirely.
- **Desktop: `Esc` while editing a saved search closes only the edit dialog**, not
  the whole picker underneath it.
- **Desktop: keyboard delete restored in the saved-searches picker.**
- **Config: quoted numbers are tolerated and JSON type errors are clearer.** A
  numeric option written as `"5"` in `config.json` no longer fails to load, and a
  genuine type mismatch now reports which key and what was expected. (#40)

### 🔧 Technical Improvements

- **Config: an explicit `Config.Validate()` pass at load** surfaces bad values
  early with actionable messages instead of failing deep in a feature. (#40)
- **Shared UI hooks `usePickerCrud` and `useFilterMode`** centralise the desktop
  edit/delete/new + list-mode-filter behaviour, so all CRUD pickers stay identical
  and future pickers get the convention for free.

## [1.25.1] - 2026-08-03

Bug-fix release for the desktop client: dead email links now open, and the
single-message AI prompt panel behaves correctly.

### 🐛 Bug Fixes

- **Desktop: some email links did nothing on click.** The reader only opened
  hrefs starting with `http(s):`, so protocol-relative links (`//host/…`, common
  in newsletters) and `mailto:` / `tel:` links were silently ignored. They now
  open in the browser / mail client; `javascript:` and `data:` stay blocked.
- **Desktop: single-message prompts now appear in `:jobs`.** A prompt run on one
  message is registered as a job the moment it starts (not only when it finishes),
  so it shows up live in the jobs list alongside bulk prompts.
- **Desktop: the prompt panel no longer loses its `Generating…` state or title.**
  Running prompts on two different messages and switching between them kept its
  own per-message state (previously a single global slot blanked the older one),
  and a restored result with a missing name falls back to a generic heading
  instead of an empty title.

## [1.25.0] - 2026-08-03

Keyboard **focus parity between the TUI and the desktop client**: `Tab` now moves
focus predictably in both, and no side-panel picker can trap it any more. Plus
release-hygiene guardrails that keep the version number and docs from drifting.

### 🚀 Features

- **Desktop: `Tab` toggles focus between the message list and the reader.** The
  GUI equivalent of the TUI's focus ring — `Tab` / `Shift+Tab` hand focus back and
  forth between the inbox list and the reading pane (lighting the reader's focused
  border and routing arrow/`j`-`k` scrolling), matching the terminal client.

### 🐛 Bug Fixes

- **TUI: `Tab` no longer gets trapped in side-panel pickers or the chat panel.**
  Every picker (labels, links, attachments, saved-queries, themes, accounts, AI
  suggestions, prompts, bulk-prompts) and the chat panel now let `Tab` /
  `Shift+Tab` cycle focus back to the message list, as the in-app help already
  promised. Pickers that assign their own meaning to `Tab` (Obsidian form, prompt
  configurator, rules manager) are unaffected.

### 🔧 Technical Improvements

- **CI guardrails against post-release drift.** The command palette and `?` help
  are now checked against the reference docs, and `VERSION`,
  `internal/version/version.go`, `CHANGELOG.md` and the Homebrew cask template are
  pinned to the same version by `go test ./internal/version/` — so a release can't
  ship with a mismatched version number or an undocumented command.

## [1.24.0] - 2026-08-01

A privacy-first, opt-in **local usage-analytics dashboard (`:stats`)** — brought
to **both the TUI and the desktop client** with command parity — plus continued
TUI↔desktop parity work (message stars and the `:numbers` column).

### 🚀 Features

- **Star / unstar messages (key `*`, `:star` / `:unstar`) — desktop & TUI.**
  Toggle Gmail's STARRED flag on the highlighted or open message; `:star` /
  `:unstar` are bulk-aware (act on the whole selection). Starred mail shows a
  dedicated ⭐ column, far-right and aligned in both clients, reflected in place
  without a refetch.
- **Local usage analytics (`:stats`) — privacy-first, opt-in.** A local-only
  usage dashboard that never leaves your machine. Off by default; enable with
  `telemetry.enabled: true` and restart. `:stats` shows totals, top commands, and
  top shortcuts over a window (`:stats <days>`, default 30); `:stats reset` wipes
  it; old events auto-prune by `telemetry.retention_days`. Only bounded
  identifiers are captured (the command word, the key, an action name) — never
  arguments, search queries, or any message content.
- **Action outcomes + timing.** The dashboard's **Actions** section reports
  per-action run counts, failures, and average wall-clock duration for
  high-value actions (archive, trash, summarize), so you can see success/failure
  rates and how long they take — complementing the command/shortcut counts which
  only measure invocation.
- **Desktop parity for `:stats`.** The desktop client now has the same local,
  opt-in usage-analytics dashboard (totals, top commands/shortcuts, and the
  Actions section) reusing the same per-account store; `:stats reset` and
  `:stats <days>` work there too. `:prompt stats` stays the AI prompt-usage view.
- **Desktop parity for message numbers (`:numbers`).** The desktop client can now
  show a 1-based message-number column in the list, toggled with `:numbers` (alias
  `:n`), so jumping with `:N` (e.g. `:14`) is easy to aim. The initial state is
  seeded from `display.show_message_numbers` in config, mirroring the TUI.

## [1.23.0] - 2026-07-31

AI features **and desktop↔TUI integration parity**. Two new AI features — bulk
prompts that run in the background and a multi-turn chat grounded on the open
email — plus the **desktop client catching up to the TUI** on Slack forwarding
and Obsidian ingest, a new Slack **"markdown"** format, and configurable chat
context. All keyboard-first, with command parity.

### 🚀 Features

- **AI background jobs (`:jobs`, key `J`) — desktop & TUI.** Applying a prompt
  across selected emails no longer blocks: it runs as a tracked background job,
  so you keep working and get a toast when it finishes. A `:jobs` picker lists
  jobs (queued / running / done / error) and lets you re-open a result, remove a
  job, or clear finished ones. On the desktop a `runExclusive` gate serializes
  the shared token stream so concurrent AI runs never interleave; the completion
  toast is configurable via `jobs.notify_on_complete`.
- **Chat with this email (`:chat`, key `X`) — desktop & TUI.** A multi-turn AI
  chat grounded on the open message, with per-message history. The desktop shows
  a chat panel in the reader (transcript + input; Enter sends, Shift+Enter
  newline); the TUI shows a side panel with color-coded turns and history restore
  on reopen. Backed by a new `ChatService` that reuses the existing streaming AI
  API — no LLM-provider changes.
- **Slack forward parity on the desktop.** The desktop's "forward to Slack" (`K`,
  `:slack`) now opens a keyboard-first picker to choose the channel and add an
  optional pre-message, and honors the configured `slack.defaults.format_style`
  instead of hardcoding one — matching the TUI. A `full` forward now includes the
  rendered message body instead of a "content not available" placeholder.
- **New Slack `markdown` forward format.** A middle ground between `summary`
  (AI-condensed) and `full` (everything): the email's HTML is rendered through the
  same clean-markdown pipeline the reader uses, dropping newsletter cruft. Pick it
  per-send from the desktop picker (keyboard: `Tab`) or set it as the default.
- **Obsidian ingest dialog on the desktop.** `O` / `:obsidian` now opens a dialog
  for an optional comment that is rendered into the note (`> **Note:** …`), at
  parity with the TUI (the desktop previously dropped the comment).
- **Desktop `:config migrate`.** Runs the config self-migration from the desktop
  command bar (it previously just opened the read-only config modal).

### 🚀 Improvements

- **AI grounds on the visible email text.** Summary, reply, touch-up and chat now
  read the *rendered* HTML text of a message instead of the raw HTML / plain-text
  part, so they no longer answer with hidden preheaders or "can't view this
  email" boilerplate.
- **Auto-refresh state is durable and config-driven.** Toggling auto-refresh now
  persists to `config.json`, and startup reads the config as the single source of
  truth (a stale local override no longer makes `enabled: true` show as off). This
  also stops a config left at `enabled: true` from silently re-arming the TUI's
  Slack new-mail digest on every launch.

### 🐛 Bug Fixes

- **TUI — opening a finished job from `:jobs` could hang** (a `QueueUpdateDraw`
  ran on the tview event goroutine); the result now renders synchronously.
- **TUI chat — no feedback on send.** The message and a "…thinking…" cue now
  appear instantly, and reopening chat restores the prior conversation.
- **Desktop — the frameless window couldn't be dragged;** the top bar is now a
  window drag region.
- **Chat with a long email answered "not mentioned" for later content.** The chat
  grounding body was capped at a fixed 8000 characters, so questions about
  anything past that point in a long newsletter were answered as if it wasn't
  there. The cap is now higher (24000) and configurable.
- **Desktop Slack format was mouse-only.** The forward picker's format selector is
  now fully keyboard-operable (`Tab` cycles format, `↑↓` the channel).

### 🔧 Configuration

- New keys, added to defaults and surfaced to existing users via `:config migrate`:
  `jobs.notify_on_complete` (default `true`), `llm.chat_template`
  (`templates/ai/chat.md`), `keys.ai_jobs` (`J`) and `keys.chat` (`X`).
- More new keys: `llm.chat_max_body_chars` (default `24000`) and
  `llm.chat_max_turns` (default `12`) tune how much email and how many prior turns
  ground the chat; `slack.defaults.format_style` now also accepts `"markdown"`.

## [1.22.1] - 2026-07-29

Maintenance release: the internal decomposition of the **GizTUI Desktop** client
is complete, plus two correctness fixes. No changes to the TUI or to the desktop
feature set — behavior is preserved.

### 🐛 Bug Fixes

- **Desktop — calendar-invite times shown in the wrong timezone.** The RSVP
  preview and picker displayed the organizer's wall-clock time instead of the
  viewer's. Invite `DTSTART`/`DTEND` are now resolved to an absolute instant
  (honoring `TZID` / UTC) and rendered in the viewer's local zone — a 09:00-CEST
  event now shows 09:00. The IANA timezone database is bundled so zones resolve in
  the packaged app.
- **Desktop — build failed on case-insensitive filesystems (macOS).** Pure-logic
  modules that collided by case only with their components were renamed
  (`compose.ts` → `composeBuilders.ts`, `messageList.ts` → `messageListModel.ts`),
  so `tsc` / `wails dev` build cleanly. Linux CI (case-sensitive) never surfaced it.

### 🛠️ Technical Improvements

- **Desktop `App.tsx` decomposition complete.** The former ~1.2k-line god
  component is now a thin orchestrator: the render moved to `AppInbox` / `AppModals`,
  global keyboard + command wiring to `useAppWiring`, and subsystem state into
  dedicated hooks (`useAiActions`, `useActionPlan`, …). **Every code file in the
  `desktop/` module (Go, TS/TSX, CSS) is now under 500 lines.** Behavior-preserving,
  covered by 62 vitest unit tests + 39 Playwright e2e specs.

---

## [1.22.0] - 2026-07-25

Maturation pass over the **GizTUI Desktop** client (introduced in 1.21.0):
command parity with the terminal, keyboard-first pickers everywhere, and a long
list of correctness and polish fixes. TUI behavior is unchanged except for a
shared date-parsing improvement.

### ⌨️ Desktop — command parity with the TUI

- **Command set aligned with the terminal.** The `:` palette now mirrors the
  TUI's taxonomy: analyzer rules and the analyzer-prompt preview are subcommands
  of `:action-plan` (`:action-plan rules`, `:action-plan prompt`), deterministic
  rules run via `:rules plan` / `:rp`, and the invented one-off commands were
  removed. Missing canonical aliases were added (`:u`, `:reply-all`, `:link`,
  `:p`/`:pr`, `:open-web`, `:save-query`/`:sq`, `:h`).
- **New commands:** `:rsvp` (open the RSVP picker), `:toggle-read` / `:t`
  (flip read↔unread), plus `:quit` / `q`.
- **Help screen** updated to list the new keys and commands.

### 🖥️ Desktop — keyboard-first pickers & UI consistency

- Keyboard navigation in the **prompt manager**, **analyzer-rules** modal and
  **saved-searches** picker (arrows to move, `d`/Delete to remove); the
  analyzer-prompt preview scrolls with the keyboard.
- Every list picker's **delete** action is now the same flat red trash icon,
  sized to match the rest of the icon set; remaining emoji (📎 / 📅) were
  replaced with SVG icons.
- Command-palette suggestion rows no longer overlap when a command shows an
  argument hint.

### 🐛 Desktop — correctness fixes

- AI panels are tracked **per message**: applying a prompt, switching messages
  and returning no longer bleeds one message's summary/prompt onto another, and
  the panel title no longer blanks out while a run is still streaming.
- Prompt results are **cached per (message, prompt)** and persisted in the DB,
  so re-running a prompt reuses the result; editing a prompt invalidates it and
  a regenerate button forces a fresh call.
- New mail from the background poll is offered via a **banner** instead of being
  injected at the top of the list (which previously scrambled the inbox and
  risked acting on the wrong message after deletes).
- `from:` / `to:` searches scoped to the inbox by matching `in:` / `label:` as
  real tokens (not substrings), and self-addressed mail is no longer hidden.
- Remote images are remembered per message with an "always load" toggle, images
  whose resize query got mangled to U+FFFD are recovered, and gray repaint bands
  when arrow-scrolling HTML email are gone.
- Links picker decodes HTML entities in anchor text and URLs (e.g.
  `Versi&oacute;n web` → `Versión web`).

### 📅 Shared

- Date parsing accepts more header formats and falls back to the message's
  `internalDate` instead of "now", fixing list dates that "jumped" (TUI + desktop).

### 📦 Desktop — packaging

- First-run **onboarding** when `credentials.json` is missing (explain + native
  file import) instead of a raw startup error.
- Release packaging fixes: build the frontend before `wails build`, working NSIS
  installer and Linux AppImage, and Homebrew cask install docs.

## [1.21.0] - 2026-07-24

### 🖥️ New — GizTUI Desktop (Wails client)

- **A native desktop client for GizTUI.** A new Wails app (`desktop/`) that reuses
  the exact same `internal/services` stack as the TUI, so it is a genuine second
  front-end rather than a separate product. It aims for **feature parity with the
  terminal** and stays **keyboard-first**: your configured keybindings drive the
  list, reader and every picker. Runs on macOS (WKWebView) today; the packaging
  for distributable installers is designed in `docs/DESKTOP_DISTRIBUTION.md`.
- **Reading & triage.** Message list with `j`/`k` preview (no accidental
  read-marking), Enter to open and scroll the message with the keyboard, archive/
  trash/read/label/move, undo, bulk mode with VIM-style range ops, and a loaded-
  message count. HTML email renders in a sanitized Shadow DOM (DOMPurify) with
  remote and `cid:` inline images proxied through the backend; Markdown from AI
  output is rendered, not shown raw.
- **Search.** Gmail search, quick from/to/subject searches, a local filter toggle,
  saved queries, and an **advanced search builder** (Ctrl+F).
- **AI & rules.** Summaries, prompt library (single + bulk), AI-suggested labels,
  the **inbox action plan** (parallel batched analysis with real progress,
  recategorize/move/label/run-prompt buckets, read-manually bucket), and the
  **deterministic rules** manager (`:rules`) with a rules-only run (`:rules run`)
  that applies your rules without the LLM.
- **Standardized keyboard-first pickers** (labels, links, prompts, saved queries,
  themes, RSVP, move-to-folder, attachments, accounts, suggested labels, rules)
  that all navigate with arrows/1-9/Enter/Escape under WKWebView.
- **Integrations.** Obsidian, Slack, RSVP to calendar invites (with the calendar
  scope), attachments download, open-in-Gmail, drafts, theme picker, account
  switching, and a browser sign-in flow with a branded OAuth redirect page.

### 🚀 Improvements (shared with the TUI)

- **Inbox analyzer runs batches in parallel** with a configurable concurrency and
  emits real per-batch progress, so large inboxes analyze faster and show
  meaningful progress instead of an indeterminate spinner.

## [1.20.1] - 2026-07-12

### 🐛 Fixes

- **Accented text no longer shows as mojibake in some emails.** Certain senders (e.g. corporate systems) declare an ISO-8859-1/windows-1252 charset on a message body that is actually UTF-8. GizTUI trusted the label and re-decoded the bytes, turning `Buenos días` into `Buenos dÃas` and `Quirónprevención` into `QuirÃ³nprevenciÃ³n`. The body decoder now trusts valid UTF-8 over a conflicting charset label, so mislabeled UTF-8 renders correctly while genuine Latin-1 mail is unaffected.
- **`:` commands no longer freeze the app on an error or usage message.** Several command handlers (`:rules <unknown>`, `:action-plan with-prompt`/unknown option, `:prompt refine`/`save` guards, `:load` when the list isn't focused) showed their error message directly on the UI event loop, which deadlocked the whole app — e.g. typing `:rules apply` and pressing Enter froze GizTUI. Those status messages are now dispatched off the event loop, so the command reports the problem and stays responsive.

## [1.20.0] - 2026-07-11

### 🚀 Features — Deterministic rules (`:rules`)

- **A new deterministic rules engine you manage with `:rules`.** Define plain, predictable rules — a Gmail search query plus one action (archive, mark read, trash, apply a label, or run an AI prompt) — and manage them in a dedicated side panel. Rules are matched **first-match-wins** in creation order, so each email is handled by exactly one rule. No AI guesswork: if the query matches, the action runs. Create a rule from scratch (`:rules new`), or turn your current search straight into one (Ctrl+S from the list, or `:rules new` while a search is active). A **Preview** button runs the query as a live search so you can check what it matches before saving.
- **Rules can mirror as real Gmail filters.** Tick **"Also in Gmail"** (Space) when saving and the rule is created as an actual Gmail filter too, shown with a ☁️ in the list. This means the rule keeps working server-side even when GizTUI isn't running. Deleting a mirrored rule deletes the Gmail filter as well.
- **Your existing Gmail filters fold into `:rules` automatically.** Every time you open `:rules`, GizTUI reads your Gmail filters and reconciles them with your rules: translatable filters become rules (mirrored, ☁️), filters that match a rule you already have are linked to it, and filters GizTUI can't represent (forwarding, combined actions, size criteria) are shown read-only so the list stays a complete picture of everything acting on your inbox.
- **`:rules` stays in sync with Gmail.** If you delete or edit a filter over in Gmail, reopening `:rules` reflects it — a mirrored rule whose filter no longer exists is dropped to follow Gmail. Rules you created only in GizTUI (no ☁️) are never touched, and nothing is removed if Gmail can't be reached (you get a warning and your local rules stay put).
- **Deleting a rule now asks for confirmation.** Pressing the delete key arms a two-press confirmation in the status bar (press again to confirm, Esc cancels) — consistent with the rest of the app. The prompt adapts: a local-only rule warns it exists nowhere else, a mirrored rule warns its Gmail filter goes too.
- **The AI Action Plan runs your deterministic rules first.** With `inbox_analyzer.deterministic_prefilter` (new, default `true`), emails that match a rule are handled deterministically before the language model sees the rest — faster, cheaper, and predictable for the mail you already know how to sort. You can run a rules-only preview any time with `:rules plan`.

### 🔧 Notes

- Filter mirroring needs the `gmail.settings.basic` OAuth scope. New installs request it automatically; if you're upgrading, re-authenticate to enable "Also in Gmail" (everything else works without it).
- New keys/commands are documented on the in-app `?` help screen and in `docs/KEYBOARD_SHORTCUTS.md`.

## [1.19.1] - 2026-06-28

### 🐛 Fixes

- **Fixed a latent crash in bulk selection.** The set of bulk-selected messages was an unsynchronized map read from background operation goroutines (bulk archive/label/prompt/Slack/Obsidian) while being modified on the event loop — a concurrent-map access that could panic under the right timing. It is now behind a small mutex-guarded type, eliminating the race.

### 🔧 Internal

- **God-object decomposition (#49) wrapped up.** Extracted `bulkState` and `focusState` from the 3,400-line `App` struct and removed dead/orphan fields; `App` is down from 129 to 89 fields. Notably, the focus name (e.g. `"labels"`) used to be written twice at ~95 call sites (`currentFocus = name` + `updateFocusIndicators(name)`) — a drift footgun now collapsed into a single `markFocus(name)` helper. No user-visible behavior change. The remaining loose field groups were analyzed and consciously left as-is (see #52).

## [1.19.0] - 2026-06-24

### 🚀 Features

- **Tab autocompletion in the `:` command bar.** Pressing Tab now cycles through the commands that match what you've typed (Shift+Tab goes back, Enter runs whichever is shown); a single match completes fully. After a command + space, Tab completes its arguments where it applies, honoring each command's real grammar:
  - subcommands — `:labels add/list/remove`, `:prompt list/create/update/export/delete/stats`, `:theme list/preview/set`, `:accounts switch`;
  - then names where a name is expected — label names (`:labels add <name>`), theme names (`:theme set <name>`), account ids (`:accounts switch <id>`), saved-query names (`:bookmark <name>`);
  - Gmail search operators after `:search` (`from:`, `has:attachment`, `is:unread`…).
  Completion is driven by a single command registry (replacing the old hand-maintained prefix map), so every command autocompletes and stays in sync. A greyed inline hint shows the first candidate as you type.

## [1.18.0] - 2026-06-22

### 🐛 Fixes

- **AI summary streaming no longer risks freezing on ESC.** The single-message summary (`y`) and thread summary (`Shift+T` / `:thread-summary`) updated the view via `QueueUpdateDraw` inside their streaming callbacks, which the project's own rules forbid because it can deadlock with the ESC handler. They now update the view directly (guarded by the request context), matching the other AI streaming paths. Pressing ESC mid-stream cancels cleanly and the app stays responsive.
- **`:expand-all` / `:collapse-all` now work on every thread.** They previously did nothing for threads you had never individually expanded/collapsed (a blanket `UPDATE` only touched threads that already had a saved state row). They now affect all currently-displayed threads.
- **Fixed three latent concurrent-map crashes.** The calendar-invite cache, the AI in-flight tracker, and the message cache used during Slack bulk-forward were read/written from multiple goroutines without synchronization. Consolidated behind a single guarded cache type.
- **Fixed latent data races** on the UI-ready / welcome-animation flags, draft-mode reads, and the AI streaming-cancel function (ESC could observe a cancel func a goroutine was clearing).

### 🔧 Changes

- **Cleaner startup — no more spurious key-conflict warnings.** The startup check no longer cries wolf over keys intentionally shared across separate screens (e.g. `a` = archive in the list vs add-rule in the Action Plan). A genuinely new same-context collision is still reported.
- **Threading keys `T` / `E` are unbound by default.** They were eclipsed by quick-search-to (`T`) and reply-all (`E`) in the global key handler and never actually toggled threading. Use the `:threads` / `:flatten` / `:expand-all` commands, or bind a free key in your config. Help screen and `docs/KEYBOARD_SHORTCUTS.md` updated.
- **`giztui --migrate-config`** (and `:config migrate`) brings an older `config.json` up to date with newly added defaults.

### 🔧 Internal

- **God-object decomposition (#49) continued, scorecard-driven.** Extracted `overlayBackup`, `draftState`, `uiLifecycle`, `layoutState`, `appCaches`, and `aiPanelState` out of the 3,400-line `App` struct — each a small, unit-tested type — fixing the latent races above along the way. Prioritized with an explicit decision scorecard (`docs/superpowers/REFACTOR-SCORECARD.md`).

## [1.17.0] - 2026-06-21

### 🚀 Features — Action Plan & faithful labels

- **Action Plan: Enter opens the email in the reader.** Pressing Enter (or →) on an email node in the Inbox Action Plan selects it in the message list and loads its body into the reader, then moves focus there so you can scroll and read immediately. An email that was archived/moved out of the list still loads by id. Category nodes keep their expand/collapse behavior; Space/`m`/`i` are unchanged. (#51)
- **Faithful labels: the analyzer no longer invents or duplicates labels.** With `inbox_analyzer.strict_labels` (new, default `true`), the analyzer's `label` action may only use a label you already have (matched ignoring case/whitespace). An email whose suggested label matches none of yours goes to "review manually" instead of creating a new/duplicate label — a real help with weak local models. Set it to `false` for the old create-on-miss behavior; if you have no labels at all, enforcement is skipped. The existing-labels prompt instruction is now imperative ("use ONLY a label from this exact list").
- **Action Plan label rows show the actual label.** A label group now reads `Label → <your label> · n/n · PRIORITY` — the real label that will be applied, dropping the redundant model-invented group name (kept for non-label actions like Archive, where it describes what's affected).

### 🔧 Internal

- **God-object refactor pilot (#49):** extracted the VIM key-sequence state out of the 3,400-line `App` struct into a self-contained, mutex-protected `vimState` type with unit tests, fixing a latent locking inconsistency (the event loop wrote the fields without a lock while a timeout goroutine used `App.mu`). No change to VIM behavior (`gg`/`G`/`s5s`/single-op-after-timeout).

## [1.16.0] - 2026-06-19

### 🚀 Features — Slack new-mail summaries

- **Per-email AI summary in the auto-refresh Slack notification.** When auto-refresh detects new mail and `auto_refresh.notify_slack` is on, the notification can now include a short AI-generated summary of each new email, rendered as a Slack blockquote under its row. Opt-in via `auto_refresh.slack_summary` (default `false`, because it adds an LLM call per email).
- **Configurable cap.** `auto_refresh.slack_summary_limit` (default `5`) bounds how many emails are summarized per refresh cycle; the rest appear without a summary. Values ≤ 0 are treated as 5.
- **Digest-specific prompt.** `auto_refresh.slack_summary_prompt` overrides the summary prompt; the tuned default produces one factual sentence with no URLs, signatures, or automation/sender boilerplate. Supports `{{body}}` and `{{max_words}}`.

### 🐛 Fixes

- **`GMAIL_TUI_CREDENTIALS` is now honored.** The environment variable was advertised but ignored when resolving the credentials path; it now takes effect (CLI flag → env var → config → default).
- Slack summaries render as blockquotes instead of `_italic_`, which previously showed literal underscores when the text contained `@mentions`, `#refs`, or URLs.

### 🔧 Internal

- Dead-code sweep (removed unused package/manager/helpers) and extraction of narrow Gmail-client interfaces (`GmailClient`, `LabelClient`, `LinkClient`, `SlackGmailClient`) so the email/label/link/slack services are unit-testable with mocks. Expanded test coverage across services, render, llm, and obsidian.

## [1.15.0] - 2026-06-14

### 🚀 Features — Text-to-speech on macOS, multilingual

- **Zero-setup TTS on macOS**: a new `say` engine uses the built-in macOS `say` command — no Piper binary, no voice model, no dependencies. The engine **auto-detects the OS** (`tts.engine: "auto"`, the default): macOS → `say`, other platforms → Piper. Just bind `keys.speak`.
- **Reads each email in its own language**: GizTUI auto-detects the text's language and picks a matching voice, so English mail is read by an English voice and Spanish by a Spanish voice. Configure per-language voices with `tts.voices` (say) or `tts.models` (Piper), e.g. `"voices": {"en":"Samantha","es":"Mónica"}`. Detection is restricted to the languages you configure, which keeps it accurate even on short emails.
- **Streaming playback for `say`**: it starts speaking almost immediately (no synthesize-the-whole-thing-first delay) and stops instantly when you press the speak key again.

### 🐛 Fixes

- Pressing the speak key to **stop** no longer surfaces a spurious "TTS failed" error — a user-requested stop is treated as a clean stop.

See **[docs/TTS.md](docs/TTS.md)** for setup (the macOS quick-start needs nothing installed).

## [1.14.0] - 2026-06-14

### 🚀 Features / Fixes — focus cycling

- **Tab now cycles through all visible panes, including the message reader.** Previously Tab effectively toggled only list ↔ side picker and skipped the reader (the focus ring matched panes by the focused widget's pointer identity, which never matched a picker's inner input). Cycling is now keyed by pane name, so the order is deterministic: **list → reader → picker / Action Plan → AI summary → Slack** (only the panes currently visible).
- **The Action Plan panel is now part of the cycle too.** Before, Tab with the Action Plan open only toggled panel ↔ list and never reached the reader; now `list → reader → Action Plan` cycle fully (the panel keeps analyzing while focus is elsewhere).
- **`Shift+Tab` cycles focus in reverse** through the same ring.

### 🐛 Fixes

- **Action Plan no longer drops selected emails.** When the analyzer model returned only a subset of the analyzed messages (omitting the rest from both categories and `read_manually`), those emails silently vanished from the plan. Any message not placed in a category is now reconciled into **Read manually**, so every selected/analyzed email always appears.

## [1.13.0] - 2026-06-14

Consolidating round from the v1.7.0→v1.12.0 end-to-end test pass.

### 🐛 Fixes

- **Thread summary key conflict**: `thread_summary` defaulted to `shift+t`, which is the same physical key as `toggle_threading` (`T`) and was always shadowed. It is now **unbound by default** — use the `:thread-summary` command, or bind a free key via `keys.thread_summary`.
- **Thread-summary panel rendered in a sliver**: the AI Summary panel was only shown *after* generation, so streamed text wrapped into a ~4-character column and opened without focus. The panel is now opened, sized, and focused **before** generating, matching the single-message summary (`y`).
- **Action Plan footer hints ignored your config**: the panel footer advertised hardcoded keys (e.g. "v prompt") instead of your actual bindings. Hints now reflect `keys.view_prompt`, `keys.remember_rule`, `keys.move`, and `keys.bulk_select`.

### ✨ Improvements

- **Better analyzer prompt structure**: the analyzer prompt now leads with the assistant role + rules + output format, and injects the existing-labels and user-interests **context right before the messages** (previously those blocks were prepended on top, burying the role). Key matching also now understands `shift+<letter>` combos.
- **Slack new-mail notifications are clickable**: each email in the auto-refresh Slack notification is now a hyperlink that opens the message directly in Gmail.

## [1.12.0] - 2026-06-13

### 🚀 Features — fully configurable key bindings

- **Every action key can now be rebound.** Keys that were previously hardwired to their defaults now honor `keys.*` config: thread summary, undo, the search-box mode toggle and advanced-search keys, prompt preview, the Action Plan remember-rule / view-prompt / move / exclude keys, the analyzer-rules add/delete keys, saved-query delete, attachment save, link copy, and compose send.
- **New `keys.*` options**: `search_toggle_mode` (default `ctrl+t`), `search_advanced` (`ctrl+f`), `prompt_preview` (`ctrl+p`), `remember_rule` (`ctrl+r`), `view_prompt` (`i`), `rule_add` (`a`), `rule_delete` (`d`), `saved_query_delete` (`d`), `attachment_save` (`ctrl+s`), `link_copy` (`ctrl+y`), `compose_send` (`ctrl+j`). They work immediately (defaults are applied at load); run `:config migrate` to also write them into an existing `config.json` for editing.
- Key matching now understands `shift+<letter>` combos (e.g. `shift+t`) in addition to `ctrl+<letter>` and plain single-character keys.

### 🐛 Fixes & cleanup

- Removed the dead `keys.prompt_test` option (it never had a handler). `:config migrate` now also **prunes obsolete keys** from your config file, not only adds new ones.
- Saved-query delete now responds to `d` only (previously `d`/`D`) and is configurable via `keys.saved_query_delete`.

## [1.11.1] - 2026-06-13

### 🐛 Fixes — configurable key bindings now actually honor the config

- **Ctrl-combo bindings for `speak` / `auto_refresh` now fire**: both keys were read from the config but only matched as a plain single character, so a `ctrl+...` binding (e.g. `keys.speak: "ctrl+e"`) was silently accepted yet never triggered. They now route through the Ctrl-combo matcher.
- **Navigation/prompt shortcuts respect reconfiguration**: `fast_up`/`fast_down`/`word_left`/`word_right`, `content_search`/`search_next`/`search_prev`, `next_thread`/`prev_thread`, and the prompt-configurator keys (`prompt_regenerate`/`save_prompt`/`prompt_apply`) were hardwired to their default keys and ignored any custom binding. They now match against the configured value (case-sensitive for plain single-character keys).

## [1.11.0] - 2026-06-13

### 🚀 Features

- **Text-to-speech (read aloud)**: an opt-in, context-aware key reads the focused panel — the message reader, an AI summary, or an Action Plan digest — aloud using a local neural TTS engine ([Piper](https://github.com/rhasspy/piper)); press it again to stop. Configure `tts.piper_path` / `tts.model_path` and bind `keys.speak`. Full setup guide in **[docs/TTS.md](docs/TTS.md)**.
- **Auto-refresh → Slack notification**: with `auto_refresh.notify_slack` enabled, auto-refresh also posts a Slack message (count + subject/sender) to your default Slack channel when new mail arrives.

## [1.10.0] - 2026-06-13

### 🚀 Features — smarter Action Plan analyzer

- **Interests / relevance**: saved analyzer rules are now treated as **interests** too. Write things like "I'm interested in AI" via `:plan rules` (or `Ctrl+R`); the analyzer surfaces matching emails (priority "high" + a note in the category description) instead of burying them in a bulk action. The rules UI is relabeled "rules & interests" for discoverability.
- **Prefers your existing labels**: the analyzer is now given your existing user labels and prefers an exact match for the `label` action, only inventing a new label when none fits — so it stops recommending labels you don't have.
- **`summarize` action (AI digest)**: the analyzer can mark a category as `summarize`; press your summarize key (`y`) on it to get a single combined AI digest of its emails in an in-place panel (Markdown, `Esc` to return) — for mail you'd rather skim than read fully or discard.

### 🐛 Fixes

- **`:plan rules` now takes focus**: opening the analyzer-rules picker via `:plan rules` left focus on the message list; it now correctly focuses the picker.

## [1.9.0] - 2026-06-13

### 🚀 Features

- **Per-item progress for bulk operations**: bulk archive / trash / mark-read / mark-unread / label now show live progress in the status bar ("Archiving 3/10…"), for both the inbox bulk selection and Action Plan group dispatch — no more "looks hung" on large selections.
- **Config self-migration**: new options added in newer releases can be pulled into your existing `config.json` with `:config migrate` (or `giztui --migrate-config`), which adds any missing default keys while preserving your values and `_comment` annotations (writing a `.bak` backup first). The app also notifies on startup when new options are available. Your config keeps working regardless; this just makes new options visible and editable on machines with an older config file.

### 🐛 Fixes

- **"Fetching email bodies…" no longer lingers**: the Action Plan analyzer's body-fetch progress message was persistent and never cleared; it now clears once the fetch completes.
- **`:arr <duration>` now enables auto-refresh**: previously `:autorefresh 1m` only set the interval and silently stayed off if it wasn't already running. It now enables and starts auto-refresh in one step.

## [1.8.0] - 2026-06-12

### 🚀 Features

- **Opt-in inbox auto-refresh**: a configurable background poll detects new inbox mail. While you are on the plain inbox with nothing open (no picker, search, bulk selection, or composer), new mail is **prepended in place** without disturbing your cursor or view; otherwise a `📬 N` pending counter appears in the status bar (load it with `R`). The status bar shows `⟳` while enabled. Configure via `auto_refresh.enabled` / `auto_refresh.interval` (default off / `5m`, 1-minute minimum); toggle at runtime with `:autorefresh` / `:arr`, set the interval with `:arr 2m`, or bind an optional key via `keys.auto_refresh`. Only the plain inbox is polled (search/folder/thread views idle the ticker).
- **Effective analyzer-prompt viewer (`v`)**: press `v` in the Action Plan to see the exact prompt the analyzer assembles — your saved-rules block plus the base prompt (default or the one opened with `:plan with-prompt`), with `{{messages}}` shown literally and a note on how it is filled per batch. Read-only, in-place; Esc returns to the tree.

### 🐛 Fixes

- **Analyzer rules UI no longer hangs**: the `:plan rules` manager was a floating modal that could not be closed (you had to kill the app). It is now an in-place side-panel picker — `a` adds a rule via an embedded input, `d` deletes, Esc closes — matching the other pickers. The `Ctrl+R` quick-remember input in the Action Plan is likewise now an in-place input instead of a floating modal.

## [1.7.0] - 2026-06-11

### 🚀 Features

- **Richer Action Plan analysis (email body, not just snippet)**: the inbox analyzer now includes each email's plain-text body — truncated, opt-in — in the LLM context in addition to subject/sender, substantially improving classification quality. Controlled by `inbox_analyzer.include_body` (default `true`) and `inbox_analyzer.body_char_limit` (default `1000`). Bodies are fetched concurrently only for the messages actually analyzed (capped at `batch_size × max_batches`), with progress feedback and graceful fallback to the snippet if a body can't be fetched. Set `include_body` to `false` for very large inboxes or slow local models.
- **Bulk-move a whole category in the Action Plan (`m`)**: pressing `m` on a category header — or the "Read manually" group — now moves *all* of that group's emails to a chosen destination at once (another category or a standard action), preserving each email's excluded/checked flag. Pressing `m` on a single email still moves just that one.

## [1.6.1] - 2026-06-10

### 🐛 Fixes

- **Action plan footer now tracks the cursor**: navigating the tree could leave the bottom hint one keystroke behind (e.g. showing email actions while a category header was highlighted). The footer now repaints in step with the selection.
- **Archiving an all-excluded action plan category no longer hangs**: pressing the archive key on a group with 0 checked emails (footer "(0)") deadlocked the UI; the warning is now shown without blocking.
- **Bulk-style prompts work on a single email**: applying a prompt that uses the `{{messages}}` placeholder to one message via the single-message picker now substitutes the email content (previously the model received no content).
- **Prompt templates show real line breaks**: the built-in prompts stored literal `\n` sequences (visible in the preview); they now contain real newlines, and a migration repairs existing databases.

### 🚀 Features / Improvements

- **Applied-prompt results render Markdown**: prompt results (single and bulk) now go through the same renderer as the email reader, so tables, headings and width-fitting work and raw `| … |`, literal `<br>`, and `??` glyphs are gone.
- **Prompt preview polish**: the preview hides the search field while open, and bolds the **Description** / **Template** headings.
- **Version on the help screen**: the `?` help now shows the GizTUI version at the top.
- **Consistent prompt-type icons**: bulk-analysis prompts now show the 🚀 icon in the single-message picker too (matching the bulk picker).

## [1.6.0] - 2026-06-09

### 🚀 Features

- **Inbox Action Plan rework (`P` / `:action-plan`)**: the AI triage panel was substantially reworked for trust and control.
  - **Selection-scoped analysis**: analyzes your current bulk selection if you have one, otherwise falls back to the unread inbox. The title shows the scope (e.g. "5 selected" vs "23 unread (inbox)").
  - **Expandable categories with per-email control**: categories are now a navigable tree — `Enter`/`→` expands a group to reveal the actual emails, `←` collapses. Press `Space` to exclude an individual email (`☑`/`☐`); actions act on checked-only. Messages the model declined to categorize appear under a "read manually" group so nothing you selected silently disappears.
  - **Move / recategorize (`m`)**: on an email, press `m` to reassign it to another destination (standard actions or an existing category). The move is recorded in the plan and applied when you dispatch that group.
  - **Non-blocking analysis**: while analysis runs you can `Tab` to the inbox to read mail; `Tab` returns to the panel. The panel keeps analyzing in the background, and `Esc` closes it only when it is focused.
  - **Preference learning**: press `Ctrl+R` to save a free-text, natural-language rule (e.g. "always keep emails from my manager") that the analyzer applies on future runs; manage saved rules with `:action-plan rules`. Rules are interpreted by the LLM (not literal matching) and stored per account.
- **In-place picker panels**: the prompt preview and the action-plan move chooser now open *inside* their panel as deeper navigation instead of as floating modals — consistent, and they no longer get stuck.

### 🐛 Fixes

- **Prompt preview was unclosable** (`p` → `Ctrl+P`): the floating preview could not be dismissed (neither `Esc` nor `Ctrl+P`) and required killing the app. It is now an in-place panel that closes reliably; `Enter` applies the highlighted prompt, `Esc`/`Ctrl+P` returns to the list. Same treatment in the bulk prompt picker.
- **Action plan "move" hung the app**: selecting a destination froze the UI (a deadlock from calling a status update synchronously on the UI thread). The status message is now dispatched off-thread.
- **Action plan footer showed the wrong context**: navigating the tree could show "email" actions while a category header was highlighted (and vice versa), because a programmatic cursor move did not refresh the selection state. The footer now always reflects the highlighted node.
- **Action plan focus/navigation after analysis**: replacing the scroll-only view with a tree fixes the lost focus and broken `↑`/`↓` navigation after a streamed analysis completed; the panel border is also highlighted when focused.
- **Email rendering**: repaired malformed 256-color palette tags (e.g. `[#0099ff-::b]`) that leaked as literal text into the reader for some Markdown headings.

### 🔧 Technical Improvements

- New `AnalyzerRulesService` + `analyzer_rules` table (DB migration v8, account-scoped) feeding user preference rules into the analyzer prompt via `InboxAnalyzerOptions.UserRules`; accessed through a dedicated `GetAnalyzerRulesService()` getter.
- Action plan panel rebuilt on `tview.TreeView`; selection derived from the current node (`syncSelectionToNode`) as the single source of truth; context-aware footer.
- Shared `showPromptPreviewInline` helper for the single and bulk prompt pickers; in-place body-swap pattern with a `currentFocus` pass-through in the global key router so focused in-panel widgets own their keys.
- Spec and implementation plans for the rework and the in-place panels under `docs/superpowers/`.

## [1.5.0] - 2026-06-08

### 🚀 Features

- **Prompt preview (Ctrl+P)**: in the prompt picker (single message or bulk), press `Ctrl+P` with a prompt highlighted to open a popup showing its description and full template, so you can confirm what a prompt does before applying it. Press `Esc` or `Ctrl+P` again to close and return to the picker. Works whether focus is on the search field or the list.

### 🐛 Fixes

- **Prompt picker applied the wrong prompt**: pressing Enter from the search field applied the first prompt regardless of what was highlighted (and the "✨ Create new with AI…" entry could never be chosen that way). Enter now acts on the highlighted item, and the highlight follows the first match while filtering. Fixed in both the single and bulk pickers.
- **Focus lost after bulk actions**: after a bulk archive, trash, mark-read/unread, Obsidian, or Slack operation, neither pane showed the active highlight until you pressed `Tab`. Focus and the focus-indicator border now return to the message list automatically.
- **Email rendering (Markdown)**:
  - Removed non-homogeneous background bars caused by glamour styling code blocks, rules and tables with their own backgrounds that clashed with the reader pane theme.
  - Eliminated the `??`/`�` glyphs in rendered tables and rules: box-drawing characters (East-Asian-Width "Ambiguous", which tcell could render double-width and clip) are now mapped to ASCII so lines fit the pane.
  - Made link URLs readable — dropped glamour's black foreground that was effectively invisible on the dark reader pane (links keep their underline).
- **In-app help (`?`) was out of date**: corrected the `M` shortcut description (it toggles Markdown rendering, it does not "export"), and documented the AI touch-up toggle (`:touch-up`), the account picker (`Ctrl+A` / `:accounts`), and the inbox Action Plan (`P` / `:action-plan`).

### 🔧 Technical Improvements

- Added a shared `focusList()` helper centralizing focus + indicator restoration, and a `promptPickerSelection()` helper shared by the picker Enter handling and the new preview.
- Render pipeline post-processing (`stripTagBackgrounds`, `asciiBoxDrawing`, `fixLinkContrast`) applied to glamour output before display, each covered by unit tests.
- Added design specs and implementation plans for the prompt preview (shipped here) and the upcoming auto-refresh inbox toggle under `docs/superpowers/`.

## [1.4.1] - 2026-06-08

### 🐛 Fixes

- **Markdown rendering polish** (follow-ups to v1.4.0):
  - Collapse intra-line duplicate call-to-action links (e.g. `Pide un Glovo [1] Pide un Glovo [1]` → `Pide un Glovo [1]`), which occurred when a newsletter rendered the same CTA as both a button image and a text link.
  - Drop orphaned emoji skin-tone modifiers (U+1F3FB–U+1F3FF) that previously survived as tofu after their base emoji was stripped.

### 🔧 Technical Improvements

- Extracted a shared `rerenderCurrentMessage` helper behind the `M` (markdown) and `:touch-up` toggles, removing duplicated re-render/fetch logic.
- Bounded the rendered-body cache to 256 entries with single-entry eviction to keep long-session memory in check.

## [1.4.0] - 2026-06-08

### ✨ Features

- **Markdown email rendering**: HTML emails now render as clean, Markdown-styled text **by default**, making newsletters and marketing emails far more readable in the terminal. Press `M` (or `:markdown` / `:md`) to toggle between the Markdown view and the original raw/plain view, per message.
  - A cleanup pipeline removes the noise that makes HTML emails unreadable: tracking-pixel images, empty layout tables, and zero-width spacer characters are stripped, and long tracking URLs are collected into a `Links` section at the bottom instead of cluttering the body.
  - Conversion is **pure-Go and in-process** (`html-to-markdown` v2), styled with `glamour` — no external runtime dependency. A proof-of-concept bake-off against Microsoft's markitdown found markitdown produced empty layout tables and charset corruption on real newsletters, so the pure-Go path was chosen.
  - New `rendering` config block: `markdown_default` (default `true`), `glamour_theme` (`dark`/`light`/`notty`/`auto`), and `drop_tracking_images` (default `true`).
  - The previous LLM whitespace touch-up (formerly on `M`) moved to the new `:touch-up` command.
  - Falls back gracefully to the existing plain-text renderer on any conversion error or for plain-text-only emails.

### 🔧 Technical Improvements

- New service-first rendering pipeline in `internal/render/markdown_render.go`; render-mode state managed by `DisplayService`; rendered output cached per message/mode/width.
- Fixed a pre-existing data race on the LLM touch-up flag (`llmTouchUpEnabled` is now an `atomic.Bool`).

## [1.3.0] - 2026-06-07

### ✨ Features

- **Inbox Action Plan**: A new AI-assisted triage panel, opened with `P` (capital P) or `:action-plan` (aliases `:plan`, `:ap`). It groups the **unread** messages already loaded in the current view into a few actionable categories (archive, mark-as-read, trash, label) and lets you dispatch each group with a single keystroke. Categories stream in progressively as each batch is analyzed; press `Esc` to cancel at any time (work done so far stays on screen). Per category, two escape hatches: `:` opens the command palette scoped to that category's messages (virtual bulk selection), and `p` opens the bulk prompt picker scoped to the same set. Override the built-in analyzer with any saved prompt via `:action-plan with-prompt <name-or-id>`.
  - Runs in **fast mode**: it uses only the subject, sender, and snippet already in memory, so it makes **no extra Gmail API calls**.
  - Degrades gracefully when the LLM returns malformed or empty output — one stricter repair retry, then any uncategorized messages fall back to a "Read manually" bucket so nothing is lost.
  - New service-first `InboxAnalyzerService` (configurable batching, JSON parsing with out-of-range guarding, cross-category de-duplication, and category merging across batches; calls the AI service directly).
  - New `inbox_analyzer` configuration block (`batch_size` default 50, `max_batches` default 10) and a configurable `keys.action_plan` shortcut (default `P`). See `docs/KEYBOARD_SHORTCUTS.md`.

### 🛠️ Technical Improvements

- The analyzer's default `batch_size` of 50 is tuned for capable cloud models. Small local models (e.g. a 7B Ollama model) can struggle to follow the JSON schema across 50 messages and may return no categories — lower `inbox_analyzer.batch_size` to 15–20 in that case (documented in `docs/KEYBOARD_SHORTCUTS.md` and `docs/CONFIGURATION.md`).
- Background analysis renders are marshalled onto the UI thread via `QueueUpdateDraw` for reliable progressive painting, the panel's keys are routed past the global shortcut handler (so quick-actions and escape hatches work from the panel's text view), and the streaming-cancel lifecycle is guarded against close/reopen races.

---

## [1.2.4] - 2026-06-03

### 🐛 Bug Fixes

- **Help screen panic on `?` (#6)**: Fixed nil pointer dereference in `generateHelpText` when `config.json` had no `obsidian` section (i.e. fresh installs running on `DefaultConfig`). The four call-sites that dereferenced `Config.Obsidian.Enabled` now go through a defensive `IsObsidianEnabled()` helper, and the helper is covered by table-driven tests including the default-config case that reproduced the crash.
- **Bulk-select checkbox refresh**: Pressing `*` in bulk mode now updates the ☑/☐ glyph on every row immediately, even when the optional message-numbers column is enabled. Replaced the hardcoded "flags is column 0" assumption in `updateBulkSelectionStyling` with a per-row scan that finds the checkbox glyph regardless of layout. Also removed the defensive prepend branch that was contaminating the numbers column with a stray glyph on first use.

### 🛠️ Technical Improvements

- **Lint debt cleanup**: `make pre-commit-check` now passes with 0 issues. Annotated 9 intentional gosec findings with justified `// #nosec` markers (G304 on validated DB paths, G204 on hardcoded per-OS open binaries, G117 on intentional OAuth token persistence, G101 on test fixture paths) and refactored 30+ `strings.Builder.WriteString(fmt.Sprintf(...))` call-sites to the more efficient `fmt.Fprintf(&builder, ...)` form (staticcheck QF1012).
- **Developer docs**: Refreshed `CLAUDE.md` with the current 12-value `GetServices()` signature, the canonical pre-commit verification command, a project layout overview, and the precise `make test` scope.

---

## [1.2.3] - 2025-09-18

### 🐛 Bug Fixes

- **Message Deletion**: Fix stale message content after deleting an item. When pressing `d`, the selection stays in place and the content panel now shows the new selected message. Added selection-change guard in `showMessage`/`showMessageWithoutFocus` and avoided updating `currentMessageID` on preview.
- **Deletion/Selection Races**: Additional safeguards to reduce race conditions when removing rows and re-selecting the next message in the list.
- **Command Parsing**: Proper quoted-string handling for `:command` arguments to avoid split errors.

### 🛠️ Technical Improvements

- **Race-condition Guard**: Validate current selection before painting content when background loading completes, preventing outdated renders.
- **CI Reliability**: Improve GitHub Actions cache configuration to reduce cache-restore failures.

---

## [1.2.2] - 2025-09-12

### 🐛 Bug Fixes

- **Theme System**: Fixed nil pointer crash when using bulk selection (S then *) by completing theme system migration
- **Bulk Selection**: Resolved focus loss issues during bulk operations with improved UX and selection logic
- **Table Synchronization**: Fixed off-by-one errors in row counting and table-to-cache synchronization

### 🛠️ Technical Improvements

- **Error Messaging**: Enhanced Gmail API validation failures with user-friendly error messages and actionable steps
- **UX Enhancement**: Improved single message selection behavior in bulk mode for better user experience
- **Focus Management**: Replaced destructive table rebuilds with targeted styling updates to preserve focus

### 📚 Documentation

- **Setup Guide**: Emphasized Gmail API enablement requirement in README setup instructions

---

## [1.2.1] - 2025-09-10

### 🐛 Bug Fixes

- **Version Display**: Fixed `go install` builds showing incorrect version (1.1.1 instead of 1.2.x)
- **Version Sync**: Synchronized hardcoded version in version.go with VERSION file

### 📚 Technical Notes

This patch release fixes a version display inconsistency where `go install github.com/ajramos/giztui/cmd/giztui@latest` would show v1.1.1 instead of the correct version. Both `make` builds and `go install` builds now consistently display the correct version.

---

## [1.2.0] - 2025-09-09

### ✨ New Features

- **Multi-Account Support**: Complete implementation of database-per-account architecture enabling seamless switching between multiple Gmail accounts
- **Account Picker**: Interactive account selection with number shortcuts (1-9) for quick switching, similar to Links picker UX
- **Hot Account Switching**: Real-time account switching with proper service re-initialization and database context switching
- **Account Commands**: New `:accounts` command with full keyboard shortcut parity for account management
- **Enhanced OAuth Flow**: Account-specific authorization messages with improved user experience

### 🛠️ Technical Improvements

- **Unified Logger Architecture**: Comprehensive account selection logging with centralized logging infrastructure
- **Graceful Credential Fallback**: Multi-level credential fallback system for robust authentication handling
- **Service Initialization**: Improved database-dependent service initialization timing and coordination
- **Cache Management**: Account-aware cache service with proper invalidation during account switching
- **UI Consistency**: Enhanced multi-account UI patterns with proper error handling and validation

### 🐛 Bug Fixes

- **Gmail Client Updates**: Resolved Gmail client not updating properly during account switching
- **Database Connections**: Fixed Obsidian export database connection issues in multi-account scenarios  
- **Service Re-initialization**: Fixed cache and database services not being reinitialized during account switching
- **UI State Management**: Resolved multi-account UI inconsistencies and enhanced account picker display
- **Control Key Shortcuts**: Fixed control key shortcut customization not working properly

---

## [1.1.1] - 2025-09-07

### 🐛 Bug Fixes

- **Version Display Fix**: Fixed version display for `go install` builds showing correct v1.1.1 instead of outdated v1.0.2
- **Help Screen Layout**: Fixed command equivalent section column alignment for better readability
- **Build Method Consistency**: Ensured version fallback values stay synchronized with VERSION file

---

## [1.1.0] - 2025-09-06

### ✨ New Features

- **Obsidian Repopack Integration**: Complete email export functionality with comprehensive context
- **New Keyboard Shortcuts**: `o repack`, `obsidian repack`, `obs repack`
- **New Commands**: `:obsidian repack`, `:obs repack` with full command parity
- **Bulk Repack Support**: Apply repack operations to multiple selected emails
- **Smart Mode Detection**: Handles both count-based and repack operations seamlessly

### 🛠️ Technical Improvements

- **Comprehensive Test Coverage**: 95%+ coverage for all repack functionality
- **Enhanced Linting**: Updated golangci-lint configuration for better code quality
- **Robust Error Handling**: Improved validation and user feedback

---

## [1.0.2] - 2025-09-04

### ✨ New Features

#### Enhanced Version Detection System
- **Smart Version Detection**: No more "unknown" Git commit for `go install` builds
- **Automatic VCS Integration**: Leverage Go 1.18+ runtime/debug.BuildInfo for automatic Git information
- **Build Method Indication**: Clear differentiation between `make`, `go-install`, and `unknown` builds
- **VCS Status Detection**: Show modification status for development builds with uncommitted changes
- **Improved Version Display**: Better formatted version strings with meaningful build information

#### Comprehensive Pre-commit System
- **Enhanced Pre-commit Hooks**: Comprehensive hooks matching CI pipeline requirements exactly
- **Format & Lint Checking**: Automatic code formatting and linting before commits
- **Essential Test Runner**: Quick smoke tests to catch breaking changes early
- **Developer Setup Script**: One-script onboarding for new contributors (`scripts/setup-dev.sh`)

#### Streamlined CI/CD Pipeline
- **Consolidated Workflow**: Single comprehensive CI/CD pipeline replacing separate workflows
- **Multi-platform Testing**: Cross-platform validation (Ubuntu + macOS)
- **Enhanced Security**: Trivy vulnerability scanning and dependency review
- **Better Reporting**: Improved PR comments with detailed CI/CD results

### 🛠️ Developer Experience Improvements

#### New Make Commands
- `make setup-hooks` - Install and configure pre-commit hooks
- `make check-hooks` - Run pre-commit hooks on all files
- `make pre-commit-check` - Run same checks as CI locally
- `make remove-hooks` - Remove pre-commit hooks

#### Enhanced Documentation
- **Development Setup Guide**: Complete contributor onboarding documentation
- **Installation Guide Updates**: Clear explanations of version differences between build methods
- **Build Method Documentation**: Comprehensive explanation of `make build` vs `go install`

### 🔧 Technical Improvements

#### Configuration Enhancements
- **Updated golangci-lint Config**: Modern configuration with version 2 support
- **Improved Linting Rules**: Comprehensive linter setup with proper exclusions
- **Pre-commit Configuration**: Hooks that mirror CI pipeline exactly

#### Build System
- **Version Injection Compatibility**: Maintains full backward compatibility with custom build metadata
- **VCS Detection**: Automatic Git commit, time, and modification status detection
- **Release Process**: Streamlined and validated release workflow

### 📚 Documentation Updates

- Enhanced installation instructions with version information explanations
- New troubleshooting section for version-related issues
- Complete developer setup and contribution guidelines
- Updated architecture documentation for new version detection system

### 🎯 Benefits for Users

- **Better Debugging**: Meaningful version information regardless of installation method
- **Improved Traceability**: Clear build method and Git commit information
- **Enhanced Developer Experience**: Easier setup and contribution process
- **Quality Assurance**: Automated checks prevent common issues from reaching CI/CD

---

## [1.0.1] - 2025-09-04

### 🚀 Performance Improvements

#### Background Preloading System
- **Phase 2.4 Background Preloading**: Implement instant navigation with intelligent message preloading
- **70% Preloading Functionality**: Restore preloading with proper pagination token preservation
- **Preload Control**: Add comprehensive preload off command to disable all preloading features consistently
- **Focus Highlighting**: Implement proper focus highlighting for preload status screen

#### Gmail API Optimization
- **Metadata Optimization**: Achieve 70-80% bandwidth reduction through selective field requests
- **Load More Improvements**: Resolve focus and pagination issues in load more functionality

### ✨ New Features

#### UI/UX Enhancements
- **Smart Recipient Truncation**: Handle long To/Cc fields with intelligent truncation and configuration support
- **Full-Screen Prompt Statistics**: Transform stats display to comprehensive prompt statistics view
- **Keyboard Shortcut Validation**: Add optional keyboard shortcut validation with comprehensive coverage

#### Command System
- **Prompt Stats Command**: New `:prompt stats` command for detailed prompt usage analytics

### 🐛 Bug Fixes

#### Navigation & Controls
- **Navigation Issues**: Correct gg navigation and number command navigation problems
- **Focus Management**: Resolve focus issues in various UI components

#### Theme & Display
- **Emoji Rendering**: Fix status bar emoji rendering issues for better terminal compatibility
- **Save Search Query Dialog**: Improve theme consistency across all dialog elements
- **Advanced Search**: Replace problematic emoji in date field validation

#### Development & Build
- **Clean Build Process**: Remove development artifacts and ensure clean production builds
- **Configuration Integration**: Complete MaxRecipientLines feature with full config integration

### 🔧 Technical Improvements

- **Documentation Updates**: Comprehensive updates for new features and configuration options
- **Build System**: Enhanced cross-platform build process and artifact management
- **Code Quality**: Various code cleanup and optimization improvements

### 📋 Configuration

- **MaxRecipientLines**: New configuration option to control recipient field truncation behavior
- **Preloading Controls**: Enhanced configuration options for background preloading system

## [1.0.0] - 2025-09-02

### 🎉 Initial Stable Release

This marks the first stable release of GizTUI (formerly Gmail TUI), featuring a complete terminal-based Gmail client with AI integration, advanced UI/UX, and powerful productivity features.

### ✨ Core Gmail Features

#### Email Management
- **Full email operations**: Read, compose, reply, reply-all, forward, archive, trash, and restore
- **Advanced search**: Gmail query syntax support with contextual shortcuts (from:current, to:current, subject:current)
- **Enhanced move operations**: Context-aware system folders (Inbox, Trash, Archive, Spam) with regular labels
- **Message threading**: Smart conversation grouping with visual hierarchy and expand/collapse controls
- **Dual view modes**: Toggle between threaded conversations and flat chronological view
- **Bulk operations**: Multi-select messages for batch actions (archive, trash, move, label)
- **VIM-style navigation**: Range operations like `d3d` (delete 3), `a5a` (archive 5), `t2t` (toggle read 2)
- **Undo functionality**: Reverse archive, trash, read/unread, and label operations

#### Message Composition
- **Complete composition UI**: Full-screen modal with proper theming and focus management
- **Advanced recipient handling**: CC/BCC support with proper recipient extraction and exclusion
- **Draft management**: Create, edit, save, and delete drafts with picker UI
- **Auto-save drafts**: Automatic draft saving during composition
- **Message threading**: Proper threading headers and Gmail compatibility
- **Real-time validation**: Email format validation and visual error indicators

#### Labels and Organization
- **Full label management**: Create, delete, apply, and remove labels
- **Contextual labels panel**: Side panel with quick toggle and live refresh
- **Browse all labels**: Full picker with incremental search and ESC navigation
- **Visual label colors**: Each label displayed with unique colors in message lists
- **Enhanced bulk labeling**: Apply labels to multiple selected messages
- **Smart label suggestions**: AI-powered label recommendations

### 🧠 AI and LLM Integration

#### Core AI Features
- **Email summarization**: Generate concise email summaries with streaming support
- **Smart label suggestions**: AI-powered label recommendations based on email content
- **Streaming LLM responses**: Real-time token streaming for immediate feedback
- **Intelligent caching**: SQLite-based caching system for AI results to avoid duplicate processing
- **Multiple LLM providers**: Support for both Ollama (local) and Amazon Bedrock (cloud)

#### Prompt Library System
- **Custom prompt templates**: Predefined and user-created prompts for various use cases
- **Variable substitution**: Auto-complete `{{from}}`, `{{subject}}`, `{{body}}`, `{{date}}`, `{{messages}}`
- **Category organization**: Organize prompts by purpose (Summary, Analysis, Action Items, etc.)
- **Usage tracking**: Monitor prompt usage patterns and effectiveness
- **Split-view interface**: Prompt picker appears as side panel (not full-screen modal)
- **Bulk prompt processing**: Apply prompts to multiple emails simultaneously for consolidated analysis

#### Advanced AI Operations
- **Thread summaries**: Generate conversation overviews with context from all messages
- **Bulk email analysis**: Consolidated insights across multiple selected messages
- **Smart content processing**: Optional LLM touch-up for better email formatting
- **Interruption support**: Cancel any streaming operation instantly with ESC key

### 🔌 Integration Features

#### Slack Integration
- **Multi-channel support**: Configure multiple Slack channels with individual webhooks
- **Bulk forwarding**: Forward multiple emails simultaneously with shared comments
- **Multiple format styles**: Summary (AI-generated), Compact, Full (TUI-processed), Raw
- **Smart variable substitution**: AI prompts support email headers and content variables
- **Progress tracking**: Real-time progress updates for bulk operations
- **TUI content fidelity**: "Full" format shows exactly what you see in the message widget

#### Obsidian Integration
- **Email ingestion**: Send emails directly to Obsidian as Markdown notes
- **Bulk ingestion**: Process multiple selected emails with shared comments
- **Template system**: Single, customizable Markdown template with variable substitution
- **Duplicate prevention**: SQLite-based history tracking prevents re-ingestion
- **Attachment support**: Include email attachments in Obsidian notes
- **Second brain organization**: Organize emails in `00-Inbox` folder for processing

#### Calendar Integration
- **Smart invitation detection**: Automatically detect calendar invitations in emails
- **Enhanced RSVP handling**: Accept, Tentative, or Decline with Google Calendar API integration
- **Meeting details display**: Shows title, organizer, date/time with proper formatting
- **iCalendar parsing**: Handles complex timezone-aware calendar data

### 🎨 Advanced UI/UX

#### Theme System
- **Runtime theme switching**: Change themes instantly without restart (`/theme set <name>`)
- **Multiple built-in themes**: Slate Blue (default), Dracula, Gmail Dark/Light, Custom Example
- **Custom theme support**: User themes in `~/.config/giztui/themes/`
- **Theme preview**: See colors before applying themes
- **Hierarchical color system**: Foundation → Semantic → Interaction → Component overrides

#### Adaptive Layout System
- **Responsive design**: Automatically adapts to terminal size changes
- **Multiple layout types**: Wide (≥120x30), Medium (≥80x25), Narrow (≥60x20), Mobile (<60x20)
- **Smart focus management**: Proper focus cycling with Tab/Shift+Tab
- **Fullscreen mode**: Press 'f' for fullscreen text view
- **Real-time resizing**: Layout updates as you resize terminal

#### Enhanced Navigation
- **VIM-style commands**: `gg` (first message), `G` (last message), `:5` (jump to message 5)
- **Content search**: `/searchterm` with `n`/`N` navigation and highlighting
- **Fast content navigation**: Paragraph jumping (`Ctrl+K/J`), word navigation (`Ctrl+H/L`)
- **Context-aware shortcuts**: Different behaviors when viewing message vs message list
- **Enhanced content navigation**: Fast browsing within message content

#### Advanced Search and Filtering
- **Local filtering**: In-memory filter with `/` including label filters (`label:Personal`)
- **Advanced search form**: Multiple fields with quick options panel
- **Size-based search**: Filter by email size (`>1MB`, `<500KB`)
- **Date range filtering**: Flexible date searches with `after:`/`before:` operators
- **Search highlighting**: Visual highlighting of search terms in results

### 🔧 Productivity Features

#### Link and Attachment Management
- **Smart link extraction**: Automatically extract links from HTML and plain text emails
- **Link picker**: Press `L` for quick link access with search and categorization
- **Cross-platform opening**: Native browser opening on macOS, Linux, Windows
- **Attachment picker**: Press `A` for attachment management with download and preview
- **Smart file handling**: Automatic filename conflict resolution and cross-platform downloads

#### Command System
- **Command parity**: Every keyboard shortcut has equivalent command (`:archive`, `:trash`, etc.)
- **Auto-completion**: Tab completion for all commands with live suggestions
- **Context awareness**: Commands automatically detect bulk mode and act appropriately
- **Command history**: Navigation through previous commands
- **k9s-style interface**: Professional command bar with bordered panel

#### Bulk Operations
- **Advanced selection**: `v`/`b`/`space` for bulk mode, `*` for select all
- **Range operations**: VIM-style `d3d`, `a5a`, `t2t` for efficient batch actions
- **Bulk AI processing**: Apply prompts to multiple emails for consolidated analysis
- **Progress indicators**: Real-time feedback for long-running bulk operations

### 🏗️ Architecture and Development

#### Service-Oriented Architecture
- **Clean separation**: UI components handle only presentation, services handle business logic
- **Service layer**: EmailService, AIService, LabelService, CacheService, etc.
- **Centralized error handling**: Consistent user feedback with ErrorHandler
- **Thread-safe operations**: Mutex-protected accessor methods for app state
- **Dependency injection**: Services automatically initialized and injected

#### Database and Caching
- **SQLite integration**: Embedded database for AI summaries, prompts, and history
- **Per-account separation**: Isolated databases by email account
- **Smart caching**: Cache AI results, prompt responses, and Obsidian history
- **Performance optimization**: Proper indexing and query optimization

#### Configuration System
- **Unified configuration**: Single `config.json` with hierarchical organization
- **Template file support**: External Markdown files for AI/Slack/Obsidian templates
- **Environment variable support**: Override paths via environment variables
- **Smart path resolution**: Relative paths resolved relative to config directory

### 🧪 Testing and Quality

#### Comprehensive Testing Framework
- **Unit tests**: Service layer testing with mocks
- **Integration tests**: Full workflow testing
- **TUI component tests**: Terminal UI component validation
- **Performance tests**: Load testing for bulk operations
- **Mock generation**: Automated mock generation with mockery

#### CI/CD Pipeline
- **Automated testing**: GitHub Actions with comprehensive test suite
- **Multi-platform builds**: Linux, macOS (Intel/ARM), Windows
- **Code quality**: golangci-lint, go vet, format checking
- **Security scanning**: Vulnerability scanning with govulncheck

### 📋 Configuration Migration
- **Directory migration**: Automatic migration from `~/.config/gmail-tui/` to `~/.config/giztui/`
- **Backward compatibility**: Seamless upgrade path for existing users
- **Environment variables**: Updated environment variable names for consistency

### 🎯 User Experience Improvements
- **Welcome screen**: Structured startup with account info and quick actions
- **Status bar**: Rich status information with operation feedback
- **Error handling**: User-friendly error messages and recovery options
- **Loading indicators**: Progress feedback for long operations
- **Keyboard shortcuts**: Fully customizable keyboard shortcuts via configuration

### 🔨 Developer Experience
- **Clean codebase**: Well-organized project structure with clear separation of concerns
- **Comprehensive documentation**: Architecture docs, theming guide, development guide
- **Build system**: Makefile with development, testing, and release targets
- **Version management**: Proper semantic versioning with build-time injection

---

## Development Notes

### Migration from Gmail TUI
This release represents the stable v1.0.0 of the project formerly known as "Gmail TUI". All references to the old naming have been updated to "GizTUI" for consistency.

### Supported Platforms
- Linux (amd64)
- macOS (Intel and Apple Silicon)
- Windows (amd64)

### Requirements
- Go 1.23.0+
- Gmail API credentials
- Terminal with 256-color support
- Optional: Ollama for local AI features
- Optional: AWS credentials for Bedrock AI

### Breaking Changes
- Configuration directory changed from `~/.config/gmail-tui/` to `~/.config/giztui/`
- Binary name changed from `gmail-tui` to `giztui`
- Module path changed from `github.com/ajramos/gmail-tui` to `github.com/ajramos/giztui`

### Credits
GizTUI is inspired by excellent terminal applications like `k9s`, `neomutt`, and `alpine`, bringing modern AI capabilities and productivity features to terminal-based email management.

[1.0.0]: https://github.com/ajramos/giztui/releases/tag/v1.0.0