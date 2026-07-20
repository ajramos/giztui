package desktop

import (
	"context"
	"fmt"
)

// DeterministicRule is the frontend-facing shape of a deterministic rule.
type DeterministicRule struct {
	ID       int64  `json:"id"`
	Query    string `json:"query"`
	Action   string `json:"action"` // archive | mark_read | trash | label | prompt
	Label    string `json:"label"`  // when Action == "label"
	PromptID int64  `json:"promptId"`
	// Synced is true when the rule is mirrored as a real Gmail filter (the ☁).
	Synced    bool  `json:"synced"`
	CreatedAt int64 `json:"createdAt"`
}

// GmailOnly is a Gmail filter that couldn't be translated to a local rule; shown
// read-only after an import.
type GmailOnly struct {
	Description string `json:"description"`
	Reason      string `json:"reason"`
}

// ImportResult summarises a Gmail-filter import.
type ImportResult struct {
	Imported    int         `json:"imported"`
	Adopted     int         `json:"adopted"`
	Removed     int         `json:"removed"`
	Unsupported []GmailOnly `json:"unsupported"`
}

// DeterministicRulesEnabled reports whether the rules subsystem is available.
func (a *API) DeterministicRulesEnabled() bool { return a.detRules != nil }

// ListDeterministicRules returns all rules in creation order.
func (a *API) ListDeterministicRules(ctx context.Context) ([]DeterministicRule, error) {
	if a.detRules == nil {
		return []DeterministicRule{}, nil
	}
	infos, err := a.detRules.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DeterministicRule, 0, len(infos))
	for _, r := range infos {
		out = append(out, DeterministicRule{
			ID:        r.ID,
			Query:     r.Query,
			Action:    r.Action,
			Label:     r.Label,
			PromptID:  r.PromptID,
			Synced:    r.GmailFilterID != "",
			CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

// SaveDeterministicRule creates a rule (query validated Gmail-side at save time).
func (a *API) SaveDeterministicRule(ctx context.Context, query, action, label string, promptID int64) error {
	if a.detRules == nil {
		return fmt.Errorf("deterministic rules unavailable")
	}
	_, err := a.detRules.SaveRule(ctx, query, action, label, promptID)
	return err
}

// UpdateDeterministicRule edits an existing rule.
func (a *API) UpdateDeterministicRule(ctx context.Context, id int64, query, action, label string, promptID int64) error {
	if a.detRules == nil {
		return fmt.Errorf("deterministic rules unavailable")
	}
	return a.detRules.UpdateRule(ctx, id, query, action, label, promptID)
}

// DeleteDeterministicRule removes a rule (and its mirrored filter, if any).
func (a *API) DeleteDeterministicRule(ctx context.Context, id int64) error {
	if a.detRules == nil {
		return fmt.Errorf("deterministic rules unavailable")
	}
	return a.detRules.DeleteRule(ctx, id)
}

// SyncDeterministicRule mirrors the rule as a Gmail filter.
func (a *API) SyncDeterministicRule(ctx context.Context, id int64) error {
	if a.detRules == nil {
		return fmt.Errorf("deterministic rules unavailable")
	}
	return a.detRules.SyncRule(ctx, id)
}

// UnsyncDeterministicRule removes the mirrored Gmail filter.
func (a *API) UnsyncDeterministicRule(ctx context.Context, id int64) error {
	if a.detRules == nil {
		return fmt.Errorf("deterministic rules unavailable")
	}
	return a.detRules.UnsyncRule(ctx, id)
}

// ImportGmailFilters reconciles the account's Gmail filters into rules.
func (a *API) ImportGmailFilters(ctx context.Context) (*ImportResult, error) {
	if a.detRules == nil {
		return nil, fmt.Errorf("deterministic rules unavailable")
	}
	res, err := a.detRules.ImportGmailFilters(ctx)
	if err != nil {
		return nil, err
	}
	out := &ImportResult{Imported: res.Imported, Adopted: res.Adopted, Removed: res.Removed}
	for _, u := range res.Unsupported {
		out.Unsupported = append(out.Unsupported, GmailOnly{Description: u.Description, Reason: u.Reason})
	}
	return out, nil
}
