import { useCallback, useMemo, useRef } from "react";
import type { KeyMap, MessageSummary } from "./apiTypes";
import type { VimOp } from "./keydownCtx";

// useKeymap owns the keyboard action layer: the chord->action lookup built
// from the config keymap, and the VIM range-operation state machine (a3a/d2d/
// t5t/l2l) with its runVimSingle/runVimRange/clearVimRange helpers + vimRange
// ref. Consumed by the keydown handler via ctx. Verbatim move.
export function useKeymap(deps: {
  keymap: KeyMap;
  obsidianOn: boolean;
  messages: MessageSummary[];
  bulkMode: boolean;
  selected: Set<string>;
  doAction: (action: "archive" | "trash" | "read" | "unread", id: string) => Promise<void>;
  bulkAction: (action: "archive" | "trash" | "read" | "unread") => Promise<void>;
  bulkActionIds: (action: "archive" | "trash" | "read" | "unread", ids: string[]) => Promise<void>;
  setBulkLabels: (v: boolean) => void;
  setBulkMode: (v: boolean) => void;
  setBulkProgress: (v: string) => void;
  setLabelsFor: (v: string | null) => void;
  setSelected: (v: Set<string>) => void;
  setSelectedId: (v: string | null) => void;
}) {
  const {
    keymap, obsidianOn, messages, bulkMode, selected, doAction,
    bulkAction, bulkActionIds, setBulkLabels, setBulkMode, setBulkProgress,
    setLabelsFor, setSelected, setSelectedId,
  } = deps;
  const vimRange = useRef<{
    key: string;
    op: "archive" | "trash" | "toggleRead" | "manageLabels";
    count: number;
    startId: string;
    timer: number;
  } | null>(null);


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

  return { chordAction, runVimSingle, runVimRange, clearVimRange, vimRange };
}
