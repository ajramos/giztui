import type {
  TelemetrySummary,
  TelemetryNameCount,
  TelemetryActionStat,
} from "./apiTypes";

// Local usage-analytics dashboard (opened by :stats), mirroring the TUI's
// telemetry view. Pure display; data is fetched by the caller. All data is
// local-only and never uploaded.

function fmtDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function BarRow({
  name,
  count,
  max,
  detail,
}: {
  name: string;
  count: number;
  max: number;
  detail?: string;
}) {
  const pct = max > 0 ? Math.max(4, Math.round((count / max) * 100)) : 0;
  return (
    <div className="tele-row">
      <span className="tele-row-name" title={name}>
        {name}
      </span>
      <span className="tele-bar-track">
        <span className="tele-bar-fill" style={{ width: `${pct}%` }} />
      </span>
      {detail ? (
        <span className="tele-row-detail">{detail}</span>
      ) : (
        <span className="tele-row-count">{count}</span>
      )}
    </div>
  );
}

function Section({
  title,
  rows,
}: {
  title: string;
  rows: TelemetryNameCount[];
}) {
  const max = rows.reduce((m, r) => Math.max(m, r.count), 0);
  return (
    <>
      <div className="tele-section-title">{title}</div>
      {rows.length === 0 ? (
        <div className="tele-empty">(none yet)</div>
      ) : (
        rows.map((r) => <BarRow key={r.name} name={r.name} count={r.count} max={max} />)
      )}
    </>
  );
}

function ActionsSection({ rows }: { rows: TelemetryActionStat[] }) {
  const max = rows.reduce((m, r) => Math.max(m, r.count), 0);
  return (
    <>
      <div className="tele-section-title">Actions (outcome · timing)</div>
      {rows.length === 0 ? (
        <div className="tele-empty">(none yet)</div>
      ) : (
        rows.map((r) => (
          <BarRow
            key={r.name}
            name={r.name}
            count={r.count}
            max={max}
            detail={`${r.count} runs · ${r.failures} failed · ${fmtDuration(r.avgDurationMs)} avg`}
          />
        ))
      )}
    </>
  );
}

export default function TelemetryModal({
  summary,
  onReset,
  onClose,
}: {
  summary: TelemetrySummary | null;
  onReset: () => void;
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
          <h3>Usage analytics</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          {!summary ? (
            <div className="placeholder">Loading…</div>
          ) : summary.totalActions === 0 ? (
            <div className="placeholder">
              No activity captured yet. Telemetry is local-only and opt-in — keep
              using GizTUI and your command/shortcut usage will appear here.
            </div>
          ) : (
            <>
              <div className="stats-summary">
                <div className="stat-tile">
                  <span className="stat-num">{summary.totalActions}</span>
                  <span className="stat-label muted">total actions</span>
                </div>
                <div className="stat-tile">
                  <span className="stat-num">{summary.totalErrors}</span>
                  <span className="stat-label muted">errors</span>
                </div>
              </div>
              <Section title="Top commands" rows={summary.topCommands} />
              <Section title="Top shortcuts (keys)" rows={summary.topShortcuts} />
              <ActionsSection rows={summary.topActions} />
            </>
          )}
          <div className="foot-hint" style={{ marginTop: 12 }}>
            last {summary?.windowDays ?? 30} days · local-only, never uploaded ·{" "}
            <button className="ghost" onClick={onReset}>
              :stats reset
            </button>{" "}
            · Esc close
          </div>
        </div>
      </div>
    </div>
  );
}
