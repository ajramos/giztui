import { describe, it, expect } from "vitest";
import {
  savedQueryCategoryLabel,
  filterSavedQueries,
  sortSavedQueriesByCategory,
  groupedSavedQueries,
} from "./savedQueriesModel";
import type { SavedQuery } from "./api";

const q = (
  id: number,
  name: string,
  category: string,
): SavedQuery => ({ id, name, query: "is:unread", description: "", category });

describe("savedQueryCategoryLabel", () => {
  it("maps blank to Default, trims otherwise", () => {
    expect(savedQueryCategoryLabel("")).toBe("Default");
    expect(savedQueryCategoryLabel("  ")).toBe("Default");
    expect(savedQueryCategoryLabel(" Work ")).toBe("Work");
  });
});

describe("filterSavedQueries", () => {
  const rows = [q(1, "Unread team", "Work"), q(2, "Invoices", "Finance"), q(3, "Misc", "")];
  it("empty returns all", () => {
    expect(filterSavedQueries(rows, "").length).toBe(3);
  });
  it("plain text filters by name", () => {
    expect(filterSavedQueries(rows, "invo").map((r) => r.id)).toEqual([2]);
  });
  it("@cat filters by category, case-insensitive", () => {
    expect(filterSavedQueries(rows, "@work").map((r) => r.id)).toEqual([1]);
    expect(filterSavedQueries(rows, "@FIN").map((r) => r.id)).toEqual([2]);
  });
  it("@default matches uncategorised", () => {
    expect(filterSavedQueries(rows, "@default").map((r) => r.id)).toEqual([3]);
  });
  it("bare @ returns all", () => {
    expect(filterSavedQueries(rows, "@").length).toBe(3);
  });
});

describe("sortSavedQueriesByCategory", () => {
  it("named groups A–Z, Default last, names within a group", () => {
    const rows = [q(1, "b", ""), q(2, "a", "Work"), q(3, "z", "Finance"), q(4, "a", ""), q(5, "B", "Work")];
    const got = sortSavedQueriesByCategory(rows).map(
      (r) => `${savedQueryCategoryLabel(r.category)}/${r.name}`,
    );
    expect(got).toEqual(["Finance/z", "Work/a", "Work/B", "Default/a", "Default/b"]);
  });
});

describe("groupedSavedQueries", () => {
  it("marks the first row of each category group", () => {
    const rows = [q(1, "a", "Work"), q(2, "b", "Work"), q(3, "c", "")];
    const got = groupedSavedQueries(rows, "");
    expect(got.map((r) => [r.category, r.groupStart])).toEqual([
      ["Work", true],
      ["Work", false],
      ["Default", true],
    ]);
  });
});
