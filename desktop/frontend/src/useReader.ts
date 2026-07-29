import { useCallback, type Dispatch, type MutableRefObject, type RefObject, type SetStateAction } from "react";
import { backend } from "./api";
import type { MessageSummary, MessageDetail, Attachment, Invite } from "./apiTypes";
import type { AiCacheEntry } from "./useAiActions";

// useReader owns opening a message into the reading pane: loadMessage (the
// openIdRef-guarded fetch that restores cached AI panels, resets per-message
// subsystems, and marks read), plus previewMessage (j/k, no mark-read) and
// openMessage (Enter/click). Reader + AI STATE stays in App and arrives via
// deps; this is a verbatim move of the landmine-heavy loader.
export function useReader(deps: {
  setMessages: Dispatch<SetStateAction<MessageSummary[]>>;
  setSelectedId: (v: string | null) => void;
  setLoadingDetail: (v: boolean) => void;
  setDetail: (v: MessageDetail | null) => void;
  setError: (e: string) => void;
  setSummary: (v: string | null) => void;
  setPromptResult: (v: string | null) => void;
  setPromptLabel: (v: string) => void;
  setTouchUpText: (v: string | null) => void;
  setViewHtml: (v: boolean) => void;
  setAttachments: (v: Attachment[]) => void;
  setAttachmentsOpen: (v: boolean) => void;
  setThreadMsgs: (v: MessageDetail[] | null) => void;
  setCollapsedMsgs: (v: Set<string>) => void;
  setInvite: (v: Invite | null) => void;
  setCsOpen: (v: boolean) => void;
  setLoadRemote: (v: boolean) => void;
  setReaderFocused: (v: boolean) => void;
  openIdRef: MutableRefObject<string | null>;
  aiCache: MutableRefObject<Map<string, AiCacheEntry>>;
  alwaysImagesRef: MutableRefObject<boolean>;
  imageOptIn: MutableRefObject<Set<string>>;
  rootRef: RefObject<HTMLDivElement>;
  rsvpEnabledRef: MutableRefObject<boolean>;
  runningLabelRef: MutableRefObject<Record<string, string>>;
  promptLabelRef: MutableRefObject<string>;
  promptForIdRef: MutableRefObject<string | null>;
  promptRunningRef: MutableRefObject<boolean>;
  summarizingRef: MutableRefObject<boolean>;
  summaryForIdRef: MutableRefObject<string | null>;
}) {
  const {
    setMessages, setSelectedId, setLoadingDetail, setDetail, setError, setSummary,
    setPromptResult, setPromptLabel, setTouchUpText, setViewHtml, setAttachments, setAttachmentsOpen,
    setThreadMsgs, setCollapsedMsgs, setInvite, setCsOpen, setLoadRemote, setReaderFocused,
    openIdRef, aiCache, alwaysImagesRef, imageOptIn, rootRef, rsvpEnabledRef,
    runningLabelRef, promptLabelRef, promptForIdRef, promptRunningRef, summarizingRef, summaryForIdRef,
  } = deps;
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

  return { loadMessage, previewMessage, openMessage };
}
