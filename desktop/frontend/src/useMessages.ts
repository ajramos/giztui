import {
  useCallback,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type RefObject,
  type SetStateAction,
} from "react";
import { backend, type MessageDetail, type MessageSummary } from "./api";
import { freshPrefix, dedupeNew } from "./messageList";

const PAGE_SIZE = 50;

// useMessages owns the inbox list subsystem: the loaded messages, the full
// (pre-local-filter) backing list, the search/active query, pagination, the
// local filter, and the background new-mail poll. Extracted from App.tsx
// unchanged.
//
// Landmine boundary: the reader (loadMessage / previewMessage) stays in App
// because it drives openIdRef + the AI panels; the list previews the first row
// through `previewRef` (a ref, so there's no declaration-order cycle). App's
// remaining list mutators (removeFromList/insertMessage/undo) use the returned
// setMessages + fullMessagesRef.
export interface Messages {
  messages: MessageSummary[];
  setMessages: Dispatch<SetStateAction<MessageSummary[]>>;
  fullMessagesRef: MutableRefObject<MessageSummary[]>;
  pendingNew: MessageSummary[];
  setPendingNew: Dispatch<SetStateAction<MessageSummary[]>>;
  query: string;
  setQuery: Dispatch<SetStateAction<string>>;
  activeQuery: string;
  nextToken: string;
  loadingList: boolean;
  loadingMore: boolean;
  localFilter: boolean;
  setLocalFilter: Dispatch<SetStateAction<boolean>>;
  load: (q: string) => Promise<void>;
  loadMore: () => Promise<void>;
  applyLocalFilter: (q: string) => void;
  checkNewMail: () => Promise<void>;
  showPendingNew: () => void;
}

export function useMessages(deps: {
  setError: (e: string) => void;
  setSelectedId: Dispatch<SetStateAction<string | null>>;
  setDetail: Dispatch<SetStateAction<MessageDetail | null>>;
  draftsView: boolean;
  previewRef: RefObject<(m: MessageSummary) => void>;
}): Messages {
  const { setError, setSelectedId, setDetail, draftsView, previewRef } = deps;

  const [messages, setMessages] = useState<MessageSummary[]>([]);
  const [pendingNew, setPendingNew] = useState<MessageSummary[]>([]);
  const [query, setQuery] = useState("");
  const [activeQuery, setActiveQuery] = useState("");
  const [nextToken, setNextToken] = useState("");
  const [loadingList, setLoadingList] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [localFilter, setLocalFilter] = useState(false);
  const fullMessagesRef = useRef<MessageSummary[]>([]);

  const load = useCallback(
    async (q: string) => {
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
        setPendingNew([]); // a fresh load already includes any new mail
        setNextToken(list.nextPageToken ?? "");
        // Select + preview the first message so the app opens ready to read.
        if (msgs.length > 0) previewRef.current?.(msgs[0]);
        else {
          setSelectedId(null);
          setDetail(null);
        }
      } catch (e) {
        setError(String(e));
      } finally {
        setLoadingList(false);
      }
    },
    [setError, setSelectedId, setDetail, previewRef],
  );

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
  }, [nextToken, loadingMore, activeQuery, setError]);

  // applyLocalFilter narrows the loaded list client-side (subject/from/snippet)
  // without hitting the network; an empty query restores the full list.
  const applyLocalFilter = useCallback(
    (q: string) => {
      const needle = q.trim().toLowerCase();
      const full = fullMessagesRef.current ?? [];
      const next = needle
        ? full.filter(
            (m) =>
              m.subject.toLowerCase().includes(needle) ||
              m.from.toLowerCase().includes(needle) ||
              m.snippet.toLowerCase().includes(needle),
          )
        : full;
      setMessages(next);
      if (next.length > 0) previewRef.current?.(next[0]);
      else {
        setSelectedId(null);
        setDetail(null);
      }
    },
    [setSelectedId, setDetail, previewRef],
  );

  // checkNewMail polls the inbox's first page and prepends any messages we don't
  // already have. Only runs on the plain inbox (no active search / drafts view).
  const checkNewMail = useCallback(async () => {
    if (activeQuery || draftsView || localFilter) return;
    try {
      const list = await backend.ListInbox("", PAGE_SIZE);
      const msgs = list.messages ?? [];
      const known = new Set((fullMessagesRef.current ?? []).map((m) => m.id));
      // freshPrefix = the contiguous run of unknown messages at the top (new mail
      // lands there); stopping at the first known id avoids treating messages that
      // shifted onto page 1 after a delete as new (which would scramble order).
      // Hold it in a banner instead of prepending — injecting rows while the user
      // reads/selects shifts everything under them and they act on the wrong one.
      setPendingNew(freshPrefix(msgs, known));
    } catch {
      /* transient; try again next tick */
    }
  }, [activeQuery, draftsView, localFilter]);

  // showPendingNew merges the held new mail into the list (banner click / manual
  // refresh). De-duped in case a manual refresh already pulled some in.
  const showPendingNew = useCallback(() => {
    setPendingNew((pending) => {
      if (pending.length === 0) return pending;
      const known = new Set((fullMessagesRef.current ?? []).map((m) => m.id));
      const toAdd = dedupeNew(pending, known);
      if (toAdd.length > 0) {
        fullMessagesRef.current = [...toAdd, ...(fullMessagesRef.current ?? [])];
        setMessages((prev) => [...toAdd, ...prev]);
      }
      return [];
    });
  }, []);

  return {
    messages,
    setMessages,
    fullMessagesRef,
    pendingNew,
    setPendingNew,
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
  };
}
