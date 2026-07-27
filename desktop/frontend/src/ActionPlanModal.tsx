import type { RefObject } from "react";
import type { ActionPlanResult, PlanCategory, MessageSummary } from "./api";
import type { PlanNode } from "./planNodes";
import { Icon } from "./Icons";
import { displayName } from "./format";

// The inbox action plan (:plan / :action-plan). Presentational: all state and
// the keyboard nav live in App (the plan tree couples to openMessage, the AI
// panels and the rules/prompt sub-modals). This renders the analyze progress,
// the category tree, and the per-category/per-email actions — a behavior-
// preserving move of the render block that used to live inline in App.tsx.
export default function ActionPlanModal({
  analyzing,
  analyzeCount,
  analyzeProgress,
  analyzeElapsed,
  plan,
  planNodes,
  planActiveNode,
  listRef,
  setActiveHover,
  expandedCats,
  setExpandedCats,
  planExcluded,
  setPlanExcluded,
  applyingAll,
  rulesEnabled,
  messages,
  onApplyCategory,
  onDispatchPrompt,
  onApplyAll,
  onOpenMessage,
  onOpenRules,
  onViewPrompt,
  onClose,
}: {
  analyzing: boolean;
  analyzeCount: number;
  analyzeProgress: { done: number; total: number } | null;
  analyzeElapsed: number;
  plan: ActionPlanResult | null;
  planNodes: PlanNode[];
  planActiveNode: PlanNode | undefined;
  listRef: RefObject<HTMLDivElement>;
  setActiveHover: (i: number) => void;
  expandedCats: Set<string>;
  setExpandedCats: (fn: (prev: Set<string>) => Set<string>) => void;
  planExcluded: Set<string>;
  setPlanExcluded: (fn: (prev: Set<string>) => Set<string>) => void;
  applyingAll: boolean;
  rulesEnabled: boolean;
  messages: MessageSummary[];
  onApplyCategory: (c: PlanCategory, move?: boolean) => void;
  onDispatchPrompt: (c: PlanCategory) => void;
  onApplyAll: () => void;
  onOpenMessage: (m: MessageSummary) => void;
  onOpenRules: () => void;
  onViewPrompt: () => void;
  onClose: () => void;
}) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal plan-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-head">
          <h3>Inbox action plan</h3>
          <span className="summary-head-actions">
            {rulesEnabled && (
              <button className="ghost tiny" onClick={() => onOpenRules()}>
                Rules (r)
              </button>
            )}
            <button className="ghost tiny" onClick={() => onViewPrompt()}>
              Prompt (p)
            </button>
            <button className="ghost" onClick={onClose}>
              {Icon.x}
            </button>
          </span>
        </div>
        <div className="modal-body">
          {analyzing ? (
            <div className="placeholder plan-analyzing">
              <div className="plan-analyzing-title">
                Analyzing {analyzeCount || messages.length} messages…
              </div>
              {analyzeProgress && analyzeProgress.total > 0 ? (
                <>
                  <div className="plan-progress">
                    {analyzeProgress.done > 0 ? (
                      <div
                        className="plan-progress-fill"
                        style={{
                          width: `${Math.round(
                            (analyzeProgress.done / analyzeProgress.total) * 100,
                          )}%`,
                        }}
                      />
                    ) : (
                      // Blocks known but none finished yet — keep animating
                      // so it reads as "working", with the total in the label.
                      <div className="plan-progress-bar" />
                    )}
                  </div>
                  <div className="muted plan-analyzing-sub">
                    Batch {analyzeProgress.done}/{analyzeProgress.total} ·{" "}
                    {analyzeElapsed}s · deterministic rules first, then AI
                  </div>
                </>
              ) : (
                <>
                  <div className="plan-progress">
                    <div className="plan-progress-bar" />
                  </div>
                  <div className="muted plan-analyzing-sub">
                    {analyzeElapsed}s elapsed · deterministic rules first, then
                    AI — this can take a moment for a large inbox
                  </div>
                </>
              )}
            </div>
          ) : !plan || plan.categories.length === 0 ? (
            <div className="placeholder">
              {plan ? "Nothing to act on" : "No plan"}
            </div>
          ) : (
            <>
              <div className="plan-summary muted">
                Analyzed {plan.totalAnalyzed} · {plan.readManually} to read
                manually
              </div>
              <div className="plan-list" ref={listRef}>
                {plan.categories.map((c, i) => (
                  <div
                    key={c.name}
                    className={
                      "plan-cat" +
                      (planActiveNode?.type === "cat" &&
                      planActiveNode.catIdx === i
                        ? " nav-active"
                        : "")
                    }
                    onMouseEnter={() =>
                      setActiveHover(
                        planNodes.findIndex(
                          (n) => n.type === "cat" && n.catIdx === i,
                        ),
                      )
                    }
                  >
                    <button
                      className="plan-cat-main"
                      title="Show emails in this category"
                      onClick={() =>
                        setExpandedCats((prev) => {
                          const n = new Set(prev);
                          if (n.has(c.name)) n.delete(c.name);
                          else n.add(c.name);
                          return n;
                        })
                      }
                    >
                      <div className="plan-cat-title">
                        <span className="conv-caret">
                          {expandedCats.has(c.name) ? "▾" : "▸"}
                        </span>
                        {c.readManually ? (
                          <span
                            className="rule-badge review-badge"
                            title="The AI left these for you to review — recategorize with m"
                          >
                            review
                          </span>
                        ) : c.byRule ? (
                          <span
                            className="rule-badge"
                            title="Resolved by a deterministic rule"
                          >
                            {Icon.bolt} rule
                          </span>
                        ) : null}
                        <strong>{c.name}</strong>
                        <span className="plan-count">{c.messageIds.length}</span>
                      </div>
                      <div className="plan-cat-desc muted">
                        {c.description}
                        {!c.readManually && !c.byRule && c.priority ? (
                          <span
                            className={"prio-tag prio-" + c.priority}
                            title="AI-assigned priority"
                          >
                            {c.priority}
                          </span>
                        ) : null}
                      </div>
                    </button>
                    {/* Read-manually has no action to apply — you recategorize
                        its emails out (m). Label buckets get two: Move (label +
                        archive, leaves inbox) primary, and Label (label only). */}
                    {!c.readManually &&
                      (c.action === "label" ? (
                        <span className="plan-cat-acts">
                          <button
                            className="tiny primary"
                            onClick={() => onApplyCategory(c, true)}
                          >
                            Move to "{c.label}"
                          </button>
                          <button
                            className="tiny"
                            onClick={() => onApplyCategory(c, false)}
                          >
                            Label "{c.label}"
                          </button>
                        </span>
                      ) : c.action === "prompt" ? (
                        <button
                          className="tiny primary"
                          disabled={!c.promptId}
                          onClick={() => onDispatchPrompt(c)}
                        >
                          Run prompt
                        </button>
                      ) : (
                        <button
                          className="tiny"
                          disabled={c.action === "none"}
                          onClick={() => onApplyCategory(c)}
                        >
                          {c.action.replace("_", " ")}
                        </button>
                      ))}
                    {expandedCats.has(c.name) && (
                      <ul className="plan-cat-emails">
                        {c.messageIds.map((id) => {
                          const m = messages.find((x) => x.id === id);
                          const excluded = planExcluded.has(id);
                          return (
                            <li
                              key={id}
                              className={
                                "plan-email" +
                                (planActiveNode?.type === "email" &&
                                planActiveNode.catIdx === i &&
                                planActiveNode.id === id
                                  ? " nav-active"
                                  : "") +
                                (excluded ? " deselected" : "")
                              }
                              onMouseEnter={() =>
                                setActiveHover(
                                  planNodes.findIndex(
                                    (n) =>
                                      n.type === "email" &&
                                      n.catIdx === i &&
                                      n.id === id,
                                  ),
                                )
                              }
                            >
                              <span
                                className="pe-check"
                                title={
                                  excluded
                                    ? "Deselected — Space to select"
                                    : "Selected — Space to deselect"
                                }
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setPlanExcluded((prev) => {
                                    const n = new Set(prev);
                                    if (n.has(id)) n.delete(id);
                                    else n.add(id);
                                    return n;
                                  });
                                }}
                              >
                                {excluded ? "☐" : "☑"}
                              </span>
                              <span
                                className="pe-body"
                                onClick={() => {
                                  if (m) onOpenMessage(m);
                                }}
                              >
                                <span className="pe-from">
                                  {m ? displayName(m.from) : id}
                                </span>
                                <span className="pe-subject muted">
                                  {m?.subject || ""}
                                </span>
                              </span>
                            </li>
                          );
                        })}
                      </ul>
                    )}
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
        <div className="modal-foot">
          <span className="foot-hint">
            ↑↓ move · → expand · Space (de)select · Enter peek/apply (label→move)
            · l label-only · m recategorize · r rules · p prompt · Esc close
          </span>
          {plan && plan.categories.length > 0 && (
            <button
              className="ghost"
              disabled={applyingAll}
              onClick={() => onApplyAll()}
            >
              {applyingAll ? "Applying…" : "Apply all"}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
