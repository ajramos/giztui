import { useEffect, useRef, type ReactNode } from "react";

// HighlightedText renders plain text with every case-insensitive occurrence of
// `query` wrapped in <mark>. The match at `activeIndex` gets the "active" style
// and is scrolled into view — this powers the reader's in-message content
// search (the TUI's "/" search within a message body).
export default function HighlightedText({
  text,
  query,
  activeIndex,
}: {
  text: string;
  query: string;
  activeIndex: number;
}) {
  const activeRef = useRef<HTMLElement>(null);

  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: "center", behavior: "smooth" });
  }, [activeIndex, query]);

  if (!query) return <>{text || "(empty body)"}</>;

  const q = query.toLowerCase();
  const lower = text.toLowerCase();
  const nodes: ReactNode[] = [];
  let from = 0;
  let matchNo = 0;
  let idx = lower.indexOf(q);
  while (idx !== -1) {
    if (idx > from) nodes.push(text.slice(from, idx));
    const isActive = matchNo === activeIndex;
    nodes.push(
      <mark
        key={idx}
        ref={isActive ? activeRef : undefined}
        className={isActive ? "cs-hit active" : "cs-hit"}
      >
        {text.slice(idx, idx + q.length)}
      </mark>,
    );
    from = idx + q.length;
    matchNo++;
    idx = lower.indexOf(q, from);
  }
  if (from < text.length) nodes.push(text.slice(from));
  return <>{nodes}</>;
}
