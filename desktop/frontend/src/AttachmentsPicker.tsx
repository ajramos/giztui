import { useEffect } from "react";
import { useListNav } from "./useListNav";
import type { Attachment } from "./api";

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

// AttachmentsPicker is the desktop equivalent of the TUI's PickerAttachments:
// a keyboard-navigable list of the message's attachments. Arrow to one, Enter
// (or 1-9) to download it, Escape to close. The reader also shows the same
// attachments as inline chips for the mouse; this is the keyboard-first path.
export default function AttachmentsPicker({
  attachments,
  busy,
  onDownload,
  onClose,
}: {
  attachments: Attachment[];
  busy: boolean;
  onDownload: (att: Attachment) => void;
  onClose: () => void;
}) {
  const nav = useListNav(attachments, {
    onEnter: (a) => onDownload(a),
    onEscape: onClose,
    windowKeys: true,
  });

  // 1-9 quick-download, matching the TUI's numbered quick-select.
  useEffect(() => {
    const h = (e: globalThis.KeyboardEvent) => {
      if (e.key >= "1" && e.key <= "9") {
        const i = Number(e.key) - 1;
        if (i < attachments.length) {
          e.preventDefault();
          onDownload(attachments[i]);
        }
      }
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [attachments, onDownload]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>📎 Attachments</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          <div className="label-list" ref={nav.listRef}>
            {attachments.length === 0 ? (
              <div className="placeholder">No attachments</div>
            ) : (
              attachments.map((a, i) => (
                <button
                  key={a.attachmentId}
                  className={"label-row" + (i === nav.active ? " nav-active" : "")}
                  disabled={busy}
                  onMouseEnter={() => nav.setActiveHover(i)}
                  onClick={() => onDownload(a)}
                  title={`${a.mimeType} · ${fmtSize(a.size)}`}
                >
                  {i < 9 && <span className="link-idx">[{i + 1}]</span>}
                  <span className="name">{a.filename}</span>
                  <span className="attach-size">{fmtSize(a.size)}</span>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            ↑↓ move · 1-9 quick · Enter download · Esc close
          </span>
        </div>
      </div>
    </div>
  );
}
