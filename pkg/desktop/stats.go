package desktop

import "context"

// UsageStats returns AI prompt usage statistics for the stats panel.
func (a *API) UsageStats(ctx context.Context) (*UsageStats, error) {
	if a.prompts == nil {
		return &UsageStats{}, nil
	}
	s, err := a.prompts.GetUsageStats(ctx)
	if err != nil {
		return nil, err
	}
	out := &UsageStats{
		TotalUsage:    s.TotalUsage,
		UniquePrompts: s.UniquePrompts,
	}
	for _, p := range s.TopPrompts {
		out.TopPrompts = append(out.TopPrompts, UsageStat{
			Name:       p.Name,
			Category:   p.Category,
			UsageCount: p.UsageCount,
		})
	}
	return out, nil
}

// ClearCaches clears the AI summary cache and prompt-result caches for the
// active account (best-effort; returns the first error encountered).
func (a *API) ClearCaches(ctx context.Context) error {
	if a.cache != nil && a.accountEmail != "" {
		if err := a.cache.ClearCache(ctx, a.accountEmail); err != nil {
			return err
		}
	}
	if a.prompts != nil {
		if err := a.prompts.ClearAllPromptCaches(ctx); err != nil {
			return err
		}
	}
	return nil
}
