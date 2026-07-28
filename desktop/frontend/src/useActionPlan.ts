import { useCallback, useEffect, useMemo, useRef } from "react";
import type { Dispatch, SetStateAction } from "react";
import { backend, applyBulkPromptStream } from "./api";
import type {
  ActionPlanResult,
  PlanCategory,
  MessageDetail,
  MessageSummary,
  AnalyzerInput,
} from "./apiTypes";
import { buildPlanNodes, type PlanNode } from "./planNodes";
import { applyPlanMove, sortPlanCategories, type MoveTarget } from "./planMove";
import { useListNav } from "./useListNav";

// useActionPlan owns the inbox action-plan / analyzer subsystem: running the
// AI analysis (or deterministic rules), the plan tree nav (planNodes/planNav),
// applying a category (archive/trash/label/move/prompt), per-email preview, the
// analyzer-rule CRUD, and the analyzer-prompt view. Plan/rules STATE stays in
// App and is passed in; this hook holds the (verbatim) logic + the nav hooks.
export function useActionPlan(deps: {
  plan: ActionPlanResult | null;
  setPlan: Dispatch<SetStateAction<ActionPlanResult | null>>;
  planOpen: boolean;
  setPlanOpen: Dispatch<SetStateAction<boolean>>;
  analyzing: boolean;
  setAnalyzing: Dispatch<SetStateAction<boolean>>;
  setAnalyzeCount: Dispatch<SetStateAction<number>>;
  setAnalyzeElapsed: Dispatch<SetStateAction<number>>;
  setAnalyzeProgress: Dispatch<SetStateAction<{ done: number; total: number } | null>>;
  planExcluded: Set<string>;
  setPlanExcluded: Dispatch<SetStateAction<Set<string>>>;
  expandedCats: Set<string>;
  setApplyingAll: Dispatch<SetStateAction<boolean>>;
  planMove: { kind: "email" | "category"; catIdx: number; id?: string } | null;
  setPlanMove: Dispatch<SetStateAction<{ kind: "email" | "category"; catIdx: number; id?: string } | null>>;
  planPreview: MessageDetail | null;
  setPlanPreview: Dispatch<SetStateAction<MessageDetail | null>>;
  setPlanPreviewLoading: Dispatch<SetStateAction<boolean>>;
  setRules: Dispatch<SetStateAction<{ id: number; text: string }[]>>;
  newRule: string;
  setNewRule: Dispatch<SetStateAction<string>>;
  rulesOpen: boolean;
  setRulesOpen: Dispatch<SetStateAction<boolean>>;
  setDetRulesOpen: Dispatch<SetStateAction<boolean>>;
  messages: MessageSummary[];
  setMessages: Dispatch<SetStateAction<MessageSummary[]>>;
  promptPreview: string | null;
  setPromptPreview: Dispatch<SetStateAction<string | null>>;
  bulkPromptText: string | null;
  setBulkPromptText: Dispatch<SetStateAction<string | null>>;
  setBulkPromptLabel: Dispatch<SetStateAction<string>>;
  setPromptRunning: Dispatch<SetStateAction<boolean>>;
  showToast: (m: string) => void;
  setError: (e: string) => void;
  clearReaderIfRemoved: (removed: Set<string>) => void;
}) {
  const {
    plan, setPlan, planOpen, setPlanOpen, analyzing, setAnalyzing,
    setAnalyzeCount, setAnalyzeElapsed, setAnalyzeProgress, planExcluded, setPlanExcluded, expandedCats,
    setApplyingAll, planMove, setPlanMove, planPreview, setPlanPreview, setPlanPreviewLoading,
    setRules, newRule, setNewRule, rulesOpen, setRulesOpen,
    setDetRulesOpen, messages, setMessages, promptPreview, setPromptPreview, bulkPromptText,
    setBulkPromptText, setBulkPromptLabel, setPromptRunning, showToast, setError, clearReaderIfRemoved,
  } = deps;
  const runActionPlan = useCallback(async () => {
    setPlanOpen(true);
    setAnalyzing(true);
    setAnalyzeCount(messages.length);
    setAnalyzeElapsed(0);
    setAnalyzeProgress(null);
    setPlan(null);
    setPlanExcluded(new Set());
    setError("");
    // Listen for real per-batch progress from the backend (no-op in the browser
    // mock, which can't emit runtime events — the timer/indeterminate bar covers
    // that case).
    const off = window.runtime?.EventsOn(
      "plan:progress",
      (...data: unknown[]) => {
        const d = data[0] as { done?: number; total?: number };
        if (typeof d?.done === "number" && typeof d?.total === "number") {
          setAnalyzeProgress({ done: d.done, total: d.total });
        }
      },
    );
    try {
      const inputs: AnalyzerInput[] = messages.map((m) => ({
        id: m.id,
        subject: m.subject,
        from: m.from,
        snippet: m.snippet,
      }));
      const res = await backend.AnalyzeInbox(inputs);
      // Order by action then name (the TUI's SortCategories), read-manually last.
      setPlan({ ...res, categories: sortPlanCategories(res.categories) });
    } catch (e) {
      setError(String(e));
    } finally {
      setAnalyzing(false);
      off?.();
    }
  }, [messages]);

  // Run ONLY the deterministic rules over the loaded inbox (the TUI's
  // ":rules plan") — no LLM. Opens the same plan panel with the rule buckets so
  // you can review and apply them (move/label/archive) without the AI pass.
  const runDeterministicRules = useCallback(async () => {
    setDetRulesOpen(false);
    setPlanOpen(true);
    setAnalyzing(true);
    setAnalyzeCount(messages.length);
    setAnalyzeElapsed(0);
    setAnalyzeProgress(null);
    setPlan(null);
    setPlanExcluded(new Set());
    setError("");
    try {
      const inputs: AnalyzerInput[] = messages.map((m) => ({
        id: m.id,
        subject: m.subject,
        from: m.from,
        snippet: m.snippet,
      }));
      const res = await backend.RunDeterministicRules(inputs);
      if (res.categories.length === 0) {
        setPlanOpen(false);
        showToast("No messages matched your deterministic rules");
        return;
      }
      setPlan({ ...res, categories: sortPlanCategories(res.categories) });
    } catch (e) {
      setError(String(e));
    } finally {
      setAnalyzing(false);
    }
  }, [messages, showToast]);

  // Tick the elapsed-seconds counter while the analysis runs so the user sees
  // steady progress (the analysis is one backend call with no sub-progress).
  useEffect(() => {
    if (!analyzing) return;
    const t = setInterval(() => setAnalyzeElapsed((s) => s + 1), 1000);
    return () => clearInterval(t);
  }, [analyzing]);

  const applyCategory = useCallback(
    // asMove: for a "label" category, move to the folder (label + archive, leaves
    // the inbox) instead of only labelling — the user's dominant workflow.
    async (cat: PlanCategory, asMove = false) => {
      // Act only on the still-checked (non-deselected) emails; deselected ones
      // stay in the category so the user can move or handle them separately.
      const ids = cat.messageIds.filter((id) => !planExcluded.has(id));
      if (!ids.length) return;
      const idSet = new Set(ids);
      try {
        let toast = `${cat.name}: ${cat.action.replace("_", " ")} · ${ids.length}`;
        if (cat.action === "archive") {
          await backend.BulkArchive(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          clearReaderIfRemoved(idSet);
        } else if (cat.action === "trash") {
          await backend.BulkTrash(ids);
          setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
          clearReaderIfRemoved(idSet);
        } else if (cat.action === "mark_read") {
          await backend.BulkMarkRead(ids);
          setMessages((prev) =>
            prev.map((m) => (idSet.has(m.id) ? { ...m, unread: false } : m)),
          );
        } else if (cat.action === "label") {
          if (asMove) {
            // Move to folder = apply label + archive → leaves the inbox list.
            await backend.BulkMoveToLabel(ids, cat.label);
            setMessages((prev) => prev.filter((m) => !idSet.has(m.id)));
            clearReaderIfRemoved(idSet);
            toast = `Moved ${ids.length} to ${cat.label}`;
          } else {
            await backend.BulkApplyLabelByName(ids, cat.label);
          }
        } else {
          return;
        }
        showToast(toast);
        // Drop the acted-on emails; keep the category (with any deselected ones)
        // only if some remain, otherwise remove it entirely.
        setPlan((prev) =>
          prev
            ? {
                ...prev,
                categories: prev.categories
                  .map((c) =>
                    c === cat
                      ? {
                          ...c,
                          messageIds: c.messageIds.filter(
                            (id) => !idSet.has(id),
                          ),
                        }
                      : c,
                  )
                  .filter((c) => c.messageIds.length > 0),
              }
            : prev,
        );
      } catch (e) {
        setError(String(e));
      }
    },
    [showToast, clearReaderIfRemoved, planExcluded],
  );

  // dispatchPromptCategory runs the prompt attached to a "prompt" bucket (from a
  // deterministic rule) over its selected emails, streaming the result into the
  // shared prompt-result modal — the TUI's dispatchActionPlanPrompt.
  const dispatchPromptCategory = useCallback(
    async (cat: PlanCategory) => {
      if (cat.action !== "prompt" || !cat.promptId) return;
      const ids = cat.messageIds.filter((id) => !planExcluded.has(id));
      if (!ids.length) return;
      setBulkPromptLabel(`${cat.name} · ${ids.length} emails`);
      setBulkPromptText("");
      setPromptRunning(true);
      setError("");
      try {
        let acc = "";
        const final = await applyBulkPromptStream(ids, cat.promptId, (tok) => {
          acc += tok;
          setBulkPromptText(acc);
        });
        setBulkPromptText(final);
      } catch (e) {
        setError(String(e));
        setBulkPromptText(null);
      } finally {
        setPromptRunning(false);
      }
    },
    [planExcluded],
  );

  // applyAllCategories runs every category's action in one go (the TUI's
  // "confirm & apply the whole plan").
  const applyAllCategories = useCallback(async () => {
    if (!plan) return;
    setApplyingAll(true);
    try {
      for (const c of [...plan.categories]) {
        await applyCategory(c);
      }
      showToast("Applied the whole plan");
    } finally {
      setApplyingAll(false);
    }
  }, [plan, applyCategory, showToast]);

  // doPlanMove reassigns the pending email (or whole category) to the chosen
  // destination, mutating the in-memory plan — nothing is applied to Gmail until
  // you dispatch that category. Mirrors the TUI's action-plan move.
  const doPlanMove = useCallback(
    (target: MoveTarget) => {
      const mv = planMove;
      setPlanMove(null);
      if (!mv || !plan) return;
      const src = plan.categories[mv.catIdx];
      // A category move carries only the still-checked (non-deselected) emails,
      // so you can deselect a few and move just the rest; a single-email move
      // carries that one id.
      const ids =
        mv.kind === "email" && mv.id
          ? [mv.id]
          : src
            ? src.messageIds.filter((id) => !planExcluded.has(id))
            : [];
      if (!ids.length) return;
      setPlan((prev) =>
        prev
          ? { ...prev, categories: applyPlanMove(prev.categories, ids, target) }
          : prev,
      );
      // The moved emails are now in the target category (included by default).
      if (planExcluded.size) {
        setPlanExcluded((prev) => {
          const n = new Set(prev);
          ids.forEach((id) => n.delete(id));
          return n;
        });
      }
      showToast(
        mv.kind === "email"
          ? `Moved email to ${target.label}`
          : `Moved ${ids.length} to ${target.label}`,
      );
    },
    [planMove, plan, planExcluded, showToast],
  );

  // openPlanPreview loads an email's content into the action-plan quickview so
  // you can peek at it (like the TUI's side-by-side reading) without leaving the
  // plan — Escape returns to the tree.
  const openPlanPreview = useCallback(async (id: string) => {
    setPlanPreviewLoading(true);
    setPlanPreview(null);
    try {
      setPlanPreview(await backend.GetMessage(id));
    } catch (e) {
      setError(String(e));
    } finally {
      setPlanPreviewLoading(false);
    }
  }, []);

  // Keyboard nav for the action-plan categories. Enter applies the highlighted
  // category's action. windowKeys is gated on planOpen because this hook lives
  // in the always-mounted App, unlike the standalone pickers.
  //
  // The plan is a TREE like the TUI: arrows move through a flattened list of
  // visible nodes — every category, plus the emails of any expanded category —
  // so you can descend into a bucket and open individual messages.
  const planNodes = useMemo<PlanNode[]>(
    () => buildPlanNodes(plan, expandedCats),
    [plan, expandedCats],
  );
  const planNav = useListNav(planNodes, {
    onEnter: (node) => {
      if (node.type === "email") {
        // Peek at the email in the quickview instead of leaving the plan.
        void openPlanPreview(node.id);
        return;
      }
      const c = plan?.categories[node.catIdx];
      if (!c || c.action === "none") return;
      if (c.action === "prompt") {
        void dispatchPromptCategory(c);
        return;
      }
      // For label buckets Enter does the dominant action: move to folder
      // (label + archive). Label-only stays on its button and the `l` key.
      void applyCategory(c, c.action === "label");
    },
    onEscape: () => setPlanOpen(false),
    // Only drive the plan while it's the topmost modal — otherwise its Escape
    // would also fire and close the plan when a sub-modal (rules/prompt/move
    // chooser) is up.
    windowKeys:
      planOpen &&
      !rulesOpen &&
      promptPreview === null &&
      planMove === null &&
      planPreview === null &&
      bulkPromptText === null,
  });
  // Ref so the key handler can act on the active node without the huge onKey
  // effect depending on planNav.active (which changes on every arrow).
  const planNodesRef = useRef(planNodes);
  planNodesRef.current = planNodes;
  const planActiveRef = useRef(planNav.active);
  planActiveRef.current = planNav.active;
  const planActiveNode = planNodes[planNav.active];

  const openRules = useCallback(async () => {
    setRulesOpen(true);
    try {
      setRules(await backend.ListAnalyzerRules());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const addRule = useCallback(async () => {
    const text = newRule.trim();
    if (!text) return;
    try {
      await backend.SaveAnalyzerRule(text);
      setNewRule("");
      setRules(await backend.ListAnalyzerRules());
    } catch (e) {
      setError(String(e));
    }
  }, [newRule]);

  const deleteRule = useCallback(async (id: number) => {
    try {
      await backend.DeleteAnalyzerRule(id);
      setRules(await backend.ListAnalyzerRules());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  // Keyboard nav for the analyzer-rules modal: WKWebView won't focus the bare
  // modal, so drive arrows from the window (useListNav) and highlight the active
  // row. Escape closes.
  const viewAnalyzerPrompt = useCallback(async () => {
    try {
      setPromptPreview(await backend.ViewAnalyzerPrompt());
    } catch (e) {
      setError(String(e));
    }
  }, []);

  // Command palette dispatcher (`:` command mode).

  return {
    runActionPlan, runDeterministicRules, applyCategory, dispatchPromptCategory,
    applyAllCategories, doPlanMove, openPlanPreview, planNodes, planNav,
    planActiveNode, planNodesRef, planActiveRef, openRules, addRule, deleteRule,
    viewAnalyzerPrompt,
  };
}
