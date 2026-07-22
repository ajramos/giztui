import { useEffect, useRef } from "react";
import DOMPurify from "dompurify";
import { backend, type Attachment } from "./api";

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
  messageId,
  attachments,
}: {
  html: string;
  loadRemote: boolean;
  // messageId + attachments let us resolve cid: inline images to their attachment
  // bytes (fetched via the backend and inlined as data URIs).
  messageId?: string;
  attachments?: Attachment[];
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

    // Never let a remote <img> load itself: stash the URL on a data attribute
    // and drop src. That blocks tracking pixels by default, and — because macOS
    // WKWebView won't load external subresources from the app origin — lets us
    // fetch approved images through the backend as data URIs (see below).
    // Pull the first URL out of a srcset ("url1 1x, url2 2x" or "url 640w, …").
    const firstSrcsetUrl = (srcset: string): string => {
      const first = srcset.split(",")[0]?.trim() || "";
      return first.split(/\s+/)[0] || "";
    };
    const hook = (node: Element) => {
      // <source> inside <picture> carries its own srcset which the browser
      // prefers over the <img src> we proxy — neutralize it so our img wins.
      if (node.nodeName === "SOURCE") {
        node.removeAttribute("srcset");
        node.removeAttribute("src");
        return;
      }
      if (node.nodeName === "IMG") {
        let src = node.getAttribute("src") || "";
        // Newsletters commonly lazy-load: the real URL sits in data-src while
        // src holds a 1x1 placeholder (often a data: URI). Prefer the lazy URL.
        const lazy =
          node.getAttribute("data-src") ||
          node.getAttribute("data-original") ||
          node.getAttribute("data-image-src") ||
          "";
        if ((!src || /^data:/i.test(src)) && lazy) src = lazy;
        // Fall back to srcset when there's still no usable src.
        if (!src || /^data:/i.test(src)) {
          const ss = node.getAttribute("srcset") || "";
          if (ss) src = firstSrcsetUrl(ss);
        }
        // srcset/sizes would otherwise override the proxied data-URI src, so
        // strip them once we've extracted what we need.
        node.removeAttribute("srcset");
        node.removeAttribute("sizes");
        // Protocol-relative //host/img.png → https.
        if (/^\/\//.test(src)) src = "https:" + src;
        if (/^https?:/i.test(src)) {
          node.setAttribute("data-remote-src", src);
          node.removeAttribute("src");
        } else if (/^cid:/i.test(src)) {
          // Inline image: stash the Content-ID; resolved to attachment bytes
          // below. These are embedded email content, not remote trackers.
          node.setAttribute("data-cid", src.slice(4).replace(/^<|>$/g, ""));
          node.removeAttribute("src");
        }
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

    // Resolve cid: inline images to their attachment bytes (always — they're
    // embedded content, no external server, so no "load remote" opt-in needed).
    const cidImgs = Array.from(
      shadow.querySelectorAll<HTMLImageElement>("img[data-cid]"),
    );
    const remoteImgs = shadow.querySelectorAll("img[data-remote-src]").length;
    if (messageId) {
      void backend
        .LogUI(
          `htmlbody msg=${messageId}: cid-imgs=${cidImgs.length} remote-imgs=${remoteImgs} attachments=${
            attachments?.length ?? 0
          } inline-image-atts=${
            (attachments ?? []).filter(
              (a) => a.inline || a.mimeType.startsWith("image/"),
            ).length
          }`,
        )
        .catch(() => undefined);
    }
    if (messageId && attachments && attachments.length > 0 && cidImgs.length) {
      const used = new Set<string>();
      // First pass: exact Content-ID matches.
      const pending: HTMLImageElement[] = [];
      cidImgs.forEach((img) => {
        const cid = img.getAttribute("data-cid") || "";
        const att = attachments.find(
          (a) => a.contentId && a.contentId === cid,
        );
        if (att) {
          used.add(att.attachmentId);
          resolve(img, att.attachmentId);
        } else {
          pending.push(img);
        }
      });
      // Second pass: for cids no Content-ID matched (some senders omit/rewrite
      // it), fall back to unused inline image attachments in order.
      const fallback = attachments.filter(
        (a) =>
          (a.inline || a.mimeType.startsWith("image/")) &&
          !used.has(a.attachmentId),
      );
      pending.forEach((img, i) => {
        const cid = img.getAttribute("data-cid") || "";
        const att = fallback[i];
        if (!att) {
          void backend
            .LogUI(`cid ${cid} could not be resolved to any attachment`)
            .catch(() => undefined);
          return;
        }
        void backend
          .LogUI(
            `cid ${cid} unmatched by Content-ID; falling back to att=${att.attachmentId} (${att.filename})`,
          )
          .catch(() => undefined);
        resolve(img, att.attachmentId);
      });
    }

    function resolve(img: HTMLImageElement, attachmentId: string) {
      if (!messageId) return;
      void backend
        .FetchInlineImage(messageId, attachmentId)
        .then((uri) => {
          if (uri) img.setAttribute("src", uri);
        })
        .catch(() => undefined);
    }

    // Once the user opts in, fetch each remote image through the backend and
    // swap in the returned data URI (WKWebView won't fetch them in-page). Log the
    // outcome per image so a failing newsletter can be diagnosed from the log
    // (the backend logs the HTTP status / reason; here we log the swap result).
    if (loadRemote) {
      const targets = Array.from(
        shadow.querySelectorAll<HTMLImageElement>("img[data-remote-src]"),
      );
      if (targets.length && messageId) {
        void backend.LogUI(
          `htmlbody msg=${messageId}: loading ${targets.length} remote image(s)`,
        );
      }
      targets.forEach((img) => {
        const url = img.getAttribute("data-remote-src");
        if (!url) return;
        void backend
          .FetchImage(url)
          .then((uri) => {
            if (uri) {
              img.setAttribute("src", uri);
            } else {
              void backend.LogUI(`htmlbody: empty image result for ${url}`);
            }
          })
          .catch((e) => {
            void backend.LogUI(`htmlbody: image failed ${url} — ${String(e)}`);
          });
      });
    }
  }, [html, loadRemote, messageId, attachments]);

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
