import { useEffect, useMemo, useState } from "react";
import { useListNav } from "./useListNav";
import { usePickerCrud } from "./usePickerCrud";
import { useConfirm } from "./useConfirm";
import { groupByCategory } from "./entityGroups";
import PromptEditModal from "./PromptEditModal";
import { Icon } from "./Icons";
import { backend, type Prompt, type PromptDetail } from "./api";

const EMPTY: PromptDetail = {
  id: 0,
  name: "",
  description: "",
  category: "",
  text: "",
};

// PromptsPicker is the single surface for prompts, built to be visually identical
// to SavedQueriesPicker: a filter (name or @category), category-grouped rows with
// inline edit (pencil / e / Shift+E) and delete (trash / d / Shift+Del, with a
// confirm step), plus New (Shift+N / footer). Editing opens a stacked modal.
export default function PromptsPicker({
  aiEnabled,
  onClose,
  onPick,
  onChanged,
}: {
  aiEnabled: boolean;
  onClose: () => void;
  onPick: (prompt: Prompt) => void;
  onChanged?: () => void;
}) {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<PromptDetail | null>(null);
  const confirm = useConfirm();

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

  // Filtered + category-grouped rows via the shared model (same as saved
  // searches), so the two pickers look and filter identically.
  const rows = useMemo(
    () => groupByCategory(prompts, filter, (p) => p.name, (p) => p.category),
    [prompts, filter],
  );
  const visible = useMemo(() => rows.map((r) => r.item), [rows]);

  const nav = useListNav(visible, {
    onEnter: (p) => onPick(p),
    onEscape: onClose,
    windowKeys: true,
  });

  const openEdit = async (id: number) => {
    setError("");
    try {
      setEditing(await backend.GetPrompt(id));
    } catch (e) {
      setError(String(e));
    }
  };
  const remove = async (p: Prompt) => {
    setError("");
    try {
      await backend.DeletePrompt(p.id);
      setPrompts((ps) => ps.filter((x) => x.id !== p.id));
      onChanged?.();
    } catch (e) {
      setError(String(e));
    }
  };
  const askRemove = (p: Prompt) =>
    confirm.ask(`Delete prompt “${p.name}”? This can’t be undone.`, () => void remove(p));
  const saveEdit = async (d: PromptDetail) => {
    if (d.id > 0) {
      await backend.UpdatePrompt(d.id, d.name, d.description, d.text, d.category);
    } else {
      await backend.CreatePrompt(d.name, d.description, d.text, d.category);
    }
    setEditing(null);
    setPrompts(await backend.ListPrompts());
    onChanged?.();
  };

  const busy = editing !== null || confirm.open;
  usePickerCrud(busy ? [] : visible, nav.active, {
    onEdit: (p) => void openEdit(p.id),
    onDelete: (p) => askRemove(p),
  });

  // Shift+N creates a prompt. The filter input is focused so a bare "n" types;
  // Shift+N is intercepted (like Shift+E / Shift+Del) and opens the empty editor.
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (busy) return;
      if (e.shiftKey && (e.key === "N" || e.key === "n")) {
        e.preventDefault();
        e.stopImmediatePropagation();
        setEditing({ ...EMPTY });
      }
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, [busy]);

  return (
    <>
      <div className="modal-overlay" onClick={onClose}>
        <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
          <div className="modal-head">
            <h3>Prompts</h3>
            <span className="help-hint muted">↑↓ · Enter · Esc</span>
            <button className="ghost" onClick={onClose}>
              ✕
            </button>
          </div>
          {error && <div className="error-banner">{error}</div>}
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
              {loading ? (
                <div className="placeholder">Loading…</div>
              ) : prompts.length === 0 ? (
                <div className="placeholder">No prompts</div>
              ) : visible.length === 0 ? (
                <div className="placeholder">No matches</div>
              ) : (
                rows.map((r, i) => (
                  <div key={r.item.id}>
                    {r.groupStart && (
                      <div className="query-group-head">{r.category}</div>
                    )}
                    <div
                      className={"query-row" + (i === nav.active ? " nav-active" : "")}
                      onMouseEnter={() => nav.setActiveHover(i)}
                    >
                      <button className="query-main" onClick={() => onPick(r.item)}>
                        <span className="prompt-name">{r.item.name}</span>
                        {r.item.description && (
                          <span className="prompt-desc">{r.item.description}</span>
                        )}
                      </button>
                      <button
                        className="ghost tiny"
                        title="Edit"
                        onClick={() => void openEdit(r.item.id)}
                      >
                        {Icon.edit}
                      </button>
                      <button
                        className="ghost tiny danger"
                        title="Delete"
                        onClick={() => askRemove(r.item)}
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
              ↑↓ move · Enter apply · ⇧E edit · ⇧⌫ delete
            </span>
            <button
              className="ghost"
              title="New prompt"
              onClick={() => setEditing({ ...EMPTY })}
            >
              {Icon.plus} New <kbd className="btn-kbd">⇧N</kbd>
            </button>
          </div>
        </div>
      </div>
      {editing && (
        <PromptEditModal
          prompt={editing}
          aiEnabled={aiEnabled}
          onSave={saveEdit}
          onClose={() => setEditing(null)}
        />
      )}
      {confirm.node}
    </>
  );
}
