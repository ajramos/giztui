package tts

import "testing"

func TestDetectLanguage_Whitelisted(t *testing.T) {
	cands := []string{"en", "es"}
	cases := map[string]string{
		"Hello, your Uber trip receipt is attached. Thanks for riding with us today.":   "en",
		"Hola, gracias por viajar con nosotros. Adjuntamos tu recibo del trayecto.":     "es",
		"Reach your goals faster with new learning resources from Microsoft Learn now.": "en",
	}
	for text, want := range cases {
		if got := DetectLanguage(text, cands); got != want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", text[:20], got, want)
		}
	}
}

func TestDetectLanguage_SingleCandidateSkipsDetection(t *testing.T) {
	// With one candidate, return it without detecting (even for clearly-other-language text).
	if got := DetectLanguage("Hello there, this is clearly English text.", []string{"es"}); got != "es" {
		t.Errorf("single candidate should be returned as-is, got %q", got)
	}
}

func TestDetectLanguage_NoCandidates(t *testing.T) {
	if got := DetectLanguage("anything", nil); got != "" {
		t.Errorf("no candidates should yield empty, got %q", got)
	}
	// Unknown ISO codes are ignored → no usable candidates → "".
	if got := DetectLanguage("anything", []string{"xx", "zz"}); got != "" {
		t.Errorf("unknown candidates should yield empty, got %q", got)
	}
}
