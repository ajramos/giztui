import { useEffect } from "react";

// ConfirmDialog is the shared confirm step before a destructive action, so the
// desktop matches the TUI's "are you sure?" pattern (the TUI arms a two-press
// confirm; the GUI shows this small dialog). Keyboard-first and focus-independent
// like the pickers: a window-level capture listener handles Enter/Escape (a bare
// modal div won't focus in WKWebView) and stops the event so the picker
// underneath doesn't also act on it. Enter confirms, Escape cancels.
export default function ConfirmDialog({
  message,
  confirmLabel = "Delete",
  onConfirm,
  onCancel,
}: {
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        onCancel();
      } else if (e.key === "Enter") {
        e.preventDefault();
        e.stopImmediatePropagation();
        onConfirm();
      }
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, [onConfirm, onCancel]);

  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal narrow confirm" onClick={(e) => e.stopPropagation()}>
        <div className="modal-body">
          <p className="confirm-msg">{message}</p>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">Enter confirm · Esc cancel</span>
          <button className="ghost" onClick={onCancel}>
            Cancel
          </button>
          <button className="danger" onClick={onConfirm}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
