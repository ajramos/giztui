package llm

import (
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
	default:
		// Fallback to Ollama for unknown providers
		return NewClient(endpoint, model, timeout), nil
	}
}
