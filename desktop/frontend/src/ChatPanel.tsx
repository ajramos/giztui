import { useEffect, useRef } from "react";
import Markdown from "./Markdown";
import { Icon } from "./Icons";
import type { ChatTurn } from "./useChat";

// ChatPanel is the "chat with this email" reader panel: a scrolling transcript
// of user/assistant turns plus a text input. Assistant text renders as Markdown;
// the in-flight reply streams into a pending bubble. Enter sends (Shift+Enter =
// newline), Escape closes.
export default function ChatPanel({
  turns,
  streaming,
  streamingText,
  input,
  onInput,
  onSend,
  onReset,
  onClose,
}: {
  turns: ChatTurn[];
  streaming: boolean;
  streamingText: string;
  input: string;
  onInput: (v: string) => void;
  onSend: () => void;
  onReset: () => void;
  onClose: () => void;
}) {
  const bodyRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Keep the newest turn / streaming text in view.
  useEffect(() => {
    bodyRef.current?.scrollTo({ top: bodyRef.current.scrollHeight });
  }, [turns, streamingText]);

  // Focus the input when the panel appears so you can type immediately.
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  return (
    <div className="summary-panel chat-panel">
      <div className="summary-head">
        <span className="summary-title">{Icon.chat} Chat with this email</span>
        <span className="summary-head-actions">
          <button className="ghost tiny" onClick={onReset} title="Clear the conversation">
            {Icon.trash} Reset
          </button>
          <button className="ghost tiny" onClick={onClose} title="Close (Esc)">
            ✕
          </button>
        </span>
      </div>

      <div className="chat-body" ref={bodyRef}>
        {turns.length === 0 && !streaming && (
          <div className="placeholder">
            Ask anything about this email — e.g. “what are the action items?”
          </div>
        )}
        {turns.map((t, i) => (
          <div key={i} className={"chat-turn chat-" + t.role}>
            {t.role === "assistant" ? (
              <Markdown text={t.text} />
            ) : (
              <span className="chat-user-text">{t.text}</span>
            )}
          </div>
        ))}
        {streaming && (
          <div className="chat-turn chat-assistant">
            {streamingText ? (
              <Markdown text={streamingText} />
            ) : (
              <span className="placeholder">Thinking…</span>
            )}
            <span className="caret">▍</span>
          </div>
        )}
      </div>

      <div className="chat-input-row">
        <textarea
          ref={inputRef}
          className="chat-input"
          rows={2}
          placeholder="Type a message… (Enter to send, Shift+Enter for newline)"
          value={input}
          onChange={(e) => onInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              e.preventDefault();
              onClose();
            } else if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              if (!streaming) onSend();
            }
          }}
        />
        <button
          className="chat-send"
          disabled={streaming || input.trim() === ""}
          onClick={onSend}
          title="Send (Enter)"
        >
          {Icon.send}
        </button>
      </div>
    </div>
  );
}
