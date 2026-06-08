package render

import (
	"regexp"
	"testing"
)

// Glamour styles code spans, tables and rules with ANSI background colors, which
// TranslateANSI faithfully turns into tview [fg:bg:attr] tags. Those backgrounds
// clash with the reader pane's themed background, producing the non-homogeneous
// bars the user reported. MarkdownToTerminal must emit no background-setting tag.
func TestMarkdownToTerminal_StripsBackgrounds(t *testing.T) {
	md := "Use `inline code` here\n\n* * *\n\n| Status | Count |\n|--------|-------|\n| On track | 4 |\n"
	out, err := MarkdownToTerminal(md, "dark", 80)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// A tview color tag with a non-empty middle field sets a background.
	bgTag := regexp.MustCompile(`\[[^\]:]*:[^\]:]+:[^\]]*\]`)
	if m := bgTag.FindString(out); m != "" {
		t.Errorf("rendered output still sets a background tag: %q", m)
	}
}
