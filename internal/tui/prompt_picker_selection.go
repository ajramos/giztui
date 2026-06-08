package tui

// promptPickerSelection maps a prompt picker list's current-item index to a
// selection. Row 0 is the "✨ Create new with AI..." entry; rows 1..N map to
// visible[index-1]. It returns isCreateNew=true for the create-new entry (and a
// visibleIndex of -1), otherwise the 0-based index into the visible-prompts slice.
//
// A negative or out-of-range index is treated as create-new so a stale/invalid
// selection never silently applies the wrong prompt.
func promptPickerSelection(currentItem, visibleCount int) (isCreateNew bool, visibleIndex int) {
	if currentItem <= 0 {
		return true, -1
	}
	vi := currentItem - 1
	if vi >= visibleCount {
		return true, -1
	}
	return false, vi
}
