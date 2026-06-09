package db

import (
	"context"
	"testing"
)

func TestAnalyzerRulesStore_SaveListDelete(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, t.TempDir()+"/rules.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	rs := NewAnalyzerRulesStore(store)
	const acct = "user@example.com"

	if _, err := rs.SaveRule(ctx, acct, "Never trash emails from tldr.tech"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := rs.SaveRule(ctx, acct, "Archive newsletters automatically"); err != nil {
		t.Fatalf("save 2: %v", err)
	}

	rules, err := rs.ListRules(ctx, acct)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(rules))
	}

	other, err := rs.ListRules(ctx, "someone@else.com")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("want 0 rules for other account, got %d", len(other))
	}

	if err := rs.DeleteRule(ctx, acct, rules[0].ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rules, _ = rs.ListRules(ctx, acct)
	if len(rules) != 1 {
		t.Fatalf("want 1 rule after delete, got %d", len(rules))
	}
}

func TestAnalyzerRulesStore_RejectsEmpty(t *testing.T) {
	ctx := context.Background()
	store, _ := Open(ctx, t.TempDir()+"/rules.db")
	defer func() { _ = store.Close() }()
	rs := NewAnalyzerRulesStore(store)

	if _, err := rs.SaveRule(ctx, "", "x"); err == nil {
		t.Fatal("expected error for empty account")
	}
	if _, err := rs.SaveRule(ctx, "a@b.c", "   "); err == nil {
		t.Fatal("expected error for blank rule text")
	}
}
