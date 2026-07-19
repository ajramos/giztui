import { useEffect, useMemo, useState } from "react";
import { backend, type Label } from "./api";
import { useListNav } from "./useListNav";

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

// Provide exactly one of messageId (single message) or bulkIds (many messages).
export default function LabelsPicker({
  messageId,
  bulkIds,
  onClose,
  onChanged,
}: {
  messageId?: string;
  bulkIds?: string[];
  onClose: () => void;
  // Reports the label NAME just added/removed so callers can update the list
  // and reader chips in place (no refetch).
  onChanged: (change: { added?: string; removed?: string }) => void;
}) {
  const bulk = bulkIds !== undefined;
  const [labels, setLabels] = useState<Label[]>([]);
  const [applied, setApplied] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [pending, setPending] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const all = await backend.ListLabels();
        setLabels(
          all
            .filter((l) => !HIDDEN.has(l.id) && !l.id.startsWith("CATEGORY_"))
            .sort((a, b) => a.name.localeCompare(b.name)),
        );
        // For a single message we can show which labels are already applied.
        if (!bulk && messageId) {
          setApplied(new Set(await backend.MessageLabelIDs(messageId)));
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setLoading(false);
      }
    })();
  }, [messageId, bulk]);

  const visible = useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return labels;
    return labels.filter((l) => l.name.toLowerCase().includes(q));
  }, [labels, filter]);

  const nav = useListNav(visible, {
    onEnter: (l) => void toggle(l),
    onEscape: onClose,
    windowKeys: true,
  });

  const toggle = async (label: Label) => {
    setPending(label.id);
    setError("");
    const isApplied = applied.has(label.id);
    try {
      if (bulk && bulkIds) {
        if (isApplied) {
          await backend.BulkRemoveLabel(bulkIds, label.id);
          applied.delete(label.id);
        } else {
          await backend.BulkApplyLabel(bulkIds, label.id);
          applied.add(label.id);
        }
      } else if (messageId) {
        if (isApplied) {
          await backend.RemoveLabel(messageId, label.id);
          applied.delete(label.id);
        } else {
          await backend.ApplyLabel(messageId, label.id);
          applied.add(label.id);
        }
      }
      setApplied(new Set(applied));
      onChanged(isApplied ? { removed: label.name } : { added: label.name });
    } catch (e) {
      setError(String(e));
    } finally {
      setPending(null);
    }
  };

  const title = bulk ? `Labels · ${bulkIds?.length ?? 0} selected` : "Labels";

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        <div className="modal-body">
          <input
            className="label-filter"
            placeholder="Filter labels… (↑↓ move · Enter toggle · Esc close)"
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
              <div className="placeholder">No labels</div>
            ) : (
              visible.map((l, i) => {
                const on = applied.has(l.id);
                return (
                  <button
                    key={l.id}
                    className={
                      "label-row" +
                      (on ? " on" : "") +
                      (i === nav.active ? " nav-active" : "")
                    }
                    disabled={pending === l.id}
                    onMouseEnter={() => nav.setActive(i)}
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
