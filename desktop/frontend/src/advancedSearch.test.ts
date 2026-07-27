import { describe, it, expect } from "vitest";
import { buildAdvancedQuery, EMPTY_ADV, type AdvFilters } from "./advancedSearch";

const adv = (o: Partial<AdvFilters>): AdvFilters => ({ ...EMPTY_ADV, ...o });

describe("buildAdvancedQuery", () => {
  it("is empty for empty filters", () => {
    expect(buildAdvancedQuery(EMPTY_ADV)).toBe("");
  });
  it("builds from/to and wraps subject in parens", () => {
    expect(buildAdvancedQuery(adv({ from: "a@x.com" }))).toBe("from:a@x.com");
    expect(buildAdvancedQuery(adv({ to: "b@x.com" }))).toBe("to:b@x.com");
    expect(buildAdvancedQuery(adv({ subject: "project update" }))).toBe(
      "subject:(project update)",
    );
  });
  it("emits the has:attachment / is:unread flags", () => {
    expect(buildAdvancedQuery(adv({ hasAttachment: true }))).toBe("has:attachment");
    expect(buildAdvancedQuery(adv({ unreadOnly: true }))).toBe("is:unread");
  });
  it("rewrites dates from yyyy-mm-dd to Gmail's yyyy/mm/dd", () => {
    expect(buildAdvancedQuery(adv({ after: "2026-07-01" }))).toBe("after:2026/07/01");
    expect(buildAdvancedQuery(adv({ before: "2026-07-31" }))).toBe("before:2026/07/31");
  });
  it("trims inputs and joins all parts in order with spaces", () => {
    const q = buildAdvancedQuery(
      adv({
        from: "  a@x.com ",
        subject: " hi ",
        hasAttachment: true,
        unreadOnly: true,
        after: "2026-01-02",
      }),
    );
    expect(q).toBe("from:a@x.com subject:(hi) has:attachment is:unread after:2026/01/02");
  });
});
