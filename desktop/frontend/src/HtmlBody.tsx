import { useEffect, useRef, useState } from "react";
import { backend } from "./api";

// HtmlBody renders an email's HTML in a locked-down sandboxed iframe.
// See the git history for the WKWebView srcdoc origin issue; we write the HTML
// into the frame's own document to keep it same-origin.
//
// TEMPORARY DIAGNOSTIC: a small badge in the corner reports whether the app can
// read the frame's document and whether click/key events reach our listeners.
// This is here to pin down why links/shortcuts fail on macOS WKWebView.
export default function HtmlBody({
  html,
  loadRemote,
}: {
  html: string;
  loadRemote: boolean;
}) {
  const ref = useRef<HTMLIFrameElement>(null);
  const [diag, setDiag] = useState({ doc: "…", body: 0, keys: 0, clicks: 0 });

  const imgSrc = loadRemote ? "* data: cid: blob:" : "data: cid:";
  const csp = [
    "default-src 'none'",
    `img-src ${imgSrc}`,
    "style-src 'unsafe-inline'",
    `font-src ${loadRemote ? "* data:" : "data:"}`,
    "media-src data:",
  ].join("; ");

  const fullHtml = `<!doctype html><html><head><meta charset="utf-8">
<meta http-equiv="Content-Security-Policy" content="${csp}">
<style>
  html,body{margin:0;padding:16px;background:#fff;color:#111;
    font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;
    font-size:14px;line-height:1.5;word-break:break-word;}
  img{max-width:100%;height:auto;}
  a{color:#1a56db;}
  table{max-width:100%;}
</style></head><body>${html}</body></html>`;

  useEffect(() => {
    const iframe = ref.current;
    if (!iframe) return;
    let detach: (() => void) | null = null;

    const onClick = (e: MouseEvent) => {
      setDiag((d) => ({ ...d, clicks: d.clicks + 1 }));
      const a = (e.target as HTMLElement | null)?.closest?.("a");
      const href = a?.getAttribute("href");
      if (href && /^https?:/i.test(href)) {
        e.preventDefault();
        void backend.OpenURL(href);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      setDiag((d) => ({ ...d, keys: d.keys + 1 }));
      window.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: e.key,
          code: e.code,
          shiftKey: e.shiftKey,
          ctrlKey: e.ctrlKey,
          metaKey: e.metaKey,
          altKey: e.altKey,
          bubbles: true,
        }),
      );
      const scrollKeys = [
        "ArrowUp",
        "ArrowDown",
        "PageUp",
        "PageDown",
        "Home",
        "End",
      ];
      if (!scrollKeys.includes(e.key)) e.preventDefault();
    };

    const render = (): boolean => {
      let doc: Document | null = null;
      try {
        doc = iframe.contentDocument;
      } catch {
        setDiag((d) => ({ ...d, doc: "THREW" }));
        return false;
      }
      if (!doc) {
        setDiag((d) => ({ ...d, doc: "null" }));
        return false;
      }
      try {
        doc.open();
        doc.write(fullHtml);
        doc.close();
      } catch {
        setDiag((d) => ({ ...d, doc: "write-fail" }));
        return false;
      }
      detach?.();
      doc.addEventListener("click", onClick);
      doc.addEventListener("keydown", onKey);
      detach = () => {
        doc?.removeEventListener("click", onClick);
        doc?.removeEventListener("keydown", onKey);
      };
      setDiag((d) => ({ ...d, doc: "ok", body: doc?.body?.innerHTML.length ?? 0 }));
      return true;
    };

    const timers: number[] = [];
    if (!render()) {
      timers.push(window.setTimeout(render, 40));
      timers.push(window.setTimeout(render, 150));
      timers.push(window.setTimeout(render, 400));
    }
    return () => {
      timers.forEach((t) => window.clearTimeout(t));
      detach?.();
    };
  }, [fullHtml]);

  return (
    <div style={{ position: "relative", height: "100%" }}>
      <div
        style={{
          position: "absolute",
          top: 4,
          right: 4,
          zIndex: 5,
          font: "11px/1.4 monospace",
          background: "rgba(0,0,0,.72)",
          color: "#fff",
          padding: "2px 6px",
          borderRadius: 4,
          pointerEvents: "none",
        }}
      >
        doc:{diag.doc} body:{diag.body} keys:{diag.keys} clicks:{diag.clicks}
      </div>
      <iframe
        ref={ref}
        className="html-body"
        title="Email content"
        sandbox="allow-same-origin allow-popups"
      />
    </div>
  );
}
