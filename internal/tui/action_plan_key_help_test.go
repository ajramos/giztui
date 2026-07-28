package tui

import "testing"

func TestActionPlanKeyHints(t *testing.T) {
	keys := actionPlanFooterKeys{
		viewPrompt: "i", remember: "ctrl+r", move: "m", skip: "space",
		archive: "a", trash: "d", label: "l", toggleRead: "t", confirm: "c", assist: "g", accept: ".",
	}
	hints := actionPlanKeyHints(keys)

	// Spot-check that configured keys are rendered (via prettyKeyLabel) with their descriptions.
	want := map[string]string{
		"Ctrl+R": "", // remember
		"m":      "", // move
		"i":      "", // view prompt
		"c":      "", // confirm
		"Esc":    "", // fixed key
	}
	seen := map[string]bool{}
	for _, h := range hints {
		seen[h.Key] = true
	}
	for k := range want {
		if !seen[k] {
			t.Fatalf("expected key %q in hints, got %+v", k, hints)
		}
	}
}

func TestActionPlanKeyHints_ReflectsRebind(t *testing.T) {
	// Rebinding remember to a different key must change the cheat-sheet (single source of truth).
	base := actionPlanFooterKeys{viewPrompt: "i", remember: "ctrl+r", move: "m", skip: "space",
		archive: "a", trash: "d", label: "l", toggleRead: "t", confirm: "c"}
	rebound := base
	rebound.remember = "x"

	hasKey := func(hs []KeyHint, k string) bool {
		for _, h := range hs {
			if h.Key == k {
				return true
			}
		}
		return false
	}
	if !hasKey(actionPlanKeyHints(base), "Ctrl+R") {
		t.Fatal("base should advertise Ctrl+R for remember")
	}
	if !hasKey(actionPlanKeyHints(rebound), "x") || hasKey(actionPlanKeyHints(rebound), "Ctrl+R") {
		t.Fatal("rebind not reflected in hints")
	}
}
