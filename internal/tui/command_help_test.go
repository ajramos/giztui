package tui

import (
	"strings"
	"testing"
)

func TestGenerateCommandHelpText_Rich(t *testing.T) {
	s := &commandSpec{
		name:    "search",
		aliases: nil,
		help: &cmdHelp{
			summary:  "Search Gmail messages.",
			syntax:   ":search <query>",
			examples: []string{":search is:unread"},
		},
	}
	app := &App{}
	got := app.generateCommandHelpText(s)
	for _, want := range []string{":search", "Search Gmail messages.", "Syntax:", ":search <query>", "Examples:", ":search is:unread", "Esc"} {
		if !strings.Contains(got, want) {
			t.Errorf("rich help missing %q in:\n%s", want, got)
		}
	}
}

func TestGenerateCommandHelpText_Fallback(t *testing.T) {
	s := &commandSpec{name: "archive", aliases: []string{"a"}}
	app := &App{}
	got := app.generateCommandHelpText(s)
	if !strings.Contains(got, ":archive") || !strings.Contains(got, "a") {
		t.Errorf("fallback must name the command + aliases:\n%s", got)
	}
	if !strings.Contains(got, "No detailed help") {
		t.Errorf("fallback must state no detailed help:\n%s", got)
	}
	if strings.Contains(got, "Syntax:") {
		t.Errorf("fallback must NOT have a Syntax block:\n%s", got)
	}
}
