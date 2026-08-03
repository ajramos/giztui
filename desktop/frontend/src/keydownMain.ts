import { replyInit, replyAllInit, forwardInit } from "./composeBuilders";
import { backend } from "./api";
import type { KeydownCtx } from "./keydownCtx";

// The keyboard handler main dispatch: reader-focused scrolling, inbox
// navigation (j/k/arrows/Enter), and the chord -> action lookup incl. the VIM
// range operations. Reached from handleKeyDown once the guards have passed.
export function handleKeyMain(e: KeyboardEvent, ctx: KeydownCtx) {
  const {
    actionPlanOn, aiEnabled, aiPromptsEnabled, attachments, bulkAction, bulkMode,
    chordAction, clearVimRange, detail, dismissAI, doAction,
    exitBulk, fullMessagesRef, headersHidden, invite, keymap, load,
    loadMore, localFilter, messages, obsidianOn, openDrafts, openInGmail,
    openMessage, openQueries, openSuggest, previewMessage, quickSearch, readerBodyRef,
    readerFocused, regenerateActive, runActionPlan, runUndo, runVimRange, runVimSingle,
    saveMessage, saveRawMessage, savedQueriesOn, searchRef, selected, selectedId,
    openObsidian, setAttachmentsOpen, setBulkLabels, setBulkMode, setBulkMove, setBulkProgress,
    setCmdOpen, setCompose, setCsIndex, setCsOpen, setCsQuery, setDetail,
    setHeadersHidden, setLabelsFor, setLinksFor, setLocalFilter, setMessages, setMoveFor,
    setPromptsOpen, setQuery, setReaderFocused, setRsvpPickerOpen, setJobsPickerOpen, openChat, setSaveQueryOpen, setSelected,
    setSelectedId, setShowHelp, setThemePickerOpen, setViewHtml, showToast, slackOn,
    summarize, threadingOn, toggleSelect, toggleStar, toggleThread,
    activeQuery, gPressedAt, vimRange, openSlackForward, themesOn,
  } = ctx;
      const chord = e.key === " " ? "space" : e.key;
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

      // Tab / Shift+Tab toggle focus between the message list and the reader —
      // the GUI equivalent of the TUI's Tab focus ring (list ⇄ reader). Only two
      // panes, so both directions just flip readerFocused, which routes the
      // arrow/j-k scrolling below and lights the reader's "reader-focused" border.
      // preventDefault always, so WKWebView can't move DOM focus onto a toolbar
      // button and start swallowing subsequent keys.
      if (e.key === "Tab") {
        e.preventDefault();
        if (detail && !bulkMode) setReaderFocused((v) => !v);
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
        e.preventDefault();
        if (bulkMode) {
          // Bulk mode: select all (TUI parity).
          setSelected(new Set(messages.map((m) => m.id)));
        } else {
          // Otherwise toggle the star on the highlighted / open message.
          const id = selectedId ?? detail?.id;
          if (id) void toggleStar(id);
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
      // Telemetry: record the shortcut key (no-op when disabled). Skip the
      // command-bar trigger — the command typed after it is captured on its own,
      // so counting the key too would double-count. Fire-and-forget.
      if (action !== "commandMode") {
        void backend.RecordShortcut(chord);
      }
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
          if (detail && obsidianOn) openObsidian();
          break;
        case "slack":
          if (detail && slackOn) openSlackForward();
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
        case "aijobs":
          // Open the AI background-jobs picker (":jobs" / default "J").
          setJobsPickerOpen(true);
          break;
        case "chat":
          // Open the chat-with-this-email panel (":chat" / default "X").
          openChat();
          break;
      }

}
