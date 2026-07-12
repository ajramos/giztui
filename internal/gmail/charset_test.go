package gmail

import "testing"

// TestDecodeToUTF8 covers the charset-decoding heuristic used on message bodies.
// The important case is a body that is actually UTF-8 but declares a single-byte
// charset (ISO-8859-1 / windows-1252) — a common sender mislabel. Blindly applying
// the declared charset to real UTF-8 bytes yields mojibake ("día" → "dÃ­as"), so
// valid UTF-8 must be trusted over a conflicting label.
func TestDecodeToUTF8(t *testing.T) {
	utf8Dias := []byte{0x64, 0xC3, 0xAD, 0x61, 0x73} // "días" in UTF-8
	latin1Dias := []byte{0x64, 0xED, 0x61, 0x73}     // "días" in ISO-8859-1 (í = 0xED)

	cases := []struct {
		name  string
		raw   []byte
		label string
		want  string
	}{
		{"utf8 mislabeled as iso-8859-1", utf8Dias, "iso-8859-1", "días"},
		{"utf8 mislabeled as windows-1252", utf8Dias, "windows-1252", "días"},
		{"genuine iso-8859-1 is converted", latin1Dias, "iso-8859-1", "días"},
		{"utf8 correctly labeled", utf8Dias, "utf-8", "días"},
		{"no charset label", utf8Dias, "", "días"},
		{"plain ascii, any label", []byte("hello"), "iso-8859-1", "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decodeToUTF8(tc.raw, tc.label); got != tc.want {
				t.Fatalf("decodeToUTF8(%q, %q) = %q, want %q", tc.raw, tc.label, got, tc.want)
			}
		})
	}
}
