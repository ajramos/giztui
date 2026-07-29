import {
  useCallback,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";
import { backend } from "./api";
import type { MessageSummary, MessageDetail } from "./apiTypes";
import { labelForAction } from "./format";
import type { Undo } from "./useUndo";

// useMailActions owns the message-mutation cluster: single-message actions
// (archive/trash/read/unread), the bulk equivalents, list add/remove/relabel,
// selection toggle, and the cursor-advance logic that keeps the reader in sync
// after a mutation. Extracted from App.tsx unchanged. Bulk *state* (bulkMode/
// selected/bulkProgress) stays in App and is passed in, so the many other
// readers of it (render, commands, AI bulk-prompt) are untouched.
export interface MailActions {
  removeFromList: (id: string) => void;
  insertMessage: (msg: MessageSummary, index: number) => void;
  applyLabelChange: (ids: Set<string>, change: { added?: string; removed?: string }) => void;
  doAction: (action: "archive" | "trash" | "read" | "unread", id: string) => Promise<void>;
  toggleSelect: (id: string) => void;
  exitBulk: () => void;
  clearReaderIfRemoved: (removed: Set<string>) => void;
  advanceAfterBulk: (removed: Set<string>) => void;
  bulkActionIds: (action: "archive" | "trash" | "read" | "unread", ids: string[]) => Promise<void>;
  bulkAction: (action: "archive" | "trash" | "read" | "unread") => Promise<void>;
}

export function useMailActions(deps: {
  messages: MessageSummary[];
  setMessages: Dispatch<SetStateAction<MessageSummary[]>>;
  fullMessagesRef: MutableRefObject<MessageSummary[]>;
  selectedId: string | null;
  setSelectedId: Dispatch<SetStateAction<string | null>>;
  setDetail: Dispatch<SetStateAction<MessageDetail | null>>;
  bulkMode: boolean;
  selected: Set<string>;
  setSelected: Dispatch<SetStateAction<Set<string>>>;
  setBulkMode: Dispatch<SetStateAction<boolean>>;
  setBulkProgress: Dispatch<SetStateAction<string>>;
  previewRef: MutableRefObject<(m: MessageSummary) => void>;
  pushUndo: Undo["pushUndo"];
  showToast: (m: string) => void;
  setError: (e: string) => void;
  setBusy: Dispatch<SetStateAction<boolean>>;
  setSummary: Dispatch<SetStateAction<string | null>>;
  setThreadMsgs: Dispatch<SetStateAction<MessageDetail[] | null>>;
}): MailActions {
  const {
    messages, setMessages, fullMessagesRef, selectedId, setSelectedId, setDetail,
    bulkMode, selected, setSelected, setBulkMode, setBulkProgress, previewRef,
    pushUndo, showToast, setError, setBusy, setSummary, setThreadMsgs,
  } = deps;

  const removeFromList = useCallback(
    (id: string) => {
      const idx = messages.findIndex((x) => x.id === id);
      setMessages((prev) => prev.filter((x) => x.id !== id));
      // Also drop it from the full (unfiltered) list, so toggling the local
      // filter doesn't resurrect a message the user just archived/trashed.
      fullMessagesRef.current = fullMessagesRef.current.filter((x) => x.id !== id);
      if (selectedId !== id) return;
      // When the message being read is removed (archive/trash), advance to the
      // neighbour and preview it — same index is now the next message, or the
      // previous one if we removed the last row — instead of going blank. This
      // matches the TUI, which keeps the cursor moving down the list.
      const rest = messages.filter((x) => x.id !== id);
      const nextMsg = rest[idx] ?? rest[idx - 1];
      if (nextMsg && !bulkMode) {
        setSelectedId(nextMsg.id);
        previewRef.current(nextMsg);
      } else {
        setSelectedId(null);
        setDetail(null);
      }
    },
    [messages, selectedId, bulkMode],
  );

  // insertMessage restores a summary at (roughly) its old position — used by undo
  // to bring an archived/trashed message back into the list.
  const insertMessage = useCallback((msg: MessageSummary, index: number) => {
    setMessages((prev) => {
      if (prev.some((m) => m.id === msg.id)) return prev;
      const next = [...prev];
      next.splice(Math.max(0, Math.min(index, next.length)), 0, msg);
      return next;
    });
  }, []);

  const doAction = useCallback(
    async (action: "archive" | "trash" | "read" | "unread", id: string) => {
      const msg = messages.find((m) => m.id === id);
      const index = messages.findIndex((m) => m.id === id);
      setBusy(true);
      setError("");
      try {
        if (action === "archive") {
          await backend.Archive(id);
          removeFromList(id);
          if (msg)
            pushUndo("archive", async () => {
              await backend.Unarchive(id);
              insertMessage(msg, index);
            });
          showToast("Archived");
        } else if (action === "trash") {
          await backend.Trash(id);
          removeFromList(id);
          if (msg)
            pushUndo("trash", async () => {
              await backend.Untrash(id);
              insertMessage(msg, index);
            });
          showToast("Moved to trash");
        } else if (action === "read") {
          await backend.MarkRead(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: false } : x)),
          );
          setDetail((d) => (d && d.id === id ? { ...d, unread: false } : d));
          pushUndo("mark read", async () => {
            await backend.MarkUnread(id);
            setMessages((prev) =>
              prev.map((x) => (x.id === id ? { ...x, unread: true } : x)),
            );
            setDetail((d) => (d && d.id === id ? { ...d, unread: true } : d));
          });
        } else if (action === "unread") {
          await backend.MarkUnread(id);
          setMessages((prev) =>
            prev.map((x) => (x.id === id ? { ...x, unread: true } : x)),
          );
          setDetail((d) => (d && d.id === id ? { ...d, unread: true } : d));
          pushUndo("mark unread", async () => {
            await backend.MarkRead(id);
            setMessages((prev) =>
              prev.map((x) => (x.id === id ? { ...x, unread: false } : x)),
            );
            setDetail((d) => (d && d.id === id ? { ...d, unread: false } : d));
          });
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setBusy(false);
      }
    },
    [messages, removeFromList, showToast, pushUndo, insertMessage],
  );

  // applyLabelChange updates the label chips of the affected messages (in the
  // list and the reader) in place after a labels-picker toggle — no refetch, so
  // it shows immediately and doesn't mark anything read.
  const applyLabelChange = useCallback(
    (ids: Set<string>, change: { added?: string; removed?: string }) => {
      const upd = (labs: string[]): string[] => {
        let next = labs;
        if (change.added && !next.includes(change.added))
          next = [...next, change.added];
        if (change.removed) next = next.filter((x) => x !== change.removed);
        return next;
      };
      setMessages((prev) =>
        prev.map((m) => (ids.has(m.id) ? { ...m, labels: upd(m.labels) } : m)),
      );
      setDetail((d) => (d && ids.has(d.id) ? { ...d, labels: upd(d.labels) } : d));
    },
    [],
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
      // Keep the full (unfiltered) list in sync so the local filter can't
      // resurrect bulk-removed messages.
      fullMessagesRef.current = fullMessagesRef.current.filter(
        (m) => !removed.has(m.id),
      );
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

  // advanceAfterBulk moves the cursor to the nearest surviving message (next,
  // else previous) after a bulk mutation removes `removed`, and previews it —
  // like the TUI, which keeps the cursor on the list instead of going blank.
  // Falls back to clearing the reader when nothing survives.
  const advanceAfterBulk = useCallback(
    (removed: Set<string>) => {
      // Keep the full (unfiltered) list in sync (see clearReaderIfRemoved).
      fullMessagesRef.current = fullMessagesRef.current.filter(
        (m) => !removed.has(m.id),
      );
      const curIdx = selectedId
        ? messages.findIndex((m) => m.id === selectedId)
        : -1;
      let next: MessageSummary | null = null;
      if (curIdx >= 0) {
        for (let i = curIdx; i < messages.length; i++) {
          if (!removed.has(messages[i].id)) {
            next = messages[i];
            break;
          }
        }
        if (!next) {
          for (let i = curIdx - 1; i >= 0; i--) {
            if (!removed.has(messages[i].id)) {
              next = messages[i];
              break;
            }
          }
        }
      }
      if (next) {
        previewRef.current(next);
      } else {
        setSelectedId(null);
        setDetail(null);
        setSummary(null);
        setThreadMsgs(null);
      }
    },
    [messages, selectedId],
  );

  // bulkActionIds runs a bulk operation on an explicit list of ids. It is the
  // shared core behind both the selection-driven bulkAction and the VIM range
  // operations (a3a, d2d, …), so undo/removal logic lives in exactly one place.
  // It does NOT touch the `selected` set — callers decide whether to clear it.
  const bulkActionIds = useCallback(
    async (
      action: "archive" | "trash" | "read" | "unread",
      ids: string[],
    ) => {
      if (ids.length === 0) return;
      const idSet = new Set(ids);
      // Snapshot the affected rows (with positions) so undo can restore them.
      const removed = messages
        .map((m, i) => ({ m, i }))
        .filter(({ m }) => idSet.has(m.id));
      setBusy(true);
      setBulkProgress(`${labelForAction(action)} ${ids.length}…`);
      setError("");
      try {
        if (action === "archive") {
          await backend.BulkArchive(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          advanceAfterBulk(idSet);
          pushUndo(`archive ${ids.length}`, async () => {
            await backend.BulkUnarchive(ids);
            removed.forEach(({ m, i }) => insertMessage(m, i));
          });
          showToast(`Archived ${ids.length}`);
        } else if (action === "trash") {
          await backend.BulkTrash(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          advanceAfterBulk(idSet);
          pushUndo(`trash ${ids.length}`, async () => {
            await backend.BulkUntrash(ids);
            removed.forEach(({ m, i }) => insertMessage(m, i));
          });
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
      } catch (e) {
        setError(String(e));
      } finally {
        setBulkProgress("");
        setBusy(false);
      }
    },
    [messages, showToast, advanceAfterBulk, pushUndo, insertMessage],
  );

  const bulkAction = useCallback(
    async (action: "archive" | "trash" | "read" | "unread") => {
      const ids = [...selected];
      if (ids.length === 0) return;
      await bulkActionIds(action, ids);
      setSelected(new Set());
    },
    [selected, bulkActionIds],
  );

  return {
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
  };
}
