import { useEffect, useMemo, useState } from "react";
import { useListNav } from "./useListNav";
import { backend, type Prompt } from "./api";

export default function PromptsPicker({
  onClose,
  onPick,
  onManage,
}: {
  onClose: () => void;
  onPick: (prompt: Prompt) => void;
  onManage?: () => void;
}) {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void (async () => {
      try {
        setPrompts(await backend.ListPrompts());
      } catch (e) {
        setError(String(e));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const visible = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return prompts;
    return prompts.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q) ||
        p.category.toLowerCase().includes(q),
    );
  }, [prompts, filter]);

  // Keyboard-first, focus-independent: drive arrows/Enter/Escape at the window
  // (WKWebView won't deliver them reliably from the focused input to our
  // handler). The filter input keeps autoFocus for typing; no onKeyDown on it,
  // to avoid double-firing. Mirrors MovePicker / the other standardized pickers.
  const nav = useListNav(visible, {
    onEnter: (p) => onPick(p),
    onEscape: onClose,
    windowKeys: true,
  });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Apply a prompt</h3>
          <span className="summary-head-actions">
            {onManage && (
              <button className="ghost tiny" onClick={onManage}>
                ⚙ Manage
              </button>
            )}
            <button className="ghost" onClick={onClose}>
              ✕
            </button>
          </span>
        </div>
        {error && <div className="error-banner">{error}</div>}
        <div className="modal-body">
          <input
            className="label-filter"
            placeholder="Filter prompts… (↑↓ · Enter apply · Esc close)"
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              nav.setActive(0);
            }}
            autoFocus
          />
          <div className="label-list" ref={nav.listRef}>
            {loading ? (
              <div className="placeholder">Loading…</div>
            ) : visible.length === 0 ? (
              <div className="placeholder">No prompts</div>
            ) : (
              visible.map((p, i) => (
                <button
                  key={p.id}
                  className={"prompt-row" + (i === nav.active ? " nav-active" : "")}
                  onMouseEnter={() => nav.setActiveHover(i)}
                  onClick={() => onPick(p)}
                >
                  <span className="prompt-name">{p.name}</span>
                  {p.description && (
                    <span className="prompt-desc">{p.description}</span>
                  )}
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            type to filter · ↑↓ move · Enter apply · Esc close
          </span>
        </div>
      </div>
    </div>
  );
}
