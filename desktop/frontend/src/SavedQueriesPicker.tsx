import { useMemo, useState } from "react";
import { useListNav } from "./useListNav";
import { Icon } from "./Icons";
import { groupedSavedQueries } from "./savedQueriesModel";
import type { SavedQuery } from "./api";

export default function SavedQueriesPicker({
  queries,
  canSaveCurrent,
  onRun,
  onDelete,
  onSaveCurrent,
  onClose,
}: {
  queries: SavedQuery[];
  canSaveCurrent: boolean;
  onRun: (q: SavedQuery) => void;
  onDelete: (id: number) => void;
  onSaveCurrent: () => void;
  onClose: () => void;
}) {
  const [filter, setFilter] = useState("");

  // Filtered + grouped rows; each carries whether it opens a new category group
  // so we can render one header per group. useListNav only sees the real query
  // rows (headers are inert decorations), so arrow/Enter indices stay correct.
  const rows = useMemo(() => groupedSavedQueries(queries, filter), [queries, filter]);
  const visible = useMemo(() => rows.map((r) => r.query), [rows]);

  const nav = useListNav(visible, {
    onEnter: onRun,
    onEscape: onClose,
    windowKeys: true,
  });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Saved searches</h3>
          <span className="help-hint muted">↑↓ · Enter · Esc</span>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <input
            className="label-filter"
            placeholder="Filter by name, or @category"
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              nav.setActive(0);
            }}
            autoFocus
          />
          <div className="label-list" ref={nav.listRef}>
            {queries.length === 0 ? (
              <div className="placeholder">No saved searches</div>
            ) : visible.length === 0 ? (
              <div className="placeholder">No matches</div>
            ) : (
              rows.map((r, i) => (
                <div key={r.query.id}>
                  {r.groupStart && (
                    <div className="query-group-head">{r.category}</div>
                  )}
                  <div
                    className={"query-row" + (i === nav.active ? " nav-active" : "")}
                    onMouseEnter={() => nav.setActiveHover(i)}
                  >
                    <button className="query-main" onClick={() => onRun(r.query)}>
                      <span className="prompt-name">{r.query.name}</span>
                      <span className="prompt-desc">{r.query.query}</span>
                    </button>
                    <button
                      className="ghost tiny danger"
                      title="Delete"
                      onClick={() => onDelete(r.query.id)}
                    >
                      {Icon.trash}
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            type to filter · @cat by category · ↑↓ move · Enter run · Esc close
          </span>
          {canSaveCurrent && (
            <button className="ghost" onClick={onSaveCurrent}>
              Save current search
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
