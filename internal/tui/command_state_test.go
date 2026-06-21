package tui

import "testing"

func TestCommandState_AddToHistory(t *testing.T) {
	var c commandState
	c.addToHistory("a")
	c.addToHistory("b")
	c.addToHistory("b") // consecutive dup → skipped
	c.addToHistory("")  // empty → skipped
	if len(c.history) != 2 || c.history[0] != "a" || c.history[1] != "b" {
		t.Fatalf("history = %v, want [a b]", c.history)
	}
	if c.historyIndex != 2 {
		t.Errorf("historyIndex = %d, want 2 (end)", c.historyIndex)
	}
	var big commandState
	for i := 0; i < 150; i++ {
		big.addToHistory(string(rune('A'+i%26)) + string(rune('0'+i%10)) + string(rune(i)))
	}
	if len(big.history) != 100 {
		t.Errorf("history len = %d, want 100 (capped)", len(big.history))
	}
}

func TestCommandState_ResetHistoryCursor(t *testing.T) {
	c := commandState{history: []string{"x", "y", "z"}}
	c.historyIndex = 0
	c.resetHistoryCursor()
	if c.historyIndex != 3 {
		t.Errorf("historyIndex = %d, want 3 (len)", c.historyIndex)
	}
}

func TestCommandState_HistoryUpDown(t *testing.T) {
	c := commandState{history: []string{"first", "second", "third"}}
	c.resetHistoryCursor()

	if txt, ok := c.historyUp(); !ok || txt != "third" {
		t.Fatalf("up #1 = %q,%v, want third,true", txt, ok)
	}
	if txt, ok := c.historyUp(); !ok || txt != "second" {
		t.Fatalf("up #2 = %q,%v, want second,true", txt, ok)
	}
	if txt, ok := c.historyUp(); !ok || txt != "first" {
		t.Fatalf("up #3 = %q,%v, want first,true", txt, ok)
	}
	if _, ok := c.historyUp(); ok {
		t.Fatal("up at top should return ok=false")
	}

	if txt, ok := c.historyDown(); !ok || txt != "second" {
		t.Fatalf("down #1 = %q,%v, want second,true", txt, ok)
	}
	if txt, ok := c.historyDown(); !ok || txt != "third" {
		t.Fatalf("down #2 = %q,%v, want third,true", txt, ok)
	}
	if txt, ok := c.historyDown(); !ok || txt != "" {
		t.Fatalf("down past end = %q,%v, want \"\",true", txt, ok)
	}
	if c.historyIndex != 3 {
		t.Errorf("cursor = %d, want 3 (parked at len)", c.historyIndex)
	}
}
