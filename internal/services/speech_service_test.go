package services

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ajramos/giztui/internal/tts"
)

type stubSynth struct{ called bool }

func (s *stubSynth) Synthesize(ctx context.Context, text string, opts tts.SynthesizeOptions) (*tts.SynthesisResult, error) {
	s.called = true
	return &tts.SynthesisResult{AudioPath: "", Engine: "stub"}, nil
}

type stubPlayer struct {
	mu     sync.Mutex
	played bool
}

func (p *stubPlayer) Play(ctx context.Context, audioPath string) error {
	p.mu.Lock()
	p.played = true
	p.mu.Unlock()
	return nil
}

func TestSpeechService_IsConfigured(t *testing.T) {
	dir := t.TempDir()
	piper := filepath.Join(dir, "piper")
	model := filepath.Join(dir, "m.onnx")
	_ = os.WriteFile(piper, []byte("x"), 0600)
	_ = os.WriteFile(model, []byte("x"), 0600)

	s := NewSpeechService(&stubSynth{}, &stubPlayer{}, "piper", piper, model)
	if !s.IsConfigured() {
		t.Fatal("should be configured when both paths exist")
	}
	s2 := NewSpeechService(&stubSynth{}, &stubPlayer{}, "piper", piper, filepath.Join(dir, "missing.onnx"))
	if s2.IsConfigured() {
		t.Fatal("should NOT be configured when the model is missing")
	}
}

func TestSpeechService_SpeakStop(t *testing.T) {
	dir := t.TempDir()
	piper := filepath.Join(dir, "piper")
	model := filepath.Join(dir, "m.onnx")
	_ = os.WriteFile(piper, []byte("x"), 0600)
	_ = os.WriteFile(model, []byte("x"), 0600)
	syn := &stubSynth{}
	pl := &stubPlayer{}
	s := NewSpeechService(syn, pl, "piper", piper, model)

	if err := s.Speak(context.Background(), "hola"); err != nil {
		t.Fatalf("Speak error: %v", err)
	}
	if !syn.called || !pl.played {
		t.Fatal("Speak should synthesize then play")
	}
	s.Stop()
}
