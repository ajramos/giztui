package services

import (
	"context"
	"testing"

	"github.com/ajramos/giztui/internal/db"
)

// Regression: ExpandAllThreads/CollapseAllThreads must affect threads that have NO prior
// thread_state row (the common case). The old blanket `UPDATE ... WHERE account_email` only
// touched existing rows, so expand-all silently did nothing for fresh threads.
func TestExpandCollapseAllThreads_AffectsFreshThreads(t *testing.T) {
	ctx := context.Background()
	store, err := db.Open(ctx, t.TempDir()+"/store.sqlite3")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := NewThreadService(nil, store, nil)
	acc := "user@example.com"
	ids := []string{"t1", "t2", "t3"} // none have a thread_state row yet

	for _, id := range ids {
		if exp, _ := svc.IsThreadExpanded(ctx, acc, id); exp {
			t.Fatalf("thread %s should start collapsed (no row)", id)
		}
	}

	if err := svc.ExpandAllThreads(ctx, acc, ids); err != nil {
		t.Fatalf("ExpandAllThreads: %v", err)
	}
	for _, id := range ids {
		exp, err := svc.IsThreadExpanded(ctx, acc, id)
		if err != nil {
			t.Fatalf("IsThreadExpanded(%s): %v", id, err)
		}
		if !exp {
			t.Fatalf("thread %s should be expanded after ExpandAllThreads", id)
		}
	}

	if err := svc.CollapseAllThreads(ctx, acc, ids); err != nil {
		t.Fatalf("CollapseAllThreads: %v", err)
	}
	for _, id := range ids {
		if exp, _ := svc.IsThreadExpanded(ctx, acc, id); exp {
			t.Fatalf("thread %s should be collapsed after CollapseAllThreads", id)
		}
	}

	// An empty ID list is a harmless no-op.
	if err := svc.ExpandAllThreads(ctx, acc, nil); err != nil {
		t.Fatalf("ExpandAllThreads(nil): %v", err)
	}
}
