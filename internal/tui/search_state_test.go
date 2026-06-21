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
	s.localFilter = "foo"
	if s.Mode() != "remote" || s.Query() != "is:unread" || s.localFilter != "foo" {
		t.Fatalf("accessors: mode=%q query=%q filter=%q", s.Mode(), s.Query(), s.localFilter)
	}
	s.clear()
	if s.Mode() != "" || s.Query() != "" || s.localFilter != "" {
		t.Fatalf("clear left state: mode=%q query=%q filter=%q", s.Mode(), s.Query(), s.localFilter)
	}
}
