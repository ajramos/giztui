package services

import (
	"testing"
	"time"
)

func TestAutoRefreshServiceState(t *testing.T) {
	s := NewAutoRefreshService(nil, false, 5*time.Minute, time.Minute)
	if s.IsEnabled() {
		t.Error("should start disabled")
	}
	s.SetEnabled(true)
	if !s.IsEnabled() {
		t.Error("SetEnabled(true) failed")
	}
	if s.Interval() != 5*time.Minute {
		t.Errorf("Interval() = %v, want 5m", s.Interval())
	}
	// Below minimum is clamped.
	s.SetInterval(10 * time.Second)
	if s.Interval() != time.Minute {
		t.Errorf("SetInterval clamp = %v, want 1m", s.Interval())
	}
	// Zero/negative is ignored (keeps previous).
	s.SetInterval(0)
	if s.Interval() != time.Minute {
		t.Errorf("SetInterval(0) changed value to %v", s.Interval())
	}
}

func TestDiffNewIDs(t *testing.T) {
	known := []string{"b", "c", "d"}
	// Fetched is newest-first; "a" is new, "b"/"c" are known.
	got := diffNewIDs([]string{"a", "b", "c"}, known)
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("diffNewIDs = %v, want [a]", got)
	}
	// Nothing new.
	if got := diffNewIDs([]string{"b", "c"}, known); len(got) != 0 {
		t.Errorf("diffNewIDs = %v, want empty", got)
	}
	// Order preserved (newest-first) for multiple new IDs.
	if got := diffNewIDs([]string{"x", "y", "b"}, known); len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("diffNewIDs = %v, want [x y]", got)
	}
	// Empty known => all fetched are new.
	if got := diffNewIDs([]string{"a", "b"}, nil); len(got) != 2 {
		t.Errorf("diffNewIDs = %v, want 2", got)
	}
}
