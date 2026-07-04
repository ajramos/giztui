package db

import (
	"context"
	"testing"
)

func TestDeterministicRulesStoreCRUD(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rules.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()
	s := NewDeterministicRulesStore(store)
	const acct = "user@example.com"

	// Save two rules; List must return them in CREATION order (first-match-wins).
	r1, err := s.SaveRule(ctx, acct, "from:newsletter@medium.com", "archive", "", 0)
	if err != nil {
		t.Fatalf("save r1: %v", err)
	}
	r2, err := s.SaveRule(ctx, acct, "from:jira@corp.com", "prompt", "", 7)
	if err != nil {
		t.Fatalf("save r2: %v", err)
	}
	rules, err := s.ListRules(ctx, acct)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rules) != 2 || rules[0].ID != r1.ID || rules[1].ID != r2.ID {
		t.Fatalf("want creation order [r1 r2], got %+v", rules)
	}
	if rules[1].PromptID != 7 || rules[1].Action != "prompt" {
		t.Fatalf("prompt rule fields lost: %+v", rules[1])
	}

	// Update r1 to a label rule.
	if err := s.UpdateRule(ctx, acct, r1.ID, "from:amazon.com", "label", "Compras", 0); err != nil {
		t.Fatalf("update: %v", err)
	}
	rules, _ = s.ListRules(ctx, acct)
	if rules[0].Query != "from:amazon.com" || rules[0].Action != "label" || rules[0].Label != "Compras" {
		t.Fatalf("update not persisted: %+v", rules[0])
	}

	// Gmail filter ID round-trip.
	if err := s.SetGmailFilterID(ctx, acct, r1.ID, "FILTER123"); err != nil {
		t.Fatalf("set filter id: %v", err)
	}
	rules, _ = s.ListRules(ctx, acct)
	if rules[0].GmailFilterID != "FILTER123" {
		t.Fatalf("filter id not persisted: %+v", rules[0])
	}

	// Account scoping.
	other, err := s.ListRules(ctx, "other@example.com")
	if err != nil || len(other) != 0 {
		t.Fatalf("account scoping broken: %v %+v", err, other)
	}

	// Delete.
	if err := s.DeleteRule(ctx, acct, r1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.DeleteRule(ctx, acct, r1.ID); err == nil {
		t.Fatal("second delete should fail with rule not found")
	}
	rules, _ = s.ListRules(ctx, acct)
	if len(rules) != 1 || rules[0].ID != r2.ID {
		t.Fatalf("want only r2 left, got %+v", rules)
	}
}

func TestDeterministicRulesStoreValidation(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rules.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()
	s := NewDeterministicRulesStore(store)

	if _, err := s.SaveRule(ctx, "", "from:x", "archive", "", 0); err == nil {
		t.Fatal("empty account must fail")
	}
	if _, err := s.SaveRule(ctx, "a@b.c", "  ", "archive", "", 0); err == nil {
		t.Fatal("empty query must fail")
	}
	if _, err := s.SaveRule(ctx, "a@b.c", "from:x", "explode", "", 0); err == nil {
		t.Fatal("unknown action must fail")
	}
}
