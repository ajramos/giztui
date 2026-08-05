import { useEffect, useMemo, useState } from "react";
import { useListNav } from "./useListNav";
import { usePickerCrud } from "./usePickerCrud";
import { useConfirm } from "./useConfirm";
import { Icon } from "./Icons";
import { groupedSavedQueries } from "./savedQueriesModel";
import type { SavedQuery } from "./api";

export default function SavedQueriesPicker({
  queries,
  canSaveCurrent,
  editing,
  onRun,
  onEdit,
  onNew,
  onDelete,
  onSaveCurrent,
  onClose,
}: {
  queries: SavedQuery[];
  canSaveCurrent: boolean;
  // True while the stacked edit/new dialog is open, so Shift+N doesn't re-trigger.
  editing: boolean;
  onRun: (q: SavedQuery) => void;
  onEdit: (q: SavedQuery) => void;
  onNew: () => void;
  onDelete: (id: number) => void;
  onSaveCurrent: () => void;
  onClose: () => void;
}) {
  const [filter, setFilter] = useState("");
  const confirm = useConfirm();

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

  const askDelete = (q: SavedQuery) =>
    confirm.ask(`Delete saved search “${q.name}”? This can’t be undone.`, () =>
      onDelete(q.id),
    );

  // Keyboard edit/delete of the highlighted row via the shared convention. The
  // filter input is always focused here, so in practice these fire as Shift+E /
  // Shift+Delete / Shift+Backspace (a bare key types into the filter); the per-row
  // pencil/trash buttons remain for the mouse. Disabled while a confirm is up.
  usePickerCrud(confirm.open ? [] : visible, nav.active, {
    onEdit,
    onDelete: askDelete,
  });

  // Shift+N creates a saved search from scratch. The filter input is focused so a
  // bare "n" types; Shift+N is intercepted (like Shift+E / Shift+Del). Disabled
  // while the edit/new dialog or a confirm is already open.
  const busy = editing || confirm.open;
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (busy) return;
      if (e.shiftKey && (e.key === "N" || e.key === "n")) {
        e.preventDefault();
        e.stopImmediatePropagation();
        onNew();
      }
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, [busy, onNew]);

  return (
    <>
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
                      className="ghost tiny"
                      title="Edit"
                      onClick={() => onEdit(r.query)}
                    >
                      {Icon.edit}
                    </button>
                    <button
                      className="ghost tiny danger"
                      title="Delete"
                      onClick={() => askDelete(r.query)}
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
            ↑↓ move · Enter run · ⇧E edit · ⇧⌫ delete
          </span>
          {canSaveCurrent && (
            <button className="ghost" onClick={onSaveCurrent}>
              Save current search
            </button>
          )}
          <button className="ghost" title="New saved search (⇧N)" onClick={onNew}>
            {Icon.plus} New
          </button>
        </div>
      </div>
    </div>
    {confirm.node}
    </>
  );
}
