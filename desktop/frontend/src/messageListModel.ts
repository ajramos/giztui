// Pure list logic for the inbox, extracted from App.tsx so the new-mail
// detection (the exact logic behind the "inbox scramble after deletes" bug) is
// unit-tested in isolation. No React, no state.

// freshPrefix returns the contiguous run of *unknown* messages at the FRONT of a
// freshly fetched inbox page. New mail always arrives at the top, so it is the
// prefix of page 1 before the first message we already have. Stopping at the
// first known id matters: after a delete, older messages shift onto page 1;
// they are unknown-to-us but NOT new, and filtering by "unknown" alone would
// prepend them to the top and scramble the order.
export function freshPrefix<T extends { id: string }>(
  fetched: T[],
  knownIds: Set<string>,
): T[] {
  const fresh: T[] = [];
  for (const m of fetched) {
    if (knownIds.has(m.id)) break;
    fresh.push(m);
  }
  return fresh;
}

// dedupeNew drops items whose id is already known — a manual refresh may have
// pulled some of the pending (banner-held) mail in before it was shown.
export function dedupeNew<T extends { id: string }>(
  pending: T[],
  knownIds: Set<string>,
): T[] {
  return pending.filter((m) => !knownIds.has(m.id));
}
