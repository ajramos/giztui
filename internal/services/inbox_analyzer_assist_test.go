package services

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestParseAssistResponse(t *testing.T) {
	batch := []string{"a", "b", "c", "d"}
	available := map[string]string{"work": "Work"}
	raw := `[
	  {"id":"a","hint":"promo","action":"archive"},
	  {"id":"b","hint":"HR notice","action":"label","label":"work"},
	  {"id":"c","hint":"unknown label","action":"label","label":"Nope"},
	  {"id":"d","hint":"just read","action":"weird"}
	]`
	got := parseAssistResponse(raw, batch, available, true)
	want := []ReadManuallySuggestion{
		{ID: "a", Hint: "promo", Action: "archive"},
		{ID: "b", Hint: "HR notice", Action: "label", Label: "Work"},
		{ID: "c", Hint: "unknown label", Action: "read"},
		{ID: "d", Hint: "just read", Action: "read"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

func TestParseAssistResponse_MissingIDReconciled(t *testing.T) {
	batch := []string{"a", "b"}
	raw := `[{"id":"a","hint":"x","action":"trash"}]`
	got := parseAssistResponse(raw, batch, map[string]string{}, false)
	if len(got) != 2 || got[1].ID != "b" || got[1].Action != "read" {
		t.Fatalf("missing id not reconciled to read: %+v", got)
	}
}

func TestParseAssistResponse_UnknownLabelDegradesToRead(t *testing.T) {
	got := parseAssistResponse(`[{"id":"a","action":"label","label":"Ghost"}]`, []string{"a"}, map[string]string{}, false)
	if got[0].Action != "read" {
		t.Fatalf("unknown label should degrade to read, got %q", got[0].Action)
	}
}

func TestAssistReadManually_EndToEnd(t *testing.T) {
	ai := &mockAIService{}
	ai.On("ApplyCustomPromptStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(`[{"id":"m1","hint":"promo","action":"archive"},{"id":"m2","hint":"read me","action":"read"}]`, nil).Once()

	svc := NewInboxAnalyzerService(ai)
	msgs := []AnalyzerMessage{{ID: "m1", From: "a@x.com", Subject: "S1"}, {ID: "m2", From: "b@x.com", Subject: "S2"}}
	got, err := svc.AssistReadManually(context.Background(), msgs, InboxAnalyzerOptions{BatchSize: 50, MaxBatches: 10})
	if err != nil {
		t.Fatalf("assist: %v", err)
	}
	if len(got) != 2 || got[0].Action != "archive" || got[1].Action != "read" {
		t.Fatalf("unexpected: %+v", got)
	}
	ai.AssertExpectations(t)
}

func TestAssistReadManually_NoAIService(t *testing.T) {
	svc := NewInboxAnalyzerService(nil)
	if _, err := svc.AssistReadManually(context.Background(), []AnalyzerMessage{{ID: "x"}}, InboxAnalyzerOptions{}); err == nil {
		t.Fatal("expected error when AI service is nil")
	}
}
