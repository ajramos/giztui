package main

// App bindings: action plan, analyzer/deterministic rules, saved queries. Split out of app.go (thin Wails wrappers over pkg/desktop.API).

import (
	"github.com/ajramos/giztui/pkg/desktop"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ActionPlanEnabled() bool {
	return a.enabled((*desktop.API).ActionPlanEnabled)
}

// AnalyzeInbox runs the AI inbox analyzer and returns an action plan, emitting
// "plan:progress" {done,total} events as each batch completes so the UI shows
// real progress.
func (a *App) AnalyzeInbox(inputs []desktop.AnalyzerInput) (*desktop.ActionPlanResult, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.AnalyzeInbox(a.ctx, inputs, func(done, total int) {
		wailsruntime.EventsEmit(a.ctx, planProgressEvent, map[string]int{
			"done": done, "total": total,
		})
	})
}

// RunDeterministicRules applies only the deterministic rules to the given
// messages (the TUI's ":rules plan") and returns them as plan categories.
func (a *App) RunDeterministicRules(inputs []desktop.AnalyzerInput) (*desktop.ActionPlanResult, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.RunDeterministicRules(a.ctx, inputs)
}

// DeterministicRulesRunnable reports whether an on-demand rules run is available.
func (a *App) DeterministicRulesRunnable() bool {
	return a.enabled((*desktop.API).DeterministicRulesRunnable)
}

// BulkApplyLabelByName applies a label by name to many messages.
func (a *App) BulkApplyLabelByName(ids []string, name string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.BulkApplyLabelByName(a.ctx, ids, name)
}

// AnalyzerRulesEnabled reports whether analyzer preference rules are available.
func (a *App) AnalyzerRulesEnabled() bool {
	return a.enabled((*desktop.API).AnalyzerRulesEnabled)
}

// ListAnalyzerRules returns the stored analyzer preference rules.
func (a *App) ListAnalyzerRules() ([]desktop.AnalyzerRule, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListAnalyzerRules(a.ctx)
}

// SaveAnalyzerRule persists a new analyzer preference rule.
func (a *App) SaveAnalyzerRule(text string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SaveAnalyzerRule(a.ctx, text)
}

// DeleteAnalyzerRule removes a stored analyzer preference rule.
func (a *App) DeleteAnalyzerRule(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeleteAnalyzerRule(a.ctx, id)
}

// --- deterministic rules (:rules) -------------------------------------------

// DeterministicRulesEnabled reports whether the rules subsystem is available.
func (a *App) DeterministicRulesEnabled() bool {
	return a.enabled((*desktop.API).DeterministicRulesEnabled)
}

// ListDeterministicRules returns all deterministic rules.
func (a *App) ListDeterministicRules() ([]desktop.DeterministicRule, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListDeterministicRules(a.ctx)
}

// SaveDeterministicRule creates a rule.
func (a *App) SaveDeterministicRule(query, action, label string, promptID int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SaveDeterministicRule(a.ctx, query, action, label, promptID)
}

// UpdateDeterministicRule edits a rule.
func (a *App) UpdateDeterministicRule(id int64, query, action, label string, promptID int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.UpdateDeterministicRule(a.ctx, id, query, action, label, promptID)
}

// DeleteDeterministicRule removes a rule.
func (a *App) DeleteDeterministicRule(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeleteDeterministicRule(a.ctx, id)
}

// SyncDeterministicRule mirrors a rule as a Gmail filter.
func (a *App) SyncDeterministicRule(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SyncDeterministicRule(a.ctx, id)
}

// UnsyncDeterministicRule removes a rule's mirrored Gmail filter.
func (a *App) UnsyncDeterministicRule(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.UnsyncDeterministicRule(a.ctx, id)
}

// ImportGmailFilters reconciles Gmail filters into rules.
func (a *App) ImportGmailFilters() (*desktop.ImportResult, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ImportGmailFilters(a.ctx)
}

// ViewAnalyzerPrompt returns the effective analyzer prompt for inspection.
func (a *App) ViewAnalyzerPrompt() (string, error) {
	api, err := a.api()
	if err != nil {
		return "", err
	}
	return api.ViewAnalyzerPrompt(a.ctx)
}

// SavedQueriesEnabled reports whether saved searches are available.
func (a *App) SavedQueriesEnabled() bool {
	return a.enabled((*desktop.API).SavedQueriesEnabled)
}

// ListSavedQueries returns the saved searches.
func (a *App) ListSavedQueries() ([]desktop.SavedQuery, error) {
	api, err := a.api()
	if err != nil {
		return nil, err
	}
	return api.ListSavedQueries(a.ctx)
}

// SaveQuery persists a named Gmail search.
func (a *App) SaveQuery(name, query string) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.SaveQuery(a.ctx, name, query)
}

// DeleteSavedQuery removes a saved search.
func (a *App) DeleteSavedQuery(id int64) error {
	api, err := a.api()
	if err != nil {
		return err
	}
	return api.DeleteSavedQuery(a.ctx, id)
}

// RecordQueryUse bumps a saved query's usage counter.
func (a *App) RecordQueryUse(id int64) {
	if api, err := a.api(); err == nil {
		api.RecordQueryUse(a.ctx, id)
	}
}
