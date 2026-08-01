package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTelemetryTestStore(t *testing.T) *db.TelemetryStore {
	t.Helper()
	store, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "tel.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return db.NewTelemetryStore(store)
}

func TestTelemetryService_DisabledIsNoOp(t *testing.T) {
	cfg := &config.Config{Telemetry: config.TelemetryConfig{Enabled: false, RetentionDays: 90}}
	svc := NewTelemetryService(newTelemetryTestStore(t), cfg, "a@x")
	defer svc.Close()

	assert.False(t, svc.IsEnabled())
	// Recording while disabled must not persist anything.
	svc.RecordEvent("command", "archive", true)
	svc.RecordEvent("shortcut", "a", true)

	sum, err := svc.Summary(context.Background(), 30)
	require.NoError(t, err)
	assert.Equal(t, 0, sum.TotalActions)
}

func TestTelemetryService_EnabledCapturesAndAggregates(t *testing.T) {
	cfg := &config.Config{Telemetry: config.TelemetryConfig{Enabled: true, RetentionDays: 90}}
	svc := NewTelemetryService(newTelemetryTestStore(t), cfg, "a@x")

	require.True(t, svc.IsEnabled())
	svc.RecordEvent("command", "archive", true)
	svc.RecordEvent("command", "archive", true)
	svc.RecordEvent("command", "search", true)
	svc.RecordEvent("shortcut", "a", true)
	svc.RecordEvent("error", "ui", false)

	// Close flushes the buffered writer deterministically.
	svc.Close()

	sum, err := svc.Summary(context.Background(), 30)
	require.NoError(t, err)
	assert.Equal(t, 5, sum.TotalActions)
	assert.Equal(t, 1, sum.TotalErrors)
	require.NotEmpty(t, sum.TopCommands)
	assert.Equal(t, "archive", sum.TopCommands[0].Name)
	assert.Equal(t, 2, sum.TopCommands[0].Count)
	require.NotEmpty(t, sum.TopShortcuts)
	assert.Equal(t, "a", sum.TopShortcuts[0].Name)

	// Reset clears everything.
	require.NoError(t, svc.Reset(context.Background()))
	sum, err = svc.Summary(context.Background(), 30)
	require.NoError(t, err)
	assert.Equal(t, 0, sum.TotalActions)
}

func TestTelemetryService_RecordActionOutcomes(t *testing.T) {
	cfg := &config.Config{Telemetry: config.TelemetryConfig{Enabled: true, RetentionDays: 90}}
	svc := NewTelemetryService(newTelemetryTestStore(t), cfg, "a@x")

	require.True(t, svc.IsEnabled())
	svc.RecordAction("archive", true, 100)
	svc.RecordAction("archive", false, 300)
	svc.RecordAction("summarize", true, 1400)

	// Close flushes the buffered writer deterministically.
	svc.Close()

	sum, err := svc.Summary(context.Background(), 30)
	require.NoError(t, err)
	require.NotEmpty(t, sum.TopActions)

	// archive ordered first (2 runs), 1 failure, avg 200ms.
	assert.Equal(t, "archive", sum.TopActions[0].Name)
	assert.Equal(t, 2, sum.TopActions[0].Count)
	assert.Equal(t, 1, sum.TopActions[0].Failures)
	assert.Equal(t, 200, sum.TopActions[0].AvgDurationMs)

	// summarize captured with its timing.
	assert.Equal(t, "summarize", sum.TopActions[1].Name)
	assert.Equal(t, 1400, sum.TopActions[1].AvgDurationMs)
}
