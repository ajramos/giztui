import { describe, it, expect } from "vitest";
import { categoryLabel, filterByNameOrCategory, groupByCategory } from "./entityGroups";

type Row = { name: string; category: string };
const rows: Row[] = [
  { name: "Beta", category: "Work" },
  { name: "Alpha", category: "Work" },
  { name: "Zeta", category: "" },
  { name: "Gamma", category: "Finance" },
];
const name = (r: Row) => r.name;
const cat = (r: Row) => r.category;

describe("categoryLabel", () => {
  it("maps blank to Default", () => {
    expect(categoryLabel("")).toBe("Default");
    expect(categoryLabel("  ")).toBe("Default");
    expect(categoryLabel(" Work ")).toBe("Work");
  });
});

describe("filterByNameOrCategory", () => {
  it("filters by name, @category, or returns all", () => {
    expect(filterByNameOrCategory(rows, "", name, cat)).toHaveLength(4);
    expect(filterByNameOrCategory(rows, "alp", name, cat).map(name)).toEqual(["Alpha"]);
    expect(filterByNameOrCategory(rows, "@work", name, cat).map(name).sort()).toEqual([
      "Alpha",
      "Beta",
    ]);
    expect(filterByNameOrCategory(rows, "@default", name, cat).map(name)).toEqual(["Zeta"]);
  });
});

describe("groupByCategory", () => {
  it("sorts named groups alphabetically, Default last, names within a group", () => {
    const g = groupByCategory(rows, "", name, cat);
    expect(g.map((r) => r.item.name)).toEqual(["Gamma", "Alpha", "Beta", "Zeta"]);
    expect(g.map((r) => r.category)).toEqual(["Finance", "Work", "Work", "Default"]);
    expect(g.map((r) => r.groupStart)).toEqual([true, true, false, true]);
  });
});
