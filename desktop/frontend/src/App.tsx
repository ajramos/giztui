import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  applyBulkPromptStream,
  applyPromptStream,
  backend,
  DEFAULT_KEYMAP,
  summarizeStream,
  threadSummaryStream,
  type AccountInfo,
  type KeyMap,
  type Label,
  type UsageStats,
  type ConfigInfo,
  type MessageDetail,
  type MessageSummary,
  type Prompt,
  type SavedQuery,
  type ActionPlanResult,
  type AnalyzerRule,
} from "./api";
import type { ComposeInit } from "./Compose";
import {
  matchesCombo,
  emailAddr,
  cleanSubject,
  countMatches,
} from "./format";
import { replyInit, replyAllInit, forwardInit } from "./compose";
import { activeAiPanel } from "./aiPanels";
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

  const runInit = useCallback(async () => {
    // Reset to the connecting state (also covers retry-after-import).
    setConnecting(true);
    setInitError("");
    setNeedCreds(false);
    // The backend builds the Gmail/service session off the main thread so the
    // window paints immediately; wait for it to be ready before the first
    // calls (up to ~45s for a cold OAuth) instead of erroring.
    for (let i = 0; i < 300; i++) {
      try {
        if (await backend.Ready()) break;
      } catch {
        break; // mock / no backend
      }
      // Surface the sign-in URL (first-run OAuth) so the modal can offer a
      // button instead of the user hunting for the URL in the logs.
      try {
        setAuthUrl(await backend.PendingAuthURL());
      } catch {
        /* mock backend has no pending auth */
      }
      await new Promise((r) => setTimeout(r, 150));
    }
    setAuthUrl("");
    setConnecting(false);
    try {
      const ie = await backend.InitError();
      if (ie) {
        // Distinguish "no credentials.json yet" (first-run) from other errors so
        // the UI can offer an import flow instead of a dead-end message.
        try {
          if (await backend.NeedsCredentials()) {
            setNeedCreds(true);
            setCredsPath(await backend.CredentialsPath());
          }
        } catch {
          /* older/mock backend without these methods */
        }
        setInitError(ie);
        return;
      }
    } catch {
      /* mock backend never errors here */
    }
      try {
        setAccount(await backend.AccountEmail());
      } catch {
        /* non-fatal */
      }
      try {
        setAiEnabled(await backend.AIEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        setAiPromptsEnabled(await backend.PromptsEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        setAccounts(await backend.ListAccounts());
      } catch {
        /* non-fatal */
      }
      try {
        setKeymap(await backend.KeyMap());
      } catch {
        /* non-fatal — defaults already set */
      }
      try {
        setAppVersion(await backend.Version());
      } catch {
        /* non-fatal */
      }
      try {
        await refreshIntegrations();
        setThreadingOn(await backend.ThreadingEnabled());
        setSavedQueriesOn(await backend.SavedQueriesEnabled());
        setActionPlanOn(await backend.ActionPlanEnabled());
        setRulesEnabled(await backend.AnalyzerRulesEnabled());
      } catch {
        /* non-fatal */
      }
      await initTheme();
      try {
        setLabels(await backend.ListLabels());
      } catch {
        /* non-fatal */
      }
      try {
        setRsvpEnabled(await backend.RSVPEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        const ar = await backend.AutoRefreshSettings();
        if (ar.intervalSeconds > 0) setAutoRefreshSecs(ar.intervalSeconds);
        // localStorage overrides the config default once the user has chosen.
        const saved = localStorage.getItem("giztui.autorefresh");
        setAutoRefresh(saved === null ? ar.enabled : saved === "on");
      } catch {
        /* non-fatal */
      }
      void load("");
  }, [load, initTheme]);

  useEffect(() => {
    void runInit();
  }, [runInit]);

  // Import a credentials.json via the native file picker, then retry init.
  const importCreds = useCallback(async () => {
    setImportErr("");
    setImporting(true);
    try {
      await backend.ImportCredentials();
      // ImportCredentials returns "" if the user cancelled the dialog; in that
      // case NeedsCredentials stays true and runInit just shows the screen again.
      await runInit();
    } catch (e) {
      setImportErr(String((e as Error)?.message ?? e));
    } finally {
      setImporting(false);
    }
  }, [runInit]);

  // Re-run init after the user placed credentials.json manually.
  const retryInit = useCallback(async () => {
    setImportErr("");
    try {
      await backend.RetryInit();
    } catch {
      /* mock backend */
    }
    await runInit();
  }, [runInit]);

  const switchAccount = useCallback(
    async (a: AccountInfo) => {
      setSwitching(true);
      setError("");
      try {
        await backend.SwitchAccount(a.id);
        setSelectedId(null);
        setDetail(null);
        setSummary(null);
        setPromptResult(null);
        setBulkMode(false);
        setSelected(new Set());
        setQuery("");
        const [email, ai, prompts, accs] = await Promise.all([
          backend.AccountEmail().catch(() => ""),
          backend.AIEnabled().catch(() => false),
          backend.PromptsEnabled().catch(() => false),
          backend.ListAccounts().catch(() => [] as AccountInfo[]),
        ]);
        setAccount(email);
        setAiEnabled(ai);
        setAiPromptsEnabled(prompts);
        if (accs.length) setAccounts(accs);
        await load("");
      } catch (e) {
        setError(String(e));
      } finally {
        setSwitching(false);
      }
    },
    [load],
  );

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

  const summarize = useCallback(async (id: string, force = false) => {
    setSummaryForId(id);
    setSummarizing(true);
    if (openIdRef.current === id) setSummary("");
    setError("");
    try {
      let acc = "";
      const final = await summarizeStream(
        id,
        (tok) => {
          acc += tok;
          // Only paint into the visible panel while this message is still open.
          if (openIdRef.current === id) setSummary(acc);
        },
        force,
      );
      updateAiCache(id, { summary: final });
      if (openIdRef.current === id) setSummary(final);
    } catch (e) {
      setError(String(e));
      if (openIdRef.current === id) setSummary(null);
    } finally {
      setSummarizing(false);
    }
  }, [updateAiCache]);

  // generateReply asks the AI to draft a reply, then opens the composer with the
  // draft prefilled so the user can edit before sending.
  const generateReply = useCallback(
    async (d: MessageDetail) => {
      setGeneratingReply(true);
      setError("");
      try {
        const draft = await backend.GenerateReply(d.id);
        setCompose({ mode: "reply", originalId: d.id, to: d.from, body: draft });
      } catch (e) {
        setError(String(e));
      } finally {
        setGeneratingReply(false);
      }
    },
    [],
  );

  // touchUp reformats the open message's body with the AI and shows the cleaned
  // version in place of the raw text (revertable).
  const touchUp = useCallback(async (id: string) => {
    setTouchingUp(true);
    setError("");
    try {
      const t = await backend.TouchUp(id);
      setTouchUpText(t);
      updateAiCache(id, { touchUp: t });
    } catch (e) {
      setError(String(e));
    } finally {
      setTouchingUp(false);
    }
  }, [updateAiCache]);


  const openStats = useCallback(async () => {
    setStatsOpen(true);
    try {
      setStats(await backend.UsageStats());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const openConfig = useCallback(async () => {
    setConfigOpen(true);
    try {
      setConfigInfo(await backend.ConfigInfo());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const clearCaches = useCallback(async () => {
    try {
      await backend.ClearCaches();
      showToast("Caches cleared");
    } catch (e) {
      setError(String(e));
    }
  }, [showToast]);

  // doMove applies a label and archives the message (Gmail "move to folder").
  const doMove = useCallback(
    async (id: string, name: string) => {
      const label = name.trim();
      if (!label) return;
      setMoveFor(null);
      try {
        await backend.MoveToLabel(id, label);
        // Advance the cursor to the next row (like the TUI / archive / trash)
        // instead of leaving the selection blank.
        removeFromList(id);
        showToast(`Moved to ${label}`);
      } catch (e) {
        setError(String(e));
      }
    },
    [removeFromList, showToast],
  );

  // doBulkMove moves every selected message to a folder (apply label + archive),
  // the bulk form of doMove — mirrors the TUI's bulk "move to folder".
  const doBulkMove = useCallback(
    async (name: string) => {
      const label = name.trim();
      if (!label) return;
      const ids = [...selected];
      setBulkMove(false);
      if (ids.length === 0) return;
      const idSet = new Set(ids);
      const removed = messages
        .map((m, i) => ({ m, i }))
        .filter(({ m }) => idSet.has(m.id));
      setBusy(true);
      setBulkProgress(`Moving ${ids.length}…`);
      setError("");
      try {
        await backend.BulkMoveToLabel(ids, label);
        setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
        advanceAfterBulk(idSet);
        setSelected(new Set());
        pushUndo(`move ${ids.length}`, async () => {
          await backend.BulkUnarchive(ids);
          removed.forEach(({ m, i }) => insertMessage(m, i));
        });
        showToast(`Moved ${ids.length} to ${label}`);
      } catch (e) {
        setError(String(e));
      } finally {
        setBulkProgress("");
        setBusy(false);
      }
    },
    [
      selected,
      messages,
      advanceAfterBulk,
      pushUndo,
      insertMessage,
      showToast,
    ],
  );

  // quickSearch runs a Gmail search derived from the current message (sender,
  // recipient, or subject), mirroring the TUI's F / T / S shortcuts.
  const quickSearch = useCallback(
    (kind: "from" | "to" | "subject", d: MessageDetail) => {
      let q = "";
      if (kind === "from") q = `from:${emailAddr(d.from) || d.from}`;
      else if (kind === "to") q = `to:${emailAddr(d.to) || d.to}`;
      else q = `subject:${JSON.stringify(cleanSubject(d.subject))}`;
      setQuery(q);
      void load(q);
    },
    [load],
  );

  const openInGmail = useCallback((id: string) => {
    void backend.OpenGmailWeb(id).catch((e) => setError(String(e)));
  }, []);

  const runPrompt = useCallback(
    async (prompt: Prompt, force = false) => {
      const bulk = bulkMode && selected.size > 0;
      if (!bulk && !detail) return;
      setPromptsOpen(false);
      setError("");

      // --- bulk (multi-message) prompt: streams into the bulk modal ----------
      if (bulk) {
        setPromptForId(null); // bulk isn't tied to the open message's panel
        setPromptRunning(true);
        setBulkPromptLabel(`${prompt.name} · ${selected.size} messages`);
        setBulkPromptText("");
        try {
          let acc = "";
          const final = await applyBulkPromptStream(
            [...selected],
            prompt.id,
            (tok) => {
              acc += tok;
              setBulkPromptText(acc);
            },
          );
          setBulkPromptText(final);
        } catch (e) {
          setError(String(e));
          setBulkPromptText(null);
        } finally {
          setPromptRunning(false);
        }
        return;
      }

      // --- single-message prompt --------------------------------------------
      const launchId = detail!.id;
      // Reuse a result already generated for this (message, prompt) — no new LLM
      // call, so dismissing then re-running the same prompt is free. force skips
      // the cache to regenerate (e.g. after editing the prompt).
      const cached = force
        ? undefined
        : aiCache.current.get(launchId)?.promptResults?.[prompt.id];
      if (cached) {
        updateAiCache(launchId, { lastPromptId: prompt.id });
        setPromptForId(launchId);
        setPromptLabel(cached.label);
        setPromptResult(cached.text);
        showToast(`${cached.label} (cached)`);
        requestAnimationFrame(() =>
          promptPanelRef.current?.scrollIntoView({ block: "start", behavior: "smooth" }),
        );
        return;
      }

      setPromptForId(launchId);
      setPromptRunning(true);
      setPromptLabel(prompt.name);
      setPromptResult("");
      runningLabelRef.current[launchId] = prompt.name;
      try {
        let acc = "";
        const final = await applyPromptStream(
          launchId,
          prompt.id,
          (tok) => {
            acc += tok;
            // Only paint into the visible panel while this message is still open.
            if (openIdRef.current === launchId) setPromptResult(acc);
          },
          force,
        );
        updateAiCache(launchId, {
          promptResults: {
            ...(aiCache.current.get(launchId)?.promptResults ?? {}),
            [prompt.id]: { text: final, label: prompt.name },
          },
          lastPromptId: prompt.id,
        });
        if (openIdRef.current === launchId) {
          setPromptResult(final);
          setPromptLabel(prompt.name);
        }
      } catch (e) {
        setError(String(e));
        if (openIdRef.current === launchId) setPromptResult(null);
      } finally {
        setPromptRunning(false);
        delete runningLabelRef.current[launchId];
      }
    },
    [detail, bulkMode, selected, showToast, updateAiCache],
  );

  // Per-panel dismiss: hide the panel and forget just enough of its cache entry
  // so it stays closed on return. Summary/touch-up drop their cached text; the
  // prompt keeps its result but clears lastPromptId (so nothing auto-restores).
  const dismissSummary = useCallback(
    (id: string | null) => {
      setSummary(null);
      if (id) updateAiCache(id, { summary: undefined });
    },
    [updateAiCache],
  );
  const dismissPrompt = useCallback(
    (id: string | null) => {
      setPromptResult(null);
      if (id) updateAiCache(id, { lastPromptId: undefined });
    },
    [updateAiCache],
  );
  const dismissTouchUp = useCallback(
    (id: string | null) => {
      setTouchUpText(null);
      if (id) updateAiCache(id, { touchUp: undefined });
    },
    [updateAiCache],
  );

  // dismissAI closes any open AI panel for the current message (summary / prompt /
  // reformat). Returns whether it dismissed anything (for the layered Escape).
  const dismissAI = useCallback(() => {
    const id = openIdRef.current;
    let any = false;
    if (summaryRef.current !== null) {
      dismissSummary(id);
      any = true;
    }
    if (promptResultRef.current !== null) {
      dismissPrompt(id);
      any = true;
    }
    if (touchUpTextRef.current !== null) {
      dismissTouchUp(id);
      any = true;
    }
    return any;
  }, [dismissSummary, dismissPrompt, dismissTouchUp]);

  // regenerateActive re-runs the AI panel currently shown for the open message:
  // the summary if one is up, otherwise the last prompt (both force a fresh call).
  const regenerateActive = useCallback(() => {
    const id = openIdRef.current;
    if (!id) return;
    const kind = activeAiPanel({
      hasSummary: summaryRef.current !== null,
      hasPrompt: promptResultRef.current !== null,
      hasTouchUp: touchUpTextRef.current !== null,
    });
    if (kind === "prompt") {
      const pid = aiCache.current.get(id)?.lastPromptId;
      if (pid != null)
        void runPrompt(
          { id: pid, name: promptLabelRef.current, description: "", category: "" },
          true,
        );
      return;
    }
    if (kind === "touchup") {
      void touchUp(id);
      return;
    }
    // summary is shown, or nothing yet → (re)generate the summary.
    if (aiEnabled) void summarize(id, true);
  }, [summarize, runPrompt, touchUp, aiEnabled]);

  const saveMessage = useCallback(
    (id: string) => {
      void backend
        .SaveMessage(id)
        .then((path) => showToast(`Saved to ${path}`))
        .catch((e) => setError(String(e)));
    },
    [showToast],
  );

  const saveRawMessage = useCallback(
    (id: string) => {
      void backend
        .SaveRawMessage(id)
        .then((path) => showToast(`Saved .eml to ${path}`))
        .catch((e) => setError(String(e)));
    },
    [showToast],
  );

  const openSuggest = useCallback(async (id: string) => {
    setSuggestFor(id);
    setSuggestions([]);
    setLoadingSuggest(true);
    setError("");
    try {
      setSuggestions(await backend.SuggestLabels(id));
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingSuggest(false);
    }
  }, []);

  const applySuggestion = useCallback(
    (name: string) => {
      if (!suggestFor) return;
      void backend
        .ApplyLabelByName(suggestFor, name)
        .then(() => showToast(`Applied "${name}"`))
        .catch((e) => setError(String(e)));
      setSuggestions((prev) => prev.filter((s) => s !== name));
    },
    [suggestFor, showToast],
  );

  const summarizeThread = useCallback(async () => {
    if (!detail) return;
    setSummarizing(true);
    setSummary("");
    setError("");
    try {
      let acc = "";
      const final = await threadSummaryStream(detail.threadId, (tok) => {
        acc += tok;
        setSummary(acc);
      });
      setSummary(final);
    } catch (e) {
      setError(String(e));
      setSummary(null);
    } finally {
      setSummarizing(false);
    }
  }, [detail]);

  const openQueries = useCallback(async () => {
    setQueriesOpen(true);
    try {
      setSavedQueries(await backend.ListSavedQueries());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const runQuery = useCallback(
    (q: SavedQuery) => {
      setQueriesOpen(false);
      void backend.RecordQueryUse(q.id).catch(() => undefined);
      setQuery(q.query);
      void load(q.query);
    },
    [load],
  );

  const deleteQuery = useCallback(async (id: number) => {
    try {
      await backend.DeleteSavedQuery(id);
      setSavedQueries(await backend.ListSavedQueries());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const doSaveQuery = useCallback(() => {
    const name = saveQueryName.trim();
    if (!name || !activeQuery) return;
    void backend
      .SaveQuery(name, activeQuery)
      .then(() => showToast(`Saved query "${name}"`))
      .catch((e) => setError(String(e)));
    setSaveQueryOpen(false);
    setSaveQueryName("");
  }, [saveQueryName, activeQuery, showToast]);

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
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement | null)?.tagName;
      const typing = tag === "INPUT" || tag === "TEXTAREA";
      const chord = e.key === " " ? "space" : e.key;

      // UI zoom (Cmd/Ctrl +/-/0) — handled first so it works everywhere, even
      // over modals or while typing. WKWebView ignores native zoom.
      if ((e.metaKey || e.ctrlKey) && !e.altKey) {
        if (e.key === "=" || e.key === "+") {
          e.preventDefault();
          bumpZoom(0.1);
          return;
        }
        if (e.key === "-" || e.key === "_") {
          e.preventDefault();
          bumpZoom(-0.1);
          return;
        }
        if (e.key === "0") {
          e.preventDefault();
          resetZoom();
          return;
        }
      }

      // Advanced search builder (default Ctrl+F / Cmd+F, from search_advanced).
      // Handled before the generic Ctrl/Cmd early-return and the typing guard so
      // it opens from anywhere — list, reader, or the search box — like the TUI.
      // Only modifier combos are honored here (a bare key would hijack typing).
      if (keymap.searchAdvanced && !advOpen && matchesCombo(e, keymap.searchAdvanced)) {
        e.preventDefault();
        setAdvOpen(true);
        return;
      }

      // Never let OS/browser clipboard & navigation combos (Cmd/Ctrl+C, V, X, A,
      // Z…) fall through to single-key actions like compose ("c"). The only
      // Cmd/Ctrl combos we act on are zoom (handled above) and the accounts
      // switcher (Cmd/Ctrl+A, handled below), so let everything else reach the
      // browser — otherwise selecting text and pressing Cmd+C opened the composer.
      if (e.metaKey || e.ctrlKey) {
        const isAccounts =
          (e.key === "a" || e.key === "A") && accounts.length > 1;
        if (!isAccounts) return;
      }

      if (showHelp) {
        if (e.key === "Escape" || chord === keymap.help) {
          setShowHelp(false);
          e.preventDefault();
        }
        return;
      }
      const anyModal =
        compose ||
        labelsFor ||
        bulkLabels ||
        promptsOpen ||
        promptManagerOpen ||
        linksFor ||
        suggestFor ||
        cmdOpen ||
        queriesOpen ||
        saveQueryOpen ||
        planOpen ||
        themePickerOpen ||
        rulesOpen ||
        promptPreview !== null ||
        advOpen ||
        statsOpen ||
        configOpen ||
        moveFor ||
        bulkMove ||
        rsvpPickerOpen ||
        detRulesOpen ||
        accountsOpen ||
        attachmentsOpen ||
        bulkPromptText !== null;
      if (anyModal) {
        // Escape closes the topmost modal from the window (WKWebView won't focus
        // a bare div, so per-modal Escape handlers on divs are unreliable). Order
        // = last-opened first, so a sub-modal (e.g. rules over the plan) closes
        // before its parent. Pickers also self-close via their own listener;
        // double-closing is harmless.
        if (e.key === "Escape") {
          e.preventDefault();
          if (accountsOpen) setAccountsOpen(false);
          else if (attachmentsOpen) setAttachmentsOpen(false);
          else if (promptPreview !== null) setPromptPreview(null);
          else if (rulesOpen) setRulesOpen(false);
          else if (bulkPromptText !== null) setBulkPromptText(null);
          else if (saveQueryOpen) setSaveQueryOpen(false);
          else if (moveFor) setMoveFor(null);
          else if (bulkMove) setBulkMove(false);
          else if (suggestFor) setSuggestFor(null);
          else if (advOpen) setAdvOpen(false);
          else if (statsOpen) setStatsOpen(false);
          else if (configOpen) setConfigOpen(false);
          else if (planPreview) setPlanPreview(null);
          else if (planMove) setPlanMove(null);
          else if (planOpen) setPlanOpen(false);
          else if (themePickerOpen) setThemePickerOpen(false);
          else if (queriesOpen) setQueriesOpen(false);
          else if (rsvpPickerOpen) setRsvpPickerOpen(false);
          else if (linksFor) setLinksFor(null);
          else if (bulkLabels) setBulkLabels(false);
          else if (labelsFor) setLabelsFor(null);
          else if (promptsOpen) setPromptsOpen(false);
          else if (promptManagerOpen) setPromptManagerOpen(false);
          else if (cmdOpen) setCmdOpen(false);
          else if (compose) setCompose(null);
          return;
        }
        // Ctrl/Cmd+A toggles the account menu closed too (it's what opened it),
        // since the menu is now part of the modal guard and swallows other keys.
        if (
          accountsOpen &&
          (e.ctrlKey || e.metaKey) &&
          (e.key === "a" || e.key === "A")
        ) {
          e.preventDefault();
          setAccountsOpen(false);
          return;
        }
        // Action-plan reachable-by-keyboard shortcuts for its header buttons.
        if (
          planOpen &&
          !rulesOpen &&
          !detRulesOpen &&
          promptPreview === null &&
          planMove === null &&
          planPreview === null &&
          bulkPromptText === null
        ) {
          if (e.key === "r") {
            e.preventDefault();
            if (rulesEnabled) void openRules();
            return;
          }
          if (e.key === "p") {
            e.preventDefault();
            void viewAnalyzerPrompt();
            return;
          }
          // l applies a label bucket as LABEL-ONLY (Enter does the move variant).
          if (e.key === "l") {
            const node = planNodesRef.current[planActiveRef.current];
            const c = node ? plan?.categories[node.catIdx] : undefined;
            if (c && c.action === "label") {
              e.preventDefault();
              void applyCategory(c, false);
              return;
            }
          }
          // m reassigns to another bucket: on an email node, that one email; on a
          // category node, the whole category (the TUI's action-plan move).
          if (e.key === "m") {
            const node = planNodesRef.current[planActiveRef.current];
            if (node) {
              e.preventDefault();
              if (node.type === "email") {
                setPlanMove({
                  kind: "email",
                  catIdx: node.catIdx,
                  id: node.id,
                });
              } else {
                setPlanMove({ kind: "category", catIdx: node.catIdx });
              }
              return;
            }
          }
          // Space toggles selection of the active email (deselect to exclude it
          // from the category's apply/move); on a category it expands/collapses.
          if (e.key === " ") {
            const node = planNodesRef.current[planActiveRef.current];
            if (node?.type === "email") {
              e.preventDefault();
              setPlanExcluded((prev) => {
                const n = new Set(prev);
                if (n.has(node.id)) n.delete(node.id);
                else n.add(node.id);
                return n;
              });
              return;
            }
            const cat = node ? plan?.categories[node.catIdx] : undefined;
            if (cat) {
              e.preventDefault();
              setExpandedCats((prev) => {
                const n = new Set(prev);
                if (n.has(cat.name)) n.delete(cat.name);
                else n.add(cat.name);
                return n;
              });
              return;
            }
          }
          // → expands the active category (or the parent of the active email);
          // ← collapses it.
          if (e.key === "ArrowRight" || e.key === "ArrowLeft") {
            const node = planNodesRef.current[planActiveRef.current];
            const cat = node ? plan?.categories[node.catIdx] : undefined;
            if (cat) {
              e.preventDefault();
              setExpandedCats((prev) => {
                const n = new Set(prev);
                if (e.key === "ArrowLeft") n.delete(cat.name);
                else n.add(cat.name);
                return n;
              });
              return;
            }
          }
        }
        return;
      }
      if (typing) {
        if (e.key === "Escape") {
          // preventDefault so macOS/WKWebView doesn't treat Escape as
          // "leave fullscreen"; we handle it ourselves.
          e.preventDefault();
          const el = e.target as HTMLElement;
          if (el === searchRef.current) {
            // TUI parity: Escape in the search box clears the filter and
            // returns to the default inbox, instead of just blurring.
            setQuery("");
            if (localFilter) {
              setLocalFilter(false);
              setMessages(fullMessagesRef.current);
            } else if (activeQuery) {
              void load("");
            }
          }
          el.blur();
        }
        return;
      }
      if (draftsView) {
        if (e.key === "Escape" || chord === keymap.drafts) {
          setDraftsView(false);
          e.preventDefault();
        } else if (chord === keymap.help) {
          setShowHelp(true);
        }
        return;
      }

      // Ctrl/Cmd+A opens the account switcher (TUI's accounts shortcut). Placed
      // after the typing guard so it never hijacks select-all inside inputs.
      if (
        (e.ctrlKey || e.metaKey) &&
        (e.key === "a" || e.key === "A") &&
        accounts.length > 1
      ) {
        e.preventDefault();
        setAccountsOpen((v) => !v);
        return;
      }

      // Content-search match navigation: once a find is active, n = next match,
      // N = previous (the TUI's in-message n/N). Typing in the search box is
      // handled by the typing guard above, so this only fires from the reader.
      if (csOpen && csQuery && detail && (chord === "n" || chord === "N")) {
        e.preventDefault();
        const total = countMatches(detail.plainText || "", csQuery);
        if (total > 0)
          setCsIndex((i) =>
            chord === "n" ? (i + 1) % total : (i - 1 + total) % total,
          );
        return;
      }

      const idx = selectedId
        ? messages.findIndex((m) => m.id === selectedId)
        : -1;
      // Navigation previews the message (shows content, scrolls to it) without
      // marking it read — in bulk mode too, so the reader follows the cursor
      // like the TUI (previewMessage also sets selectedId as the highlight).
      const moveCursor = (i: number) => {
        if (i < 0 || i >= messages.length) return;
        previewMessage(messages[i]);
      };
      const hasSel = bulkMode && selected.size > 0;

      // Layered Escape: an open AI panel (summary / prompt / reformat) is closed
      // first, before Escape hands the reader back / exits a search.
      if (chord === "Escape" && dismissAI()) {
        e.preventDefault();
        return;
      }

      // --- reader-focused scrolling (TUI parity) ---
      // After Enter/click-open (or a click in the reader), arrows & j/k scroll
      // the message body instead of moving the inbox cursor; Space/PageDown page
      // through it; Escape hands focus back to the list (a second Escape then
      // closes the reader). Only nav keys are intercepted here — actions
      // (archive, reply, summarize, …) still fall through to the switch below.
      if (readerFocused && detail && !bulkMode) {
        const sc = readerBodyRef.current;
        const page = sc ? sc.clientHeight * 0.9 : 400;
        if (chord === "j" || chord === "ArrowDown") {
          e.preventDefault();
          sc?.scrollBy({ top: 80 });
          return;
        }
        if (chord === "k" || chord === "ArrowUp") {
          e.preventDefault();
          sc?.scrollBy({ top: -80 });
          return;
        }
        if (chord === "PageDown") {
          e.preventDefault();
          sc?.scrollBy({ top: page });
          return;
        }
        // NB: Space is intentionally NOT a reader-scroll key. It always means
        // "select this message" (bulk select) — even while a message is open — so
        // it falls through to the selection handler below. Page the reader with
        // PageDown / j / ArrowDown instead.
        if (chord === "PageUp") {
          e.preventDefault();
          sc?.scrollBy({ top: -page });
          return;
        }
        if (chord === "Home") {
          e.preventDefault();
          sc?.scrollTo({ top: 0 });
          return;
        }
        if (chord === "End") {
          e.preventDefault();
          sc?.scrollTo({ top: sc.scrollHeight });
          return;
        }
        if (chord === "Enter") {
          // Already open — don't reload/re-mark; just consume it.
          e.preventDefault();
          return;
        }
        if (chord === "Escape") {
          e.preventDefault();
          setReaderFocused(false);
          return;
        }
      }

      // --- list navigation (not remappable, like the TUI table) ---
      if (chord === "j" || chord === "ArrowDown") {
        e.preventDefault();
        moveCursor(idx < 0 ? 0 : Math.min(messages.length - 1, idx + 1));
        return;
      }
      if (chord === "k" || chord === "ArrowUp") {
        e.preventDefault();
        moveCursor(idx <= 0 ? 0 : idx - 1);
        return;
      }
      if (chord === "Enter") {
        e.preventDefault();
        if (idx >= 0) {
          if (bulkMode) toggleSelect(messages[idx].id);
          else void openMessage(messages[idx]);
        }
        return;
      }
      // Bulk selection toggle. Handled here in the list-nav block — not via the
      // remappable action switch — so it fires reliably on the highlighted row.
      // Outside bulk mode it enters bulk mode and selects the current row;
      // inside, it toggles and advances like the TUI.
      // Space always works (the TUI default and what everyone reaches for), in
      // addition to the configured bulk_select key (a literal " " also means
      // space). Space isn't a remappable action here, so this never clashes.
      // Exception: if bulk_select is bound to the SAME key as search (some
      // configs put "s" on both), that key must NOT hijack search in the list —
      // it falls through to the search action below, and bulk-select stays on
      // Space. Otherwise the search key is silently shadowed and never searches.
      const bulkKey =
        keymap.bulkSelect === " " ? "space" : keymap.bulkSelect || "space";
      const bulkKeyShadowsSearch =
        bulkKey !== "space" && bulkKey === keymap.search;
      // Space ALWAYS selects. The configured bulk key selects too — unless it
      // collides with search, in which case that key searches instead (Space
      // still selects, so bulk is never lost).
      if (
        (chord === "space" || (chord === bulkKey && !bulkKeyShadowsSearch)) &&
        messages.length > 0
      ) {
        e.preventDefault();
        const i = idx >= 0 ? idx : 0;
        if (i < messages.length) {
          // Toggle the highlighted message in place — never auto-advance
          // (matches the TUI; you move with j/k). Entering bulk selects it.
          if (!bulkMode) setBulkMode(true);
          toggleSelect(messages[i].id);
          setSelectedId(messages[i].id);
        }
        return;
      }
      if (chord === "Escape") {
        // preventDefault so WKWebView doesn't leave fullscreen on macOS when we
        // consume Escape to close the reader / exit a search (TUI parity).
        if (bulkMode) {
          e.preventDefault();
          exitBulk();
        } else if (detail) {
          e.preventDefault();
          setSelectedId(null);
          setDetail(null);
          setReaderFocused(false);
        } else if (activeQuery || localFilter) {
          // No reader/bulk open: Escape exits the active search back to the
          // default inbox (TUI parity).
          e.preventDefault();
          setQuery("");
          if (localFilter) {
            setLocalFilter(false);
            setMessages(fullMessagesRef.current);
          } else {
            void load("");
          }
        }
        return;
      }
      if (chord === "*") {
        if (bulkMode) {
          e.preventDefault();
          setSelected(new Set(messages.map((m) => m.id)));
        }
        return;
      }
      if (chord === keymap.contentSearch) {
        e.preventDefault();
        // With a message open, "/" searches within its body (like the TUI).
        // Otherwise it focuses the inbox search box.
        if (detail && !bulkMode) {
          setViewHtml(false); // content search runs over the plain-text body
          setCsOpen(true);
          setCsQuery("");
          setCsIndex(0);
        } else {
          searchRef.current?.focus();
        }
        return;
      }

      // --- goto top/bottom ---
      if (chord === keymap.gotoBottom) {
        e.preventDefault();
        moveCursor(messages.length - 1);
        return;
      }
      if (keymap.gotoTop === "gg") {
        if (chord === "g") {
          e.preventDefault();
          const now = Date.now();
          if (now - gPressedAt.current < (keymap.vimTimeoutMs || 1000)) {
            moveCursor(0);
            gPressedAt.current = 0;
          } else {
            gPressedAt.current = now;
          }
          return;
        }
      } else if (chord === keymap.gotoTop) {
        e.preventDefault();
        moveCursor(0);
        return;
      }

      // --- VIM range operations (a3a, d2d, t5t, l2l) ---
      // A single press of a range-op key fires the normal single-message action
      // after vimRangeTimeoutMs; key-count-key applies it to the next N rows.
      // This mirrors the TUI, where the same keys buffer before acting.
      const rangeOps: Record<
        string,
        "archive" | "trash" | "toggleRead" | "manageLabels"
      > = {};
      if (keymap.archive) rangeOps[keymap.archive] = "archive";
      if (keymap.trash) rangeOps[keymap.trash] = "trash";
      if (keymap.toggleRead) rangeOps[keymap.toggleRead] = "toggleRead";
      if (keymap.manageLabels) rangeOps[keymap.manageLabels] = "manageLabels";
      const rangeTimeout = keymap.vimRangeTimeoutMs || 2000;

      // Accumulate a count while a range operation is pending.
      if (vimRange.current && chord >= "0" && chord <= "9") {
        e.preventDefault();
        const st = vimRange.current;
        st.count = st.count * 10 + Number(chord);
        if (st.timer) window.clearTimeout(st.timer);
        st.timer = window.setTimeout(() => {
          if (vimRange.current === st) {
            runVimSingle(st.op, st.startId);
            clearVimRange();
          }
        }, rangeTimeout);
        setBulkProgress(`${st.key}${st.count}… (press ${st.key})`);
        return;
      }

      const rop = rangeOps[chord];
      if (rop) {
        e.preventDefault();
        // With an explicit bulk selection, act on it immediately (the TUI's "VIM
        // BULK FIX") instead of starting a count sequence. A range means "the
        // next N rows", which is meaningless once you've hand-picked a set — and
        // the 2s buffer made the operation look hung (progress bar appears, sits
        // idle, then the toast + removal land seconds later).
        if (bulkMode && selected.size > 0) {
          if (vimRange.current) clearVimRange();
          runVimSingle(rop, selectedId ?? messages[0]?.id ?? "");
          return;
        }
        const pending = vimRange.current;
        if (pending && pending.key === chord) {
          // Second press of the same key completes the range.
          const count = pending.count || 1;
          const startId = pending.startId;
          clearVimRange();
          runVimRange(rop, count, startId);
          return;
        }
        // A different pending op: fire its single fallback now, then start fresh.
        if (pending) {
          const ps = pending;
          clearVimRange();
          runVimSingle(ps.op, ps.startId);
        }
        // Start a new sequence, capturing the row under the cursor as the target.
        const startId = selectedId ?? messages[0]?.id ?? null;
        if (!startId) return;
        const st = { key: chord, op: rop, count: 0, startId, timer: 0 };
        vimRange.current = st;
        st.timer = window.setTimeout(() => {
          if (vimRange.current === st) {
            runVimSingle(rop, startId);
            clearVimRange();
          }
        }, rangeTimeout);
        setBulkProgress(`${chord}… (count then ${chord})`);
        return;
      }

      // Any other key while a range is pending: fire the pending single on its
      // original target, then fall through to handle this key normally.
      if (vimRange.current) {
        const ps = vimRange.current;
        clearVimRange();
        runVimSingle(ps.op, ps.startId);
      }

      // --- config-driven actions ---
      const action = chordAction[chord];
      if (!action) return;
      e.preventDefault();
      switch (action) {
        case "summarize":
          if (detail && aiEnabled && !bulkMode) void summarize(detail.id);
          break;
        case "regenerateSummary":
          // Regenerate the active AI panel: the shown prompt, else the summary.
          // (Uppercase of the summarize key — a real shortcut, e.g. Shift+Y.)
          if (detail && !bulkMode) regenerateActive();
          break;
        case "prompt":
          if (
            aiPromptsEnabled &&
            ((detail && !bulkMode) || (bulkMode && selected.size > 0))
          )
            setPromptsOpen(true);
          break;
        case "archive":
          if (hasSel) void bulkAction("archive");
          else if (detail) void doAction("archive", detail.id);
          break;
        case "quit":
          void backend.Quit();
          break;
        case "trash":
          if (hasSel) void bulkAction("trash");
          else if (detail) void doAction("trash", detail.id);
          break;
        case "toggleRead":
          if (hasSel) void bulkAction("unread");
          else if (detail)
            void doAction(detail.unread ? "read" : "unread", detail.id);
          break;
        case "manageLabels":
          if (hasSel) setBulkLabels(true);
          else if (detail) setLabelsFor(detail.id);
          break;
        case "compose":
          setCompose({ mode: "new" });
          break;
        case "reply":
          if (detail && !bulkMode) setCompose(replyInit(detail));
          break;
        case "forward":
          if (detail && !bulkMode) setCompose(forwardInit(detail));
          break;
        case "search":
          searchRef.current?.focus();
          break;
        case "refresh":
          void load(activeQuery);
          break;
        case "loadMore":
          void loadMore();
          break;
        case "drafts":
          openDrafts();
          break;
        case "openGmail":
          if (detail) openInGmail(detail.id);
          break;
        case "markdown":
          if (detail && detail.html && detail.html.trim())
            setViewHtml((v) => !v);
          break;
        case "bulkMode":
          if (bulkMode) exitBulk();
          else {
            setBulkMode(true);
            if (!selectedId && messages.length > 0)
              setSelectedId(messages[0].id);
          }
          break;
        case "bulkSelect":
          if (bulkMode && selectedId) toggleSelect(selectedId);
          break;
        case "help":
          setShowHelp(true);
          break;
        case "links":
          if (detail) setLinksFor(detail.id);
          break;
        case "replyAll":
          if (detail && !bulkMode) setCompose(replyAllInit(detail));
          break;
        case "saveMessage":
          if (detail) saveMessage(detail.id);
          break;
        case "suggestLabel":
          if (detail && aiEnabled && !bulkMode) void openSuggest(detail.id);
          break;
        case "obsidian":
          if (detail && obsidianOn) sendObsidian(detail.id);
          break;
        case "slack":
          if (detail && slackOn) forwardSlack(detail.id);
          break;
        case "commandMode":
          setCmdOpen(true);
          break;
        case "threading":
          if (detail && threadingOn) void toggleThread();
          break;
        case "savedQueries":
          if (savedQueriesOn) void openQueries();
          break;
        case "saveQuery":
          if (savedQueriesOn && activeQuery) setSaveQueryOpen(true);
          break;
        case "actionPlan":
          if (actionPlanOn) void runActionPlan();
          break;
        case "themePicker":
          if (themesOn) setThemePickerOpen(true);
          break;
        case "move":
          if (hasSel) setBulkMove(true);
          else if (detail && !bulkMode) setMoveFor(detail.id);
          break;
        case "toggleHeaders":
          if (detail) {
            setHeadersHidden(!headersHidden);
            showToast(headersHidden ? "Headers visible" : "Headers hidden");
          }
          break;
        case "searchFrom":
          if (detail) quickSearch("from", detail);
          break;
        case "searchTo":
          if (detail) quickSearch("to", detail);
          break;
        case "searchSubject":
          if (detail) quickSearch("subject", detail);
          break;
        case "undo":
          void runUndo();
          break;
        case "unread":
          void load("is:unread");
          break;
        case "archived":
          void load("in:archive");
          break;
        case "saveRaw":
          if (detail) saveRawMessage(detail.id);
          break;
        case "attachments":
          // Open the keyboard-navigable attachments picker (TUI's PickerAttachments).
          // The reader still shows the same attachments as inline chips for the mouse.
          if (detail && attachments.length > 0) setAttachmentsOpen(true);
          break;
        case "rsvp":
          // Open the keyboard-navigable RSVP picker for the current invite
          // (TUI's V). Only meaningful when the open message is an invite.
          if (detail && invite?.isInvite) setRsvpPickerOpen(true);
          break;
      }
    };
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

