import {
  useEffect,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import { useListNav } from "./useListNav";
import { backend, type Prompt, type PromptDetail } from "./api";

const EMPTY: PromptDetail = {
  id: 0,
  name: "",
  description: "",
  category: "",
  text: "",
};

// PromptManager is the desktop equivalent of the TUI's prompt configurator:
// create, edit, refine (via the LLM) and delete saved prompt templates.
export default function PromptManager({
  aiEnabled,
  onClose,
  onChanged,
}: {
  aiEnabled: boolean;
  onClose: () => void;
  onChanged: () => void;
}) {
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [editing, setEditing] = useState<PromptDetail | null>(null);
  const [busy, setBusy] = useState(false);
  const [refining, setRefining] = useState(false);
  const [error, setError] = useState("");

  // reload refreshes the list. onChanged is fired only by actual mutations
  // (save/remove) below, not here — so opening the manager doesn't look like an
  // edit to the parent (which invalidates caches on onChanged).
  const reload = async () => {
    try {
      setPrompts(await backend.ListPrompts());
    } catch (e) {
      setError(String(e));
    }
  };

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const edit = async (id: number) => {
    setError("");
    try {
      setEditing(await backend.GetPrompt(id));
    } catch (e) {
      setError(String(e));
    }
  };

  const save = async () => {
    if (!editing) return;
    setBusy(true);
    setError("");
    try {
      if (editing.id > 0) {
        await backend.UpdatePrompt(
          editing.id,
          editing.name,
          editing.description,
          editing.text,
          editing.category,
        );
      } else {
        await backend.CreatePrompt(
          editing.name,
          editing.description,
          editing.text,
          editing.category,
        );
      }
      setEditing(null);
      await reload();
      onChanged();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (id: number) => {
    setError("");
    try {
      await backend.DeletePrompt(id);
      await reload();
      onChanged();
    } catch (e) {
      setError(String(e));
    }
  };

  const refine = async () => {
    if (!editing) return;
    setRefining(true);
    setError("");
    try {
      const improved = await backend.RefinePromptText(editing.text);
      setEditing({ ...editing, text: improved });
    } catch (e) {
      setError(String(e));
    } finally {
      setRefining(false);
    }
  };

  // Keyboard-first list nav (WKWebView won't focus a bare div, so drive keys from
  // the window). useListNav owns the arrow/Enter movement; a capture-phase
  // listener adds delete/new and beats the App's own window handler so its
  // shortcuts don't fire underneath the manager.
  const nav = useListNav(prompts, {
    onEnter: (p) => void edit(p.id),
    onEscape: onClose,
  });
  const promptsRef = useRef(prompts);
  promptsRef.current = prompts;
  const activeRef = useRef(nav.active);
  activeRef.current = nav.active;
  const editingRef = useRef(editing);
  editingRef.current = editing;
  const navKeyRef = useRef(nav.onKeyDown);
  navKeyRef.current = nav.onKeyDown;

  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        if (editingRef.current) setEditing(null);
        else onClose();
        return;
      }
      // In the edit form, let the inputs/textarea handle their own keys.
      if (editingRef.current) return;
      const p = promptsRef.current[activeRef.current];
      if (e.key === "n" || e.key === "+") {
        e.preventDefault();
        e.stopImmediatePropagation();
        setEditing({ ...EMPTY });
        return;
      }
      if (e.key === "Enter" && p) {
        e.preventDefault();
        e.stopImmediatePropagation();
        void edit(p.id);
        return;
      }
      if ((e.key === "d" || e.key === "Delete" || e.key === "Backspace") && p) {
        e.preventDefault();
        e.stopImmediatePropagation();
        void remove(p.id);
        return;
      }
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.stopImmediatePropagation();
        navKeyRef.current(e as unknown as ReactKeyboardEvent);
      }
    };
    // Capture phase so we run before (and can stop) the App's window keydown.
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>{editing ? (editing.id ? "Edit prompt" : "New prompt") : "Manage prompts"}</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        {!editing ? (
          <>
            <div className="modal-body">
              <div className="label-list" ref={nav.listRef}>
                {prompts.length === 0 ? (
                  <div className="placeholder">No prompts yet</div>
                ) : (
                  prompts.map((p, i) => (
                    <div
                      key={p.id}
                      className="prompt-manage-row"
                      onMouseEnter={() => nav.setActiveHover(i)}
                    >
                      <button
                        className={"prompt-row" + (i === nav.active ? " nav-active" : "")}
                        onClick={() => void edit(p.id)}
                      >
                        <span className="prompt-name">{p.name}</span>
                        {p.description && (
                          <span className="prompt-desc">{p.description}</span>
                        )}
                      </button>
                      <button
                        className="ghost tiny danger"
                        title="Delete"
                        onClick={() => void remove(p.id)}
                      >
                        🗑
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
            <div className="modal-foot">
              <span className="foot-hint">
                ↑↓ move · Enter edit · d delete · n new · Esc close
              </span>
              <button onClick={() => setEditing({ ...EMPTY })}>＋ New prompt</button>
            </div>
          </>
        ) : (
          <>
            <div className="modal-body">
              <div className="field">
                <label>Name</label>
                <input
                  value={editing.name}
                  onChange={(e) => setEditing({ ...editing, name: e.target.value })}
                  placeholder="e.g. Extract action items"
                  autoFocus
                />
              </div>
              <div className="field">
                <label>Description</label>
                <input
                  value={editing.description}
                  onChange={(e) =>
                    setEditing({ ...editing, description: e.target.value })
                  }
                  placeholder="Short description"
                />
              </div>
              <div className="field">
                <label>Category</label>
                <input
                  value={editing.category}
                  onChange={(e) =>
                    setEditing({ ...editing, category: e.target.value })
                  }
                  placeholder="general"
                />
              </div>
              <div className="field">
                <label>
                  Prompt text
                  <span className="muted"> — use {"{{body}}"} for the email</span>
                </label>
                <textarea
                  className="prompt-text-area"
                  value={editing.text}
                  onChange={(e) => setEditing({ ...editing, text: e.target.value })}
                  placeholder="Summarize the following email:\n\n{{body}}"
                  rows={8}
                />
              </div>
            </div>
            <div className="modal-foot">
              <button className="ghost" onClick={() => setEditing(null)}>
                Cancel
              </button>
              {aiEnabled && (
                <button
                  className="ghost"
                  disabled={refining || !editing.text.trim()}
                  onClick={() => void refine()}
                >
                  {refining ? "Refining…" : "✦ Refine with AI"}
                </button>
              )}
              <button
                onClick={() => void save()}
                disabled={busy || !editing.name.trim() || !editing.text.trim()}
              >
                {busy ? "Saving…" : "Save"}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
