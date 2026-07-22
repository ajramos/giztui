import { useEffect } from "react";
import { useListNav } from "./useListNav";

// SuggestPicker shows AI-suggested labels as a keyboard-navigable list (the TUI's
// PickerAI): arrow to a suggestion, Enter (or 1-9) to apply it, Escape to close.
// Applying keeps the picker open so you can apply several; the parent removes an
// applied suggestion from the list.
export default function SuggestPicker({
  suggestions,
  loading,
  onApply,
  onClose,
}: {
  suggestions: string[];
  loading: boolean;
  onApply: (name: string) => void;
  onClose: () => void;
}) {
  const nav = useListNav(suggestions, {
    onEnter: (s) => onApply(s),
    onEscape: onClose,
    windowKeys: true,
  });

  // 1-9 quick-apply, matching the TUI's numbered quick-select in list pickers.
  useEffect(() => {
    const h = (e: globalThis.KeyboardEvent) => {
      if (e.key >= "1" && e.key <= "9") {
        const i = Number(e.key) - 1;
        if (i < suggestions.length) {
          e.preventDefault();
          onApply(suggestions[i]);
        }
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [suggestions, onApply]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>✦ Suggested labels</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="label-list" ref={nav.listRef}>
            {loading ? (
              <div className="placeholder">Thinking…</div>
            ) : suggestions.length === 0 ? (
              <div className="placeholder">No suggestions</div>
            ) : (
              suggestions.map((s, i) => (
                <button
                  key={s}
                  className={"label-row" + (i === nav.active ? " nav-active" : "")}
                  onMouseEnter={() => nav.setActiveHover(i)}
                  onClick={() => onApply(s)}
                >
                  {i < 9 && <span className="link-idx">[{i + 1}]</span>}{" "}
                  <span className="name">{s}</span>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            ↑↓ move · 1-9 quick · Enter apply · Esc close
          </span>
        </div>
      </div>
    </div>
  );
}
