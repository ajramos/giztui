import { Fragment } from "react";
import { backend } from "./api";

// Matches http(s) URLs and bare www. hosts. Trailing punctuation that is
// usually sentence/paren noise is trimmed off the match below.
const URL_RE = /((?:https?:\/\/|www\.)[^\s<>()[\]]+)/gi;

// Trailing characters that are almost never part of the URL itself.
function trimTrail(url: string): { url: string; trail: string } {
  const m = url.match(/[.,;:!?)\]}'"]+$/);
  if (!m) return { url, trail: "" };
  return { url: url.slice(0, url.length - m[0].length), trail: m[0] };
}

// PlainBody renders a plain-text email body with clickable links. The TUI lets
// you open links from the body; on the desktop the plain-text view is a <pre>,
// so URLs were inert. Here each URL becomes an <a> that opens in the system
// browser via the backend (never navigating the app itself).
export default function PlainBody({ text }: { text: string }) {
  const parts: React.ReactNode[] = [];
  let last = 0;
  let key = 0;
  for (const m of text.matchAll(URL_RE)) {
    const start = m.index ?? 0;
    const raw = m[0];
    if (start > last) parts.push(<Fragment key={key++}>{text.slice(last, start)}</Fragment>);
    const { url, trail } = trimTrail(raw);
    const href = url.startsWith("www.") ? `https://${url}` : url;
    parts.push(
      <a
        key={key++}
        href={href}
        onClick={(e) => {
          e.preventDefault();
          void backend.OpenURL(href).catch(() => undefined);
        }}
      >
        {url}
      </a>,
    );
    if (trail) parts.push(<Fragment key={key++}>{trail}</Fragment>);
    last = start + raw.length;
  }
  if (last < text.length) parts.push(<Fragment key={key++}>{text.slice(last)}</Fragment>);
  return <pre className="plain">{parts}</pre>;
}
