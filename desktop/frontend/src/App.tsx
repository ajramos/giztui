import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
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
  type MessageDetail,
  type MessageSummary,
  type Prompt,
  type SavedQuery,
  type AnalyzerInput,
  type ActionPlanResult,
  type PlanCategory,
} from "./api";
import Compose, { type ComposeInit } from "./Compose";
import LabelsPicker from "./LabelsPicker";
import PromptsPicker from "./PromptsPicker";
import LinksPicker from "./LinksPicker";
import AccountSwitcher from "./AccountSwitcher";
import HtmlBody from "./HtmlBody";
import Help from "./Help";
import CommandBar, { type CommandDef } from "./CommandBar";
import { Icon, IconBtn } from "./Icons";

const PAGE_SIZE = 50;

// Command palette entries (`:` command mode), mirroring the TUI's command set.
const COMMANDS: CommandDef[] = [
  { names: ["search", "s"], desc: "Gmail search", arg: "<query>" },
  { names: ["unread"], desc: "Show unread only" },
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
  { names: ["suggest"], desc: "Suggest labels (AI)" },
  { names: ["obsidian"], desc: "Send to Obsidian" },
  { names: ["slack"], desc: "Forward to Slack" },
  { names: ["gmail", "web"], desc: "Open in Gmail" },
  { names: ["queries", "q"], desc: "Saved searches" },
  { names: ["savequery"], desc: "Save current search" },
  { names: ["plan", "actionplan"], desc: "AI inbox action plan" },
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
  const [bulkLabels, setBulkLabels] = useState(false);
  const [aiPromptsEnabled, setAiPromptsEnabled] = useState(false);
  const [promptsOpen, setPromptsOpen] = useState(false);
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
  const [loadingThread, setLoadingThread] = useState(false);
  const [savedQueriesOn, setSavedQueriesOn] = useState(false);
  const [queriesOpen, setQueriesOpen] = useState(false);
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [saveQueryOpen, setSaveQueryOpen] = useState(false);
  const [saveQueryName, setSaveQueryName] = useState("");
  const [actionPlanOn, setActionPlanOn] = useState(false);
  const [planOpen, setPlanOpen] = useState(false);
  const [plan, setPlan] = useState<ActionPlanResult | null>(null);
  const [analyzing, setAnalyzing] = useState(false);

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
      setMessages((prev) => [...prev, ...(list.messages ?? [])]);
      setNextToken(list.nextPageToken ?? "");
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingMore(false);
    }
  }, [nextToken, loadingMore, activeQuery]);

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
      } catch {
        /* non-fatal */
      }
      void load("");
    })();
  }, [load]);

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

  const bulkAction = useCallback(
    async (action: "archive" | "trash" | "read" | "unread") => {
      const ids = [...selected];
      if (ids.length === 0) return;
      setBusy(true);
      setError("");
      try {
        if (action === "archive") {
          await backend.BulkArchive(ids);
          setMessages((prev) => prev.filter((m) => !selected.has(m.id)));
          showToast(`Archived ${ids.length}`);
        } else if (action === "trash") {
          await backend.BulkTrash(ids);
          setMessages((prev) => prev.filter((m) => !selected.has(m.id)));
          showToast(`Moved ${ids.length} to trash`);
        } else if (action === "read") {
          await backend.BulkMarkRead(ids);
          setMessages((prev) =>
            prev.map((m) => (selected.has(m.id) ? { ...m, unread: false } : m)),
          );
          showToast(`Marked ${ids.length} read`);
        } else if (action === "unread") {
          await backend.BulkMarkUnread(ids);
          setMessages((prev) =>
            prev.map((m) => (selected.has(m.id) ? { ...m, unread: true } : m)),
          );
          showToast(`Marked ${ids.length} unread`);
        }
        setSelected(new Set());
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [selected, showToast],
  );

  const summarize = useCallback(async (id: string) => {
    setSummarizing(true);
    setSummary("");
    setError("");
    try {
      let acc = "";
      const final = await summarizeStream(id, (tok) => {
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
  }, []);

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
      if (!detail) return;
      setPromptsOpen(false);
      setPromptLabel(prompt.name);
      setPromptRunning(true);
      setPromptResult("");
      setError("");
      try {
        let acc = "";
        const final = await applyPromptStream(detail.id, prompt.id, (tok) => {
          acc += tok;
          setPromptResult(acc);
        });
        setPromptResult(final);
      } catch (e) {
        setError(String(e));
        setPromptResult(null);
      } finally {
        setPromptRunning(false);
      }
    },
    [detail],
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
        } else if (cat.action === "trash") {
          await backend.BulkTrash(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
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
    [showToast],
  );

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
          if (d && aiPromptsEnabled) setPromptsOpen(true);
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
      showToast,
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
    add(keymap.threading, "threading");
    add(keymap.savedQueries, "savedQueries");
    add(keymap.saveQuery, "saveQuery");
    add(keymap.actionPlan, "actionPlan");
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
        linksFor ||
        suggestFor ||
        cmdOpen ||
        queriesOpen ||
        saveQueryOpen ||
        planOpen
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
      if (chord === "/") {
        e.preventDefault();
        searchRef.current?.focus();
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
          if (detail && aiPromptsEnabled && !bulkMode) setPromptsOpen(true);
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
    actionPlanOn,
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
            void load(q);
          }}
        >
          <input
            ref={searchRef}
            type="text"
            placeholder="Search mail (press / or s) — Gmail operators supported: from:, has:attachment…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button type="submit">Search</button>
          {activeQuery && (
            <button
              type="button"
              className="ghost"
              onClick={() => {
                setQuery("");
                void load("");
              }}
            >
              Clear
            </button>
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
          <button onClick={() => setCompose({ mode: "new" })} title="Compose (c)">
            Compose
          </button>
          <button
            className={draftsView ? "" : "ghost"}
            onClick={() => {
              if (draftsView) setDraftsView(false);
              else openDrafts();
            }}
            title="Drafts (D)"
          >
            Drafts
          </button>
          <button
            className={bulkMode ? "" : "ghost"}
            onClick={() => {
              if (bulkMode) exitBulk();
              else {
                setBulkMode(true);
                if (!selectedId && messages.length > 0)
                  setSelectedId(messages[0].id);
              }
            }}
            title="Select mode (v)"
          >
            Select
          </button>
          {savedQueriesOn && (
            <button
              className="ghost"
              onClick={() => void openQueries()}
              title="Saved searches (Q)"
            >
              ★
            </button>
          )}
          <button className="ghost" onClick={() => setShowHelp(true)} title="Shortcuts (?)">
            ?
          </button>
          <button
            className="ghost"
            onClick={() => void load(activeQuery)}
            title="Refresh (R)"
          >
            ⟳
          </button>
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
              <span className="bulk-count">{selected.size} selected</span>
              <div className="bulk-actions">
                <button
                  className="tiny"
                  disabled={busy || selected.size === 0}
                  onClick={() => void bulkAction("archive")}
                >
                  Archive
                </button>
                <button
                  className="tiny danger"
                  disabled={busy || selected.size === 0}
                  onClick={() => void bulkAction("trash")}
                >
                  Trash
                </button>
                <button
                  className="tiny ghost"
                  disabled={busy || selected.size === 0}
                  onClick={() => void bulkAction("read")}
                >
                  Read
                </button>
                <button
                  className="tiny ghost"
                  disabled={busy || selected.size === 0}
                  onClick={() => void bulkAction("unread")}
                >
                  Unread
                </button>
                <button
                  className="tiny ghost"
                  disabled={busy || selected.size === 0}
                  onClick={() => setBulkLabels(true)}
                >
                  Label…
                </button>
                <button
                  className="tiny ghost"
                  onClick={() => setSelected(new Set(messages.map((m) => m.id)))}
                >
                  All
                </button>
                <button className="tiny ghost" onClick={exitBulk}>
                  Done
                </button>
              </div>
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
                <div className="actions">
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
                  {aiEnabled && (
                    <IconBtn
                      icon={Icon.summarize}
                      label={summarizing ? "Summarizing…" : "Summarize (AI)"}
                      disabled={summarizing}
                      onClick={() => void summarize(detail.id)}
                    />
                  )}
                  {aiPromptsEnabled && (
                    <IconBtn
                      icon={Icon.prompt}
                      label={promptRunning ? "Running…" : "Apply a prompt"}
                      disabled={promptRunning}
                      onClick={() => setPromptsOpen(true)}
                    />
                  )}
                  {aiEnabled && (
                    <IconBtn
                      icon={Icon.tag2}
                      label="Suggest labels (AI)"
                      onClick={() => void openSuggest(detail.id)}
                    />
                  )}
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
                  <IconBtn
                    icon={Icon.link}
                    label="Links"
                    onClick={() => setLinksFor(detail.id)}
                  />
                  {obsidianOn && (
                    <IconBtn
                      icon={Icon.obsidian}
                      label="Send to Obsidian"
                      onClick={() => sendObsidian(detail.id)}
                    />
                  )}
                  {slackOn && (
                    <IconBtn
                      icon={Icon.slack}
                      label="Forward to Slack"
                      onClick={() => forwardSlack(detail.id)}
                    />
                  )}
                  <IconBtn
                    icon={Icon.save}
                    label="Save to file"
                    onClick={() => saveMessage(detail.id)}
                  />
                  <span className="actions-sep" />
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
                  <IconBtn
                    icon={Icon.external}
                    label="Open in Gmail"
                    onClick={() => openInGmail(detail.id)}
                  />
                </div>
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
                      {summary && (
                        <button
                          className="ghost tiny"
                          onClick={() => setSummary(null)}
                        >
                          dismiss
                        </button>
                      )}
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
                {loadingThread ? (
                  <div className="placeholder">Loading conversation…</div>
                ) : threadMsgs ? (
                  <div className="conversation">
                    <div className="conv-head">
                      <span>Conversation · {threadMsgs.length} messages</span>
                      {aiEnabled && (
                        <button
                          className="tiny"
                          disabled={summarizing}
                          onClick={() => void summarizeThread()}
                        >
                          {summarizing ? "Summarizing…" : "✦ Summarize conversation"}
                        </button>
                      )}
                    </div>
                    {threadMsgs.map((m) => (
                      <div
                        key={m.id}
                        className={"conv-msg" + (m.unread ? " unread" : "")}
                      >
                        <div className="conv-msg-head">
                          <strong>{displayName(m.from)}</strong>
                          <span className="muted">{formatFull(m.date)}</span>
                        </div>
                        <pre className="plain">{m.plainText || "(empty)"}</pre>
                      </div>
                    ))}
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
                  <pre className="plain">{detail.plainText || "(empty body)"}</pre>
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
      {planOpen && (
        <div className="modal-overlay" onClick={() => setPlanOpen(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-head">
              <h3>✦ Inbox action plan</h3>
              <button className="ghost" onClick={() => setPlanOpen(false)}>
                ✕
              </button>
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
              <button onClick={() => setPlanOpen(false)}>Done</button>
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
