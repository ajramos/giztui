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
