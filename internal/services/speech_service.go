package services

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/tts"
)

// SpeechServiceImpl composes a Synthesizer + Player and manages start/stop.
type SpeechServiceImpl struct {
	synth     tts.Synthesizer
	player    tts.Player
	engine    string // "piper" (default) or "say"
	piperPath string
	modelPath string

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewSpeechService(synth tts.Synthesizer, player tts.Player, engine, piperPath, modelPath string) *SpeechServiceImpl {
	return &SpeechServiceImpl{synth: synth, player: player, engine: engine, piperPath: piperPath, modelPath: modelPath}
}

// IsConfigured reports whether the active engine can synthesize. For "say" (macOS) that means the
// command exists; for "piper" (the default) the binary and model files must both be present.
func (s *SpeechServiceImpl) IsConfigured() bool {
	if s.engine == "say" {
		_, err := exec.LookPath("say")
		return err == nil
	}
	return fileExists(s.piperPath) && fileExists(s.modelPath)
}

func (s *SpeechServiceImpl) IsSpeaking() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

func (s *SpeechServiceImpl) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *SpeechServiceImpl) Speak(ctx context.Context, text string) error {
	if strings.TrimSpace(text) == "" {
		return tts.ErrEmptyText
	}
	if !s.IsConfigured() {
		return tts.ErrNotConfigured
	}
	s.Stop() // cancel any in-flight speech
	cctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
		cancel()
	}()

	res, err := s.synth.Synthesize(cctx, text, tts.SynthesizeOptions{ModelPath: s.modelPath})
	if err != nil {
		if cctx.Err() != nil {
			return nil // cancelled by Stop() — a user-requested stop, not a failure
		}
		return err
	}
	defer func() {
		if res.AudioPath != "" {
			_ = os.Remove(res.AudioPath)
		}
	}()
	if err := s.player.Play(cctx, res.AudioPath); err != nil && cctx.Err() == nil {
		return err
	}
	return nil
}
