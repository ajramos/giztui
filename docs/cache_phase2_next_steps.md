# Intelligent Cache – Phase 2: Next Steps

This document outlines the remaining work and proposed improvements for the next phase of the caching and sync system.

## Goals
- Improve robustness and performance of local cache and incremental sync
- Prepare for offline-friendly message operations with later reconciliation
- Reduce API calls and avoid duplicated background work

## Scope (Phase 2)
- Keep Inbox as source of truth; add background periodic sync and better error handling
- Add cache maintenance policies and operational commands
- Do not implement offline queue yet (planned for a later phase)

## Work Items

### 1) Cache maintenance policy
- Size/age limits for `messages_body` and `messages_meta` (e.g., keep last N days or newest M messages)
- Scheduled `VACUUM`/`ANALYZE` (manual `:cache vacuum`, optional auto after threshold)
- Touch `last_accessed` on reads to support LRU-style eviction in the future

### 2) Incremental sync improvements
- Background sync ticker (e.g., every 2–5 minutes, jittered) with discreet status
- Robust error handling:
  - History API: handle “historyId not found” by re-initializing baseline with a clear status message and retry
  - Network/5xx: retries with backoff; 401/403: surface auth issue and suggestion to re-consent
- Cap pages/time per sync pass to keep UI responsive

### 3) De-duplication guards
- Prevent concurrent work on the same message ID (content fetch, summary generation)
- Simple in-memory maps with timeouts to gate re-entrancy

### 4) Labels and metadata freshness
- Proactively refresh and cache `labels` table periodically
- Ensure list chips render with latest label names

### 5) Configuration & commands
- Config toggles: `prefetch_top_n`, sync interval, max pages per sync, limits for cache size/age
- New maintenance commands:
  - `:cache vacuum` — run `VACUUM` (and optionally `ANALYZE`)
  - `:cache resync` — clear `last_history_id` and reinitialize baseline (no changes applied)

### 6) UX polish
- Help screen: short section on cache/sync commands (done in Phase 1; keep updated)
- Optional debug indicator for cache hits/misses (hidden behind a debug flag)

### 7) Testing
- Unit tests for `internal/cache.Store` (migrations, CRUD, Stats, Clear*)
- Unit/smoke tests for `reloadMessages` + `syncIncremental` with a fake Gmail client (cover baseline init, updates, no-op)
- LLM summary timeout/stream paths are already improved; add fast-failing tests

### 8) Documentation
- Expand the Cache & Sync guide with maintenance policy, new commands, and troubleshooting for history baseline resets
- Keep README commands list in sync

## Out-of-scope for Phase 2 (planned later)
- Offline operations queue (mark read, archive, move, apply/remove label) with reconciliation and conflict handling
- Attachments cache and export utilities

## Risks & Mitigations
- Over-aggressive sync causing rate limits → cap pages, add intervals, exponential backoff
- Baseline invalidation loops → detect and log once, reinitialize baseline, show a friendly message
- DB bloat → enforce limits + provide maintenance commands

## Acceptance Criteria
- Periodic sync runs without blocking UI and updates the list consistently
- Cache policy prevents unbounded growth
- Commands `:cache vacuum` and `:cache resync` work and are documented
- Tests cover main cache and sync flows


