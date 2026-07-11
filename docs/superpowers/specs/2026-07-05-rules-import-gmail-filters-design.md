# Rules manager: automatic Gmail filter import — Design

Date: 2026-07-05 · Status: approved by user (chat) · Branch: feat/deterministic-rules

## Goal

`:rules` must be the complete picture of everything acting on the inbox. Gmail
filters created outside the app are invisible today; the manager should surface
them automatically — no separate `:rules import` command (user explicitly
rejected a manual command).

## Behavior

1. Opening `:rules` shows local rules instantly, then fetches Gmail filters in a
   background goroutine (one-shot per open).
2. Every Gmail filter not yet linked to a rule is handled one of three ways:
   - **Adopt**: an existing unmirrored rule with the same query + action + label
     gets the filter's ID stamped (`gmail_filter_id`) — it becomes ☁️ without
     creating a duplicate.
   - **Import**: translatable filters become new rules (created via the store,
     skipping the live query validation — the query already lives in Gmail),
     immediately linked (☁️ from birth).
   - **Gmail-only**: filters the rule model can't represent are appended to the
     list as read-only rows (`☁️ <summary>  (Gmail only)`); Enter/'d' on them
     shows an info message pointing at Gmail. They are re-fetched live on each
     open, never stored.
3. A status message reports "Imported N Gmail filter(s)" when N+adopted > 0.
   Fetch failure (offline, missing `gmail.settings.basic` scope) → one warning;
   the panel keeps working on local rules.
4. Deleting an imported (☁️) rule deletes the real Gmail filter — existing
   `DeleteRule` semantics, explicitly approved by the user for imported filters.

## Translation table (filter → rule)

Criteria → query (joined with spaces):
- `from`/`to`/`subject` → `from:(…)` / `to:(…)` / `subject:(…)`
- `query` → verbatim; `negatedQuery` → `-(…)`
- `hasAttachment` → `has:attachment`
- `size`/`sizeComparison` set, or `excludeChats` → **Gmail-only** (reason: size/chat criteria)

Action → rule action (exact match only; anything else → Gmail-only with reason):
- remove INBOX → `archive` · remove UNREAD → `mark_read`
- add TRASH → `trash` · add one user label → `label` (ID resolved to name)
- forwarding, multiple labels, combos (e.g. label + skip inbox) → **Gmail-only**
  ("does several things at once" / "forwards mail") — importing a lossy subset
  would silently rewrite the filter on the next edit/sync.

## Components

- `internal/gmail/filters.go` — `ListFilters()` (Users.Settings.Filters.List).
- `internal/services/deterministic_rules_import.go` — translation helpers +
  `ImportGmailFilters(ctx) (*GmailImportResult, error)`; `GmailFilterAPI` gains
  `ListFilters`.
- `internal/services/interfaces.go` — `GmailImportResult{Imported, Adopted, Unsupported []GmailOnlyFilter}`,
  `GmailOnlyFilter{ID, Description, Reason}`; interface method.
- `internal/tui/rules_manager.go` — background import on open, Gmail-only rows,
  index guards on Enter/'d', post-import list refresh.
- Help (`app.go` `:rules` entry) — note that Gmail filters appear automatically.

No config option: the whole point is completeness by default. (Revisit only if
someone asks to opt out.)
