package tts

import "context"

// Synthesizer turns text into an audio file. Decoupled so the engine can be swapped.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesisResult, error)
}
