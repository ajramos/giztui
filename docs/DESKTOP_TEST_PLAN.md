# 🧪 GizTUI Desktop — Test Plan

A manual test plan covering **every feature** of the desktop client. Work through
it top to bottom the first time; afterwards use it as a regression checklist.

- ⌨️ = keyboard shortcut · `:cmd` = command palette · 🖱️ = mouse/UI
- **Exp:** = expected result

Tick each row as you go. Anything that doesn't match **Exp** is a bug — note the
message subject/account and what you saw.

---

## 0. Setup & preconditions

| # | Check | Exp |
|---|-------|-----|
| 0.1 | The TUI already works with your account (`~/.config/giztui/`) | Config + OAuth token exist |
| 0.2 | `cd desktop && wails build` then open `build/bin/GizTUI Desktop.app` | App launches, no blank window |
| 0.3 | For AI tests: an LLM provider is configured (`llm.enabled`) | AI actions are visible/enabled |
| 0.4 | For RSVP: your token has the `calendar.events` scope (requested at sign-in; older tokens re-authorize once) | RSVP picker (`V`) works on invites |

> If a whole feature area is greyed out/absent, it's because the matching config
> is off (AI, Obsidian, Slack, threading, saved queries, calendar). That's
> expected — enable it in `config.json` to test it.

---

## 1. Launch & app shell

| # | Action | Exp |
|---|--------|-----|
| 1.0 | First launch on a machine with **no token** yet | System browser opens Google sign-in; app shows a sign-in screen ("Open sign-in in browser" / "Copy link"); after approving, it continues automatically |
| 1.1 | Launch the app (token already present) | Slim topbar, inbox list on the left, reading pane on the right |
| 1.2 | Observe the topbar | Icon-only buttons: search (🔍), account, compose, drafts, select, ★, toolbar toggle, auto-refresh (🕐), help, refresh. Hover shows tooltips |
| 1.3 | Without clicking anything, press ⌨️ `j` | The list cursor moves — shortcuts work on launch (no click needed) |
| 1.4 | ⌨️ `?` (or 🖱️ help icon) | Keyboard-shortcut overlay opens; `?`/Esc closes it |

---

## 2. Inbox list & navigation

| # | Action | Exp |
|---|--------|-----|
| 2.1 | ⌨️ `j` / `k` (or ↓/↑) | Cursor moves; the message **previews** in the reader **without** being marked read |
| 2.2 | ⌨️ `Enter` (or 🖱️ click a row) | Message opens and **is** marked read (unread dot clears) |
| 2.3 | ⌨️ `gg` | Jumps to the top of the list (press `g` twice within the vim timeout) |
| 2.4 | ⌨️ `G` | Jumps to the bottom of the loaded list |
| 2.5 | ⌨️ `N` (or `:refresh` bottom) | Loads the next page of messages (appended) |
| 2.6 | ⌨️ `R` / `:refresh` | Reloads the current view |
| 2.7 | ⌨️ `Esc` on an open message | Closes the reader (list stays) |
| 2.8 | Observe rows | Sender, subject, snippet, relative date, unread dot, label chips |

---

## 3. Reading a message

| # | Action | Exp |
|---|--------|-----|
| 3.1 | Open a plain-text email | Body renders as text; sender name + address, To, date shown |
| 3.2 | Open an HTML email | Rich HTML renders (Shadow DOM, sanitized); remote images blocked by default; inline `cid:` images show automatically |
| 3.3 | ⌨️ `:images` | Remote images load (only after opt-in), fetched via the backend |
| 3.4 | ⌨️ `M` (HTML/text toggle in toolbar) | Switches between HTML and plain-text view |
| 3.5 | With an HTML email, 🖱️ click **inside the body**, then press ⌨️ `a` | Archive still fires — keystrokes work while the email has focus |
| 3.6 | 🖱️ click a **link** in an HTML email | Opens in your **system browser** (not inside the app) |
| 3.7 | ⌨️ `h` / `:headers` | Header detail expands (cc, thread id, message id); toggles off |
| 3.8 | ⌨️ `L` / `:links` | Links picker lists every URL; picking one opens it in the browser |
| 3.9 | Open a message with an attachment | Attachment chips show under the toolbar (name + size) |
| 3.10 | 🖱️ click an attachment chip | Downloads to the configured download dir; toast shows the path |

