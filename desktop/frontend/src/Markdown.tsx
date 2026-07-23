import React from "react";
import { backend } from "./api";

// Markdown renders the subset of Markdown that LLM prompt/summary output uses:
// headings, bold/italic/inline-code, links, and bullet/numbered lists. It builds
// real React nodes (never dangerouslySetInnerHTML), so untrusted model output
// can't inject HTML. Deliberately small — no dependency, keeps the Wails bundle
// self-contained. Anything it doesn't recognize renders as plain paragraph text.

// renderInline parses inline spans: **bold**, *italic* / _italic_, `code`,
// and [text](url). Links open in the system browser via the backend.
function renderInline(text: string, keyBase: string): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  const re =
    /(\*\*([^*]+)\*\*|`([^`]+)`|\[([^\]]+)\]\(([^)]+)\)|\*([^*\n]+)\*|_([^_\n]+)_)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const key = keyBase + i++;
    if (m[2] !== undefined) out.push(<strong key={key}>{m[2]}</strong>);
    else if (m[3] !== undefined) out.push(<code key={key}>{m[3]}</code>);
    else if (m[4] !== undefined) {
      const url = m[5];
      out.push(
        <a
          key={key}
          href={url}
          onClick={(e) => {
            e.preventDefault();
            if (/^https?:/i.test(url)) void backend.OpenURL(url);
          }}
        >
          {m[4]}
        </a>,
      );
    } else if (m[6] !== undefined) out.push(<em key={key}>{m[6]}</em>);
    else if (m[7] !== undefined) out.push(<em key={key}>{m[7]}</em>);
    last = re.lastIndex;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

export default function Markdown({ text }: { text: string }) {
  const lines = (text || "").replace(/\r\n/g, "\n").split("\n");
  const blocks: React.ReactNode[] = [];
  let key = 0;
  let list: { ordered: boolean; content: string }[] = [];
  let para: string[] = [];

  const flushList = () => {
    if (!list.length) return;
    const ordered = list[0].ordered;
    const k = key++;
    const items = list.map((it, idx) => (
      <li key={idx}>{renderInline(it.content, `li${k}-${idx}-`)}</li>
    ));
    blocks.push(
      ordered ? <ol key={k}>{items}</ol> : <ul key={k}>{items}</ul>,
    );
    list = [];
  };
  const flushPara = () => {
    if (!para.length) return;
    const k = key++;
    blocks.push(<p key={k}>{renderInline(para.join(" "), `p${k}-`)}</p>);
    para = [];
  };

  for (const raw of lines) {
    const line = raw.replace(/\s+$/, "");
    const heading = line.match(/^(#{1,6})\s+(.*)$/);
    const bullet = line.match(/^\s*[-*+]\s+(.*)$/);
    const numbered = line.match(/^\s*(\d+)[.)]\s+(.*)$/);
    if (heading) {
      flushPara();
      flushList();
      const lvl = Math.min(6, heading[1].length + 2);
      const k = key++;
      blocks.push(
        React.createElement(
          `h${lvl}`,
          { key: k, className: "md-h" },
          renderInline(heading[2], `h${k}-`),
        ),
      );
    } else if (bullet) {
      flushPara();
      if (list.length && list[0].ordered) flushList();
      list.push({ ordered: false, content: bullet[1] });
    } else if (numbered) {
      flushPara();
      if (list.length && !list[0].ordered) flushList();
      list.push({ ordered: true, content: numbered[2] });
    } else if (line.trim() === "") {
      flushPara();
      flushList();
    } else {
      flushList();
      para.push(line);
    }
  }
  flushPara();
  flushList();

  return <div className="md-body">{blocks}</div>;
}
