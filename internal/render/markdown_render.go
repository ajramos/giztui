package render

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/charmbracelet/glamour"
	"github.com/derailed/tview"

	gmailwrap "github.com/ajramos/giztui/internal/gmail"
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

// mdLinkRe matches a Markdown inline link [text](http(s)://url).
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

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
	md = sanitizeForTerminal(md)               // defined in format.go
	md = dedupeNearDuplicateParagraphs(md, 32) // defined in format.go
	return strings.TrimSpace(md)
}

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
