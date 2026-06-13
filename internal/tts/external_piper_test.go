package tts

import (
	"context"
	"errors"
	"testing"
)

func TestExternalPiper_Validation(t *testing.T) {
	s := &ExternalPiperSynthesizer{PiperPath: "/nonexistent/piper"}

	if _, err := s.Synthesize(context.Background(), "  ", SynthesizeOptions{ModelPath: "/nonexistent/model.onnx"}); !errors.Is(err, ErrEmptyText) {
		t.Fatalf("empty text should return ErrEmptyText, got %v", err)
	}
	if _, err := s.Synthesize(context.Background(), "hola", SynthesizeOptions{ModelPath: "/nonexistent/model.onnx"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("missing piper/model should return ErrNotConfigured, got %v", err)
	}
}
