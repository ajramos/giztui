import { buildAdvancedQuery, type AdvFilters } from "./advancedSearch";

// Advanced search builder (opened by Ctrl+F / :advanced). Presentational: the
// `adv` form state stays in App (behavior-preserving move); this renders the
// form, previews the built query, and hands the final query to onSearch.
export default function AdvancedSearchModal({
  adv,
  onChange,
  onSearch,
  onClose,
}: {
  adv: AdvFilters;
  onChange: (next: AdvFilters) => void;
  onSearch: (query: string) => void;
  onClose: () => void;
}) {
  const query = buildAdvancedQuery(adv);
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
        }}
      >
        <div className="modal-head">
          <h3>Advanced search</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="field">
            <label>From</label>
            <input
              value={adv.from}
              onChange={(e) => onChange({ ...adv, from: e.target.value })}
              placeholder="sender@example.com"
              autoFocus
            />
          </div>
          <div className="field">
            <label>To</label>
            <input
              value={adv.to}
              onChange={(e) => onChange({ ...adv, to: e.target.value })}
              placeholder="recipient@example.com"
            />
          </div>
          <div className="field">
            <label>Subject</label>
            <input
              value={adv.subject}
              onChange={(e) => onChange({ ...adv, subject: e.target.value })}
              placeholder="words in the subject"
            />
          </div>
          <div className="adv-row">
            <label className="adv-check">
              <input
                type="checkbox"
                checked={adv.hasAttachment}
                onChange={(e) => onChange({ ...adv, hasAttachment: e.target.checked })}
              />
              Has attachment
            </label>
            <label className="adv-check">
              <input
                type="checkbox"
                checked={adv.unreadOnly}
                onChange={(e) => onChange({ ...adv, unreadOnly: e.target.checked })}
              />
              Unread only
            </label>
          </div>
          <div className="adv-row">
            <div className="field">
              <label>After</label>
              <input
                type="date"
                value={adv.after}
                onChange={(e) => onChange({ ...adv, after: e.target.value })}
              />
            </div>
            <div className="field">
              <label>Before</label>
              <input
                type="date"
                value={adv.before}
                onChange={(e) => onChange({ ...adv, before: e.target.value })}
              />
            </div>
          </div>
          <div className="field readonly">
            <label>Query preview</label>
            <div className="ro-value">{query || "(empty)"}</div>
          </div>
        </div>
        <div className="modal-foot">
          <button className="ghost" onClick={onClose}>
            Cancel
          </button>
          <button onClick={() => onSearch(query)} disabled={!query}>
            Search
          </button>
        </div>
      </div>
    </div>
  );
}
