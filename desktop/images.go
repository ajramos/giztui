package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// shortURL trims a URL for log lines so a long tracking URL doesn't flood the log.
func shortURL(u string) string {
	if len(u) > 120 {
		return u[:120] + "…"
	}
	return u
}

// sanitizeImageURL strips the junk a broken upstream quoted-printable decode
// leaves in an image URL. It shows up in marketing-CDN resize queries like
// "…/PUPPET.png?width=1200&…" when the sender fails to escape the '=' as "=3D",
// so the QP decoder treats the two following hex digits as an escape. Two
// flavours result, depending on the byte:
//   - a control byte (< 0x20 / 0x7f): e.g. "width=12" → byte 0x12 → "…?width\x1200&…".
//   - byte 0x80-0xFF: not valid UTF-8, so the charset decode replaces it with the
//     Unicode replacement char U+FFFD: e.g. "width=80" → "…?width�0&…".
// Both make net/url.Parse reject the URL (or the CDN 400). The image lives at the
// path regardless of the now-bogus query param, so dropping the stray rune
// recovers it — the param just becomes a harmless valueless key (e.g. "width0").
func sanitizeImageURL(u string) string {
	bad := func(r rune) bool { return r < 0x20 || r == 0x7f || r == 0xfffd }
	if strings.IndexFunc(u, bad) < 0 {
		return u
	}
	var b strings.Builder
	b.Grow(len(u))
	for _, r := range u {
		if bad(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// FetchImage downloads a remote image and returns it as a data: URI. Email HTML
// is rendered in-page (Shadow DOM), and macOS WKWebView won't load external
// subresources from the app's custom-scheme origin — so once the user opts in to
// loading images, the frontend routes each remote <img> through here. This also
// keeps image fetches on the app's network path (honouring the proxy) instead
// of the webview's.
func (a *App) FetchImage(rawURL string) (string, error) {
	rawURL = sanitizeImageURL(strings.TrimSpace(rawURL))
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "", fmt.Errorf("unsupported URL scheme")
	}
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithTimeout(base, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	// Present as a normal browser. Many newsletter image CDNs (HubSpot, beehiiv,
	// Mailchimp, …) return 403 to unknown User-Agents, which is why an image can
	// load in Gmail's proxy yet fail here — Gmail fetches with a browser-like UA.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("image: fetch error url=%s err=%v", shortURL(rawURL), err)
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		// The most useful failure to see in the log: the CDN said no (403 = UA/hotlink
		// block, 404 = gone, 3xx that didn't resolve, …).
		log.Printf("image: HTTP %d url=%s (final=%s)", resp.StatusCode, shortURL(rawURL), shortURL(resp.Request.URL.String()))
		return "", fmt.Errorf("image fetch failed: %s", resp.Status)
	}
	// Cap the payload so a rogue URL can't balloon memory / the data URI.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 15<<20))
	if err != nil {
		log.Printf("image: read error url=%s err=%v", shortURL(rawURL), err)
		return "", err
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		ct = http.DetectContentType(data)
	}
	if !strings.HasPrefix(ct, "image/") {
		log.Printf("image: not an image url=%s content-type=%q bytes=%d", shortURL(rawURL), resp.Header.Get("Content-Type"), len(data))
		return "", fmt.Errorf("not an image (%s)", ct)
	}
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	log.Printf("image: ok url=%s type=%s bytes=%d", shortURL(rawURL), ct, len(data))
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
