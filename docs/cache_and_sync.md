# Cache & Incremental Sync (SQLite + Gmail History API)

This document explains how the local cache works and how incremental synchronization is performed.

## Overview

- Embedded SQLite database per account (`~/.config/gmail-tui/cache/gmail-<account>.sqlite3`).
- Tables:
  - `ai_summaries(message_id, account_email, summary, updated_at)`
  - `messages_meta(id, thread_id, snippet, internal_date, label_ids_json, history_id, updated_at, last_accessed)`
  - `messages_body(message_id, plain_text, html, updated_at)`
  - `labels(id, name)`
  - `sync_state(key, value)` → `last_history_id` baseline

## Startup Behavior

1. Load recent `messages_meta` from cache and paint the list immediately.
2. Fetch latest messages from Gmail in background and repaint.
3. Prefetch `messages_body` for the top ~15 messages.

## Incremental Sync (History API)

- Baseline `last_history_id` is initialized on first run (no changes applied).
- On subsequent runs, we request history since the baseline and update:
  - Visible rows (labels/unread indicators)
  - Disk cache (`messages_meta`)
  - Then set `last_history_id` to the latest observed

### Status Indicators

- `🔄 Syncing…` while fetching
- `✅ Synced N` when changes were applied to visible rows
- `✅ Up to date` when no changes

### Manual Commands

- `:sync` → force incremental sync now
- `:cache stats` → show table counts + `last_history_id`
- `:cache clear-all|clear-summaries|clear-messages|clear-sync`

## AI Summaries Cache

- Summaries are saved after generation and reused across sessions.
- Force regeneration with `Y` or `:summary refresh`.

## Troubleshooting

- I don't see `Syncing…`
  - On first run only the baseline is initialized. Execute `:sync` after you have new changes in Gmail.
  - Ensure your token has Gmail scopes (read/modify). Re-auth if needed.

- `:cache stats` shows zeros
  - The app may not have cached yet. Navigate through messages or wait for prefetch.
  - Verify the DB path exists under `~/.config/gmail-tui/cache/`.

- Summary pane hangs after clearing cache
  - Make sure streaming is enabled and Ollama is reachable if using streaming.
  - Try `:summary refresh` (or press `Y`).

- High verbosity logs
  - Current builds include extra logs for diagnostics. These will be reduced later behind a debug flag.

## Notes

- All UI updates occur on the main thread using `QueueUpdateDraw`.
- Network operations never run inside input handlers.
- For performance, heavy processing (e.g., markdown rendering) is done off-thread and applied once.


