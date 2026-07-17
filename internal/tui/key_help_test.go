package tui

import (
	"strings"
	"testing"
)

func TestFormatKeyHelp(t *testing.T) {
	hints := []KeyHint{
		{Key: "g", Desc: "Assist"},
		{Key: "Ctrl+R", Desc: "Remember rule"},
		{Key: "Esc", Desc: "Close"},
	}
	out := formatKeyHelp("Action Plan", hints)

	// Title present on the first line.
	if !strings.HasPrefix(out, "Action Plan") {
		t.Fatalf("title missing; got:\n%s", out)
	}
	// Every key and description appears.
	for _, h := range hints {
		if !strings.Contains(out, h.Key) || !strings.Contains(out, h.Desc) {
			t.Fatalf("missing %q/%q in:\n%s", h.Key, h.Desc, out)
		}
	}
	// Keys are left-aligned to the same column: the descriptions must line up.
	// "Ctrl+R" is the widest key (6). Find each desc's byte offset on its line; all equal.
	var descCols []int
	for _, line := range strings.Split(out, "\n") {
		for _, h := range hints {
			if strings.Contains(line, h.Desc) {
				descCols = append(descCols, strings.Index(line, h.Desc))
			}
		}
	}
	for i := 1; i < len(descCols); i++ {
		if descCols[i] != descCols[0] {
			t.Fatalf("descriptions not column-aligned: %v\n%s", descCols, out)
		}
	}
}

func TestFormatKeyHelp_EmptyHints(t *testing.T) {
	out := formatKeyHelp("Empty", nil)
	if strings.TrimSpace(out) != "Empty" {
		t.Fatalf("empty hints should render just the title, got %q", out)
	}
}
