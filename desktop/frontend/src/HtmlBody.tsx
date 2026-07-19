import { useEffect, useRef } from "react";
import { backend } from "./api";

// HtmlBody renders an email's HTML in a locked-down sandboxed iframe:
// - no scripts (sandbox omits allow-scripts), so the email can't run JS,
// - a strict CSP that blocks all remote content by default (privacy: no tracking
//   pixels or remote images) until the user opts in via loadRemote,
// - allow-same-origin lets THIS component (not the email) read the frame so it
//   can (a) open links in the system browser and (b) forward keystrokes back to
//   the app, so GizTUI's keyboard shortcuts keep working while reading.
// Emails are authored for light backgrounds, so the frame renders on white.
export default function HtmlBody({
  html,
  loadRemote,
}: {
  html: string;
  loadRemote: boolean;
}) {
  const ref = useRef<HTMLIFrameElement>(null);

  const imgSrc = loadRemote ? "* data: cid: blob:" : "data: cid:";
  const csp = [
    "default-src 'none'",
    `img-src ${imgSrc}`,
    "style-src 'unsafe-inline'",
    `font-src ${loadRemote ? "* data:" : "data:"}`,
    "media-src data:",
  ].join("; ");

  const srcDoc = `<!doctype html><html><head><meta charset="utf-8">
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
    // Cleanup for whichever document we last attached to. wire() is idempotent:
    // it detaches the previous listeners and re-attaches to the CURRENT
    // contentDocument. This matters because a srcDoc iframe first exposes an
    // empty about:blank document and then swaps in a fresh document once the
    // srcDoc content parses — we must end up bound to that final document, not
    // the throwaway about:blank (the earlier bug: shortcuts/links were wired to
    // the stale doc, so nothing forwarded once the email rendered).
    let detach: (() => void) | null = null;

    // Open links in the system browser instead of navigating the frame.
    const onClick = (e: MouseEvent) => {
      const a = (e.target as HTMLElement | null)?.closest?.("a");
      const href = a?.getAttribute("href");
      if (href && /^https?:/i.test(href)) {
        e.preventDefault();
        void backend.OpenURL(href);
      }
    };
    // Forward keystrokes to the app so shortcuts work while the iframe (which
    // steals focus once clicked) has focus. Re-dispatch a copy on the parent
    // and swallow the original so the email doesn't also scroll/act on it.
    const onKey = (e: KeyboardEvent) => {
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
      // Let native scrolling keys still scroll the email; swallow the rest so
      // a shortcut key doesn't also do something inside the frame.
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

    const wire = () => {
      const doc = iframe.contentDocument;
      if (!doc || !doc.body) return;
      // Re-attach to the current document (detach any prior binding first).
      detach?.();
      doc.addEventListener("click", onClick);
      doc.addEventListener("keydown", onKey);
      detach = () => {
        doc.removeEventListener("click", onClick);
        doc.removeEventListener("keydown", onKey);
      };
    };

    // The load event fires once the srcDoc's final document is in place — the
    // authoritative moment to bind. The delayed retries cover engines where the
    // load event already fired before this effect ran, or fires late.
    iframe.addEventListener("load", wire);
    const t1 = window.setTimeout(wire, 120);
    const t2 = window.setTimeout(wire, 400);
    return () => {
      iframe.removeEventListener("load", wire);
      window.clearTimeout(t1);
      window.clearTimeout(t2);
      detach?.();
    };
  }, [html, loadRemote]);

  return (
    <iframe
      ref={ref}
      className="html-body"
      title="Email content"
      sandbox="allow-same-origin allow-popups"
      srcDoc={srcDoc}
    />
  );
}
