package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
	"github.com/stretchr/testify/assert"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestBuildAnalyzerMessages(t *testing.T) {
	mk := func(id, subj, from, snippet string, unread bool) *gmailapi.Message {
		labels := []string{}
		if unread {
			labels = append(labels, "UNREAD")
		}
		return &gmailapi.Message{
			Id:       id,
			Snippet:  snippet,
			LabelIds: labels,
			Payload: &gmailapi.MessagePart{Headers: []*gmailapi.MessagePartHeader{
				{Name: "Subject", Value: subj},
				{Name: "From", Value: from},
			}},
		}
	}

	metas := []*gmailapi.Message{
		mk("m1", "Hello", "a@x.com", "snip1", true),
		mk("m2", "Read one", "b@x.com", "snip2", false), // read → excluded
		mk("m3", "World", "c@x.com", "snip3", true),
		nil, // defensive: nil entries are skipped
	}

	got := buildAnalyzerMessages(metas)
	assert.Len(t, got, 2)
	assert.Equal(t, services.AnalyzerMessage{ID: "m1", Subject: "Hello", From: "a@x.com", Snippet: "snip1"}, got[0])
	assert.Equal(t, "m3", got[1].ID)
}

func TestRenderActionPlanText(t *testing.T) {
	plan := &services.ActionPlan{
		TotalAnalyzed: 30,
		BatchesTotal:  1,
		BatchesDone:   1,
		Categories: []services.ActionPlanCategory{
			{Name: "Newsletters", Priority: "low", Description: "marketing", Action: "archive", MessageIDs: []string{"m1", "m2"}},
			{Name: "Follow up", Priority: "high", Description: "needs reply", Action: "label", Label: "needs-reply", MessageIDs: []string{"m3"}},
		},
		ReadManually: []services.AnalyzerMessage{{ID: "m4", Subject: "Budget", From: "cfo@x.com"}},
	}

	out := renderActionPlanText(plan, 1)

	assert.Contains(t, out, "Newsletters")
	assert.Contains(t, out, "Archive 2")
	assert.Contains(t, out, "[a]")
	assert.Contains(t, out, "needs-reply")
	assert.Contains(t, out, "Read manually (1)")
	assert.Contains(t, out, "Budget")
	assert.Contains(t, out, "▸")
	// The marker must land on the SELECTED category (index 1 = "Follow up"), not index 0.
	assert.Contains(t, out, "▸ [l] Label 1 Follow up")
	assert.NotContains(t, out, "▸ [a]")
}

func TestActionKeyHint(t *testing.T) {
	a := &App{}
	a.Keys.Archive = "a"
	a.Keys.ToggleRead = "t"
	a.Keys.Trash = "d"
	a.Keys.ManageLabels = "l"
	assert.Equal(t, "a", a.actionKeyHint("archive"))
	assert.Equal(t, "t", a.actionKeyHint("mark_read"))
	assert.Equal(t, "d", a.actionKeyHint("trash"))
	assert.Equal(t, "l", a.actionKeyHint("label"))
	assert.Equal(t, "", a.actionKeyHint("none"))
}
