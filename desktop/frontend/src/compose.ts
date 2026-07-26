// Pure builders that turn an open message into the initial state for the
// composer (reply / reply-all / forward). Extracted from App.tsx so the
// recipient/subject/body derivation can be unit-tested without React.
import type { MessageDetail } from "./api";
import type { ComposeInit } from "./Compose";

export function replyInit(d: MessageDetail): ComposeInit {
  return { mode: "reply", originalId: d.id, to: d.from };
}

// Reply-all: reply to the sender, adding the original To/Cc recipients as Cc
// (de-duplicated, and never re-adding the sender).
export function replyAllInit(d: MessageDetail): ComposeInit {
  const extra = [d.to, d.cc]
    .filter(Boolean)
    .join(", ")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s && !d.from.includes(s));
  return {
    mode: "reply",
    originalId: d.id,
    to: d.from,
    cc: [...new Set(extra)].join(", "),
  };
}

export function forwardInit(d: MessageDetail): ComposeInit {
  return {
    mode: "new",
    subject: d.subject.startsWith("Fwd:") ? d.subject : `Fwd: ${d.subject}`,
    body: `\n\n---------- Forwarded message ----------\nFrom: ${d.from}\nDate: ${d.date}\nSubject: ${d.subject}\nTo: ${d.to}\n\n${d.plainText}`,
  };
}