### 3a. Content search (in-message)

| # | Action | Exp |
|---|--------|-----|
| 3a.1 | With a message open, ⌨️ `/` | A "Find in message" bar opens over the body (view switches to text) |
| 3a.2 | Type a word present in the body | Matches highlight; the active match is a different color; counter shows `n/total` |
| 3a.3 | ⌨️ `Enter` / `Shift+Enter` (or ↓/↑ in the bar) | Jumps to next/previous match and scrolls to it |
| 3a.4 | ⌨️ `Esc` (or ✕) | Closes the search bar |

---

## 4. Triage & undo

| # | Action | Exp |
|---|--------|-----|
| 4.1 | ⌨️ `a` / `:archive` on the open message | Message leaves the list; toast "Archived"; the reader closes (no stale email) |
| 4.2 | ⌨️ `U` / `:undo` (or 🖱️ topbar ↶) | The archived message returns to its old position; toast "Undone: archive" |
| 4.3 | ⌨️ `d` / `:trash` | Message leaves the list; toast "Moved to trash" |
| 4.4 | ⌨️ `U` | Message is restored (untrashed) to the list |
| 4.5 | ⌨️ `t` / `:read` `:markunread` | Toggles read/unread; the row's unread dot updates |
| 4.6 | ⌨️ `U` after a read/unread toggle | The read state reverts |
| 4.7 | ⌨️ `U` with nothing to undo | Toast "Nothing to undo" |

---

## 5. Move & labels

| # | Action | Exp |
|---|--------|-----|
| 5.1 | ⌨️ `l` / `:labels` | Labels picker opens; current labels are checked |
| 5.2 | Toggle a label in the picker | Label is applied/removed on the message |
| 5.3 | ⌨️ `m` / `:move` (or `⋯` → Move to…) | Keyboard move picker opens: filter, ↑↓, Enter |
| 5.4 | Type/pick a folder, Enter | Message gets the label **and** is archived (leaves the inbox); cursor advances to the next row; toast "Moved to …" |
| 5.5 | `:move Receipts` (with arg) | Moves directly without the picker |
| 5.6 | In bulk mode, select 2+, ⌨️ `m` (or the bulk-bar Move button) | "Move N to folder" picker; Enter moves all, clears selection, advances cursor |

---

## 6. Compose, reply, forward, drafts

| # | Action | Exp |
|---|--------|-----|
| 6.1 | ⌨️ `c` / `:compose` (or topbar ✎) | Compose window opens (To/Subject/Body) |
| 6.2 | Fill + send | Toast "Message sent"; window closes |
| 6.3 | ⌨️ `r` / `:reply` on a message | Compose opens in reply mode, To prefilled with the sender |
| 6.4 | ⌨️ `E` / `:replyall` | Reply mode with original To/Cc added as Cc |
| 6.5 | ⌨️ `f` / `:forward` | New message, subject `Fwd: …` |
| 6.6 | In compose, "Save draft" | Toast; the draft appears under Drafts |
| 6.7 | ⌨️ `D` / `:drafts` (or topbar) | Drafts list opens |
| 6.8 | Open a draft, edit, save | Draft updates |
| 6.9 | Send a draft | It sends and is removed from Drafts |
| 6.10 | Delete a draft | It disappears |

---

## 7. Search

