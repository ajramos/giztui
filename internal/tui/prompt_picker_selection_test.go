package tui

import "testing"

// The prompt pickers (single and bulk) render "✨ Create new with AI..." as row 0,
// followed by the visible prompts at rows 1..N. promptPickerSelection maps the
// list's current-item index to either the create-new entry or a 0-based index
// into the visible-prompts slice — the single source of truth shared by the
// per-item closures and the Enter-in-search-field handler.
func TestPromptPickerSelection(t *testing.T) {
	// Row 0 is the "Create new with AI" entry.
	if isCreateNew, vi := promptPickerSelection(0, 5); !isCreateNew || vi != -1 {
		t.Errorf("index 0 => (%v, %d), want (true, -1)", isCreateNew, vi)
	}
	// Row 1 maps to visible[0] (the first real prompt).
	if isCreateNew, vi := promptPickerSelection(1, 5); isCreateNew || vi != 0 {
		t.Errorf("index 1 => (%v, %d), want (false, 0)", isCreateNew, vi)
	}
	// Row 3 maps to visible[2].
	if isCreateNew, vi := promptPickerSelection(3, 5); isCreateNew || vi != 2 {
		t.Errorf("index 3 => (%v, %d), want (false, 2)", isCreateNew, vi)
	}
	// Negative/no selection is treated as create-new (safe default, never applies a prompt).
	if isCreateNew, vi := promptPickerSelection(-1, 5); !isCreateNew || vi != -1 {
		t.Errorf("index -1 => (%v, %d), want (true, -1)", isCreateNew, vi)
	}
	// Out-of-range index does not map to a prompt (guards against stale selection).
	if isCreateNew, _ := promptPickerSelection(6, 5); !isCreateNew {
		t.Errorf("index 6 with 5 visible => isCreateNew=%v, want true", isCreateNew)
	}
}
