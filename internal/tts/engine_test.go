package tts

import (
	"runtime"
	"testing"
)

func TestResolveEngine(t *testing.T) {
	// Explicit values are honored regardless of OS.
	if got := ResolveEngine("say"); got != "say" {
		t.Fatalf(`ResolveEngine("say") = %q, want "say"`, got)
	}
	if got := ResolveEngine("piper"); got != "piper" {
		t.Fatalf(`ResolveEngine("piper") = %q, want "piper"`, got)
	}

	// "auto"/"" resolve by OS: macOS → say, everything else → piper.
	want := "piper"
	if runtime.GOOS == "darwin" {
		want = "say"
	}
	for _, in := range []string{"", "auto", "bogus"} {
		if got := ResolveEngine(in); got != want {
			t.Fatalf("ResolveEngine(%q) = %q, want %q (GOOS=%s)", in, got, want, runtime.GOOS)
		}
	}
}
