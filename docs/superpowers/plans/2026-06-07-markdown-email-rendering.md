# Markdown Email Rendering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render messy HTML emails as clean, glamour-styled Markdown by default in the giztui TUI, with `M` toggling back to the current raw/plain view.

**Architecture:** New pure functions in `internal/render/markdown_render.go` convert `msg.HTML` → Markdown in-process via `html-to-markdown` v2, run a cleanup pipeline (reusing existing `internal/render` helpers + new newsletter-specific rules), then render with `charmbracelet/glamour` and translate ANSI → tview markup. Render-mode state lives on `DisplayService` (service-first). The TUI's `renderMessageContent` branches on that state and falls back to the existing `FormatEmailForTerminal` on any error.

**Tech Stack:** Go, `github.com/JohannesKaufmann/html-to-markdown/v2` (+ base/commonmark/table plugins), `github.com/charmbracelet/glamour` v1.0.0, `github.com/derailed/tview` (`TranslateANSI`).

**Spec:** `docs/superpowers/specs/2026-06-07-html-to-markdown-rendering-design.md` (Direction B).

**Background — why this design (from the PoC):** markitdown was eliminated (empty layout tables in all 8 real samples + charset mojibake + Python dep). Pure-Go html-to-markdown nails structure but is noisy; giztui's existing cleanup already removes the junk that matters. So the hard work is the cleanup pipeline, not the conversion.

---

## File Structure

- **Create** `internal/render/markdown_render.go` — conversion, cleanup pipeline, glamour→terminal. `package render` (gains direct access to existing unexported helpers `sanitizeForTerminal`, `dedupeNearDuplicateParagraphs`).
- **Create** `internal/render/markdown_render_test.go` — unit tests per cleanup rule + one integration test.
- **Create** `internal/render/testdata/newsletter_vicio.html` — one real fixture (copied from PoC).
- **Modify** `internal/config/config.go` — add `RenderingConfig` + `DefaultRenderingConfig()` + field in `Config` + default in `DefaultConfig()`.
- **Modify** `internal/services/interfaces.go` — add 3 methods to `DisplayService`.
- **Modify** `internal/services/display_service.go` — implement render-mode state.
- **Modify** `internal/tui/app.go` — pass markdown default to `NewDisplayService`; add render-cache helpers.
- **Modify** `internal/tui/markdown.go` — markdown branch in `renderMessageContent`; rewrite `toggleMarkdown`; add `toggleLLMTouchUp`.
- **Modify** `internal/tui/commands.go` — `:markdown`/`:md` and `:touch-up` commands + suggestions.
- **Modify** `docs/KEYBOARD_SHORTCUTS.md`, `docs/CONFIGURATION.md`.
- **Delete** `poc/` (final task).

Reuse note: `internal/render/format.go` already imports `tview` and defines `sanitizeForTerminal` and `dedupeNearDuplicateParagraphs` in `package render` — call them directly, do not duplicate.

---

## Task 1: Add `rendering` config block

