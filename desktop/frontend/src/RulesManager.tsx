import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { backend, type DeterministicRule, type Prompt } from "./api";
import { useListNav } from "./useListNav";

const ACTIONS = ["archive", "mark_read", "trash", "label", "prompt"] as const;
const ACTION_ICON: Record<string, string> = {
  archive: "📁",
  mark_read: "👁️",
  trash: "🗑️",
  label: "🔖",
  prompt: "🎯",
};

type FormState = {
  id: number | null;
  query: string;
  action: string;
  label: string;
  promptId: number;
};

const EMPTY: FormState = { id: null, query: "", action: "archive", label: "", promptId: 0 };

// RulesManager is the deterministic-rules (:rules) manager: list rules with
// their action + Gmail-sync state, add/edit/delete, toggle sync, and import
// existing Gmail filters — all keyboard-first, matching the TUI.
export default function RulesManager({ onClose }: { onClose: () => void }) {
  const [rules, setRules] = useState<DeterministicRule[]>([]);
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState<FormState | null>(null);
  const [status, setStatus] = useState("");

  const reload = async () => {
    try {
      setRules(await backend.ListDeterministicRules());
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void reload();
    void backend
      .ListPrompts()
      .then(setPrompts)
      .catch(() => undefined);
  }, []);

  const nav = useListNav(rules, { onEscape: onClose });

  const del = async (id: number) => {
    setBusy(true);
    setError("");
    try {
      await backend.DeleteDeterministicRule(id);
      await reload();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const toggleSync = async (r: DeterministicRule) => {
    setBusy(true);
    setError("");
    try {
      if (r.synced) await backend.UnsyncDeterministicRule(r.id);
      else await backend.SyncDeterministicRule(r.id);
      await reload();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const doImport = async () => {
    setBusy(true);
    setError("");
    setStatus("Importing Gmail filters…");
    try {
      const r = await backend.ImportGmailFilters();
      setStatus(
        `Imported ${r.imported}, adopted ${r.adopted}, removed ${r.removed}` +
          (r.unsupported.length ? `, ${r.unsupported.length} unsupported` : ""),
      );
      await reload();
    } catch (e) {
      setError(String(e));
      setStatus("");
    } finally {
      setBusy(false);
    }
  };
  const save = async () => {
    if (!form || !form.query.trim()) return;
    setBusy(true);
    setError("");
    try {
      if (form.id === null) {
        await backend.SaveDeterministicRule(
          form.query.trim(),
          form.action,
          form.action === "label" ? form.label.trim() : "",
          form.action === "prompt" ? form.promptId : 0,
        );
      } else {
        await backend.UpdateDeterministicRule(
          form.id,
          form.query.trim(),
          form.action,
          form.action === "label" ? form.label.trim() : "",
          form.action === "prompt" ? form.promptId : 0,
        );
      }
      setForm(null);
      await reload();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  // List-mode keyboard: a add · Enter edit · d delete · s sync · i import ·
  // arrows move · Esc close. Bound to the window (WKWebView focus), only in list
  // mode. Refs keep it bound once while seeing fresh state.
  const rulesRef = useRef(rules);
  rulesRef.current = rules;
  const navKeyRef = useRef(nav.onKeyDown);
  navKeyRef.current = nav.onKeyDown;
  const activeRef = useRef(nav.active);
  activeRef.current = nav.active;
  useEffect(() => {
    if (form) return;
    const h = (e: globalThis.KeyboardEvent) => {
      const r = rulesRef.current[activeRef.current];
      if (e.key === "a") {
        e.preventDefault();
        setForm({ ...EMPTY });
        return;
      }
      if (e.key === "i") {
        e.preventDefault();
        void doImport();
        return;
      }
      if (e.key === "Enter" && r) {
        e.preventDefault();
        setForm({ id: r.id, query: r.query, action: r.action, label: r.label, promptId: r.promptId });
        return;
      }
      if (e.key === "d" && r) {
        e.preventDefault();
        void del(r.id);
        return;
      }
      if (e.key === "s" && r) {
        e.preventDefault();
        void toggleSync(r);
        return;
      }
      navKeyRef.current(e as unknown as KeyboardEvent);
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [form]);

  const promptName = (id: number) =>
    prompts.find((p) => p.id === id)?.name ?? `#${id}`;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>⚡ Deterministic rules</h3>
          <span className="summary-head-actions">
            {!form && (
              <>
                <button className="ghost tiny" disabled={busy} onClick={() => setForm({ ...EMPTY })}>
                  + Add
                </button>
                <button className="ghost tiny" disabled={busy} onClick={() => void doImport()}>
                  ⬇ Import Gmail
                </button>
              </>
            )}
            <button className="ghost" onClick={onClose}>
              ✕
            </button>
          </span>
        </div>
        {error && <div className="error-banner">{error}</div>}
        {status && !error && <div className="plan-summary muted">{status}</div>}

        {form ? (
          <div className="modal-body">
            <div className="field">
              <label>Gmail query (e.g. from:github.com)</label>
              <input
                autoFocus
                value={form.query}
                onChange={(e) => setForm({ ...form, query: e.target.value })}
                onKeyDown={(e) => {
                  if (e.key === "Escape") setForm(null);
                }}
                placeholder="from:(alerts@pagertree.com)"
              />
            </div>
            <div className="field">
              <label>Action</label>
              <select
                value={form.action}
                onChange={(e) => setForm({ ...form, action: e.target.value })}
              >
                {ACTIONS.map((a) => (
                  <option key={a} value={a}>
                    {ACTION_ICON[a]} {a}
                  </option>
                ))}
              </select>
            </div>
            {form.action === "label" && (
              <div className="field">
                <label>Label name</label>
                <input
                  value={form.label}
                  onChange={(e) => setForm({ ...form, label: e.target.value })}
                  placeholder="Newsletter"
                />
              </div>
            )}
            {form.action === "prompt" && (
              <div className="field">
                <label>Prompt</label>
                <select
                  value={form.promptId}
                  onChange={(e) => setForm({ ...form, promptId: Number(e.target.value) })}
                >
                  <option value={0}>— pick a prompt —</option>
                  {prompts.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <div className="modal-foot">
              <button className="ghost" onClick={() => setForm(null)}>
                Cancel
              </button>
              <button disabled={busy || !form.query.trim()} onClick={() => void save()}>
                {form.id === null ? "Add rule" : "Save"}
              </button>
            </div>
          </div>
        ) : (
          <>
            <div className="modal-body">
              <div className="label-list" ref={nav.listRef}>
                {loading ? (
                  <div className="placeholder">Loading…</div>
                ) : rules.length === 0 ? (
                  <div className="placeholder">No rules yet — press “a” to add one</div>
                ) : (
                  rules.map((r, i) => (
                    <div
                      key={r.id}
                      className={"rule-row" + (i === nav.active ? " nav-active" : "")}
                      onMouseEnter={() => nav.setActive(i)}
                      onClick={() =>
                        setForm({ id: r.id, query: r.query, action: r.action, label: r.label, promptId: r.promptId })
                      }
                    >
                      <span className="rule-action">
                        {ACTION_ICON[r.action] ?? "•"}{" "}
                        {r.action === "label"
                          ? `Label “${r.label}”`
                          : r.action === "prompt"
                            ? `Prompt ${promptName(r.promptId)}`
                            : r.action.replace("_", " ")}
                      </span>
                      <span className="rule-query">{r.query}</span>
                      <button
                        className="ghost tiny"
                        title={r.synced ? "Synced to Gmail — click to unsync" : "Sync as a Gmail filter"}
                        onClick={(e) => {
                          e.stopPropagation();
                          void toggleSync(r);
                        }}
                      >
                        {r.synced ? "☁︎" : "○"}
                      </button>
                      <button
                        className="ghost tiny"
                        title="Delete"
                        onClick={(e) => {
                          e.stopPropagation();
                          void del(r.id);
                        }}
                      >
                        ✕
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
            <div className="modal-foot">
              <span className="foot-hint">
                a add · Enter edit · d delete · s sync (☁) · i import · Esc close
              </span>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
