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
	md = collapseDuplicateHalves(md)
	md = sanitizeForTerminal(md)               // defined in format.go
	md = dedupeNearDuplicateParagraphs(md, 32) // defined in format.go
	return strings.TrimSpace(md)
}

// collapseDuplicateHalves collapses lines that are an exact phrase repeated twice
// (e.g. "Pide un Glovo [1] Pide un Glovo [1]"), which happens when a newsletter
// renders the same CTA as both a button-image link and a text link side by side.
// Only lines containing a link bracket "]" are considered, to avoid collapsing
// legitimately repeated prose.
func collapseDuplicateHalves(md string) string {
	lines := strings.Split(md, "\n")
	for i, ln := range lines {
		lines[i] = collapseLineDuplicate(ln)
	}
	return strings.Join(lines, "\n")
}

func collapseLineDuplicate(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, "]") {
		return line
	}
	fields := strings.Fields(trimmed)
	n := len(fields)
	if n < 2 || n%2 != 0 {
		return line
	}
	half := n / 2
	for k := 0; k < half; k++ {
		if fields[k] != fields[half+k] {
			return line
		}
	}
	// Halves are identical: keep one, preserving the original leading indent.
	lead := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	return lead + strings.Join(fields[:half], " ")
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
	result := stripTagBackgrounds(string(tview.TranslateANSI([]byte(out))))
	result = fixPaletteColorTags(result)
	result = fixLinkContrast(result)
	result = asciiBoxDrawing(result)
	return result, nil
}

// palette256TagRe matches the malformed tview color tag that tview.TranslateANSI emits
// for 256-palette foregrounds (the ones glamour's themes use for headings): a stray "-"
// glued to the 6-hex foreground, e.g. "[#0099ff-::b]". The dash makes the foreground
// token unparseable, so tview renders the whole tag as literal text in the reader pane.
var palette256TagRe = regexp.MustCompile(`(\[#[0-9a-fA-F]{6})-(:)`)

// fixPaletteColorTags strips that stray dash, turning "[#0099ff-::b]" back into the valid
// "[#0099ff::b]" so the color applies instead of leaking the raw tag into the message body.
func fixPaletteColorTags(s string) string {
	return palette256TagRe.ReplaceAllString(s, "$1$2")
}

// blackFgTagRe matches a tview color tag whose foreground is "black".
var blackFgTagRe = regexp.MustCompile(`\[black(:[^\]]*)\]`)

// fixLinkContrast drops the black foreground glamour's dark style assigns to link
// URLs. Black is effectively invisible on the dark reader pane; clearing the
// foreground lets the URL inherit the readable pane text color while keeping its
// underline, so it stays recognizable as a link.
func fixLinkContrast(s string) string {
	return blackFgTagRe.ReplaceAllString(s, "[$1]")
}

// asciiBoxDrawing replaces Unicode box-drawing characters (U+2500–U+257F) emitted
// by glamour for tables and thematic breaks with ASCII equivalents. Box-drawing
// glyphs are East-Asian-Width "Ambiguous", so tcell can render them double-width
// while glamour laid them out as single-width; a width-filling rule then overflows
// the reader pane and tcell clips the boundary glyph to U+FFFD (the "??"/"�" the
// user saw). ASCII |, - and + are unambiguously width 1, keeping both layers in
// agreement so the rule fits and renders cleanly.
func asciiBoxDrawing(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x2500 || r > 0x257F {
			return r
		}
		switch r {
		case '│', '┃', '║', '╎', '╏', '┆', '┇', '┊', '┋':
			return '|'
		case '─', '━', '═', '╌', '╍', '┄', '┅', '┈', '┉':
			return '-'
		default:
			// Corners, tees, crosses and arcs all become an ASCII junction.
			return '+'
		}
	}, s)
}

// colorTagRe matches a 3-field tview color tag [foreground:background:flags].
var colorTagRe = regexp.MustCompile(`\[([^\]:]*):([^\]:]*):([^\]]*)\]`)

// stripTagBackgrounds blanks the background field of tview color tags produced by
// TranslateANSI. Glamour assumes a fixed terminal background and styles code
// spans, rules and tables with their own background colors; left intact, those
// clash with the reader pane's themed background and render as non-homogeneous
// bars. Foreground colors and attributes are preserved, so only the background
// inherits the pane theme — keeping the message body visually uniform.
func stripTagBackgrounds(s string) string {
	return colorTagRe.ReplaceAllString(s, "[$1::$3]")
}
