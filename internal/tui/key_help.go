package tui

import (
	"strings"
	"unicode/utf8"
)

// KeyHint is one row of a context cheat-sheet: a display-formatted key and what it does.
type KeyHint struct {
	Key  string // already display-formatted, e.g. "Ctrl+R", "Esc", "g"
	Desc string
}

// formatKeyHelp renders a cheat-sheet: the title, a blank line, then one "key  description"
// row per hint with keys left-padded to a common width so descriptions align. An empty hint
// list renders just the title (defensive — for panels that declare none yet).
func formatKeyHelp(title string, hints []KeyHint) string {
	if len(hints) == 0 {
		return title
	}
	width := 0
	for _, h := range hints {
		if n := utf8.RuneCountInString(h.Key); n > width {
			width = n
		}
	}
	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	for _, h := range hints {
		b.WriteString("  ")
		b.WriteString(h.Key)
		b.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(h.Key)))
		b.WriteString("   ")
		b.WriteString(h.Desc)
		b.WriteString("\n")
	}
	return b.String()
}
