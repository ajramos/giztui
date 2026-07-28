package tui

import (
	"testing"
	"time"
)

// parseICalInstant must honor the TZID: a floating value in Australia/Melbourne is NOT UTC.
// Melbourne in July is AEST (UTC+10), so 17:00 there == 07:00 UTC — independent of the viewer.
func TestParseICalInstant(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantUTC  time.Time
		dateOnly bool
		ok       bool
	}{
		{"tzid melbourne", ";TZID=Australia/Melbourne:20260731T170000",
			time.Date(2026, 7, 31, 7, 0, 0, 0, time.UTC), false, true},
		{"utc Z stays utc", "20260115T100000Z",
			time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), false, true},
		{"tzid new york winter", ";TZID=America/New_York:20260115T090000",
			time.Date(2026, 1, 15, 14, 0, 0, 0, time.UTC), false, true}, // EST = UTC-5
		{"date only all-day", ";VALUE=DATE:20260731", time.Time{}, true, true},
		{"garbage", "not-a-date", time.Time{}, false, false},
		{"empty", "", time.Time{}, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, dateOnly, ok := parseICalInstant(c.in)
			if ok != c.ok || dateOnly != c.dateOnly {
				t.Fatalf("ok=%v dateOnly=%v, want ok=%v dateOnly=%v", ok, dateOnly, c.ok, c.dateOnly)
			}
			if c.ok && !c.dateOnly && !got.UTC().Equal(c.wantUTC) {
				t.Fatalf("instant = %s, want %s", got.UTC().Format(time.RFC3339), c.wantUTC.Format(time.RFC3339))
			}
		})
	}
}

// End-to-end: the real invite (17:00 Melbourne) must display as 9:00 AM for a viewer in Madrid
// (CEST = UTC+2 in July). This is the exact bug the user reported.
func TestFormatMeetingTimeRange_ConvertsToLocal(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Skipf("tz data unavailable: %v", err)
	}
	saved := time.Local
	time.Local = madrid
	defer func() { time.Local = saved }()

	got := formatMeetingTimeRange(
		";TZID=Australia/Melbourne:20260731T170000",
		";TZID=Australia/Melbourne:20260731T172500",
	)
	want := "Fri, Jul 31 2026, 9:00 AM - 9:25 AM"
	if got != want {
		t.Fatalf("meeting time = %q, want %q", got, want)
	}
}
