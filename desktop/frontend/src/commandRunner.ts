import { parseCommand } from "./commands";
import { backend } from "./api";
import { replyInit, replyAllInit, forwardInit } from "./composeBuilders";
import type { CommandCtx } from "./commandCtx";

// runCommand executes a typed command (":archive", ":search foo", …). Lifted
// verbatim from App.tsx; every value/handler it touches arrives via ctx so the
// ~490-line switch lives here instead of bloating App. Behavior-preserving.
export function runCommand(input: string, ctx: CommandCtx) {
  const {
    detail, load, doAction, activeQuery, openDrafts, saveMessage,
    openObsidian, openSlackForward, obsidianOn, slackOn, aiEnabled, aiPromptsEnabled,
    summarize, openChat, openSuggest, openInGmail, openQueries, savedQueriesOn, runActionPlan,
    runDeterministicRules, actionPlanOn, bulkMode, selected, bulkAction, showToast, doMove,
    doBulkMove, generateReply, quickSearch, themesOn, applyTheme, rulesEnabled,
    openRules, viewAnalyzerPrompt, toggleToolbar, touchUp, touchUpText, localFilter,
    applyLocalFilter, query, runUndo, toggleAutoRefresh, toggleNumbers, saveRawMessage, invite,
    respondInvite, openStats, openTelemetry, resetTelemetry, openConfig, clearCaches, loadMore, attachments,
    threadingOn, toggleThread, threadMsgs, summarizeThread, messages, previewMessage,
    accounts, applyLabelChange, bumpZoom, resetZoom, setZoom, dismissAI,
    regenerateActive, setError, setCompose, setSelected, setSelectedId, setMessages,
    setLabelsFor, setLinksFor, setMoveFor, setTouchUpText, setCsQuery, setCsIndex,
    setCsOpen, setCollapsedMsgs, setLocalFilter, setBulkMode, setViewHtml, setHeadersHidden,
    setLoadRemote, setAlwaysImagesOn, setAccountsOpen, setAdvOpen, setAttachmentsOpen, setBulkMove,
    setDetRulesOpen, setPromptsOpen, setRsvpPickerOpen, setSaveQueryOpen, setShowHelp,
    setJobsPickerOpen,
    setThemePickerOpen, alwaysImagesRef, imageOptIn, fullMessagesRef,
  } = ctx;
        const { cmd, arg } = parseCommand(input);
      // Telemetry: record the command word only (never args), skipping bare
      // numeric jumps (:5). No-op when telemetry is disabled. Fire-and-forget.
      if (cmd && !/^\d+$/.test(cmd)) {
        void backend.RecordCommand(cmd);
      }
      const d = detail;
      // Move the cursor/preview to a 1-based row (shared by :g, :$ and :N).
      const gotoRow = (n1: number) => {
        const i = Math.min(Math.max(0, n1 - 1), messages.length - 1);
        if (!messages[i]) return;
        if (bulkMode) setSelectedId(messages[i].id);
        else previewMessage(messages[i]);
      };
      switch (cmd) {
        case "search":
        case "s":
          void load(arg);
          break;
        case "inbox":
        case "i":
          void load("");
          break;
        case "unread":
        case "u":
          void load("is:unread");
          break;
        case "archived":
        case "arch-search":
        case "b":
          void load("in:archive");
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
        case "toggle-read":
        case "t":
          // TUI parity: :t / :toggle-read flips read↔unread on the open message
          // (the desktop also has explicit :read / :markunread). Read the live
          // list row so it reflects any change since the detail was fetched.
          if (d) {
            const cur = messages.find((m) => m.id === d.id);
            void doAction(cur?.unread ? "read" : "unread", d.id);
          }
          break;
        case "star":
        case "st":
          if (bulkMode && selected.size > 0) void bulkAction("star");
          else if (d) void doAction("star", d.id);
          break;
        case "unstar":
        case "unst":
          if (bulkMode && selected.size > 0) void bulkAction("unstar");
          else if (d) void doAction("unstar", d.id);
          break;
        case "labels":
        case "l":
          if (d) setLabelsFor(d.id);
          break;
        case "compose":
        case "c":
        case "new":
          setCompose({ mode: "new" });
          break;
        case "reply":
        case "r":
          if (d) setCompose(replyInit(d));
          break;
        case "replyall":
        case "reply-all":
        case "ra":
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
        case "dr":
          openDrafts();
          break;
        case "links":
        case "link":
          if (d) setLinksFor(d.id);
          break;
        case "save":
          // NOTE: intentional divergence from the TUI, where :save is an alias
          // of :save-query. Here :save saves the message to a file (more
          // intuitive); saving a search is :save-query / :sq.
          if (d) saveMessage(d.id);
          break;
        case "summarize":
        case "sum":
        case "summary":
          if (d && aiEnabled) void summarize(d.id);
          break;
        case "chat":
          if (d && aiEnabled) openChat();
          break;
        case "prompt":
        case "pr":
        case "p": {
          // ":prompt stats" (TUI parity) opens the AI prompt-usage dashboard;
          // bare ":prompt" applies a prompt.
          const sub = arg.trim().toLowerCase();
          if (sub === "stats" || sub === "s") {
            void openStats();
          } else if (aiPromptsEnabled && (d || (bulkMode && selected.size > 0))) {
            setPromptsOpen(true);
          }
          break;
        }
        case "suggest":
          if (d && aiEnabled) void openSuggest(d.id);
          break;
        case "obsidian":
        case "obs":
          // Open the ingest dialog (optional comment → "> **Note:** …" in the note).
          if (d && obsidianOn) openObsidian();
          break;
        case "slack":
        case "sl":
          // Open the forward picker (channel + pre-message); the send honors the
          // configured format_style on the backend.
          if (d && slackOn) openSlackForward();
          break;
        case "gmail":
        case "web":
        case "open-web":
        case "o":
          if (d) openInGmail(d.id);
          break;
        case "queries":
          if (savedQueriesOn) void openQueries();
          break;
        case "regenerate":
        case "regen":
          regenerateActive();
          break;
        case "dismiss":
        case "close-ai":
          dismissAI();
          break;
        case "quit":
        case "q":
        case "exit":
          void backend.Quit();
          break;
        case "accounts":
        case "acc":
          if (accounts.length > 1) setAccountsOpen(true);
          break;
        case "plan":
        case "actionplan":
        case "action-plan":
        case "ap": {
          // Mirrors the TUI's :action-plan [rules|prompt]: bare opens the plan,
          // subcommands reach the things that live inside the plan panel.
          const sub = arg.trim().toLowerCase();
          if (sub === "rules") {
            if (rulesEnabled) void openRules();
          } else if (sub === "prompt" || sub === "view-prompt") {
            if (actionPlanOn) void viewAnalyzerPrompt();
          } else if (sub === "") {
            if (actionPlanOn) void runActionPlan();
          } else {
            showToast("Usage: :action-plan [rules | prompt]");
          }
          break;
        }
        case "label":
        case "lbl":
          // Add a label by name to the current message (the picker handles
          // removal). Reflects the chip in place without a refetch.
          if (d && arg) {
            const name = arg;
            void backend
              .ApplyLabelByName(d.id, name)
              .then(() => {
                applyLabelChange(new Set([d.id]), { added: name });
                showToast(`Labeled: ${name}`);
              })
              .catch((e) => setError(String(e)));
          }
          break;
        case "select":
        case "sel": {
          // ":select all" / ":select none" (matching the * key), or by 1-based
          // number / range, e.g. ":select 3" or ":select 2-6".
          const spec = arg.trim().toLowerCase();
          if (spec === "all" || spec === "*") {
            if (messages.length > 0) {
              setBulkMode(true);
              setSelected(new Set(messages.map((x) => x.id)));
              showToast(`Selected ${messages.length}`);
            }
            break;
          }
          if (spec === "none" || spec === "clear") {
            setSelected(new Set());
            showToast("Selection cleared");
            break;
          }
          const m = spec.match(/^(\d+)(?:\s*-\s*(\d+))?$/);
          if (!m) {
            showToast("Usage: :select all | none | N | N-M");
            break;
          }
          const a = Math.max(1, Number(m[1]));
          const b = m[2] ? Number(m[2]) : a;
          const lo = Math.min(a, b);
          const hi = Math.max(a, b);
          const ids = messages.slice(lo - 1, hi).map((x) => x.id);
          if (ids.length > 0) {
            setBulkMode(true);
            setSelected((prev) => new Set([...prev, ...ids]));
            setSelectedId(ids[0]);
            showToast(`Selected ${ids.length}`);
          }
          break;
        }
        case "savequery":
        case "save-query":
        case "sq":
          if (savedQueriesOn && activeQuery) setSaveQueryOpen(true);
          break;
        case "move":
        case "mv":
          if (bulkMode && selected.size > 0) {
            if (arg) void doBulkMove(arg);
            else setBulkMove(true);
          } else if (d) {
            if (arg) void doMove(d.id, arg);
            else setMoveFor(d.id);
          }
          break;
        case "headers":
        case "toggle-headers":
          if (d) setHeadersHidden((v) => !v);
          break;
        case "markdown":
        case "md":
          if (d && d.html && d.html.trim()) setViewHtml((v) => !v);
          break;
        case "images":
        case "remote":
        case "img":
          setLoadRemote((v) => {
            const next = !v;
            if (d) {
              if (next) imageOptIn.current.add(d.id);
              else imageOptIn.current.delete(d.id);
            }
            return next;
          });
          break;
        case "images-always":
        case "always-images":
        case "imgall":
          setAlwaysImagesOn(!alwaysImagesRef.current);
          break;
        case "load":
        case "more":
        case "next":
          void loadMore();
          break;
        case "attachments":
        case "attach":
          if (d && attachments.length > 0) {
            setAttachmentsOpen(true);
          }
          break;
        case "threads":
        case "thr":
          if (d && threadingOn) void toggleThread();
          break;
        case "flatten":
        case "flat":
        case "expand-all":
        case "expand":
          if (threadMsgs) setCollapsedMsgs(new Set());
          break;
        case "collapse-all":
        case "collapse":
          if (threadMsgs)
            setCollapsedMsgs(new Set(threadMsgs.map((m) => m.id)));
          break;
        case "thread-summary":
        case "th-sum":
          if (threadMsgs && aiEnabled) void summarizeThread();
          break;
        case "toolbar":
          toggleToolbar();
          break;
        case "zoom-in":
        case "zi":
          bumpZoom(0.1);
          break;
        case "zoom-out":
        case "zo":
          bumpZoom(-0.1);
          break;
        case "zoom-reset":
          resetZoom();
          break;
        case "zoom": {
          const n = Number(arg);
          if (arg && n >= 0.6 && n <= 2.4) setZoom(n);
          else if (!arg) resetZoom();
          break;
        }
        case "autorefresh":
        case "arr":
          toggleAutoRefresh();
          break;
        case "numbers":
        case "n":
          // TUI parity: toggle the 1-based message-number column (makes :N jumps
          // like :14 easy to aim). Seeded from config.display.show_message_numbers.
          toggleNumbers();
          break;
        case "save-raw":
        case "saveraw":
          if (d) saveRawMessage(d.id);
          break;
        case "rsvp":
          // TUI parity: open the keyboard-navigable RSVP picker (same as the V key).
          if (d && invite?.isInvite) setRsvpPickerOpen(true);
          break;
        case "jobs":
        case "aijobs":
          // Open the AI background-jobs picker (browse/re-open/remove jobs).
          setJobsPickerOpen(true);
          break;
        case "accept":
          if (d && invite?.isInvite) void respondInvite(d.id, "accepted");
          break;
        case "tentative":
        case "maybe":
          if (d && invite?.isInvite) void respondInvite(d.id, "tentative");
          break;
        case "decline":
          if (d && invite?.isInvite) void respondInvite(d.id, "declined");
          break;
        case "undo":
          void runUndo();
          break;
        case "advanced":
        case "adv":
          setAdvOpen(true);
          break;
        case "stats":
        case "usage": {
          // Local usage-analytics dashboard (TUI parity). ":stats reset" wipes it;
          // ":stats <days>" sets the window. Prompt usage lives at ":prompt stats".
          const a = arg.trim().toLowerCase();
          if (a === "reset") {
            void openTelemetry().then(() => resetTelemetry());
          } else {
            const days = parseInt(a, 10);
            void openTelemetry(Number.isFinite(days) && days > 0 ? days : undefined);
          }
          break;
        }
        case "config":
        case "cfg":
          // ":config migrate" runs the config self-migration (TUI parity);
          // ":config" with no arg opens the read-only config info modal.
          if (arg.trim().toLowerCase() === "migrate") {
            void backend
              .MigrateConfig()
              .then((msg) => showToast(msg))
              .catch((e) => setError(`Config migrate failed: ${String(e)}`));
          } else {
            void openConfig();
          }
          break;
        case "llm": {
          // ChatGPT subscription login parity with the TUI's :llm command.
          const sub = arg.trim().toLowerCase().split(/\s+/)[0] ?? "";
          if (sub === "logout") {
            void backend
              .LLMLogout()
              .then(() => showToast("✓ ChatGPT tokens removed"))
              .catch((e) => setError(`ChatGPT logout failed: ${String(e)}`));
          } else if (sub === "login") {
            showToast("🔗 ChatGPT login URL copied to clipboard — paste it in your browser");
            void backend
              .LLMLogin()
              .then(() => showToast("✓ ChatGPT login complete"))
              .catch((e) => setError(`ChatGPT login failed: ${String(e)}`));
          } else {
            // ":llm" / ":llm status" → report the active engine + login state.
            void backend.ConfigInfo().then((info) => {
              if (!info.llmModel) return showToast("AI: disabled");
              const base = `AI: ${info.llmProvider} · ${info.llmModel}`;
              if (info.llmNeedsLogin) {
                showToast(base + (info.llmLoggedIn ? " · logged in" : " · not logged in (:llm login chatgpt)"));
              } else {
                showToast(base);
              }
            });
          }
          break;
        }
        case "cache":
          void clearCaches();
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
        case "prompt-new":
          // Prompts are managed inline in the picker now (edit/delete/new), so
          // these aliases open the same single surface as ":prompt".
          if (aiPromptsEnabled) setPromptsOpen(true);
          break;
        case "rules":
        case "ru":
          // ":rules run|plan" runs ONLY the deterministic rules over the inbox
          // (the TUI's :rules plan) and opens the plan panel to review/apply
          // them — no LLM. Bare ":rules" opens the manager.
          if (["run", "apply", "plan"].includes(arg.toLowerCase().trim())) {
            void runDeterministicRules();
          } else {
            setDetRulesOpen(true);
          }
          break;
        case "rp": // TUI parity: shorthand for :rules plan
          void runDeterministicRules();
          break;
        case "help":
        case "h":
          setShowHelp(true);
          break;
        case "g":
        case "goto": {
          // :g <n> jumps to row n (1-based); :g with no arg goes to the top.
          const n = arg ? Number(arg) : 1;
          if (Number.isFinite(n)) gotoRow(n);
          break;
        }
        case "bottom":
        case "end":
        case "$":
          gotoRow(messages.length);
          break;
        default:
          if (/^\d+$/.test(cmd)) {
            // Bare numeric command (:5) jumps to that row, like the TUI.
            gotoRow(Number(cmd));
          } else {
            showToast(`Unknown command: ${cmd}`);
          }
      }

}
