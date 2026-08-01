package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

// telemetryBar renders a proportional block bar of the given width.
func telemetryBar(count, max, width int) string {
	if max <= 0 || width <= 0 {
		return ""
	}
	filled := int(float64(count)/float64(max)*float64(width) + 0.5)
	if filled < 1 && count > 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat(" ", width-filled)
}

// telemetrySection formats a titled list of name→count rows as aligned bars.
func telemetrySection(title string, rows []services.TelemetryNameCount) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[::b]%s[::-]\n", title))
	if len(rows) == 0 {
		b.WriteString("  (none yet)\n")
		return b.String()
	}
	max := 0
	nameW := 0
	for _, r := range rows {
		if r.Count > max {
			max = r.Count
		}
		if len(r.Name) > nameW {
			nameW = len(r.Name)
		}
	}
	if nameW > 18 {
		nameW = 18
	}
	for _, r := range rows {
		name := r.Name
		if len(name) > nameW {
			name = name[:nameW]
		}
		b.WriteString(fmt.Sprintf("  %-*s  %s  %d\n", nameW, name, telemetryBar(r.Count, max, 18), r.Count))
	}
	return b.String()
}

// generateTelemetryContent builds the usage-analytics dashboard text.
func (a *App) generateTelemetryContent(s *services.TelemetrySummary) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Local usage analytics · last %d days\n\n", s.WindowDays))
	b.WriteString(fmt.Sprintf("  Total actions:  %d\n", s.TotalActions))
	b.WriteString(fmt.Sprintf("  Errors:         %d\n\n", s.TotalErrors))

	if s.TotalActions == 0 {
		b.WriteString("No activity captured yet in this window.\n\n")
		b.WriteString("Telemetry is local-only and opt-in. Keep using GizTUI and\n")
		b.WriteString("your command/shortcut usage will appear here.\n\n")
	} else {
		b.WriteString(telemetrySection("Top commands", s.TopCommands))
		b.WriteString("\n")
		b.WriteString(telemetrySection("Top shortcuts (keys)", s.TopShortcuts))
		b.WriteString("\n")
	}

	b.WriteString("[::d]All data stays on this machine and is never uploaded.\n")
	b.WriteString("Reset with :stats reset · window: :stats <days> · prompt usage: :prompt stats · Esc to close.[::-]\n")
	return b.String()
}
