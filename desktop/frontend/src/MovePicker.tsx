import { useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useListNav } from "./useListNav";
import type { Label } from "./api";

// System labels you can't "move to".
const HIDDEN = new Set([
  "INBOX",
  "SENT",
  "DRAFT",
  "TRASH",
  "SPAM",
  "CHAT",
  "UNREAD",
  "IMPORTANT",
  "STARRED",
]);

// MovePicker replaces the old free-text + <datalist> move dialog, which didn't
// work with the keyboard on WKWebView. Filter as you type, arrow to a folder,
// Enter to move (apply the label + archive). Enter with no match moves to the
// typed name (creating the label), mirroring the TUI's move-to-folder.
export default function MovePicker({
  labels,
  onMove,
  onClose,
}: {
  labels: Label[];
  onMove: (name: string) => void;
  onClose: () => void;
}) {
  const [filter, setFilter] = useState("");
  const visible = useMemo(() => {
    const q = filter.trim().toLowerCase();
    const usable = labels.filter(
      (l) => !HIDDEN.has(l.id) && !l.id.startsWith("CATEGORY_"),
    );
    const list = q
      ? usable.filter((l) => l.name.toLowerCase().includes(q))
      : usable;
    return list.sort((a, b) => a.name.localeCompare(b.name));
  }, [labels, filter]);

  const nav = useListNav(visible, { onEscape: onClose });

  // Single window listener (WKWebView won't focus a bare div): Enter moves to
  // the active folder, or to the typed name when nothing matches; arrows/Escape
  // delegate to the list nav.
  const filterRef = useRef(filter);
  filterRef.current = filter;
  const visibleRef = useRef(visible);
  visibleRef.current = visible;
  const navKeyRef = useRef(nav.onKeyDown);
  navKeyRef.current = nav.onKeyDown;
  const activeRef = useRef(nav.active);
  activeRef.current = nav.active;

  const onKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      const chosen = visibleRef.current[activeRef.current]?.name ?? filterRef.current.trim();
      if (chosen) onMove(chosen);
      return;
    }
    navKeyRef.current(e);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Move to folder</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <input
            className="label-filter"
            placeholder="Filter folders… (↑↓ · Enter move · Esc close)"
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              nav.setActive(0);
            }}
            onKeyDown={onKeyDown}
            autoFocus
          />
          <div className="label-list" ref={nav.listRef}>
            {visible.length === 0 ? (
              <div className="placeholder">
                {filter.trim()
                  ? `Press Enter to move to “${filter.trim()}”`
                  : "No folders"}
              </div>
            ) : (
              visible.map((l, i) => (
                <button
                  key={l.id}
                  className={"label-row" + (i === nav.active ? " nav-active" : "")}
                  onMouseEnter={() => nav.setActive(i)}
                  onClick={() => onMove(l.name)}
                >
                  <span className="name">{l.name}</span>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <button onClick={onClose}>Cancel</button>
        </div>
      </div>
    </div>
  );
}
