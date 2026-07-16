package tui

import (
	"fmt"
	"net/mail"
	"sort"
	"strings"

	"github.com/ajramos/giztui/internal/services"
)

type readManuallyGroup struct {
	senderKey  string // normalized address, lowercased
	senderDisp string // first-seen raw From, for display
	msgs       []services.AnalyzerMessage
}

// normalizeSender extracts a lowercased email address from a From header, falling back to the
// trimmed/lowercased raw value when it doesn't parse.
func normalizeSender(from string) string {
	if addr, err := mail.ParseAddress(strings.TrimSpace(from)); err == nil {
		return strings.ToLower(strings.TrimSpace(addr.Address))
	}
	return strings.ToLower(strings.TrimSpace(from))
}

// groupReadManuallyBySender groups messages by normalized sender, ordered by descending group
// size then senderKey; within a group, input order is preserved.
func groupReadManuallyBySender(msgs []services.AnalyzerMessage) []readManuallyGroup {
	idx := map[string]int{}
	var groups []readManuallyGroup
	for _, m := range msgs {
		key := normalizeSender(m.From)
		if i, ok := idx[key]; ok {
			groups[i].msgs = append(groups[i].msgs, m)
			continue
		}
		idx[key] = len(groups)
		groups = append(groups, readManuallyGroup{senderKey: key, senderDisp: strings.TrimSpace(m.From), msgs: []services.AnalyzerMessage{m}})
	}
	sort.SliceStable(groups, func(a, b int) bool {
		if len(groups[a].msgs) != len(groups[b].msgs) {
			return len(groups[a].msgs) > len(groups[b].msgs)
		}
		return groups[a].senderKey < groups[b].senderKey
	})
	return groups
}

// senderExpandKey is the state.expanded map key for a sender group under read-manually.
func senderExpandKey(senderKey string) string {
	return "\x00read-manually:" + senderKey
}

// readManuallyLeafLabel renders one email leaf, appending the AI hint/suggestion when present.
func readManuallyLeafLabel(m services.AnalyzerMessage, sug services.ReadManuallySuggestion, hasSug bool) string {
	subject := strings.TrimSpace(m.Subject)
	if subject == "" {
		subject = "(no subject)"
	}
	if !hasSug || (sug.Hint == "" && sug.Action == "read") {
		return subject
	}
	if sug.Action == "read" {
		return subject + " — 💡 " + sug.Hint
	}
	verb := actionVerbLabel(sug.Action)
	if sug.Action == "label" && sug.Label != "" {
		verb = verb + " " + sug.Label
	}
	out := subject
	if sug.Hint != "" {
		out += " — 💡 " + sug.Hint
	}
	return out + " · suggests: " + verb
}

// assistReadManually runs the on-demand AI pass over the current read-manually bucket and
// re-renders with hints/suggested actions. Called on the event loop; work runs in a goroutine.
func (a *App) assistReadManually(state *actionPlanState) {
	if state == nil || state.plan == nil || len(state.plan.ReadManually) == 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing in Read manually to assist")
		return
	}
	analyzer := a.GetInboxAnalyzerService()
	if analyzer == nil {
		go a.GetErrorHandler().ShowWarning(a.ctx, "AI analyzer not available")
		return
	}
	msgs := append([]services.AnalyzerMessage(nil), state.plan.ReadManually...)
	opts := services.InboxAnalyzerOptions{
		BatchSize:       a.Config.InboxAnalyzer.BatchSize,
		MaxBatches:      a.Config.InboxAnalyzer.MaxBatches,
		BodyCharLimit:   a.Config.InboxAnalyzer.BodyCharLimit,
		AvailableLabels: a.userLabelNames(),
		StrictLabels:    a.Config.InboxAnalyzer.StrictLabels,
	}
	go func() {
		a.GetErrorHandler().ShowProgress(a.ctx, fmt.Sprintf("Assisting %d email(s)…", len(msgs)))
		sug, err := analyzer.AssistReadManually(a.ctx, msgs, opts)
		a.GetErrorHandler().ClearProgress()
		if err != nil {
			a.GetErrorHandler().ShowWarning(a.ctx, "Could not get AI suggestions — showing the list only")
			return
		}
		m := make(map[string]services.ReadManuallySuggestion, len(sug))
		for _, s := range sug {
			m[s.ID] = s
		}
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state {
				return
			}
			state.rmSuggestions = m
			a.rebuildActionPlanTree(state)
		})
		a.GetErrorHandler().ShowSuccess(a.ctx, "AI suggestions ready")
	}()
}

// dropReadManually removes every id in ids from plan.ReadManually.
func dropReadManually(plan *services.ActionPlan, ids []string) {
	for _, id := range ids {
		plan.ReadManually = removeReadManuallyByID(plan.ReadManually, id)
	}
}

// acceptReadManuallySuggestions applies the AI-suggested action to the given read-manually email
// IDs. It buckets ids by (action,label) from state.rmSuggestions, skips "read"/missing suggestions,
// runs one bulk op per bucket in a worker goroutine, then drops the applied ids and re-renders.
// Called on the event loop; work runs in a goroutine (threading rules apply).
func (a *App) acceptReadManuallySuggestions(state *actionPlanState, ids []string) {
	if state == nil || state.rmSuggestions == nil {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Press the assist key first to get suggestions")
		return
	}
	type key struct{ action, label string }
	buckets := map[key][]string{}
	for _, id := range ids {
		s, ok := state.rmSuggestions[id]
		if !ok || s.Action == "read" {
			continue
		}
		buckets[key{s.Action, s.Label}] = append(buckets[key{s.Action, s.Label}], id)
	}
	if len(buckets) == 0 {
		go a.GetErrorHandler().ShowInfo(a.ctx, "No actionable suggestions here")
		return
	}
	emailService, _, labelService, _, _, _, _, _, _, _, _, _ := a.GetServices()
	go func() {
		var applied []string
		for k, kids := range buckets {
			if err := a.runActionPlanBulkOp(emailService, labelService, k.action, kids, k.label); err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("%s failed: %v", k.action, err))
				continue
			}
			applied = append(applied, kids...)
		}
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state {
				return
			}
			dropReadManually(state.plan, applied)
			a.rebuildActionPlanTree(state)
		})
		a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied %d suggestion(s)", len(applied)))
	}()
}
