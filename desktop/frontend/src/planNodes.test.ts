import { describe, it, expect } from "vitest";
import { buildPlanNodes } from "./planNodes";
import type { ActionPlanResult, PlanCategory } from "./api";

const cat = (name: string, messageIds: string[]): PlanCategory => ({
  name,
  priority: "low",
  description: "",
  action: "none",
  label: "",
  messageIds,
});

const plan = (cats: PlanCategory[]): ActionPlanResult => ({
  categories: cats,
  totalAnalyzed: 0,
  readManually: 0,
});

describe("buildPlanNodes", () => {
  it("returns [] for a null plan", () => {
    expect(buildPlanNodes(null, new Set())).toEqual([]);
  });

  it("collapsed categories yield one cat node each, no emails", () => {
    const p = plan([cat("A", ["1", "2"]), cat("B", ["3"])]);
    expect(buildPlanNodes(p, new Set())).toEqual([
      { type: "cat", catIdx: 0 },
      { type: "cat", catIdx: 1 },
    ]);
  });

  it("an expanded category emits its emails after its cat node", () => {
    const p = plan([cat("A", ["1", "2"]), cat("B", ["3"])]);
    expect(buildPlanNodes(p, new Set(["A"]))).toEqual([
      { type: "cat", catIdx: 0 },
      { type: "email", catIdx: 0, id: "1" },
      { type: "email", catIdx: 0, id: "2" },
      { type: "cat", catIdx: 1 },
    ]);
  });

  it("expands by category NAME, not index, and preserves order", () => {
    const p = plan([cat("A", ["1"]), cat("B", ["2", "3"])]);
    expect(buildPlanNodes(p, new Set(["B"]))).toEqual([
      { type: "cat", catIdx: 0 },
      { type: "cat", catIdx: 1 },
      { type: "email", catIdx: 1, id: "2" },
      { type: "email", catIdx: 1, id: "3" },
    ]);
  });

  it("an expanded but empty category adds no email nodes", () => {
    const p = plan([cat("A", [])]);
    expect(buildPlanNodes(p, new Set(["A"]))).toEqual([{ type: "cat", catIdx: 0 }]);
  });
});
