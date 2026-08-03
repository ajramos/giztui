package desktop

import (
	"context"
	"fmt"
	"strings"
)

// SavedQueriesEnabled reports whether saved searches are available.
func (a *API) SavedQueriesEnabled() bool { return a.query != nil }

// ListSavedQueries returns the saved searches.
func (a *API) ListSavedQueries(ctx context.Context) ([]SavedQuery, error) {
	if a.query == nil {
		return []SavedQuery{}, nil
	}
	qs, err := a.query.ListQueries(ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]SavedQuery, 0, len(qs))
	for _, q := range qs {
		if q == nil {
			continue
		}
		out = append(out, SavedQuery{
			ID: q.ID, Name: q.Name, Query: q.Query,
			Description: q.Description, Category: q.Category,
		})
	}
	return out, nil
}

// SaveQuery persists a named Gmail search under an optional free-form category
// (empty → the picker's "Default" group).
func (a *API) SaveQuery(ctx context.Context, name, query, category string) error {
	if a.query == nil {
		return fmt.Errorf("saved queries are not available")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(query) == "" {
		return fmt.Errorf("a name and query are required")
	}
	_, err := a.query.SaveQuery(ctx, name, query, "", strings.TrimSpace(category))
	return err
}

// UpdateSavedQuery edits an existing saved search (name, query, category) by id.
func (a *API) UpdateSavedQuery(ctx context.Context, id int64, name, query, category string) error {
	if a.query == nil {
		return fmt.Errorf("saved queries are not available")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(query) == "" {
		return fmt.Errorf("a name and query are required")
	}
	return a.query.UpdateQuery(ctx, id, strings.TrimSpace(name), strings.TrimSpace(query), "", strings.TrimSpace(category))
}

// DeleteSavedQuery removes a saved search.
func (a *API) DeleteSavedQuery(ctx context.Context, id int64) error {
	if a.query == nil {
		return fmt.Errorf("saved queries are not available")
	}
	return a.query.DeleteQuery(ctx, id)
}

// RecordQueryUse bumps a saved query's usage counter (best-effort).
func (a *API) RecordQueryUse(ctx context.Context, id int64) {
	if a.query != nil {
		_ = a.query.RecordQueryUsage(ctx, id)
	}
}
