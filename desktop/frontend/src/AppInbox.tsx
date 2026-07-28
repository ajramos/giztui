import type { Dispatch, MutableRefObject, RefObject, SetStateAction } from "react";
import MessageList from "./MessageList";
import Reader from "./Reader";
import TopBar from "./TopBar";
import { replyInit, forwardInit } from "./compose";
import type { ComposeInit } from "./Compose";
import type { AiCacheEntry } from "./useAiActions";
import type {
  AccountInfo,
  Attachment,
  DraftSummary,
  Invite,
  KeyMap,
  MessageDetail,
  MessageSummary,
  Prompt,
} from "./api";

const PAGE_SIZE = 50;

// The inbox surface split off App's render: the top bar (search / accounts /
// bulk / drafts / toolbar), the error+toast banners, and the body (message list
// + reader). Purely presentational — App owns all state and passes it in; the
// JSX (including its inline handlers) is a verbatim move.
export interface AppInboxProps {
  // top bar
  query: string;
  setQuery: (v: string) => void;
  localFilter: boolean;
  setLocalFilter: (v: boolean) => void;
  searchRef: RefObject<HTMLInputElement>;
  keymap: KeyMap;
  activeQuery: string;
  applyLocalFilter: (q: string) => void;
  load: (q: string) => Promise<void>;
  setMessages: (v: MessageSummary[]) => void;
  fullMessagesRef: MutableRefObject<MessageSummary[]>;
  setAdvOpen: (v: boolean) => void;
  accounts: AccountInfo[];
  account: string;
  switching: boolean;
  accountsOpen: boolean;
  setAccountsOpen: (v: boolean) => void;
  switchAccount: (a: AccountInfo) => Promise<void>;
  undoLabel: string;
  runUndo: () => Promise<void>;
  setCompose: (v: ComposeInit | null) => void;
  draftsView: boolean;
  setDraftsView: (v: boolean) => void;
  openDrafts: () => void;
  bulkMode: boolean;
  exitBulk: () => void;
  setBulkMode: (v: boolean) => void;
  selectedId: string | null;
  messages: MessageSummary[];
  setSelectedId: (v: string | null) => void;
  savedQueriesOn: boolean;
  openQueries: () => Promise<void>;
  showToolbar: boolean;
  toggleToolbar: () => void;
  autoRefresh: boolean;
  autoRefreshSecs: number;
  toggleAutoRefresh: () => void;
  setShowHelp: (v: boolean) => void;
  // banners
  error: string;
  toast: string;
  // message list
  setReaderFocused: (v: boolean) => void;
  drafts: DraftSummary[];
  loadingDrafts: boolean;
  loadDrafts: () => Promise<void>;
  openDraft: (d: DraftSummary) => void;
  pendingNew: MessageSummary[];
  showPendingNew: () => void;
  loadingList: boolean;
  nextToken: string;
  selected: Set<string>;
  busy: boolean;
  bulkProgress: string;
  bulkAction: (action: "archive" | "trash" | "read" | "unread") => Promise<void>;
  setBulkLabels: (v: boolean) => void;
  setBulkMove: (v: boolean) => void;
  setSelected: Dispatch<SetStateAction<Set<string>>>;
  toggleSelect: (id: string) => void;
  openMessage: (m: MessageSummary) => void;
  loadingMore: boolean;
  loadMore: () => Promise<void>;
  // reader
  detail: MessageDetail | null;
  readerFocused: boolean;
  headersHidden: boolean;
  headersExpanded: boolean;
  attachments: Attachment[];
  downloadAttachment: (att: Attachment) => void;
  aiEnabled: boolean;
  aiPromptsEnabled: boolean;
  obsidianOn: boolean;
  slackOn: boolean;
  threadingOn: boolean;
  threadMsgs: MessageDetail[] | null;
  viewHtml: boolean;
  summarizing: boolean;
  promptRunning: boolean;
  generatingReply: boolean;
  touchingUp: boolean;
  touchUpText: string | null;
  setLabelsFor: (v: string | null) => void;
  doAction: (action: "archive" | "trash" | "read" | "unread", id: string) => Promise<void>;
  setViewHtml: Dispatch<SetStateAction<boolean>>;
  toggleThread: () => Promise<void>;
  summarize: (id: string, force?: boolean) => Promise<void>;
  setPromptsOpen: (v: boolean) => void;
  generateReply: (d: MessageDetail) => Promise<void>;
  setTouchUpText: (v: string | null) => void;
  touchUp: (id: string) => Promise<void>;
  openSuggest: (id: string) => Promise<void>;
  setMoveFor: (v: string | null) => void;
  quickSearch: (kind: "from" | "to" | "subject", d: MessageDetail) => void;
  setLinksFor: (v: string | null) => void;
  sendObsidian: (id: string) => void;
  forwardSlack: (id: string) => void;
  saveMessage: (id: string) => void;
  saveRawMessage: (id: string) => void;
  setHeadersHidden: Dispatch<SetStateAction<boolean>>;
  setHeadersExpanded: Dispatch<SetStateAction<boolean>>;
  openInGmail: (id: string) => void;
  readerBodyRef: RefObject<HTMLDivElement>;
  invite: Invite | null;
  rsvpBusy: string;
  respondInvite: (id: string, status: "accepted" | "tentative" | "declined") => Promise<void>;
  summaryPanelRef: RefObject<HTMLDivElement>;
  summary: string | null;
  summaryForId: string | null;
  dismissSummary: (id: string | null) => void;
  promptPanelRef: RefObject<HTMLDivElement>;
  promptLabel: string;
  promptResult: string | null;
  promptForId: string | null;
  aiCache: MutableRefObject<Map<string, AiCacheEntry>>;
  runPrompt: (prompt: Prompt, force?: boolean) => Promise<void>;
  dismissPrompt: (id: string | null) => void;
  csOpen: boolean;
  csQuery: string;
  csIndex: number;
  setCsQuery: Dispatch<SetStateAction<string>>;
  setCsIndex: Dispatch<SetStateAction<number>>;
  setCsOpen: (v: boolean) => void;
  touchUpRef: RefObject<HTMLDivElement>;
  dismissTouchUp: (id: string | null) => void;
  loadingThread: boolean;
  collapsedMsgs: Set<string>;
  setCollapsedMsgs: Dispatch<SetStateAction<Set<string>>>;
  summarizeThread: () => Promise<void>;
  loadingDetail: boolean;
  loadRemote: boolean;
  setLoadRemote: (v: boolean) => void;
  imageOptIn: MutableRefObject<Set<string>>;
  setAlwaysImagesOn: (on: boolean) => void;
}

