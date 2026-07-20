package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FetchImage downloads a remote image and returns it as a data: URI. Email HTML
// is rendered in-page (Shadow DOM), and macOS WKWebView won't load external
// subresources from the app's custom-scheme origin — so once the user opts in to
// loading images, the frontend routes each remote <img> through here. This also
// keeps image fetches on the app's network path (honouring the proxy) instead
// of the webview's.
func (a *App) FetchImage(rawURL string) (string, error) {
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
	req.Header.Set("User-Agent", "GizTUI-Desktop")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image fetch failed: %s", resp.Status)
	}
	// Cap the payload so a rogue URL can't balloon memory / the data URI.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 15<<20))
	if err != nil {
		return "", err
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		ct = http.DetectContentType(data)
	}
	if !strings.HasPrefix(ct, "image/") {
		return "", fmt.Errorf("not an image (%s)", ct)
	}
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
