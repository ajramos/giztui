package tui

import (
	"reflect"
	"testing"

	"github.com/ajramos/giztui/internal/config"
)

func TestCommandCandidates_Names(t *testing.T) {
	a := &App{}

	// "arch" matches canonical names "archive" and "archived" (sorted: "archive" < "archived").
	if got := a.commandCandidates("arch"); !reflect.DeepEqual(got, []string{"archive", "archived"}) {
		t.Fatalf("arch -> %v, want [archive archived]", got)
	}

	// Alias maps to its canonical name: "a" (an alias of archive) must include "archive".
	foundArchive := false
	for _, c := range a.commandCandidates("a") {
		if c == "archive" {
			foundArchive = true
		}
	}
	if !foundArchive {
		t.Fatalf("prefix 'a' should include canonical 'archive'")
	}

	// Unique match completes fully: only "attachments" (name) / "attach" (alias) start with "atta".
	if got := a.commandCandidates("atta"); !reflect.DeepEqual(got, []string{"attachments"}) {
		t.Fatalf("atta -> %v, want [attachments]", got)
	}

	// No match -> nil.
	if got := a.commandCandidates("zzzzz"); got != nil {
		t.Fatalf("zzzzz -> %v, want nil", got)
	}

	// Blank -> nil.
	if got := a.commandCandidates("   "); got != nil {
		t.Fatalf("blank -> %v, want nil", got)
	}
}

// Drift guard: every command in the registry has a non-empty canonical name and no duplicate names.
func TestCommandRegistry_NoDuplicateNames(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range commandRegistry {
		if s.name == "" {
			t.Fatal("registry has an entry with empty name")
		}
		if seen[s.name] {
			t.Fatalf("duplicate registry name: %q", s.name)
		}
		seen[s.name] = true
	}
}

func TestCompleteLabelsArg(t *testing.T) {
	a := &App{}
	a.cmd.labelNames = []string{"Work", "Personal", "Worklog"}

	// First token = subcommand.
	if got := completeLabelsArg(a, ""); len(got) != 3 || got[0] != "add" || got[1] != "list" || got[2] != "remove" {
		t.Fatalf("'' -> %v, want [add list remove]", got)
	}
	if got := completeLabelsArg(a, "re"); len(got) != 1 || got[0] != "remove" {
		t.Fatalf("'re' -> %v, want [remove]", got)
	}
	// After add/remove → a label name.
	if got := completeLabelsArg(a, "add wor"); len(got) != 2 || got[0] != "add Work" || got[1] != "add Worklog" {
		t.Fatalf("'add wor' -> %v, want [add Work add Worklog]", got)
	}
	// list takes no name.
	if got := completeLabelsArg(a, "list x"); got != nil {
		t.Fatalf("'list x' -> %v, want nil", got)
	}
}

func TestCommandCandidates_ArgGrammar(t *testing.T) {
	a := &App{}
	a.cmd.labelNames = []string{"Work", "Personal"}
	a.cmd.themeNames = []string{"gmail-dark", "gruvbox"}
	a.cmd.queryNames = []string{"Unread VIP", "Receipts"}

	cases := []struct {
		in   string
		want []string
	}{
		{"labels ", []string{"labels add", "labels list", "labels remove"}},
		{"labels add wor", []string{"labels add Work"}},
		{"prompt li", []string{"prompt list"}},
		{"theme ", []string{"theme list", "theme preview", "theme set"}},
		{"theme set gr", []string{"theme set gruvbox"}},
		{"bookmark Unread V", []string{"bookmark Unread VIP"}},
		{"search ha", []string{"search has:attachment"}},
		{"search from:x ha", []string{"search from:x has:attachment"}},
		// Commands whose argument is a number / message-id have NO completer:
		{"move 3", nil},
		{"label 3", nil},
		{"slack 5", nil},
		{"archive x", nil},
	}
	for _, c := range cases {
		got := a.commandCandidates(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("commandCandidates(%q) -> %v, want %v", c.in, got, c.want)
		}
	}
}

func TestArgCompleters_Subcommands(t *testing.T) {
	a := &App{}
	// prompt is a subcommand dispatcher (no name completion).
	if got := completePromptArg(a, ""); len(got) != 6 {
		t.Fatalf("prompt '' -> %v, want 6 subcommands", got)
	}
	if got := completePromptArg(a, "list x"); got != nil {
		t.Fatalf("prompt 'list x' -> %v, want nil", got)
	}
	// theme set <name>
	a.cmd.themeNames = []string{"gmail-dark", "gruvbox"}
	if got := completeThemeArg(a, "set gm"); len(got) != 1 || got[0] != "set gmail-dark" {
		t.Fatalf("theme 'set gm' -> %v, want [set gmail-dark]", got)
	}
	// accounts switch <id>
	a.Config = &config.Config{}
	a.Config.Accounts = []config.AccountConfig{{ID: "personal"}, {ID: "work"}}
	if got := completeAccountsArg(a, ""); len(got) != 1 || got[0] != "switch" {
		t.Fatalf("accounts '' -> %v, want [switch]", got)
	}
	if got := completeAccountsArg(a, "switch w"); len(got) != 1 || got[0] != "switch work" {
		t.Fatalf("accounts 'switch w' -> %v, want [switch work]", got)
	}
}

func TestArgCompleters_Wired(t *testing.T) {
	for _, name := range []string{"search", "labels", "prompt", "theme", "bookmark", "accounts"} {
		if s := lookupCommand(name); s == nil || s.completeArg == nil {
			t.Fatalf("command %q should have an arg completer", name)
		}
	}
	// These take a number / message id — they must NOT have a completer.
	for _, name := range []string{"slack", "label", "move"} {
		if s := lookupCommand(name); s == nil || s.completeArg != nil {
			t.Fatalf("command %q must NOT have an arg completer (takes a number)", name)
		}
	}
}
