package tui

import (
	"sync"
	"testing"
)

func TestBulkState_Basics(t *testing.T) {
	b := newBulkState()
	if b.isMode() || b.count() != 0 || b.isSelected("x") {
		t.Fatal("fresh bulkState must be empty/off")
	}
	b.setMode(true)
	if !b.isMode() {
		t.Fatal("setMode(true)")
	}
	b.add("a")
	b.add("b")
	if b.count() != 2 || !b.isSelected("a") {
		t.Fatalf("after add: count=%d", b.count())
	}
	if got := b.toggle("a"); got || b.isSelected("a") {
		t.Fatal("toggle('a') should remove it and return false")
	}
	if got := b.toggle("c"); !got || !b.isSelected("c") {
		t.Fatal("toggle('c') should add it and return true")
	}
	b.remove("b")
	if b.isSelected("b") {
		t.Fatal("remove('b')")
	}
	ids := b.ids()
	if len(ids) != 1 || ids[0] != "c" {
		t.Fatalf("ids() = %v, want [c]", ids)
	}
	// ids() is an independent copy.
	ids[0] = "mutated"
	if !b.isSelected("c") {
		t.Fatal("mutating ids() result must not affect state")
	}
	b.clear()
	if b.count() != 0 {
		t.Fatal("clear()")
	}
}

func TestBulkState_Race(t *testing.T) {
	b := newBulkState()
	var wg sync.WaitGroup
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for i := 0; i < len(ids); i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				b.add(id)
				_ = b.count()
				_ = b.ids()
				_ = b.isSelected(id)
				b.remove(id)
			}
		}(ids[i])
	}
	wg.Wait()
}