| # | Action | Exp |
|---|--------|-----|
| 7.1 | ⌨️ `s` (or 🖱️ the search field) then a Gmail query, Enter | List filters to Gmail results (`from:`, `has:attachment`, etc. all work) |
| 7.2 | 🖱️ the ✕ clear button | Returns to the inbox |
| 7.3 | `:search from:x@y.com has:attachment` | Same as typing it |
| 7.4 | On an open message, ⌨️ `F` / `:from` | Searches `from:<this sender>` |
| 7.5 | ⌨️ `T` / `:to` | Searches `to:<this recipient>` |
| 7.6 | ⌨️ `S` / `:subject` | Searches the (cleaned) subject |
| 7.7 | 🖱️ the sliders icon / `:advanced` | Advanced builder opens (from/to/subject/has-attachment/unread/after/before) with a **live query preview**; Search runs it |
| 7.8 | 🖱️ the search-mode toggle / `:local` | Switches to **Local filter**: typing narrows the **loaded** list instantly (no network); toggle back to Gmail |
| 7.9 | ⌨️ `Q` / `:queries` | Saved searches picker opens; picking one runs it |
| 7.10 | Run a search, then ⌨️ `Z` / `:savequery` | Save-query modal; the query is saved and appears under `Q` |

---

## 8. Bulk / selection mode

| # | Action | Exp |
|---|--------|-----|
| 8.1 | ⌨️ `Space` on the inbox (not in bulk) | Enters bulk mode **and** selects the current row |
| 8.2 | ⌨️ `Space` again on other rows | Toggles selection and advances the cursor |
| 8.3 | ⌨️ `v` | Toggles bulk mode on/off |
| 8.4 | ⌨️ `*` | Selects all loaded messages |
| 8.5 | Observe the bulk bar | "N selected" + **the same icon buttons** as the reader (archive/trash/read/unread/label │ select-all/done) |
| 8.6 | 🖱️ bulk Archive/Trash | A progress bar shows while running; the selected rows leave; toast "Archived N" |
| 8.7 | ⌨️ `U` after a bulk archive/trash | All affected rows are restored to their positions |
| 8.8 | If the open message was in the bulk set | The reader closes (no stale email) |
| 8.9 | 🖱️ bulk Label… | Labels picker applies to all selected |
| 8.10 | 🖱️ Done / ⌨️ `Esc` | Exits bulk mode |

---

## 9. AI on a single message

> Requires an LLM provider. If AI is off, these are hidden.

| # | Action | Exp |
|---|--------|-----|
| 9.1 | ⌨️ `y` / `:summarize` (or `⋯` → Summarize) | Summary panel streams tokens live, then settles |
| 9.2 | 🖱️ "regenerate" on the summary | Re-runs bypassing the cache |
| 9.3 | ⌨️ `p` / `:prompt` (or `⋯` → Apply a prompt) | Prompts picker opens; picking one streams the result into a panel |
| 9.4 | ⌨️ `o` / `:suggest` (or `⋯` → Suggest labels) | AI suggests labels; clicking a chip applies it |
| 9.5 | `⋯` → Draft reply (AI) / `:draft` | The LLM drafts a reply; compose opens **prefilled** to edit before sending |
| 9.6 | `⋯` → Touch-up (AI) / `:touch-up` | The body is reformatted; a "Reformatted by AI" bar shows with "show original" to revert |

### 9a. Prompts — inline CRUD (same model as saved searches)

| # | Action | Exp |
|---|--------|-----|
| 9a.1 | `:prompt` (or `:prompts`) | Prompts picker lists your prompts; Enter applies the highlighted one |
| 9a.2 | ✎ / `Shift+E` on a prompt | Edit dialog with Name/Description/Category/Text (`{{body}}`) opens over the picker |
| 9a.3 | "✦ Refine with AI" in the edit dialog | The LLM improves the prompt text in place |
| 9a.4 | Save | The picker reflects the new/edited prompt in place; it's applyable |
| 9a.5 | 🗑 / `Shift+Del` on a prompt | It's deleted from the list |
| 9a.6 | ＋ New prompt (footer) | Same edit dialog with empty fields; Save adds a row |
| 9a.7 | `Escape` in the edit dialog | Closes only the dialog; the picker stays open |

