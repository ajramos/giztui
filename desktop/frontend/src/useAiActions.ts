import { useCallback, type MutableRefObject, type RefObject } from "react";
import { backend, summarizeStream, applyPromptStream, applyBulkPromptStream, threadSummaryStream } from "./api";
import type { MessageDetail, Prompt } from "./apiTypes";
import type { ComposeInit } from "./Compose";
import { activeAiPanel } from "./aiPanels";

export interface AiCacheEntry {
  summary?: string;
  touchUp?: string;
  promptResults?: Record<number, { text: string; label: string }>;
  lastPromptId?: number;
}

// useAiActions owns the AI action functions: summarize, draft-reply, touch-up,
// run-prompt (single + bulk), the per-panel dismiss/dismiss-all, regenerate,
// and thread-summary. AI STATE + the openIdRef/aiCache/mirror refs stay in App
// (loadMessage consumes them); this hook receives them via deps. Verbatim move.
export function useAiActions(deps: {
  detail: MessageDetail | null;
  bulkMode: boolean;
  selected: Set<string>;
  aiEnabled: boolean;
  showToast: (m: string) => void;
  setError: (e: string) => void;
  setSummary: (v: string | null) => void;
  setSummarizing: (v: boolean) => void;
  setSummaryForId: (v: string | null) => void;
  setPromptResult: (v: string | null) => void;
  setPromptLabel: (v: string) => void;
  setPromptRunning: (v: boolean) => void;
  setPromptForId: (v: string | null) => void;
  setPromptsOpen: (v: boolean) => void;
  setGeneratingReply: (v: boolean) => void;
  setTouchUpText: (v: string | null) => void;
  setTouchingUp: (v: boolean) => void;
  setCompose: (v: ComposeInit | null) => void;
  setBulkPromptLabel: (v: string) => void;
  setBulkPromptText: (v: string | null) => void;
  openIdRef: MutableRefObject<string | null>;
  aiCache: MutableRefObject<Map<string, AiCacheEntry>>;
  updateAiCache: (id: string, patch: Partial<{ summary: string | undefined; touchUp: string | undefined; lastPromptId: number | undefined; promptResults: Record<number, { text: string; label: string }> }>) => void;
  summaryRef: MutableRefObject<string | null>;
  promptResultRef: MutableRefObject<string | null>;
  touchUpTextRef: MutableRefObject<string | null>;
  promptLabelRef: MutableRefObject<string>;
  runningLabelRef: MutableRefObject<Record<string, string>>;
  promptPanelRef: RefObject<HTMLDivElement>;
}) {
  const {
    detail, bulkMode, selected, aiEnabled, showToast, setError,
    setSummary, setSummarizing, setSummaryForId, setPromptResult, setPromptLabel, setPromptRunning,
    setPromptForId, setPromptsOpen, setGeneratingReply, setTouchUpText, setTouchingUp, setCompose,
    setBulkPromptLabel, setBulkPromptText, openIdRef, aiCache, updateAiCache, summaryRef,
    promptResultRef, touchUpTextRef, promptLabelRef, runningLabelRef, promptPanelRef,
  } = deps;
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

  return {
    summarize, generateReply, touchUp, runPrompt, dismissSummary,
    dismissPrompt, dismissTouchUp, dismissAI, regenerateActive, summarizeThread,
  };
}
