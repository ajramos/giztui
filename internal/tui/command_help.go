package tui

import (
	"fmt"
	"strings"
)

// generateCommandHelpText renders focused help for a single command, shown by :help <cmd> in the
// reader pane (same overlay as the full help). Rich when spec.help is set, otherwise an auto fallback
// derived from the registry (name + aliases). tview dynamic-color tags style the title.
func generateCommandHelpText(s *commandSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[::b] :%s [::-]\n\n", s.name)

	if s.help == nil {
		b.WriteString("No detailed help for this command.\n\n")
		writeAliases(&b, s)
		b.WriteString("\nPress Esc to return. Press ? for the full command/shortcut list.\n")
		return b.String()
	}

	b.WriteString(s.help.summary + "\n\n")
	if s.help.syntax != "" {
		fmt.Fprintf(&b, "Syntax:\n    %s\n\n", s.help.syntax)
	}
	if len(s.help.examples) > 0 {
		b.WriteString("Examples:\n")
		for _, ex := range s.help.examples {
			fmt.Fprintf(&b, "    %s\n", ex)
		}
		b.WriteString("\n")
	}
	writeAliases(&b, s)
	b.WriteString("\nPress Esc to return.\n")
	return b.String()
}

func writeAliases(b *strings.Builder, s *commandSpec) {
	if len(s.aliases) == 0 {
		b.WriteString("Aliases: (none)\n")
		return
	}
	fmt.Fprintf(b, "Aliases: %s\n", strings.Join(s.aliases, ", "))
}
