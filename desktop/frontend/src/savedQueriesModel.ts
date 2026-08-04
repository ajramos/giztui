import type { SavedQuery } from "./api";
import { categoryLabel, filterByNameOrCategory, groupByCategory } from "./entityGroups";

// Saved-search grouping: thin adapters over the shared entityGroups helpers so
// saved searches and prompts (and any future entity picker) filter/sort/group
// through one code path and stay visually identical.
const qName = (q: SavedQuery) => q.name;
const qCat = (q: SavedQuery) => q.category;

export function savedQueryCategoryLabel(category: string): string {
  return categoryLabel(category);
}

export function filterSavedQueries(queries: SavedQuery[], filter: string): SavedQuery[] {
  return filterByNameOrCategory(queries, filter, qName, qCat);
}

export function sortSavedQueriesByCategory(queries: SavedQuery[]): SavedQuery[] {
  return groupByCategory(queries, "", qName, qCat).map((r) => r.item);
}

export function groupedSavedQueries(
  queries: SavedQuery[],
  filter: string,
): { query: SavedQuery; category: string; groupStart: boolean }[] {
  return groupByCategory(queries, filter, qName, qCat).map((r) => ({
    query: r.item,
    category: r.category,
    groupStart: r.groupStart,
  }));
}
