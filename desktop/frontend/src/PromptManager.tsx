import { useEffect, useState } from "react";
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

  const reload = async () => {
    try {
      setPrompts(await backend.ListPrompts());
      onChanged();
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

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            if (editing) setEditing(null);
            else onClose();
          }
        }}
      >
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
              <div className="label-list">
                {prompts.length === 0 ? (
                  <div className="placeholder">No prompts yet</div>
                ) : (
                  prompts.map((p) => (
                    <div key={p.id} className="prompt-manage-row">
                      <button
                        className="prompt-row"
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
