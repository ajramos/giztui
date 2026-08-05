import { useCallback, type Dispatch, type SetStateAction } from "react";
import { backend } from "./api";
import type { MessageSummary, MessageDetail, ConfigInfo, SavedQuery, UsageStats, TelemetrySummary } from "./apiTypes";
import { emailAddr, cleanSubject } from "./format";

// useMiscActions groups the smaller action handlers: usage-stats / config /
// clear-caches modals, move-to-folder (single + bulk), search-from-sender,
// open-in-Gmail, save message/.eml, label suggestions, and saved-query CRUD.
// Verbatim move; the state they touch stays in App and arrives via deps.
export function useMiscActions(deps: {
  messages: MessageSummary[];
  selected: Set<string>;
  activeQuery: string;
  suggestFor: string | null;
  saveQueryName: string;
  saveQueryCategory: string;
  showToast: (m: string) => void;
  setError: (e: string) => void;
  load: (q: string) => Promise<void>;
  removeFromList: (id: string) => void;
  insertMessage: (msg: MessageSummary, index: number) => void;
  advanceAfterBulk: (removed: Set<string>) => void;
  pushUndo: (label: string, run: () => Promise<void>) => void;
  setBulkMove: Dispatch<SetStateAction<boolean>>;
  setBulkProgress: Dispatch<SetStateAction<string>>;
  setBusy: Dispatch<SetStateAction<boolean>>;
  setConfigInfo: Dispatch<SetStateAction<ConfigInfo | null>>;
  setConfigOpen: Dispatch<SetStateAction<boolean>>;
  setLoadingSuggest: Dispatch<SetStateAction<boolean>>;
  setMessages: Dispatch<SetStateAction<MessageSummary[]>>;
  setMoveFor: Dispatch<SetStateAction<string | null>>;
  setQueriesOpen: Dispatch<SetStateAction<boolean>>;
  setQuery: Dispatch<SetStateAction<string>>;
  setSavedQueries: Dispatch<SetStateAction<SavedQuery[]>>;
  setSelected: Dispatch<SetStateAction<Set<string>>>;
  setStats: Dispatch<SetStateAction<UsageStats | null>>;
  setStatsOpen: Dispatch<SetStateAction<boolean>>;
  setTelemetry: Dispatch<SetStateAction<TelemetrySummary | null>>;
  setTelemetryOpen: Dispatch<SetStateAction<boolean>>;
  setSuggestFor: Dispatch<SetStateAction<string | null>>;
  setSuggestions: Dispatch<SetStateAction<string[]>>;
  setSaveQueryOpen: Dispatch<SetStateAction<boolean>>;
  setSaveQueryName: Dispatch<SetStateAction<string>>;
  setSaveQueryCategory: Dispatch<SetStateAction<string>>;
}) {
  const {
    messages, selected, activeQuery, suggestFor, saveQueryName, saveQueryCategory,
    showToast, setError, load, removeFromList, insertMessage, advanceAfterBulk,
    pushUndo, setBulkMove, setBulkProgress, setBusy, setConfigInfo, setConfigOpen,
    setLoadingSuggest, setMessages, setMoveFor, setQueriesOpen, setQuery, setSavedQueries,
    setSelected, setStats, setStatsOpen, setTelemetry, setTelemetryOpen, setSuggestFor, setSuggestions,
    setSaveQueryOpen, setSaveQueryName, setSaveQueryCategory,
  } = deps;
  const openStats = useCallback(async () => {
    setStatsOpen(true);
    try {
      setStats(await backend.UsageStats());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  // Local usage-analytics dashboard (:stats). Fetches the summary for the given
  // window (default 30 days) and opens the modal.
  const openTelemetry = useCallback(async (days?: number) => {
    setTelemetry(null);
    setTelemetryOpen(true);
    try {
      setTelemetry(await backend.TelemetrySummary(days ?? 30));
    } catch (e) {
      setError(String(e));
    }
  }, []);

  // :stats reset — wipe local telemetry, then refresh the open dashboard.
  const resetTelemetry = useCallback(async () => {
    try {
      await backend.TelemetryReset();
      setTelemetry(await backend.TelemetrySummary(30));
      showToast("Usage analytics reset");
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

  const updateQuery = useCallback(
    async (id: number, name: string, query: string, category: string) => {
      try {
        // id 0 = a brand-new saved search (the "New saved search" flow); a real
        // id edits the existing one. Both land back in the picker list.
        if (id > 0) await backend.UpdateSavedQuery(id, name, query, category);
        else await backend.SaveQuery(name, query, category);
        setSavedQueries(await backend.ListSavedQueries());
        showToast(id > 0 ? `Updated query "${name}"` : `Saved query "${name}"`);
      } catch (e) {
        setError(String(e));
      }
    },
    [showToast],
  );

  const doSaveQuery = useCallback(() => {
    const name = saveQueryName.trim();
    if (!name || !activeQuery) return;
    void backend
      .SaveQuery(name, activeQuery, saveQueryCategory.trim())
      .then(() => showToast(`Saved query "${name}"`))
      .catch((e) => setError(String(e)));
    setSaveQueryOpen(false);
    setSaveQueryName("");
    setSaveQueryCategory("");
  }, [saveQueryName, saveQueryCategory, activeQuery, showToast]);

  return {
    openStats, openTelemetry, resetTelemetry, openConfig, clearCaches, doMove, doBulkMove, quickSearch, openInGmail, saveMessage, saveRawMessage, openSuggest, applySuggestion, openQueries, runQuery, deleteQuery, updateQuery, doSaveQuery,
  };
}
