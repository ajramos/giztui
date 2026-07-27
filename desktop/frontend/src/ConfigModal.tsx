import type { ConfigInfo } from "./api";

// Read-only configuration summary (opened by :config). Pure display modal; the
// only action is "Clear AI caches", delegated to the parent.
export default function ConfigModal({
  info,
  onClearCaches,
  onClose,
}: {
  info: ConfigInfo | null;
  onClearCaches: () => void;
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
          <h3>Configuration</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          {!info ? (
            <div className="placeholder">Loading…</div>
          ) : (
            <div className="config-list">
              {(
                [
                  ["Account", info.account],
                  ["Config file", info.configPath],
                  ["Log file", info.logPath],
                  [
                    "LLM",
                    info.llmModel
                      ? `${info.llmProvider} · ${info.llmModel}`
                      : "disabled",
                  ],
                  ["Theme", info.theme || "default"],
                  ["Downloads", info.downloadPath],
                  ["Obsidian", info.obsidianOn ? "on" : "off"],
                  ["Slack", info.slackOn ? "on" : "off"],
                  ["Auto-refresh", info.autoRefresh ? "on" : "off"],
                ] as [string, string][]
              ).map(([k, v]) => (
                <div key={k} className="config-row">
                  <span className="config-key muted">{k}</span>
                  <span className="config-val">{v}</span>
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="modal-foot">
          <button className="ghost" onClick={onClearCaches}>
            Clear AI caches
          </button>
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
