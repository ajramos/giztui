package services

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/config"
	"github.com/ajramos/giztui/internal/tts"
)

// SpeechServiceImpl composes a Synthesizer + Player and manages start/stop. It picks the voice
// (say) or model (piper) per call by auto-detecting the text's language against the configured
// per-language maps.
type SpeechServiceImpl struct {
	synth  tts.Synthesizer
	player tts.Player
	engine string // resolved engine: "piper" or "say"
	cfg    config.TTSConfig

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewSpeechService(synth tts.Synthesizer, player tts.Player, engine string, cfg config.TTSConfig) *SpeechServiceImpl {
	return &SpeechServiceImpl{synth: synth, player: player, engine: engine, cfg: cfg}
}

// IsConfigured reports whether the active engine can synthesize. For "say" (macOS) that means the
// command exists; for "piper" (the default) the binary and model files must both be present.
func (s *SpeechServiceImpl) IsConfigured() bool {
	if s.engine == "say" {
		_, err := exec.LookPath("say")
		return err == nil
	}
	return fileExists(s.cfg.PiperPath) && fileExists(s.cfg.ModelPath)
}

// resolveByLanguage detects the text's language (restricted to the configured per-language map
// keys) and returns the matching voice (say) or model (piper), falling back to the defaults.
func (s *SpeechServiceImpl) resolveByLanguage(text string) tts.SynthesizeOptions {
	opts := tts.SynthesizeOptions{Voice: s.cfg.Voice, ModelPath: s.cfg.ModelPath}
	perLang := s.cfg.Models
	if s.engine == "say" {
		perLang = s.cfg.Voices
	}
	if len(perLang) == 0 {
		return opts // no per-language overrides → defaults
	}
	candidates := make([]string, 0, len(perLang))
	for k := range perLang {
		candidates = append(candidates, k)
	}
	lang := tts.DetectLanguage(text, candidates)
	if v, ok := perLang[lang]; ok && strings.TrimSpace(v) != "" {
		if s.engine == "say" {
			opts.Voice = v
		} else {
			opts.ModelPath = v
		}
	}
	return opts
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

	res, err := s.synth.Synthesize(cctx, text, s.resolveByLanguage(text))
	if err != nil {
		if cctx.Err() != nil {
			return nil // cancelled by Stop() — a user-requested stop, not a failure
		}
		return err
	}
	// A synthesizer that plays directly (e.g. macOS `say`) returns no file — nothing left to do.
	if res.AudioPath == "" {
		return nil
	}
	defer func() { _ = os.Remove(res.AudioPath) }()
	if err := s.player.Play(cctx, res.AudioPath); err != nil && cctx.Err() == nil {
		return err
	}
	return nil
}
