// Advanced-search filters and the pure Gmail-query builder, extracted from
// App.tsx so the query construction is unit-tested in isolation.

export interface AdvFilters {
  from: string;
  to: string;
  subject: string;
  hasAttachment: boolean;
  unreadOnly: boolean;
  after: string; // yyyy-mm-dd (from an <input type=date>)
  before: string;
}

export const EMPTY_ADV: AdvFilters = {
  from: "",
  to: "",
  subject: "",
  hasAttachment: false,
  unreadOnly: false,
  after: "",
  before: "",
};

// buildAdvancedQuery turns the form into a Gmail search string. Dates are
// rewritten from the input's yyyy-mm-dd to Gmail's yyyy/mm/dd.
export function buildAdvancedQuery(adv: AdvFilters): string {
  const parts: string[] = [];
  if (adv.from.trim()) parts.push(`from:${adv.from.trim()}`);
  if (adv.to.trim()) parts.push(`to:${adv.to.trim()}`);
  if (adv.subject.trim()) parts.push(`subject:(${adv.subject.trim()})`);
  if (adv.hasAttachment) parts.push("has:attachment");
  if (adv.unreadOnly) parts.push("is:unread");
  if (adv.after.trim()) parts.push(`after:${adv.after.trim().replace(/-/g, "/")}`);
  if (adv.before.trim())
    parts.push(`before:${adv.before.trim().replace(/-/g, "/")}`);
  return parts.join(" ");
}
