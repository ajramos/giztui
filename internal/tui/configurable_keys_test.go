package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/config"
)

// goldenConfigurableBindingOrder is the exact, ordered list of configurable
// key bindings as they existed when handleConfigurableKey was a hand-written
// switch statement. The switch matched top-to-bottom (first non-empty binding
// whose key equals the pressed rune wins), so ORDER IS BEHAVIOUR: two keys
// configured to the same rune resolve to whichever appears first here.
//
// This is a ratchet/characterization snapshot for the switch→table refactor:
// if a mechanical edit drops, renames, reorders or duplicates a binding, this
// test fails. It intentionally does NOT execute handlers (they have side
// effects) — it verifies the shape of the mapping, not the actions.
var goldenConfigurableBindingOrder = []string{
	"summarize",
	"force_regenerate_summary",
	"generate_reply",
	"suggest_label",
	"reply",
	"reply_all",
	"forward",
	"compose",
	"refresh",
	"autorefresh",
	"speak",
	"search",
	"unread",
	"archived",
	"search_from",
	"search_to",
	"search_subject",
	"toggle_read",
	"trash",
	"archive",
	"drafts",
	"attachments",
	"move",
	"manage_labels",
	"quit",
	"obsidian",
	"slack",
	"markdown",
	"save_message",
	"save_raw",
	"rsvp",
	"ai_jobs",
	"chat",
	"link_picker",
	"theme_picker",
	"open_gmail",
	"bulk_mode",
	"command_mode",
	"help",
	"load_more",
	"toggle_headers",
	"toggle_threading",
	"expand_all_threads",
	"collapse_all_threads",
	"bulk_select",
	"action_plan",
	"save_query",
	"query_bookmarks",
	"undo",
}

func TestConfigurableBindings_OrderSnapshot(t *testing.T) {
	a := &App{}
	got := a.configurableBindings()

	if len(got) != len(goldenConfigurableBindingOrder) {
		t.Fatalf("binding count = %d, want %d (a binding was added or removed — update the golden snapshot deliberately, not by reflex)", len(got), len(goldenConfigurableBindingOrder))
	}
	for i, want := range goldenConfigurableBindingOrder {
		if got[i].name != want {
			t.Errorf("binding[%d] = %q, want %q (order is behaviour: first match wins)", i, got[i].name, want)
		}
	}
}

func TestConfigurableBindings_Invariants(t *testing.T) {
	a := &App{}
	seen := map[string]bool{}
	for i, b := range a.configurableBindings() {
		if b.name == "" {
			t.Errorf("binding[%d] has empty name", i)
		}
		if b.key == nil {
			t.Errorf("binding[%d] (%s) has nil key pointer", i, b.name)
		}
		if b.handler == nil {
			t.Errorf("binding[%d] (%s) has nil handler", i, b.name)
		}
		if seen[b.name] {
			t.Errorf("duplicate binding name %q", b.name)
		}
		seen[b.name] = true
	}
}

// TestConfigurableBindings_Wiring pins the name→config-field wiring for the
// cases most prone to a copy/paste slip in a mechanical switch→table move:
// the bulk-aware operations and compose. Each field is set to a unique
// sentinel and the matching binding must point at it.
func TestConfigurableBindings_Wiring(t *testing.T) {
	a := &App{Keys: config.KeyBindings{
		Trash:        "\x01trash",
		Archive:      "\x01archive",
		Move:         "\x01move",
		ManageLabels: "\x01labels",
		ToggleRead:   "\x01read",
		Compose:      "\x01compose",
		Slack:        "\x01slack",
	}}
	want := map[string]string{
		"trash":         "\x01trash",
		"archive":       "\x01archive",
		"move":          "\x01move",
		"manage_labels": "\x01labels",
		"toggle_read":   "\x01read",
		"compose":       "\x01compose",
		"slack":         "\x01slack",
	}
	byName := map[string]*string{}
	for _, b := range a.configurableBindings() {
		byName[b.name] = b.key
	}
	for name, sentinel := range want {
		p, ok := byName[name]
		if !ok {
			t.Errorf("binding %q missing", name)
			continue
		}
		if p == nil || *p != sentinel {
			t.Errorf("binding %q wired to wrong field: got %q, want %q", name, deref(p), sentinel)
		}
	}
}

func deref(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
