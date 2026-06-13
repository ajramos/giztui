package tts

import "errors"

var (
	// ErrEmptyText is returned when there is nothing to synthesize.
	ErrEmptyText = errors.New("tts: empty text")
	// ErrNotConfigured is returned when the engine binary or model is missing.
	ErrNotConfigured = errors.New("tts: engine not configured (piper binary or model missing)")
)