export default function AppInbox(p: AppInboxProps) {
  const {
    query, setQuery, localFilter, setLocalFilter, searchRef, keymap, activeQuery,
    applyLocalFilter, load, setMessages, fullMessagesRef, setAdvOpen, accounts, account,
    switching, accountsOpen, setAccountsOpen, switchAccount, undoLabel, runUndo, setCompose,
    draftsView, setDraftsView, openDrafts, bulkMode, exitBulk, setBulkMode, selectedId, messages,
    setSelectedId, savedQueriesOn, openQueries, showToolbar, toggleToolbar, autoRefresh,
    autoRefreshSecs, toggleAutoRefresh, setShowHelp, error, toast, setReaderFocused, drafts,
    loadingDrafts, loadDrafts, openDraft, pendingNew, showPendingNew, loadingList, nextToken,
    selected, busy, bulkProgress, bulkAction, setBulkLabels, setBulkMove, setSelected, toggleSelect,
    openMessage, loadingMore, loadMore, detail, readerFocused, headersHidden, headersExpanded,
    attachments, downloadAttachment, aiEnabled, aiPromptsEnabled, obsidianOn, slackOn, threadingOn,
    threadMsgs, viewHtml, summarizing, promptRunning, generatingReply, touchingUp, touchUpText,
    setLabelsFor, doAction, setViewHtml, toggleThread, summarize, setPromptsOpen, generateReply,
    setTouchUpText, touchUp, openSuggest, setMoveFor, quickSearch, setLinksFor, sendObsidian,
    forwardSlack, saveMessage, saveRawMessage, setHeadersHidden, setHeadersExpanded, openInGmail,
    readerBodyRef, invite, rsvpBusy, respondInvite, summaryPanelRef, summary, summaryForId,
    dismissSummary, promptPanelRef, promptLabel, promptResult, promptForId, aiCache, runPrompt,
    dismissPrompt, csOpen, csQuery, csIndex, setCsQuery, setCsIndex, setCsOpen, touchUpRef,
    dismissTouchUp, loadingThread, collapsedMsgs, setCollapsedMsgs, summarizeThread, loadingDetail,
    loadRemote, setLoadRemote, imageOptIn, setAlwaysImagesOn,
  } = p;
  return (
    <>
      <TopBar
        query={query}
        localFilter={localFilter}
        searchRef={searchRef}
        searchHint={keymap.search}
        activeQuery={activeQuery}
        onQueryChange={(v) => {
          setQuery(v);
          if (localFilter) applyLocalFilter(v);
        }}
        onSubmitSearch={() => {
          const q = query.trim();
          searchRef.current?.blur();
          if (localFilter) applyLocalFilter(q);
          else void load(q);
        }}
        onToggleFilterMode={() => {
          const next = !localFilter;
          setLocalFilter(next);
          if (next) applyLocalFilter(query);
          else setMessages(fullMessagesRef.current);
        }}
        onSearchEscape={() => {
          setQuery("");
          if (localFilter) {
            setLocalFilter(false);
            setMessages(fullMessagesRef.current);
          } else if (activeQuery) {
            void load("");
          }
        }}
        onAdvanced={() => setAdvOpen(true)}
        onClearSearch={() => {
          setQuery("");
          if (localFilter) setMessages(fullMessagesRef.current);
          else void load("");
        }}
        accounts={accounts}
        account={account}
        switching={switching}
        accountsOpen={accountsOpen}
        onAccountsOpenChange={setAccountsOpen}
        onSwitchAccount={(a) => void switchAccount(a)}
        undoLabel={undoLabel}
        onUndo={() => void runUndo()}
        onCompose={() => setCompose({ mode: "new" })}
        draftsView={draftsView}
        onToggleDrafts={() => {
          if (draftsView) setDraftsView(false);
          else openDrafts();
        }}
        bulkMode={bulkMode}
        onToggleBulk={() => {
          if (bulkMode) exitBulk();
          else {
            setBulkMode(true);
            if (!selectedId && messages.length > 0)
              setSelectedId(messages[0].id);
          }
        }}
        savedQueriesOn={savedQueriesOn}
        onOpenQueries={() => void openQueries()}
        showToolbar={showToolbar}
        onToggleToolbar={toggleToolbar}
        autoRefresh={autoRefresh}
        autoRefreshSecs={autoRefreshSecs}
        onToggleAutoRefresh={toggleAutoRefresh}
        onHelp={() => setShowHelp(true)}
        onRefresh={() => void load(activeQuery)}
      />

      {error && <div className="error-banner">{error}</div>}
      {toast && <div className="toast">{toast}</div>}

      <div className="body">
        <MessageList
          pageSize={PAGE_SIZE}
          onBlurReader={() => setReaderFocused(false)}
          draftsView={draftsView}
          drafts={drafts}
          loadingDrafts={loadingDrafts}
          onRefreshDrafts={() => void loadDrafts()}
          onBackToInbox={() => setDraftsView(false)}
          onOpenDraft={(d) => void openDraft(d)}
          pendingNew={pendingNew}
          onShowPendingNew={showPendingNew}
          loadingList={loadingList}
          messages={messages}
          localFilter={localFilter}
          fullCount={fullMessagesRef.current.length}
          nextToken={nextToken}
          activeQuery={activeQuery}
          selectedId={selectedId}
          bulkMode={bulkMode}
          selected={selected}
          busy={busy}
          bulkProgress={bulkProgress}
          onBulkAction={(action) => void bulkAction(action)}
          onBulkLabels={() => setBulkLabels(true)}
          onBulkMove={() => setBulkMove(true)}
          onSelectAll={() => setSelected(new Set(messages.map((m) => m.id)))}
          onExitBulk={exitBulk}
          onToggleSelect={toggleSelect}
          onOpenMessage={(m) => void openMessage(m)}
          loadingMore={loadingMore}
          onLoadMore={() => void loadMore()}
        />

        <Reader
          detail={detail}
          readerFocused={readerFocused}
          onFocusReader={() => setReaderFocused(true)}
          headersHidden={headersHidden}
          headersExpanded={headersExpanded}
          showToolbar={showToolbar}
          busy={busy}
          attachments={attachments}
          onDownloadAttachment={(att) => void downloadAttachment(att)}
          aiEnabled={aiEnabled}
          aiPromptsEnabled={aiPromptsEnabled}
          obsidianOn={obsidianOn}
          slackOn={slackOn}
          threadingOn={threadingOn}
          hasThread={!!threadMsgs}
          viewHtml={viewHtml}
          summarizing={summarizing}
          promptRunning={promptRunning}
          generatingReply={generatingReply}
          touchingUp={touchingUp}
          touchUpShown={touchUpText !== null}
          onReply={() => detail && setCompose(replyInit(detail))}
          onForward={() => detail && setCompose(forwardInit(detail))}
          onLabels={() => detail && setLabelsFor(detail.id)}
          onArchive={() => detail && void doAction("archive", detail.id)}
          onTrash={() => detail && void doAction("trash", detail.id)}
          onToggleRead={() =>
            detail && void doAction(detail.unread ? "read" : "unread", detail.id)
          }
          onToggleHtml={() => setViewHtml((v) => !v)}
          onToggleThread={() => void toggleThread()}
          onSummarize={() => detail && void summarize(detail.id)}
          onApplyPrompt={() => setPromptsOpen(true)}
          onDraftReply={() => detail && void generateReply(detail)}
          onTouchUp={() =>
            detail &&
            (touchUpText !== null ? setTouchUpText(null) : void touchUp(detail.id))
          }
          onSuggestLabels={() => detail && void openSuggest(detail.id)}
          onMove={() => detail && setMoveFor(detail.id)}
          onSearchSender={() => detail && quickSearch("from", detail)}
          onLinks={() => detail && setLinksFor(detail.id)}
          onObsidian={() => detail && sendObsidian(detail.id)}
          onSlack={() => detail && forwardSlack(detail.id)}
          onSave={() => detail && saveMessage(detail.id)}
          onSaveRaw={() => detail && saveRawMessage(detail.id)}
          onToggleHeaderBlock={() => setHeadersHidden((v) => !v)}
          onToggleFullHeaders={() => setHeadersExpanded((v) => !v)}
          onOpenGmail={() => detail && openInGmail(detail.id)}
          readerBodyRef={readerBodyRef}
          invite={invite}
          rsvpBusy={rsvpBusy}
          onRespond={(status) => detail && void respondInvite(detail.id, status)}
          summaryPanelRef={summaryPanelRef}
          summary={summary}
          summaryForId={summaryForId}
          onRegenerateSummary={() => detail && void summarize(detail.id, true)}
          onDismissSummary={() => detail && dismissSummary(detail.id)}
          promptPanelRef={promptPanelRef}
          promptLabel={promptLabel}
          promptResult={promptResult}
          promptForId={promptForId}
          onRegeneratePrompt={() => {
            if (!detail) return;
            const pid = aiCache.current.get(detail.id)?.lastPromptId;
            if (pid != null)
              void runPrompt(
                { id: pid, name: promptLabel, description: "", category: "" },
                true,
              );
          }}
          onDismissPrompt={() => detail && dismissPrompt(detail.id)}
          csOpen={csOpen}
          csQuery={csQuery}
          csIndex={csIndex}
          setCsQuery={setCsQuery}
          setCsIndex={setCsIndex}
          onCloseSearch={() => {
            setCsOpen(false);
            setCsQuery("");
          }}
          touchUpText={touchUpText}
          touchUpRef={touchUpRef}
          onDismissTouchUp={() => detail && dismissTouchUp(detail.id)}
          loadingThread={loadingThread}
          threadMsgs={threadMsgs}
          collapsedMsgs={collapsedMsgs}
          setCollapsedMsgs={setCollapsedMsgs}
          onSummarizeThread={() => void summarizeThread()}
          loadingDetail={loadingDetail}
          loadRemote={loadRemote}
          onLoadImages={() => {
            if (!detail) return;
            setLoadRemote(true);
            imageOptIn.current.add(detail.id);
          }}
          onAlwaysImages={() => setAlwaysImagesOn(true)}
        />
      </div>
    </>
  );
}
