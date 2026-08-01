package services

import (
	"context"
	"sync"
	"time"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/db"
)

// TelemetryServiceImpl is a local-only, opt-in usage-analytics recorder. Events
// are buffered on a channel and written to SQLite in small batches by a single
// background goroutine, so RecordEvent never blocks the UI. When disabled, no
// goroutine runs and RecordEvent is a no-op. Nothing is ever uploaded.
type TelemetryServiceImpl struct {
	store        *db.TelemetryStore
	config       *config.Config
	accountEmail string

	events chan db.TelemetryEvent
	done   chan struct{}
	wg     sync.WaitGroup
}

const (
	telemetryBufferSize = 256
	telemetryFlushEvery = 2 * time.Second
	telemetryBatchMax   = 64
)

// NewTelemetryService builds a telemetry recorder for the given account's store.
// The background writer only starts when telemetry is enabled in config; toggling
// it on takes effect on the next launch (or account switch).
func NewTelemetryService(store *db.TelemetryStore, cfg *config.Config, accountEmail string) *TelemetryServiceImpl {
	s := &TelemetryServiceImpl{
		store:        store,
		config:       cfg,
		accountEmail: accountEmail,
	}
	if s.IsEnabled() && store != nil {
		s.events = make(chan db.TelemetryEvent, telemetryBufferSize)
		s.done = make(chan struct{})
		// Prune stale events on startup (best-effort, off the caller's path).
		go s.pruneOnStartup()
		s.wg.Add(1)
		go s.run()
	}
	return s
}

// IsEnabled reports the live opt-in state from config.
func (s *TelemetryServiceImpl) IsEnabled() bool {
	return s != nil && s.config != nil && s.config.Telemetry.Enabled
}

// RecordEvent enqueues an event without blocking. Dropped silently if the buffer
// is full (telemetry is best-effort and must never slow the UI) or disabled.
func (s *TelemetryServiceImpl) RecordEvent(kind, name string, ok bool) {
	s.enqueue(db.TelemetryEvent{Kind: kind, Name: name, OK: ok})
}

// RecordAction records the outcome (ok + wall-clock duration) of a named action.
func (s *TelemetryServiceImpl) RecordAction(name string, ok bool, durationMs int64) {
	s.enqueue(db.TelemetryEvent{Kind: "action", Name: name, OK: ok, DurationMs: durationMs})
}

// enqueue stamps the event with time/account and hands it to the writer without
// blocking. No-op when disabled or when the buffer is full (best-effort).
func (s *TelemetryServiceImpl) enqueue(ev db.TelemetryEvent) {
	if s == nil || s.events == nil || !s.IsEnabled() {
		return
	}
	ev.TS = time.Now().Unix()
	ev.AccountEmail = s.accountEmail
	select {
	case s.events <- ev:
	default:
		// Buffer full — drop rather than block.
	}
}

// run is the single background writer: it batches events and flushes them on a
// timer or when the batch fills up, and drains on shutdown.
func (s *TelemetryServiceImpl) run() {
	defer s.wg.Done()
	ticker := time.NewTicker(telemetryFlushEvery)
	defer ticker.Stop()
	batch := make([]db.TelemetryEvent, 0, telemetryBatchMax)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		// Best-effort write; a failed flush drops that batch rather than retrying.
		_ = s.store.InsertEvents(context.Background(), batch)
		batch = batch[:0]
	}

	for {
		select {
		case ev := <-s.events:
			batch = append(batch, ev)
			if len(batch) >= telemetryBatchMax {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.done:
			// Drain whatever is still queued, then final flush.
			for {
				select {
				case ev := <-s.events:
					batch = append(batch, ev)
				default:
					flush()
					return
				}
			}
		}
	}
}

func (s *TelemetryServiceImpl) pruneOnStartup() {
	if s.store == nil || s.config == nil {
		return
	}
	days := s.config.Telemetry.RetentionDays
	if days <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days).Unix()
	_, _ = s.store.Prune(context.Background(), cutoff)
}

// Summary aggregates usage over the last windowDays for the dashboard.
func (s *TelemetryServiceImpl) Summary(ctx context.Context, windowDays int) (*TelemetrySummary, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	since := time.Now().AddDate(0, 0, -windowDays).Unix()

	total, errs, err := s.store.Totals(ctx, "", since)
	if err != nil {
		return nil, err
	}
	cmds, err := s.store.TopByKind(ctx, "", "command", since, 10)
	if err != nil {
		return nil, err
	}
	keys, err := s.store.TopByKind(ctx, "", "shortcut", since, 10)
	if err != nil {
		return nil, err
	}
	actions, err := s.store.ActionStats(ctx, "", since, 10)
	if err != nil {
		return nil, err
	}
	return &TelemetrySummary{
		WindowDays:   windowDays,
		TotalActions: total,
		TotalErrors:  errs,
		TopCommands:  toNameCounts(cmds),
		TopShortcuts: toNameCounts(keys),
		TopActions:   toActionStats(actions),
	}, nil
}

// Reset deletes all telemetry for the active account's database.
func (s *TelemetryServiceImpl) Reset(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.store.Clear(ctx, "")
}

// Close flushes buffered events and stops the writer.
func (s *TelemetryServiceImpl) Close() {
	if s == nil || s.done == nil {
		return
	}
	close(s.done)
	s.wg.Wait()
	s.done = nil
}

func toNameCounts(in []db.NameCount) []TelemetryNameCount {
	out := make([]TelemetryNameCount, 0, len(in))
	for _, nc := range in {
		out = append(out, TelemetryNameCount{Name: nc.Name, Count: nc.Count})
	}
	return out
}

func toActionStats(in []db.ActionStat) []TelemetryActionStat {
	out := make([]TelemetryActionStat, 0, len(in))
	for _, s := range in {
		out = append(out, TelemetryActionStat{
			Name:          s.Name,
			Count:         s.Count,
			Failures:      s.Failures,
			AvgDurationMs: s.AvgDurationMs,
		})
	}
	return out
}
