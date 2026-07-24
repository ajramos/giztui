import type { PlanCategory } from "./api";

// A destination the action-plan move chooser can pick: a standard action (which
// resolves to / creates the category with that action) or an existing category.
export type MoveTarget =
  | { kind: "action"; action: string; label: string }
  | { kind: "category"; catName: string; label: string };

// Standard action destinations, mirroring the TUI's move chooser. "Keep (read
// manually)" is omitted: the desktop plan tracks read-manually as a count, not a
// movable message list.
const ACTION_TARGETS: MoveTarget[] = [
  { kind: "action", action: "archive", label: "Archive" },
  { kind: "action", action: "mark_read", label: "Mark read" },
  { kind: "action", action: "none", label: "No action" },
  { kind: "action", action: "trash", label: "Trash" },
];

// actionVerbLabel names a category auto-created for an action (matches the TUI).
export function actionVerbLabel(action: string): string {
  return (
    (
      {
        archive: "Archive",
        trash: "Trash",
        mark_read: "Mark read",
        label: "Label",
        none: "No action",
      } as Record<string, string>
    )[action] ?? action
  );
}

// buildMoveTargets returns the destination list: standard actions + existing
// categories (excluding the source), as one list sorted by label with
// case-insensitive duplicates dropped (a verb-named category and its action
// resolve to the same place), mirroring actionPlanMoveTargets in the TUI.
export function buildMoveTargets(
  cats: PlanCategory[],
  srcCatName: string,
): MoveTarget[] {
  const targets: MoveTarget[] = [...ACTION_TARGETS];
  for (const c of cats) {
    if (c.name === srcCatName) continue;
    const verb = actionVerbLabel(c.action);
    const label = verb !== c.name ? `${verb} · ${c.name}` : c.name;
    targets.push({ kind: "category", catName: c.name, label });
  }
  targets.sort((a, b) =>
    a.label.toLowerCase().localeCompare(b.label.toLowerCase()),
  );
  const out: MoveTarget[] = [];
  for (const t of targets) {
    if (
      out.length &&
      out[out.length - 1].label.toLowerCase() === t.label.toLowerCase()
    )
      continue;
    out.push(t);
  }
  return out;
}

// applyPlanMove returns a NEW categories array with msgIds removed from wherever
// they live and re-added to the target — resolving an action target to the first
// category with that action, or creating one. Empty categories are pruned and
// the result re-sorted by priority. Pure: it never mutates the input.
export function applyPlanMove(
  cats: PlanCategory[],
  msgIds: string[],
  target: MoveTarget,
): PlanCategory[] {
  const idSet = new Set(msgIds);
  let next: PlanCategory[] = cats.map((c) => ({
    ...c,
    messageIds: c.messageIds.filter((id) => !idSet.has(id)),
  }));

  if (target.kind === "category") {
    const idx = next.findIndex((c) => c.name === target.catName);
    if (idx >= 0)
      next[idx] = {
        ...next[idx],
        messageIds: [...next[idx].messageIds, ...msgIds],
      };
  } else {
    const idx = next.findIndex((c) => c.action === target.action);
    if (idx >= 0) {
      next[idx] = {
        ...next[idx],
        messageIds: [...next[idx].messageIds, ...msgIds],
      };
    } else {
      next.push({
        name: actionVerbLabel(target.action),
        priority: "medium",
        description: "",
        action: target.action,
        label: "",
        messageIds: [...msgIds],
        byRule: false,
      });
    }
  }

  return sortPlanCategories(next.filter((c) => c.messageIds.length > 0));
}

// sortPlanCategories orders categories by action then name — the TUI's
// SortCategories — with the read-manually bucket always pinned last. Keeps the
// order stable and matching the terminal (not by priority).
export function sortPlanCategories(cats: PlanCategory[]): PlanCategory[] {
  return [...cats].sort((a, b) => {
    if (!!a.readManually !== !!b.readManually) return a.readManually ? 1 : -1;
    const aa = a.action.toLowerCase();
    const ba = b.action.toLowerCase();
    if (aa !== ba) return aa < ba ? -1 : 1;
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });
}