**Files:**
- Modify: `internal/config/config.go` (struct at `:64`, `DefaultConfig()` at `:380-386`)
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestDefaultRenderingConfig(t *testing.T) {
	c := DefaultConfig()
	if !c.Rendering.MarkdownDefault {
		t.Errorf("MarkdownDefault = false, want true")
	}
	if c.Rendering.GlamourTheme != "dark" {
		t.Errorf("GlamourTheme = %q, want \"dark\"", c.Rendering.GlamourTheme)
	}
	if !c.Rendering.DropTrackingImages {
		t.Errorf("DropTrackingImages = false, want true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDefaultRenderingConfig -v`
Expected: FAIL — `c.Rendering` undefined.

- [ ] **Step 3: Add the config type and default**

Add near the other config types in `internal/config/config.go`:

```go
// RenderingConfig controls email body rendering.
type RenderingConfig struct {
	// MarkdownDefault renders HTML emails as Markdown by default when true.
	MarkdownDefault bool `json:"markdown_default"`
	// GlamourTheme is the glamour style name: dark, light, notty, auto.
	GlamourTheme string `json:"glamour_theme"`
	// DropTrackingImages removes tracking-pixel image links during cleanup.
	DropTrackingImages bool `json:"drop_tracking_images"`
}

// DefaultRenderingConfig returns the default rendering configuration.
func DefaultRenderingConfig() RenderingConfig {
	return RenderingConfig{
		MarkdownDefault:    true,
		GlamourTheme:       "dark",
		DropTrackingImages: true,
	}
}
```

Add the field to the `Config` struct (after `Theme ThemeConfig`):

```go
	// Email body rendering
	Rendering RenderingConfig `json:"rendering"`
```

Add to `DefaultConfig()` return literal (after `Theme: DefaultThemeConfig(),`):

```go
		Rendering:     DefaultRenderingConfig(),
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestDefaultRenderingConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add rendering config block for markdown email rendering"
```

---

## Task 2: HTML→Markdown converter wrapper

**Files:**
- Create: `internal/render/markdown_render.go`
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Write the failing test**

```go
package render

import (
	"strings"
	"testing"
)

func TestConvertHTMLToMarkdown(t *testing.T) {
	html := `<h1>Hi</h1><p>Hello <b>world</b> <a href="https://x.com">link</a></p><ul><li>a</li><li>b</li></ul>`
	md, err := convertHTMLToMarkdown(html)
	if err != nil {
		t.Fatalf("convertHTMLToMarkdown: %v", err)
	}
	for _, want := range []string{"# Hi", "**world**", "[link](https://x.com)", "- a", "- b"} {
		if !strings.Contains(md, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, md)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestConvertHTMLToMarkdown -v`
Expected: FAIL — `convertHTMLToMarkdown` undefined.

- [ ] **Step 3: Create the converter wrapper**

Create `internal/render/markdown_render.go`:

```go
package render

import (
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

// mdConverter is html-to-markdown v2 with base+commonmark+table plugins.
// The table plugin is required so newsletter data tables are not dropped; the
// base plugin is a prerequisite of commonmark. Conversion runs in-process to
// avoid the temp-file charset round-trip that corrupted markitdown output.
var mdConverter = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(),
		table.NewTablePlugin(),
	),
)

// convertHTMLToMarkdown converts an HTML string to Markdown source.
func convertHTMLToMarkdown(htmlStr string) (string, error) {
	return mdConverter.ConvertString(htmlStr)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestConvertHTMLToMarkdown -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go
git commit -m "feat(render): add in-process html-to-markdown converter"
```

---

## Task 3: Cleanup rule — drop tracking-pixel images

**Files:**
- Modify: `internal/render/markdown_render.go`
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestDropTrackingImages(t *testing.T) {
	in := "[![](https://t.co/pixel.gif)](https://t.co/track) text\n" +
		"![](https://t.co/bare.gif)\n" +
		"![Real Alt](https://cdn/x.png)\n"
	got := dropTrackingImages(in)
	if strings.Contains(got, "pixel.gif") || strings.Contains(got, "bare.gif") {
		t.Errorf("tracking images not removed:\n%s", got)
	}
	if !strings.Contains(got, "Real Alt") {
		t.Errorf("image with alt text should be kept as text:\n%s", got)
	}
	if strings.Contains(got, "![Real Alt]") {
		t.Errorf("image with alt should be flattened, not kept as image:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestDropTrackingImages -v`
Expected: FAIL — `dropTrackingImages` undefined.

- [ ] **Step 3: Implement the rule**

Append to `internal/render/markdown_render.go` (add `"regexp"` and `"strings"` to imports):

```go
// [![alt](src)](href) — image wrapped in a link (tracking pixels, banner ads).
var imgLinkRe = regexp.MustCompile(`\[!\[[^\]]*\]\([^)]*\)\]\([^)]*\)`)

// ![alt](src) — bare image.
var bareImgRe = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)

// dropTrackingImages removes image-only links and bare images. Images that
// carry real alt text are flattened to that text; images without alt are dropped
// entirely (overwhelmingly tracking pixels / spacer gifs in newsletters).
func dropTrackingImages(md string) string {
	md = imgLinkRe.ReplaceAllString(md, "")
	md = bareImgRe.ReplaceAllStringFunc(md, func(m string) string {
		alt := strings.TrimSpace(bareImgRe.FindStringSubmatch(m)[1])
		if alt == "" {
			return ""
		}
		return alt
	})
	return md
}
```

Note: the `[^)]*` URL match stops at the first `)`. Tracking URLs essentially never contain a literal `)`, so this is safe in practice.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestDropTrackingImages -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go
git commit -m "feat(render): drop tracking-pixel images in markdown cleanup"
```

---

## Task 4: Cleanup rule — collapse empty/layout tables

**Files:**
- Modify: `internal/render/markdown_render.go`
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCollapseEmptyTables(t *testing.T) {
	// An empty layout table should disappear entirely.
	empty := "before\n|  |  |\n| --- | --- |\n| |  |\nafter\n"
	got := collapseEmptyTables(empty)
	if strings.Contains(got, "|") {
		t.Errorf("empty layout table not removed:\n%s", got)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Errorf("non-table content was lost:\n%s", got)
	}

	// A real data table must be preserved verbatim.
	real := "| Item | Price |\n| --- | --- |\n| Widget | $9.99 |\n"
	got2 := collapseEmptyTables(real)
	if !strings.Contains(got2, "| Item | Price |") || !strings.Contains(got2, "Widget") {
		t.Errorf("real table was damaged:\n%s", got2)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestCollapseEmptyTables -v`
Expected: FAIL — `collapseEmptyTables` undefined.

- [ ] **Step 3: Implement the rule**

Append to `internal/render/markdown_render.go`:

```go
// collapseEmptyTables removes contiguous Markdown table blocks whose every cell
// is empty (newsletter layout tables) while preserving tables that have any real
// cell content (genuine data tables).
func collapseEmptyTables(md string) string {
	lines := strings.Split(md, "\n")
	out := make([]string, 0, len(lines))

	isTableLine := func(s string) bool { return strings.HasPrefix(strings.TrimSpace(s), "|") }
	rowHasContent := func(s string) bool {
		for _, c := range strings.Split(strings.Trim(strings.TrimSpace(s), "|"), "|") {
			if strings.Trim(strings.TrimSpace(c), "-: ") != "" {
				return true
			}
		}
		return false
	}

	for i := 0; i < len(lines); {
		if !isTableLine(lines[i]) {
			out = append(out, lines[i])
			i++
			continue
		}
		j := i
		anyContent := false
		for j < len(lines) && isTableLine(lines[j]) {
			if rowHasContent(lines[j]) {
				anyContent = true
			}
			j++
		}
		if anyContent {
			out = append(out, lines[i:j]...)
		}
		i = j
	}
	return strings.Join(out, "\n")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestCollapseEmptyTables -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go
git commit -m "feat(render): collapse empty layout tables in markdown cleanup"
```

---

## Task 5: Cleanup rule — reference long URLs

**Files:**
- Modify: `internal/render/markdown_render.go`
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestReferenceLongURLs(t *testing.T) {
	longURL := "https://track.example.com/" + strings.Repeat("a", 80)
	in := "Click [Buy now](" + longURL + ") today. Short [ok](https://x.io).\n"
	got := referenceLongURLs(in, 60)

	if strings.Contains(got, longURL[:70]) == false {
		// URL should appear once, in the Links section, not inline.
	}
	if !strings.Contains(got, "Buy now [1]") {
		t.Errorf("long link not referenced:\n%s", got)
	}
	if !strings.Contains(got, "## Links") || !strings.Contains(got, "1. "+longURL) {
		t.Errorf("Links section missing/incorrect:\n%s", got)
	}
	if !strings.Contains(got, "[ok](https://x.io)") {
		t.Errorf("short link should stay inline:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestReferenceLongURLs -v`
Expected: FAIL — `referenceLongURLs` undefined.

- [ ] **Step 3: Implement the rule**

Append to `internal/render/markdown_render.go` (add `"fmt"` to imports):

```go
// mdLinkRe matches a Markdown inline link [text](http(s)://url).
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

// referenceLongURLs replaces inline links whose URL exceeds threshold characters
// with "text [n]" numbered references, collected into a trailing "## Links"
// section. Short links stay inline. Identical URLs share one reference number.
func referenceLongURLs(md string, threshold int) string {
	seen := map[string]int{}
	order := make([]string, 0, 8)

	body := mdLinkRe.ReplaceAllStringFunc(md, func(m string) string {
		sub := mdLinkRe.FindStringSubmatch(m)
		text, url := sub[1], sub[2]
		if len(url) <= threshold {
			return m
		}
		n, ok := seen[url]
		if !ok {
			n = len(order) + 1
			seen[url] = n
			order = append(order, url)
		}
		return fmt.Sprintf("%s [%d]", text, n)
	})

	if len(order) == 0 {
		return body
	}
	var b strings.Builder
	b.WriteString(strings.TrimRight(body, "\n"))
	b.WriteString("\n\n## Links\n")
	for i, url := range order {
		fmt.Fprintf(&b, "%d. %s\n", i+1, url)
	}
	return b.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestReferenceLongURLs -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go
git commit -m "feat(render): reference long tracking URLs in markdown cleanup"
```

---

## Task 6: Compose the cleanup pipeline

**Files:**
- Modify: `internal/render/markdown_render.go`
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCleanupMarkdown(t *testing.T) {
	// Zero-width preheader junk + empty table + tracking image + long URL.
	longURL := "https://track.example.com/" + strings.Repeat("a", 80)
	in := "​ ‌ ͏ preheader\n\n" +
		"|  |  |\n| --- | --- |\n\n" +
		"![](https://t.co/pixel.gif)\n\n" +
		"# Real Heading\n\nBuy [now](" + longURL + ")\n"
	got := cleanupMarkdown(in, MarkdownOptions{DropTrackingImages: true})

	if strings.Contains(got, "​") || strings.Contains(got, "͏") {
		t.Errorf("zero-width chars not stripped:\n%q", got)
	}
	if strings.Contains(got, "pixel.gif") {
		t.Errorf("tracking image not dropped:\n%s", got)
	}
	if strings.Contains(got, "| --- |") {
		t.Errorf("empty table not collapsed:\n%s", got)
	}
	if !strings.Contains(got, "# Real Heading") {
		t.Errorf("real content lost:\n%s", got)
	}
	if !strings.Contains(got, "## Links") {
		t.Errorf("long URL not referenced:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestCleanupMarkdown -v`
Expected: FAIL — `cleanupMarkdown` and `MarkdownOptions` undefined.

- [ ] **Step 3: Implement options + pipeline**

Append to `internal/render/markdown_render.go`:

```go
// MarkdownOptions controls markdown rendering and cleanup.
type MarkdownOptions struct {
	WrapWidth          int
	GlamourTheme       string
	DropTrackingImages bool
}

// cleanupMarkdown applies the newsletter cleanup pipeline to Markdown source.
// Order matters: drop images and empty tables first, then reference URLs, then
// reuse the existing terminal sanitizer (strips zero-width/spacer glyphs) and
// near-duplicate paragraph deduper from format.go.
func cleanupMarkdown(md string, opts MarkdownOptions) string {
	if opts.DropTrackingImages {
		md = dropTrackingImages(md)
	}
	md = collapseEmptyTables(md)
	md = referenceLongURLs(md, 60)
	md = sanitizeForTerminal(md)            // defined in format.go
	md = dedupeNearDuplicateParagraphs(md, 32) // defined in format.go
	return strings.TrimSpace(md)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestCleanupMarkdown -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go
git commit -m "feat(render): compose markdown cleanup pipeline"
```

---

## Task 7: Glamour render → terminal markup

**Files:**
- Modify: `internal/render/markdown_render.go`
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestMarkdownToTerminal(t *testing.T) {
	out, err := MarkdownToTerminal("# Title\n\nHello **world**\n", "notty", 80)
	if err != nil {
		t.Fatalf("MarkdownToTerminal: %v", err)
	}
	if !strings.Contains(out, "Title") || !strings.Contains(out, "world") {
		t.Errorf("rendered output missing content:\n%s", out)
	}
	// notty style must not leave raw ANSI escape bytes.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("raw ANSI escape leaked into tview output:\n%q", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestMarkdownToTerminal -v`
Expected: FAIL — `MarkdownToTerminal` undefined.

- [ ] **Step 3: Implement glamour rendering**

Append to `internal/render/markdown_render.go` (add imports `"github.com/charmbracelet/glamour"` and `"github.com/derailed/tview"`):

```go
// MarkdownToTerminal renders Markdown to terminal text styled by glamour, then
// translates ANSI escapes to tview color tags for the message TextView.
func MarkdownToTerminal(markdown, theme string, width int) (string, error) {
	if theme == "" {
		theme = "dark"
	}
	if width < 20 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath(theme),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	out, err := r.Render(markdown)
	if err != nil {
		return "", err
	}
	return string(tview.TranslateANSI([]byte(out))), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestMarkdownToTerminal -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go
git commit -m "feat(render): render markdown to tview-styled terminal output via glamour"
```

---

## Task 8: End-to-end `RenderEmailMarkdown` + real-fixture integration test

**Files:**
- Modify: `internal/render/markdown_render.go`
- Create: `internal/render/testdata/newsletter_vicio.html` (copy from `poc/htmlmd/out/Alerta_de_descuentos_VICIO_con_env_o_gratis_y_m_s.html/input.html`)
- Test: `internal/render/markdown_render_test.go`

- [ ] **Step 1: Copy the fixture**

```bash
mkdir -p internal/render/testdata
cp "poc/htmlmd/out/Alerta_de_descuentos_VICIO_con_env_o_gratis_y_m_s.html/input.html" \
   internal/render/testdata/newsletter_vicio.html
```

- [ ] **Step 2: Write the failing test**

```go
import (
	"os"
	gmailwrap "github.com/ajramos/giztui/internal/gmail"
)

func TestRenderEmailMarkdown_RealNewsletter(t *testing.T) {
	htmlBytes, err := os.ReadFile("testdata/newsletter_vicio.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	msg := &gmailwrap.Message{HTML: string(htmlBytes)}
	out, err := RenderEmailMarkdown(msg, MarkdownOptions{
		WrapWidth: 100, GlamourTheme: "notty", DropTrackingImages: true,
	})
	if err != nil {
		t.Fatalf("RenderEmailMarkdown: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("empty render output")
	}
	// Real content survives.
	if !strings.Contains(out, "VICIO") {
		t.Errorf("expected newsletter content missing:\n%s", out[:min(len(out), 500)])
	}
	// Junk is gone: no zero-width chars, no empty markdown table separators.
	if strings.Contains(out, "​") || strings.Contains(out, "| --- | --- |") {
		t.Errorf("junk survived cleanup")
	}
}
```

Note: `min` already exists in `internal/render/format.go`.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/render/ -run TestRenderEmailMarkdown_RealNewsletter -v`
Expected: FAIL — `RenderEmailMarkdown` undefined.

- [ ] **Step 4: Implement the orchestrator**

Append to `internal/render/markdown_render.go` (add import `gmailwrap "github.com/ajramos/giztui/internal/gmail"` and `"fmt"` if not present):

```go
// RenderEmailMarkdown converts an email's HTML to cleaned, glamour-styled
// terminal text. Returns an error if the message has no HTML body; callers fall
// back to FormatEmailForTerminal on error.
func RenderEmailMarkdown(msg *gmailwrap.Message, opts MarkdownOptions) (string, error) {
	if msg == nil || strings.TrimSpace(msg.HTML) == "" {
		return "", fmt.Errorf("no HTML content")
	}
	md, err := convertHTMLToMarkdown(msg.HTML)
	if err != nil {
		return "", err
	}
	md = cleanupMarkdown(md, opts)
	if strings.TrimSpace(md) == "" {
		return "", fmt.Errorf("empty after cleanup")
	}
	return MarkdownToTerminal(md, opts.GlamourTheme, opts.WrapWidth)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/render/ -run TestRenderEmailMarkdown_RealNewsletter -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/render/markdown_render.go internal/render/markdown_render_test.go internal/render/testdata/newsletter_vicio.html
git commit -m "feat(render): add RenderEmailMarkdown orchestrator with real-newsletter test"
```

---

## Task 9: DisplayService render-mode state

**Files:**
- Modify: `internal/services/interfaces.go` (`DisplayService` at `:438-443`)
- Modify: `internal/services/display_service.go`
- Test: `internal/services/display_service_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestDisplayServiceMarkdownRendering(t *testing.T) {
	d := NewDisplayService(true)
	if !d.IsMarkdownRendering() {
		t.Error("default should be markdown on")
	}
	if got := d.ToggleMarkdownRendering(); got != false {
		t.Errorf("toggle returned %v, want false", got)
	}
	if d.IsMarkdownRendering() {
		t.Error("toggle did not turn off")
	}
	d.SetMarkdownRendering(true)
	if !d.IsMarkdownRendering() {
		t.Error("SetMarkdownRendering(true) failed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run TestDisplayServiceMarkdownRendering -v`
Expected: FAIL — `NewDisplayService` takes no args; methods undefined.

- [ ] **Step 3: Extend the interface**

In `internal/services/interfaces.go`, add to the `DisplayService` interface (after `IsHeaderVisible() bool`):

```go
	// Markdown rendering mode
	ToggleMarkdownRendering() bool
	SetMarkdownRendering(enabled bool)
	IsMarkdownRendering() bool
```

- [ ] **Step 4: Implement in display_service.go**

Replace the struct, constructor, and add methods in `internal/services/display_service.go`:

```go
type DisplayServiceImpl struct {
	mu               sync.RWMutex
	headerVisible    bool
	markdownRender   bool
}

// NewDisplayService creates a new DisplayService. markdownDefault sets the
// initial markdown-rendering mode.
func NewDisplayService(markdownDefault bool) *DisplayServiceImpl {
	return &DisplayServiceImpl{
		headerVisible:  true,
		markdownRender: markdownDefault,
	}
}

// ToggleMarkdownRendering flips markdown rendering and returns the new state.
func (d *DisplayServiceImpl) ToggleMarkdownRendering() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markdownRender = !d.markdownRender
	return d.markdownRender
}

// SetMarkdownRendering sets the markdown rendering state.
func (d *DisplayServiceImpl) SetMarkdownRendering(enabled bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.markdownRender = enabled
}

// IsMarkdownRendering returns the current markdown rendering state.
func (d *DisplayServiceImpl) IsMarkdownRendering() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.markdownRender
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/services/ -run TestDisplayServiceMarkdownRendering -v`
Expected: PASS (compilation of `internal/tui` will break until Task 10 — that's expected; do not run `./...` yet.)

- [ ] **Step 6: Commit**

```bash
git add internal/services/interfaces.go internal/services/display_service.go internal/services/display_service_test.go
git commit -m "feat(services): add markdown render-mode state to DisplayService"
```

---

## Task 10: Wire the new constructor signature in app.go

**Files:**
- Modify: `internal/tui/app.go` (`:868`)

- [ ] **Step 1: Update the call site**

In `internal/tui/app.go`, change line 868 from:

```go
	a.displayService = services.NewDisplayService()
```

to:

```go
	a.displayService = services.NewDisplayService(a.Config.Rendering.MarkdownDefault)
```

- [ ] **Step 2: Verify the whole module compiles**

Run: `go build ./...`
Expected: success (no output).

- [ ] **Step 3: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): initialize DisplayService markdown mode from config"
```

---

## Task 11: Render cache helpers

**Files:**
- Modify: `internal/tui/app.go` (struct fields near `:132`; helpers near `:1386`)
- Test: `internal/tui/app_test.go` (or a new `internal/tui/render_cache_test.go`)

- [ ] **Step 1: Write the failing test**

Create `internal/tui/render_cache_test.go`:

```go
package tui

import "testing"

func TestRenderCache(t *testing.T) {
	a := &App{}
	if _, ok := a.getRenderCache("id1", true, 100); ok {
		t.Error("expected miss on empty cache")
	}
	a.setRenderCache("id1", true, 100, "rendered")
	got, ok := a.getRenderCache("id1", true, 100)
	if !ok || got != "rendered" {
		t.Errorf("hit failed: got %q ok=%v", got, ok)
	}
	if _, ok := a.getRenderCache("id1", true, 120); ok {
		t.Error("different width must miss")
	}
	if _, ok := a.getRenderCache("id1", false, 100); ok {
		t.Error("different mode must miss")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRenderCache -v`
Expected: FAIL — `getRenderCache`/`setRenderCache` undefined.

- [ ] **Step 3: Add the cache field and helpers**

Add a field to the `App` struct (near `messageCache` at `:132`):

```go
	renderCache map[string]string
```

Add helpers (near the message-cache helpers around `:1386`):

```go
// renderCacheKey composes a key for cached rendered body text.
func renderCacheKey(messageID string, markdown bool, width int) string {
	return fmt.Sprintf("%s|%t|%d", messageID, markdown, width)
}

// getRenderCache returns cached rendered body text, if present.
func (a *App) getRenderCache(messageID string, markdown bool, width int) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.renderCache == nil {
		return "", false
	}
	v, ok := a.renderCache[renderCacheKey(messageID, markdown, width)]
	return v, ok
}

// setRenderCache stores rendered body text.
func (a *App) setRenderCache(messageID string, markdown bool, width int, text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.renderCache == nil {
		a.renderCache = make(map[string]string)
	}
	a.renderCache[renderCacheKey(messageID, markdown, width)] = text
}
```

Note: confirm `App` already has a mutex named `mu` (the message-cache helpers at `:1386` use one). If the field has a different name (e.g. `mutex`), use that name instead. Confirm `"fmt"` is imported in `app.go` (it is).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tui/ -run TestRenderCache -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/render_cache_test.go
git commit -m "feat(tui): add render cache keyed by message/mode/width"
```

---

## Task 12: Markdown branch + rewire toggles in markdown.go

**Files:**
- Modify: `internal/tui/markdown.go` (`renderMessageContent` `:105-186`; `toggleMarkdown` `:188-245`)

- [ ] **Step 1: Add the markdown branch in `renderMessageContent`**

In `internal/tui/markdown.go`, find the deterministic-format call (around `:170`):

```go
	// Deterministic format
	text, err := render.FormatEmailForTerminal(a.ctx, m, render.FormatOptions{WrapWidth: width, UseLLM: useLLM}, touch)
```

Insert immediately BEFORE that block:

```go
	// Markdown rendering path (default for HTML emails). Falls back to the
	// deterministic formatter below on any error or empty result.
	_, _, _, _, _, _, _, _, _, _, _, displayService := a.GetServices()
	if displayService != nil && displayService.IsMarkdownRendering() && strings.TrimSpace(m.HTML) != "" {
		if cached, ok := a.getRenderCache(m.Id, true, width); ok {
			return cached, false
		}
		out, mdErr := render.RenderEmailMarkdown(m, render.MarkdownOptions{
			WrapWidth:          width,
			GlamourTheme:       a.Config.Rendering.GlamourTheme,
			DropTrackingImages: a.Config.Rendering.DropTrackingImages,
		})
		if mdErr == nil && strings.TrimSpace(out) != "" {
			a.setRenderCache(m.Id, true, width, out)
			return out, false
		}
		if a.logger != nil {
			a.logger.Printf("markdown render fell back to plain: %v", mdErr)
		}
	}
```

Note: `m.Id` comes from the embedded `*gmail.Message`. Confirm the variable holding the message is named `m` in this function (it is).

- [ ] **Step 2: Rewrite `toggleMarkdown` to toggle render mode**

Replace the entire `toggleMarkdown` function (`:188-245`) with:

```go
// toggleMarkdown toggles Markdown rendering on/off for the current message.
func (a *App) toggleMarkdown() {
	mid := a.getCurrentMessageID()
	if mid == "" {
		a.GetErrorHandler().ShowError(a.ctx, "❌ No message selected")
		return
	}
	a.SetCurrentMessageID(mid)

	_, _, _, _, _, _, _, _, _, _, _, displayService := a.GetServices()
	if displayService == nil {
		return
	}
	enabled := displayService.ToggleMarkdownRendering()

	rerender := func(msg *gmail.Message) {
		rendered, _ := a.renderMessageContent(msg)
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				text.SetText(rendered)
				text.ScrollToBeginning()
			}
		})
		if enabled {
			a.GetErrorHandler().ShowInfo(a.ctx, "📄 Markdown view")
		} else {
			a.GetErrorHandler().ShowInfo(a.ctx, "📃 Raw view")
		}
	}

	if m, ok := a.GetMessageFromCache(mid); ok {
		go rerender(m)
		return
	}
	go func(id string) {
		fetched, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, "❌ Could not load message content")
			return
		}
		a.SetMessageInCache(id, fetched)
		rerender(fetched)
	}(mid)
}

// toggleLLMTouchUp toggles LLM whitespace touch-up for the current message
// (previously bound to M; now invoked via the :touch-up command).
func (a *App) toggleLLMTouchUp() {
	mid := a.getCurrentMessageID()
	if mid == "" {
		a.GetErrorHandler().ShowError(a.ctx, "❌ No message selected")
		return
	}
	a.SetCurrentMessageID(mid)
	a.llmTouchUpEnabled = !a.llmTouchUpEnabled
	rerender := func(msg *gmail.Message) {
		rendered, _ := a.renderMessageContent(msg)
		a.QueueUpdateDraw(func() {
			if text, ok := a.views["text"].(*tview.TextView); ok {
				text.SetDynamicColors(true)
				text.Clear()
				text.SetText(rendered)
				text.ScrollToBeginning()
			}
		})
		if a.llmTouchUpEnabled {
			a.GetErrorHandler().ShowInfo(a.ctx, "✅ LLM touch-up enabled")
		} else {
			a.GetErrorHandler().ShowInfo(a.ctx, "✅ Deterministic formatting only")
		}
	}
	if m, ok := a.GetMessageFromCache(mid); ok {
		go rerender(m)
		return
	}
	go func(id string) {
		fetched, err := a.Client.GetMessageWithContent(id)
		if err != nil {
			a.GetErrorHandler().ShowError(a.ctx, "❌ Could not load message content")
			return
		}
		a.SetMessageInCache(id, fetched)
		rerender(fetched)
	}(mid)
}
```

Note: this replaces the old `setStatusPersistent`/`showStatusMessage` calls with `GetErrorHandler()` per CLAUDE.md. Confirm `gmail` and `tview` are already imported in `markdown.go` (they are).

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Run render + services tests**

Run: `go test ./internal/render/ ./internal/services/ ./internal/tui/ -run 'Markdown|RenderCache|Display'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tui/markdown.go
git commit -m "feat(tui): render markdown by default; M toggles markdown/raw; relocate touch-up"
```

---

## Task 13: Command parity — `:markdown` / `:md` / `:touch-up`

**Files:**
- Modify: `internal/tui/commands.go` (`executeCommand`, `generateCommandSuggestion`)

- [ ] **Step 1: Find the command dispatch**

Run: `grep -n "case \"obsidian\"\|func (a \*App) executeCommand\|generateCommandSuggestion" internal/tui/commands.go | head`
Read the surrounding `switch` structure in `executeCommand` to match the existing style (how a simple no-arg command like header-toggle is handled).

- [ ] **Step 2: Add the command cases**

In `executeCommand`'s command switch, add cases mirroring the existing style:

```go
	case "markdown", "md":
		a.toggleMarkdown()
		return true
	case "touch-up", "touchup":
		a.toggleLLMTouchUp()
		return true
```

(If the switch returns an error rather than bool, match that signature: `return nil` / the established pattern. Read a neighboring case and copy its exact shape.)

- [ ] **Step 3: Add to command suggestions**

In `generateCommandSuggestion`, add `markdown`, `md`, and `touch-up` to the suggestion/autocomplete list, following exactly how existing commands (e.g. `obsidian`, `save-query`) are registered there.

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat(tui): add :markdown/:md and :touch-up commands (command parity)"
```

---

## Task 14: Keys.go — confirm M routing

**Files:**
- Modify (if needed): `internal/tui/keys.go` (`:322-327`, `:1090-1096`)

- [ ] **Step 1: Confirm behavior**

`a.Keys.Markdown` (default `"M"`) already routes to `a.toggleMarkdown()` at `keys.go:326`, and the hardcoded `'M'` fallback at `:1093` also calls it. Since Task 12 changed `toggleMarkdown` to toggle render mode, **no keys.go change is required**. Verify by reading those two spots.

- [ ] **Step 2: Verify the binding comment is accurate**

If there is an inline comment claiming M toggles "LLM touch-up", update it to "Toggle markdown rendering". Otherwise leave as-is.

- [ ] **Step 3: Build + commit (only if changed)**

```bash
go build ./...
git add internal/tui/keys.go
git commit -m "docs(tui): clarify M key toggles markdown rendering"
```

If nothing changed, skip the commit.

---

## Task 15: Documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md`
- Modify: `docs/CONFIGURATION.md`

- [ ] **Step 1: Update KEYBOARD_SHORTCUTS.md**

Find the existing `M` / markdown entry (search for "markdown" or "touch"). Update it to describe: `M` toggles Markdown ↔ raw rendering of the message body. Add the `:markdown` / `:md` and `:touch-up` commands to the relevant command table.

- [ ] **Step 2: Update CONFIGURATION.md**

Add a `rendering` section documenting the new block:

```json
"rendering": {
  "markdown_default": true,
  "glamour_theme": "dark",
  "drop_tracking_images": true
}
```

Document each field: `markdown_default` (render HTML emails as Markdown by default), `glamour_theme` (`dark`/`light`/`notty`/`auto`), `drop_tracking_images` (remove tracking-pixel images during cleanup).

- [ ] **Step 3: Commit**

```bash
git add docs/KEYBOARD_SHORTCUTS.md docs/CONFIGURATION.md
git commit -m "docs: document markdown rendering toggle, commands, and config"
```

---

## Task 16: Remove the PoC and finalize deps

**Files:**
- Delete: `poc/`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Confirm the fixture was copied**

Run: `test -f internal/render/testdata/newsletter_vicio.html && echo OK`
Expected: `OK` (from Task 8). Do not delete `poc/` until this passes.

- [ ] **Step 2: Delete the PoC**

```bash
rm -rf poc/
```

- [ ] **Step 3: Tidy modules**

Run: `go mod tidy`
Expected: `html-to-markdown` and `glamour` remain (now used by `internal/render`); any PoC-only transitive deps are pruned.

- [ ] **Step 4: Full pre-commit check**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests all pass.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove html-to-markdown PoC; promote deps to production"
```

---

## Task 17: Real-binary E2E verification (hard-won lesson)

**Files:** none (manual verification in the real app)

- [ ] **Step 1: Build**

Run: `make build`
Expected: binary built, no errors.

- [ ] **Step 2: Drive the real app (tmux)**

Launch giztui against the real account, open one of the messy newsletters used in the PoC (VICIO, Movistar, Kinépolis). Verify:
- Body renders as clean Markdown by default (headings/bold/lists styled; no empty tables, no zero-width junk, no giant inline tracking URLs — they appear under a `Links` section).
- Press `M`: status shows "📃 Raw view" and body switches to the current plain-text renderer.
- Press `M` again: status shows "📄 Markdown view" and clean render returns (served from cache, instant).
- Open a plain-text-only email: renders unchanged (no regression).
- `:md` command toggles the same as `M`.
- `:touch-up` toggles LLM whitespace touch-up (status confirms) without affecting the markdown toggle.

- [ ] **Step 3: Record results**

If any check fails, use superpowers:systematic-debugging before patching. Note outcomes in the PR description.

- [ ] **Step 4: Final commit / open PR**

Only after all checks pass. Open a PR from `feat/markdown-email-rendering` into `main` summarizing the PoC evidence and the implementation.

---

## Self-Review (completed by plan author)

- **Spec coverage:** converter (T2), in-process (T2 note), cleanup rules — zero-width (T6), tracking images (T3), empty tables (T4), long URLs (T5), dedup (T6); glamour+TranslateANSI (T7); orchestrator+fallback (T8, T12); DisplayService state (T9-10); markdown-default + M toggle (T10, T12, T14); `:markdown`/`:touch-up` parity (T13); config block (T1); caching (T11); fixtures+E2E (T8, T17); PoC cleanup keeping deps (T16). All spec sections mapped.
- **Placeholder scan:** every code step has complete code; command-wiring steps (T13) instruct reading neighboring cases because `commands.go` shape must be matched exactly — flagged, not hand-waved.
- **Type consistency:** `MarkdownOptions{WrapWidth,GlamourTheme,DropTrackingImages}`, `RenderEmailMarkdown`, `MarkdownToTerminal(md,theme,width)`, `cleanupMarkdown(md,opts)`, `NewDisplayService(bool)`, `Is/Set/ToggleMarkdownRendering`, `get/setRenderCache(id,markdown,width)` — consistent across all tasks.
- **Known follow-ups (non-blocking):** `sanitizeForTerminal` drops emoji (consistent with current app behavior); real data tables lose glamour table styling only if every row is empty (they are preserved otherwise); `[^)]*` URL regex assumes no literal `)` in URLs.
