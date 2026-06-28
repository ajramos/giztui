package tui

import (
	"sync"
	"testing"
)

func TestFocusState_Basics(t *testing.T) {
	var f focusState
	if f.cur() != "" || f.is("list") || f.viewName() != "" {
		t.Fatal("zero focusState must be empty")
	}
	f.set("labels")
	if f.cur() != "labels" || !f.is("labels") || f.is("list") {
		t.Fatalf("set/is: cur=%q", f.cur())
	}
	f.setView("thread")
	if f.viewName() != "thread" {
		t.Fatalf("view=%q", f.viewName())
	}
}

func TestFocusState_Race(t *testing.T) {
	var f focusState
	names := []string{"list", "text", "labels", "summary"}
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				f.set(name)
				_ = f.cur()
				_ = f.is(name)
				f.setView("flat")
				_ = f.viewName()
			}
		}(names[n%len(names)])
	}
	wg.Wait()
}
