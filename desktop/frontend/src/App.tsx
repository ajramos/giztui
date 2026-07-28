import { useCallback, useEffect, useRef, useState } from "react";
import {
  DEFAULT_KEYMAP,
  type AccountInfo,
  type KeyMap,
  type Label,
  type UsageStats,
  type ConfigInfo,
  type MessageDetail,
  type MessageSummary,
  type SavedQuery,
  type ActionPlanResult,
  type AnalyzerRule,
} from "./api";
import type { ComposeInit } from "./Compose";
import AppModals from "./AppModals";
import AppInbox from "./AppInbox";
import StartupScreens from "./StartupScreens";
import { useUndo } from "./useUndo";
import { useIntegrations } from "./useIntegrations";
import { useAutoRefresh } from "./useAutoRefresh";
import { useDrafts } from "./useDrafts";
import { useMessages } from "./useMessages";
import { useMailActions } from "./useMailActions";
import { type AdvFilters, EMPTY_ADV } from "./advancedSearch";
import { runCommand } from "./commandRunner";
import { useActionPlan } from "./useActionPlan";
import { handleKeyDown } from "./keydownHandler";
import { useAiActions } from "./useAiActions";
import { useMiscActions } from "./useMiscActions";
import { useReader } from "./useReader";
import { useKeymap } from "./useKeymap";
import type { KeydownCtx } from "./keydownCtx";
import type { CommandCtx } from "./commandCtx";
import { useBootstrap } from "./useBootstrap";
import { useZoom } from "./useZoom";
import { useTheme } from "./useTheme";
import { useAttachments } from "./useAttachments";
import { useRsvp } from "./useRsvp";
import { useThreading } from "./useThreading";

