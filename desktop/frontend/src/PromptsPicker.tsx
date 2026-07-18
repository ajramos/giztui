import { useEffect, useMemo, useState } from "react";
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
  const [active, setActive] = useState(0);

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

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            onClose();
          } else if (e.key === "ArrowDown") {
            e.preventDefault();
            setActive((i) => Math.min(visible.length - 1, i + 1));
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setActive((i) => Math.max(0, i - 1));
          } else if (e.key === "Enter") {
            e.preventDefault();
            if (visible[active]) onPick(visible[active]);
          }
        }}
      >
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
            placeholder="Filter prompts…"
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              setActive(0);
            }}
            autoFocus
          />
          <div className="label-list">
            {loading ? (
              <div className="placeholder">Loading…</div>
            ) : visible.length === 0 ? (
              <div className="placeholder">No prompts</div>
            ) : (
              visible.map((p, i) => (
                <button
                  key={p.id}
                  className={"prompt-row" + (i === active ? " active" : "")}
                  onMouseEnter={() => setActive(i)}
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
      </div>
    </div>
  );
}
