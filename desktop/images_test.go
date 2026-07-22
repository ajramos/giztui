package main

import "testing"

func TestSanitizeImageURL(t *testing.T) {
	cases := []struct{ in, want string }{
		// Quoted-printable artifact: width=1200 → width\x1200. Control byte dropped.
		{"https://x/PUPPET.png?width\x1200&name=PUPPET.png", "https://x/PUPPET.png?width00&name=PUPPET.png"},
		{"https://x/a.png", "https://x/a.png"},
		{"https://x/a.png?w=1\x7f2", "https://x/a.png?w=12"},
		{"", ""},
	}
	for _, c := range cases {
		if got := sanitizeImageURL(c.in); got != c.want {
			t.Errorf("sanitizeImageURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
