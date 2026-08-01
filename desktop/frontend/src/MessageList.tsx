import type { MessageSummary, DraftSummary } from "./api";
import { Icon, IconBtn } from "./Icons";
import { displayName, formatDate } from "./format";

// The left pane: the drafts list (when draftsView) or the inbox/search message
// list with its new-mail banner, count header, bulk toolbar, rows, and
// load-more. Presentational — every bit of state and every handler stays in
// App; this is a behavior-preserving extraction of the `<aside className="list">`
// block. Split out so App.tsx stops being a wall of render.
export default function MessageList({
  pageSize,
  onBlurReader,
  draftsView,
  drafts,
  loadingDrafts,
  onRefreshDrafts,
  onBackToInbox,
  onOpenDraft,
  pendingNew,
  onShowPendingNew,
  loadingList,
  messages,
  localFilter,
  fullCount,
  nextToken,
  activeQuery,
  selectedId,
  bulkMode,
  selected,
  busy,
  bulkProgress,
  onBulkAction,
  onBulkLabels,
  onBulkMove,
  onSelectAll,
  onExitBulk,
  onToggleSelect,
  onOpenMessage,
  loadingMore,
  onLoadMore,
}: {
  pageSize: number;
  onBlurReader: () => void;
  draftsView: boolean;
  drafts: DraftSummary[];
  loadingDrafts: boolean;
  onRefreshDrafts: () => void;
  onBackToInbox: () => void;
  onOpenDraft: (d: DraftSummary) => void;
  pendingNew: MessageSummary[];
  onShowPendingNew: () => void;
  loadingList: boolean;
  messages: MessageSummary[];
  localFilter: boolean;
  fullCount: number;
  nextToken: string;
  activeQuery: string;
  selectedId: string | null;
  bulkMode: boolean;
  selected: Set<string>;
  busy: boolean;
  bulkProgress: string;
  onBulkAction: (action: "archive" | "trash" | "read" | "unread") => void;
  onBulkLabels: () => void;
  onBulkMove: () => void;
  onSelectAll: () => void;
  onExitBulk: () => void;
  onToggleSelect: (id: string) => void;
  onOpenMessage: (m: MessageSummary) => void;
  loadingMore: boolean;
  onLoadMore: () => void;
}) {
  return (
    <aside className="list" onMouseDown={onBlurReader}>
      {draftsView ? (
        <>
          <div className="bulk-bar">
            <span className="bulk-count">Drafts</span>
            <div className="bulk-actions">
              <button className="tiny ghost" onClick={onRefreshDrafts}>
                Refresh
              </button>
              <button className="tiny ghost" onClick={onBackToInbox}>
                Back to inbox
              </button>
            </div>
          </div>
          {loadingDrafts ? (
            <div className="placeholder">Loading…</div>
          ) : drafts.length === 0 ? (
            <div className="placeholder">No drafts</div>
          ) : (
            <ul>
              {drafts.map((d) => (
                <li key={d.id} className="row" onClick={() => onOpenDraft(d)}>
                  <div className="row-top">
                    <span className="from">
                      {d.to ? `To: ${d.to}` : "(no recipient)"}
                    </span>
                  </div>
                  <div className="subject">{d.subject || "(no subject)"}</div>
                  <div className="snippet">{d.snippet}</div>
                </li>
              ))}
            </ul>
          )}
        </>
      ) : (
        <>
          {/* New mail arrived in the background: show a banner instead of
              injecting it, so the list never shifts under an in-progress action.
              Clicking (or refreshing) merges it in. */}
          {pendingNew.length > 0 && (
            <button className="new-mail-bar" onClick={onShowPendingNew}>
              ↑ {pendingNew.length} new message{pendingNew.length > 1 ? "s" : ""} —
              show
            </button>
          )}
          {/* Loaded-message count (TUI parity — the list title's message tally).
              Shows how many are loaded, a trailing "+" when more can be fetched,
              and "N of M" while a local filter narrows the loaded set. */}
          {!loadingList && messages.length > 0 && (
            <div className="list-head">
              <span className="list-count">
                {localFilter
                  ? `${messages.length} of ${fullCount} emails`
                  : nextToken
                    ? `${messages.length} emails loaded`
                    : `${messages.length} ${messages.length === 1 ? "email" : "emails"}`}
              </span>
              {activeQuery && !localFilter && (
                <span className="list-scope muted">· search</span>
              )}
            </div>
          )}
          {bulkMode && (
            <div className="bulk-bar">
              <div className="bulk-top">
                <span className="bulk-count">{selected.size} selected</span>
                {/* Same IconBtn format as the reader toolbar for consistency. */}
                <div className="actions">
                  <IconBtn
                    icon={Icon.archive}
                    label="Archive"
                    disabled={busy || selected.size === 0}
                    onClick={() => onBulkAction("archive")}
                  />
                  <IconBtn
                    icon={Icon.trash}
                    label="Trash"
                    danger
                    disabled={busy || selected.size === 0}
                    onClick={() => onBulkAction("trash")}
                  />
                  <IconBtn
                    icon={Icon.mailOpen}
                    label="Mark read"
                    disabled={busy || selected.size === 0}
                    onClick={() => onBulkAction("read")}
                  />
                  <IconBtn
                    icon={Icon.mail}
                    label="Mark unread"
                    disabled={busy || selected.size === 0}
                    onClick={() => onBulkAction("unread")}
                  />
                  <IconBtn
                    icon={Icon.label}
                    label="Label…"
                    disabled={busy || selected.size === 0}
                    onClick={onBulkLabels}
                  />
                  <IconBtn
                    icon={Icon.folder}
                    label="Move to folder…"
                    disabled={busy || selected.size === 0}
                    onClick={onBulkMove}
                  />
                  <span className="actions-sep" />
                  <IconBtn
                    icon={Icon.checkAll}
                    label="Select all"
                    disabled={busy}
                    onClick={onSelectAll}
                  />
                  <IconBtn
                    icon={Icon.check}
                    label="Done"
                    primary
                    onClick={onExitBulk}
                  />
                </div>
              </div>
              {bulkProgress && (
                <div className="bulk-progress">
                  <div className="bulk-progress-bar" />
                  <span className="bulk-progress-label">{bulkProgress}</span>
                </div>
              )}
            </div>
          )}
          {loadingList ? (
            <div className="placeholder">Loading…</div>
          ) : messages.length === 0 ? (
            <div className="placeholder">No messages</div>
          ) : (
            <>
              <ul>
                {messages.map((m) => (
                  <li
                    key={m.id}
                    className={
                      "row" +
                      (m.id === selectedId ? " selected" : "") +
                      (m.unread ? " unread" : "") +
                      (bulkMode && selected.has(m.id) ? " checked" : "")
                    }
                    onClick={() =>
                      bulkMode ? onToggleSelect(m.id) : onOpenMessage(m)
                    }
                  >
                    <div className="row-top">
                      {bulkMode && (
                        <span className="row-check">
                          {selected.has(m.id) ? "☑" : "☐"}
                        </span>
                      )}
                      <span className="from">{displayName(m.from)}</span>
                      <span className="date">{formatDate(m.date)}</span>
                      <span
                        className="star-slot"
                        title={m.starred ? "Starred" : undefined}
                      >
                        {m.starred ? Icon.star : null}
                      </span>
                    </div>
                    <div className="subject">{m.subject || "(no subject)"}</div>
                    <div className="snippet">{m.snippet}</div>
                    {m.labels.length > 0 && (
                      <div className="labels">
                        {m.labels.map((l) => (
                          <span key={l} className="label-chip">
                            {l}
                          </span>
                        ))}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
              {nextToken && (
                <button
                  className="load-more"
                  disabled={loadingMore}
                  onClick={onLoadMore}
                >
                  {loadingMore ? "Loading…" : `Load ${pageSize} more`}
                </button>
              )}
            </>
          )}
        </>
      )}
    </aside>
  );
}
