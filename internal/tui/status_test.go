package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/services"
)

func TestStatusBaselineAutoRefreshIndicator(t *testing.T) {
	a := &App{}
	a.autoRefreshService = services.NewAutoRefreshService(nil, false, time.Minute, time.Minute)

	// Disabled => no ⟳.
	if strings.Contains(a.statusBaseline(), "⟳") {
		t.Error("disabled auto-refresh should not show ⟳")
	}

	// Enabled => ⟳ present.
	a.autoRefreshService.SetEnabled(true)
	if !strings.Contains(a.statusBaseline(), "⟳") {
		t.Error("enabled auto-refresh should show ⟳")
	}

	// Pending count => 📬N present.
	a.SetPendingNewCount(3)
	if !strings.Contains(a.statusBaseline(), "📬3") {
		t.Errorf("pending count should show 📬3, got %q", a.statusBaseline())
	}
}
