// Name-and-save dialog for the current Gmail search (opened by :savequery).
// Presentational: the name state stays in App (behavior-preserving pure move),
// so this component just renders and forwards events.
export default function SaveQueryModal({
  name,
  onNameChange,
  category,
  onCategoryChange,
  query,
  onSave,
  onClose,
}: {
  name: string;
  onNameChange: (value: string) => void;
  category: string;
  onCategoryChange: (value: string) => void;
  query: string;
  onSave: () => void;
  onClose: () => void;
}) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
          else if (e.key === "Enter") onSave();
        }}
      >
        <div className="modal-head">
          <h3>Save search</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="field">
            <label>Name</label>
            <input
              value={name}
              onChange={(e) => onNameChange(e.target.value)}
              placeholder="e.g. Unread from team"
              autoFocus
            />
          </div>
          <div className="field">
            <label>Category (optional)</label>
            <input
              value={category}
              onChange={(e) => onCategoryChange(e.target.value)}
              placeholder="e.g. Work — groups it in the picker"
            />
          </div>
          <div className="field readonly">
            <label>Query</label>
            <div className="ro-value">{query}</div>
          </div>
        </div>
        <div className="modal-foot">
          <button className="ghost" onClick={onClose}>
            Cancel
          </button>
          <button onClick={onSave} disabled={!name.trim()}>
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
