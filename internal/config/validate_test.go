package config

import (
	"testing"

	"github.com/ajramos/giztui/internal/obsidian"
)

func asVErr(t *testing.T, err error) *ConfigValidationError {
	t.Helper()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	vce, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected *ConfigValidationError, got %T", err)
	}
	return vce
}

func hasIssue(vce *ConfigValidationError, field string, fatal bool) bool {
	for _, is := range vce.Issues {
		if is.Field == field && is.Fatal == fatal {
			return true
		}
	}
	return false
}

// The shipped defaults must always validate clean — otherwise every fresh start
// would print spurious warnings.
func TestValidate_DefaultConfigIsClean(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("DefaultConfig() must validate clean, got: %v", err)
	}
}

func TestValidate_LLMEnabledNeedsModel(t *testing.T) {
	c := DefaultConfig()
	c.LLM.Enabled = true
	c.LLM.Provider = "ollama"
	c.LLM.Endpoint = "http://localhost:11434"
	c.LLM.Model = ""
	vce := asVErr(t, c.Validate())
	if !hasIssue(vce, "llm.model", true) {
		t.Errorf("expected fatal llm.model issue, got: %+v", vce.Issues)
	}
}

func TestValidate_UnknownProviderIsWarningNotFatal(t *testing.T) {
	c := DefaultConfig()
	c.LLM.Enabled = true
	c.LLM.Provider = "openai" // unsupported → falls back to ollama
	c.LLM.Model = "gpt"
	vce := asVErr(t, c.Validate())
	if !hasIssue(vce, "llm.provider", false) {
		t.Errorf("expected non-fatal llm.provider warning, got: %+v", vce.Issues)
	}
	if vce.HasFatal() {
		t.Errorf("unknown provider must not be fatal, got fatal: %+v", vce.Fatal())
	}
}

func TestValidate_BadTimeoutIsFatal(t *testing.T) {
	c := DefaultConfig()
	c.LLM.Enabled = true
	c.LLM.Provider = "ollama"
	c.LLM.Endpoint = "http://localhost:11434"
	c.LLM.Model = "llama3"
	c.LLM.Timeout = "notaduration"
	vce := asVErr(t, c.Validate())
	if !hasIssue(vce, "llm.timeout", true) {
		t.Errorf("expected fatal llm.timeout issue, got: %+v", vce.Issues)
	}
}

func TestValidate_ObsidianEnabledNeedsVault(t *testing.T) {
	c := DefaultConfig()
	c.Obsidian = &obsidian.ObsidianConfig{Enabled: true, VaultPath: ""}
	vce := asVErr(t, c.Validate())
	if !hasIssue(vce, "obsidian.vault_path", true) {
		t.Errorf("expected fatal obsidian.vault_path issue, got: %+v", vce.Issues)
	}
}

// One pass must surface every problem at once, split into fatal vs warning.
func TestValidate_CollectsAllAtOnce(t *testing.T) {
	c := DefaultConfig()
	c.LLM.Enabled = true
	c.LLM.Provider = "ollama"
	c.LLM.Endpoint = ""                                  // warning
	c.LLM.Model = ""                                     // fatal
	c.LLM.Timeout = "bad"                                // fatal
	c.Obsidian = &obsidian.ObsidianConfig{Enabled: true} // fatal (no vault)
	c.Threading.MaxThreadDepth = -1                      // warning

	vce := asVErr(t, c.Validate())
	if got := len(vce.Fatal()); got < 3 {
		t.Errorf("expected >=3 fatal issues, got %d: %+v", got, vce.Fatal())
	}
	if got := len(vce.Warnings()); got < 2 {
		t.Errorf("expected >=2 warnings, got %d: %+v", got, vce.Warnings())
	}
	if vce.Error() == "" {
		t.Error("Error() should summarise the fatal issues")
	}
}
