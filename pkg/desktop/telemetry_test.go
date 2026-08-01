package desktop

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
	"github.com/ajramos/giztui/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetry_CaptureAndSummary(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(ctx, filepath.Join(t.TempDir(), "tel.db"))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	cfg := &config.Config{Telemetry: config.TelemetryConfig{Enabled: true, RetentionDays: 90}}
	tel := services.NewTelemetryService(db.NewTelemetryStore(store), cfg, "a@x")

	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}, Telemetry: tel})

	// Action outcome (backend-instrumented) + command/shortcut capture (frontend
	// bindings) all flow through the same recorder.
	require.NoError(t, api.Archive(ctx, "m1"))
	api.RecordCommand("search")
	api.RecordCommand("search")
	api.RecordShortcut("j")

	// Deterministic flush before aggregation.
	tel.Close()

	sum, err := api.TelemetrySummary(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, 30, sum.WindowDays)
	assert.Equal(t, 4, sum.TotalActions) // 1 action + 2 commands + 1 shortcut

	require.NotEmpty(t, sum.TopActions)
	assert.Equal(t, "archive", sum.TopActions[0].Name)
	assert.Equal(t, 1, sum.TopActions[0].Count)
	assert.Equal(t, 0, sum.TopActions[0].Failures)

	require.NotEmpty(t, sum.TopCommands)
	assert.Equal(t, "search", sum.TopCommands[0].Name)
	assert.Equal(t, 2, sum.TopCommands[0].Count)

	require.NotEmpty(t, sum.TopShortcuts)
	assert.Equal(t, "j", sum.TopShortcuts[0].Name)
}

func TestTelemetry_DisabledIsNoOp(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(ctx, filepath.Join(t.TempDir(), "tel.db"))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	cfg := &config.Config{Telemetry: config.TelemetryConfig{Enabled: false, RetentionDays: 90}}
	tel := services.NewTelemetryService(db.NewTelemetryStore(store), cfg, "a@x")
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}, Telemetry: tel})

	assert.False(t, api.TelemetryEnabled())
	require.NoError(t, api.Archive(ctx, "m1"))
	api.RecordCommand("search")
	tel.Close()

	sum, err := api.TelemetrySummary(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, 0, sum.TotalActions)
}

// TestTelemetry_NilServiceSafe ensures the API tolerates no telemetry service
// (e.g. no local DB) without panicking.
func TestTelemetry_NilServiceSafe(t *testing.T) {
	ctx := context.Background()
	api := NewAPI(Deps{Repo: &fakeRepo{}, Email: &fakeEmail{}, Mail: &fakeMail{}})
	assert.False(t, api.TelemetryEnabled())
	require.NoError(t, api.Archive(ctx, "m1"))
	api.RecordCommand("search")
	api.RecordShortcut("j")
	sum, err := api.TelemetrySummary(ctx, 30)
	require.NoError(t, err)
	assert.Equal(t, 0, sum.TotalActions)
}
