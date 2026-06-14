package tts

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SaySynthesizer uses the macOS built-in `say` command — zero external dependencies, always
// available on macOS. It speaks **directly** (streaming): `say` starts talking almost immediately
// and is killed instantly on cancel, with no temp file or separate player. opts.ModelPath is
// ignored; an optional Voice selects a macOS system voice. Because it plays itself, Synthesize
// returns an empty AudioPath — the SpeechService takes that as "already played".
type SaySynthesizer struct {
	Voice string // optional macOS voice name (e.g. "Mónica"); empty = system default
}

func (s *SaySynthesizer) Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesisResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyText
	}
	if _, err := exec.LookPath("say"); err != nil {
		return nil, ErrNotConfigured
	}
	var args []string
	if v := strings.TrimSpace(s.Voice); v != "" {
		args = append(args, "-v", v)
	}
	cmd := exec.CommandContext(ctx, "say", args...) // #nosec G204 -- voice is operator-configured; text via stdin
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tts: say failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return &SynthesisResult{AudioPath: "", Engine: "say"}, nil
}
