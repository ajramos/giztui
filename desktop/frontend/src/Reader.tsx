import type { Dispatch, RefObject, SetStateAction } from "react";
import type { MessageDetail, Attachment, Invite } from "./api";
import type { ChatBundle } from "./useChat";
import { Icon } from "./Icons";
import { displayName, emailAddr, formatFull, formatSize } from "./format";
import ReaderToolbar from "./ReaderToolbar";
import ReaderBody from "./ReaderBody";

// The right pane. Owns the `<main className="reader">` shell: the header block
// (subject/meta/labels), the attachment bar, and composes ReaderToolbar +
// ReaderBody. Presentational — App keeps all state and the landmine refs; this
// is a behavior-preserving extraction of the reader `<main>`. State stays in App
// so this is a fat pass-through for now; it collapses once a useReader hook lands.
export default function Reader(props: {
  detail: MessageDetail | null;
  readerFocused: boolean;
  onFocusReader: () => void;
  headersHidden: boolean;
  headersExpanded: boolean;
  showToolbar: boolean;
  busy: boolean;
  attachments: Attachment[];
  onDownloadAttachment: (att: Attachment) => void;
  // toolbar
  aiEnabled: boolean;
  aiPromptsEnabled: boolean;
  obsidianOn: boolean;
  slackOn: boolean;
  threadingOn: boolean;
  hasThread: boolean;
  viewHtml: boolean;
  summarizing: boolean;
  chat: ChatBundle;
  promptRunning: boolean;
  promptGenerating: boolean;
  generatingReply: boolean;
  touchingUp: boolean;
  touchUpShown: boolean;
  onReply: () => void;
  onForward: () => void;
  onLabels: () => void;
  onArchive: () => void;
  onTrash: () => void;
  onToggleRead: () => void;
  onToggleHtml: () => void;
  onToggleThread: () => void;
  onSummarize: () => void;
  onApplyPrompt: () => void;
  onDraftReply: () => void;
  onTouchUp: () => void;
  onSuggestLabels: () => void;
  onMove: () => void;
  onSearchSender: () => void;
  onLinks: () => void;
  onObsidian: () => void;
  onSlack: () => void;
  onSave: () => void;
  onSaveRaw: () => void;
  onToggleHeaderBlock: () => void;
  onToggleFullHeaders: () => void;
  onOpenGmail: () => void;
  // body
  readerBodyRef: RefObject<HTMLDivElement>;
  invite: Invite | null;
  rsvpBusy: string;
  rsvpDone: string;
  onRespond: (status: "accepted" | "tentative" | "declined") => void;
  summaryPanelRef: RefObject<HTMLDivElement>;
  summary: string | null;
  summaryForId: string | null;
  onRegenerateSummary: () => void;
  onDismissSummary: () => void;
  promptPanelRef: RefObject<HTMLDivElement>;
  promptLabel: string;
  promptResult: string | null;
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
  onSummarizeThread: () => void;
  loadingDetail: boolean;
  loadRemote: boolean;
  onLoadImages: () => void;
  onAlwaysImages: () => void;
}) {
  const { detail } = props;
  return (
    <main
      className={"reader" + (props.readerFocused && detail ? " reader-focused" : "")}
      onMouseDown={() => {
        // Clicking / selecting inside the reader hands it the keyboard, so arrows
        // scroll the body (fixes: selecting text didn't grab focus).
        if (detail) props.onFocusReader();
      }}
    >
      {detail ? (
        <>
          <div className="reader-head">
            <h2>{detail.subject || "(no subject)"}</h2>
            <div className="meta">
              {!props.headersHidden && (
                <>
                  <div>
                    <strong>{displayName(detail.from)}</strong>{" "}
                    <span className="muted">{emailAddr(detail.from)}</span>
                  </div>
                  <div className="muted">to {detail.to}</div>
                  <div className="muted">{formatFull(detail.date)}</div>
                  {props.headersExpanded && (
                    <div className="headers-detail">
                      {detail.cc && <div className="muted">cc {detail.cc}</div>}
                      <div className="muted">thread {detail.threadId}</div>
                      <div className="muted">id {detail.id}</div>
                    </div>
                  )}
                </>
              )}
              {detail.labels.length > 0 && (
                <div className="labels reader-labels">
                  {detail.labels.map((l) => (
                    <span key={l} className="label-chip">
                      {l}
                    </span>
                  ))}
                </div>
              )}
            </div>
            {props.showToolbar && (
              <ReaderToolbar
                detail={detail}
                busy={props.busy}
                aiEnabled={props.aiEnabled}
                aiPromptsEnabled={props.aiPromptsEnabled}
                obsidianOn={props.obsidianOn}
                slackOn={props.slackOn}
                threadingOn={props.threadingOn}
                hasThread={props.hasThread}
                viewHtml={props.viewHtml}
                summarizing={props.summarizing}
                promptRunning={props.promptRunning}
                generatingReply={props.generatingReply}
                touchingUp={props.touchingUp}
                touchUpShown={props.touchUpShown}
                headersHidden={props.headersHidden}
                headersExpanded={props.headersExpanded}
                onReply={props.onReply}
                onForward={props.onForward}
                onLabels={props.onLabels}
                onArchive={props.onArchive}
                onTrash={props.onTrash}
                onToggleRead={props.onToggleRead}
                onToggleHtml={props.onToggleHtml}
                onToggleThread={props.onToggleThread}
                onSummarize={props.onSummarize}
                onApplyPrompt={props.onApplyPrompt}
                onDraftReply={props.onDraftReply}
                onTouchUp={props.onTouchUp}
                onSuggestLabels={props.onSuggestLabels}
                onMove={props.onMove}
                onSearchSender={props.onSearchSender}
                onLinks={props.onLinks}
                onObsidian={props.onObsidian}
                onSlack={props.onSlack}
                onSave={props.onSave}
                onSaveRaw={props.onSaveRaw}
                onToggleHeaderBlock={props.onToggleHeaderBlock}
                onToggleFullHeaders={props.onToggleFullHeaders}
                onOpenGmail={props.onOpenGmail}
              />
            )}
            {props.attachments.length > 0 && (
              <div className="attach-bar">
                {props.attachments.map((att) => (
                  <button
                    key={att.attachmentId}
                    className="attach-chip"
                    disabled={props.busy}
                    title={`${att.mimeType} · ${formatSize(att.size)}`}
                    onClick={() => props.onDownloadAttachment(att)}
                  >
                    {Icon.paperclip} {att.filename}
                    <span className="attach-size">{formatSize(att.size)}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
          <ReaderBody
            detail={detail}
            chat={props.chat}
            readerBodyRef={props.readerBodyRef}
            invite={props.invite}
            rsvpBusy={props.rsvpBusy}
            rsvpDone={props.rsvpDone}
            onRespond={props.onRespond}
            summaryPanelRef={props.summaryPanelRef}
            summary={props.summary}
            summarizing={props.summarizing}
            summaryForId={props.summaryForId}
            onRegenerateSummary={props.onRegenerateSummary}
            onDismissSummary={props.onDismissSummary}
            promptPanelRef={props.promptPanelRef}
            promptLabel={props.promptLabel}
            promptResult={props.promptResult}
            promptGenerating={props.promptGenerating}
            onRegeneratePrompt={props.onRegeneratePrompt}
            onDismissPrompt={props.onDismissPrompt}
            csOpen={props.csOpen}
            csQuery={props.csQuery}
            csIndex={props.csIndex}
            setCsQuery={props.setCsQuery}
            setCsIndex={props.setCsIndex}
            onCloseSearch={props.onCloseSearch}
            touchUpText={props.touchUpText}
            touchUpRef={props.touchUpRef}
            onDismissTouchUp={props.onDismissTouchUp}
            loadingThread={props.loadingThread}
            threadMsgs={props.threadMsgs}
            collapsedMsgs={props.collapsedMsgs}
            setCollapsedMsgs={props.setCollapsedMsgs}
            aiEnabled={props.aiEnabled}
            onSummarizeThread={props.onSummarizeThread}
            loadingDetail={props.loadingDetail}
            viewHtml={props.viewHtml}
            loadRemote={props.loadRemote}
            onLoadImages={props.onLoadImages}
            onAlwaysImages={props.onAlwaysImages}
            attachments={props.attachments}
          />
        </>
      ) : (
        <div className="empty-reader">
          <p>Select a message to read it here.</p>
          <p className="muted">
            Press <kbd>?</kbd> for keyboard shortcuts.
          </p>
        </div>
      )}
    </main>
  );
}
