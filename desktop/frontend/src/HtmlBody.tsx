import { useEffect, useRef } from "react";
import DOMPurify from "dompurify";
import { backend } from "./api";

// HtmlBody renders an email's HTML directly in the page inside a Shadow DOM.
//
// Why not an iframe? On macOS WKWebView, click and keydown events that happen
// INSIDE a sandboxed iframe are never delivered to listeners the app attaches to
// the frame's document (verified: the frame is readable and renders, yet no
// events fire). That broke link opening and every keyboard shortcut while the
// reader had focus. Rendering in the page means events fire normally and the
// app's global shortcuts keep working; a Shadow DOM keeps the email's CSS from
// leaking into (or inheriting from) the app. Because a Shadow DOM is NOT a
// security boundary, the HTML is sanitized with DOMPurify first so no script,
// event handler, or javascript: URL from the untrusted email can run.
export default function HtmlBody({
  html,
  loadRemote,
}: {
  html: string;
  loadRemote: boolean;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const shadowRef = useRef<ShadowRoot | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    if (!shadowRef.current) {
      shadowRef.current = host.attachShadow({ mode: "open" });
    }
    const shadow = shadowRef.current;

    // Block remote images (tracking pixels) until the user opts in. Scoped to
    // this sanitize call by adding/removing the hook around it.
    const hook = (node: Element) => {
      if (node.nodeName === "IMG") {
        const src = node.getAttribute("src") || "";
        if (!loadRemote && /^https?:/i.test(src)) node.removeAttribute("src");
      }
    };
    DOMPurify.addHook("afterSanitizeAttributes", hook);
    const clean = DOMPurify.sanitize(html, {
      FORBID_TAGS: [
        "script",
        "iframe",
        "object",
        "embed",
        "form",
        "base",
        "meta",
        "link",
        "noscript",
      ],
      FORBID_ATTR: ["ping"],
    });
    DOMPurify.removeHook("afterSanitizeAttributes");

    // Emails expect a light background; render on white with the app-neutral
    // font. All styling lives inside the shadow so it can't affect the app.
    shadow.innerHTML = `<style>
      :host{ display:block; }
      .wrap{ background:#fff; color:#111; padding:16px; border-radius:8px;
        font:14px/1.5 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;
        word-break:break-word; }
      .wrap a{ color:#1a56db; cursor:pointer; }
      .wrap img{ max-width:100%; height:auto; }
      .wrap table{ max-width:100%; }
    </style><div class="wrap">${clean}</div>`;
  }, [html, loadRemote]);

  // Open links in the system browser. Clicks inside the shadow tree are composed,
  // so a listener on the host sees them and composedPath() reveals the anchor.
  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const onClick = (e: MouseEvent) => {
      const path = (e.composedPath?.() ?? []) as HTMLElement[];
      const a = path.find(
        (el) => el && el.nodeType === 1 && el.tagName === "A",
      ) as HTMLAnchorElement | undefined;
      const href = a?.getAttribute("href");
      if (href && /^https?:/i.test(href)) {
        e.preventDefault();
        void backend.OpenURL(href).catch(() => undefined);
      }
    };
    host.addEventListener("click", onClick);
    return () => host.removeEventListener("click", onClick);
  }, []);

  return <div ref={hostRef} className="html-body" />;
}
