package config

import "testing"

func baseCfg() *Config {
	g := DefaultLLMConfig() // ollama, llama3.2, ollama endpoint, custom templates
	g.SummarizeTemplate = "templates/custom/summarize.md"
	return &Config{
		LLM: g,
		Accounts: []AccountConfig{
			{ID: "work"}, // no override → inherits global
			{ID: "personal", LLM: &LLMConfig{
				Provider: "openai",
				Model:    "gpt-4o",
				APIKey:   "sk-personal",
				// endpoint omitted on purpose; templates omitted → inherit global
			}},
		},
	}
}

func TestEffectiveLLM_InheritsGlobal(t *testing.T) {
	c := baseCfg()
	for _, id := range []string{"", "work", "does-not-exist"} {
		eff := c.EffectiveLLM(id)
		if eff.Provider != c.LLM.Provider || eff.Model != c.LLM.Model {
			t.Errorf("id=%q: expected global (%s/%s), got %s/%s",
				id, c.LLM.Provider, c.LLM.Model, eff.Provider, eff.Model)
		}
	}
}

func TestEffectiveLLM_Override(t *testing.T) {
	c := baseCfg()
	eff := c.EffectiveLLM("personal")

	// Core fields come from the override.
	if eff.Provider != "openai" || eff.Model != "gpt-4o" || eff.APIKey != "sk-personal" {
		t.Fatalf("override core not applied: %+v", eff)
	}
	// Provider changed without a new endpoint → the ollama endpoint must NOT leak in.
	if eff.Endpoint != "" {
		t.Errorf("endpoint should be dropped when switching provider, got %q", eff.Endpoint)
	}
	// Cosmetic fields the override didn't set inherit the GLOBAL (custom) value.
	if eff.SummarizeTemplate != c.LLM.SummarizeTemplate {
		t.Errorf("cosmetic template should inherit global %q, got %q",
			c.LLM.SummarizeTemplate, eff.SummarizeTemplate)
	}
	// Boolean toggles inherit the global values (v1 limitation).
	if eff.Enabled != c.LLM.Enabled || eff.StreamEnabled != c.LLM.StreamEnabled {
		t.Errorf("bool toggles should inherit global (enabled=%v stream=%v), got enabled=%v stream=%v",
			c.LLM.Enabled, c.LLM.StreamEnabled, eff.Enabled, eff.StreamEnabled)
	}
	// The global config is not mutated by resolution.
	if c.LLM.Provider != "ollama" {
		t.Errorf("global LLM mutated: %s", c.LLM.Provider)
	}
}

func TestEffectiveLLM_ExplicitEndpointHonored(t *testing.T) {
	c := baseCfg()
	c.Accounts[1].LLM.Endpoint = "https://api.openai.com/v1"
	if got := c.EffectiveLLM("personal").Endpoint; got != "https://api.openai.com/v1" {
		t.Errorf("explicit override endpoint not honored: %q", got)
	}
}

func TestBuildEffectiveProvider_MissingCredentials(t *testing.T) {
	// A configured-but-uncredentialed account must NOT build a provider — AI is
	// disabled with an error, not "enabled" and failing every request.
	mk := func(ov *LLMConfig) *Config {
		g := DefaultLLMConfig()
		g.Enabled = true
		return &Config{LLM: g, Accounts: []AccountConfig{{ID: "a", LLM: ov}}}
	}

	// openai without api_key → error.
	if prov, err := mk(&LLMConfig{Enabled: true, Provider: "openai", Model: "gpt-4o-mini"}).BuildEffectiveProvider("a"); err == nil || prov != nil {
		t.Errorf("openai without api_key: want error+nil, got prov=%v err=%v", prov, err)
	}
	// vertex without project/region and without api_key → error.
	if prov, err := mk(&LLMConfig{Enabled: true, Provider: "vertex", Model: "gemini-1.5-flash"}).BuildEffectiveProvider("a"); err == nil || prov != nil {
		t.Errorf("vertex without creds: want error+nil, got prov=%v err=%v", prov, err)
	}
	// vertex with an api_key → builds.
	if prov, err := mk(&LLMConfig{Enabled: true, Provider: "vertex", Model: "gemini-1.5-flash", APIKey: "AIza"}).BuildEffectiveProvider("a"); err != nil || prov == nil {
		t.Errorf("vertex with api_key: want provider, got prov=%v err=%v", prov, err)
	}
	// vertex with project+region → builds.
	if prov, err := mk(&LLMConfig{Enabled: true, Provider: "vertex", Model: "gemini-1.5-pro", Project: "p", Region: "us-central1"}).BuildEffectiveProvider("a"); err != nil || prov == nil {
		t.Errorf("vertex with project+region: want provider, got prov=%v err=%v", prov, err)
	}
}
