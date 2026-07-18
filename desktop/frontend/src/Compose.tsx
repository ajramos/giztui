import { useState } from "react";
import { backend } from "./api";

export interface ComposeInit {
  mode: "new" | "reply";
  originalId?: string;
  to?: string;
  subject?: string;
  body?: string;
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
  const [cc, setCc] = useState("");
  const [subject, setSubject] = useState(init.subject ?? "");
  const [body, setBody] = useState(init.body ?? "");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState("");

  const isReply = init.mode === "reply";

  const send = async () => {
    setSending(true);
    setError("");
    try {
      const ccList = cc
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      if (isReply && init.originalId) {
        await backend.Reply(init.originalId, body, ccList);
      } else {
        await backend.SendMail(to, subject, body, ccList, []);
      }
      onSent(isReply ? "Reply sent" : "Message sent");
    } catch (e) {
      setError(String(e));
    } finally {
      setSending(false);
    }
  };

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
          <h3>{isReply ? "Reply" : "New message"}</h3>
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
          <button className="ghost" onClick={onClose} disabled={sending}>
            Cancel
          </button>
          <button onClick={() => void send()} disabled={sending || !body.trim()}>
            {sending ? "Sending…" : "Send"}
          </button>
        </div>
      </div>
    </div>
  );
}
