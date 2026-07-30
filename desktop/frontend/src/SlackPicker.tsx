import { useEffect, useRef, useState } from "react";
import { backend } from "./api";
import type { SlackChannel } from "./apiTypes";
import { Icon } from "./Icons";

// SlackPicker is the desktop's "forward to Slack" dialog (TUI parity): pick a
// channel and add an optional pre-message. Keyboard-first and focus-independent
// per the desktop picker conventions — a window listener drives channel selection
// (arrows / 1-9) while the pre-message input, when focused, owns typing + Enter.
//
// The channel list defaults to the configured default channel. Enter sends to the
// selected channel; Escape closes. The actual send + toast lives in the caller's
// forwardSlack (which also honors the configured format_style on the backend).
export default function SlackPicker({
  onSend,
  onClose,
}: {
  onSend: (channelID: string, message: string) => void;
  onClose: () => void;
}) {
  const [channels, setChannels] = useState<SlackChannel[]>([]);
  const [active, setActive] = useState(0);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);

  // Fresh values for the window listener (registered once).
  const activeRef = useRef(0);
  const channelsRef = useRef<SlackChannel[]>([]);
  const messageRef = useRef("");
  activeRef.current = active;
  channelsRef.current = channels;
  messageRef.current = message;

  useEffect(() => {
    let alive = true;
    void backend
      .SlackChannels()
      .then((cs) => {
        if (!alive) return;
        setChannels(cs);
        const def = Math.max(0, cs.findIndex((c) => c.default));
        setActive(def);
        setLoading(false);
      })
      .catch(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, []);

  const send = () => {
    const ch = channelsRef.current[activeRef.current];
    if (ch) onSend(ch.id, messageRef.current.trim());
  };

  // Window-level keys (WKWebView won't focus a bare div). Escape always closes.
  // Digits/arrows select a channel — but only when the pre-message input is NOT
  // the target, so typing "1" into the message doesn't jump channels. Enter from
  // the window (input not focused) sends.
  useEffect(() => {
    const handler = (e: globalThis.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
        return;
      }
      const inInput =
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement;
      if (inInput) return; // the input handles its own Enter/typing
      const n = channelsRef.current.length;
      if (e.key === "Enter") {
        e.preventDefault();
        send();
      } else if (/^[1-9]$/.test(e.key)) {
        const i = Number(e.key) - 1;
        if (i < n) {
          e.preventDefault();
          setActive(i);
        }
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        setActive((a) => Math.min(n - 1, a + 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setActive((a) => Math.max(0, a - 1));
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [onClose]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3 className="head-with-ico">
            <span className="head-ico">{Icon.slack}</span>
            Forward to Slack
          </h3>
          <button className="ghost" onClick={onClose}>
            ✕
          </button>
        </div>
        <div className="modal-body">
          {loading ? (
            <div className="muted">Loading channels…</div>
          ) : channels.length === 0 ? (
            <div className="muted">No Slack channels configured.</div>
          ) : (
            <>
              <div className="label-list slack-channel-list">
                {channels.map((c, i) => (
                  <button
                    key={c.id || c.name}
                    className={"prompt-row" + (i === active ? " nav-active" : "")}
                    onMouseEnter={() => setActive(i)}
                    onClick={() => {
                      setActive(i);
                      onSend(c.id, message.trim());
                    }}
                  >
                    <span className="prompt-name">
                      <span className="link-idx">[{i + 1}]</span>
                      <span className="rule-ico">{Icon.slack}</span> {c.name}
                      {c.default ? <span className="muted"> · default</span> : null}
                    </span>
                    {c.description ? (
                      <span className="prompt-desc">{c.description}</span>
                    ) : null}
                  </button>
                ))}
              </div>
              <input
                className="slack-premessage"
                type="text"
                placeholder="Optional pre-message (e.g. heads up on this…)"
                value={message}
                autoFocus
                onChange={(e) => setMessage(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    e.stopPropagation();
                    send();
                  } else if (e.key === "Escape") {
                    e.preventDefault();
                    e.stopPropagation();
                    onClose();
                  }
                }}
              />
            </>
          )}
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            ↑↓ / 1-9 channel · Enter send · Esc close
          </span>
        </div>
      </div>
    </div>
  );
}
