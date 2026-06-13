package services

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/tts"
)

// SpeechServiceImpl composes a Synthesizer + Player and manages start/stop.
type SpeechServiceImpl struct {
	synth     tts.Synthesizer
	player    tts.Player
	piperPath string
	modelPath string

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewSpeechService(synth tts.Synthesizer, player tts.Player, piperPath, modelPath string) *SpeechServiceImpl {
	return &SpeechServiceImpl{synth: synth, player: player, piperPath: piperPath, modelPath: modelPath}
}

func (s *SpeechServiceImpl) IsConfigured() bool {
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
		return err
	}
	defer func() {
		if res.AudioPath != "" {
			_ = os.Remove(res.AudioPath)
		}
	}()
	return s.player.Play(cctx, res.AudioPath)
}
