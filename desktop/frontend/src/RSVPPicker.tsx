import { useEffect, useRef, type KeyboardEvent } from "react";
import { useListNav } from "./useListNav";

type Status = "accepted" | "tentative" | "declined";

const OPTS: { key: Status; icon: string; label: string; desc: string }[] = [
  { key: "accepted", icon: "✅", label: "Accept", desc: "I'll be there" },
  { key: "tentative", icon: "🤔", label: "Maybe", desc: "Maybe attending" },
  { key: "declined", icon: "❌", label: "Decline", desc: "Cannot attend" },
];

// A keyboard-navigable RSVP picker (the TUI's RSVP panel). Arrows/Enter or 1-3
// pick a response; Esc closes.
export default function RSVPPicker({
  summary,
  when,
  busy,
  onRespond,
  onClose,
}: {
  summary: string;
  when: string;
  busy: string;
  onRespond: (status: Status) => void;
  onClose: () => void;
}) {
  const nav = useListNav(OPTS, {
    onEnter: (o) => onRespond(o.key),
    onEscape: onClose,
  });

  // Window-level keys (WKWebView won't focus a bare div): 1-3 pick directly,
  // everything else delegates to the list nav. A ref keeps state fresh.
  const navKeyRef = useRef(nav.onKeyDown);
  navKeyRef.current = nav.onKeyDown;
  useEffect(() => {
    const handler = (e: globalThis.KeyboardEvent) => {
      const m = e.key.match(/^[1-3]$/);
      if (m) {
        e.preventDefault();
        onRespond(OPTS[Number(e.key) - 1].key);
        return;
      }
      navKeyRef.current(e as unknown as KeyboardEvent);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onRespond]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>📅 {summary || "Calendar invite"}</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          {when && <div className="rsvp-when muted">{when}</div>}
          <div className="label-list" ref={nav.listRef}>
            {OPTS.map((o, i) => (
              <button
                key={o.key}
                className={"prompt-row" + (i === nav.active ? " nav-active" : "")}
                disabled={!!busy}
                onMouseEnter={() => nav.setActive(i)}
                onClick={() => onRespond(o.key)}
              >
                <span className="prompt-name">
                  <span className="link-idx">[{i + 1}]</span> {o.icon} {o.label}
                </span>
                <span className="prompt-desc">
                  {busy === o.key ? "Sending…" : o.desc}
                </span>
              </button>
            ))}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">↑↓ move · Enter / 1-3 select · Esc close</span>
        </div>
      </div>
    </div>
  );
}
