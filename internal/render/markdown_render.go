package render

import (
	"regexp"
	"strings"

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
