package tts

// SynthesizeOptions configures a single synthesis call. The fields are per-call so the caller can
// pick a voice/model by the detected language of the text.
type SynthesizeOptions struct {
	ModelPath string // path to the .onnx voice model (engine "piper")
	Voice     string // voice name (engine "say"); empty = system default
}

// SynthesisResult describes the generated audio.
type SynthesisResult struct {
	AudioPath string
	Engine    string
	Model     string
}
