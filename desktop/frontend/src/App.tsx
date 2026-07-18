import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  applyBulkPromptStream,
  applyPromptStream,
  backend,
  DEFAULT_KEYMAP,
  isWails,
  summarizeStream,
  threadSummaryStream,
  type AccountInfo,
  type Attachment,
  type DraftSummary,
  type KeyMap,
  type Label,
  type MessageDetail,
  type MessageSummary,
  type Prompt,
  type SavedQuery,
  type AnalyzerInput,
  type ActionPlanResult,
  type PlanCategory,
  type AnalyzerRule,
} from "./api";
import Compose, { type ComposeInit } from "./Compose";
import LabelsPicker from "./LabelsPicker";
import PromptsPicker from "./PromptsPicker";
import PromptManager from "./PromptManager";
import LinksPicker from "./LinksPicker";
import AccountSwitcher from "./AccountSwitcher";
import HtmlBody from "./HtmlBody";
import HighlightedText from "./HighlightedText";
import Help from "./Help";
import CommandBar, { type CommandDef } from "./CommandBar";
import MoreMenu from "./MoreMenu";
import { Icon, IconBtn } from "./Icons";

const PAGE_SIZE = 50;

// Command palette entries (`:` command mode), mirroring the TUI's command set.
const COMMANDS: CommandDef[] = [
  { names: ["search", "s"], desc: "Gmail search", arg: "<query>" },
  { names: ["unread"], desc: "Show unread only" },
  { names: ["advanced", "adv"], desc: "Advanced search builder" },
  { names: ["local"], desc: "Toggle local filter / Gmail search" },
  { names: ["archive", "a"], desc: "Archive message" },
  { names: ["trash", "d"], desc: "Trash message" },
  { names: ["read"], desc: "Mark read" },
  { names: ["markunread"], desc: "Mark unread" },
  { names: ["labels", "l"], desc: "Manage labels" },
  { names: ["compose", "c"], desc: "New message" },
  { names: ["reply", "r"], desc: "Reply" },
  { names: ["replyall"], desc: "Reply all" },
  { names: ["forward", "f"], desc: "Forward" },
  { names: ["refresh"], desc: "Refresh inbox" },
  { names: ["drafts"], desc: "Drafts" },
  { names: ["links"], desc: "Links in message" },
  { names: ["save"], desc: "Save to file" },
  { names: ["summarize", "sum"], desc: "AI summary" },
  { names: ["prompt"], desc: "Apply a prompt" },
  { names: ["prompts", "prompt-new"], desc: "Manage prompts" },
  { names: ["suggest"], desc: "Suggest labels (AI)" },
  { names: ["obsidian"], desc: "Send to Obsidian" },
  { names: ["slack"], desc: "Forward to Slack" },
  { names: ["gmail", "web"], desc: "Open in Gmail" },
  { names: ["queries", "q"], desc: "Saved searches" },
  { names: ["savequery"], desc: "Save current search" },
  { names: ["plan", "actionplan"], desc: "AI inbox action plan" },
  { names: ["rules"], desc: "Analyzer preference rules" },
  { names: ["move", "mv"], desc: "Move to folder", arg: "[label]" },
  { names: ["draft", "replyai"], desc: "Draft reply (AI)" },
  { names: ["find"], desc: "Find in message", arg: "<text>" },
  { names: ["from"], desc: "Search from this sender" },
  { names: ["to"], desc: "Search to this recipient" },
  { names: ["subject"], desc: "Search this subject" },
  { names: ["headers"], desc: "Toggle headers" },
  { names: ["toolbar"], desc: "Show/hide reader toolbar" },
  { names: ["touch-up", "touchup"], desc: "Reformat message with AI" },
  { names: ["theme", "th"], desc: "Change theme", arg: "[name]" },
  { names: ["help"], desc: "Keyboard shortcuts" },
];

