package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTelemetryStore(t *testing.T) (*TelemetryStore, func()) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "telemetry_test.db")
	store, err := Open(ctx, dbPath)
	require.NoError(t, err)
	ts := NewTelemetryStore(store)
	require.NotNil(t, ts)
	return ts, func() { _ = store.Close() }
}

func TestNewTelemetryStore_Nil(t *testing.T) {
	assert.Nil(t, NewTelemetryStore(nil))
}

func TestTelemetry_InsertAggregatePrune(t *testing.T) {
	ts, done := newTelemetryStore(t)
	defer done()
	ctx := context.Background()
	now := time.Now().Unix()

	events := []TelemetryEvent{
		{TS: now, AccountEmail: "a@x", Kind: "command", Name: "archive", OK: true},
		{TS: now, AccountEmail: "a@x", Kind: "command", Name: "archive", OK: true},
		{TS: now, AccountEmail: "a@x", Kind: "command", Name: "search", OK: true},
		{TS: now, AccountEmail: "a@x", Kind: "shortcut", Name: "a", OK: true},
		{TS: now, AccountEmail: "a@x", Kind: "error", Name: "ui", OK: false},
	}
	require.NoError(t, ts.InsertEvents(ctx, events))

	// TopByKind: archive (2) before search (1).
	top, err := ts.TopByKind(ctx, "", "command", now-3600, 10)
	require.NoError(t, err)
	require.Len(t, top, 2)
	assert.Equal(t, "archive", top[0].Name)
	assert.Equal(t, 2, top[0].Count)
	assert.Equal(t, "search", top[1].Name)

	// Totals: 5 events, 1 error.
	total, errs, err := ts.Totals(ctx, "", now-3600)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Equal(t, 1, errs)

	// Account scoping: a different account sees nothing.
	total2, _, err := ts.Totals(ctx, "other@x", now-3600)
	require.NoError(t, err)
	assert.Equal(t, 0, total2)
}

func TestTelemetry_PruneAndClear(t *testing.T) {
	ts, done := newTelemetryStore(t)
	defer done()
	ctx := context.Background()
	now := time.Now().Unix()
	old := now - 100*24*3600 // 100 days ago

	require.NoError(t, ts.InsertEvents(ctx, []TelemetryEvent{
		{TS: old, Kind: "command", Name: "old", OK: true},
		{TS: now, Kind: "command", Name: "new", OK: true},
	}))

	// Prune events older than 90 days: removes exactly the old one.
	cutoff := now - 90*24*3600
	removed, err := ts.Prune(ctx, cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)

	total, _, err := ts.Totals(ctx, "", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, total)

	// Clear wipes everything.
	require.NoError(t, ts.Clear(ctx, ""))
	total, _, err = ts.Totals(ctx, "", 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestTelemetry_InsertEmpty(t *testing.T) {
	ts, done := newTelemetryStore(t)
	defer done()
	assert.NoError(t, ts.InsertEvents(context.Background(), nil))
}
