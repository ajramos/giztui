import type { Dispatch, RefObject, SetStateAction } from "react";
import type { MessageDetail, Attachment, Invite } from "./api";
import { Icon } from "./Icons";
import { displayName, formatFull, formatICSDate, countMatches } from "./format";
import AiPanel from "./AiPanel";
import ChatPanel from "./ChatPanel";
import type { ChatBundle } from "./useChat";
import HtmlBody from "./HtmlBody";
import PlainBody from "./PlainBody";
import HighlightedText from "./HighlightedText";

// The reader body: RSVP bar, the two AI panels (summary + prompt), the in-message
// content-search box, and the mutually-exclusive body view (touch-up / thread /
// HTML / highlighted / plain). Presentational — App owns all state and the
// landmine refs (aiCache/imageOptIn); this receives values, scroll-target refs,
// and plain handlers. Behavior-preserving extraction of `<div className="reader-body">`.
export default function ReaderBody({
  detail,
  readerBodyRef,
  invite,
  rsvpBusy,
  onRespond,
  summaryPanelRef,
  summary,
  summarizing,
  summaryForId,
  onRegenerateSummary,
  onDismissSummary,
  promptPanelRef,
  promptLabel,
  promptResult,
  promptRunning,
  promptForId,
  onRegeneratePrompt,
  onDismissPrompt,
  csOpen,
  csQuery,
  csIndex,
  setCsQuery,
  setCsIndex,
  onCloseSearch,
  touchUpText,
  touchUpRef,
  onDismissTouchUp,
  loadingThread,
  threadMsgs,
  collapsedMsgs,
  setCollapsedMsgs,
  aiEnabled,
  onSummarizeThread,
  loadingDetail,
  viewHtml,
  loadRemote,
  onLoadImages,
  onAlwaysImages,
  attachments,
  chat,
}: {
  detail: MessageDetail;
  readerBodyRef: RefObject<HTMLDivElement>;
  invite: Invite | null;
  rsvpBusy: string;
  onRespond: (status: "accepted" | "tentative" | "declined") => void;
  summaryPanelRef: RefObject<HTMLDivElement>;
  summary: string | null;
  summarizing: boolean;
  summaryForId: string | null;
  onRegenerateSummary: () => void;
  onDismissSummary: () => void;
  promptPanelRef: RefObject<HTMLDivElement>;
  promptLabel: string;
  promptResult: string | null;
  promptRunning: boolean;
  promptForId: string | null;
  onRegeneratePrompt: () => void;
  onDismissPrompt: () => void;
  csOpen: boolean;
  csQuery: string;
  csIndex: number;
  setCsQuery: Dispatch<SetStateAction<string>>;
  setCsIndex: Dispatch<SetStateAction<number>>;
  onCloseSearch: () => void;
  touchUpText: string | null;
  touchUpRef: RefObject<HTMLDivElement>;
  onDismissTouchUp: () => void;
  loadingThread: boolean;
  threadMsgs: MessageDetail[] | null;
  collapsedMsgs: Set<string>;
  setCollapsedMsgs: Dispatch<SetStateAction<Set<string>>>;
  aiEnabled: boolean;
  onSummarizeThread: () => void;
  loadingDetail: boolean;
  viewHtml: boolean;
  loadRemote: boolean;
  onLoadImages: () => void;
  onAlwaysImages: () => void;
  attachments: Attachment[];
  chat: ChatBundle;
}) {
  return (
    <div className="reader-body" ref={readerBodyRef}>
      {invite?.isInvite && (
        <div className="rsvp-bar">
          <div className="rsvp-info">
            <span className="rsvp-title">
              {Icon.calendar} {invite.summary || "Calendar invite"}
            </span>
            {invite.dtStart && (
              <span className="rsvp-when muted">
                {formatICSDate(invite.dtStart)}
              </span>
            )}
          </div>
          <div className="rsvp-actions">
            <button
              className="rsvp-btn accept"
              disabled={!!rsvpBusy}
              onClick={() => onRespond("accepted")}
            >
              <span className="rsvp-ico">{Icon.check}</span>
              {rsvpBusy === "accepted" ? "…" : "Accept"}
            </button>
            <button
              className="rsvp-btn maybe"
              disabled={!!rsvpBusy}
              onClick={() => onRespond("tentative")}
            >
              <span className="rsvp-ico">{Icon.help}</span>
              {rsvpBusy === "tentative" ? "…" : "Maybe"}
            </button>
            <button
              className="rsvp-btn decline"
              disabled={!!rsvpBusy}
              onClick={() => onRespond("declined")}
            >
              <span className="rsvp-ico">{Icon.x}</span>
              {rsvpBusy === "declined" ? "…" : "Decline"}
            </button>
          </div>
        </div>
      )}
      <AiPanel
        ref={summaryPanelRef}
        title="AI summary"
        text={summary}
        // "Generating…" only for a summary launched on THIS message, so a run
        // started elsewhere never paints over the open email.
        generating={summarizing && summaryForId === detail.id}
        regenerateTitle="Regenerate (ignore cache)"
        onRegenerate={onRegenerateSummary}
        onDismiss={onDismissSummary}
      />
      <AiPanel
        ref={promptPanelRef}
        className="prompt-panel"
        title={promptLabel}
        text={promptResult}
        // "Generating…" only for a prompt launched on THIS message; a run started
        // elsewhere must not paint over the open email.
        generating={promptRunning && promptForId === detail.id}
        regenerateTitle="Regenerate (ignore the saved result and call the LLM again)"
        dismissTitle="Hide (kept for this email — re-run the prompt to show it again without regenerating)"
        onRegenerate={onRegeneratePrompt}
        onDismiss={onDismissPrompt}
      />
      {chat.chatOpen && chat.chatForId === detail.id && (
        <ChatPanel
          turns={chat.chatTurns}
          streaming={chat.chatStreaming}
          streamingText={chat.chatStreamingText}
          input={chat.chatInput}
          onInput={chat.setChatInput}
          onSend={chat.sendChat}
          onReset={chat.resetChat}
          onClose={chat.closeChat}
        />
      )}
      {csOpen && (
        <div className="content-search">
          <input
            autoFocus
            value={csQuery}
            placeholder="Find in message…"
            onChange={(e) => {
              setCsQuery(e.target.value);
              setCsIndex(0);
            }}
            onKeyDown={(e) => {
              const total = countMatches(detail.plainText || "", csQuery);
              if (e.key === "Escape") {
                onCloseSearch();
              } else if (e.key === "Enter") {
                e.preventDefault();
                if (total > 0)
                  setCsIndex((i) =>
                    e.shiftKey ? (i - 1 + total) % total : (i + 1) % total,
                  );
              }
            }}
          />
          <span className="cs-count">
            {csQuery
              ? `${
                  countMatches(detail.plainText || "", csQuery) === 0
                    ? 0
                    : csIndex + 1
                }/${countMatches(detail.plainText || "", csQuery)}`
              : ""}
          </span>
          <button
            className="tiny"
            title="Previous (Shift+Enter)"
            onClick={() => {
              const total = countMatches(detail.plainText || "", csQuery);
              if (total > 0) setCsIndex((i) => (i - 1 + total) % total);
            }}
          >
            ↑
          </button>
          <button
            className="tiny"
            title="Next (Enter)"
            onClick={() => {
              const total = countMatches(detail.plainText || "", csQuery);
              if (total > 0) setCsIndex((i) => (i + 1) % total);
            }}
          >
            ↓
          </button>
          <button className="ghost tiny" onClick={onCloseSearch}>
            ✕
          </button>
        </div>
      )}
      {touchUpText !== null ? (
        <div className="touchup" ref={touchUpRef}>
          <div className="touchup-head">
            <span>✦ Reformatted by AI</span>
            <button className="ghost tiny" onClick={onDismissTouchUp}>
              show original
            </button>
          </div>
          <pre className="plain">{touchUpText}</pre>
        </div>
      ) : loadingThread ? (
        <div className="placeholder">Loading conversation…</div>
      ) : threadMsgs ? (
        <div className="conversation">
          <div className="conv-head">
            <span>Conversation · {threadMsgs.length} messages</span>
            <span className="summary-head-actions">
              <button
                className="ghost tiny"
                onClick={() => setCollapsedMsgs(new Set())}
              >
                Expand all
              </button>
              <button
                className="ghost tiny"
                onClick={() =>
                  setCollapsedMsgs(new Set(threadMsgs.map((m) => m.id)))
                }
              >
                Collapse all
              </button>
              {aiEnabled && (
                <button
                  className="tiny"
                  disabled={summarizing}
                  onClick={onSummarizeThread}
                >
                  {summarizing ? "Summarizing…" : "✦ Summarize"}
                </button>
              )}
            </span>
          </div>
          {threadMsgs.map((m) => {
            const collapsed = collapsedMsgs.has(m.id);
            return (
              <div
                key={m.id}
                className={
                  "conv-msg" +
                  (m.unread ? " unread" : "") +
                  (collapsed ? " collapsed" : "")
                }
              >
                <button
                  className="conv-msg-head"
                  onClick={() =>
                    setCollapsedMsgs((prev) => {
                      const next = new Set(prev);
                      if (next.has(m.id)) next.delete(m.id);
                      else next.add(m.id);
                      return next;
                    })
                  }
                >
                  <span className="conv-caret">{collapsed ? "▸" : "▾"}</span>
                  <strong>{displayName(m.from)}</strong>
                  {collapsed && (
                    <span className="conv-snippet">
                      {(m.plainText || "").slice(0, 80)}
                    </span>
                  )}
                  <span className="conv-date muted">{formatFull(m.date)}</span>
                </button>
                {!collapsed && (
                  <pre className="plain">{m.plainText || "(empty)"}</pre>
                )}
              </div>
            );
          })}
        </div>
      ) : loadingDetail ? (
        <div className="placeholder">Loading…</div>
      ) : viewHtml && detail.html && detail.html.trim() ? (
        <div className="html-wrap">
          {!loadRemote && (
            <div className="remote-bar">
              Remote images blocked for privacy.
              <button className="tiny" onClick={onLoadImages}>
                Load images
              </button>
              <button
                className="tiny"
                title="Always load remote images (toggle with :images-always)"
                onClick={onAlwaysImages}
              >
                Always
              </button>
            </div>
          )}
          <HtmlBody
            html={detail.html}
            loadRemote={loadRemote}
            messageId={detail.id}
            attachments={attachments}
          />
        </div>
      ) : csOpen && csQuery ? (
        <pre className="plain">
          <HighlightedText
            text={detail.plainText || "(empty body)"}
            query={csQuery}
            activeIndex={csIndex}
          />
        </pre>
      ) : (
        <PlainBody text={detail.plainText || "(empty body)"} />
      )}
    </div>
  );
}
