package services

import (
	"context"
	"fmt"
	"strings"

	gmailapi "google.golang.org/api/gmail/v1"
)

// importQueryFromCriteria translates a Gmail filter's criteria into a search query.
// Size and chat-exclusion criteria have no query equivalent → error (Gmail-only).
func importQueryFromCriteria(c *gmailapi.FilterCriteria) (string, error) {
	if c == nil {
		return "", fmt.Errorf("filter has no criteria")
	}
	if c.Size != 0 || c.SizeComparison != "" {
		return "", fmt.Errorf("size criteria can't be expressed as a rule query")
	}
	if c.ExcludeChats {
		return "", fmt.Errorf("chat-exclusion criteria can't be expressed as a rule query")
	}
	var parts []string
	if v := strings.TrimSpace(c.From); v != "" {
		parts = append(parts, "from:("+v+")")
	}
	if v := strings.TrimSpace(c.To); v != "" {
		parts = append(parts, "to:("+v+")")
	}
	if v := strings.TrimSpace(c.Subject); v != "" {
		parts = append(parts, "subject:("+v+")")
	}
	if v := strings.TrimSpace(c.Query); v != "" {
		parts = append(parts, v)
	}
	if v := strings.TrimSpace(c.NegatedQuery); v != "" {
		parts = append(parts, "-("+v+")")
	}
	if c.HasAttachment {
		parts = append(parts, "has:attachment")
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("filter has no usable criteria")
	}
	return strings.Join(parts, " "), nil
}

// importActionFromFilter maps a filter's action onto a single rule action.
// Only exact one-action shapes translate — a lossy import (e.g. keeping just the
// label of a "label + skip inbox" filter) would silently rewrite the real filter
// on the next edit/sync, so combos stay Gmail-only. labelName resolves label IDs.
func importActionFromFilter(f *gmailapi.Filter, labelName map[string]string) (action, label string, err error) {
	if f == nil || f.Action == nil {
		return "", "", fmt.Errorf("filter has no action")
	}
	a := f.Action
	if a.Forward != "" {
		return "", "", fmt.Errorf("forwards mail to %s", a.Forward)
	}
	add, remove := a.AddLabelIds, a.RemoveLabelIds
	switch {
	case len(add) == 0 && len(remove) == 1 && remove[0] == "INBOX":
		return "archive", "", nil
	case len(add) == 0 && len(remove) == 1 && remove[0] == "UNREAD":
		return "mark_read", "", nil
	case len(remove) == 0 && len(add) == 1 && add[0] == "TRASH":
		return "trash", "", nil
	case len(remove) == 0 && len(add) == 1:
		name := labelName[add[0]]
		if name == "" {
			return "", "", fmt.Errorf("applies an unknown label (%s)", add[0])
		}
		return "label", name, nil
	case len(add)+len(remove) > 1:
		return "", "", fmt.Errorf("does several things at once")
	default:
		return "", "", fmt.Errorf("has no action a rule can perform")
	}
}

// describeGmailFilter renders a best-effort "criteria → action" one-liner for the
// read-only Gmail-only rows.
func describeGmailFilter(f *gmailapi.Filter, labelName map[string]string) string {
	criteria := "(no criteria)"
	if q, err := importQueryFromCriteria(f.Criteria); err == nil {
		criteria = q
	} else if f.Criteria != nil {
		if f.Criteria.Size != 0 {
			criteria = fmt.Sprintf("size %s %d", f.Criteria.SizeComparison, f.Criteria.Size)
		}
	}
	var acts []string
	if f.Action != nil {
		if f.Action.Forward != "" {
			acts = append(acts, "forward to "+f.Action.Forward)
		}
		for _, id := range f.Action.AddLabelIds {
			name := labelName[id]
			if name == "" {
				name = id
			}
			acts = append(acts, "+"+name)
		}
		for _, id := range f.Action.RemoveLabelIds {
			name := labelName[id]
			if name == "" {
				name = id
			}
			acts = append(acts, "-"+name)
		}
	}
	if len(acts) == 0 {
		return criteria
	}
	return criteria + " → " + strings.Join(acts, " ")
}

// importLabelNames returns a Gmail label ID → display name map (empty when the
// label service is unavailable — label filters then surface as Gmail-only).
func (s *DeterministicRulesServiceImpl) importLabelNames(ctx context.Context) map[string]string {
	names := map[string]string{}
	if s.labels == nil {
		return names
	}
	labels, err := s.labels.ListLabels(ctx)
	if err != nil {
		return names
	}
	for _, l := range labels {
		if l != nil {
			names[l.Id] = l.Name
		}
	}
	return names
}

// ImportGmailFilters folds the account's Gmail filters into the rules list.
// See the interface doc for adopt/import/Gmail-only semantics. New rules are
// written through the store directly — their queries already live in Gmail, so
// the save-time query validation round-trip would be wasted API calls.
func (s *DeterministicRulesServiceImpl) ImportGmailFilters(ctx context.Context) (*GmailImportResult, error) {
	if s.store == nil {
		return nil, fmt.Errorf("deterministic rules store not available")
	}
	acct, err := s.account()
	if err != nil {
		return nil, err
	}
	if s.filters == nil {
		return nil, fmt.Errorf("no Gmail client available")
	}
	filters, err := s.filters.ListFilters()
	if err != nil {
		return nil, fmt.Errorf("could not read Gmail filters: %w", err)
	}
	rules, err := s.ListRules(ctx)
	if err != nil {
		return nil, err
	}

	linked := map[string]bool{} // filter IDs already owned by a rule
	type ruleKey struct{ query, action, label string }
	key := func(q, a, l string) ruleKey {
		return ruleKey{strings.ToLower(strings.TrimSpace(q)), a, strings.ToLower(strings.TrimSpace(l))}
	}
	byKey := map[ruleKey]*DeterministicRuleInfo{}
	for i := range rules {
		if rules[i].GmailFilterID != "" {
			linked[rules[i].GmailFilterID] = true
		}
		k := key(rules[i].Query, rules[i].Action, rules[i].Label)
		if _, ok := byKey[k]; !ok {
			byKey[k] = &rules[i]
		}
	}

	labelName := s.importLabelNames(ctx)
	res := &GmailImportResult{}
	for _, f := range filters {
		if f == nil || f.Id == "" || linked[f.Id] {
			continue
		}
		query, qErr := importQueryFromCriteria(f.Criteria)
		action, label, aErr := importActionFromFilter(f, labelName)
		if qErr != nil || aErr != nil {
			reason := qErr
			if reason == nil {
				reason = aErr
			}
			res.Unsupported = append(res.Unsupported, GmailOnlyFilter{
				ID: f.Id, Description: describeGmailFilter(f, labelName), Reason: reason.Error(),
			})
			continue
		}
		if r, ok := byKey[key(query, action, label)]; ok {
			if r.GmailFilterID == "" {
				if err := s.store.SetGmailFilterID(ctx, acct, r.ID, f.Id); err == nil {
					r.GmailFilterID = f.Id
					linked[f.Id] = true
					res.Adopted++
				}
			}
			continue // identical rule already exists (mirrored or just adopted)
		}
		saved, err := s.store.SaveRule(ctx, acct, query, action, label, 0)
		if err != nil {
			continue // best effort — remaining filters still get their chance
		}
		if err := s.store.SetGmailFilterID(ctx, acct, saved.ID, f.Id); err == nil {
			linked[f.Id] = true
			info := ruleInfoFromDB(saved)
			info.GmailFilterID = f.Id
			byKey[key(query, action, label)] = &info
			res.Imported++
		}
	}
	return res, nil
}
