import { useState } from "react";
import { Icon } from "./Icons";

// ObsidianDialog is the desktop's "send to Obsidian" prompt (TUI parity): an
// optional comment/pre-message that the ingest renders into the note as
// "> **Note:** …". Keyboard-first — the comment input holds focus (WKWebView
// focuses inputs reliably); Enter ingests, Escape closes.
export default function ObsidianDialog({
  onSend,
  onClose,
}: {
  onSend: (comment: string) => void;
  onClose: () => void;
}) {
  const [comment, setComment] = useState("");

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3 className="head-with-ico">
            <span className="head-ico">{Icon.cloud}</span>
            Send to Obsidian
          </h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <input
            className="slack-premessage"
            type="text"
            placeholder="Optional note (added to the top of the ingested file)…"
            value={comment}
            autoFocus
            onChange={(e) => setComment(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                e.stopPropagation();
                onSend(comment.trim());
              } else if (e.key === "Escape") {
                e.preventDefault();
                e.stopPropagation();
                onClose();
              }
            }}
          />
        </div>
        <div className="modal-foot">
          <span className="foot-hint">Enter send · Esc close</span>
        </div>
      </div>
    </div>
  );
}
