# 📊 Telemetry — event model & tracking plan

GizTUI's usage analytics are **privacy-first, local-only, and opt-in**. This
document is the **tracking plan**: the canonical catalog of every event the app
emits, the data model, the privacy invariants, and the rules for adding events.

> Keep this file in sync with the code. Adding, renaming, or removing a
> telemetry event **must** update the [tracking plan table](#-tracking-plan)
> below. A telemetry event that isn't in this doc is a bug.

## 🔒 Privacy invariants (non-negotiable)

1. **Local-only.** Events are written to the account's local SQLite DB and are
   **never uploaded** anywhere. There is no network path.
2. **Opt-in.** Capture is off by default (`telemetry.enabled: false`) and only
   starts when the user turns it on and restarts.
3. **No content, ever.** Only short, bounded identifiers are stored (a command
   name, a key, an error category). **Never** store command arguments, search
   queries, subjects, senders, bodies, labels, or any message content.
4. **Bounded cardinality.** `name` values are a small, closed vocabulary
   (command names, single keys, fixed categories) — not free text.
5. **Reversible.** `:stats reset` wipes all events; retention prunes old ones.

If a proposed event can't satisfy all five, it doesn't ship.

## ⚙️ Config

```json
"telemetry": {
  "enabled": false,        // opt-in; takes effect on restart
  "retention_days": 90     // events older than this are pruned on startup (<=0 disables pruning)
}
```

## 🗄️ Data model

One flat table, one row per event (`internal/db/telemetry_store.go`, migrations
v11 + v12):

```sql
CREATE TABLE telemetry_events (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  ts            INTEGER NOT NULL,           -- unix seconds
  account_email TEXT NOT NULL DEFAULT '',   -- informational (DB is already per-account)
  kind          TEXT NOT NULL,              -- event class (see tracking plan)
  name          TEXT NOT NULL,              -- bounded identifier within the kind
  ok            INTEGER NOT NULL DEFAULT 1, -- 1 = ok, 0 = error/failure
  duration_ms   INTEGER NOT NULL DEFAULT 0  -- wall-clock ms for `action` events (0 otherwise)
);
```

```go
type TelemetryEvent struct {
    TS           int64
    AccountEmail string
    Kind         string
    Name         string
    OK           bool
    DurationMs   int64
}
```

Capture is async and best-effort: `TelemetryService.RecordEvent` /
`RecordAction` (in `internal/services/telemetry_service.go`) enqueue onto a
buffered channel that a single background goroutine flushes to SQLite in batches;
it is a no-op when disabled and never blocks the UI. Aggregation for the `:stats`
dashboard is three queries: `TopByKind(kind, since, limit)`, `Totals(since)`, and
`ActionStats(since, limit)` (per-action count, failures, and average
`duration_ms`).

## 📋 Tracking plan

Every event currently emitted. **This table is the source of truth.**

| `kind` | `name` values | `ok` | Emitted where | Notes |
|--------|---------------|------|---------------|-------|
| `command` | the command word only (`archive`, `search`, `star`, `labels`, `chat`, …) | always `1` | `executeCommand` (`internal/tui/commands.go`) | Recorded **before** dispatch — measures invocation, not success. **Args are never captured.** |
| `shortcut` | the key rune pressed (`a`, `d`, `y`, `t`, `*`, …) | always `1` | `handleConfigurableKey` call site (`internal/tui/keys.go`) | Single-rune keys only; `ctrl+…` combos aren't captured yet. The command-bar trigger (`command_mode`, default `:`) is **excluded** — the command it opens is captured on its own, so recording the key too would double-count. |
| `error` | `ui` | `0` | `ErrorHandler.ShowMessage` at `LogLevelError` (`internal/tui/error_handler.go`) | Counts error-level user messages. No message text stored. |
| `action` | the action word only (`archive`, `trash`, `summarize`) | outcome: `1` success, `0` failure | at each action's call site via `a.recordAction(name, start, err)` (`internal/tui/app.go`) | Also records `duration_ms` (wall-clock). Measures **outcome + timing** of high-value actions, unlike `command`/`shortcut` which only measure invocation. |

**Derived metrics** shown by `:stats`: total actions (`COUNT(*)`), total errors
(`SUM(ok = 0)`), top commands and top shortcuts (`GROUP BY name`), and — for
`action` events — per-action run count, failures, and average `duration_ms`
(the **Actions** section).

## ➕ Adding a new event

1. **Check the invariants** above — especially "no content, bounded cardinality".
2. Call `a.recordTelemetry(kind, name, ok)` at the capture site (it's a no-op
   when telemetry is disabled, safe from any goroutine). Reuse an existing `kind`
   when it fits; introduce a new one only for a genuinely different event class.
   To measure an action's **outcome + timing**, capture a `start := time.Now()`
   before the operation and call `a.recordAction(name, start, err)` after it
   (emits a `kind="action"` event with `ok` and `duration_ms`).
3. **Add a row to the [tracking plan](#-tracking-plan)** describing the new
   `kind`/`name`, `ok` semantics, and where it's emitted.
4. If the dashboard should surface it, extend `TelemetrySummary` +
   `generateTelemetryContent` and add a store query.
5. Add/extend tests (`internal/db/telemetry_store_test.go`,
   `internal/services/telemetry_service_test.go`).

Most new features need **no** new event: a new command or keyboard shortcut is
already captured generically by the `command`/`shortcut` hooks. Add a bespoke
event only when a distinct action is worth measuring on its own (e.g. an
outcome/failure rate, or a state you can't infer from a command name).

## 🖥️ Dashboard commands

| Command | Effect |
|---------|--------|
| `:stats` | Open the usage dashboard (totals, top commands, top shortcuts, actions) |
| `:stats <days>` | Set the window (default 30) |
| `:stats reset` | Delete all captured telemetry for the account |
| `Esc` | Close the dashboard |

Prompt-template usage stats are separate and live under `:prompt stats`.

## 🧭 Scope & roadmap

**TUI-only** so far. Captures commands, shortcuts, and error counts, plus
per-action **outcome + timing** for a handful of high-value actions (`action`
events with `ok` and `duration_ms`: archive, trash, summarize).
Deferred (epic #41): **more instrumented actions** (send, search), **reports**
(weekly/monthly), **CSV/JSON export**, `ctrl+…` shortcut capture, and **desktop
parity**.
