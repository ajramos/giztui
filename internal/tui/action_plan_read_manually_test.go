package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/services"
)

func TestGroupReadManuallyBySender(t *testing.T) {
	msgs := []services.AnalyzerMessage{
		{ID: "1", From: "Ana García <ana@x.com>"},
		{ID: "2", From: "news@acme.com"},
		{ID: "3", From: "ana@x.com"},
		{ID: "4", From: "news@acme.com"},
		{ID: "5", From: "news@acme.com"},
	}
	groups := groupReadManuallyBySender(msgs)
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(groups))
	}
	if groups[0].senderKey != "news@acme.com" || len(groups[0].msgs) != 3 {
		t.Fatalf("group0 = %+v", groups[0])
	}
	if groups[1].senderKey != "ana@x.com" || len(groups[1].msgs) != 2 {
		t.Fatalf("group1 = %+v", groups[1])
	}
	if groups[1].msgs[0].ID != "1" || groups[1].msgs[1].ID != "3" {
		t.Fatalf("within-group order not preserved: %+v", groups[1].msgs)
	}
}

func TestSenderExpandKey(t *testing.T) {
	if got := senderExpandKey("news@acme.com"); got != "\x00read-manually:news@acme.com" {
		t.Fatalf("got %q", got)
	}
}

func TestDropReadManually(t *testing.T) {
	plan := &services.ActionPlan{ReadManually: []services.AnalyzerMessage{
		{ID: "1", From: "a@x"}, {ID: "2", From: "a@x"}, {ID: "3", From: "b@y"},
	}}
	dropReadManually(plan, []string{"1", "3"})
	if len(plan.ReadManually) != 1 || plan.ReadManually[0].ID != "2" {
		t.Fatalf("bucket after drop = %+v", plan.ReadManually)
	}
}

func TestReadManuallyLeafLabel(t *testing.T) {
	m := services.AnalyzerMessage{Subject: "Hello"}
	// no suggestion -> plain subject
	if got := readManuallyLeafLabel(m, services.ReadManuallySuggestion{}, false); got != "Hello" {
		t.Fatalf("plain: %q", got)
	}
	// read action with hint -> hint only, no "suggests:"
	if got := readManuallyLeafLabel(m, services.ReadManuallySuggestion{Hint: "fyi", Action: "read"}, true); got != "Hello — 💡 fyi" {
		t.Fatalf("read+hint: %q", got)
	}
	// archive suggestion -> hint + suggests
	got := readManuallyLeafLabel(m, services.ReadManuallySuggestion{Hint: "promo", Action: "archive"}, true)
	if got != "Hello — 💡 promo · suggests: "+actionVerbLabel("archive") {
		t.Fatalf("archive: %q", got)
	}
	// label suggestion -> appends label name
	gl := readManuallyLeafLabel(m, services.ReadManuallySuggestion{Hint: "hr", Action: "label", Label: "Work"}, true)
	if gl != "Hello — 💡 hr · suggests: "+actionVerbLabel("label")+" Work" {
		t.Fatalf("label: %q", gl)
	}
	// empty subject -> (no subject)
	if got := readManuallyLeafLabel(services.AnalyzerMessage{}, services.ReadManuallySuggestion{}, false); got != "(no subject)" {
		t.Fatalf("empty: %q", got)
	}
}
