package tts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExternalPiperSynthesizer runs the local Piper binary: text on stdin → a temp WAV file.
type ExternalPiperSynthesizer struct {
	PiperPath string
}

func (s *ExternalPiperSynthesizer) Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesisResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyText
	}
	if !fileExists(s.PiperPath) || !fileExists(opts.ModelPath) {
		return nil, ErrNotConfigured
	}
	tmp, err := os.CreateTemp("", "giztui-tts-*.wav")
	if err != nil {
		return nil, fmt.Errorf("tts: temp file: %w", err)
	}
	_ = tmp.Close()
	wav := tmp.Name()

	cmd := exec.CommandContext(ctx, s.PiperPath, "--model", opts.ModelPath, "--output_file", wav) // #nosec G204 -- piper path is operator-configured
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(wav)
		return nil, fmt.Errorf("tts: piper failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return &SynthesisResult{AudioPath: wav, Engine: "piper", Model: opts.ModelPath}, nil
}

func fileExists(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
