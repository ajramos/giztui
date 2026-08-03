import { useEffect, useMemo, useState } from "react";
import { useListNav } from "./useListNav";
import { usePickerCrud } from "./usePickerCrud";
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

// PromptsPicker is the single surface for prompts: pick one to apply, and manage
// them inline — edit (pencil / e / Shift+E), delete (trash / d / Shift+Del) and
// create (New prompt) — exactly like SavedQueriesPicker. Editing/creating opens a
// stacked PromptEditModal; mutations reflect in the local list so nothing refetches.
export default function PromptsPicker({
  aiEnabled,
  onClose,
  onPick,
  onChanged,
}: {
  aiEnabled: boolean;
  onClose: () => void;
  onPick: (prompt: Prompt) => void;
  // Fired after a create/edit/delete so the parent can drop cached prompt results.
  onChanged?: () => void;
}) {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<PromptDetail | null>(null);

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
  // (WKWebView won't deliver them reliably from the focused input to our handler).
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
  const remove = async (id: number) => {
    setError("");
    try {
      await backend.DeletePrompt(id);
      setPrompts((ps) => ps.filter((p) => p.id !== id));
      onChanged?.();
    } catch (e) {
      setError(String(e));
    }
  };
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

  // Shared edit/delete keys (e / Shift+E, d / Delete / Backspace / Shift+Del). The
  // filter input is focused, so these fire as the Shift variants; disabled while
  // the edit modal is open (pass no items) so keys belong to the form.
  usePickerCrud(editing ? [] : visible, nav.active, {
    onEdit: (p) => void openEdit(p.id),
    onDelete: (p) => void remove(p.id),
  });

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
              placeholder="Filter prompts…"
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
                visible.map((p, i) => (
                  <div
                    key={p.id}
                    className={"query-row" + (i === nav.active ? " nav-active" : "")}
                    onMouseEnter={() => nav.setActiveHover(i)}
                  >
                    <button className="query-main" onClick={() => onPick(p)}>
                      <span className="prompt-name">{p.name}</span>
                      {p.description && (
                        <span className="prompt-desc">{p.description}</span>
                      )}
                    </button>
                    <button
                      className="ghost tiny"
                      title="Edit"
                      onClick={() => void openEdit(p.id)}
                    >
                      {Icon.edit}
                    </button>
                    <button
                      className="ghost tiny danger"
                      title="Delete"
                      onClick={() => void remove(p.id)}
                    >
                      {Icon.trash}
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
          <div className="modal-foot">
            <span className="foot-hint">
              type to filter · ↑↓ move · Enter apply · ⇧E edit · ⇧⌫ delete · Esc close
            </span>
            <button className="ghost" onClick={() => setEditing({ ...EMPTY })}>
              {Icon.plus} New prompt
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
    </>
  );
}
