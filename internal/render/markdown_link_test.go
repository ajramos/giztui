package render

import (
	"strings"
	"testing"
)

// Glamour's dark style paints link URLs with a black foreground, which is
// effectively invisible on the dark reader pane. fixLinkContrast must drop that
// black foreground so the URL inherits the readable pane text color (it stays
// underlined, so it's still recognizable as a link).
func TestMarkdownToTerminal_LinkNotBlack(t *testing.T) {
	md := "See [our site](https://example.com/some/long/path) now\n"
	out, err := MarkdownToTerminal(md, "dark", 80)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if strings.Contains(out, "[black:") {
		t.Errorf("link URL still rendered with black foreground: %q", out)
	}
}
