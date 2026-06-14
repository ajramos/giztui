package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ajramos/giztui/internal/tts"
)

// blockingSynth blocks until the context is cancelled, then returns an error — emulating the
// external `say`/piper process being killed by Stop().
type blockingSynth struct{ started chan struct{} }

func (b *blockingSynth) Synthesize(ctx context.Context, text string, opts tts.SynthesizeOptions) (*tts.SynthesisResult, error) {
	close(b.started)
	<-ctx.Done()
	return nil, fmt.Errorf("tts: say failed: signal: killed")
}

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

// Stopping playback (Stop() cancels the context, killing the external process) must NOT surface as
// a "TTS failed" error — it is a user-requested stop.
func TestSpeechService_StopSuppressesCancelError(t *testing.T) {
	dir := t.TempDir()
	piper := filepath.Join(dir, "piper")
	model := filepath.Join(dir, "m.onnx")
	_ = os.WriteFile(piper, []byte("x"), 0600)
	_ = os.WriteFile(model, []byte("x"), 0600)

	started := make(chan struct{})
	s := NewSpeechService(&blockingSynth{started: started}, &stubPlayer{}, "piper", piper, model)

	done := make(chan error, 1)
	go func() { done <- s.Speak(context.Background(), "hola") }()
	<-started
	s.Stop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("user-requested Stop must not surface as an error, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Speak did not return after Stop")
	}
}
