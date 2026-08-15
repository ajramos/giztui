package config

import (
	"os"

	"github.com/ajramos/giztui/internal/llm"
)

// BuildEffectiveProvider resolves the effective LLM for accountID and constructs
// the corresponding llm.Provider. It centralizes the provider-name default
// (ollama) and the Bedrock region / AWS_REGION fallback that the startup, TUI
// account-switch, and desktop call sites previously duplicated.
//
// It returns (nil, nil) when AI is disabled or no model is configured, so
// callers degrade to "AI off" without treating it as an error. A non-nil error
// means a configured provider failed to build (e.g. unknown provider name or a
// Bedrock credential/region problem); callers should log it and leave AI off.
func (c *Config) BuildEffectiveProvider(accountID string) (llm.Provider, error) {
	eff := c.EffectiveLLM(accountID)
	if !eff.Enabled || eff.Model == "" {
		return nil, nil
	}

	providerName := eff.Provider
	if providerName == "" {
		providerName = "ollama"
	}

	// For Bedrock the "endpoint" arg is treated as the AWS region, with an
	// AWS_REGION environment fallback (mirrors the historical wiring).
	arg := eff.Endpoint
	if providerName == "bedrock" {
		region := eff.Region
		if region == "" {
			region = os.Getenv("AWS_REGION")
		}
		arg = region
	}

	return llm.NewProviderFromConfig(providerName, arg, eff.Model, c.GetLLMTimeout(), eff.APIKey)
}
