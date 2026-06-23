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

func TestCompleteLabelArg(t *testing.T) {
	a := &App{}
	a.cmd.labelNames = []string{"Work", "Personal", "Worklog", "travel"}

	// Case-insensitive prefix, sorted.
	got := completeLabelArg(a, "wor")
	want := []string{"Work", "Worklog"}
	if len(got) != 2 || got[0] != "Work" || got[1] != "Worklog" {
		t.Fatalf("wor -> %v, want %v", got, want)
	}

	// Empty prefix returns all (sorted, case-insensitive).
	all := completeLabelArg(a, "")
	if len(all) != 4 {
		t.Fatalf("empty prefix -> %d candidates, want 4", len(all))
	}

	// No match -> nil.
	if got := completeLabelArg(a, "zzz"); got != nil {
		t.Fatalf("zzz -> %v, want nil", got)
	}
}

func TestCommandCandidates_LabelArg(t *testing.T) {
	a := &App{}
	a.cmd.labelNames = []string{"Work", "Personal"}

	// "labels add wor" -> "labels add Work".
	got := a.commandCandidates("labels add wor")
	if len(got) != 1 || got[0] != "labels add Work" {
		t.Fatalf("labels add wor -> %v, want [labels add Work]", got)
	}

	// Command without an arg completer yields nil in arg position.
	if got := a.commandCandidates("archive x"); got != nil {
		t.Fatalf("archive x -> %v, want nil", got)
	}
}

func TestArgCompleters_Dynamic(t *testing.T) {
	a := &App{}
	a.cmd.promptNames = []string{"Summarize", "Translate", "Sentiment"}
	if got := completePromptArg(a, "s"); len(got) != 2 || got[0] != "Sentiment" || got[1] != "Summarize" {
		t.Fatalf("prompt 's' -> %v, want [Sentiment Summarize]", got)
	}
	a.cmd.themeNames = []string{"gmail-dark", "gruvbox", "dracula"}
	if got := completeThemeArg(a, "gr"); len(got) != 1 || got[0] != "gruvbox" {
		t.Fatalf("theme 'gr' -> %v, want [gruvbox]", got)
	}
	a.cmd.queryNames = []string{"Unread VIP", "Receipts"}
	if got := completeQueryArg(a, "rec"); len(got) != 1 || got[0] != "Receipts" {
		t.Fatalf("query 'rec' -> %v, want [Receipts]", got)
	}
}

func TestArgCompleters_Static(t *testing.T) {
	a := &App{}
	if got := completeSearchArg(a, "ha"); len(got) != 1 || got[0] != "has:attachment" {
		t.Fatalf("search 'ha' -> %v, want [has:attachment]", got)
	}
	if got := completeSearchArg(a, "is:"); len(got) < 3 {
		t.Fatalf("search 'is:' -> %v, want several", got)
	}

	a.Config = &config.Config{}
	a.Config.Slack.Channels = []config.SlackChannel{{Name: "team-updates"}, {Name: "random"}}
	if got := completeSlackArg(a, "te"); len(got) != 1 || got[0] != "team-updates" {
		t.Fatalf("slack 'te' -> %v, want [team-updates]", got)
	}
	a.Config.Accounts = []config.AccountConfig{{ID: "personal"}, {ID: "work"}}
	if got := completeAccountArg(a, "w"); len(got) != 1 || got[0] != "work" {
		t.Fatalf("account 'w' -> %v, want [work]", got)
	}
}

func TestArgCompleters_Wired(t *testing.T) {
	for _, name := range []string{"search", "slack", "prompt", "theme", "bookmark", "accounts", "labels", "label", "move"} {
		if s := lookupCommand(name); s == nil || s.completeArg == nil {
			t.Fatalf("command %q should have an arg completer", name)
		}
	}
}
