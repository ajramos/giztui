import type { SavedQuery } from "./api";

// savedQueryCategoryLabel is the display name of a query's category: the free-form
// string, or "Default" for uncategorised entries. Grouping, the "@category"
// filter, and the headers all share this so they agree. Mirrors the TUI helper.
export function savedQueryCategoryLabel(category: string): string {
  const t = (category || "").trim();
  return t === "" ? "Default" : t;
}

// filterSavedQueries narrows by the picker filter: a value starting with "@"
// filters by category (case-insensitive substring, e.g. "@work"); any other
// non-empty value filters by name. Empty returns everything.
export function filterSavedQueries(
  queries: SavedQuery[],
  filter: string,
): SavedQuery[] {
  const f = filter.trim().toLowerCase();
  if (f === "") return queries.slice();
  if (f.startsWith("@")) {
    const cat = f.slice(1).trim();
    if (cat === "") return queries.slice();
    return queries.filter((q) =>
      savedQueryCategoryLabel(q.category).toLowerCase().includes(cat),
    );
  }
  return queries.filter((q) => q.name.toLowerCase().includes(f));
}

// sortSavedQueriesByCategory orders queries by category — named groups
// alphabetically, the uncategorised "Default" group last — then by name, so the
// picker can render contiguous, headed groups.
export function sortSavedQueriesByCategory(queries: SavedQuery[]): SavedQuery[] {
  return queries.slice().sort((a, b) => {
    const ca = savedQueryCategoryLabel(a.category);
    const cb = savedQueryCategoryLabel(b.category);
    const da = ca === "Default";
    const db = cb === "Default";
    if (da !== db) return da ? 1 : -1; // Default group sinks to the bottom
    const byCat = ca.toLowerCase().localeCompare(cb.toLowerCase());
    if (byCat !== 0) return byCat;
    return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
  });
}

// groupedSavedQueries filters + sorts, and annotates each row with whether it
// starts a new category group (so the picker renders one header per group).
export function groupedSavedQueries(
  queries: SavedQuery[],
  filter: string,
): { query: SavedQuery; category: string; groupStart: boolean }[] {
  const rows = sortSavedQueriesByCategory(filterSavedQueries(queries, filter));
  let last = "";
  return rows.map((q, i) => {
    const category = savedQueryCategoryLabel(q.category);
    const groupStart = i === 0 || category !== last;
    last = category;
    return { query: q, category, groupStart };
  });
}
