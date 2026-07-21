import { useEffect, useMemo, useRef, useState } from "react";
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
  count,
}: {
  labels: Label[];
  onMove: (name: string) => void;
  onClose: () => void;
  // When set (bulk move), the header shows how many messages will be moved.
  count?: number;
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

  // Enter moves to the active folder, or to the typed name when nothing matches
  // (creating the label). Keys are driven at the window (WKWebView won't deliver
  // Enter from a focused input to our handler reliably — the whole reason the
  // other pickers use windowKeys), so we handle Enter here and let useListNav's
  // window binding own the arrows/Escape.
  const filterRef = useRef(filter);
  filterRef.current = filter;
  const visibleRef = useRef(visible);
  visibleRef.current = visible;
  const activeRef = useRef(0);
  const nav = useListNav(visible, {
    onEnter: (l) => onMove(l.name),
    onEscape: onClose,
    windowKeys: true,
  });
  activeRef.current = nav.active;

  // useListNav's onEnter only fires when there's a match; a window listener
  // covers the empty-list case (Enter creates + moves to the typed folder).
  useEffect(() => {
    const h = (e: globalThis.KeyboardEvent) => {
      if (e.key !== "Enter") return;
      if (visibleRef.current.length > 0) return; // handled by useListNav
      const typed = filterRef.current.trim();
      if (typed) {
        e.preventDefault();
        onMove(typed);
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [onMove]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>{count ? `Move ${count} to folder` : "Move to folder"}</h3>
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
                  onMouseEnter={() => nav.setActiveHover(i)}
                  onClick={() => onMove(l.name)}
                >
                  <span className="name">{l.name}</span>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            type to filter · ↑↓ move · Enter move · Esc close
          </span>
        </div>
      </div>
    </div>
  );
}
