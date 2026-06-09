package tui

import (
	"strings"
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

func TestActionPlanFooterText(t *testing.T) {
	onCat := actionPlanFooterText(true, "a", "archive", 7)
	if !strings.Contains(onCat, "[a]") || !strings.Contains(onCat, "archive 7") || !strings.Contains(onCat, "[^R]") {
		t.Fatalf("category footer wrong: %q", onCat)
	}
	onEmail := actionPlanFooterText(false, "a", "archive", 7)
	if !strings.Contains(onEmail, "[space]") || !strings.Contains(onEmail, "[^R]") {
		t.Fatalf("email footer wrong: %q", onEmail)
	}
}

func TestCheckedIDs(t *testing.T) {
	all := []string{"a", "b", "c"}
	excluded := map[string]bool{"b": true}
	got := checkedIDs(all, excluded)
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("want [a c], got %v", got)
	}
	if len(checkedIDs(all, map[string]bool{"a": true, "b": true, "c": true})) != 0 {
		t.Fatal("all excluded should yield empty")
	}
}

func TestActionPlanHeaderText(t *testing.T) {
	// Before the first batch (total==0): analyzing indicator, no batch counts.
	pre := actionPlanHeaderText("5 selected", 0, 0, true)
	if !strings.Contains(pre, "5 selected") || !strings.Contains(pre, "analyzing") {
		t.Fatalf("pre-batch header missing scope/indicator: %q", pre)
	}
	if strings.Contains(pre, "batch") {
		t.Fatalf("pre-batch header should not show batch counts: %q", pre)
	}
	// Mid-analysis: batch counts + analyzing status.
	mid := actionPlanHeaderText("23 unread (inbox)", 1, 3, true)
	if !strings.Contains(mid, "batch 1/3") || !strings.Contains(mid, "analyzing") {
		t.Fatalf("mid header wrong: %q", mid)
	}
	// Completed: batch counts + done status.
	done := actionPlanHeaderText("23 unread (inbox)", 3, 3, false)
	if !strings.Contains(done, "batch 3/3") || !strings.Contains(done, "done") {
		t.Fatalf("done header wrong: %q", done)
	}
}
