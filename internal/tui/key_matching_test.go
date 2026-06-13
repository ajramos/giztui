package tui

import (
	"testing"

	"github.com/derailed/tcell/v2"
)

// rune event helper
func runeEvent(r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, r, mod)
}

func TestMatchesKeyCombo_Shift(t *testing.T) {
	a := &App{}

	if !a.matchesKeyCombo(runeEvent('T', tcell.ModShift), "shift+t") {
		t.Error("shift+t should match Shift+T (uppercase rune)")
	}
	if a.matchesKeyCombo(runeEvent('t', tcell.ModNone), "shift+t") {
		t.Error("shift+t must NOT match a plain lowercase t")
	}
	if a.matchesKeyCombo(runeEvent('T', tcell.ModShift), "shift+") {
		t.Error("malformed shift+ (no letter) must not match")
	}
}

func TestMatchesConfiguredKey_CaseSensitivePlainKeys(t *testing.T) {
	a := &App{}

	// Plain single-character bindings must be case-sensitive (unlike matchesKeyCombo,
	// which lowercases). This is why search_prev "N" and search_next "n" stay distinct.
	if !a.matchesConfiguredKey(runeEvent('N', tcell.ModNone), "N") {
		t.Error(`"N" should match an uppercase N`)
	}
	if a.matchesConfiguredKey(runeEvent('n', tcell.ModNone), "N") {
		t.Error(`"N" must NOT match a lowercase n`)
	}
	if !a.matchesConfiguredKey(runeEvent('i', tcell.ModNone), "i") {
		t.Error(`"i" should match i`)
	}
}

func TestMatchesConfiguredKey_SpaceToken(t *testing.T) {
	a := &App{}

	if !a.matchesConfiguredKey(runeEvent(' ', tcell.ModNone), "space") {
		t.Error(`"space" should match the space rune`)
	}
	if a.matchesConfiguredKey(runeEvent('x', tcell.ModNone), "space") {
		t.Error(`"space" must not match a non-space rune`)
	}
}

func TestMatchesConfiguredKey_Empty(t *testing.T) {
	a := &App{}
	if a.matchesConfiguredKey(runeEvent('a', tcell.ModNone), "") {
		t.Error("an empty binding must never match")
	}
}
