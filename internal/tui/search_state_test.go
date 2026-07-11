package tui

import (
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func metas(ids ...string) []*gmailapi.Message {
	out := make([]*gmailapi.Message, len(ids))
	for i, id := range ids {
		out[i] = &gmailapi.Message{Id: id}
	}
	return out
}

func TestSearchState_SnapshotCopyIndependence(t *testing.T) {
	var s searchState
	ids := []string{"a", "b", "c"}
	s.captureSnapshot(ids, metas("a", "b", "c"), "tok", "b")

	ids[0] = "MUT"
	gotIDs, gotMeta, tok, sel := s.snapshot()
	if gotIDs[0] != "a" || tok != "tok" || sel != "b" || len(gotMeta) != 3 {
		t.Fatalf("snapshot not isolated from source: %v tok=%q sel=%q", gotIDs, tok, sel)
	}
	gotIDs[1] = "MUT"
	again, _, _, _ := s.snapshot()
	if again[1] != "b" {
		t.Fatalf("snapshot not isolated from returned copy: %v", again)
	}
}

func TestSearchState_RemoveFromSnapshotByID(t *testing.T) {
	var s searchState
	s.captureSnapshot([]string{"a", "b", "c"}, metas("a", "b", "c"), "", "")
	s.removeFromSnapshotByID("b")
	ids, meta, _, _ := s.snapshot()
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "c" {
		t.Fatalf("ids = %v, want [a c]", ids)
	}
	if len(meta) != 2 || meta[0].Id != "a" || meta[1].Id != "c" {
		t.Fatalf("meta misaligned: %v", meta)
	}
	s.removeFromSnapshotByID("zzz")
	if ids2, _, _, _ := s.snapshot(); len(ids2) != 2 {
		t.Fatalf("missing id should be a no-op, got %v", ids2)
	}
}

func TestSearchState_RemoveFromSnapshotByIDs(t *testing.T) {
	var s searchState
	s.captureSnapshot([]string{"a", "b", "c", "d"}, metas("a", "b", "c", "d"), "", "")
	s.removeFromSnapshotByIDs([]string{"b", "d"})
	ids, meta, _, _ := s.snapshot()
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "c" {
		t.Fatalf("ids = %v, want [a c]", ids)
	}
	if meta[0].Id != "a" || meta[1].Id != "c" {
		t.Fatalf("meta misaligned: %v", meta)
	}
}

func TestSearchState_Accessors(t *testing.T) {
	var s searchState
	s.SetMode("remote")
	s.SetQuery("is:unread")
	s.SetOriginal("unread stuff")
	s.localFilter = "foo"
	if s.Mode() != "remote" || s.Query() != "is:unread" || s.localFilter != "foo" {
		t.Fatalf("accessors: mode=%q query=%q filter=%q", s.Mode(), s.Query(), s.localFilter)
	}
	if s.Original() != "unread stuff" {
		t.Fatalf("Original() = %q, want %q", s.Original(), "unread stuff")
	}
	s.clear()
	if s.Mode() != "" || s.Query() != "" || s.Original() != "" || s.localFilter != "" {
		t.Fatalf("clear left state: mode=%q query=%q original=%q filter=%q", s.Mode(), s.Query(), s.Original(), s.localFilter)
	}
}

// activeSearchPrefill feeds both the ctrl+s shortcut and :rules new — it must
// return the user-typed query only for a remote search (local filters are not
// Gmail queries and can't seed a rule verbatim).
func TestActiveSearchPrefill(t *testing.T) {
	a := &App{}

	if got := a.activeSearchPrefill(); got != "" {
		t.Fatalf("no search: prefill = %q, want empty", got)
	}

	a.search.SetMode("remote")
	a.search.SetQuery("github -in:sent -in:draft -in:chat -in:spam -in:trash in:inbox")
	a.search.SetOriginal("github")
	if got := a.activeSearchPrefill(); got != "github" {
		t.Fatalf("remote search: prefill = %q, want the user-typed query", got)
	}

	// Older code paths may set only the effective query — fall back to it.
	a.search.SetOriginal("")
	if got := a.activeSearchPrefill(); got == "" {
		t.Fatal("remote search without original: want fallback to effective query, got empty")
	}

	a.search.clear()
	a.search.SetMode("local")
	a.search.SetQuery("foo")
	if got := a.activeSearchPrefill(); got != "" {
		t.Fatalf("local filter: prefill = %q, want empty", got)
	}
}
