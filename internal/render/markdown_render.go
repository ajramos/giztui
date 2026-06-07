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
