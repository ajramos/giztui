import { useState } from "react";
import type { ConfigInfo } from "./api";
import { backend } from "./api";

// Read-only configuration summary (opened by :config). Mostly a display modal;
// besides "Clear AI caches" it offers ChatGPT subscription login/logout when the
// active engine is a subscription provider that needs an interactive OAuth login.
export default function ConfigModal({
  info,
  onClearCaches,
  onClose,
}: {
  info: ConfigInfo | null;
  onClearCaches: () => void;
  onClose: () => void;
}) {
  // Local login state so the row reflects a login/logout without a full reload;
  // seeded from the backend snapshot and updated after each action.
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string>("");
  const isLoggedIn = loggedIn ?? info?.llmLoggedIn ?? false;

  async function login() {
    setBusy(true);
    setErr("");
    try {
      await backend.LLMLogin();
      setLoggedIn(true);
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  }

  async function logout() {
    setBusy(true);
    setErr("");
    try {
      await backend.LLMLogout();
      setLoggedIn(false);
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  }

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
              {info.llmNeedsLogin && (
                <div className="config-row">
                  <span className="config-key muted">Subscription</span>
                  <span className="config-val">
                    {isLoggedIn ? "logged in" : "not logged in"}
                    {"  "}
                    <button
                      className="ghost"
                      disabled={busy}
                      onClick={() => void (isLoggedIn ? logout() : login())}
                    >
                      {busy
                        ? "Waiting… (browser opened; URL also copied)"
                        : isLoggedIn
                          ? "Log out"
                          : "Log in with ChatGPT"}
                    </button>
                  </span>
                </div>
              )}
              {err && <div className="config-row error">{err}</div>}
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
