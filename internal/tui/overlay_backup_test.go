package tui

import "testing"

func TestOverlayBackup_ActiveAndClear(t *testing.T) {
	var b overlayBackup
	// Zero value is inactive.
	if b.active() {
		t.Fatal("zero overlayBackup must be inactive")
	}
	// A non-empty text marks it active (the restore guard).
	b.text = "body"
	b.header = "hdr"
	b.title = "ttl"
	if !b.active() {
		t.Fatal("backup with non-empty text must be active")
	}
	// clear resets all three fields and makes it inactive.
	b.clear()
	if b.text != "" || b.header != "" || b.title != "" {
		t.Fatalf("clear left state: %q %q %q", b.text, b.header, b.title)
	}
	if b.active() {
		t.Fatal("cleared backup must be inactive")
	}
	// active keys off text specifically (header/title alone do not count).
	var b2 overlayBackup
	b2.header = "hdr"
	if b2.active() {
		t.Fatal("active must key off text, not header")
	}
}
