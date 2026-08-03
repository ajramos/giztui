import { useState } from "react";
import type { SavedQuery } from "./api";

// Edit dialog for an existing saved search — name, query and category are all
// editable (unlike SaveQueryModal, whose query is the fixed current search). Self
// contained: it seeds its fields from the query and hands the edited values back.
export default function EditQueryModal({
  query,
  onSave,
  onClose,
}: {
  query: SavedQuery;
  onSave: (name: string, queryText: string, category: string) => void;
  onClose: () => void;
}) {
  const [name, setName] = useState(query.name);
  const [text, setText] = useState(query.query);
  const [category, setCategory] = useState(query.category || "");

  const canSave = name.trim() !== "" && text.trim() !== "";
  const submit = () => {
    if (canSave) onSave(name.trim(), text.trim(), category.trim());
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
          else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) submit();
        }}
      >
        <div className="modal-head">
          <h3>Edit saved search</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="field">
            <label>Name</label>
            <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
          </div>
          <div className="field">
            <label>Query</label>
            <input value={text} onChange={(e) => setText(e.target.value)} />
          </div>
          <div className="field">
            <label>Category (optional)</label>
            <input
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="e.g. Work — groups it in the picker"
            />
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">⌘/Ctrl+Enter save · Esc cancel</span>
          <button className="ghost" onClick={onClose}>
            Cancel
          </button>
          <button onClick={submit} disabled={!canSave}>
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
