package main

import "testing"

func TestSanitizeImageURL(t *testing.T) {
	cases := []struct{ in, want string }{
		// Quoted-printable artifact: width=1200 → width\x1200. Control byte dropped.
		{"https://x/PUPPET.png?width\x1200&name=PUPPET.png", "https://x/PUPPET.png?width00&name=PUPPET.png"},
		// QP escape of byte 0x80 → U+FFFD after the charset decode: width=800 →
		// width�0. The replacement char is dropped, recovering the image.
		{"https://x/Confirm.png?width�0&upscale=true&name=Confirm.png", "https://x/Confirm.png?width0&upscale=true&name=Confirm.png"},
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
