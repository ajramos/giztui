import { useEffect, useRef, useState, type KeyboardEvent } from "react";
import { backend, type Link } from "./api";
import { useListNav } from "./useListNav";

export default function LinksPicker({
  messageId,
  onClose,
}: {
  messageId: string;
  onClose: () => void;
}) {
  const [links, setLinks] = useState<Link[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void (async () => {
      try {
        setLinks(await backend.ListLinks(messageId));
      } catch (e) {
        setError(String(e));
      } finally {
        setLoading(false);
      }
    })();
  }, [messageId]);

  const open = (l: Link) => {
    void backend.OpenURL(l.url).catch(() => undefined);
  };

  const nav = useListNav(links, { onEnter: open, onEscape: onClose });

  // This picker has no text input to hold focus, and WKWebView won't reliably
  // focus a bare div — so drive the keyboard from a window listener instead.
  // The app's global handler bows out while a picker is open, so there's no
  // clash. Refs keep the listener bound once while always seeing fresh state.
  const linksRef = useRef(links);
  linksRef.current = links;
  const navKeyRef = useRef(nav.onKeyDown);
  navKeyRef.current = nav.onKeyDown;
  useEffect(() => {
    const handler = (e: globalThis.KeyboardEvent) => {
      if (/^[1-9]$/.test(e.key)) {
        const l = linksRef.current[Number(e.key) - 1];
        if (l) {
          e.preventDefault();
          open(l);
        }
        return;
      }
      navKeyRef.current(e as unknown as KeyboardEvent);
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Links in message</h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        {error && <div className="error-banner">{error}</div>}
        <div className="modal-body">
          <div className="label-list" ref={nav.listRef}>
            {loading ? (
              <div className="placeholder">Loading…</div>
            ) : links.length === 0 ? (
              <div className="placeholder">No links</div>
            ) : (
              links.map((l, i) => (
                <button
                  key={l.index}
                  className={"prompt-row" + (i === nav.active ? " nav-active" : "")}
                  onMouseEnter={() => nav.setActive(i)}
                  onClick={() => open(l)}
                >
                  <span className="prompt-name">
                    <span className="link-idx">[{i + 1}]</span> {l.text || l.url}
                  </span>
                  <span className="prompt-desc">{l.url}</span>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="modal-foot">
          <span className="foot-hint">↑↓ move · Enter / 1-9 open · Esc close</span>
        </div>
      </div>
    </div>
  );
}