export default function App() {
  const [account, setAccount] = useState("");
  const [initError, setInitError] = useState("");
  const [messages, setMessages] = useState<MessageSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<MessageDetail | null>(null);
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [query, setQuery] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [nextToken, setNextToken] = useState("");
  const [loadingList, setLoadingList] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [aiEnabled, setAiEnabled] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [summarizing, setSummarizing] = useState(false);
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
  const [accounts, setAccounts] = useState<AccountInfo[]>([]);
  const [switching, setSwitching] = useState(false);
  const [viewHtml, setViewHtml] = useState(false);
  const [loadRemote, setLoadRemote] = useState(false);
  const [draftsView, setDraftsView] = useState(false);
  const [drafts, setDrafts] = useState<DraftSummary[]>([]);
  const [loadingDrafts, setLoadingDrafts] = useState(false);
  const [keymap, setKeymap] = useState<KeyMap>(DEFAULT_KEYMAP);
  const [linksFor, setLinksFor] = useState<string | null>(null);
  const [obsidianOn, setObsidianOn] = useState(false);
  const [slackOn, setSlackOn] = useState(false);
  const [suggestFor, setSuggestFor] = useState<string | null>(null);
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [loadingSuggest, setLoadingSuggest] = useState(false);
  const [cmdOpen, setCmdOpen] = useState(false);
  const [threadingOn, setThreadingOn] = useState(false);
  const [threadMsgs, setThreadMsgs] = useState<MessageDetail[] | null>(null);
  const [collapsedMsgs, setCollapsedMsgs] = useState<Set<string>>(new Set());
  const [loadingThread, setLoadingThread] = useState(false);
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
  const [applyingAll, setApplyingAll] = useState(false);
  const [rulesEnabled, setRulesEnabled] = useState(false);
  const [rulesOpen, setRulesOpen] = useState(false);
  const [rules, setRules] = useState<AnalyzerRule[]>([]);
  const [newRule, setNewRule] = useState("");
  const [promptPreview, setPromptPreview] = useState<string | null>(null);
  const [themesOn, setThemesOn] = useState(false);
  const [themePickerOpen, setThemePickerOpen] = useState(false);
  const [themeNames, setThemeNames] = useState<string[]>([]);
  const [currentTheme, setCurrentTheme] = useState("");
  const [generatingReply, setGeneratingReply] = useState(false);
  const [touchUpText, setTouchUpText] = useState<string | null>(null);
  const [touchingUp, setTouchingUp] = useState(false);
  const [headersExpanded, setHeadersExpanded] = useState(false);
  const [moveFor, setMoveFor] = useState<string | null>(null);
  const [moveName, setMoveName] = useState("");
  const [labels, setLabels] = useState<Label[]>([]);
  const [csOpen, setCsOpen] = useState(false);
  const [csQuery, setCsQuery] = useState("");
  const [csIndex, setCsIndex] = useState(0);
  // The reader toolbar is optional — GizTUI is keyboard-first, so users can
  // hide it and drive everything from the keyboard. The choice is persisted.
  const [showToolbar, setShowToolbar] = useState(
    () => localStorage.getItem("giztui.toolbar") !== "off",
  );
  // Local filter mode: narrow the already-loaded list client-side instead of
  // running a remote Gmail search (the TUI's search_toggle_mode).
  const [localFilter, setLocalFilter] = useState(false);
  const [advOpen, setAdvOpen] = useState(false);
  const [adv, setAdv] = useState({
    from: "",
    to: "",
    subject: "",
    hasAttachment: false,
    unreadOnly: false,
    after: "",
    before: "",
  });
  const fullMessagesRef = useRef<MessageSummary[]>([]);
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

  // macOS WKWebView does not give the web content keyboard focus until it is
  // clicked, so global shortcuts appear dead on launch. Focus the app shell on
  // mount and whenever the window regains focus.
  useEffect(() => {
    const focusApp = () => {
      const active = document.activeElement?.tagName;
      if (active !== "INPUT" && active !== "TEXTAREA") rootRef.current?.focus();
    };
    focusApp();
    window.addEventListener("focus", focusApp);
    return () => window.removeEventListener("focus", focusApp);
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

  // applyTheme fetches a theme's palette from the backend and maps it onto the
  // CSS custom properties the stylesheet reads. Empty name = the configured
  // (current) theme, resolved on startup.
  const applyTheme = useCallback(async (name: string) => {
    try {
      const c = await backend.GetThemeColors(name);
      if (!c) return;
      const root = document.documentElement.style;
      const set = (k: string, v: string) => {
        if (v) root.setProperty(k, v);
      };
      // Map the theme palette onto the CSS custom properties the stylesheet
      // actually reads. Elevated surfaces reuse the input background when the
      // theme provides one, otherwise they fall back to the base background.
      const elev = c.inputBg || c.bg;
      set("--bg", c.bg);
      set("--bg-elev", elev);
      set("--bg-row", c.bg);
      set("--bg-selected", c.selectionBg);
      set("--border", c.border);
      set("--text", c.fg);
      set("--text-muted", c.muted);
      set("--accent", c.accent);
      set("--accent-strong", c.primary || c.accent);
      set("--danger", c.danger);
      set("--chip-bg", c.selectionBg || elev);
      set("--chip-text", c.fg);
      set("--unread-dot", c.unread || c.accent);
      if (c.name) setCurrentTheme(c.name);
    } catch {
      /* non-fatal: keep the default palette */
    }
  }, []);

  const load = useCallback(async (q: string) => {
    setLoadingList(true);
    setError("");
    setActiveQuery(q);
    try {
      const list = q
        ? await backend.Search(q, "", PAGE_SIZE)
        : await backend.ListInbox("", PAGE_SIZE);
      const msgs = list.messages ?? [];
      setMessages(msgs);
      fullMessagesRef.current = msgs;
      setNextToken(list.nextPageToken ?? "");
      // Select + preview the first message so the app opens ready to read.
      if (msgs.length > 0) previewRef.current(msgs[0]);
      else {
        setSelectedId(null);
        setDetail(null);
      }
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingList(false);
    }
  }, []);

  const loadMore = useCallback(async () => {
    if (!nextToken || loadingMore) return;
    setLoadingMore(true);
    try {
      const list = activeQuery
        ? await backend.Search(activeQuery, nextToken, PAGE_SIZE)
        : await backend.ListInbox(nextToken, PAGE_SIZE);
      const more = list.messages ?? [];
      fullMessagesRef.current = [...fullMessagesRef.current, ...more];
      setMessages((prev) => [...prev, ...more]);
      setNextToken(list.nextPageToken ?? "");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingMore(false);
    }
  }, [nextToken, loadingMore, activeQuery]);

  // applyLocalFilter narrows the loaded list client-side (subject/from/snippet)
  // without hitting the network; an empty query restores the full list.
  const applyLocalFilter = useCallback((q: string) => {
    const needle = q.trim().toLowerCase();
    const full = fullMessagesRef.current;
    const next = needle
      ? full.filter(
          (m) =>
            m.subject.toLowerCase().includes(needle) ||
            m.from.toLowerCase().includes(needle) ||
            m.snippet.toLowerCase().includes(needle),
        )
      : full;
    setMessages(next);
    if (next.length > 0) previewRef.current(next[0]);
    else {
      setSelectedId(null);
      setDetail(null);
    }
  }, []);

  // buildAdvancedQuery assembles a Gmail search string from the builder fields.
  const buildAdvancedQuery = useCallback((): string => {
    const parts: string[] = [];
    if (adv.from.trim()) parts.push(`from:${adv.from.trim()}`);
    if (adv.to.trim()) parts.push(`to:${adv.to.trim()}`);
    if (adv.subject.trim()) parts.push(`subject:(${adv.subject.trim()})`);
    if (adv.hasAttachment) parts.push("has:attachment");
    if (adv.unreadOnly) parts.push("is:unread");
    if (adv.after.trim()) parts.push(`after:${adv.after.trim().replace(/-/g, "/")}`);
    if (adv.before.trim())
      parts.push(`before:${adv.before.trim().replace(/-/g, "/")}`);
    return parts.join(" ");
  }, [adv]);

  useEffect(() => {
    void (async () => {
      try {
        const ie = await backend.InitError();
        if (ie) {
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
        setObsidianOn(await backend.ObsidianEnabled());
        setSlackOn(await backend.SlackEnabled());
        setThreadingOn(await backend.ThreadingEnabled());
        setSavedQueriesOn(await backend.SavedQueriesEnabled());
        setActionPlanOn(await backend.ActionPlanEnabled());
        setRulesEnabled(await backend.AnalyzerRulesEnabled());
      } catch {
        /* non-fatal */
      }
      try {
        const on = await backend.ThemesEnabled();
        setThemesOn(on);
        if (on) {
          setThemeNames(await backend.ListThemes());
          await applyTheme(""); // apply the configured theme
        }
      } catch {
        /* non-fatal */
      }
      try {
        setLabels(await backend.ListLabels());
      } catch {
        /* non-fatal */
      }
      void load("");
    })();
  }, [load, applyTheme]);

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
      setSelectedId(m.id);
      setLoadingDetail(true);
      setError("");
      setSummary(null);
      setPromptResult(null);
      setAttachments([]);
      setThreadMsgs(null);
      setCollapsedMsgs(new Set());
      setTouchUpText(null);
      setCsOpen(false);
      // Keep keyboard focus on the app shell (not the HTML iframe) so shortcuts
      // keep working while reading.
      requestAnimationFrame(() => {
        const active = document.activeElement?.tagName;
        if (active !== "INPUT" && active !== "TEXTAREA") rootRef.current?.focus();
      });
      try {
        const d = await backend.GetMessage(m.id);
        setDetail(d);
        setViewHtml(!!(d.html && d.html.trim()));
        setLoadRemote(false);
        void backend
          .ListAttachments(m.id)
          .then(setAttachments)
          .catch(() => undefined);
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
    (m: MessageSummary) => void loadMessage(m, false),
    [loadMessage],
  );
  const openMessage = useCallback(
    (m: MessageSummary) => void loadMessage(m, true),
    [loadMessage],
  );
  // Ref so load() can preview the first message without a declaration-order cycle.
  const previewRef = useRef<(m: MessageSummary) => void>(() => {});
  previewRef.current = previewMessage;

  const removeFromList = useCallback(
    (id: string) => {
      setMessages((prev) => prev.filter((x) => x.id !== id));
      if (selectedId === id) {
        setSelectedId(null);
        setDetail(null);
      }
    },
    [selectedId],
  );

  const doAction = useCallback(
    async (action: "archive" | "trash" | "read" | "unread", id: string) => {
      setBusy(true);
      setError("");
      try {
        if (action === "archive") {
          await backend.Archive(id);
          removeFromList(id);
          showToast("Archived");
        } else if (action === "trash") {
          await backend.Trash(id);
          removeFromList(id);
          showToast("Moved to trash");
        } else if (action === "read") {
          await backend.MarkRead(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: false } : x)),
          );
          setDetail((d) => (d && d.id === id ? { ...d, unread: false } : d));
        } else if (action === "unread") {
          await backend.MarkUnread(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: true } : x)),
          );
          setDetail((d) => (d && d.id === id ? { ...d, unread: true } : d));
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [removeFromList, showToast],
  );

  const toggleSelect = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const exitBulk = useCallback(() => {
    setBulkMode(false);
    setSelected(new Set());
  }, []);

  // clearReaderIfRemoved closes the reading pane when the message it shows was
  // just removed from the list (archived/trashed), so a stale email doesn't
  // linger on screen.
  const clearReaderIfRemoved = useCallback(
    (removed: Set<string>) => {
      setDetail((d) => {
        if (d && removed.has(d.id)) {
          setSelectedId(null);
          setSummary(null);
          setThreadMsgs(null);
          return null;
        }
        return d;
      });
    },
    [],
  );

  const bulkAction = useCallback(
    async (action: "archive" | "trash" | "read" | "unread") => {
      const ids = [...selected];
      if (ids.length === 0) return;
      const idSet = new Set(ids);
      setBusy(true);
      setBulkProgress(`${labelForAction(action)} ${ids.length}…`);
      setError("");
      try {
        if (action === "archive") {
          await backend.BulkArchive(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          clearReaderIfRemoved(idSet);
          showToast(`Archived ${ids.length}`);
        } else if (action === "trash") {
          await backend.BulkTrash(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          clearReaderIfRemoved(idSet);
          showToast(`Moved ${ids.length} to trash`);
        } else if (action === "read") {
          await backend.BulkMarkRead(ids);
          setMessages((prev) =>
            prev.map((m) => (idSet.has(m.id) ? { ...m, unread: false } : m)),
          );
          showToast(`Marked ${ids.length} read`);
        } else if (action === "unread") {
          await backend.BulkMarkUnread(ids);
          setMessages((prev) =>
            prev.map((m) => (idSet.has(m.id) ? { ...m, unread: true } : m)),
          );
          showToast(`Marked ${ids.length} unread`);
        }
        setSelected(new Set());
      } catch (e) {
        setError(String(e));
      } finally {
        setBulkProgress("");
        setBusy(false);
      }
    },
    [selected, showToast, clearReaderIfRemoved],
  );

  const summarize = useCallback(async (id: string, force = false) => {
    setSummarizing(true);
    setSummary("");
    setError("");
    try {
      let acc = "";
      const final = await summarizeStream(
        id,
        (tok) => {
          acc += tok;
          setSummary(acc);
        },
        force,
      );
      setSummary(final);
    } catch (e) {
      setError(String(e));
      setSummary(null);
    } finally {
      setSummarizing(false);
    }
  }, []);

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
      setTouchUpText(await backend.TouchUp(id));
    } catch (e) {
      setError(String(e));
    } finally {
      setTouchingUp(false);
    }
  }, []);

  // doMove applies a label and archives the message (Gmail "move to folder").
  const doMove = useCallback(
    async (id: string, name: string) => {
      const label = name.trim();
      if (!label) return;
      setMoveFor(null);
      setMoveName("");
      try {
        await backend.MoveToLabel(id, label);
        setMessages((prev) => prev.filter((m) => m.id !== id));
        if (selectedId === id) {
          setSelectedId(null);
          setDetail(null);
        }
        showToast(`Moved to ${label}`);
      } catch (e) {
        setError(String(e));
      }
    },
    [selectedId, showToast],
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

  const loadDrafts = useCallback(async () => {
    setLoadingDrafts(true);
    setError("");
    try {
      setDrafts(await backend.ListDrafts());
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingDrafts(false);
    }
  }, []);

  const openDrafts = useCallback(() => {
    setDraftsView(true);
    setSelectedId(null);
    setDetail(null);
    void loadDrafts();
  }, [loadDrafts]);

  const openDraft = useCallback(async (d: DraftSummary) => {
    setError("");
    try {
      const det = await backend.GetDraft(d.id);
      setCompose({
        mode: "draft",
        draftId: det.id,
        to: det.to,
        cc: det.cc,
        subject: det.subject,
        body: det.body,
      });
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const openInGmail = useCallback((id: string) => {
    void backend.OpenGmailWeb(id).catch((e) => setError(String(e)));
  }, []);

  const runPrompt = useCallback(
    async (prompt: Prompt) => {
      const bulk = bulkMode && selected.size > 0;
      if (!bulk && !detail) return;
      setPromptsOpen(false);
      setPromptRunning(true);
      setError("");
      if (bulk) {
        setBulkPromptLabel(`${prompt.name} · ${selected.size} messages`);
        setBulkPromptText("");
      } else {
        setPromptLabel(prompt.name);
        setPromptResult("");
      }
      try {
        let acc = "";
        const onTok = (tok: string) => {
          acc += tok;
          if (bulk) setBulkPromptText(acc);
          else setPromptResult(acc);
        };
        const final = bulk
          ? await applyBulkPromptStream([...selected], prompt.id, onTok)
          : await applyPromptStream(detail!.id, prompt.id, onTok);
        if (bulk) setBulkPromptText(final);
        else setPromptResult(final);
      } catch (e) {
        setError(String(e));
        if (bulk) setBulkPromptText(null);
        else setPromptResult(null);
      } finally {
        setPromptRunning(false);
      }
    },
    [detail, bulkMode, selected],
  );

  const replyInit = (d: MessageDetail): ComposeInit => ({
    mode: "reply",
    originalId: d.id,
    to: d.from,
  });

  // Reply-all: reply threaded, adding the original To/Cc recipients as Cc.
  const replyAllInit = (d: MessageDetail): ComposeInit => {
    const extra = [d.to, d.cc]
      .filter(Boolean)
      .join(", ")
      .split(",")
      .map((s) => s.trim())
      .filter((s) => s && !d.from.includes(s));
    return {
      mode: "reply",
      originalId: d.id,
      to: d.from,
      cc: [...new Set(extra)].join(", "),
    };
  };

  const saveMessage = useCallback(
    (id: string) => {
      void backend
        .SaveMessage(id)
        .then((path) => showToast(`Saved to ${path}`))
        .catch((e) => setError(String(e)));
    },
    [showToast],
  );

  const sendObsidian = useCallback(
    (id: string) => {
      showToast("Sending to Obsidian…");
      void backend
        .SendToObsidian(id)
        .then((p) => showToast(p ? `Saved to Obsidian: ${p}` : "Saved to Obsidian"))
        .catch((e) => setError(String(e)));
    },
    [showToast],
  );

  const forwardSlack = useCallback(
    (id: string) => {
      showToast("Forwarding to Slack…");
      void backend
        .ForwardToSlack(id)
        .then(() => showToast("Forwarded to Slack"))
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

  const toggleThread = useCallback(async () => {
    if (!detail) return;
    if (threadMsgs) {
      setThreadMsgs(null);
      return;
    }
    setLoadingThread(true);
    setError("");
    try {
      const msgs = await backend.GetThread(detail.threadId);
      setThreadMsgs(msgs);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingThread(false);
    }
  }, [detail, threadMsgs]);

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

  const runActionPlan = useCallback(async () => {
    setPlanOpen(true);
    setAnalyzing(true);
    setPlan(null);
    setError("");
    try {
      const inputs: AnalyzerInput[] = messages.map((m) => ({
        id: m.id,
        subject: m.subject,
        from: m.from,
        snippet: m.snippet,
      }));
      setPlan(await backend.AnalyzeInbox(inputs));
    } catch (e) {
      setError(String(e));
    } finally {
      setAnalyzing(false);
    }
  }, [messages]);

  const applyCategory = useCallback(
    async (cat: PlanCategory) => {
      const ids = cat.messageIds;
      if (!ids.length) return;
      const idSet = new Set(ids);
      try {
        if (cat.action === "archive") {
          await backend.BulkArchive(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          clearReaderIfRemoved(idSet);
        } else if (cat.action === "trash") {
          await backend.BulkTrash(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          clearReaderIfRemoved(idSet);
        } else if (cat.action === "mark_read") {
          await backend.BulkMarkRead(ids);
          setMessages((prev) =>
            prev.map((m) => (idSet.has(m.id) ? { ...m, unread: false } : m)),
          );
        } else if (cat.action === "label") {
          await backend.BulkApplyLabelByName(ids, cat.label);
        } else {
          return;
        }
        showToast(`${cat.name}: ${cat.action.replace("_", " ")} · ${ids.length}`);
        setPlan((prev) =>
          prev
            ? { ...prev, categories: prev.categories.filter((c) => c !== cat) }
            : prev,
        );
      } catch (e) {
        setError(String(e));
      }
    },
    [showToast, clearReaderIfRemoved],
  );

  // applyAllCategories runs every category's action in one go (the TUI's
  // "confirm & apply the whole plan").
  const applyAllCategories = useCallback(async () => {
    if (!plan) return;
    setApplyingAll(true);
    try {
      for (const c of [...plan.categories]) {
        await applyCategory(c);
      }
      showToast("Applied the whole plan");
    } finally {
      setApplyingAll(false);
    }
  }, [plan, applyCategory, showToast]);

  const openRules = useCallback(async () => {
    setRulesOpen(true);
    try {
      setRules(await backend.ListAnalyzerRules());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const addRule = useCallback(async () => {
    const text = newRule.trim();
    if (!text) return;
    try {
      await backend.SaveAnalyzerRule(text);
      setNewRule("");
      setRules(await backend.ListAnalyzerRules());
    } catch (e) {
      setError(String(e));
    }
  }, [newRule]);

  const deleteRule = useCallback(async (id: number) => {
    try {
      await backend.DeleteAnalyzerRule(id);
      setRules(await backend.ListAnalyzerRules());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const viewAnalyzerPrompt = useCallback(async () => {
    try {
      setPromptPreview(await backend.ViewAnalyzerPrompt());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  // Command palette dispatcher (`:` command mode).
  const executeCommand = useCallback(
    (input: string) => {
      const parts = input.trim().split(/\s+/);
      const cmd = (parts[0] || "").toLowerCase();
      const arg = parts.slice(1).join(" ");
      const d = detail;
      switch (cmd) {
        case "search":
        case "s":
          void load(arg);
          break;
        case "unread":
          void load("is:unread");
          break;
        case "archive":
        case "a":
          if (d) void doAction("archive", d.id);
          break;
        case "trash":
        case "d":
          if (d) void doAction("trash", d.id);
          break;
        case "read":
          if (d) void doAction("read", d.id);
          break;
        case "markunread":
          if (d) void doAction("unread", d.id);
          break;
        case "labels":
        case "l":
          if (d) setLabelsFor(d.id);
          break;
        case "compose":
        case "c":
          setCompose({ mode: "new" });
          break;
        case "reply":
        case "r":
          if (d) setCompose(replyInit(d));
          break;
        case "replyall":
          if (d) setCompose(replyAllInit(d));
          break;
        case "forward":
        case "f":
          if (d) setCompose(forwardInit(d));
          break;
        case "refresh":
          void load(activeQuery);
          break;
        case "drafts":
          openDrafts();
          break;
        case "links":
          if (d) setLinksFor(d.id);
          break;
        case "save":
          if (d) saveMessage(d.id);
          break;
        case "summarize":
        case "sum":
          if (d && aiEnabled) void summarize(d.id);
          break;
        case "prompt":
          if (aiPromptsEnabled && (d || (bulkMode && selected.size > 0)))
            setPromptsOpen(true);
          break;
        case "suggest":
          if (d && aiEnabled) void openSuggest(d.id);
          break;
        case "obsidian":
          if (d && obsidianOn) sendObsidian(d.id);
          break;
        case "slack":
          if (d && slackOn) forwardSlack(d.id);
          break;
        case "gmail":
        case "web":
          if (d) openInGmail(d.id);
          break;
        case "queries":
        case "q":
          if (savedQueriesOn) void openQueries();
          break;
        case "plan":
        case "actionplan":
          if (actionPlanOn) void runActionPlan();
          break;
        case "savequery":
          if (savedQueriesOn && activeQuery) setSaveQueryOpen(true);
          break;
        case "move":
        case "mv":
          if (d) {
            if (arg) void doMove(d.id, arg);
            else {
              setMoveName("");
              setMoveFor(d.id);
            }
          }
          break;
        case "headers":
          if (d) setHeadersExpanded((v) => !v);
          break;
        case "toolbar":
          toggleToolbar();
          break;
        case "advanced":
        case "adv":
          setAdvOpen(true);
          break;
        case "local":
          if (localFilter) {
            setLocalFilter(false);
            setMessages(fullMessagesRef.current);
          } else {
            setLocalFilter(true);
            applyLocalFilter(query);
          }
          break;
        case "touch-up":
        case "touchup":
          if (d && aiEnabled) {
            if (touchUpText !== null) setTouchUpText(null);
            else void touchUp(d.id);
          }
          break;
        case "replyai":
        case "draft":
          if (d && aiEnabled) void generateReply(d);
          break;
        case "from":
          if (d) quickSearch("from", d);
          break;
        case "to":
          if (d) quickSearch("to", d);
          break;
        case "subject":
          if (d) quickSearch("subject", d);
          break;
        case "find":
          if (d) {
            setViewHtml(false);
            setCsQuery(arg);
            setCsIndex(0);
            setCsOpen(true);
          }
          break;
        case "theme":
        case "th":
          if (themesOn) {
            if (arg) {
              void applyTheme(arg);
              showToast(`Theme: ${arg}`);
            } else setThemePickerOpen(true);
          }
          break;
        case "prompts":
        case "prompt-manage":
        case "prompt-new":
          if (aiPromptsEnabled) setPromptManagerOpen(true);
          break;
        case "rules":
          if (rulesEnabled) void openRules();
          break;
        case "help":
          setShowHelp(true);
          break;
        default:
          showToast(`Unknown command: ${cmd}`);
      }
    },
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
      actionPlanOn,
      bulkMode,
      selected,
      showToast,
      doMove,
      generateReply,
      quickSearch,
      themesOn,
      applyTheme,
      rulesEnabled,
      openRules,
      toggleToolbar,
      touchUp,
      touchUpText,
      localFilter,
      applyLocalFilter,
      query,
    ],
  );

  const forwardInit = (d: MessageDetail): ComposeInit => ({
    mode: "new",
    subject: d.subject.startsWith("Fwd:") ? d.subject : `Fwd: ${d.subject}`,
    body: `\n\n---------- Forwarded message ----------\nFrom: ${d.from}\nDate: ${d.date}\nSubject: ${d.subject}\nTo: ${d.to}\n\n${d.plainText}`,
  });

  const downloadAttachment = useCallback(
    async (att: Attachment) => {
      if (!detail) return;
      setBusy(true);
      try {
        const path = await backend.DownloadAttachment(
          detail.id,
          att.attachmentId,
          att.filename,
        );
        showToast(`Saved to ${path}`);
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [detail, showToast],
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
    add(keymap.openGmail, "openGmail");
    add(keymap.bulkMode, "bulkMode");
    add(keymap.bulkSelect, "bulkSelect");
    add(keymap.markdown, "markdown");
    add(keymap.help, "help");
    add(keymap.linkPicker, "links");
    add(keymap.replyAll, "replyAll");
    add(keymap.saveMessage, "saveMessage");
    add(keymap.suggestLabel, "suggestLabel");
    add(keymap.obsidian, "obsidian");
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
    return m;
  }, [keymap]);

  // Global keyboard shortcuts, driven by the user's GizTUI keybindings. List
  // navigation (j/k/arrows/Enter/Esc) mirrors the TUI's native table: j/k move a
  // cursor without opening; Enter opens. This avoids marking mail read while
  // just scanning.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement | null)?.tagName;
      const typing = tag === "INPUT" || tag === "TEXTAREA";
      const chord = e.key === " " ? "space" : e.key;

      if (showHelp) {
        if (e.key === "Escape" || chord === keymap.help) {
          setShowHelp(false);
          e.preventDefault();
        }
        return;
      }
      if (
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
        moveFor ||
        bulkPromptText !== null
      )
        return;
      if (typing) {
        if (e.key === "Escape") (e.target as HTMLElement).blur();
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

      const idx = selectedId
        ? messages.findIndex((m) => m.id === selectedId)
        : -1;
      // Navigation previews the message (shows content, scrolls to it) without
      // marking it read. In bulk mode it only moves the highlight.
      const moveCursor = (i: number) => {
        if (i < 0 || i >= messages.length) return;
        if (bulkMode) setSelectedId(messages[i].id);
        else previewMessage(messages[i]);
      };
      const hasSel = bulkMode && selected.size > 0;

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
      // Bulk selection toggle (default "space"). Handled here in the list-nav
      // block — not via the remappable action switch — so it fires reliably on
      // the highlighted row and advances the cursor like the TUI.
      if (bulkMode && chord === keymap.bulkSelect) {
        e.preventDefault();
        const i = idx >= 0 ? idx : 0;
        if (i < messages.length) {
          toggleSelect(messages[i].id);
          if (i + 1 < messages.length) setSelectedId(messages[i + 1].id);
        }
        return;
      }
      if (chord === "Escape") {
        if (bulkMode) exitBulk();
        else if (detail) {
          setSelectedId(null);
          setDetail(null);
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

      // --- config-driven actions ---
      const action = chordAction[chord];
      if (!action) return;
      e.preventDefault();
      switch (action) {
        case "summarize":
          if (detail && aiEnabled && !bulkMode) void summarize(detail.id);
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
          if (detail && !bulkMode) {
            setMoveName("");
            setMoveFor(detail.id);
          }
          break;
        case "toggleHeaders":
          if (detail) setHeadersExpanded((v) => !v);
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
    themePickerOpen,
    rulesOpen,
    promptPreview,
    advOpen,
    moveFor,
    bulkPromptText,
    actionPlanOn,
    themesOn,
    activeQuery,
    bulkMode,
    selected,
    draftsView,
    openMessage,
    previewMessage,
    doAction,
    bulkAction,
    toggleSelect,
    exitBulk,
    summarize,
    quickSearch,
    toggleThread,
    openQueries,
    runActionPlan,
    saveMessage,
    sendObsidian,
    forwardSlack,
    openSuggest,
    openDrafts,
    openInGmail,
    load,
    loadMore,
  ]);

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
      </div>
    );
  }

  return (
    <div className="app" ref={rootRef} tabIndex={-1}>
      <header className="topbar">
        <div className="brand">
          <span className="logo">✦</span> GizTUI
          <span className="subtitle">Desktop</span>
        </div>
        <form
          className="searchbox"
          onSubmit={(e) => {
            e.preventDefault();
            const q = query.trim();
            searchRef.current?.blur();
            if (localFilter) applyLocalFilter(q);
            else void load(q);
          }}
        >
          <IconBtn
            icon={localFilter ? Icon.filter : Icon.search}
            label={
              localFilter
                ? "Local filter — click for Gmail search"
                : "Gmail search — click to filter loaded list"
            }
            primary={localFilter}
            onClick={() => {
              const next = !localFilter;
              setLocalFilter(next);
              if (next) applyLocalFilter(query);
              else setMessages(fullMessagesRef.current);
            }}
          />
          <input
            ref={searchRef}
            type="text"
            placeholder={
              localFilter
                ? "Filter loaded messages…"
                : "Search mail (press / or s) — Gmail operators: from:, has:attachment…"
            }
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              if (localFilter) applyLocalFilter(e.target.value);
            }}
          />
          {!localFilter && (
            <button
              type="submit"
              className="icon-btn primary"
              aria-label="Search"
              data-tip="Search"
            >
              {Icon.search}
            </button>
          )}
          <IconBtn
            icon={Icon.sliders}
            label="Advanced search"
            onClick={() => setAdvOpen(true)}
          />
          {(activeQuery || (localFilter && query)) && (
            <IconBtn
              icon={Icon.x}
              label="Clear search"
              onClick={() => {
                setQuery("");
                if (localFilter) setMessages(fullMessagesRef.current);
                else void load("");
              }}
            />
          )}
        </form>
        <div className="account">
          {!isWails() && <span className="badge">mock</span>}
          <AccountSwitcher
            accounts={accounts}
            email={account}
            switching={switching}
            onSwitch={(a) => void switchAccount(a)}
          />
          {/* Same IconBtn format as the reader/bulk toolbars for one consistent
              button language across the app. */}
          <div className="actions topbar-actions">
            <IconBtn
              icon={Icon.edit}
              label="Compose (c)"
              primary
              onClick={() => setCompose({ mode: "new" })}
            />
            <IconBtn
              icon={Icon.drafts}
              label="Drafts (D)"
              primary={draftsView}
              onClick={() => {
                if (draftsView) setDraftsView(false);
                else openDrafts();
              }}
            />
            <IconBtn
              icon={Icon.checkAll}
              label="Select mode (v)"
              primary={bulkMode}
              onClick={() => {
                if (bulkMode) exitBulk();
                else {
                  setBulkMode(true);
                  if (!selectedId && messages.length > 0)
                    setSelectedId(messages[0].id);
                }
              }}
            />
            {savedQueriesOn && (
              <IconBtn
                icon={Icon.bookmark}
                label="Saved searches (Q)"
                onClick={() => void openQueries()}
              />
            )}
            <IconBtn
              icon={Icon.layout}
              label={
                showToolbar
                  ? "Hide reader toolbar (:toolbar)"
                  : "Show reader toolbar (:toolbar)"
              }
              primary={showToolbar}
              onClick={toggleToolbar}
            />
            <IconBtn
              icon={Icon.help}
              label="Shortcuts (?)"
              onClick={() => setShowHelp(true)}
            />
            <IconBtn
              icon={Icon.refresh}
              label="Refresh (R)"
              onClick={() => void load(activeQuery)}
            />
          </div>
        </div>
      </header>

      {error && <div className="error-banner">{error}</div>}
      {toast && <div className="toast">{toast}</div>}

      <div className="body">
        <aside className="list">
          {draftsView ? (
            <>
              <div className="bulk-bar">
                <span className="bulk-count">Drafts</span>
                <div className="bulk-actions">
                  <button className="tiny ghost" onClick={() => void loadDrafts()}>
                    Refresh
                  </button>
                  <button className="tiny ghost" onClick={() => setDraftsView(false)}>
                    Back to inbox
                  </button>
                </div>
              </div>
              {loadingDrafts ? (
                <div className="placeholder">Loading…</div>
              ) : drafts.length === 0 ? (
                <div className="placeholder">No drafts</div>
              ) : (
                <ul>
                  {drafts.map((d) => (
                    <li
                      key={d.id}
                      className="row"
                      onClick={() => void openDraft(d)}
                    >
                      <div className="row-top">
                        <span className="from">
                          {d.to ? `To: ${d.to}` : "(no recipient)"}
                        </span>
                      </div>
                      <div className="subject">{d.subject || "(no subject)"}</div>
                      <div className="snippet">{d.snippet}</div>
                    </li>
                  ))}
                </ul>
              )}
            </>
          ) : (
            <>
          {bulkMode && (
            <div className="bulk-bar">
              <div className="bulk-top">
                <span className="bulk-count">{selected.size} selected</span>
                {/* Same IconBtn format as the reader toolbar for consistency. */}
                <div className="actions">
                  <IconBtn
                    icon={Icon.archive}
                    label="Archive"
                    disabled={busy || selected.size === 0}
                    onClick={() => void bulkAction("archive")}
                  />
                  <IconBtn
                    icon={Icon.trash}
                    label="Trash"
                    danger
                    disabled={busy || selected.size === 0}
                    onClick={() => void bulkAction("trash")}
                  />
                  <IconBtn
                    icon={Icon.mailOpen}
                    label="Mark read"
                    disabled={busy || selected.size === 0}
                    onClick={() => void bulkAction("read")}
                  />
                  <IconBtn
                    icon={Icon.mail}
                    label="Mark unread"
                    disabled={busy || selected.size === 0}
                    onClick={() => void bulkAction("unread")}
                  />
                  <IconBtn
                    icon={Icon.label}
                    label="Label…"
                    disabled={busy || selected.size === 0}
                    onClick={() => setBulkLabels(true)}
                  />
                  <span className="actions-sep" />
                  <IconBtn
                    icon={Icon.checkAll}
                    label="Select all"
                    disabled={busy}
                    onClick={() =>
                      setSelected(new Set(messages.map((m) => m.id)))
                    }
                  />
                  <IconBtn
                    icon={Icon.check}
                    label="Done"
                    primary
                    onClick={exitBulk}
                  />
                </div>
              </div>
              {bulkProgress && (
                <div className="bulk-progress">
                  <div className="bulk-progress-bar" />
                  <span className="bulk-progress-label">{bulkProgress}</span>
                </div>
              )}
            </div>
          )}
          {loadingList ? (
            <div className="placeholder">Loading…</div>
          ) : messages.length === 0 ? (
            <div className="placeholder">No messages</div>
          ) : (
            <>
              <ul>
                {messages.map((m) => (
                  <li
                    key={m.id}
                    className={
                      "row" +
                      (m.id === selectedId ? " selected" : "") +
                      (m.unread ? " unread" : "") +
                      (bulkMode && selected.has(m.id) ? " checked" : "")
                    }
                    onClick={() =>
                      bulkMode ? toggleSelect(m.id) : void openMessage(m)
                    }
                  >
                    <div className="row-top">
                      {bulkMode && (
                        <span className="row-check">
                          {selected.has(m.id) ? "☑" : "☐"}
                        </span>
                      )}
                      <span className="from">{displayName(m.from)}</span>
                      <span className="date">{formatDate(m.date)}</span>
                    </div>
                    <div className="subject">{m.subject || "(no subject)"}</div>
                    <div className="snippet">{m.snippet}</div>
                    {m.labels.length > 0 && (
                      <div className="labels">
                        {m.labels.map((l) => (
                          <span key={l} className="label-chip">
                            {l}
                          </span>
                        ))}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
              {nextToken && (
                <button
                  className="load-more"
                  disabled={loadingMore}
                  onClick={() => void loadMore()}
                >
                  {loadingMore ? "Loading…" : "Load more (N)"}
                </button>
              )}
            </>
          )}
            </>
          )}
        </aside>

        <main className="reader">
          {detail ? (
            <>
              <div className="reader-head">
                <h2>{detail.subject || "(no subject)"}</h2>
                <div className="meta">
                  <div>
                    <strong>{displayName(detail.from)}</strong>{" "}
                    <span className="muted">{emailAddr(detail.from)}</span>
                  </div>
                  <div className="muted">to {detail.to}</div>
                  <div className="muted">{formatFull(detail.date)}</div>
                  {headersExpanded && (
                    <div className="headers-detail">
                      {detail.cc && (
                        <div className="muted">cc {detail.cc}</div>
                      )}
                      <div className="muted">thread {detail.threadId}</div>
                      <div className="muted">id {detail.id}</div>
                    </div>
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
                {showToolbar && (
                <div className="actions">
                  {/* Primary actions stay visible; everything else collapses
                      into the "⋯" overflow so the toolbar never wraps.
                      The whole bar is optional (keyboard-first) — hide it from
                      the topbar ▤ toggle or :toolbar. */}
                  <IconBtn
                    icon={Icon.reply}
                    label="Reply"
                    primary
                    onClick={() => setCompose(replyInit(detail))}
                  />
                  <IconBtn
                    icon={Icon.forward}
                    label="Forward"
                    onClick={() => setCompose(forwardInit(detail))}
                  />
                  <IconBtn
                    icon={Icon.label}
                    label="Labels"
                    onClick={() => setLabelsFor(detail.id)}
                  />
                  <span className="actions-sep" />
                  <IconBtn
                    icon={Icon.archive}
                    label="Archive"
                    disabled={busy}
                    onClick={() => void doAction("archive", detail.id)}
                  />
                  <IconBtn
                    icon={Icon.trash}
                    label="Trash"
                    danger
                    disabled={busy}
                    onClick={() => void doAction("trash", detail.id)}
                  />
                  <IconBtn
                    icon={detail.unread ? Icon.mailOpen : Icon.mail}
                    label={detail.unread ? "Mark read" : "Mark unread"}
                    disabled={busy}
                    onClick={() =>
                      void doAction(detail.unread ? "read" : "unread", detail.id)
                    }
                  />
                  {detail.html && detail.html.trim() && (
                    <IconBtn
                      icon={viewHtml ? Icon.text : Icon.code}
                      label={viewHtml ? "Show plain text" : "Show HTML"}
                      onClick={() => setViewHtml((v) => !v)}
                    />
                  )}
                  {threadingOn && (
                    <IconBtn
                      icon={Icon.thread}
                      label={threadMsgs ? "Hide conversation" : "Show conversation"}
                      primary={!!threadMsgs}
                      onClick={() => void toggleThread()}
                    />
                  )}
                  <span className="actions-sep" />
                  <MoreMenu
                    items={[
                      {
                        icon: Icon.summarize,
                        label: summarizing ? "Summarizing…" : "Summarize (AI)",
                        disabled: summarizing,
                        hidden: !aiEnabled,
                        onClick: () => void summarize(detail.id),
                      },
                      {
                        icon: Icon.prompt,
                        label: promptRunning ? "Running…" : "Apply a prompt",
                        disabled: promptRunning,
                        hidden: !aiPromptsEnabled,
                        onClick: () => setPromptsOpen(true),
                      },
                      {
                        icon: Icon.reply,
                        label: generatingReply ? "Drafting…" : "Draft reply (AI)",
                        disabled: generatingReply,
                        hidden: !aiEnabled,
                        onClick: () => void generateReply(detail),
                      },
                      {
                        icon: Icon.summarize,
                        label: touchingUp
                          ? "Reformatting…"
                          : touchUpText !== null
                            ? "Show original"
                            : "Touch-up (AI)",
                        disabled: touchingUp,
                        hidden: !aiEnabled,
                        onClick: () =>
                          touchUpText !== null
                            ? setTouchUpText(null)
                            : void touchUp(detail.id),
                      },
                      {
                        icon: Icon.tag2,
                        label: "Suggest labels (AI)",
                        hidden: !aiEnabled,
                        onClick: () => void openSuggest(detail.id),
                      },
                      {
                        icon: Icon.folder,
                        label: "Move to…",
                        onClick: () => {
                          setMoveName("");
                          setMoveFor(detail.id);
                        },
                      },
                      {
                        icon: Icon.search,
                        label: "Search from sender",
                        onClick: () => quickSearch("from", detail),
                      },
                      {
                        icon: Icon.link,
                        label: "Links",
                        onClick: () => setLinksFor(detail.id),
                      },
                      {
                        icon: Icon.obsidian,
                        label: "Send to Obsidian",
                        hidden: !obsidianOn,
                        onClick: () => sendObsidian(detail.id),
                      },
                      {
                        icon: Icon.slack,
                        label: "Forward to Slack",
                        hidden: !slackOn,
                        onClick: () => forwardSlack(detail.id),
                      },
                      {
                        icon: Icon.save,
                        label: "Save to file",
                        onClick: () => saveMessage(detail.id),
                      },
                      {
                        icon: Icon.text,
                        label: headersExpanded ? "Hide headers" : "Show headers",
                        onClick: () => setHeadersExpanded((v) => !v),
                      },
                      {
                        icon: Icon.external,
                        label: "Open in Gmail",
                        onClick: () => openInGmail(detail.id),
                      },
                    ]}
                  />
                </div>
                )}
                {attachments.length > 0 && (
                  <div className="attach-bar">
                    {attachments.map((att) => (
                      <button
                        key={att.attachmentId}
                        className="attach-chip"
                        disabled={busy}
                        title={`${att.mimeType} · ${formatSize(att.size)}`}
                        onClick={() => void downloadAttachment(att)}
                      >
                        📎 {att.filename}
                        <span className="attach-size">{formatSize(att.size)}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <div className="reader-body">
                {(summarizing || summary) && (
                  <div className="summary-panel">
                    <div className="summary-head">
                      <span>✦ AI summary</span>
                      <span className="summary-head-actions">
                        {summary && !summarizing && (
                          <button
                            className="ghost tiny"
                            title="Regenerate (ignore cache)"
                            onClick={() => void summarize(detail.id, true)}
                          >
                            regenerate
                          </button>
                        )}
                        {summary && (
                          <button
                            className="ghost tiny"
                            onClick={() => setSummary(null)}
                          >
                            dismiss
                          </button>
                        )}
                      </span>
                    </div>
                    {summarizing && !summary ? (
                      <div className="muted">Generating…</div>
                    ) : (
                      <pre className="summary-text">
                        {summary}
                        {summarizing && <span className="caret">▍</span>}
                      </pre>
                    )}
                  </div>
                )}
                {(promptRunning || promptResult) && (
                  <div className="summary-panel prompt-panel">
                    <div className="summary-head">
                      <span>✦ {promptLabel}</span>
                      {promptResult && !promptRunning && (
                        <button
                          className="ghost tiny"
                          onClick={() => setPromptResult(null)}
                        >
                          dismiss
                        </button>
                      )}
                    </div>
                    {promptRunning && !promptResult ? (
                      <div className="muted">Generating…</div>
                    ) : (
                      <pre className="summary-text">
                        {promptResult}
                        {promptRunning && <span className="caret">▍</span>}
                      </pre>
                    )}
                  </div>
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
                        const total = countMatches(
                          detail.plainText || "",
                          csQuery,
                        );
                        if (e.key === "Escape") {
                          setCsOpen(false);
                          setCsQuery("");
                        } else if (e.key === "Enter") {
                          e.preventDefault();
                          if (total > 0)
                            setCsIndex((i) =>
                              e.shiftKey
                                ? (i - 1 + total) % total
                                : (i + 1) % total,
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
                        const total = countMatches(
                          detail.plainText || "",
                          csQuery,
                        );
                        if (total > 0)
                          setCsIndex((i) => (i - 1 + total) % total);
                      }}
                    >
                      ↑
                    </button>
                    <button
                      className="tiny"
                      title="Next (Enter)"
                      onClick={() => {
                        const total = countMatches(
                          detail.plainText || "",
                          csQuery,
                        );
                        if (total > 0) setCsIndex((i) => (i + 1) % total);
                      }}
                    >
                      ↓
                    </button>
                    <button
                      className="ghost tiny"
                      onClick={() => {
                        setCsOpen(false);
                        setCsQuery("");
                      }}
                    >
                      ✕
                    </button>
                  </div>
                )}
                {touchUpText !== null ? (
                  <div className="touchup">
                    <div className="touchup-head">
                      <span>✦ Reformatted by AI</span>
                      <button
                        className="ghost tiny"
                        onClick={() => setTouchUpText(null)}
                      >
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
                            setCollapsedMsgs(
                              new Set(threadMsgs.map((m) => m.id)),
                            )
                          }
                        >
                          Collapse all
                        </button>
                        {aiEnabled && (
                          <button
                            className="tiny"
                            disabled={summarizing}
                            onClick={() => void summarizeThread()}
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
                            <span className="conv-caret">
                              {collapsed ? "▸" : "▾"}
                            </span>
                            <strong>{displayName(m.from)}</strong>
                            {collapsed && (
                              <span className="conv-snippet">
                                {(m.plainText || "").slice(0, 80)}
                              </span>
                            )}
                            <span className="conv-date muted">
                              {formatFull(m.date)}
                            </span>
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
                        <button
                          className="tiny"
                          onClick={() => setLoadRemote(true)}
                        >
                          Load images
                        </button>
                      </div>
                    )}
                    <HtmlBody html={detail.html} loadRemote={loadRemote} />
                  </div>
                ) : (
                  <pre className="plain">
                    {csOpen && csQuery ? (
                      <HighlightedText
                        text={detail.plainText || "(empty body)"}
                        query={csQuery}
                        activeIndex={csIndex}
                      />
                    ) : (
                      detail.plainText || "(empty body)"
                    )}
                  </pre>
                )}
              </div>
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
      </div>

      {compose && (
        <Compose
          init={compose}
          onClose={() => setCompose(null)}
          onSent={(msg) => {
            setCompose(null);
            showToast(msg);
            if (draftsView) void loadDrafts();
          }}
        />
      )}
      {labelsFor && (
        <LabelsPicker
          messageId={labelsFor}
          onClose={() => setLabelsFor(null)}
          onChanged={() => {
            if (selectedId === labelsFor) void openMessage({ id: labelsFor } as MessageSummary);
          }}
        />
      )}
      {bulkLabels && (
        <LabelsPicker
          bulkIds={[...selected]}
          onClose={() => setBulkLabels(false)}
          onChanged={() => undefined}
        />
      )}
      {promptsOpen && (
        <PromptsPicker
          onClose={() => setPromptsOpen(false)}
          onPick={(p) => void runPrompt(p)}
          onManage={
            aiPromptsEnabled
              ? () => {
                  setPromptsOpen(false);
                  setPromptManagerOpen(true);
                }
              : undefined
          }
        />
      )}
      {promptManagerOpen && (
        <PromptManager
          aiEnabled={aiEnabled}
          onClose={() => setPromptManagerOpen(false)}
          onChanged={() => {
            /* prompts reload themselves inside the manager */
          }}
        />
      )}
      {linksFor && (
        <LinksPicker messageId={linksFor} onClose={() => setLinksFor(null)} />
      )}
      {suggestFor && (
        <div className="modal-overlay" onClick={() => setSuggestFor(null)}>
          <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>✦ Suggested labels</h3>
              <button className="ghost" onClick={() => setSuggestFor(null)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              {loadingSuggest ? (
                <div className="placeholder">Thinking…</div>
              ) : suggestions.length === 0 ? (
                <div className="placeholder">No suggestions</div>
              ) : (
                <div className="labels">
                  {suggestions.map((s) => (
                    <button
                      key={s}
                      className="label-chip suggest-chip"
                      onClick={() => applySuggestion(s)}
                    >
                      + {s}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div className="modal-foot">
              <button onClick={() => setSuggestFor(null)}>Done</button>
            </div>
          </div>
        </div>
      )}
      {queriesOpen && (
        <div className="modal-overlay" onClick={() => setQueriesOpen(false)}>
          <div className="modal narrow" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Saved searches</h3>
              <button className="ghost" onClick={() => setQueriesOpen(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="label-list">
                {savedQueries.length === 0 ? (
                  <div className="placeholder">No saved searches</div>
                ) : (
                  savedQueries.map((q) => (
                    <div key={q.id} className="query-row">
                      <button className="query-main" onClick={() => runQuery(q)}>
                        <span className="prompt-name">{q.name}</span>
                        <span className="prompt-desc">{q.query}</span>
                      </button>
                      <button
                        className="ghost tiny"
                        onClick={() => void deleteQuery(q.id)}
                      >
                        ✕
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>
            <div className="modal-foot">
              {activeQuery && (
                <button
                  className="ghost"
                  onClick={() => {
                    setQueriesOpen(false);
                    setSaveQueryOpen(true);
                  }}
                >
                  Save current search
                </button>
              )}
              <button onClick={() => setQueriesOpen(false)}>Done</button>
            </div>
          </div>
        </div>
      )}
      {saveQueryOpen && (
        <div className="modal-overlay" onClick={() => setSaveQueryOpen(false)}>
          <div
            className="modal narrow"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              if (e.key === "Escape") setSaveQueryOpen(false);
              else if (e.key === "Enter") doSaveQuery();
            }}
          >
            <div className="modal-head">
              <h3>Save search</h3>
              <button className="ghost" onClick={() => setSaveQueryOpen(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="field">
                <label>Name</label>
                <input
                  value={saveQueryName}
                  onChange={(e) => setSaveQueryName(e.target.value)}
                  placeholder="e.g. Unread from team"
                  autoFocus
                />
              </div>
              <div className="field readonly">
                <label>Query</label>
                <div className="ro-value">{activeQuery}</div>
              </div>
            </div>
            <div className="modal-foot">
              <button className="ghost" onClick={() => setSaveQueryOpen(false)}>
                Cancel
              </button>
              <button onClick={doSaveQuery} disabled={!saveQueryName.trim()}>
                Save
              </button>
            </div>
          </div>
        </div>
      )}
      {bulkPromptText !== null && (
        <div className="modal-overlay" onClick={() => setBulkPromptText(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>✦ {bulkPromptLabel}</h3>
              <button className="ghost" onClick={() => setBulkPromptText(null)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              {promptRunning && !bulkPromptText ? (
                <div className="placeholder">Generating…</div>
              ) : (
                <pre className="summary-text">
                  {bulkPromptText}
                  {promptRunning && <span className="caret">▍</span>}
                </pre>
              )}
            </div>
            <div className="modal-foot">
              <button onClick={() => setBulkPromptText(null)}>Done</button>
            </div>
          </div>
        </div>
      )}
      {planOpen && (
        <div className="modal-overlay" onClick={() => setPlanOpen(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>✦ Inbox action plan</h3>
              <span className="summary-head-actions">
                {rulesEnabled && (
                  <button className="ghost tiny" onClick={() => void openRules()}>
                    ⚙ Rules
                  </button>
                )}
                <button
                  className="ghost tiny"
                  onClick={() => void viewAnalyzerPrompt()}
                >
                  View prompt
                </button>
                <button className="ghost" onClick={() => setPlanOpen(false)}>
                  ✕
                </button>
              </span>
            </div>
            <div className="modal-body">
              {analyzing ? (
                <div className="placeholder">Analyzing your inbox…</div>
              ) : !plan || plan.categories.length === 0 ? (
                <div className="placeholder">
                  {plan ? "Nothing to act on" : "No plan"}
                </div>
              ) : (
                <>
                  <div className="plan-summary muted">
                    Analyzed {plan.totalAnalyzed} · {plan.readManually} to read
                    manually
                  </div>
                  <div className="plan-list">
                    {plan.categories.map((c) => (
                      <div key={c.name} className="plan-cat">
                        <div className="plan-cat-main">
                          <div className="plan-cat-title">
                            <span className={"prio prio-" + c.priority}>
                              {c.priority}
                            </span>
                            <strong>{c.name}</strong>
                            <span className="plan-count">
                              {c.messageIds.length}
                            </span>
                          </div>
                          <div className="plan-cat-desc muted">
                            {c.description}
                          </div>
                        </div>
                        <button
                          className="tiny"
                          disabled={c.action === "none" || c.action === "prompt"}
                          onClick={() => void applyCategory(c)}
                        >
                          {c.action === "label"
                            ? `Label "${c.label}"`
                            : c.action.replace("_", " ")}
                        </button>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>
            <div className="modal-foot">
              {plan && plan.categories.length > 0 && (
                <button
                  className="ghost"
                  disabled={applyingAll}
                  onClick={() => void applyAllCategories()}
                >
                  {applyingAll ? "Applying…" : "Apply all"}
                </button>
              )}
              <button onClick={() => setPlanOpen(false)}>Done</button>
            </div>
          </div>
        </div>
      )}
      {rulesOpen && (
        <div className="modal-overlay" onClick={() => setRulesOpen(false)}>
          <div
            className="modal narrow"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              if (e.key === "Escape") setRulesOpen(false);
            }}
          >
            <div className="modal-head">
              <h3>Analyzer rules</h3>
              <button className="ghost" onClick={() => setRulesOpen(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="muted plan-summary">
                Natural-language preferences the analyzer follows when planning.
              </div>
              <div className="label-list">
                {rules.length === 0 ? (
                  <div className="placeholder">No rules yet</div>
                ) : (
                  rules.map((r) => (
                    <div key={r.id} className="prompt-manage-row">
                      <span className="rule-text">{r.text}</span>
                      <button
                        className="ghost tiny danger"
                        title="Delete"
                        onClick={() => void deleteRule(r.id)}
                      >
                        🗑
                      </button>
                    </div>
                  ))
                )}
              </div>
              <div className="field">
                <input
                  value={newRule}
                  onChange={(e) => setNewRule(e.target.value)}
                  placeholder="e.g. Always archive newsletters"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") void addRule();
                  }}
                />
              </div>
            </div>
            <div className="modal-foot">
              <button onClick={() => void addRule()} disabled={!newRule.trim()}>
                Add rule
              </button>
            </div>
          </div>
        </div>
      )}
      {promptPreview !== null && (
        <div className="modal-overlay" onClick={() => setPromptPreview(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Analyzer prompt</h3>
              <button className="ghost" onClick={() => setPromptPreview(null)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <pre className="summary-text">{promptPreview}</pre>
            </div>
            <div className="modal-foot">
              <button onClick={() => setPromptPreview(null)}>Close</button>
            </div>
          </div>
        </div>
      )}
      {cmdOpen && (
        <CommandBar
          commands={COMMANDS}
          onRun={executeCommand}
          onClose={() => setCmdOpen(false)}
        />
      )}
      {themePickerOpen && (
        <div
          className="modal-overlay"
          onClick={() => setThemePickerOpen(false)}
        >
          <div
            className="modal narrow"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              if (e.key === "Escape") setThemePickerOpen(false);
            }}
          >
            <div className="modal-head">
              <h3>Theme</h3>
              <button className="ghost" onClick={() => setThemePickerOpen(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="theme-list">
                {themeNames.map((name) => (
                  <button
                    key={name}
                    className={
                      "theme-item" + (name === currentTheme ? " active" : "")
                    }
                    onClick={() => {
                      void applyTheme(name);
                      setThemePickerOpen(false);
                      showToast(`Theme: ${name}`);
                    }}
                  >
                    <span className="theme-dot" />
                    {name}
                    {name === currentTheme && (
                      <span className="theme-check">✓</span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
      {moveFor && (
        <div
          className="modal-overlay"
          onClick={() => setMoveFor(null)}
        >
          <div
            className="modal narrow"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              if (e.key === "Escape") setMoveFor(null);
              else if (e.key === "Enter") void doMove(moveFor, moveName);
            }}
          >
            <div className="modal-head">
              <h3>Move to folder</h3>
              <button className="ghost" onClick={() => setMoveFor(null)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="field">
                <label>Label (applies it and archives the message)</label>
                <input
                  value={moveName}
                  onChange={(e) => setMoveName(e.target.value)}
                  placeholder="e.g. Work / Receipts"
                  list="move-label-list"
                  autoFocus
                />
                <datalist id="move-label-list">
                  {labels.map((l) => (
                    <option key={l.id} value={l.name} />
                  ))}
                </datalist>
              </div>
            </div>
            <div className="modal-foot">
              <button className="ghost" onClick={() => setMoveFor(null)}>
                Cancel
              </button>
              <button
                onClick={() => void doMove(moveFor, moveName)}
                disabled={!moveName.trim()}
              >
                Move
              </button>
            </div>
          </div>
        </div>
      )}
      {advOpen && (
        <div className="modal-overlay" onClick={() => setAdvOpen(false)}>
          <div
            className="modal narrow"
            onClick={(e) => e.stopPropagation()}
            onKeyDown={(e) => {
              if (e.key === "Escape") setAdvOpen(false);
            }}
          >
            <div className="modal-head">
              <h3>Advanced search</h3>
              <button className="ghost" onClick={() => setAdvOpen(false)}>
                ✕
              </button>
            </div>
            <div className="modal-body">
              <div className="field">
                <label>From</label>
                <input
                  value={adv.from}
                  onChange={(e) => setAdv({ ...adv, from: e.target.value })}
                  placeholder="sender@example.com"
                  autoFocus
                />
              </div>
              <div className="field">
                <label>To</label>
                <input
                  value={adv.to}
                  onChange={(e) => setAdv({ ...adv, to: e.target.value })}
                  placeholder="recipient@example.com"
                />
              </div>
              <div className="field">
                <label>Subject</label>
                <input
                  value={adv.subject}
                  onChange={(e) => setAdv({ ...adv, subject: e.target.value })}
                  placeholder="words in the subject"
                />
              </div>
              <div className="adv-row">
                <label className="adv-check">
                  <input
                    type="checkbox"
                    checked={adv.hasAttachment}
                    onChange={(e) =>
                      setAdv({ ...adv, hasAttachment: e.target.checked })
                    }
                  />
                  Has attachment
                </label>
                <label className="adv-check">
                  <input
                    type="checkbox"
                    checked={adv.unreadOnly}
                    onChange={(e) =>
                      setAdv({ ...adv, unreadOnly: e.target.checked })
                    }
                  />
                  Unread only
                </label>
              </div>
              <div className="adv-row">
                <div className="field">
                  <label>After</label>
                  <input
                    type="date"
                    value={adv.after}
                    onChange={(e) => setAdv({ ...adv, after: e.target.value })}
                  />
                </div>
                <div className="field">
                  <label>Before</label>
                  <input
                    type="date"
                    value={adv.before}
                    onChange={(e) => setAdv({ ...adv, before: e.target.value })}
                  />
                </div>
              </div>
              <div className="field readonly">
                <label>Query preview</label>
                <div className="ro-value">
                  {buildAdvancedQuery() || "(empty)"}
                </div>
              </div>
            </div>
            <div className="modal-foot">
              <button className="ghost" onClick={() => setAdvOpen(false)}>
                Cancel
              </button>
              <button
                onClick={() => {
                  const q = buildAdvancedQuery();
                  if (!q) return;
                  setLocalFilter(false);
                  setQuery(q);
                  setAdvOpen(false);
                  void load(q);
                }}
                disabled={!buildAdvancedQuery()}
              >
                Search
              </button>
            </div>
          </div>
        </div>
      )}
      {showHelp && <Help onClose={() => setShowHelp(false)} />}
    </div>
  );
}

// --- helpers -----------------------------------------------------------------

function displayName(from: string): string {
  const m = from.match(/^\s*"?([^"<]+?)"?\s*</);
  if (m) return m[1].trim();
  return from.split("@")[0] || from;
}

function emailAddr(from: string): string {
  const m = from.match(/<([^>]+)>/);
  return m ? m[1] : from;
}

// labelForAction is the present-participle verb shown while a bulk action runs.
function labelForAction(action: string): string {
  switch (action) {
    case "archive":
      return "Archiving";
    case "trash":
      return "Trashing";
    case "read":
      return "Marking read";
    case "unread":
      return "Marking unread";
    default:
      return "Working on";
  }
}

// cleanSubject strips Re:/Fwd: prefixes so a subject search matches the thread.
function cleanSubject(subject: string): string {
  return subject.replace(/^(\s*(re|fwd|fw)\s*:\s*)+/i, "").trim() || subject;
}

// countMatches returns how many times query occurs in text (case-insensitive).
function countMatches(text: string, query: string): number {
  if (!query) return 0;
  const q = query.toLowerCase();
  const t = text.toLowerCase();
  let n = 0;
  let i = t.indexOf(q);
  while (i !== -1) {
    n++;
    i = t.indexOf(q, i + q.length);
  }
  return n;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const mins = Math.floor((Date.now() - d.getTime()) / 60000);
  if (mins < 1) return "now";
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d`;
  return d.toLocaleDateString([], { month: "short", day: "numeric" });
}

function formatFull(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString();
}

function formatSize(bytes: number): string {
  if (bytes <= 0) return "";
  const units = ["B", "KB", "MB", "GB"];
  let n = bytes;
  let u = 0;
  while (n >= 1024 && u < units.length - 1) {
    n /= 1024;
    u++;
  }
  return `${n.toFixed(u === 0 ? 0 : 1)} ${units[u]}`;
}
