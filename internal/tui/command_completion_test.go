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
