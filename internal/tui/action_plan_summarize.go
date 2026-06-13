package tui

import (
	"fmt"
	"strings"

	"github.com/ajramos/giztui/internal/services"
	tcell "github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	gmailapi "google.golang.org/api/gmail/v1"
)

// buildSummarizeInput renders the category's emails into a numbered blob for the AI digest:
// "N. Subject: … | From: …\n   <body up to limit, or snippet if no body>". metaByID supplies
// subject/from/snippet; bodies (id->plain text) supplies the body when available.
func buildSummarizeInput(ids []string, bodies map[string]string, metaByID map[string]*gmailapi.Message, limit int) string {
	var b strings.Builder
	for i, id := range ids {
		subject, from, snippet := "(unknown)", "", ""
		if m := metaByID[id]; m != nil {
			subject = extractHeaderValue(m, "Subject")
			from = extractHeaderValue(m, "From")
			snippet = m.Snippet
		}
		body := bodies[id]
		if strings.TrimSpace(body) == "" {
			body = snippet
		}
		fmt.Fprintf(&b, "%d. Subject: %s | From: %s\n   %s\n", i+1, subject, from, truncateForSummary(body, limit))
	}
	return b.String()
}

// truncateForSummary collapses whitespace and cuts to limit runes (limit <= 0 → no cut).
func truncateForSummary(text string, limit int) string {
	collapsed := strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	if limit <= 0 || len([]rune(collapsed)) <= limit {
		return collapsed
	}
	return string([]rune(collapsed)[:limit])
}

// dispatchActionPlanSummarize body-swaps the tree for an in-place panel showing a combined AI
// digest of the focused category's checked emails. Esc returns to the tree.
func (a *App) dispatchActionPlanSummarize(state *actionPlanState) {
	cat := a.currentActionPlanCategory(state)
	if cat == nil {
		return
	}
	ids := checkedIDs(cat.MessageIDs, state.excluded)
	if len(ids) == 0 {
		go a.GetErrorHandler().ShowWarning(a.ctx, "All emails in this category are excluded — nothing to summarize")
		return
	}

	colors := a.GetComponentColors("ai")
	view := tview.NewTextView().SetWrap(true).SetWordWrap(true)
	view.SetBackgroundColor(colors.Background.Color())
	view.SetTextColor(colors.Text.Color())
	view.SetText(fmt.Sprintf("⏳ Summarizing %d email(s)…", len(ids)))

	restore := func() {
		state.container.RemoveItem(view)
		state.container.RemoveItem(state.footer)
		state.container.AddItem(state.tree, 0, 1, true)
		state.container.AddItem(state.footer, 1, 0, false)
		a.currentFocus = "action_plan"
		a.SetFocus(state.tree)
		a.renderActionPlanPanel(state)
	}
	view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEscape {
			restore()
			return nil
		}
		return ev
	})

	state.container.RemoveItem(state.tree)
	state.container.RemoveItem(state.footer)
	state.container.AddItem(view, 0, 1, true)
	state.container.AddItem(state.footer, 1, 0, false)
	state.container.SetTitle(fmt.Sprintf(" 📝 Digest of %q ", cat.Name))
	state.footer.SetText(" ↑/↓ scroll  |  Esc to go back ")
	a.currentFocus = "action_plan_summary"
	a.SetFocus(view)

	limit := 1000
	if a.Config != nil {
		limit = a.Config.InboxAnalyzer.BodyCharLimit
	}
	emailService, aiService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	go func() {
		var bodies map[string]string
		if emailService != nil {
			bodies, _ = emailService.GetMessagePlainTexts(a.ctx, ids, 0)
		}
		blob := buildSummarizeInput(ids, bodies, state.metaByID, limit)
		if aiService == nil {
			a.QueueUpdateDraw(func() {
				if a.actionPlanState == state {
					view.SetText("⚠️ AI service not available")
				}
			})
			return
		}
		res, err := aiService.GenerateSummary(a.ctx, blob, services.SummaryOptions{})
		a.QueueUpdateDraw(func() {
			if a.actionPlanState != state || a.ctx.Err() != nil {
				return
			}
			if err != nil {
				view.SetText(fmt.Sprintf("⚠️ Could not summarize: %v", err))
				return
			}
			view.SetText(a.renderPromptResult(res.Summary))
		})
	}()
}
