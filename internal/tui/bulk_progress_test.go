package tui

import "testing"

func TestBulkProgressVerb(t *testing.T) {
	cases := map[string]string{
		"archive":     "Archiving",
		"trash":       "Trashing",
		"mark_read":   "Marking read",
		"mark_unread": "Marking unread",
		"label":       "Applying label",
		"something":   "Processing",
	}
	for action, want := range cases {
		if got := bulkProgressVerb(action); got != want {
			t.Errorf("bulkProgressVerb(%q) = %q, want %q", action, got, want)
		}
	}
}
