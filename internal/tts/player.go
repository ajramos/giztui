package tts

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// Player plays an audio file through the OS. Killable via the context.
type Player interface {
	Play(ctx context.Context, audioPath string) error
}

// OSPlayer uses the platform's audio CLI.
type OSPlayer struct{}

func (OSPlayer) Play(ctx context.Context, audioPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "afplay", audioPath).Run() // #nosec G204 -- audioPath is a temp file we created
	case "linux":
		if _, err := exec.LookPath("paplay"); err == nil {
			return exec.CommandContext(ctx, "paplay", audioPath).Run() // #nosec G204
		}
		return exec.CommandContext(ctx, "aplay", audioPath).Run() // #nosec G204
	default:
		return fmt.Errorf("tts: audio playback unsupported on %s", runtime.GOOS)
	}
}
