package render

import (
	"testing"
)

// Glamour renders tables and thematic breaks with box-drawing characters
// (│ ─ ┼ …, U+2500–U+257F). Those are East-Asian-Width "Ambiguous": tcell may
// render them double-width, so a width-filling rule overflows the reader pane
// and the boundary glyph is clipped to U+FFFD (the "??" the user saw). Mapping
// them to width-1 ASCII keeps glamour's and tcell's width math in agreement.
func TestMarkdownToTerminal_NoBoxDrawing(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |\n\n* * *\n\nDone\n"
	out, err := MarkdownToTerminal(md, "dark", 80)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	for _, r := range out {
		if r >= 0x2500 && r <= 0x257F {
			t.Errorf("output still contains box-drawing rune U+%04X %q", r, string(r))
			break
		}
	}
}
