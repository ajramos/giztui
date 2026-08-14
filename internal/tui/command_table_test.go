package tui

import (
	"reflect"
	"sort"
	"testing"
)

// goldenCommandTokens is the exact set of command tokens (canonical names and
// every alias) recognised by executeCommand when it was a hand-written switch.
// It is a characterization/ratchet snapshot for the switch→map refactor: if a
// mechanical edit drops, renames or adds a token, this test fails so the change
// is deliberate rather than accidental. It does NOT execute handlers.
//
// NOTE: "G" and "U" are dead aliases — executeCommand lowercases the command
// before dispatch, so an uppercase token can never reach the table. They are
// preserved verbatim to keep the refactor a pure transform; removing them would
// be a separate, behaviour-neutral cleanup.
var goldenCommandTokens = []string{
	"/", "?", "G", "U", "a", "acc", "accounts", "action-plan", "aijobs", "ap",
	"arch-search", "archive", "archived", "arr", "attach", "attachments",
	"autorefresh", "b", "bm", "bookmark", "bookmarks", "c", "cache", "cfg",
	"chat", "collapse", "collapse-all", "compose", "config", "d", "dr",
	"drafts", "expand", "expand-all", "f", "flat", "flatten", "forward", "g",
	"gmail", "h", "headers", "help", "i", "inbox", "jobs", "l", "label",
	"labels", "lbl", "link", "links", "load", "markdown", "md", "more", "move",
	"mv", "n", "new", "next", "numbers", "o", "obs", "obsidian", "open-web",
	"p", "pl", "plan", "pn", "pr", "preload", "prf", "prompt", "prompt-new",
	"prompt-refine", "prompt-save", "ps", "q", "qb", "queries", "query", "quit",
	"r", "ra", "read", "refresh", "reply", "reply-all", "rp", "rsvp", "ru",
	"rules", "s", "save", "save-query", "search", "sel", "select", "sl",
	"slack", "sq", "st", "star", "stats", "summary", "t", "th", "th-sum",
	"theme", "thr", "thread-summary", "threads", "toggle-headers", "toggle-read",
	"touch-up", "touchup", "trash", "u", "undo", "unread", "unst", "unstar",
	"usage", "web",
}

func TestCommandTable_TokenSnapshot(t *testing.T) {
	a := &App{}
	got := make([]string, 0)
	for k := range a.commandTable() {
		got = append(got, k)
	}
	sort.Strings(got)

	want := append([]string(nil), goldenCommandTokens...)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("command token count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCommandTable_NoNilHandlers(t *testing.T) {
	a := &App{}
	for tok, h := range a.commandTable() {
		if h == nil {
			t.Errorf("command %q has a nil handler", tok)
		}
	}
}

// TestCommandTable_AliasGrouping pins that every alias of a command still
// resolves to the SAME handler — the exact thing a copy/paste slip in a
// switch→map move would break (an alias silently pointing at the wrong action).
// Handlers are compared by code pointer, so no action is executed.
func TestCommandTable_AliasGrouping(t *testing.T) {
	a := &App{}
	table := a.commandTable()
	groups := [][]string{
		{"labels", "l"},
		{"archive", "a"},
		{"trash", "d"},
		{"move", "mv"},
		{"reply", "r"},
		{"reply-all", "ra"},
		{"forward", "f"},
		{"prompt", "pr", "p"},
		{"read", "toggle-read", "t"},
		{"gmail", "web", "open-web", "o"},
		{"bookmarks", "queries", "bm", "qb"},
		{"jobs", "aijobs"},
		{"markdown", "md"},
		{"touch-up", "touchup"},
		{"archived", "arch-search", "b"},
	}
	for _, g := range groups {
		var want uintptr
		for i, tok := range g {
			h, ok := table[tok]
			if !ok {
				t.Errorf("group %v: token %q missing", g, tok)
				continue
			}
			p := reflect.ValueOf(h).Pointer()
			if i == 0 {
				want = p
			} else if p != want {
				t.Errorf("group %v: alias %q resolves to a different handler than %q", g, tok, g[0])
			}
		}
	}
}
