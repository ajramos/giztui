package tui

import (
	"reflect"
	"testing"
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
