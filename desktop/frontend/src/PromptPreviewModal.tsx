import { useEffect, useRef } from "react";

// The analyzer-prompt preview (opened by :action-plan prompt / the plan's `p`).
// A long scrollable <pre>; WKWebView won't focus the bare modal body, so arrows/
// PageUp/etc. are driven from a window-level listener while it's mounted. Escape
// is handled by the App's global modal chain. Self-contained: the scroll ref +
// effect live here now.
export default function PromptPreviewModal({
  text,
  onClose,
}: {
  text: string;
  onClose: () => void;
}) {
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      const el = bodyRef.current;
      if (!el) return;
      const line = 48;
      const page = el.clientHeight * 0.9;
      let dy = 0;
      switch (e.key) {
        case "ArrowDown": case "j": dy = line; break;
        case "ArrowUp": case "k": dy = -line; break;
        case "PageDown": case " ": dy = page; break;
        case "PageUp": dy = -page; break;
        case "Home": el.scrollTo({ top: 0 }); e.preventDefault(); return;
        case "End": el.scrollTo({ top: el.scrollHeight }); e.preventDefault(); return;
        default: return;
      }
      el.scrollBy({ top: dy });
      e.preventDefault();
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, []);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Analyzer prompt</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body" ref={bodyRef}>
          <pre className="summary-text">{text}</pre>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">↑↓ scroll · Esc close</span>
          <button onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}
