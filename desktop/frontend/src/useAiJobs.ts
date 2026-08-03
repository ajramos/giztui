import { useCallback, useRef, useState } from "react";
import type { Dispatch, SetStateAction } from "react";
import { applyBulkPromptStream } from "./api";

// A background AI job. Today the only kind is a bulk prompt applied across
// several messages; the shape is deliberately extensible (chat, notebooklm…).
export interface AiJob {
  id: string;
  // "bulk_prompt" streams in the background here; "prompt" is a single-message
  // reader prompt that already ran inline and is recorded as a done job so it
  // shows up in the :jobs list too (same surface, no re-run).
  kind: "bulk_prompt" | "prompt";
  label: string;
  status: "queued" | "running" | "done" | "error";
  text: string; // accumulated stream / final result
  error?: string;
  createdAt: number;
  finishedAt?: number;
  // context needed to run (and, later, to re-open/re-run) the job:
  messageIds: string[];
  promptId: number;
}

export interface EnqueueSpec {
  label: string;
  messageIds: string[];
  promptId: number;
}

// useAiJobs turns the old single-slot, blocking bulk-prompt modal into a small
// background jobs registry. Enqueuing a job returns immediately; the job streams
// in the background and the user can keep working. A job's result is shown in the
// existing dialog (driven here via the bulkPromptText/label/running values), but
// closing that dialog no longer cancels or loses the run — the job keeps going
// and finishes with a toast.
//
// Correctness note: the single-message prompt (applyPromptStream) and the bulk
// prompt (applyBulkPromptStream) both stream over the SAME Wails event
// ("prompt:token"). To stop two streams from interleaving their tokens now that
// bulk is non-blocking, every prompt-token stream — jobs AND the reader's
// single-message prompt — is funneled through runExclusive, which serializes
// them in call order. useAiActions wraps its single-prompt stream in the same
// gate. (Real concurrency would need per-job event names on the backend; that is
// the PR2/phase-2 upgrade.)
export function useAiJobs(deps: {
  showToast: (m: string) => void;
  setError: (e: string) => void;
  notifyOnComplete?: boolean;
}) {
  const { showToast, setError, notifyOnComplete = true } = deps;

  const [jobs, setJobs] = useState<AiJob[]>([]);
  // Which job the result dialog is showing; null = dialog closed.
  const [viewJobId, setViewJobId] = useState<string | null>(null);
  // The :jobs picker (browse/re-open/remove jobs).
  const [jobsPickerOpen, setJobsPickerOpen] = useState(false);
  const seqRef = useRef(0);

  // Serialize any prompt-token stream (jobs + the reader's single prompt). Each
  // caller awaits the previous stream's completion before starting, so the shared
  // "prompt:token" event is only ever driven by one stream at a time.
  const gateRef = useRef<Promise<void>>(Promise.resolve());
  const runExclusive = useCallback(
    async <T>(fn: () => Promise<T>): Promise<T> => {
      const prev = gateRef.current;
      let release!: () => void;
      gateRef.current = new Promise<void>((res) => (release = res));
      await prev;
      try {
        return await fn();
      } finally {
        release();
      }
    },
    [],
  );

  const patchJob = useCallback((id: string, patch: Partial<AiJob>) => {
    setJobs((prev) => prev.map((j) => (j.id === id ? { ...j, ...patch } : j)));
  }, []);

  // recordInlineJob logs a single-message reader prompt into the same jobs list
  // as a job that's already done — the prompt streamed inline in the reader, so
  // there's nothing to run here; this just gives it a row in :jobs (parity with
  // the user's expectation that every prompt run is visible there). Errors are
  // recorded too so a failed inline prompt is still traceable.
  const recordInlineJob = useCallback(
    (spec: EnqueueSpec & { text: string; error?: string }) => {
      const now = Date.now();
      const job: AiJob = {
        id: `job-${++seqRef.current}-${now}`,
        kind: "prompt",
        label: spec.label,
        status: spec.error ? "error" : "done",
        text: spec.text,
        error: spec.error,
        createdAt: now,
        finishedAt: now,
        messageIds: spec.messageIds,
        promptId: spec.promptId,
      };
      setJobs((prev) => [...prev, job]);
    },
    [],
  );

  const enqueueJob = useCallback(
    (spec: EnqueueSpec) => {
      const id = `job-${++seqRef.current}-${Date.now()}`;
      const job: AiJob = {
        id,
        kind: "bulk_prompt",
        label: spec.label,
        status: "queued",
        text: "",
        createdAt: Date.now(),
        messageIds: spec.messageIds,
        promptId: spec.promptId,
      };
      setJobs((prev) => [...prev, job]);
      setViewJobId(id); // show this job in the dialog, matching the old behavior
      void runExclusive(async () => {
        patchJob(id, { status: "running" });
        let acc = "";
        try {
          const final = await applyBulkPromptStream(
            job.messageIds,
            job.promptId,
            (tok) => {
              acc += tok;
              patchJob(id, { text: acc });
            },
          );
          patchJob(id, { status: "done", text: final, finishedAt: Date.now() });
          if (notifyOnComplete) showToast(`✓ ${job.label}`);
        } catch (e) {
          patchJob(id, { status: "error", error: String(e), finishedAt: Date.now() });
          setError(String(e));
          if (notifyOnComplete) showToast(`✗ ${job.label}`);
        }
      });
    },
    [runExclusive, patchJob, notifyOnComplete, showToast, setError],
  );

  // Open a job's result in the dialog (used by the :jobs picker in PR2).
  const openJob = useCallback((id: string) => setViewJobId(id), []);
  const removeJob = useCallback(
    (id: string) => {
      setJobs((prev) => prev.filter((j) => j.id !== id));
      setViewJobId((cur) => (cur === id ? null : cur));
    },
    [],
  );
  const clearFinished = useCallback(() => {
    setJobs((prev) => prev.filter((j) => j.status === "queued" || j.status === "running"));
  }, []);

  // --- Values the existing result dialog consumes, derived from the viewed job.
  const viewedJob = jobs.find((j) => j.id === viewJobId) ?? null;
  const bulkPromptText = viewedJob ? viewedJob.text : null; // null closes the dialog
  const bulkPromptLabel = viewedJob?.label ?? "";
  const bulkJobRunning =
    !!viewedJob && (viewedJob.status === "running" || viewedJob.status === "queued");
  const anyJobRunning = jobs.some((j) => j.status === "running" || j.status === "queued");

  // A drop-in for the old setBulkPromptText: callers only ever set it to null to
  // close the dialog (Escape / overlay click / Done). Keep that contract; other
  // values are ignored since the text is now owned by the job.
  const bulkPromptTextRef = useRef(bulkPromptText);
  bulkPromptTextRef.current = bulkPromptText;
  const setBulkPromptText: Dispatch<SetStateAction<string | null>> = useCallback((v) => {
    const next =
      typeof v === "function"
        ? (v as (p: string | null) => string | null)(bulkPromptTextRef.current)
        : v;
    if (next === null) setViewJobId(null);
  }, []);

  return {
    jobs,
    viewJobId,
    jobsPickerOpen,
    setJobsPickerOpen,
    // result-dialog bindings (same names the modal/keydown chain already use)
    bulkPromptText,
    setBulkPromptText,
    bulkPromptLabel,
    bulkJobRunning,
    anyJobRunning,
    // actions
    enqueueJob,
    recordInlineJob,
    runExclusive,
    openJob,
    removeJob,
    clearFinished,
  };
}
