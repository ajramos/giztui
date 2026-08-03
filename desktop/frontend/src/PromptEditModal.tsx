import { useState } from "react";
import { Icon } from "./Icons";
import { backend, type PromptDetail } from "./api";

// PromptEditModal is the shared create/edit form for a prompt — name,
// description, category, prompt text, plus an optional LLM "Refine". It mirrors
// EditQueryModal: it seeds its fields from the given prompt, stops Escape and
// Cmd/Ctrl+Enter from reaching the picker underneath (so those keys act on this
// modal only), and hands the edited values back via onSave. The caller decides
// create vs update from the id (0 = new). Used by PromptsPicker so prompts get
// the same inline edit experience as saved searches.
export default function PromptEditModal({
  prompt,
  aiEnabled,
  onSave,
  onClose,
}: {
  prompt: PromptDetail;
  aiEnabled: boolean;
  onSave: (detail: PromptDetail) => void | Promise<void>;
  onClose: () => void;
}) {
  const [draft, setDraft] = useState<PromptDetail>(prompt);
  const [busy, setBusy] = useState(false);
  const [refining, setRefining] = useState(false);
  const [error, setError] = useState("");

  const canSave = draft.name.trim() !== "" && draft.text.trim() !== "";
  const submit = async () => {
    if (!canSave || busy) return;
    setBusy(true);
    setError("");
    try {
      await onSave({ ...draft, name: draft.name.trim() });
    } catch (e) {
      setError(String(e));
      setBusy(false);
    }
  };
  const refine = async () => {
    setRefining(true);
    setError("");
    try {
      setDraft({ ...draft, text: await backend.RefinePromptText(draft.text) });
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
            e.stopPropagation();
            onClose();
          } else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.stopPropagation();
            void submit();
          }
        }}
      >
        <div className="modal-head">
          <h3>{prompt.id ? "Edit prompt" : "New prompt"}</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        <div className="modal-body">
          <div className="field">
            <label>Name</label>
            <input
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              placeholder="e.g. Extract action items"
              autoFocus
            />
          </div>
          <div className="field">
            <label>Description</label>
            <input
              value={draft.description}
              onChange={(e) => setDraft({ ...draft, description: e.target.value })}
              placeholder="Short description"
            />
          </div>
          <div className="field">
            <label>Category</label>
            <input
              value={draft.category}
              onChange={(e) => setDraft({ ...draft, category: e.target.value })}
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
              value={draft.text}
              onChange={(e) => setDraft({ ...draft, text: e.target.value })}
              placeholder="Summarize the following email:\n\n{{body}}"
              rows={8}
            />
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">⌘/Ctrl+Enter save · Esc cancel</span>
          <button className="ghost" onClick={onClose}>
            Cancel
          </button>
          {aiEnabled && (
            <button
              className="ghost"
              disabled={refining || !draft.text.trim()}
              onClick={() => void refine()}
            >
              {refining ? "Refining…" : <>{Icon.summarize} Refine with AI</>}
            </button>
          )}
          <button onClick={() => void submit()} disabled={busy || !canSave}>
            {busy ? "Saving…" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
