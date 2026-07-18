import { useState } from "react";
import { backend } from "./api";

export interface ComposeInit {
  mode: "new" | "reply" | "draft";
  originalId?: string;
  draftId?: string;
  to?: string;
  subject?: string;
  body?: string;
  cc?: string;
}

export default function Compose({
  init,
  onClose,
  onSent,
}: {
  init: ComposeInit;
  onClose: () => void;
  onSent: (msg: string) => void;
}) {
  const [to, setTo] = useState(init.to ?? "");
  const [cc, setCc] = useState(init.cc ?? "");
  const [subject, setSubject] = useState(init.subject ?? "");
  const [body, setBody] = useState(init.body ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const isReply = init.mode === "reply";
  const isDraft = init.mode === "draft";
  const canSaveDraft = !isReply; // new or draft

  const ccList = () =>
    cc
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);

  const send = async () => {
    setBusy(true);
    setError("");
    try {
      if (isReply && init.originalId) {
        await backend.Reply(init.originalId, body, ccList());
      } else {
        await backend.SendMail(to, subject, body, ccList(), []);
        // A sent draft is no longer a draft.
        if (isDraft && init.draftId) {
          await backend.DeleteDraft(init.draftId).catch(() => undefined);
        }
      }
      onSent(isReply ? "Reply sent" : "Message sent");
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const saveDraft = async () => {
    setBusy(true);
    setError("");
    try {
      if (init.draftId) {
        await backend.UpdateDraft(init.draftId, to, subject, body, ccList());
      } else {
        await backend.SaveDraft(to, subject, body, ccList());
      }
      onSent("Draft saved");
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!init.draftId) return;
    setBusy(true);
    setError("");
    try {
      await backend.DeleteDraft(init.draftId);
      onSent("Draft deleted");
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const title = isReply ? "Reply" : isDraft ? "Edit draft" : "New message";

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
        }}
      >
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        <div className="modal-body">
          {isReply ? (
            <div className="field readonly">
              <label>To</label>
              <div className="ro-value">{init.to}</div>
            </div>
          ) : (
            <div className="field">
              <label>To</label>
              <input
                value={to}
                onChange={(e) => setTo(e.target.value)}
                placeholder="recipient@example.com"
                autoFocus
              />
            </div>
          )}
          <div className="field">
            <label>Cc</label>
            <input
              value={cc}
              onChange={(e) => setCc(e.target.value)}
              placeholder="comma-separated (optional)"
            />
          </div>
          {!isReply && (
            <div className="field">
              <label>Subject</label>
              <input value={subject} onChange={(e) => setSubject(e.target.value)} />
            </div>
          )}
          <div className="field grow">
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Write your message…"
              autoFocus={isReply}
            />
          </div>
        </div>
        <div className="modal-foot">
          {isDraft && (
            <button className="danger" onClick={() => void remove()} disabled={busy}>
              Delete
            </button>
          )}
          <span className="foot-spacer" />
          <button className="ghost" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          {canSaveDraft && (
            <button
              className="ghost"
              onClick={() => void saveDraft()}
              disabled={busy || (!body.trim() && !subject.trim())}
            >
              Save draft
            </button>
          )}
          <button onClick={() => void send()} disabled={busy || !body.trim()}>
            {busy ? "Working…" : "Send"}
          </button>
        </div>
      </div>
    </div>
  );
}
