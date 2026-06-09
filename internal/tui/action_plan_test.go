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

	// Drive the renderer with configured keys via App.actionKeyHint (proves body hints
	// honor user keybindings, not hardcoded defaults).
	a := &App{}
	a.Keys.Archive, a.Keys.ToggleRead, a.Keys.Trash, a.Keys.ManageLabels = "a", "t", "d", "l"
	out := renderActionPlanText(plan, 1, a.actionKeyHint)

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

func msgWith(id, from, subj string, unread bool) *gmailapi.Message {
	m := &gmailapi.Message{Id: id, Snippet: "snip", Payload: &gmailapi.MessagePart{
		Headers: []*gmailapi.MessagePartHeader{
			{Name: "From", Value: from}, {Name: "Subject", Value: subj},
		},
	}}
	if unread {
		m.LabelIds = []string{"UNREAD"}
	}
	return m
}

func TestBuildAnalyzerMessagesForSelection(t *testing.T) {
	metas := []*gmailapi.Message{
		msgWith("1", "a@x.com", "S1", true),
		msgWith("2", "b@x.com", "S2", false), // read, but explicitly selected
		msgWith("3", "c@x.com", "S3", true),
	}
	selected := map[string]bool{"2": true, "3": true}

	got := buildAnalyzerMessagesForSelection(metas, selected)
	if len(got) != 2 {
		t.Fatalf("want 2 selected (incl. read), got %d", len(got))
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids["2"] || !ids["3"] {
		t.Fatalf("expected ids 2 and 3, got %+v", ids)
	}
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
