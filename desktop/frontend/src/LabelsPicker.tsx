import { useEffect, useMemo, useState } from "react";
import { backend, type Label } from "./api";

// System labels the user cannot meaningfully toggle from here.
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

export default function LabelsPicker({
  messageId,
  onClose,
  onChanged,
}: {
  messageId: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const [labels, setLabels] = useState<Label[]>([]);
  const [applied, setApplied] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [pending, setPending] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const [all, mine] = await Promise.all([
          backend.ListLabels(),
          backend.MessageLabelIDs(messageId),
        ]);
        setLabels(
          all
            .filter((l) => !HIDDEN.has(l.id) && !l.id.startsWith("CATEGORY_"))
            .sort((a, b) => a.name.localeCompare(b.name)),
        );
        setApplied(new Set(mine));
      } catch (e) {
        setError(String(e));
      } finally {
        setLoading(false);
      }
    })();
  }, [messageId]);

  const visible = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return labels;
    return labels.filter((l) => l.name.toLowerCase().includes(q));
  }, [labels, filter]);

  const toggle = async (label: Label) => {
    setPending(label.id);
    setError("");
    const isApplied = applied.has(label.id);
    try {
      if (isApplied) {
        await backend.RemoveLabel(messageId, label.id);
        applied.delete(label.id);
      } else {
        await backend.ApplyLabel(messageId, label.id);
        applied.add(label.id);
      }
      setApplied(new Set(applied));
      onChanged();
    } catch (e) {
      setError(String(e));
    } finally {
      setPending(null);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Labels</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        <div className="modal-body">
          <input
            className="label-filter"
            placeholder="Filter labels…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            autoFocus
          />
          <div className="label-list">
            {loading ? (
              <div className="placeholder">Loading…</div>
            ) : visible.length === 0 ? (
              <div className="placeholder">No labels</div>
            ) : (
              visible.map((l) => {
                const on = applied.has(l.id);
                return (
                  <button
                    key={l.id}
                    className={"label-row" + (on ? " on" : "")}
                    disabled={pending === l.id}
                    onClick={() => void toggle(l)}
                  >
                    <span className="check">{on ? "✓" : ""}</span>
                    <span className="name">{l.name}</span>
                  </button>
                );
              })
            )}
          </div>
        </div>
        <div className="modal-foot">
          <button onClick={onClose}>Done</button>
        </div>
      </div>
    </div>
  );
}
