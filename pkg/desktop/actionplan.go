package desktop

import (
	"context"
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

// ActionPlanEnabled reports whether the AI inbox action plan is available.
func (a *API) ActionPlanEnabled() bool { return a.analyzer != nil }

// AnalyzeInbox runs the AI inbox analyzer over the given messages and returns a
// categorized action plan. The frontend passes the message data it already has
// to avoid re-fetching bodies.
func (a *API) AnalyzeInbox(ctx context.Context, inputs []AnalyzerInput) (*ActionPlanResult, error) {
	if a.analyzer == nil {
		return nil, fmt.Errorf("the inbox action plan needs an LLM provider")
	}
	msgs := make([]services.AnalyzerMessage, 0, len(inputs))
	for _, in := range inputs {
		msgs = append(msgs, services.AnalyzerMessage{
			ID: in.ID, Subject: in.Subject, From: in.From, Snippet: in.Snippet,
		})
	}
	var available []string
	if ls, err := a.labels.ListLabels(ctx); err == nil {
		for _, l := range ls {
			if l == nil {
				continue
			}
			if _, sys := systemLabels[l.Id]; sys || strings.HasPrefix(l.Id, "CATEGORY_") {
				continue
			}
			available = append(available, l.Name)
		}
	}
	plan, err := a.analyzer.Analyze(ctx, msgs, services.InboxAnalyzerOptions{
		BatchSize:       50,
		MaxBatches:      5,
		AvailableLabels: available,
	}, nil)
	if err != nil {
		return nil, err
	}
	res := &ActionPlanResult{
		TotalAnalyzed: plan.TotalAnalyzed,
		ReadManually:  len(plan.ReadManually),
	}
	for _, c := range plan.Categories {
		res.Categories = append(res.Categories, PlanCategory{
			Name: c.Name, Priority: c.Priority, Description: c.Description,
			Action: c.Action, Label: c.Label, MessageIDs: c.MessageIDs,
		})
	}
	return res, nil
}

// BulkApplyLabelByName applies a label (by name, creating it if needed) to many
// messages — used when applying the "label" action of a plan category.
func (a *API) BulkApplyLabelByName(ctx context.Context, ids []string, name string) error {
	if len(ids) == 0 {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("label name is required")
	}
	var labelID string
	ls, err := a.labels.ListLabels(ctx)
	if err != nil {
		return err
	}
	for _, l := range ls {
		if l != nil && strings.EqualFold(l.Name, name) {
			labelID = l.Id
			break
		}
	}
	if labelID == "" {
		created, err := a.labels.CreateLabel(ctx, name)
		if err != nil {
			return err
		}
		labelID = created.Id
	}
	return a.labels.BulkApplyLabel(ctx, ids, labelID)
}
