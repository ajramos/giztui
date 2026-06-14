package llm

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDetectBedrockFamily(t *testing.T) {
	cases := map[string]string{
		"anthropic.claude-3-sonnet": "anthropic",
		"us.anthropic.claude-v2":    "anthropic",
		"meta.llama3":               "meta",
		"amazon.titan-text":         "titan",
		"qwen2.5":                   "",
		"":                          "",
	}
	for in, want := range cases {
		if got := detectBedrockFamily(in); got != want {
			t.Errorf("detectBedrockFamily(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewProviderFromConfig_OllamaFallback(t *testing.T) {
	// ollama, empty, and unknown providers all resolve to the Ollama client (no network).
	for _, p := range []string{"ollama", "", "somethingunknown"} {
		prov, err := NewProviderFromConfig(p, "http://localhost:11434", "m", time.Second, "")
		if err != nil {
			t.Fatalf("provider %q: unexpected error %v", p, err)
		}
		if prov == nil || prov.Name() != "ollama" {
			t.Errorf("provider %q should fall back to ollama, got %v", p, prov)
		}
	}
}

func TestAnnotateBedrockError(t *testing.T) {
	if annotateBedrockError(nil, "m") != nil {
		t.Error("nil error should annotate to nil")
	}
	thr := errors.New("ValidationException: on-demand throughput isn't supported")
	if got := annotateBedrockError(thr, "anthropic.claude"); !strings.Contains(got.Error(), "inference profile") {
		t.Errorf("throughput error should be annotated, got %v", got)
	}
	inv := errors.New("The provided model identifier is invalid")
	if got := annotateBedrockError(inv, "m"); !strings.Contains(got.Error(), "inference profile ID") {
		t.Errorf("invalid-model error should be annotated, got %v", got)
	}
	other := errors.New("some other error")
	if got := annotateBedrockError(other, "m"); got != other {
		t.Error("an unrelated error should pass through unchanged")
	}
}
