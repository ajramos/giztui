// Shared category grouping for every "editable entity" picker (saved searches,
// prompts, …) so they filter, sort and group identically — the same headed,
// @category-filterable layout everywhere. Generic over the row type via accessor
// callbacks; savedQueriesModel and PromptsPicker both delegate here.

// categoryLabel is the display name of an entity's category: the free-form
// string, or "Default" for uncategorised entries. Grouping, the "@category"
// filter and the headers all share it so they agree.
export function categoryLabel(category: string): string {
  const t = (category || "").trim();
  return t === "" ? "Default" : t;
}

// filterByNameOrCategory narrows by the picker filter: a value starting with "@"
// filters by category (case-insensitive substring, e.g. "@work"); any other
// non-empty value filters by name. Empty returns everything.
export function filterByNameOrCategory<T>(
  items: T[],
  filter: string,
  getName: (t: T) => string,
  getCategory: (t: T) => string,
): T[] {
  const f = filter.trim().toLowerCase();
  if (f === "") return items.slice();
  if (f.startsWith("@")) {
    const cat = f.slice(1).trim();
    if (cat === "") return items.slice();
    return items.filter((t) =>
      categoryLabel(getCategory(t)).toLowerCase().includes(cat),
    );
  }
  return items.filter((t) => getName(t).toLowerCase().includes(f));
}

// groupByCategory filters + sorts (named groups alphabetically, "Default" last,
// then by name) and annotates each row with whether it starts a new category
// group, so a picker can render one header per group.
export function groupByCategory<T>(
  items: T[],
  filter: string,
  getName: (t: T) => string,
  getCategory: (t: T) => string,
): { item: T; category: string; groupStart: boolean }[] {
  const sorted = filterByNameOrCategory(items, filter, getName, getCategory).sort(
    (a, b) => {
      const ca = categoryLabel(getCategory(a));
      const cb = categoryLabel(getCategory(b));
      const da = ca === "Default";
      const db = cb === "Default";
      if (da !== db) return da ? 1 : -1; // Default group sinks to the bottom
      const byCat = ca.toLowerCase().localeCompare(cb.toLowerCase());
      if (byCat !== 0) return byCat;
      return getName(a).toLowerCase().localeCompare(getName(b).toLowerCase());
    },
  );
  let last = "";
  return sorted.map((item, i) => {
    const category = categoryLabel(getCategory(item));
    const groupStart = i === 0 || category !== last;
    last = category;
    return { item, category, groupStart };
  });
}
