// HtmlBody renders an email's HTML in a locked-down sandboxed iframe:
// - no scripts (sandbox omits allow-scripts),
// - a strict CSP that blocks all remote content by default (privacy: no tracking
//   pixels or remote images) until the user opts in via loadRemote,
// - links open outside the frame.
// Emails are authored for light backgrounds, so the frame renders on white.
export default function HtmlBody({
  html,
  loadRemote,
}: {
  html: string;
  loadRemote: boolean;
}) {
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
<base target="_blank">
<style>
  html,body{margin:0;padding:16px;background:#fff;color:#111;
    font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;
    font-size:14px;line-height:1.5;word-break:break-word;}
  img{max-width:100%;height:auto;}
  a{color:#1a56db;}
  table{max-width:100%;}
</style></head><body>${html}</body></html>`;

  return (
    <iframe
      className="html-body"
      title="Email content"
      sandbox="allow-popups allow-popups-to-escape-sandbox"
      srcDoc={srcDoc}
    />
  );
}
