import type { UsageStats } from "./api";

// AI prompt-usage stats (opened by :stats). Pure display modal.
export default function StatsModal({
  stats,
  onClose,
}: {
  stats: UsageStats | null;
  onClose: () => void;
}) {
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
          <h3>AI usage</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          {!stats ? (
            <div className="placeholder">Loading…</div>
          ) : (
            <>
              <div className="stats-summary">
                <div className="stat-tile">
                  <span className="stat-num">{stats.totalUsage}</span>
                  <span className="stat-label muted">total runs</span>
                </div>
                <div className="stat-tile">
                  <span className="stat-num">{stats.uniquePrompts}</span>
                  <span className="stat-label muted">prompts used</span>
                </div>
              </div>
              <div className="label-list">
                {stats.topPrompts.length === 0 ? (
                  <div className="placeholder">No usage yet</div>
                ) : (
                  stats.topPrompts.map((p) => (
                    <div key={p.name} className="stat-row">
                      <span className="stat-row-name">{p.name}</span>
                      {p.category && (
                        <span className="stat-row-cat muted">{p.category}</span>
                      )}
                      <span className="stat-row-count">{p.usageCount}</span>
                    </div>
                  ))
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
