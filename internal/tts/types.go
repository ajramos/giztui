package tts

// SynthesizeOptions configures a single synthesis call.
type SynthesizeOptions struct {
	ModelPath string // path to the .onnx voice model
}

// SynthesisResult describes the generated audio.
type SynthesisResult struct {
	AudioPath string
	Engine    string
	Model     string
}
