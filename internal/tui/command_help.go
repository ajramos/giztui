package tui

import (
	"fmt"
	"strings"
)

// generateCommandHelpText renders focused help for a single command, shown by :help <cmd> in the
// reader pane (same overlay as the full help). Rich when spec.help is set, otherwise an auto fallback
// derived from the registry (name + aliases). tview dynamic-color tags (theme-aware) style the title
// and section headings so it matches the look of the full help screen.
func (a *App) generateCommandHelpText(s *commandSpec) string {
	colors := a.GetComponentColors("general")
	title := colors.Title.String()
	accent := colors.Accent.String()
	heading := func(h string) string { return fmt.Sprintf("[%s::b]%s[-:-:-]", accent, h) }

	var b strings.Builder
	fmt.Fprintf(&b, "[%s::b] :%s [-:-:-]\n\n", title, s.name)

	if s.help == nil {
		b.WriteString("No detailed help for this command.\n\n")
		writeAliases(&b, s, heading)
		b.WriteString("\nPress Esc to return. Press ? for the full command/shortcut list.\n")
		return b.String()
	}

	b.WriteString(s.help.summary + "\n\n")
	if s.help.syntax != "" {
		fmt.Fprintf(&b, "%s\n    %s\n\n", heading("Syntax:"), s.help.syntax)
	}
	if len(s.help.examples) > 0 {
		fmt.Fprintf(&b, "%s\n", heading("Examples:"))
		for _, ex := range s.help.examples {
			fmt.Fprintf(&b, "    %s\n", ex)
		}
		b.WriteString("\n")
	}
	writeAliases(&b, s, heading)
	b.WriteString("\nPress Esc to return.\n")
	return b.String()
}

func writeAliases(b *strings.Builder, s *commandSpec, heading func(string) string) {
	if len(s.aliases) == 0 {
		fmt.Fprintf(b, "%s (none)\n", heading("Aliases:"))
		return
	}
	fmt.Fprintf(b, "%s %s\n", heading("Aliases:"), strings.Join(s.aliases, ", "))
}
