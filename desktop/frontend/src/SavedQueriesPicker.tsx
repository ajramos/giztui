import { useEffect, useRef } from "react";
import { useListNav } from "./useListNav";
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
  const modalRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    modalRef.current?.focus();
  }, []);

  const nav = useListNav(queries, { onEnter: onRun, onEscape: onClose });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={modalRef}
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={nav.onKeyDown}
        tabIndex={-1}
      >
        <div className="modal-head">
          <h3>Saved searches</h3>
          <span className="help-hint muted">↑↓ · Enter · Esc</span>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="label-list" ref={nav.listRef}>
            {queries.length === 0 ? (
              <div className="placeholder">No saved searches</div>
            ) : (
              queries.map((q, i) => (
                <div
                  key={q.id}
                  className={"query-row" + (i === nav.active ? " nav-active" : "")}
                  onMouseEnter={() => nav.setActive(i)}
                >
                  <button className="query-main" onClick={() => onRun(q)}>
                    <span className="prompt-name">{q.name}</span>
                    <span className="prompt-desc">{q.query}</span>
                  </button>
                  <button
                    className="ghost tiny"
                    title="Delete"
                    onClick={() => onDelete(q.id)}
                  >
                    ✕
                  </button>
                </div>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          {canSaveCurrent && (
            <button className="ghost" onClick={onSaveCurrent}>
              Save current search
            </button>
          )}
          <button onClick={onClose}>Done</button>
        </div>
      </div>
    </div>
  );
}
