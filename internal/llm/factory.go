package llm

import (
	"fmt"
	"time"
)

// NewProviderFromConfig creates a Provider from config fields
// provider: provider name (e.g., "ollama", "bedrock")
// endpoint: provider-specific endpoint; for Bedrock use AWS region (e.g., "us-east-1")
// model: model identifier
// timeout: request timeout
// apiKey: optional API key for providers that require it (unused for Ollama/Bedrock)
func NewProviderFromConfig(provider, endpoint, model string, timeout time.Duration, apiKey string) (Provider, error) {
	switch provider {
	case "ollama", "":
		return NewClient(endpoint, model, timeout), nil
	case "bedrock":
		// endpoint is treated as region for Bedrock
		br, err := NewBedrock(endpoint, model, timeout)
		if err != nil {
			return nil, err
		}
		return br, nil
	case "openai":
		// Metered OpenAI API (api.openai.com) with an API key. endpoint may point
		// at a compatible gateway; empty → the public API. Require the key up
		// front so a misconfigured account disables AI with a clear log instead
		// of failing on every request while appearing "enabled".
		if apiKey == "" {
			return nil, fmt.Errorf("openai provider requires api_key")
		}
		return NewOpenAI(apiKey, endpoint, model, timeout), nil
	case "chatgpt":
		// ChatGPT Plus/Pro subscription via OAuth (no api key). Credentials live
		// in the machine-wide authstore; run `:llm login chatgpt` once.
		return NewChatGPT(model, timeout), nil
	default:
		// Unknown providers are a configuration error (typically a per-account
		// typo). Return an error instead of silently running Ollama so callers
		// disable AI with a clear log rather than using the wrong engine.
		return nil, fmt.Errorf("unknown LLM provider %q (supported: ollama, bedrock, openai, chatgpt, vertex)", provider)
	}
}
