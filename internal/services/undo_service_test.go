package services

import (
	"context"
	"testing"
)

// fakeUndoGmailClient records the low-level Gmail calls the undo service makes
// so tests can assert HOW an operation is reversed (untrash vs. label modify).
type fakeUndoGmailClient struct {
	untrashed  []string
	applied    []string // "messageID:labelID"
	removed    []string // "messageID:labelID"
	untrashErr error
}

func (f *fakeUndoGmailClient) ApplyLabel(messageID, labelID string) error {
	f.applied = append(f.applied, messageID+":"+labelID)
	return nil
}
func (f *fakeUndoGmailClient) RemoveLabel(messageID, labelID string) error {
	f.removed = append(f.removed, messageID+":"+labelID)
	return nil
}
func (f *fakeUndoGmailClient) UntrashMessage(messageID string) error {
	if f.untrashErr != nil {
		return f.untrashErr
	}
	f.untrashed = append(f.untrashed, messageID)
	return nil
}

// TestUndoTrash_UsesUntrashEndpoint guards the fix for the "undo trash then
// reopen and the email is gone" bug: Gmail's TRASH label cannot be removed via
// messages.modify, so the undo MUST call messages.untrash — not a label modify —
// or the message stays trashed on the server and disappears on the next sync.
func TestUndoTrash_UsesUntrashEndpoint(t *testing.T) {
	fake := &fakeUndoGmailClient{}
	svc := &UndoServiceImpl{gmailClient: fake}
	ctx := context.Background()

	action := &UndoableAction{
		Type:       UndoActionTrash,
		MessageIDs: []string{"m1", "m2"},
		PrevState: map[string]ActionState{
			"m1": {Labels: []string{"INBOX", "UNREAD"}, IsInInbox: true},
			"m2": {Labels: []string{"INBOX"}, IsInInbox: true},
		},
		Description: "Trash message",
	}
	if err := svc.RecordAction(ctx, action); err != nil {
		t.Fatalf("RecordAction: %v", err)
	}

	res, err := svc.UndoLastAction(ctx)
	if err != nil {
		t.Fatalf("UndoLastAction: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got errors: %v", res.Errors)
	}

	// Both messages restored via the untrash endpoint...
	if len(fake.untrashed) != 2 || fake.untrashed[0] != "m1" || fake.untrashed[1] != "m2" {
		t.Fatalf("expected untrash of [m1 m2], got %v", fake.untrashed)
	}
	// ...and NOT via a label modify (the buggy path that never removes TRASH).
	if len(fake.removed) != 0 {
		t.Fatalf("undo trash must not RemoveLabel, got %v", fake.removed)
	}
	if len(fake.applied) != 0 {
		t.Fatalf("undo trash must not ApplyLabel, got %v", fake.applied)
	}

	// Single-level undo is consumed on success.
	if svc.HasUndoableAction() {
		t.Fatalf("expected undo history cleared after success")
	}
}

// TestUndoTrash_PropagatesError ensures a failed untrash surfaces as a failed
// undo (so the UI can report it) rather than silently "succeeding".
func TestUndoTrash_PropagatesError(t *testing.T) {
	fake := &fakeUndoGmailClient{untrashErr: context.DeadlineExceeded}
	svc := &UndoServiceImpl{gmailClient: fake}
	ctx := context.Background()

	action := &UndoableAction{
		Type:       UndoActionTrash,
		MessageIDs: []string{"m1"},
		PrevState:  map[string]ActionState{"m1": {Labels: []string{"INBOX"}}},
	}
	if err := svc.RecordAction(ctx, action); err != nil {
		t.Fatalf("RecordAction: %v", err)
	}

	res, err := svc.UndoLastAction(ctx)
	if err != nil {
		t.Fatalf("UndoLastAction returned transport error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected undo to report failure when untrash fails")
	}
	if len(res.Errors) == 0 {
		t.Fatalf("expected error detail on failed undo")
	}
	// On failure the action is retained so the user can retry.
	if !svc.HasUndoableAction() {
		t.Fatalf("expected undo history retained after failure")
	}
}
