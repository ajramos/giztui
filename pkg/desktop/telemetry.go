package desktop

import (
	"context"
	"time"
)

// Local, opt-in usage analytics for the desktop, mirroring the TUI. Capture is a
// no-op unless the user turned telemetry on in config; nothing is ever uploaded.
// Commands and shortcuts are recorded from the frontend (RecordCommand /
// RecordShortcut); action outcomes (archive/trash/summarize) are recorded in the
// backend methods via recordAction.

// TelemetryEnabled reports whether local usage analytics are on (opt-in).
func (a *API) TelemetryEnabled() bool {
	return a.telemetry != nil && a.telemetry.IsEnabled()
}

// RecordCommand records a command invocation by its bare name (no arguments).
// No-op when telemetry is disabled.
func (a *API) RecordCommand(name string) {
	if a.telemetry != nil {
		a.telemetry.RecordEvent("command", name, true)
	}
}

// RecordShortcut records a single keyboard shortcut keypress. No-op when
// telemetry is disabled.
func (a *API) RecordShortcut(key string) {
	if a.telemetry != nil {
		a.telemetry.RecordEvent("shortcut", key, true)
	}
}

// recordAction records the outcome (ok + wall-clock duration) of a named backend
// action. No-op when telemetry is disabled.
func (a *API) recordAction(name string, start time.Time, err error) {
	if a.telemetry != nil {
		a.telemetry.RecordAction(name, err == nil, time.Since(start).Milliseconds())
	}
}

// TelemetrySummary aggregates local usage over the last windowDays for the
// ":stats" dashboard. Returns an empty summary when telemetry is unavailable.
func (a *API) TelemetrySummary(ctx context.Context, windowDays int) (*TelemetrySummary, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	if a.telemetry == nil {
		return &TelemetrySummary{WindowDays: windowDays}, nil
	}
	s, err := a.telemetry.Summary(ctx, windowDays)
	if err != nil {
		return nil, err
	}
	out := &TelemetrySummary{
		WindowDays:   s.WindowDays,
		TotalActions: s.TotalActions,
		TotalErrors:  s.TotalErrors,
		TopCommands:  make([]TelemetryNameCount, 0, len(s.TopCommands)),
		TopShortcuts: make([]TelemetryNameCount, 0, len(s.TopShortcuts)),
		TopActions:   make([]TelemetryActionStat, 0, len(s.TopActions)),
	}
	for _, c := range s.TopCommands {
		out.TopCommands = append(out.TopCommands, TelemetryNameCount{Name: c.Name, Count: c.Count})
	}
	for _, k := range s.TopShortcuts {
		out.TopShortcuts = append(out.TopShortcuts, TelemetryNameCount{Name: k.Name, Count: k.Count})
	}
	for _, ac := range s.TopActions {
		out.TopActions = append(out.TopActions, TelemetryActionStat{
			Name:          ac.Name,
			Count:         ac.Count,
			Failures:      ac.Failures,
			AvgDurationMs: ac.AvgDurationMs,
		})
	}
	return out, nil
}

// TelemetryReset deletes all captured telemetry for the active account.
func (a *API) TelemetryReset(ctx context.Context) error {
	if a.telemetry == nil {
		return nil
	}
	return a.telemetry.Reset(ctx)
}
