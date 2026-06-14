package tts

import (
	"context"
	"testing"
)

func TestSaySynthesizer_EmptyText(t *testing.T) {
	s := &SaySynthesizer{}
	if _, err := s.Synthesize(context.Background(), "   ", SynthesizeOptions{}); err != ErrEmptyText {
		t.Fatalf("empty text: got %v, want ErrEmptyText", err)
	}
}
