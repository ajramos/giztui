import { useEffect, useState } from "react";
import { backend, type Link } from "./api";

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
  const [active, setActive] = useState(0);

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

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal narrow"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => {
          if (e.key === "Escape") onClose();
          else if (e.key === "ArrowDown") {
            e.preventDefault();
            setActive((i) => Math.min(links.length - 1, i + 1));
          } else if (e.key === "ArrowUp") {
            e.preventDefault();
            setActive((i) => Math.max(0, i - 1));
          } else if (e.key === "Enter") {
            e.preventDefault();
            if (links[active]) open(links[active]);
          } else if (/^[1-9]$/.test(e.key)) {
            const i = Number(e.key) - 1;
            if (links[i]) open(links[i]);
          }
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
          <div className="label-list">
            {loading ? (
              <div className="placeholder">Loading…</div>
            ) : links.length === 0 ? (
              <div className="placeholder">No links</div>
            ) : (
              links.map((l, i) => (
                <button
                  key={l.index}
                  className={"prompt-row" + (i === active ? " active" : "")}
                  onMouseEnter={() => setActive(i)}
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
          <span className="foot-hint">Enter / 1-9 to open · Esc to close</span>
        </div>
      </div>
    </div>
  );
}
