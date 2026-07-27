import type { ActionPlanResult } from "./api";

// A flattened, keyboard-navigable view of the action plan tree: every category,
// plus the emails of any expanded category. Arrows move through this list so you
// can descend into a bucket and open individual messages (mirrors the TUI tree).
export type PlanNode =
  | { type: "cat"; catIdx: number }
  | { type: "email"; catIdx: number; id: string };

// Flatten plan.categories into visible nodes given the set of expanded category
// NAMES. Pure derivation of the App's planNodes useMemo — no React, so it's unit
// testable in isolation.
export function buildPlanNodes(
  plan: ActionPlanResult | null,
  expandedCats: Set<string>,
): PlanNode[] {
  const nodes: PlanNode[] = [];
  (plan?.categories ?? []).forEach((c, ci) => {
    nodes.push({ type: "cat", catIdx: ci });
    if (expandedCats.has(c.name)) {
      c.messageIds.forEach((id) => nodes.push({ type: "email", catIdx: ci, id }));
    }
  });
  return nodes;
}
