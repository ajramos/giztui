import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  backend,
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
import { replyInit, forwardInit } from "./compose";
import ModalsPrimary from "./ModalsPrimary";
import ModalsSecondary from "./ModalsSecondary";
import MessageList from "./MessageList";
import Reader from "./Reader";
import TopBar from "./TopBar";
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
import type { KeydownCtx } from "./keydownCtx";
import { useBootstrap } from "./useBootstrap";
import { useZoom } from "./useZoom";
import { useTheme } from "./useTheme";
import { useAttachments } from "./useAttachments";
import { useRsvp } from "./useRsvp";
import { useThreading } from "./useThreading";

const PAGE_SIZE = 50;

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
  // Refs to the AI result panels so we can scroll them into view when they
  // appear — otherwise, if the reader is scrolled down, the panel renders above
  // the fold and it looks like nothing happened.
  const summaryPanelRef = useRef<HTMLDivElement>(null);
  const promptPanelRef = useRef<HTMLDivElement>(null);
  const touchUpRef = useRef<HTMLDivElement>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [aiEnabled, setAiEnabled] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [summarizing, setSummarizing] = useState(false);
  // The message a summary run/result belongs to, so a summary started on one
  // email doesn't paint its "Generating…" / stream over another you navigated to.
  const [summaryForId, setSummaryForId] = useState<string | null>(null);
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
  const [promptResult, setPromptResult] = useState<string | null>(null);
  const [promptLabel, setPromptLabel] = useState("");
  const [promptRunning, setPromptRunning] = useState(false);
  // The message a single-message prompt result/run belongs to, so a run started
  // on one email doesn't paint its "Generating…" over a different email you've
  // since navigated to.
  const [promptForId, setPromptForId] = useState<string | null>(null);
  // Always the id of the message currently open in the reader, so a streaming
  // prompt can tell it should stop updating the visible panel once you move away.
  const openIdRef = useRef<string | null>(null);
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
  // Remember AI results per message (session) so navigating away and back shows
  // the summary / prompt output / reformat again instead of a blank panel. The
  // backend also caches, but the frontend state was cleared on every open.
  const aiCache = useRef<
    Map<
      string,
      {
        summary?: string;
        touchUp?: string;
        // Prompt results keyed by prompt id, so re-running a prompt you already
        // ran on this message reuses the result (no new LLM call / tokens).
        promptResults?: Record<number, { text: string; label: string }>;
        // Which prompt result to restore when you return to this message (cleared
        // on dismiss, so a dismissed panel stays closed but the result is kept).
        lastPromptId?: number;
      }
    >
  >(new Map());
  // updateAiCache merges a patch into a message's cache entry (creating it if
  // needed); a key set to undefined deletes it. Consolidates the repeated
  // get-or-{}/mutate/set dance around aiCache.current.
  const updateAiCache = useCallback(
    (
      id: string,
      patch: Partial<{
        summary: string | undefined;
        touchUp: string | undefined;
        lastPromptId: number | undefined;
        promptResults: Record<number, { text: string; label: string }>;
      }>,
    ) => {
      const e = aiCache.current.get(id) ?? {};
      for (const [k, v] of Object.entries(patch)) {
        if (v === undefined) delete (e as Record<string, unknown>)[k];
        else (e as Record<string, unknown>)[k] = v;
      }
      aiCache.current.set(id, e);
    },
    [],
  );
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
  const [generatingReply, setGeneratingReply] = useState(false);
  const [touchUpText, setTouchUpText] = useState<string | null>(null);
  const [touchingUp, setTouchingUp] = useState(false);
  // Refs mirroring the AI-panel state so the keydown handler / commands can read
  // fresh values without stale closures (for :dismiss, :regenerate, layered Esc).
  const summaryRef = useRef(summary);
  summaryRef.current = summary;
  const promptResultRef = useRef(promptResult);
  promptResultRef.current = promptResult;
  const promptLabelRef = useRef(promptLabel);
  promptLabelRef.current = promptLabel;
  const promptRunningRef = useRef(promptRunning);
  promptRunningRef.current = promptRunning;
  const promptForIdRef = useRef(promptForId);
  promptForIdRef.current = promptForId;
  const summarizingRef = useRef(summarizing);
  summarizingRef.current = summarizing;
  const summaryForIdRef = useRef(summaryForId);
  summaryForIdRef.current = summaryForId;
  // Label of the prompt currently streaming, keyed by message id, so returning
  // to a message mid-run can restore its panel title (the global promptLabel is
  // reset when you navigate away).
  const runningLabelRef = useRef<Record<string, string>>({});
  const touchUpTextRef = useRef(touchUpText);
  touchUpTextRef.current = touchUpText;
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
  // VIM range-operation state machine (a3a, d2d, t5t, l2l). Held in a ref so the
  // pending count/timer survive re-renders without re-registering the key
  // handler. `op` is the pending operation key; `count` accumulates digits;
  // `startId` is the message under the cursor when the sequence began (so a
  // stray cursor move during the wait doesn't change the target); `timer` fires
  // the single-message fallback if no second key completes the range.
  const vimRange = useRef<{
    key: string;
    op: "archive" | "trash" | "toggleRead" | "manageLabels";
    count: number;
    startId: string;
    timer: number;
  } | null>(null);

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

  // When an AI result panel starts, reveal it (the panels render at the top of
  // the reader, so if you'd scrolled down they'd appear above the fold and look
  // like a no-op) and flash a toast so there's immediate feedback either way.
  useEffect(() => {
    if (summarizing && summaryForId && summaryForId === detail?.id) {
      summaryPanelRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
      showToast("Summarizing…");
    }
  }, [summarizing, summaryForId, detail?.id, showToast]);
  useEffect(() => {
    // Only for a single-message prompt on the message that's actually open (a
    // bulk run streams into its own modal; promptForId is null for it).
    if (promptRunning && promptForId && promptForId === detail?.id) {
      promptPanelRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
      showToast(promptLabel ? `Applying ${promptLabel}…` : "Applying prompt…");
    }
  }, [promptRunning, promptForId, detail?.id, promptLabel, showToast]);
  useEffect(() => {
    if (touchingUp) showToast("Reformatting…");
  }, [touchingUp, showToast]);
  useEffect(() => {
    // The reformatted panel only mounts once the result is set, so reveal it then.
    if (touchUpText !== null)
      touchUpRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
  }, [touchUpText]);

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
  const loadMessage = useCallback(
    async (m: MessageSummary, markRead: boolean) => {
      // Record the newly-open message synchronously so a prompt still streaming
      // for the PREVIOUS message stops writing into the visible panel.
      openIdRef.current = m.id;
      setSelectedId(m.id);
      setLoadingDetail(true);
      setError("");
      // Restore any AI results computed for this message earlier this session.
      const ai = aiCache.current.get(m.id);
      // If an AI run for THIS message is still streaming, keep the live panel
      // (its result isn't cached yet). The stream repaints the body on the next
      // token; the title would otherwise blank out, so restore it explicitly.
      const summaryStreamingHere =
        summarizingRef.current && summaryForIdRef.current === m.id;
      const promptStreamingHere =
        promptRunningRef.current && promptForIdRef.current === m.id;
      if (!summaryStreamingHere) setSummary(ai?.summary ?? null);
      const lastPrompt =
        ai?.lastPromptId != null ? ai.promptResults?.[ai.lastPromptId] : undefined;
      if (promptStreamingHere) {
        setPromptLabel(runningLabelRef.current[m.id] ?? promptLabelRef.current);
        // Blank the body (not another message's cached text); the in-flight
        // stream repaints the full accumulated text on its next token.
        setPromptResult("");
      } else {
        setPromptResult(lastPrompt?.text ?? null);
        setPromptLabel(lastPrompt?.label ?? "");
      }
      const hadSessionPrompt = ai?.lastPromptId != null;
      setAttachments([]);
      setAttachmentsOpen(false);
      setThreadMsgs(null);
      setCollapsedMsgs(new Set());
      setTouchUpText(ai?.touchUp ?? null);
      setInvite(null);
      setCsOpen(false);
      // Keep keyboard focus on the app shell (not the HTML iframe) so shortcuts
      // keep working while reading.
      requestAnimationFrame(() => {
        const active = document.activeElement?.tagName;
        if (active !== "INPUT" && active !== "TEXTAREA") rootRef.current?.focus();
      });
      try {
        const d = await backend.GetMessage(m.id);
        // A faster navigation (e.g. j/k to a cached message) may have already
        // opened another message while this fetch was in flight — abandon the
        // stale result so the reader never shows a different email's body.
        if (openIdRef.current !== m.id) return;
        setDetail(d);
        setViewHtml(!!(d.html && d.html.trim()));
        // Show remote images automatically if the global "always" is on or the
        // user already opted this message in earlier this session.
        setLoadRemote(alwaysImagesRef.current || imageOptIn.current.has(m.id));
        // Restore prompt results persisted in the DB (survive app restarts). Seed
        // the session cache so re-running reuses them, and surface the most recent
        // if the session didn't already show one.
        void backend
          .CachedPrompts(m.id)
          .then((cached) => {
            if (openIdRef.current !== m.id || !cached || cached.length === 0) return;
            const e = aiCache.current.get(m.id) ?? {};
            const pr = { ...(e.promptResults ?? {}) };
            for (const c of cached) {
              if (pr[c.promptId] === undefined)
                pr[c.promptId] = { text: c.text, label: c.name };
            }
            e.promptResults = pr;
            if (!hadSessionPrompt && e.lastPromptId === undefined) {
              e.lastPromptId = cached[0].promptId;
              setPromptResult(cached[0].text);
              setPromptLabel(cached[0].name);
            }
            aiCache.current.set(m.id, e);
          })
          .catch(() => undefined);
        void backend
          .ListAttachments(m.id)
          .then((atts) => {
            if (openIdRef.current === m.id) setAttachments(atts);
          })
          .catch(() => undefined);
        if (rsvpEnabledRef.current) {
          void backend
            .InviteInfo(m.id)
            .then((inv) => {
              if (openIdRef.current === m.id) setInvite(inv.isInvite ? inv : null);
            })
            .catch(() => undefined);
        }
        if (markRead && m.unread) {
          void backend.MarkRead(m.id).catch(() => undefined);
          setMessages((prev) =>
            prev.map((x) => (x.id === m.id ? { ...x, unread: false } : x)),
          );
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setLoadingDetail(false);
      }
    },
    [],
  );

  const previewMessage = useCallback(
    (m: MessageSummary) => {
      // Previewing (j/k in the list) keeps focus on the list.
      setReaderFocused(false);
      void loadMessage(m, false);
    },
    [loadMessage],
  );
  const openMessage = useCallback(
    (m: MessageSummary) => {
      // Opening (Enter / click) hands the keyboard to the reader so arrows scroll
      // the message body, like the TUI.
      setReaderFocused(true);
      void loadMessage(m, true);
    },
    [loadMessage],
  );
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
  type VimOp = "archive" | "trash" | "toggleRead" | "manageLabels";

  // runVimSingle fires the single-message fallback when a range never completes.
  // It respects an active bulk selection (matching the TUI's "VIM BULK FIX").
  const runVimSingle = useCallback(
    (op: VimOp, startId: string) => {
      const hasSel = bulkMode && selected.size > 0;
      switch (op) {
        case "archive":
          if (hasSel) void bulkAction("archive");
          else void doAction("archive", startId);
          break;
        case "trash":
          if (hasSel) void bulkAction("trash");
          else void doAction("trash", startId);
          break;
        case "toggleRead": {
          if (hasSel) {
            void bulkAction("unread");
          } else {
            const m = messages.find((x) => x.id === startId);
            void doAction(m?.unread ? "read" : "unread", startId);
          }
          break;
        }
        case "manageLabels":
          if (hasSel) setBulkLabels(true);
          else setLabelsFor(startId);
          break;
      }
    },
    [bulkMode, selected, bulkAction, doAction, messages],
  );

  // runVimRange applies an operation to `count` messages starting at the row that
  // was under the cursor when the sequence began.
  const runVimRange = useCallback(
    (op: VimOp, count: number, startId: string) => {
      const startIdx = messages.findIndex((m) => m.id === startId);
      if (startIdx < 0) return;
      const slice = messages.slice(startIdx, startIdx + count);
      const ids = slice.map((m) => m.id);
      if (ids.length === 0) return;
      switch (op) {
        case "archive":
          void bulkActionIds("archive", ids);
          break;
        case "trash":
          void bulkActionIds("trash", ids);
          break;
        case "toggleRead": {
          // Toggle each message by its own current state (unread → read and
          // vice-versa), like the TUI's toggleReadRange.
          const toRead = slice.filter((m) => m.unread).map((m) => m.id);
          const toUnread = slice.filter((m) => !m.unread).map((m) => m.id);
          if (toRead.length) void bulkActionIds("read", toRead);
          if (toUnread.length) void bulkActionIds("unread", toUnread);
          break;
        }
        case "manageLabels":
          // Select the range into bulk mode and open the bulk labels picker.
          setBulkMode(true);
          setSelected(new Set(ids));
          setSelectedId(ids[0]);
          setBulkLabels(true);
          break;
      }
    },
    [messages, bulkActionIds],
  );

  // clearVimRange cancels any pending range sequence and its fallback timer.
  const clearVimRange = useCallback(() => {
    if (vimRange.current?.timer) window.clearTimeout(vimRange.current.timer);
    vimRange.current = null;
    setBulkProgress("");
  }, []);

  const {
    summarize, generateReply, touchUp, runPrompt, dismissSummary,
    dismissPrompt, dismissTouchUp, dismissAI, regenerateActive, summarizeThread,
  } = useAiActions({
    detail, bulkMode, selected, aiEnabled, showToast, setError,
    setSummary, setSummarizing, setSummaryForId, setPromptResult, setPromptLabel, setPromptRunning,
    setPromptForId, setPromptsOpen, setGeneratingReply, setTouchUpText, setTouchingUp, setCompose,
    setBulkPromptLabel, setBulkPromptText, openIdRef, aiCache, updateAiCache, summaryRef,
    promptResultRef, touchUpTextRef, promptLabelRef, runningLabelRef, promptPanelRef,
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
  const executeCommand = useCallback(
    (input: string) =>
      runCommand(input, {
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
      }),
    [
      detail,
      load,
      doAction,
      activeQuery,
      openDrafts,
      saveMessage,
      sendObsidian,
      forwardSlack,
      obsidianOn,
      slackOn,
      aiEnabled,
      aiPromptsEnabled,
      summarize,
      openSuggest,
      openInGmail,
      openQueries,
      savedQueriesOn,
      runActionPlan,
      runDeterministicRules,
      actionPlanOn,
      bulkMode,
      selected,
      showToast,
      doMove,
      doBulkMove,
      generateReply,
      quickSearch,
      themesOn,
      applyTheme,
      rulesEnabled,
      openRules,
      viewAnalyzerPrompt,
      toggleToolbar,
      touchUp,
      touchUpText,
      localFilter,
      applyLocalFilter,
      query,
      runUndo,
      toggleAutoRefresh,
      saveRawMessage,
      invite,
      respondInvite,
      openStats,
      openConfig,
      clearCaches,
      loadMore,
      attachments,
      threadingOn,
      toggleThread,
      threadMsgs,
      summarizeThread,
      messages,
      previewMessage,
      accounts,
      applyLabelChange,
      bumpZoom,
      resetZoom,
      setZoom,
    ],
  );

  // Invert the (config-driven) keymap into a chord → action lookup. gotoTop is
  // handled separately (it may be the "gg" vim sequence).
  const chordAction = useMemo(() => {
    const m: Record<string, string> = {};
    // First binding wins (mirrors the TUI's first-matching-case behavior when
    // two actions share a default key, e.g. open_gmail vs obsidian).
    const add = (key: string, action: string) => {
      if (key && key !== "gg" && !m[key]) m[key] = action;
    };
    add(keymap.summarize, "summarize");
    add(keymap.prompt, "prompt");
    add(keymap.archive, "archive");
    add(keymap.trash, "trash");
    add(keymap.toggleRead, "toggleRead");
    add(keymap.manageLabels, "manageLabels");
    add(keymap.compose, "compose");
    add(keymap.reply, "reply");
    add(keymap.forward, "forward");
    add(keymap.search, "search");
    add(keymap.refresh, "refresh");
    add(keymap.loadMore, "loadMore");
    add(keymap.drafts, "drafts");
    // "O" defaults to both obsidian and open-in-gmail. The TUI checks obsidian
    // first (first-match wins), so when Obsidian is enabled register it first to
    // match; otherwise fall through to open-in-gmail so the key is still useful.
    if (obsidianOn) {
      add(keymap.obsidian, "obsidian");
      add(keymap.openGmail, "openGmail");
    } else {
      add(keymap.openGmail, "openGmail");
      add(keymap.obsidian, "obsidian");
    }
    add(keymap.bulkMode, "bulkMode");
    add(keymap.bulkSelect, "bulkSelect");
    add(keymap.markdown, "markdown");
    add(keymap.help, "help");
    add(keymap.linkPicker, "links");
    add(keymap.replyAll, "replyAll");
    add(keymap.saveMessage, "saveMessage");
    add(keymap.saveRaw, "saveRaw");
    add(keymap.suggestLabel, "suggestLabel");
    add(keymap.slack, "slack");
    add(keymap.commandMode, "commandMode");
    // Quick searches derived from the current message. Registered before
    // threading so "T" resolves to search-to (matching the TUI, where
    // toggle_threading is unbound by default); threading stays on its toolbar
    // button and the :threads command.
    add(keymap.searchFrom, "searchFrom");
    add(keymap.searchTo, "searchTo");
    add(keymap.searchSubject, "searchSubject");
    add(keymap.threading, "threading");
    add(keymap.savedQueries, "savedQueries");
    add(keymap.saveQuery, "saveQuery");
    add(keymap.actionPlan, "actionPlan");
    add(keymap.themePicker, "themePicker");
    // generateReply ("g") is intentionally not registered here: the "gg" goto-top
    // sequence intercepts "g" first, so it would be dead. It lives in the reader's
    // "⋯" menu and the :reply-ai command instead.
    add(keymap.move, "move");
    add(keymap.toggleHeaders, "toggleHeaders");
    add(keymap.undo, "undo");
    add(keymap.unread, "unread");
    add(keymap.archived, "archived");
    add(keymap.attachments, "attachments");
    add(keymap.rsvp, "rsvp");
    add(keymap.quit, "quit");
    // Uppercase of the summarize key force-regenerates the summary (ignoring the
    // cache), mirroring the TUI's y → Y. Registered last so a user's own binding
    // for that key wins.
    add(keymap.summarize.toUpperCase(), "regenerateSummary");
    return m;
  }, [keymap, obsidianOn]);

  // Global keyboard shortcuts, driven by the user's GizTUI keybindings. List
  // navigation (j/k/arrows/Enter/Esc) mirrors the TUI's native table: j/k move a
  // cursor without opening; Enter opens. This avoids marking mail read while
  // just scanning.
  useEffect(() => {
    const ctx: KeydownCtx = {
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
    const onKey = (e: KeyboardEvent) => handleKeyDown(e, ctx);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [
    keymap,
    chordAction,
    messages,
    selectedId,
    detail,
    aiEnabled,
    aiPromptsEnabled,
    compose,
    labelsFor,
    bulkLabels,
    promptsOpen,
    promptManagerOpen,
    showHelp,
    linksFor,
    suggestFor,
    cmdOpen,
    obsidianOn,
    slackOn,
    threadingOn,
    threadMsgs,
    savedQueriesOn,
    queriesOpen,
    saveQueryOpen,
    planOpen,
    planMove,
    planPreview,
    applyCategory,
    themePickerOpen,
    rulesOpen,
    promptPreview,
    advOpen,
    statsOpen,
    configOpen,
    moveFor,
    bulkPromptText,
    actionPlanOn,
    themesOn,
    activeQuery,
    localFilter,
    bulkMode,
    selected,
    draftsView,
    openMessage,
    previewMessage,
    doAction,
    bulkAction,
    runVimSingle,
    runVimRange,
    clearVimRange,
    toggleSelect,
    exitBulk,
    summarize,
    quickSearch,
    runUndo,
    toggleThread,
    openQueries,
    runActionPlan,
    saveMessage,
    sendObsidian,
    forwardSlack,
    openSuggest,
    openDrafts,
    openInGmail,
    saveRawMessage,
    attachments,
    headersHidden,
    showToast,
    csOpen,
    csQuery,
    invite,
    accounts,
    bumpZoom,
    resetZoom,
    rsvpPickerOpen,
    detRulesOpen,
    accountsOpen,
    attachmentsOpen,
    readerFocused,
    rulesEnabled,
    openRules,
    viewAnalyzerPrompt,
    plan,
    load,
    loadMore,
  ]);

  if (needCreds) {
    return (
      <div className="fatal onboarding">
        <span className="logo" aria-hidden="true">
          ✦
        </span>
        <h1>Welcome to GizTUI Desktop</h1>
        <p className="fatal-msg">
          To connect to Gmail, GizTUI needs your own Google API credentials — a
          one-time <code>credentials.json</code> (an OAuth client you create in
          Google Cloud). Your email never passes through anyone else's servers.
        </p>
        <ol className="onboarding-steps">
          <li>
            In the Google Cloud Console, <b>enable the Gmail API</b> and create
            an <b>OAuth client ID</b> of type <b>Desktop app</b>.
          </li>
          <li>
            <b>Download</b> the client's <code>credentials.json</code>.
          </li>
          <li>
            Click <b>Choose credentials.json…</b> below to import it (GizTUI
            copies it to <code>{credsPath || "~/.config/giztui/credentials.json"}</code>),
            then sign in.
          </li>
        </ol>
        {importErr && <p className="fatal-msg onboarding-err">{importErr}</p>}
        <div className="signin-actions">
          <button
            className="primary"
            disabled={importing}
            onClick={() => void importCreds()}
          >
            {importing ? "Importing…" : "Choose credentials.json…"}
          </button>
          <button
            onClick={() =>
              void backend.OpenURL(
                "https://github.com/ajramos/giztui/blob/main/docs/GETTING_STARTED.md#gmail-api-setup",
              )
            }
          >
            Open the setup guide
          </button>
          <button disabled={importing} onClick={() => void retryInit()}>
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (initError) {
    return (
      <div className="fatal">
        <h1>GizTUI Desktop</h1>
        <p className="fatal-msg">Could not start a Gmail session:</p>
        <pre>{initError}</pre>
        <p className="hint">
          Make sure GizTUI is configured (run <code>giztui --setup</code>) and
          that <code>~/.config/giztui/</code> holds valid credentials and token.
        </p>
        <div className="signin-actions">
          <button className="primary" onClick={() => void retryInit()}>
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (connecting) {
    return (
      <div className="connecting">
        <span className="logo">✦</span>
        <h1>GizTUI Desktop</h1>
        {authUrl ? (
          <div className="signin">
            <p>Sign in to your Google account to continue.</p>
            <p className="muted">
              We opened your browser to grant access. Once you approve, this
              window continues automatically.
            </p>
            <div className="signin-actions">
              <button
                className="primary"
                onClick={() => void backend.OpenAuthURL()}
              >
                Open sign-in in browser
              </button>
              <button
                onClick={() => void navigator.clipboard?.writeText(authUrl)}
              >
                Copy link
              </button>
            </div>
            <p className="muted signin-url">{authUrl}</p>
          </div>
        ) : (
          <p className="muted">Connecting to Gmail…</p>
        )}
      </div>
    );
  }

  return (
    <div className="app" ref={rootRef} tabIndex={-1}>
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

      <ModalsPrimary
        compose={compose}
        setCompose={setCompose}
        showToast={showToast}
        draftsView={draftsView}
        loadDrafts={loadDrafts}
        labelsFor={labelsFor}
        setLabelsFor={setLabelsFor}
        applyLabelChange={applyLabelChange}
        bulkLabels={bulkLabels}
        setBulkLabels={setBulkLabels}
        selected={selected}
        promptsOpen={promptsOpen}
        setPromptsOpen={setPromptsOpen}
        runPrompt={runPrompt}
        aiPromptsEnabled={aiPromptsEnabled}
        setPromptManagerOpen={setPromptManagerOpen}
        promptManagerOpen={promptManagerOpen}
        aiEnabled={aiEnabled}
        aiCache={aiCache}
        setPromptResult={setPromptResult}
        linksFor={linksFor}
        setLinksFor={setLinksFor}
        suggestFor={suggestFor}
        setSuggestFor={setSuggestFor}
        suggestions={suggestions}
        loadingSuggest={loadingSuggest}
        applySuggestion={applySuggestion}
        attachmentsOpen={attachmentsOpen}
        setAttachmentsOpen={setAttachmentsOpen}
        attachments={attachments}
        busy={busy}
        downloadAttachment={downloadAttachment}
        queriesOpen={queriesOpen}
        setQueriesOpen={setQueriesOpen}
        savedQueries={savedQueries}
        activeQuery={activeQuery}
        runQuery={runQuery}
        deleteQuery={deleteQuery}
        setSaveQueryOpen={setSaveQueryOpen}
        rsvpPickerOpen={rsvpPickerOpen}
        setRsvpPickerOpen={setRsvpPickerOpen}
        detail={detail}
        invite={invite}
        rsvpBusy={rsvpBusy}
        respondInvite={respondInvite}
        saveQueryOpen={saveQueryOpen}
        saveQueryName={saveQueryName}
        setSaveQueryName={setSaveQueryName}
        doSaveQuery={doSaveQuery}
      />
      <ModalsSecondary
        bulkPromptText={bulkPromptText}
        setBulkPromptText={setBulkPromptText}
        bulkPromptLabel={bulkPromptLabel}
        promptRunning={promptRunning}
        planOpen={planOpen}
        analyzing={analyzing}
        analyzeCount={analyzeCount}
        analyzeProgress={analyzeProgress}
        analyzeElapsed={analyzeElapsed}
        plan={plan}
        planNodes={planNodes}
        planActiveNode={planActiveNode}
        planNav={planNav}
        expandedCats={expandedCats}
        setExpandedCats={setExpandedCats}
        planExcluded={planExcluded}
        setPlanExcluded={setPlanExcluded}
        applyingAll={applyingAll}
        rulesEnabled={rulesEnabled}
        messages={messages}
        applyCategory={applyCategory}
        dispatchPromptCategory={dispatchPromptCategory}
        applyAllCategories={applyAllCategories}
        setPlanOpen={setPlanOpen}
        openMessage={openMessage}
        openRules={openRules}
        viewAnalyzerPrompt={viewAnalyzerPrompt}
        planMove={planMove}
        setPlanMove={setPlanMove}
        doPlanMove={doPlanMove}
        planPreview={planPreview}
        planPreviewLoading={planPreviewLoading}
        setPlanPreview={setPlanPreview}
        detRulesOpen={detRulesOpen}
        setDetRulesOpen={setDetRulesOpen}
        runDeterministicRules={runDeterministicRules}
        rulesOpen={rulesOpen}
        rules={rules}
        newRule={newRule}
        setNewRule={setNewRule}
        addRule={addRule}
        deleteRule={deleteRule}
        setRulesOpen={setRulesOpen}
        promptPreview={promptPreview}
        setPromptPreview={setPromptPreview}
        cmdOpen={cmdOpen}
        executeCommand={executeCommand}
        setCmdOpen={setCmdOpen}
        themePickerOpen={themePickerOpen}
        themeNames={themeNames}
        currentTheme={currentTheme}
        applyTheme={applyTheme}
        setThemePickerOpen={setThemePickerOpen}
        showToast={showToast}
        moveFor={moveFor}
        labels={labels}
        doMove={doMove}
        setMoveFor={setMoveFor}
        bulkMove={bulkMove}
        selected={selected}
        doBulkMove={doBulkMove}
        setBulkMove={setBulkMove}
        advOpen={advOpen}
        adv={adv}
        setAdv={setAdv}
        setLocalFilter={setLocalFilter}
        setQuery={setQuery}
        load={load}
        setAdvOpen={setAdvOpen}
        statsOpen={statsOpen}
        stats={stats}
        setStatsOpen={setStatsOpen}
        configOpen={configOpen}
        configInfo={configInfo}
        clearCaches={clearCaches}
        setConfigOpen={setConfigOpen}
        showHelp={showHelp}
        keymap={keymap}
        aiEnabled={aiEnabled}
        aiPromptsEnabled={aiPromptsEnabled}
        obsidianOn={obsidianOn}
        slackOn={slackOn}
        threadingOn={threadingOn}
        savedQueriesOn={savedQueriesOn}
        actionPlanOn={actionPlanOn}
        rsvpEnabled={rsvpEnabled}
        themesOn={themesOn}
        appVersion={appVersion}
        setShowHelp={setShowHelp}
      />
    </div>
  );
}

// --- helpers -----------------------------------------------------------------

