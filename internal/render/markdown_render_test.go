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

func TestReferenceLongURLs(t *testing.T) {
	longURL := "https://track.example.com/" + strings.Repeat("a", 80)
	in := "Click [Buy now](" + longURL + ") today. Short [ok](https://x.io).\n"
	got := referenceLongURLs(in, 60)

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
