import { useEffect, useRef } from "react";
import { backend } from "./api";

// HtmlBody renders an email's HTML in a locked-down sandboxed iframe:
// - no scripts (sandbox omits allow-scripts), so the email can't run JS,
// - a strict CSP that blocks all remote content by default (privacy: no tracking
//   pixels or remote images) until the user opts in via loadRemote,
// - allow-same-origin lets THIS component (not the email) read the frame so it
//   can (a) open links in the system browser and (b) forward keystrokes back to
//   the app, so GizTUI's keyboard shortcuts keep working while reading.
//
// We do NOT use the `srcdoc` attribute: WKWebView (the macOS engine) gives a
// srcdoc document an opaque origin even with allow-same-origin, so the parent
// cannot read it — link interception and keystroke forwarding silently die once
// an email renders. Writing the HTML into the frame's about:blank document via
// document.write keeps it same-origin on every engine, so our listeners bind to
// the real rendered document.
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

    // Write the email into the frame's own (same-origin) document and bind the
    // click/key listeners to it. Idempotent: safe to run more than once.
    const render = (): boolean => {
      const doc = iframe.contentDocument;
      if (!doc) return false;
      try {
        doc.open();
        doc.write(fullHtml);
        doc.close();
      } catch {
        return false;
      }
      detach?.();
      // Bind at the document only. keydown targets the focused element and
      // bubbles up to the document, so one document listener catches every key
      // exactly once — binding the window too would double-fire and cancel out
      // toggle shortcuts.
      doc.addEventListener("click", onClick);
      doc.addEventListener("keydown", onKey);
      detach = () => {
        doc.removeEventListener("click", onClick);
        doc.removeEventListener("keydown", onKey);
      };
      return true;
    };

    // contentDocument is normally the ready about:blank on mount; retry a couple
    // of times in case the frame element isn't attached for a tick.
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
    <iframe
      ref={ref}
      className="html-body"
      title="Email content"
      sandbox="allow-same-origin allow-popups"
    />
  );
}