---

## 10. Inbox action plan & rules

> Requires an LLM provider.

| # | Action | Exp |
|---|--------|-----|
| 10.1 | ⌨️ `P` / `:plan` | Deterministic-rules first pass, then AI; categories show priority, action, count; rule-matched ones tagged "rule" |
| 10.2 | ⌨️ →/`Space` on a category | Expands to list its emails; ↓ descends into them |
| 10.3 | ⌨️ `Enter` on an email | Quick-view opens (peek at the body); `Esc` returns to the plan; "Open in reader" jumps to the full message |
| 10.4 | ⌨️ `Space` on an email | Toggles selection (deselected = unchecked + struck); category apply/move act only on the still-selected subset |
| 10.5 | ⌨️ `Enter` on a category | Applies that category's action to its **selected** messages (archive/mark-read; a `label` bucket does **move to folder** = label + archive) |
| 10.5b | A `label` bucket shows two buttons | **Move to "X"** (label + archive, leaves inbox — Enter / primary) and **Label "X"** (label only — `l` key) |
| 10.5c | A `prompt` bucket (from a rule) shows **Run prompt** | Enter / the button runs the rule's saved prompt over the selected emails; result streams into a modal |
| 10.6 | ⌨️ `m` on an email | Move chooser (1-9 / ↑↓); reassigns that email to another bucket (in-memory until you apply) |
| 10.7 | ⌨️ `m` on a category | Move chooser; reassigns the still-selected emails of the bucket; an emptied source category is pruned |
| 10.7b | The **Read manually** bucket (review badge, last) | Expand/select/peek like any category; `m` recategorizes its emails into a real category; no apply button |
| 10.8 | 🖱️ **Apply all** | Runs every category's action in one go |
| 10.9 | ⌨️ `p` | Shows the effective analyzer prompt (rules block + base) |
| 10.10 | ⌨️ `r` | Analyzer rules: add a natural-language rule; re-run `:plan` respects it |
| 10.11 | `:rules` | Deterministic rules manager: add/edit/delete, Gmail sync, import filters |

---

## 11. Threading / conversation

> Requires threading enabled.

| # | Action | Exp |
|---|--------|-----|
| 11.1 | Open a message that's part of a thread, 🖱️ the conversation icon (or `:threads`) | The full conversation renders (all messages) |
| 11.2 | 🖱️ a message header in the conversation | It collapses to a one-line header (sender · snippet · date); click again to expand |
| 11.3 | 🖱️ Collapse all / Expand all | All messages collapse/expand |
| 11.4 | 🖱️ ✦ Summarize (conversation) | Streams a summary of the whole thread |

---

## 12. Integrations

| # | Action | Exp |
|---|--------|-----|
| 12.1 | ⌨️ `O` / `:gmail` (or `⋯` → Open in Gmail) | Opens the message in Gmail in your system browser |
| 12.2 | `⋯` → Send to Obsidian / `:obsidian` *(if enabled)* | Saves to your vault; toast with the note path |
| 12.3 | `⋯` → Forward to Slack / `:slack` *(if enabled)* | Forwards to the configured channel |
| 12.4 | `⋯` → Save to file / ⌨️ `w` / `:save` | Saves a `.txt`; toast with the path |
| 12.5 | `⋯` → Save raw (.eml) / `:save-raw` | Saves the full `.eml`; toast with the path |

---

## 13. RSVP (calendar invites)

> Requires the `calendar.events` scope on your token.

| # | Action | Exp |
|---|--------|-----|
| 13.1 | Open a message that **is a calendar invite** | An RSVP bar shows the event summary + start time |
| 13.2 | ⌨️ `V` | Keyboard RSVP picker opens (Accept / Tentative / Decline, 1-3) |
| 13.3 | Pick **Accept** / `:accept` | Toast "RSVP: accepted"; your status updates in Google Calendar |
| 13.4 | `:tentative` / `:decline` | Status set to tentative / declined |
| 13.5 | Open a non-invite message | No RSVP bar |
| 13.6 | With a token lacking the scope | Actionable error prompting re-authorization (not a raw 403) |

