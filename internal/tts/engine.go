package tts

import "runtime"

// ResolveEngine maps a configured engine value to a concrete engine. "say" and "piper" are honored
// as-is; anything else ("" or "auto") auto-selects by OS: macOS gets the built-in "say" (zero
// dependencies, always available), every other platform gets the cross-platform "piper".
func ResolveEngine(configured string) string {
	switch configured {
	case "say", "piper":
		return configured
	default:
		if runtime.GOOS == "darwin" {
			return "say"
		}
		return "piper"
	}
}
