package desktop

import "testing"

func TestResolveICSDateTime(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		// Zoned: 17:00 Australia/Sydney (UTC+10 in July) == 07:00 UTC.
		{"tzid_sydney", ";TZID=Australia/Sydney:20260731T170000", "2026-07-31T07:00:00Z"},
		// Zoned: 09:00 Europe/Madrid (CEST, UTC+2) == 07:00 UTC.
		{"tzid_madrid", ";TZID=Europe/Madrid:20260731T090000", "2026-07-31T07:00:00Z"},
		// Already UTC.
		{"utc_z", "20260731T070000Z", "2026-07-31T07:00:00Z"},
		// All-day: kept as raw digits (frontend shows the date, no time).
		{"all_day_value_date", ";VALUE=DATE:20260731", "20260731"},
		{"all_day_bare", "20260731", "20260731"},
		// Unknown zone: falls through to floating (wall clock as UTC).
		{"floating", "20260731T170000", "2026-07-31T17:00:00Z"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveICSDateTime(c.in); got != c.want {
				t.Errorf("resolveICSDateTime(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