---

## 14. Auto-refresh

| # | Action | Exp |
|---|--------|-----|
| 14.1 | 🖱️ the topbar 🕐 (or `:autorefresh`) | Toggles auto-refresh; toast "Auto-refresh on/off"; the icon highlights when on |
| 14.2 | Leave it on and send yourself a new email | Within the interval, the new message is prepended with a "1 new message" toast |
| 14.3 | Turn it on, then start a search | Auto-refresh stays quiet during searches/drafts view |
| 14.4 | Restart the app | The on/off choice is remembered |

---

## 15. Themes

| # | Action | Exp |
|---|--------|-----|
| 15.1 | Launch the app | Your configured theme (`theme.current`) is applied |
| 15.2 | ⌨️ `H` / `:theme` | Theme picker lists available themes; current is checked |
| 15.3 | Pick a theme (or `:theme dracula`) | Colors change **live** across the whole app; toast with the theme name |

---

## 16. UI: optional toolbar & consistency

| # | Action | Exp |
|---|--------|-----|
| 16.1 | 🖱️ the topbar toolbar toggle (▤) / `:toolbar` | The reader's action toolbar hides/shows; choice persists across restarts |
| 16.2 | With the toolbar hidden, use ⌨️ shortcuts / `⋯`… wait, the toolbar is the reader bar | Everything still works via keyboard/commands (keyboard-first) |
| 16.3 | Compare the topbar, reader toolbar, and bulk bar | All three use the **same** icon-button style |
| 16.4 | Resize the window narrow | The reader toolbar stays compact (secondary actions in `⋯`, never wraps endlessly) |

---

## 17. Multi-account

> Requires 2+ configured accounts.

| # | Action | Exp |
|---|--------|-----|
| 17.1 | 🖱️ the account dropdown in the topbar | Lists configured accounts; active is marked |
| 17.2 | Switch account | The whole stack rebuilds; inbox, labels, AI, drafts reflect the new account |

---

## 18. Stats, config & cache

| # | Action | Exp |
|---|--------|-----|
| 18.1 | `:stats` (or `:usage`) | AI usage panel: total runs, unique prompts, per-prompt ranking |
| 18.2 | `:config` (or `:cfg`) | Read-only config: account, config path, LLM, theme, downloads, Obsidian/Slack/auto-refresh |
| 18.3 | `:cache` (or the button in `:config`) | Toast "Caches cleared"; next summary/prompt regenerates instead of using cache |

---

## 19. Command palette & help parity

| # | Action | Exp |
|---|--------|-----|
| 19.1 | ⌨️ `:` | Command bar opens with autocomplete over all commands |
| 19.2 | Type a partial command | Suggestions filter; Enter runs the highlighted one |
| 19.3 | An unknown command | Toast "Unknown command: …" |
| 19.4 | ⌨️ `?` | Every key/command in this plan is listed on the help overlay |
| 19.5 | Custom keybindings in your `config.json` | The desktop uses **your** keys, not just defaults (e.g. remap `archive` and verify) |

---

## Known non-goals (not ported, by design)

These TUI features are intentionally **not** in the desktop client because they
don't map well to a GUI. Their absence is expected, not a bug:

- **TTS** (read aloud) — needs a native voice engine; low value with the email on screen.
- **Vim ranges** (`d3d` = trash 3) — a terminal idiom; click/`Space` selection replaces it.
- **Thread-grouped list navigation** (next/prev thread in the list) — the list is
  flat; the full conversation view (expand/collapse) covers the reading need.

---

## Reporting a failure

For anything that fails **Exp**, note:

1. The test number (e.g. `4.2`).
2. Account + message subject (or "any message").
3. What you saw vs. what was expected.
4. Whether an error toast appeared (and its text).