export default function App() {
  const [account, setAccount] = useState("");
  const [initError, setInitError] = useState("");
  const [connecting, setConnecting] = useState(true);
  // First-run: startup failed because credentials.json is missing. Drives a
  // dedicated onboarding screen (explain + import) instead of the raw error.
  const [needCreds, setNeedCreds] = useState(false);
  const [credsPath, setCredsPath] = useState("");
  const [importErr, setImportErr] = useState("");
  const [importing, setImporting] = useState(false);
  // OAuth consent URL while first-run sign-in is pending (desktop only).
  const [authUrl, setAuthUrl] = useState("");
  // New mail found by the background poll, held OUT of the list until the user
  // asks to show it (via the banner or refresh) — auto-injecting it shifts rows
  // under an in-progress operation and risks acting on the wrong message.
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<MessageDetail | null>(null);
  // Whether keyboard focus is "in" the reader: after Enter/click-open (or a click
  // inside the reader) the arrow/j/k keys scroll the message instead of moving
  // the inbox cursor. Escape returns focus to the list. Mirrors the TUI, where
  // opening a message hands the arrows to the reader.
  const [readerFocused, setReaderFocused] = useState(false);
  const readerBodyRef = useRef<HTMLDivElement>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [aiEnabled, setAiEnabled] = useState(false);
  const [compose, setCompose] = useState<ComposeInit | null>(null);
  const [labelsFor, setLabelsFor] = useState<string | null>(null);
  const [showHelp, setShowHelp] = useState(false);
  const [toast, setToast] = useState("");
  const [bulkMode, setBulkMode] = useState(false);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [bulkProgress, setBulkProgress] = useState("");
  const [bulkLabels, setBulkLabels] = useState(false);
  const [aiPromptsEnabled, setAiPromptsEnabled] = useState(false);
  const [promptsOpen, setPromptsOpen] = useState(false);
  const [promptManagerOpen, setPromptManagerOpen] = useState(false);
  const [accounts, setAccounts] = useState<AccountInfo[]>([]);
  const [accountsOpen, setAccountsOpen] = useState(false);
  const [switching, setSwitching] = useState(false);
  const [viewHtml, setViewHtml] = useState(false);
  const [loadRemote, setLoadRemote] = useState(false);
  // "Always load remote images": persisted global default (off = ask per message).
  const [alwaysImages, setAlwaysImages] = useState<boolean>(
    () => localStorage.getItem("giztui.alwaysImages") === "on",
  );
  const alwaysImagesRef = useRef(alwaysImages);
  alwaysImagesRef.current = alwaysImages;
  // Per-message opt-in (session only), so returning to a message you already
  // revealed images for doesn't ask again.
  const imageOptIn = useRef<Set<string>>(new Set());
  const [keymap, setKeymap] = useState<KeyMap>(DEFAULT_KEYMAP);
  const [appVersion, setAppVersion] = useState("");
  const [linksFor, setLinksFor] = useState<string | null>(null);
  const [suggestFor, setSuggestFor] = useState<string | null>(null);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [loadingSuggest, setLoadingSuggest] = useState(false);
  const [cmdOpen, setCmdOpen] = useState(false);
  const [savedQueriesOn, setSavedQueriesOn] = useState(false);
  const [queriesOpen, setQueriesOpen] = useState(false);
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [saveQueryOpen, setSaveQueryOpen] = useState(false);
  const [saveQueryName, setSaveQueryName] = useState("");
  const [bulkPromptText, setBulkPromptText] = useState<string | null>(null);
  const [bulkPromptLabel, setBulkPromptLabel] = useState("");
  const [actionPlanOn, setActionPlanOn] = useState(false);
  const [planOpen, setPlanOpen] = useState(false);
  const [plan, setPlan] = useState<ActionPlanResult | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  // Live feedback while the inbox analysis runs (a single, possibly slow LLM
  // call): how many messages we're analyzing and elapsed seconds, so it never
  // looks hung.
  const [analyzeCount, setAnalyzeCount] = useState(0);
  const [analyzeElapsed, setAnalyzeElapsed] = useState(0);
  // Real batch progress (done/total) emitted by the backend as each AI batch
  // finishes; null until the first event (or in browser mock, which can't emit).
  const [analyzeProgress, setAnalyzeProgress] = useState<{
    done: number;
    total: number;
  } | null>(null);
  const [applyingAll, setApplyingAll] = useState(false);
  const [rulesEnabled, setRulesEnabled] = useState(false);
  const [rulesOpen, setRulesOpen] = useState(false);
  const [detRulesOpen, setDetRulesOpen] = useState(false);
  // Action-plan categories the user has expanded to see their emails.
  const [expandedCats, setExpandedCats] = useState<Set<string>>(new Set());
  // Pending action-plan reassignment: a single email or a whole category being
  // moved to another bucket (opens the destination chooser).
  const [planMove, setPlanMove] = useState<{
    kind: "email" | "category";
    catIdx: number;
    id?: string;
  } | null>(null);
  // Action-plan emails toggled OFF (deselected). Emails are included by default;
  // Space excludes one so category apply/move act only on the checked subset,
  // mirroring the TUI's excluded map.
  const [planExcluded, setPlanExcluded] = useState<Set<string>>(new Set());
  // Quickview: peek at an email's content without leaving the action plan.
  const [planPreview, setPlanPreview] = useState<MessageDetail | null>(null);
  const [planPreviewLoading, setPlanPreviewLoading] = useState(false);
  const [rules, setRules] = useState<AnalyzerRule[]>([]);
  const [newRule, setNewRule] = useState("");
  const [promptPreview, setPromptPreview] = useState<string | null>(null);
  // Theme subsystem (enablement, names, current, picker, applyTheme) lives in useTheme.
  const {
    themesOn,
    themePickerOpen,
    setThemePickerOpen,
    themeNames,
    currentTheme,
    applyTheme,
    initTheme,
  } = useTheme();
  const [statsOpen, setStatsOpen] = useState(false);
  const [stats, setStats] = useState<UsageStats | null>(null);
  const [configOpen, setConfigOpen] = useState(false);
  const [configInfo, setConfigInfo] = useState<ConfigInfo | null>(null);
  // headersExpanded reveals the extra cc/thread/id detail (via the ⋯ menu).
  // headersHidden collapses the whole From/To/Date block for more body room and
  // is what the `h` key toggles, matching the TUI's toggle_headers.
  const [headersExpanded, setHeadersExpanded] = useState(false);
  const [headersHidden, setHeadersHidden] = useState(false);
  const [moveFor, setMoveFor] = useState<string | null>(null);
  const [bulkMove, setBulkMove] = useState(false);
  const [labels, setLabels] = useState<Label[]>([]);
  const [csOpen, setCsOpen] = useState(false);
  const [csQuery, setCsQuery] = useState("");
  const [csIndex, setCsIndex] = useState(0);
  // The reader toolbar is optional — GizTUI is keyboard-first, so users can
  // hide it and drive everything from the keyboard. The choice is persisted.
  const [showToolbar, setShowToolbar] = useState(
    () => localStorage.getItem("giztui.toolbar") !== "off",
  );
  // UI zoom (Cmd/Ctrl +/-/0) lives in useZoom — CSS `zoom` on the root, persisted.
  const { setZoom, bumpZoom, resetZoom } = useZoom();
  // Local filter mode: narrow the already-loaded list client-side instead of
  // running a remote Gmail search (the TUI's search_toggle_mode).
  const [advOpen, setAdvOpen] = useState(false);
  const [adv, setAdv] = useState<AdvFilters>(EMPTY_ADV);
  // Background inbox auto-refresh (opt-in; seeded from config, then remembered).
  const toggleToolbar = useCallback(() => {
    setShowToolbar((v) => {
      const next = !v;
      localStorage.setItem("giztui.toolbar", next ? "on" : "off");
      return next;
    });
  }, []);

  const searchRef = useRef<HTMLInputElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const gPressedAt = useRef(0);
  // Latest keydown-context, refreshed every render so the once-registered window
  // listener reads fresh state without re-subscribing (see the keydown effect).
  const kdCtxRef = useRef<KeydownCtx | null>(null);
  // Latest command-context, refreshed every render so executeCommand stays stable.
  const cmdCtxRef = useRef<CommandCtx | null>(null);
  // VIM range-operation state machine (a3a, d2d, t5t, l2l). Held in a ref so the
  // pending count/timer survive re-renders without re-registering the key
  // handler. `op` is the pending operation key; `count` accumulates digits;
  // `startId` is the message under the cursor when the sequence began (so a
  // stray cursor move during the wait doesn't change the target); `timer` fires
  // the single-message fallback if no second key completes the range.

  // macOS WKWebView does not reliably give the web content keyboard focus until
  // it is clicked, so global shortcuts (space, etc.) appear dead on launch.
  // Focus the app shell on mount — with a few retries, since the webview may not
  // accept focus on the very first tick — on any pointer down, and whenever the
  // window regains focus.
  useEffect(() => {
    const focusApp = () => {
      const active = document.activeElement?.tagName;
      if (active !== "INPUT" && active !== "TEXTAREA") rootRef.current?.focus();
    };
    // On pointer down, only grab focus when the click isn't on a control that
    // needs it (input, textarea, button, link) — so text fields still work.
    const onPointer = (e: PointerEvent) => {
      const t = e.target as HTMLElement | null;
      if (t && t.closest("input, textarea, button, a, select")) return;
      focusApp();
    };
    focusApp();
    [60, 200, 500].forEach((ms) => setTimeout(focusApp, ms));
    window.addEventListener("focus", focusApp);
    document.addEventListener("pointerdown", onPointer, true);
    return () => {
      window.removeEventListener("focus", focusApp);
      document.removeEventListener("pointerdown", onPointer, true);
    };
  }, []);

  // Keep the selected row visible when navigating by keyboard.
  useEffect(() => {
    if (!selectedId) return;
    const el = document.querySelector(".row.selected");
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedId]);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    setTimeout(() => setToast(""), 2500);
  }, []);

  const { obsidianOn, slackOn, refresh: refreshIntegrations, sendObsidian, forwardSlack } =
    useIntegrations({ showToast, setError });
  const {
    draftsView,
    setDraftsView,
    drafts,
    loadingDrafts,
    loadDrafts,
    openDrafts,
    openDraft,
  } = useDrafts({ setError, setSelectedId, setDetail, setCompose });

  // previewRef breaks the declaration-order cycle: useMessages.load() previews the
  // first row, but the previewer (previewMessage → loadMessage) is defined below
  // and owns the openIdRef/AI landmine, so it stays in App and is wired via the ref.
  const previewRef = useRef<(m: MessageSummary) => void>(() => {});
  const {
    messages,
    setMessages,
    fullMessagesRef,
    pendingNew,
    query,
    setQuery,
    activeQuery,
    nextToken,
    loadingList,
    loadingMore,
    localFilter,
    setLocalFilter,
    load,
    loadMore,
    applyLocalFilter,
    checkNewMail,
    showPendingNew,
  } = useMessages({ setError, setSelectedId, setDetail, draftsView, previewRef });

  // Per-message subsystems extracted from App.tsx (F3.2). Their per-message data
  // is still fetched inside loadMessage (gated by openIdRef) via the setters
  // below; the hooks own the state + standalone actions.
  const {
    attachments,
    setAttachments,
    attachmentsOpen,
    setAttachmentsOpen,
    downloadAttachment,
  } = useAttachments(detail, { setBusy, setError, showToast });
  const {
    rsvpEnabled,
    setRsvpEnabled,
    rsvpEnabledRef,
    invite,
    setInvite,
    rsvpBusy,
    rsvpPickerOpen,
    setRsvpPickerOpen,
    respondInvite,
  } = useRsvp({ setError, showToast });
  const {
    threadingOn,
    setThreadingOn,
    threadMsgs,
    setThreadMsgs,
    collapsedMsgs,
    setCollapsedMsgs,
    loadingThread,
    toggleThread,
  } = useThreading(detail, { setError });

  // Global "always load remote images" toggle, persisted across launches. Turning
  // it on reveals images in the current message too.
  const setAlwaysImagesOn = useCallback(
    (on: boolean) => {
      setAlwaysImages(on);
      localStorage.setItem("giztui.alwaysImages", on ? "on" : "off");
      if (on) setLoadRemote(true);
      showToast(on ? "Always loading remote images" : "Images: ask per message");
    },
    [showToast],
  );

  const {
    autoRefresh,
    setAutoRefresh,
    autoRefreshSecs,
    setAutoRefreshSecs,
    toggleAutoRefresh,
  } = useAutoRefresh({ showToast, onTick: () => void checkNewMail() });

  // The AI subsystem owns its panel state + the openIdRef/aiCache/mirror-ref
  // landmines and returns them so loadMessage (below) and the render can consume
  // them by their original names. Declared before useReader/useBootstrap because
  // those reset/clear this state.
  const {
    summary, setSummary, summarizing, summaryForId,
    promptResult, setPromptResult, promptLabel, setPromptLabel, promptRunning, setPromptRunning, promptForId,
    generatingReply, touchUpText, setTouchUpText, touchingUp,
    summaryPanelRef, promptPanelRef, touchUpRef,
    openIdRef, aiCache, runningLabelRef, promptLabelRef, promptForIdRef,
    promptRunningRef, summarizingRef, summaryForIdRef,
    summarize, generateReply, touchUp, runPrompt, dismissSummary,
    dismissPrompt, dismissTouchUp, dismissAI, regenerateActive, summarizeThread,
  } = useAiActions({
    detail, bulkMode, selected, aiEnabled, showToast, setError,
    setPromptsOpen, setCompose, setBulkPromptLabel, setBulkPromptText,
  });

  const { importCreds, retryInit, switchAccount } = useBootstrap({
    load, initTheme, refreshIntegrations, setConnecting, setInitError, setNeedCreds,
    setAuthUrl, setCredsPath, setError, setAccount, setAiEnabled, setAiPromptsEnabled,
    setAccounts, setKeymap, setAppVersion, setThreadingOn, setSavedQueriesOn, setActionPlanOn,
    setRulesEnabled, setLabels, setRsvpEnabled, setAutoRefreshSecs, setAutoRefresh, setImportErr,
    setImporting, setSwitching, setSelectedId, setDetail, setSummary, setPromptResult,
    setBulkMode, setSelected, setQuery,
  });

  // loadMessage shows a message in the reading pane. markRead=false is used for
  // cursor navigation (preview) so scanning the inbox never marks mail read;
  // markRead=true is used for a deliberate open (Enter / click).
  const { previewMessage, openMessage } = useReader({
    setMessages, setSelectedId, setLoadingDetail, setDetail, setError, setSummary,
    setPromptResult, setPromptLabel, setTouchUpText, setViewHtml, setAttachments, setAttachmentsOpen,
    setThreadMsgs, setCollapsedMsgs, setInvite, setCsOpen, setLoadRemote, setReaderFocused,
    openIdRef, aiCache, alwaysImagesRef, imageOptIn, rootRef, rsvpEnabledRef,
    runningLabelRef, promptLabelRef, promptForIdRef, promptRunningRef, summarizingRef, summaryForIdRef,
  });
  // Ref so load() can preview the first message without a declaration-order cycle.
  previewRef.current = previewMessage;


  // Undo stack: each mutating action pushes a reversal closure; the undo key runs
  // the most recent one (GizTUI's "U").
  const { undoLabel, pushUndo, runUndo } = useUndo({ showToast, setError });

  const {
    removeFromList,
    insertMessage,
    applyLabelChange,
    doAction,
    toggleSelect,
    exitBulk,
    clearReaderIfRemoved,
    advanceAfterBulk,
    bulkActionIds,
    bulkAction,
  } = useMailActions({
    messages, setMessages, fullMessagesRef, selectedId, setSelectedId, setDetail,
    bulkMode, selected, setSelected, setBulkMode, setBulkProgress, previewRef,
    pushUndo, showToast, setError, setBusy, setSummary, setThreadMsgs,
  });

  // --- VIM range operations (a3a, d2d, t5t, l2l) ---------------------------
  // The operation keys these cover. A single press fires the normal
  // single-message action after a short timeout; press-count-press applies it to
  // the next N messages, exactly like the TUI's "VIM Power Operations".
  const { chordAction, runVimSingle, runVimRange, clearVimRange, vimRange } = useKeymap({
    keymap, obsidianOn, messages, bulkMode, selected, doAction,
    bulkAction, bulkActionIds, setBulkLabels, setBulkMode, setBulkProgress,
    setLabelsFor, setSelected, setSelectedId,
  });

  const {
    openStats, openConfig, clearCaches, doMove, doBulkMove, quickSearch, openInGmail, saveMessage, saveRawMessage, openSuggest, applySuggestion, openQueries, runQuery, deleteQuery, doSaveQuery,
  } = useMiscActions({
    messages, selected, activeQuery, suggestFor, saveQueryName, showToast,
    setError, load, removeFromList, insertMessage, advanceAfterBulk, pushUndo,
    setBulkMove, setBulkProgress, setBusy, setConfigInfo, setConfigOpen, setLoadingSuggest,
    setMessages, setMoveFor, setQueriesOpen, setQuery, setSavedQueries, setSelected,
    setStats, setStatsOpen, setSuggestFor, setSuggestions, setSaveQueryOpen, setSaveQueryName,
  });

  const {
    runActionPlan, runDeterministicRules, applyCategory, dispatchPromptCategory,
    applyAllCategories, doPlanMove, planNodes, planNav,
    planActiveNode, planNodesRef, planActiveRef, openRules, addRule, deleteRule,
    viewAnalyzerPrompt,
  } = useActionPlan({
    plan, setPlan, planOpen, setPlanOpen, analyzing, setAnalyzing,
    setAnalyzeCount, setAnalyzeElapsed, setAnalyzeProgress, planExcluded, setPlanExcluded, expandedCats,
    setApplyingAll, planMove, setPlanMove, planPreview, setPlanPreview, setPlanPreviewLoading,
    setRules, newRule, setNewRule, rulesOpen, setRulesOpen,
    setDetRulesOpen, messages, setMessages, promptPreview, setPromptPreview, bulkPromptText,
    setBulkPromptText, setBulkPromptLabel, setPromptRunning, showToast, setError, clearReaderIfRemoved,
  });
  // Rebuilt every render into a ref so executeCommand stays stable (no giant
  // deps array, no CommandBar churn) while always seeing fresh state.
  cmdCtxRef.current = {
    detail, load, doAction, activeQuery, openDrafts, saveMessage,
    sendObsidian, forwardSlack, obsidianOn, slackOn, aiEnabled, aiPromptsEnabled,
    summarize, openSuggest, openInGmail, openQueries, savedQueriesOn, runActionPlan,
    runDeterministicRules, actionPlanOn, bulkMode, selected, showToast, doMove,
    doBulkMove, generateReply, quickSearch, themesOn, applyTheme, rulesEnabled,
    openRules, viewAnalyzerPrompt, toggleToolbar, touchUp, touchUpText, localFilter,
    applyLocalFilter, query, runUndo, toggleAutoRefresh, saveRawMessage, invite,
    respondInvite, openStats, openConfig, clearCaches, loadMore, attachments,
    threadingOn, toggleThread, threadMsgs, summarizeThread, messages, previewMessage,
    accounts, applyLabelChange, bumpZoom, resetZoom, setZoom, dismissAI,
    regenerateActive, setError, setCompose, setSelected, setSelectedId, setMessages,
    setLabelsFor, setLinksFor, setMoveFor, setTouchUpText, setCsQuery, setCsIndex,
    setCsOpen, setCollapsedMsgs, setLocalFilter, setBulkMode, setViewHtml, setHeadersHidden,
    setLoadRemote, setAlwaysImagesOn, setAccountsOpen, setAdvOpen, setAttachmentsOpen, setBulkMove,
    setDetRulesOpen, setPromptManagerOpen, setPromptsOpen, setRsvpPickerOpen, setSaveQueryOpen, setShowHelp,
    setThemePickerOpen, alwaysImagesRef, imageOptIn, fullMessagesRef,
  };
  const executeCommand = useCallback(
    (input: string) => runCommand(input, cmdCtxRef.current!),
    [],
  );

  // Invert the (config-driven) keymap into a chord → action lookup. gotoTop is
  // handled separately (it may be the "gg" vim sequence).

  // Global keyboard shortcuts, driven by the user's GizTUI keybindings. List
  // navigation (j/k/arrows/Enter/Esc) mirrors the TUI's native table: j/k move a
  // cursor without opening; Enter opens. This avoids marking mail read while
  // just scanning.
  // Rebuilt every render into a ref so the single window listener (registered
  // once, below) always reads fresh state without a giant exhaustive-deps array
  // or re-subscribing on every keystroke-relevant change.
  kdCtxRef.current = {
    attachmentsOpen, activeQuery, gPressedAt, vimRange, forwardSlack, themesOn,
    accounts, accountsOpen, actionPlanOn, advOpen, aiEnabled, aiPromptsEnabled,
    applyCategory, attachments, bulkAction, bulkLabels, bulkMode, bulkMove,
    bulkPromptText, bumpZoom, chordAction, clearVimRange, cmdOpen, compose,
    configOpen, csOpen, csQuery, detRulesOpen, detail, dismissAI,
    doAction, draftsView, exitBulk, fullMessagesRef, headersHidden, invite,
    keymap, labelsFor, linksFor, load, loadMore, localFilter,
    messages, moveFor, obsidianOn, openDrafts, openInGmail, openMessage,
    openQueries, openRules, openSuggest, plan, planActiveRef, planMove,
    planNodesRef, planOpen, planPreview, previewMessage, promptManagerOpen, promptPreview,
    promptsOpen, queriesOpen, quickSearch, readerBodyRef, readerFocused, regenerateActive,
    resetZoom, rsvpPickerOpen, rulesEnabled, rulesOpen, runActionPlan, runUndo,
    runVimRange, runVimSingle, saveMessage, saveQueryOpen, saveRawMessage, savedQueriesOn,
    searchRef, selected, selectedId, sendObsidian, setAccountsOpen, setAdvOpen,
    setAttachmentsOpen, setBulkLabels, setBulkMode, setBulkMove, setBulkProgress, setBulkPromptText,
    setCmdOpen, setCompose, setConfigOpen, setCsIndex, setCsOpen, setCsQuery,
    setDetail, setDraftsView, setExpandedCats, setHeadersHidden, setLabelsFor, setLinksFor,
    setLocalFilter, setMessages, setMoveFor, setPlanExcluded, setPlanMove, setPlanOpen,
    setPlanPreview, setPromptManagerOpen, setPromptPreview, setPromptsOpen, setQueriesOpen, setQuery,
    setReaderFocused, setRsvpPickerOpen, setRulesOpen, setSaveQueryOpen, setSelected, setSelectedId,
    setShowHelp, setStatsOpen, setSuggestFor, setThemePickerOpen, setViewHtml, showHelp,
    showToast, slackOn, statsOpen, suggestFor, summarize, themePickerOpen,
    threadingOn, toggleSelect, toggleThread, viewAnalyzerPrompt,
  };
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (kdCtxRef.current) handleKeyDown(e, kdCtxRef.current);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  if (needCreds || initError || connecting) {
    return (
      <StartupScreens
        needCreds={needCreds}
        initError={initError}
        connecting={connecting}
        credsPath={credsPath}
        importErr={importErr}
        importing={importing}
        authUrl={authUrl}
        importCreds={importCreds}
        retryInit={retryInit}
      />
    );
  }

  // Props for the modal/picker stack (AppModals forwards to ModalsPrimary +
  // ModalsSecondary). All were verbatim name={name} pass-throughs.
  const modalProps = {
    compose, setCompose, showToast, draftsView, loadDrafts, labelsFor, setLabelsFor,
    applyLabelChange, bulkLabels, setBulkLabels, selected, promptsOpen, setPromptsOpen,
    runPrompt, aiPromptsEnabled, setPromptManagerOpen, promptManagerOpen, aiEnabled, aiCache,
    setPromptResult, linksFor, setLinksFor, suggestFor, setSuggestFor, suggestions, loadingSuggest,
    applySuggestion, attachmentsOpen, setAttachmentsOpen, attachments, busy, downloadAttachment,
    queriesOpen, setQueriesOpen, savedQueries, activeQuery, runQuery, deleteQuery, setSaveQueryOpen,
    rsvpPickerOpen, setRsvpPickerOpen, detail, invite, rsvpBusy, respondInvite, saveQueryOpen,
    saveQueryName, setSaveQueryName, doSaveQuery,
    bulkPromptText, setBulkPromptText, bulkPromptLabel, promptRunning, planOpen, analyzing,
    analyzeCount, analyzeProgress, analyzeElapsed, plan, planNodes, planActiveNode, planNav,
    expandedCats, setExpandedCats, planExcluded, setPlanExcluded, applyingAll, rulesEnabled, messages,
    applyCategory, dispatchPromptCategory, applyAllCategories, setPlanOpen, openMessage, openRules,
    viewAnalyzerPrompt, planMove, setPlanMove, doPlanMove, planPreview, planPreviewLoading, setPlanPreview,
    detRulesOpen, setDetRulesOpen, runDeterministicRules, rulesOpen, rules, newRule, setNewRule,
    addRule, deleteRule, setRulesOpen, promptPreview, setPromptPreview, cmdOpen, executeCommand,
    setCmdOpen, themePickerOpen, themeNames, currentTheme, applyTheme, setThemePickerOpen, moveFor,
    labels, doMove, setMoveFor, bulkMove, doBulkMove, setBulkMove, advOpen, adv, setAdv,
    setLocalFilter, setQuery, load, setAdvOpen, statsOpen, stats, setStatsOpen, configOpen, configInfo,
    clearCaches, setConfigOpen, showHelp, keymap, obsidianOn, slackOn, threadingOn, savedQueriesOn,
    actionPlanOn, rsvpEnabled, themesOn, appVersion, setShowHelp,
  };

  // Props for the inbox surface (top bar + banners + list/reader). All were
  // verbatim pass-throughs or inline handlers now living in AppInbox.
  const inboxProps = {
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
  };

  return (
    <div className="app" ref={rootRef} tabIndex={-1}>
      <AppInbox {...inboxProps} />
      <AppModals {...modalProps} />
    </div>
  );
}

// --- helpers -----------------------------------------------------------------

