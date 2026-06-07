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
