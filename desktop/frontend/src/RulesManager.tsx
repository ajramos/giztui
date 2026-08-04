import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { backend, type DeterministicRule, type Prompt } from "./api";
import { useListNav } from "./useListNav";
import { usePickerCrud } from "./usePickerCrud";
import { useConfirm } from "./useConfirm";
import { Icon } from "./Icons";

const ACTIONS = ["archive", "mark_read", "trash", "label", "prompt"] as const;
const ACTION_ICON: Record<string, ReactNode> = {
  archive: Icon.archive,
  mark_read: Icon.mailOpened,
  trash: Icon.trash,
  label: Icon.label,
  prompt: Icon.prompt,
};
const actionLabel = (a: string) =>
  a === "mark_read" ? "Mark read" : a.charAt(0).toUpperCase() + a.slice(1);

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
export default function RulesManager({
  onClose,
  onRun,
}: {
  onClose: () => void;
  // Run the rules over the inbox now (opens the plan panel); optional.
  onRun?: () => void;
}) {
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
  const confirm = useConfirm();
  const askDelete = (r: DeterministicRule) =>
    confirm.ask(`Delete this rule?\n\n${r.query}`, () => void del(r.id));

  // Shared edit/delete keys (e / Shift+E, d / Delete / Backspace / Shift+Del),
  // matching every other CRUD picker. Disabled while the add/edit form or a
  // confirm is open (pass no items) so those keys belong to the form/dialog.
  usePickerCrud(form || confirm.open ? [] : rules, nav.active, {
    onEdit: (r) =>
      setForm({ id: r.id, query: r.query, action: r.action, label: r.label, promptId: r.promptId }),
    onDelete: askDelete,
  });

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

  // Keyboard, bound once to the window (WKWebView won't focus a bare div). One
  // handler owns the whole modal so Escape is form-aware: in the add/edit form it
  // cancels back to the list; in the list it closes the modal. stopImmediate-
  // Propagation keeps the App's global Escape from ALSO closing the modal (which
  // made Escape in the form close everything). Refs keep it bound once with fresh
  // state. List-mode keys: a add · Enter edit · s sync · i import · r run (e/d
  // edit+delete come from usePickerCrud).
  const formRef = useRef(form);
  formRef.current = form;
  const rulesRef = useRef(rules);
  rulesRef.current = rules;
  const navKeyRef = useRef(nav.onKeyDown);
  navKeyRef.current = nav.onKeyDown;
  const activeRef = useRef(nav.active);
  activeRef.current = nav.active;
  const onRunRef = useRef(onRun);
  onRunRef.current = onRun;
  useEffect(() => {
    const h = (e: globalThis.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        if (formRef.current) setForm(null);
        else onClose();
        return;
      }
      // In the form, let the inputs handle their own keys (typing "a"/"i" etc.).
      if (formRef.current) return;
      const r = rulesRef.current[activeRef.current];
      if (e.key === "r" && onRunRef.current) {
        e.preventDefault();
        onRunRef.current();
        return;
      }
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
      // e/Shift+E edit and d/Delete/Backspace delete are handled by usePickerCrud.
      if (e.key === "s" && r) {
        e.preventDefault();
        void toggleSync(r);
        return;
      }
      navKeyRef.current(e as unknown as KeyboardEvent);
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const promptName = (id: number) =>
    prompts.find((p) => p.id === id)?.name ?? `#${id}`;

  return (
    <>
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Deterministic rules</h3>
          <span className="summary-head-actions">
            {!form && (
              <>
                {onRun && (
                  <button className="ghost tiny row-icon" disabled={busy} onClick={onRun} title="Run these rules over the inbox now">
                    {Icon.bolt} Run
                  </button>
                )}
                <button className="ghost tiny row-icon" disabled={busy} onClick={() => setForm({ ...EMPTY })}>
                  {Icon.plus} Add
                </button>
                <button className="ghost tiny row-icon" disabled={busy} onClick={() => void doImport()}>
                  {Icon.download} Import
                </button>
              </>
            )}
            <button className="ghost" onClick={onClose}>
              {Icon.x}
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
                    {actionLabel(a)}
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
                      onMouseEnter={() => nav.setActiveHover(i)}
                      onClick={() =>
                        setForm({ id: r.id, query: r.query, action: r.action, label: r.label, promptId: r.promptId })
                      }
                    >
                      <span className="rule-action">
                        <span className="rule-ico">{ACTION_ICON[r.action]}</span>
                        {r.action === "label"
                          ? `Label “${r.label}”`
                          : r.action === "prompt"
                            ? `Prompt ${promptName(r.promptId)}`
                            : actionLabel(r.action)}
                      </span>
                      <span className="rule-query">{r.query}</span>
                      <button
                        className={"ghost tiny rule-sync" + (r.synced ? " on" : "")}
                        title={r.synced ? "Synced to Gmail — click to unsync" : "Sync as a Gmail filter"}
                        onClick={(e) => {
                          e.stopPropagation();
                          void toggleSync(r);
                        }}
                      >
                        {Icon.cloud}
                      </button>
                      <button
                        className="ghost tiny danger"
                        title="Delete"
                        onClick={(e) => {
                          e.stopPropagation();
                          askDelete(r);
                        }}
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
                {onRun ? "r run · " : ""}a add · Enter/e edit · d delete · s sync ·
                i import · Esc close
              </span>
            </div>
          </>
        )}
      </div>
    </div>
    {confirm.node}
    </>
  );
}
