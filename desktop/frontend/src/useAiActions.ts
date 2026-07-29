import { useCallback, useEffect, useRef, useState } from "react";
import { backend, summarizeStream, applyPromptStream, threadSummaryStream } from "./api";
import type { MessageDetail, Prompt } from "./apiTypes";
import type { ComposeInit } from "./Compose";
import type { EnqueueSpec } from "./useAiJobs";
import { activeAiPanel } from "./aiPanels";

export interface AiCacheEntry {
  summary?: string;
  touchUp?: string;
  promptResults?: Record<number, { text: string; label: string }>;
  lastPromptId?: number;
}

// useAi owns the entire AI subsystem: the panel state (summary / prompt /
// touch-up), the coupling landmines (openIdRef, the per-message aiCache, and the
// mirror refs the keydown handler/commands read without stale closures), the
// reveal/toast effects, and the action functions (summarize, draft-reply,
// touch-up, run-prompt single+bulk, per-panel dismiss, regenerate, thread
// summary). App wires only the external inputs below and consumes the returned
// state/setters/refs by their original names, so loadMessage and the render keep
// working unchanged. The action bodies are a verbatim move.
export function useAiActions(deps: {
  detail: MessageDetail | null;
  bulkMode: boolean;
  selected: Set<string>;
  aiEnabled: boolean;
  showToast: (m: string) => void;
  setError: (e: string) => void;
  setPromptsOpen: (v: boolean) => void;
  setCompose: (v: ComposeInit | null) => void;
  // Bulk prompts now run as background jobs; the reader's single-message prompt
  // stream is serialized through runExclusive (shared "prompt:token" event).
  enqueueJob: (spec: EnqueueSpec) => void;
  runExclusive: <T>(fn: () => Promise<T>) => Promise<T>;
}) {
  const {
    detail, bulkMode, selected, aiEnabled, showToast, setError,
    setPromptsOpen, setCompose, enqueueJob, runExclusive,
  } = deps;

  // Refs to the AI result panels so we can scroll them into view when they
  // appear — otherwise, if the reader is scrolled down, the panel renders above
  // the fold and it looks like nothing happened.
  const summaryPanelRef = useRef<HTMLDivElement>(null);
  const promptPanelRef = useRef<HTMLDivElement>(null);
  const touchUpRef = useRef<HTMLDivElement>(null);
  const [summary, setSummary] = useState<string | null>(null);
  const [summarizing, setSummarizing] = useState(false);
  // The message a summary run/result belongs to, so a summary started on one
  // email doesn't paint its "Generating…" / stream over another you navigated to.
  const [summaryForId, setSummaryForId] = useState<string | null>(null);
  const [promptResult, setPromptResult] = useState<string | null>(null);
  const [promptLabel, setPromptLabel] = useState("");
  const [promptRunning, setPromptRunning] = useState(false);
  // The message a single-message prompt result/run belongs to, so a run started
  // on one email doesn't paint its "Generating…" over a different email you've
  // since navigated to.
  const [promptForId, setPromptForId] = useState<string | null>(null);
  // Always the id of the message currently open in the reader, so a streaming
  // prompt can tell it should stop updating the visible panel once you move away.
  const openIdRef = useRef<string | null>(null);
  const [generatingReply, setGeneratingReply] = useState(false);
  const [touchUpText, setTouchUpText] = useState<string | null>(null);
  const [touchingUp, setTouchingUp] = useState(false);
  // Remember AI results per message (session) so navigating away and back shows
  // the summary / prompt output / reformat again instead of a blank panel. The
  // backend also caches, but the frontend state was cleared on every open.
  const aiCache = useRef<Map<string, AiCacheEntry>>(new Map());
  // updateAiCache merges a patch into a message's cache entry (creating it if
  // needed); a key set to undefined deletes it. Consolidates the repeated
  // get-or-{}/mutate/set dance around aiCache.current.
  const updateAiCache = useCallback(
    (
      id: string,
      patch: Partial<{
        summary: string | undefined;
        touchUp: string | undefined;
        lastPromptId: number | undefined;
        promptResults: Record<number, { text: string; label: string }>;
      }>,
    ) => {
      const e = aiCache.current.get(id) ?? {};
      for (const [k, v] of Object.entries(patch)) {
        if (v === undefined) delete (e as Record<string, unknown>)[k];
        else (e as Record<string, unknown>)[k] = v;
      }
      aiCache.current.set(id, e);
    },
    [],
  );
  // Refs mirroring the AI-panel state so the keydown handler / commands can read
  // fresh values without stale closures (for :dismiss, :regenerate, layered Esc).
  const summaryRef = useRef(summary);
  summaryRef.current = summary;
  const promptResultRef = useRef(promptResult);
  promptResultRef.current = promptResult;
  const promptLabelRef = useRef(promptLabel);
  promptLabelRef.current = promptLabel;
  const promptRunningRef = useRef(promptRunning);
  promptRunningRef.current = promptRunning;
  const promptForIdRef = useRef(promptForId);
  promptForIdRef.current = promptForId;
  const summarizingRef = useRef(summarizing);
  summarizingRef.current = summarizing;
  const summaryForIdRef = useRef(summaryForId);
  summaryForIdRef.current = summaryForId;
  // Label of the prompt currently streaming, keyed by message id, so returning
  // to a message mid-run can restore its panel title (the global promptLabel is
  // reset when you navigate away).
  const runningLabelRef = useRef<Record<string, string>>({});
  const touchUpTextRef = useRef(touchUpText);
  touchUpTextRef.current = touchUpText;

  // When an AI result panel starts, reveal it (the panels render at the top of
  // the reader, so if you'd scrolled down they'd appear above the fold and look
  // like a no-op) and flash a toast so there's immediate feedback either way.
  useEffect(() => {
    if (summarizing && summaryForId && summaryForId === detail?.id) {
      summaryPanelRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
      showToast("Summarizing…");
    }
  }, [summarizing, summaryForId, detail?.id, showToast]);
  useEffect(() => {
    // Only for a single-message prompt on the message that's actually open (a
    // bulk run streams into its own modal; promptForId is null for it).
    if (promptRunning && promptForId && promptForId === detail?.id) {
      promptPanelRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
      showToast(promptLabel ? `Applying ${promptLabel}…` : "Applying prompt…");
    }
  }, [promptRunning, promptForId, detail?.id, promptLabel, showToast]);
  useEffect(() => {
    if (touchingUp) showToast("Reformatting…");
  }, [touchingUp, showToast]);
  useEffect(() => {
    // The reformatted panel only mounts once the result is set, so reveal it then.
    if (touchUpText !== null)
      touchUpRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
  }, [touchUpText]);

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

      // --- bulk (multi-message) prompt: runs as a background AI job -----------
      // Non-blocking: enqueue and return. The job streams in the background (see
      // useAiJobs), shows in the result dialog, and toasts on completion; closing
      // the dialog no longer cancels or loses it.
      if (bulk) {
        enqueueJob({
          label: `${prompt.name} · ${selected.size} messages`,
          messageIds: [...selected],
          promptId: prompt.id,
        });
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
        // Serialized against any running bulk job (shared "prompt:token" event):
        // if a job is streaming, this waits its turn instead of interleaving.
        const final = await runExclusive(() =>
          applyPromptStream(
            launchId,
            prompt.id,
            (tok) => {
              acc += tok;
              // Only paint into the visible panel while this message is still open.
              if (openIdRef.current === launchId) setPromptResult(acc);
            },
            force,
          ),
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
    [detail, bulkMode, selected, showToast, updateAiCache, enqueueJob, runExclusive],
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
    // panel state
    summary, setSummary, summarizing, summaryForId,
    promptResult, setPromptResult, promptLabel, setPromptLabel, promptRunning, setPromptRunning, promptForId,
    generatingReply, touchUpText, setTouchUpText, touchingUp,
    // DOM refs
    summaryPanelRef, promptPanelRef, touchUpRef,
    // landmine refs (shared with loadMessage / keydown / commands)
    openIdRef, aiCache, runningLabelRef, promptLabelRef, promptForIdRef,
    promptRunningRef, summarizingRef, summaryForIdRef,
    // actions
    summarize, generateReply, touchUp, runPrompt, dismissSummary,
    dismissPrompt, dismissTouchUp, dismissAI, regenerateActive, summarizeThread,
  };
}
