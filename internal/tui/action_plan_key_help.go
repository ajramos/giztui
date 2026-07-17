package tui

// actionPlanKeyHints builds the full ordered cheat-sheet for the Action Plan panel from the
// same actionPlanFooterKeys the footer uses, so footer teaser and cheat-sheet never drift.
// Fixed tview keys (arrows/Enter/Tab/Esc) are literals; configured keys use prettyKeyLabel.
func actionPlanKeyHints(keys actionPlanFooterKeys) []KeyHint {
	return []KeyHint{
		{Key: "↑/↓", Desc: "Move between nodes"},
		{Key: "Enter/→", Desc: "Expand category / open email"},
		{Key: "←", Desc: "Collapse category"},
		{Key: prettyKeyLabel(keys.skip), Desc: "Exclude / include email"},
		{Key: prettyKeyLabel(keys.archive), Desc: "Archive the category's checked emails"},
		{Key: prettyKeyLabel(keys.trash), Desc: "Trash the category's checked emails"},
		{Key: prettyKeyLabel(keys.label), Desc: "Apply the category's label"},
		{Key: prettyKeyLabel(keys.toggleRead), Desc: "Mark the category's checked emails read"},
		{Key: prettyKeyLabel(keys.move), Desc: "Move email / category to another label"},
		{Key: prettyKeyLabel(keys.viewPrompt), Desc: "View the effective analyzer prompt"},
		{Key: prettyKeyLabel(keys.remember), Desc: "Remember a rule / interest"},
		{Key: prettyKeyLabel(keys.confirm), Desc: "Confirm & apply the whole plan (two-press)"},
		{Key: "Tab", Desc: "Move focus to the inbox"},
		{Key: "Esc", Desc: "Close the panel"},
	}
}
