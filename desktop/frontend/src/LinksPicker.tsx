import { useEffect, useRef, useState } from "react";
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
  const modalRef = useRef<HTMLDivElement>(null);

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

  // Focus the modal so its keyboard handler fires without a click first.
  useEffect(() => {
    modalRef.current?.focus();
  }, []);

  const open = (l: Link) => {
    void backend.OpenURL(l.url).catch(() => undefined);
  };

  const nav = useListNav(links, { onEnter: open, onEscape: onClose });

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        ref={modalRef}
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (/^[1-9]$/.test(e.key)) {
            const i = Number(e.key) - 1;
            if (links[i]) open(links[i]);
            return;
          }
          nav.onKeyDown(e);
        }}
        tabIndex={-1}
      >
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
