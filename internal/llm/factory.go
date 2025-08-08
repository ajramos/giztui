package llm

import (
	"time"
)

// NewProviderFromConfig creates a Provider from config fields
func NewProviderFromConfig(provider, endpoint, model string, timeout time.Duration, apiKey string) Provider {
	switch provider {
	case "ollama", "":
		return NewClient(endpoint, model, timeout)
	default:
		// For now return ollama client as default; future: implement other providers
		return NewClient(endpoint, model, timeout)
	}
}
