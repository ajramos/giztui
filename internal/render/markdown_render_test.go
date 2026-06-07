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
