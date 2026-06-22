package config

import (
	"strings"
	"testing"
)

// The default key bindings intentionally reuse a few keys across mutually-exclusive UI
// contexts (e.g. 'a' = archive in the list / rule_add in the action plan). Those are
// allowlisted and must NOT produce duplicate-key warnings.
func TestValidateKeyboardConfig_ContextSeparatedNotWarned(t *testing.T) {
	keys := DefaultConfig().Keys
	for _, w := range ValidateKeyboardConfig(keys) {
		if strings.Contains(w, "assigned to multiple") {
			t.Errorf("default keys must not warn about context-separated duplicates, got: %q", w)
		}
	}
}

// A genuinely new same-context collision (a pair NOT in the allowlist) must still be reported,
// so the allowlist suppresses only the known-safe overlaps.
func TestValidateKeyboardConfig_RealCollisionWarned(t *testing.T) {
	keys := DefaultConfig().Keys
	keys.Archive = "ctrl+x"
	keys.Compose = "ctrl+x" // archive+compose on ctrl+x is not allowlisted → real conflict

	found := false
	for _, w := range ValidateKeyboardConfig(keys) {
		if strings.Contains(w, "Key 'ctrl+x'") && strings.Contains(w, "assigned to multiple") {
			found = true
		}
	}
	if !found {
		t.Error("a new same-context collision (ctrl+x) should still be warned")
	}
}
