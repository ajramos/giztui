# Prompt Configurator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a natural-language-driven prompt creation/refinement UI that integrates with the existing GizTUI prompts picker, eliminating the friction of writing prompt templates by hand.

**Architecture:** A new `PromptGeneratorService` (NL → prompt via LLM) sits alongside the existing `PromptService`. A new TUI panel `prompt_configurator.go` provides the UX (intent input + editable prompt + LLM refinement + save). The existing single (`prompts.go`) and bulk (`bulk_prompts.go`) prompt pickers are enriched with a "✨ Create new with AI" entry point. All keys are configurable. All status messages go through `ErrorHandler`. ESC handlers run synchronously.

**Tech Stack:** Go 1.25.11, tview/tcell, sqlite (via existing `Store`), mockery for service mocks, testify for assertions.

**Spec reference:** `docs/superpowers/specs/2026-06-06-inbox-analysis-prompt-configurator-design.md`

---

## File structure

| Path | Action | Responsibility |
|---|---|---|
| `internal/services/interfaces.go` | MODIFY | Add `PromptGeneratorService` interface + `PromptGenerationOptions` + `GeneratedPrompt` types |
| `internal/services/prompt_generator_service.go` | CREATE | Implementation of `PromptGeneratorService` using `AIService` |
| `internal/services/prompt_generator_service_test.go` | CREATE | Unit tests with mocked `AIService` |
| `internal/services/mocks/MockPromptGeneratorService.go` | CREATE (via mockery) | Generated mock |
| `internal/config/config.go` | MODIFY | Add new fields to `KeyBindings` + defaults |
| `internal/tui/app.go` | MODIFY | Add `PickerPromptConfigurator` enum, helper method, service wiring |
| `internal/tui/prompt_configurator.go` | CREATE | UI panel: open/close, intent input, edit, refine, apply, save |
| `internal/tui/prompts.go` | MODIFY | Add "Create new with AI" item + relax category filter |
| `internal/tui/bulk_prompts.go` | MODIFY | Add "Create new with AI" item + relax category filter |
| `internal/tui/keys.go` | MODIFY | Add ESC handling for `PickerPromptConfigurator` |
| `internal/tui/commands.go` | MODIFY | Add `:prompt-new`, `:prompt-refine`, `:prompt-save` cases + handlers; update suggestion generator |
| `docs/KEYBOARD_SHORTCUTS.md` | MODIFY | Document new shortcuts + commands |

**Reused without modification:** `PromptService`, `AIService`, `EmailService`, `BulkPromptService`, `ErrorHandler`, `GetComponentColors`, `setActivePicker`, `streamingCancel` pattern, `MessagePreloader`.

---

## Task 1 — Add service interface and types

**Files:**
- Modify: `internal/services/interfaces.go` (add at end before the last existing struct)

- [ ] **Step 1: Read the current end of the file to find insertion point**

Run: `tail -20 internal/services/interfaces.go`
Expected: shows last lines of the file (around `AccountValidationResult` struct).

- [ ] **Step 2: Append the new interface and types**

Append to `internal/services/interfaces.go`:

```go
// PromptGeneratorService converts natural-language intent into prompt templates
// and refines existing prompts via LLM. Used by the Prompt Configurator UI.
type PromptGeneratorService interface {
	// GenerateFromIntent produces a prompt template from a natural-language description.
	GenerateFromIntent(ctx context.Context, intent string, opts PromptGenerationOptions) (*GeneratedPrompt, error)

	// RefinePrompt applies a refinement instruction to an existing prompt.
	RefinePrompt(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions) (*GeneratedPrompt, error)

	// Streaming variants — onToken is invoked for each token as it arrives.
	GenerateFromIntentStream(ctx context.Context, intent string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error)
	RefinePromptStream(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error)
}

// PromptGenerationOptions controls how a prompt is generated or refined.
type PromptGenerationOptions struct {
	// TargetMode hints what context the prompt will run in:
	// "single" (one email body via {{body}}), "bulk" (many via {{messages}}),
	// or "analyzer" (categorization output expected). Empty = auto-detect.
	TargetMode string

	// OutputFormat hints the desired LLM output structure:
	// "markdown" (default), "json", "plain".
	OutputFormat string

	// Language for the generated prompt itself (default: "en").
	Language string
}

// GeneratedPrompt is the result of generation or refinement.
type GeneratedPrompt struct {
	// PromptText is the actual template, ready to use (with {{body}}/{{messages}} placeholders).
	PromptText string

	// SuggestedName is a short label proposed by the LLM (used as default in the save dialog).
	SuggestedName string

	// SuggestedDesc is a one-line description proposed by the LLM.
	SuggestedDesc string

	// DetectedMode is what the LLM thinks this prompt is suited for ("single"/"bulk"/"analyzer").
	DetectedMode string

	// Duration is the elapsed time of the LLM call.
	Duration time.Duration
}
```

- [ ] **Step 3: Verify the file compiles**

Run: `go build ./internal/services/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/services/interfaces.go
git commit -m "feat(services): add PromptGeneratorService interface and types"
```

---

## Task 2 — Create service skeleton and failing test

**Files:**
- Create: `internal/services/prompt_generator_service.go`
- Create: `internal/services/prompt_generator_service_test.go`

- [ ] **Step 1: Create the service file with a constructor and stub methods**

Create `internal/services/prompt_generator_service.go`:

```go
package services

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PromptGeneratorServiceImpl implements PromptGeneratorService using AIService.
type PromptGeneratorServiceImpl struct {
	aiService AIService
}

// NewPromptGeneratorService creates a new generator service.
func NewPromptGeneratorService(aiService AIService) *PromptGeneratorServiceImpl {
	return &PromptGeneratorServiceImpl{aiService: aiService}
}

// GenerateFromIntent produces a prompt template from natural-language intent.
func (s *PromptGeneratorServiceImpl) GenerateFromIntent(ctx context.Context, intent string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// RefinePrompt applies a refinement to an existing prompt.
func (s *PromptGeneratorServiceImpl) RefinePrompt(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// GenerateFromIntentStream is the streaming version of GenerateFromIntent.
func (s *PromptGeneratorServiceImpl) GenerateFromIntentStream(ctx context.Context, intent string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// RefinePromptStream is the streaming version of RefinePrompt.
func (s *PromptGeneratorServiceImpl) RefinePromptStream(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	return nil, fmt.Errorf("not implemented")
}

// buildGenerationPrompt constructs the meta-prompt sent to the LLM for intent-based generation.
func (s *PromptGeneratorServiceImpl) buildGenerationPrompt(intent string, opts PromptGenerationOptions) string {
	var b strings.Builder
	b.WriteString("You are a prompt engineer. The user describes an intent and you write a clean, reusable prompt template they can apply to email content.\n\n")
	b.WriteString("Rules for your output:\n")
	b.WriteString("1. Use {{body}} as placeholder for a single email body, or {{messages}} for multiple email bodies, depending on the intent.\n")
	b.WriteString("2. Return ONLY the prompt template followed by metadata on separate lines.\n")
	b.WriteString("3. Metadata format (each on its own line, after the template):\n")
	b.WriteString("   __NAME__: short kebab-case name (max 40 chars)\n")
	b.WriteString("   __DESC__: one-line description (max 120 chars)\n")
	b.WriteString("   __MODE__: one of single|bulk|analyzer\n\n")
	if opts.OutputFormat != "" {
		fmt.Fprintf(&b, "Constraint: the prompt MUST instruct the LLM to output in %s format.\n", opts.OutputFormat)
	}
	if opts.TargetMode != "" {
		fmt.Fprintf(&b, "Constraint: target this prompt for the %s context.\n", opts.TargetMode)
	}
	fmt.Fprintf(&b, "\nUser intent:\n%s\n", intent)
	return b.String()
}

// buildRefinementPrompt constructs the meta-prompt for refinement.
func (s *PromptGeneratorServiceImpl) buildRefinementPrompt(currentPrompt string, refinement string, opts PromptGenerationOptions) string {
	var b strings.Builder
	b.WriteString("You are a prompt engineer. The user has a current prompt and wants to apply a refinement.\n\n")
	b.WriteString("Return the FULL revised prompt followed by the same metadata block as before:\n")
	b.WriteString("   __NAME__: short kebab-case name\n")
	b.WriteString("   __DESC__: one-line description\n")
	b.WriteString("   __MODE__: one of single|bulk|analyzer\n\n")
	fmt.Fprintf(&b, "Current prompt:\n%s\n\n", currentPrompt)
	fmt.Fprintf(&b, "Refinement requested:\n%s\n", refinement)
	return b.String()
}

// parseGeneratedOutput extracts the prompt body and metadata from the LLM response.
func (s *PromptGeneratorServiceImpl) parseGeneratedOutput(raw string, duration time.Duration) *GeneratedPrompt {
	result := &GeneratedPrompt{Duration: duration}
	lines := strings.Split(raw, "\n")
	var bodyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "__NAME__:"):
			result.SuggestedName = strings.TrimSpace(strings.TrimPrefix(trimmed, "__NAME__:"))
		case strings.HasPrefix(trimmed, "__DESC__:"):
			result.SuggestedDesc = strings.TrimSpace(strings.TrimPrefix(trimmed, "__DESC__:"))
		case strings.HasPrefix(trimmed, "__MODE__:"):
			result.DetectedMode = strings.TrimSpace(strings.TrimPrefix(trimmed, "__MODE__:"))
		default:
			bodyLines = append(bodyLines, line)
		}
	}
	result.PromptText = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return result
}
```

- [ ] **Step 2: Create the test file with the first failing test**

Create `internal/services/prompt_generator_service_test.go`:

```go
package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewPromptGeneratorService verifies the constructor stores the AIService.
func TestNewPromptGeneratorService(t *testing.T) {
	service := NewPromptGeneratorService(nil)

	assert.NotNil(t, service)
	assert.Nil(t, service.aiService)
}

// TestPromptGeneratorServiceImpl_GenerateFromIntent_NilAIService verifies error path.
func TestPromptGeneratorServiceImpl_GenerateFromIntent_NilAIService(t *testing.T) {
	service := &PromptGeneratorServiceImpl{aiService: nil}

	result, err := service.GenerateFromIntent(context.Background(), "find urgent emails", PromptGenerationOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestPromptGeneratorServiceImpl_parseGeneratedOutput_HappyPath verifies parsing.
func TestPromptGeneratorServiceImpl_parseGeneratedOutput_HappyPath(t *testing.T) {
	service := &PromptGeneratorServiceImpl{}

	raw := `You are an email triage assistant.
Analyze the email {{body}} and identify urgency.

__NAME__: triage-urgency
__DESC__: classify single email by urgency level
__MODE__: single`

	result := service.parseGeneratedOutput(raw, 0)

	assert.Equal(t, "triage-urgency", result.SuggestedName)
	assert.Equal(t, "classify single email by urgency level", result.SuggestedDesc)
	assert.Equal(t, "single", result.DetectedMode)
	assert.Contains(t, result.PromptText, "{{body}}")
	assert.NotContains(t, result.PromptText, "__NAME__")
}

// TestPromptGeneratorServiceImpl_parseGeneratedOutput_MissingMetadata verifies graceful handling.
func TestPromptGeneratorServiceImpl_parseGeneratedOutput_MissingMetadata(t *testing.T) {
	service := &PromptGeneratorServiceImpl{}

	raw := `You are an email assistant. Analyze {{body}} please.`

	result := service.parseGeneratedOutput(raw, 0)

	assert.Equal(t, "", result.SuggestedName)
	assert.Equal(t, "", result.SuggestedDesc)
	assert.Equal(t, "", result.DetectedMode)
	assert.Equal(t, "You are an email assistant. Analyze {{body}} please.", result.PromptText)
}

// TestPromptGeneratorServiceImpl_buildGenerationPrompt_IncludesIntent verifies meta-prompt assembly.
func TestPromptGeneratorServiceImpl_buildGenerationPrompt_IncludesIntent(t *testing.T) {
	service := &PromptGeneratorServiceImpl{}

	prompt := service.buildGenerationPrompt("identify urgent replies needed", PromptGenerationOptions{
		OutputFormat: "json",
		TargetMode:   "bulk",
	})

	assert.Contains(t, prompt, "identify urgent replies needed")
	assert.Contains(t, prompt, "json")
	assert.Contains(t, prompt, "bulk")
	assert.Contains(t, prompt, "__NAME__")
	assert.Contains(t, prompt, "{{body}}")
	assert.Contains(t, prompt, "{{messages}}")
}
```

- [ ] **Step 3: Run the tests to verify the structural tests pass and the NilAIService one passes**

Run: `go test ./internal/services/ -run TestPromptGeneratorService -v`
Expected: 5 PASS. The error-path tests pass because the stub returns an error; the parse/build tests pass because those helpers are already implemented.

- [ ] **Step 4: Commit**

```bash
git add internal/services/prompt_generator_service.go internal/services/prompt_generator_service_test.go
git commit -m "feat(services): scaffold PromptGeneratorService with parse/build helpers"
```

---

## Task 3 — Implement GenerateFromIntent (non-streaming)

**Files:**
- Modify: `internal/services/prompt_generator_service.go`
- Modify: `internal/services/prompt_generator_service_test.go`

- [ ] **Step 1: Add a test that uses the existing AIService mock**

Look at `internal/services/mocks/` and confirm `AIService.go` mock exists.

Run: `ls internal/services/mocks/AIService.go`
Expected: file exists.

Append to `internal/services/prompt_generator_service_test.go`:

```go
import (
	mocks "github.com/ajramos/giztui/internal/services/mocks"
	"github.com/stretchr/testify/mock"
)

// TestPromptGeneratorServiceImpl_GenerateFromIntent_Success verifies a happy path with a mocked AI.
func TestPromptGeneratorServiceImpl_GenerateFromIntent_Success(t *testing.T) {
	mockAI := mocks.NewAIService(t)

	canned := `Analyze the email {{body}} and identify urgency.

__NAME__: triage-urgency
__DESC__: classify by urgency
__MODE__: single`

	mockAI.On("ApplyCustomPrompt",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return(canned, nil)

	service := NewPromptGeneratorService(mockAI)

	result, err := service.GenerateFromIntent(context.Background(), "classify by urgency", PromptGenerationOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "triage-urgency", result.SuggestedName)
	assert.Equal(t, "single", result.DetectedMode)
	assert.Contains(t, result.PromptText, "{{body}}")
}

// TestPromptGeneratorServiceImpl_GenerateFromIntent_AIServiceFailure verifies error surfacing.
func TestPromptGeneratorServiceImpl_GenerateFromIntent_AIServiceFailure(t *testing.T) {
	mockAI := mocks.NewAIService(t)

	mockAI.On("ApplyCustomPrompt",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return("", assert.AnError)

	service := NewPromptGeneratorService(mockAI)

	result, err := service.GenerateFromIntent(context.Background(), "anything", PromptGenerationOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
}
```

- [ ] **Step 2: Run the new tests, expect them to fail**

Run: `go test ./internal/services/ -run TestPromptGeneratorServiceImpl_GenerateFromIntent_Success -v`
Expected: FAIL because `GenerateFromIntent` returns "not implemented".

- [ ] **Step 3: Implement GenerateFromIntent**

Replace the body of `GenerateFromIntent` in `internal/services/prompt_generator_service.go`:

```go
func (s *PromptGeneratorServiceImpl) GenerateFromIntent(ctx context.Context, intent string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(intent) == "" {
		return nil, fmt.Errorf("intent cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildGenerationPrompt(intent, opts)

	// Pass an empty content because the meta-prompt is self-contained.
	raw, err := s.aiService.ApplyCustomPrompt(ctx, "", metaPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}
```

- [ ] **Step 4: Run the tests, expect them to pass**

Run: `go test ./internal/services/ -run TestPromptGeneratorServiceImpl_GenerateFromIntent -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/prompt_generator_service.go internal/services/prompt_generator_service_test.go
git commit -m "feat(services): implement GenerateFromIntent with mocked AI tests"
```

---

## Task 4 — Implement RefinePrompt (non-streaming)

**Files:**
- Modify: `internal/services/prompt_generator_service.go`
- Modify: `internal/services/prompt_generator_service_test.go`

- [ ] **Step 1: Add the test**

Append to `internal/services/prompt_generator_service_test.go`:

```go
// TestPromptGeneratorServiceImpl_RefinePrompt_Success verifies refinement happy path.
func TestPromptGeneratorServiceImpl_RefinePrompt_Success(t *testing.T) {
	mockAI := mocks.NewAIService(t)

	refined := `Analyze the email {{body}} and output ONLY valid JSON with field "urgency_level".

__NAME__: triage-urgency-json
__DESC__: classify urgency as JSON
__MODE__: single`

	mockAI.On("ApplyCustomPrompt",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return(refined, nil)

	service := NewPromptGeneratorService(mockAI)

	current := "Analyze the email {{body}} and identify urgency."
	result, err := service.RefinePrompt(context.Background(), current, "output as JSON", PromptGenerationOptions{})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.PromptText, "JSON")
	assert.Equal(t, "triage-urgency-json", result.SuggestedName)
}

// TestPromptGeneratorServiceImpl_RefinePrompt_EmptyCurrent verifies validation.
func TestPromptGeneratorServiceImpl_RefinePrompt_EmptyCurrent(t *testing.T) {
	service := NewPromptGeneratorService(nil)

	result, err := service.RefinePrompt(context.Background(), "", "make it stricter", PromptGenerationOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
}
```

- [ ] **Step 2: Run the tests, expect failure**

Run: `go test ./internal/services/ -run TestPromptGeneratorServiceImpl_RefinePrompt -v`
Expected: FAIL (not implemented).

- [ ] **Step 3: Implement RefinePrompt**

Replace the body of `RefinePrompt` in `internal/services/prompt_generator_service.go`:

```go
func (s *PromptGeneratorServiceImpl) RefinePrompt(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions) (*GeneratedPrompt, error) {
	if strings.TrimSpace(currentPrompt) == "" {
		return nil, fmt.Errorf("current prompt cannot be empty")
	}
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(refinement) == "" {
		return nil, fmt.Errorf("refinement instruction cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildRefinementPrompt(currentPrompt, refinement, opts)

	raw, err := s.aiService.ApplyCustomPrompt(ctx, "", metaPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM refinement failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}
```

- [ ] **Step 4: Run the tests, expect pass**

Run: `go test ./internal/services/ -run TestPromptGeneratorServiceImpl_RefinePrompt -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/prompt_generator_service.go internal/services/prompt_generator_service_test.go
git commit -m "feat(services): implement RefinePrompt with refinement validation"
```

---

## Task 5 — Implement streaming variants

**Files:**
- Modify: `internal/services/prompt_generator_service.go`
- Modify: `internal/services/prompt_generator_service_test.go`

- [ ] **Step 1: Add the streaming test**

Append to `internal/services/prompt_generator_service_test.go`:

```go
// TestPromptGeneratorServiceImpl_GenerateFromIntentStream_Success verifies tokens are emitted.
func TestPromptGeneratorServiceImpl_GenerateFromIntentStream_Success(t *testing.T) {
	mockAI := mocks.NewAIService(t)

	full := `Analyze {{body}}.

__NAME__: simple
__DESC__: simple prompt
__MODE__: single`

	mockAI.On("ApplyCustomPromptStream",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.AnythingOfType("func(string)"),
	).Run(func(args mock.Arguments) {
		// Invoke the streaming callback with chunks.
		cb := args.Get(4).(func(string))
		cb("Analyze ")
		cb("{{body}}.\n\n")
		cb("__NAME__: simple\n")
		cb("__DESC__: simple prompt\n")
		cb("__MODE__: single")
	}).Return(full, nil)

	service := NewPromptGeneratorService(mockAI)

	var tokens []string
	result, err := service.GenerateFromIntentStream(context.Background(), "describe", PromptGenerationOptions{}, func(t string) {
		tokens = append(tokens, t)
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(tokens), 0)
	assert.Equal(t, "simple", result.SuggestedName)
}
```

- [ ] **Step 2: Run, expect failure**

Run: `go test ./internal/services/ -run TestPromptGeneratorServiceImpl_GenerateFromIntentStream -v`
Expected: FAIL (not implemented).

- [ ] **Step 3: Implement both streaming variants**

Replace the bodies of `GenerateFromIntentStream` and `RefinePromptStream` in `internal/services/prompt_generator_service.go`:

```go
func (s *PromptGeneratorServiceImpl) GenerateFromIntentStream(ctx context.Context, intent string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(intent) == "" {
		return nil, fmt.Errorf("intent cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildGenerationPrompt(intent, opts)

	raw, err := s.aiService.ApplyCustomPromptStream(ctx, "", metaPrompt, nil, func(token string) {
		if onToken != nil {
			onToken(token)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("LLM streaming generation failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}

func (s *PromptGeneratorServiceImpl) RefinePromptStream(ctx context.Context, currentPrompt string, refinement string, opts PromptGenerationOptions, onToken func(string)) (*GeneratedPrompt, error) {
	if strings.TrimSpace(currentPrompt) == "" {
		return nil, fmt.Errorf("current prompt cannot be empty")
	}
	if s.aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}
	if strings.TrimSpace(refinement) == "" {
		return nil, fmt.Errorf("refinement instruction cannot be empty")
	}

	start := time.Now()
	metaPrompt := s.buildRefinementPrompt(currentPrompt, refinement, opts)

	raw, err := s.aiService.ApplyCustomPromptStream(ctx, "", metaPrompt, nil, func(token string) {
		if onToken != nil {
			onToken(token)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("LLM streaming refinement failed: %w", err)
	}

	return s.parseGeneratedOutput(raw, time.Since(start)), nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/services/ -run TestPromptGeneratorServiceImpl -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/prompt_generator_service.go internal/services/prompt_generator_service_test.go
git commit -m "feat(services): implement streaming variants of generate and refine"
```

---

## Task 6 — Generate mock for PromptGeneratorService

**Files:**
- Create: `internal/services/mocks/PromptGeneratorService.go` (via mockery)

- [ ] **Step 1: Run mockery using the existing Makefile target**

Run: `make test-mocks`
Expected: mocks regenerated. New file `internal/services/mocks/PromptGeneratorService.go` should exist.

- [ ] **Step 2: Verify the mock was generated**

Run: `ls internal/services/mocks/PromptGeneratorService.go`
Expected: file exists.

- [ ] **Step 3: Run all service tests to confirm no regression**

Run: `make test-unit`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/services/mocks/PromptGeneratorService.go
git commit -m "test(services): generate mock for PromptGeneratorService"
```

---

## Task 7 — Add KeyBindings configuration for configurator

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add new fields to the `KeyBindings` struct**

Insert before the `// Validation settings` block at the bottom of the `KeyBindings` struct in `internal/config/config.go`:

```go
	// Prompt Configurator (Feature 2)
	PromptRegenerate string `json:"prompt_regenerate"` // Regenerate prompt via LLM in configurator
	SavePrompt       string `json:"save_prompt"`       // Save active prompt to library
	PromptApply      string `json:"prompt_apply"`      // Apply the active prompt to scoped context
	PromptTest       string `json:"prompt_test"`       // Test the prompt on one message (stretch)
```

- [ ] **Step 2: Add defaults in `DefaultKeyBindings()`**

Insert before the `// Validation settings` block at the bottom of `DefaultKeyBindings()`:

```go
		// Prompt Configurator
		PromptRegenerate: "ctrl+r",
		SavePrompt:       "ctrl+s",
		PromptApply:      "enter", // applied within the editable prompt box
		PromptTest:       "ctrl+t",
```

- [ ] **Step 3: Verify the config still compiles and tests pass**

Run: `go build ./internal/config/... && go test ./internal/config/... -v`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add configurator keybindings with defaults"
```

---

## Task 8 — Add PickerPromptConfigurator enum and helper

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add the enum constant**

In `internal/tui/app.go`, find the `ActivePicker` constants block (around line 29). Add a new constant inside the block:

```go
	PickerPromptConfigurator ActivePicker = "prompt_configurator"
```

Place it after `PickerBulkPrompts` for logical grouping. The result should look like:

```go
const (
	PickerNone               ActivePicker = ""
	PickerLabels             ActivePicker = "labels"
	PickerDrafts             ActivePicker = "drafts"
	PickerObsidian           ActivePicker = "obsidian"
	PickerAttachments        ActivePicker = "attachments"
	PickerLinks              ActivePicker = "links"
	PickerPrompts            ActivePicker = "prompts"
	PickerBulkPrompts        ActivePicker = "bulk_prompts"
	PickerPromptConfigurator ActivePicker = "prompt_configurator"
	PickerSavedQueries       ActivePicker = "saved_queries"
	PickerThemes             ActivePicker = "themes"
	PickerAI                 ActivePicker = "ai_labels"
	PickerContentSearch      ActivePicker = "content_search"
	PickerRSVP               ActivePicker = "rsvp"
	PickerAccounts           ActivePicker = "accounts"
)
```

- [ ] **Step 2: Add the helper method**

Locate `isLabelsPickerActive()` in `internal/tui/app.go` (around line 3204). Immediately after it, add:

```go
// isPromptConfiguratorActive returns true if the Prompt Configurator picker is currently active.
func (a *App) isPromptConfiguratorActive() bool {
	return a.currentActivePicker == PickerPromptConfigurator
}
```

- [ ] **Step 3: Verify the code compiles**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): add PickerPromptConfigurator enum and helper"
```

---

## Task 9 — Wire PromptGeneratorService into App initialization

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add the service field to the App struct**

In `internal/tui/app.go`, find the App struct field declaration block around line 191 (where `bulkPromptService` lives). Add a new field:

```go
	promptGeneratorService services.PromptGeneratorService
```

Place it after `promptService` to keep prompt-related fields grouped.

- [ ] **Step 2: Initialize the service in `initServices()`**

Find `initServices()` around line 585. Locate where `bulkPromptService` is initialized (around line 720). Immediately after that initialization, add:

```go
	// Prompt generator (NL → prompt template via LLM)
	if a.aiService != nil {
		a.promptGeneratorService = services.NewPromptGeneratorService(a.aiService)
		if a.logger != nil {
			a.logger.Printf("initServices: prompt generator service initialized: %v", a.promptGeneratorService != nil)
		}
	}
```

- [ ] **Step 3: Initialize in `reinitializeServices()` too**

Find `reinitializeServices()` around line 470. After the block that initializes `bulkPromptService` (around line 511), add:

```go
	// Initialize prompt generator if AI is available and generator is nil
	if a.aiService != nil && a.promptGeneratorService == nil {
		a.promptGeneratorService = services.NewPromptGeneratorService(a.aiService)
		if a.logger != nil {
			a.logger.Printf("reinitializeServices: prompt generator service initialized: %v", a.promptGeneratorService != nil)
		}
	}
```

- [ ] **Step 4: Add an accessor method on App**

Find a logical place for service accessors (near `GetServices()` around line 1387). Add:

```go
// GetPromptGeneratorService returns the prompt generator service or nil if not initialized.
func (a *App) GetPromptGeneratorService() services.PromptGeneratorService {
	return a.promptGeneratorService
}
```

Rationale: rather than extending the 12-return `GetServices()` signature (which would require updating every call site), we add a dedicated accessor for this service.

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat(tui): wire PromptGeneratorService into App initialization"
```

---

## Task 10 — Create configurator panel shell (open/close)

**Files:**
- Create: `internal/tui/prompt_configurator.go`

- [ ] **Step 1: Create the file with the panel skeleton**

Create `internal/tui/prompt_configurator.go`:

```go
package tui

import (
	"context"
	"fmt"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

// promptConfiguratorContext describes what messages the configurator will act upon when Apply is pressed.
type promptConfiguratorContext struct {
	// Mode is "single", "bulk", or "draft" (no context, draft-only).
	mode string
	// messageID is set when mode == "single".
	messageID string
	// messageIDs is set when mode == "bulk".
	messageIDs []string
	// categoryName, if non-empty, indicates the context came from an action plan category.
	categoryName string
}

// promptConfiguratorState holds the mutable state of the configurator panel.
type promptConfiguratorState struct {
	ctx              promptConfiguratorContext
	currentPrompt    string
	suggestedName    string
	suggestedDesc    string
	detectedMode     string
	intentInput      *tview.InputField
	promptArea       *tview.TextArea
	refineInput      *tview.InputField
	statusLine       *tview.TextView
	container        *tview.Flex
	streamingCancel  context.CancelFunc
}

// openPromptConfigurator opens the configurator panel with the given context.
func (a *App) openPromptConfigurator(pctx promptConfiguratorContext) {
	if a.logger != nil {
		a.logger.Printf("openPromptConfigurator: mode=%s msgCount=%d", pctx.mode, len(pctx.messageIDs))
	}

	if a.GetPromptGeneratorService() == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt generator service not available - check LLM configuration")
		return
	}

	state := &promptConfiguratorState{ctx: pctx}

	colors := a.GetComponentColors("prompts")
	bgColor := colors.Background.Color()

	// Intent input
	state.intentInput = tview.NewInputField().
		SetLabel("Intent: ").
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	state.intentInput.SetBackgroundColor(bgColor)

	// Editable prompt area
	state.promptArea = tview.NewTextArea().
		SetPlaceholder("Generated prompt will appear here. Edit freely.")
	state.promptArea.SetBackgroundColor(bgColor)
	state.promptArea.SetTextStyle(tcell.StyleDefault.Background(bgColor).Foreground(colors.Text.Color()))
	state.promptArea.SetBorder(true)
	state.promptArea.SetTitle(" Editable prompt ")
	state.promptArea.SetTitleColor(colors.Title.Color())

	// Refine input
	state.refineInput = tview.NewInputField().
		SetLabel("Refine: ").
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	state.refineInput.SetBackgroundColor(bgColor)

	// Status line
	state.statusLine = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetText(fmt.Sprintf(" %s apply  %s refine  %s save  Esc cancel ",
			a.Keys.PromptApply, a.Keys.PromptRegenerate, a.Keys.SavePrompt))
	state.statusLine.SetTextColor(colors.Text.Color())
	state.statusLine.SetBackgroundColor(bgColor)

	// Container
	state.container = tview.NewFlex().SetDirection(tview.FlexRow)
	state.container.SetBackgroundColor(bgColor)
	state.container.SetBorder(true)
	state.container.SetTitle(promptConfiguratorTitle(pctx))
	state.container.SetTitleColor(colors.Title.Color())
	state.container.AddItem(state.intentInput, 1, 0, true)
	state.container.AddItem(state.promptArea, 0, 1, false)
	state.container.AddItem(state.refineInput, 1, 0, false)
	state.container.AddItem(state.statusLine, 1, 0, false)

	a.promptConfiguratorState = state

	// Attach to the content split
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = state.container
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}

	a.SetFocus(state.intentInput)
	a.currentFocus = "prompt_configurator"
	a.updateFocusIndicators("prompt_configurator")
	a.setActivePicker(PickerPromptConfigurator)
}

// closePromptConfigurator closes the configurator and restores the original view.
func (a *App) closePromptConfigurator() {
	// Synchronous cleanup — NEVER use QueueUpdateDraw in close paths (CLAUDE.md rule).
	if a.promptConfiguratorState != nil && a.promptConfiguratorState.streamingCancel != nil {
		a.promptConfiguratorState.streamingCancel()
		a.promptConfiguratorState.streamingCancel = nil
	}

	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.ResizeItem(a.labelsView, 0, 0)
		}
	}

	a.setActivePicker(PickerNone)
	a.promptConfiguratorState = nil

	if list, ok := a.views["list"].(*tview.Table); ok {
		a.SetFocus(list)
	}
	a.currentFocus = "list"
	a.updateFocusIndicators("list")
}

// promptConfiguratorTitle returns the panel title appropriate for the context.
func promptConfiguratorTitle(pctx promptConfiguratorContext) string {
	switch pctx.mode {
	case "single":
		return " ✨ Prompt Configurator (1 msg scoped) "
	case "bulk":
		if pctx.categoryName != "" {
			return fmt.Sprintf(" ✨ Prompt Configurator (%d msgs from %q) ", len(pctx.messageIDs), pctx.categoryName)
		}
		return fmt.Sprintf(" ✨ Prompt Configurator (%d msgs scoped) ", len(pctx.messageIDs))
	default:
		return " ✨ Prompt Configurator (draft only) "
	}
}
```

- [ ] **Step 2: Add the state field to the App struct**

In `internal/tui/app.go`, find the prompt-related fields (around line 191). Add after `promptGeneratorService`:

```go
	promptConfiguratorState *promptConfiguratorState
```

- [ ] **Step 3: Verify it builds**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/prompt_configurator.go internal/tui/app.go
git commit -m "feat(tui): scaffold prompt configurator panel with open/close"
```

---

## Task 11 — Wire intent input to generation

**Files:**
- Modify: `internal/tui/prompt_configurator.go`

- [ ] **Step 1: Implement generation when user presses Enter in intent input**

In `internal/tui/prompt_configurator.go`, modify `openPromptConfigurator` to attach a done-func to the intent input. Insert this block after the line `state.intentInput.SetBackgroundColor(bgColor)`:

```go
	state.intentInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.closePromptConfigurator()
			return
		}
		if key == tcell.KeyEnter {
			intent := state.intentInput.GetText()
			if intent == "" {
				return
			}
			go a.generateConfiguratorPrompt(intent)
		}
	})
```

- [ ] **Step 2: Implement `generateConfiguratorPrompt` helper**

Append to `internal/tui/prompt_configurator.go`:

```go
// generateConfiguratorPrompt invokes the LLM streaming to fill the editable prompt area.
func (a *App) generateConfiguratorPrompt(intent string) {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	gen := a.GetPromptGeneratorService()
	if gen == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt generator service not available")
		return
	}

	a.GetErrorHandler().ShowProgress(a.ctx, "Generating prompt...")

	// Clear and show loading
	a.QueueUpdateDraw(func() {
		if a.promptConfiguratorState != nil && a.promptConfiguratorState.promptArea != nil {
			a.promptConfiguratorState.promptArea.SetText("Generating...", false)
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	state.streamingCancel = cancel
	defer func() {
		cancel()
		if a.promptConfiguratorState != nil {
			a.promptConfiguratorState.streamingCancel = nil
		}
	}()

	var accumulator string

	result, err := gen.GenerateFromIntentStream(ctx, intent, PromptGenerationOptions{
		TargetMode: state.ctx.mode,
	}, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		accumulator += token
		// Direct UI update — CLAUDE.md prohibits QueueUpdateDraw inside streaming callbacks.
		if a.promptConfiguratorState != nil && a.promptConfiguratorState.promptArea != nil && ctx.Err() == nil {
			a.promptConfiguratorState.promptArea.SetText(accumulator, false)
		}
	})

	if err != nil {
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Prompt generation canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to generate prompt: %v", err))
		return
	}

	// Final update with the parsed PromptText (cleaner than the raw accumulator).
	a.QueueUpdateDraw(func() {
		if a.promptConfiguratorState != nil {
			a.promptConfiguratorState.currentPrompt = result.PromptText
			a.promptConfiguratorState.suggestedName = result.SuggestedName
			a.promptConfiguratorState.suggestedDesc = result.SuggestedDesc
			a.promptConfiguratorState.detectedMode = result.DetectedMode
			if a.promptConfiguratorState.promptArea != nil {
				a.promptConfiguratorState.promptArea.SetText(result.PromptText, false)
			}
		}
	})

	a.GetErrorHandler().ClearProgress()
	a.GetErrorHandler().ShowSuccess(a.ctx, "Prompt generated. Edit or refine before applying.")
}
```

Add the import `"github.com/ajramos/giztui/internal/services"` to the file if not already present. Note: `PromptGenerationOptions` is defined in `services` package — replace `PromptGenerationOptions{...}` with `services.PromptGenerationOptions{...}` in the call.

Corrected call:

```go
	result, err := gen.GenerateFromIntentStream(ctx, intent, services.PromptGenerationOptions{
		TargetMode: state.ctx.mode,
	}, func(token string) {
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/prompt_configurator.go
git commit -m "feat(tui): wire intent input to streaming prompt generation"
```

---

## Task 12 — Wire refine input to LLM refinement

**Files:**
- Modify: `internal/tui/prompt_configurator.go`

- [ ] **Step 1: Attach a done-func to the refine input**

In `internal/tui/prompt_configurator.go`, locate the `openPromptConfigurator` function. Insert after `state.refineInput.SetBackgroundColor(bgColor)`:

```go
	state.refineInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			a.closePromptConfigurator()
			return
		}
		if key == tcell.KeyEnter {
			refinement := state.refineInput.GetText()
			if refinement == "" {
				return
			}
			// Use whatever is currently in the editable box as the source.
			current := state.promptArea.GetText()
			go a.refineConfiguratorPrompt(current, refinement)
		}
	})
```

- [ ] **Step 2: Implement `refineConfiguratorPrompt`**

Append to `internal/tui/prompt_configurator.go`:

```go
// refineConfiguratorPrompt invokes the LLM streaming to refine the current prompt.
func (a *App) refineConfiguratorPrompt(currentPrompt string, refinement string) {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	gen := a.GetPromptGeneratorService()
	if gen == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt generator service not available")
		return
	}

	a.GetErrorHandler().ShowProgress(a.ctx, "Refining prompt...")

	// Show loading while preserving the previous prompt as fallback if cancelled.
	previous := currentPrompt
	a.QueueUpdateDraw(func() {
		if a.promptConfiguratorState != nil && a.promptConfiguratorState.promptArea != nil {
			a.promptConfiguratorState.promptArea.SetText("Refining...", false)
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	state.streamingCancel = cancel
	defer func() {
		cancel()
		if a.promptConfiguratorState != nil {
			a.promptConfiguratorState.streamingCancel = nil
		}
	}()

	var accumulator string

	result, err := gen.RefinePromptStream(ctx, currentPrompt, refinement, services.PromptGenerationOptions{
		TargetMode: state.ctx.mode,
	}, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		accumulator += token
		if a.promptConfiguratorState != nil && a.promptConfiguratorState.promptArea != nil && ctx.Err() == nil {
			a.promptConfiguratorState.promptArea.SetText(accumulator, false)
		}
	})

	if err != nil {
		// Restore previous prompt on cancellation or failure.
		a.QueueUpdateDraw(func() {
			if a.promptConfiguratorState != nil && a.promptConfiguratorState.promptArea != nil {
				a.promptConfiguratorState.promptArea.SetText(previous, false)
			}
		})
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Refinement canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to refine prompt: %v", err))
		return
	}

	a.QueueUpdateDraw(func() {
		if a.promptConfiguratorState != nil {
			a.promptConfiguratorState.currentPrompt = result.PromptText
			a.promptConfiguratorState.suggestedName = result.SuggestedName
			a.promptConfiguratorState.suggestedDesc = result.SuggestedDesc
			a.promptConfiguratorState.detectedMode = result.DetectedMode
			if a.promptConfiguratorState.promptArea != nil {
				a.promptConfiguratorState.promptArea.SetText(result.PromptText, false)
			}
			if a.promptConfiguratorState.refineInput != nil {
				a.promptConfiguratorState.refineInput.SetText("")
			}
		}
	})

	a.GetErrorHandler().ClearProgress()
	a.GetErrorHandler().ShowSuccess(a.ctx, "Prompt refined.")
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/prompt_configurator.go
git commit -m "feat(tui): wire refine input to streaming LLM refinement"
```

---

## Task 13 — Implement apply action

**Files:**
- Modify: `internal/tui/prompt_configurator.go`

- [ ] **Step 1: Add an input capture on the prompt area for the Apply key**

In `internal/tui/prompt_configurator.go`, locate `openPromptConfigurator`. After the `promptArea` is set up, insert:

```go
	state.promptArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Ctrl+R = regenerate (reuse last intent)
		if event.Key() == tcell.KeyCtrlR {
			intent := state.intentInput.GetText()
			if intent != "" {
				go a.generateConfiguratorPrompt(intent)
			}
			return nil
		}
		// Ctrl+S = save
		if event.Key() == tcell.KeyCtrlS {
			go a.savePromptFromConfigurator()
			return nil
		}
		// Ctrl+Enter or just Enter is for newline in TextArea — apply uses a dedicated key path.
		// Esc handled by parent panel
		if event.Key() == tcell.KeyEscape {
			a.closePromptConfigurator()
			return nil
		}
		return event
	})
```

- [ ] **Step 2: Implement `applyConfiguratorPrompt`**

Append to `internal/tui/prompt_configurator.go`:

```go
// applyConfiguratorPrompt runs the current prompt against the context defined when the panel was opened.
func (a *App) applyConfiguratorPrompt() {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	current := state.promptArea.GetText()
	if current == "" {
		a.GetErrorHandler().ShowWarning(a.ctx, "Prompt is empty — generate or type one first")
		return
	}

	switch state.ctx.mode {
	case "single":
		if state.ctx.messageID == "" {
			a.GetErrorHandler().ShowWarning(a.ctx, "No message context — Apply disabled in draft mode")
			return
		}
		go a.applyEphemeralPromptToMessage(state.ctx.messageID, current, state.suggestedName)
	case "bulk":
		if len(state.ctx.messageIDs) == 0 {
			a.GetErrorHandler().ShowWarning(a.ctx, "No messages in bulk context — Apply disabled")
			return
		}
		go a.applyEphemeralPromptToBulk(state.ctx.messageIDs, current, state.suggestedName)
	default:
		a.GetErrorHandler().ShowWarning(a.ctx, "No message context — save the prompt first, then use it from the picker")
	}
}

// applyEphemeralPromptToMessage runs an unsaved prompt against a single message.
// Uses AIService directly because the prompt has no DB ID yet.
func (a *App) applyEphemeralPromptToMessage(messageID string, promptText string, displayName string) {
	a.closePromptConfigurator()

	_, aiService, _, _, _, _, _, _, _, _, _, _ := a.GetServices()
	if aiService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "AI service not available")
		return
	}

	message, err := a.Client.GetMessageWithContent(messageID)
	if err != nil {
		a.GetErrorHandler().ShowError(a.ctx, "Failed to load message content")
		return
	}

	content := message.PlainText
	if len([]rune(content)) > 8000 {
		content = string([]rune(content)[:8000])
	}

	name := displayName
	if name == "" {
		name = "Custom Prompt"
	}

	a.QueueUpdateDraw(func() {
		if !a.aiSummaryVisible {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiSummaryVisible = true
		}
		if a.aiSummaryView != nil {
			a.aiPanelInPromptMode = true
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 %s ", name))
			a.aiSummaryView.SetText("🤖 Applying prompt...")
			a.aiSummaryView.ScrollToBeginning()
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	a.streamingCancel = cancel
	defer func() {
		cancel()
		a.streamingCancel = nil
	}()

	var b stringBuilder
	result, err := aiService.ApplyCustomPromptStream(ctx, content, promptText, nil, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		b.WriteString(token)
		if ctx.Err() == nil && a.aiSummaryView != nil {
			a.aiSummaryView.SetText(b.String())
			a.aiSummaryView.ScrollToEnd()
		}
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Apply canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply prompt: %v", err))
		return
	}

	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetText(result)
			a.aiSummaryView.ScrollToBeginning()
		}
	})
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s", name))
}

// applyEphemeralPromptToBulk runs an unsaved prompt against a bulk selection.
func (a *App) applyEphemeralPromptToBulk(messageIDs []string, promptText string, displayName string) {
	a.closePromptConfigurator()

	_, aiService, _, _, _, repository, _, _, _, _, _, _ := a.GetServices()
	if aiService == nil || repository == nil {
		a.GetErrorHandler().ShowError(a.ctx, "AI or repository service not available")
		return
	}

	name := displayName
	if name == "" {
		name = "Custom Bulk Prompt"
	}

	// Build combined content from messages, similar to BulkPromptService.combineMessageContents.
	var combined stringBuilder
	combined.WriteString("---START EMAILS---\n")
	for i, id := range messageIDs {
		msg, err := repository.GetMessage(a.ctx, id)
		if err != nil || msg == nil {
			continue
		}
		// Use snippet (already loaded by preloader) — see spec §8.1 "fast mode".
		combined.WriteString(fmt.Sprintf("---START EMAIL %d---\n", i+1))
		if msg.Snippet != "" {
			combined.WriteString(msg.Snippet)
		}
		combined.WriteString(fmt.Sprintf("\n---END EMAIL %d---\n", i+1))
	}
	combined.WriteString("---END OF EMAILS---\n")

	// Substitute placeholders.
	finalPrompt := promptText
	finalPrompt = strings.ReplaceAll(finalPrompt, "{{messages}}", combined.String())
	finalPrompt = strings.ReplaceAll(finalPrompt, "{{body}}", combined.String())

	a.QueueUpdateDraw(func() {
		if !a.aiSummaryVisible {
			if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
				split.ResizeItem(a.aiSummaryView, 0, 1)
			}
			a.aiSummaryVisible = true
		}
		if a.aiSummaryView != nil {
			a.aiPanelInPromptMode = true
			a.aiSummaryView.SetTitle(fmt.Sprintf(" 🤖 %s (%d msgs) ", name, len(messageIDs)))
			a.aiSummaryView.SetText("🤖 Applying bulk prompt...")
			a.SetFocus(a.aiSummaryView)
			a.currentFocus = "summary"
			a.updateFocusIndicators("summary")
		}
	})

	ctx, cancel := context.WithCancel(a.ctx)
	a.streamingCancel = cancel
	defer func() {
		cancel()
		a.streamingCancel = nil
	}()

	var b stringBuilder
	result, err := aiService.ApplyCustomPromptStream(ctx, combined.String(), finalPrompt, nil, func(token string) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		b.WriteString(token)
		if ctx.Err() == nil && a.aiSummaryView != nil {
			a.aiSummaryView.SetText(b.String())
			a.aiSummaryView.ScrollToEnd()
		}
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			a.GetErrorHandler().ShowInfo(a.ctx, "Bulk apply canceled")
			return
		}
		a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to apply bulk prompt: %v", err))
		return
	}

	a.QueueUpdateDraw(func() {
		if a.aiSummaryView != nil {
			a.aiSummaryView.SetText(result)
			a.aiSummaryView.ScrollToBeginning()
		}
	})
	a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Applied: %s (%d msgs)", name, len(messageIDs)))
}
```

Add these imports at the top of the file if not already present:

```go
import (
	"strings"
	stringBuilder "strings"  // Alias only used because we use Builder in code above
)
```

Note: `stringBuilder` is just a placeholder for `strings.Builder`. Simpler: use `var b strings.Builder` directly. Adjust the code above to replace `var b stringBuilder` with `var b strings.Builder` and `combined stringBuilder` with `combined strings.Builder`. Remove the alias import.

- [ ] **Step 2: Hook the Apply key in `app.go` global key dispatcher**

The Apply key in the configurator is `Enter` on a dedicated key handler outside the editable area. Since TextArea consumes Enter for newlines, we wire Apply via a global key handler when the configurator picker is active. In `internal/tui/keys.go`, find where ESC is handled for active pickers and add a check for `PickerPromptConfigurator`. Insert a case that intercepts the user-configured Apply key (default Enter ignored to allow newlines — we'll bind to a separate key like `Ctrl+G` for "go/apply").

Refine the spec: change `Keys.PromptApply` default from `"enter"` to `"ctrl+g"` to avoid clashing with TextArea newline. Update `internal/config/config.go`:

```go
		PromptApply:      "ctrl+g", // Ctrl+G = go/apply, doesn't clash with TextArea newline
```

Now wire it in `prompt_configurator.go` inside the `promptArea.SetInputCapture`:

```go
		// Ctrl+G = apply
		if event.Key() == tcell.KeyCtrlG {
			go a.applyConfiguratorPrompt()
			return nil
		}
```

Add this case before the existing Ctrl+R / Ctrl+S checks. Also update the status line text:

```go
	state.statusLine = tview.NewTextView().
		SetTextAlign(tview.AlignRight).
		SetText(" Ctrl+G apply  Ctrl+R refine  Ctrl+S save  Esc cancel ")
```

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/prompt_configurator.go internal/config/config.go
git commit -m "feat(tui): implement apply for ephemeral prompt (single + bulk)"
```

---

## Task 14 — Implement save dialog

**Files:**
- Modify: `internal/tui/prompt_configurator.go`

- [ ] **Step 1: Implement `savePromptFromConfigurator`**

Append to `internal/tui/prompt_configurator.go`:

```go
// savePromptFromConfigurator opens a modal dialog to save the current prompt to the library.
func (a *App) savePromptFromConfigurator() {
	state := a.promptConfiguratorState
	if state == nil {
		return
	}

	current := state.promptArea.GetText()
	if current == "" {
		a.GetErrorHandler().ShowWarning(a.ctx, "Cannot save empty prompt")
		return
	}

	_, _, _, _, _, _, promptService, _, _, _, _, _ := a.GetServices()
	if promptService == nil {
		a.GetErrorHandler().ShowError(a.ctx, "Prompt service not available")
		return
	}

	colors := a.GetComponentColors("prompts")
	bgColor := colors.Background.Color()

	nameInput := tview.NewInputField().
		SetLabel("Name: ").
		SetText(state.suggestedName).
		SetFieldWidth(40).
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	descInput := tview.NewInputField().
		SetLabel("Desc: ").
		SetText(state.suggestedDesc).
		SetFieldWidth(60).
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())
	catInput := tview.NewInputField().
		SetLabel("Cat:  ").
		SetText("custom").
		SetFieldWidth(20).
		SetLabelColor(colors.Title.Color()).
		SetFieldBackgroundColor(bgColor).
		SetFieldTextColor(colors.Text.Color())

	form := tview.NewFlex().SetDirection(tview.FlexRow)
	form.SetBackgroundColor(bgColor)
	form.SetBorder(true)
	form.SetTitle(" 💾 Save Prompt ")
	form.SetTitleColor(colors.Title.Color())
	form.AddItem(nameInput, 1, 0, true)
	form.AddItem(descInput, 1, 0, false)
	form.AddItem(catInput, 1, 0, false)

	helpText := tview.NewTextView().
		SetText(" Enter on Cat: save  Tab: next field  Esc: cancel ").
		SetTextColor(colors.Text.Color())
	helpText.SetBackgroundColor(bgColor)
	form.AddItem(helpText, 1, 0, false)

	doSave := func() {
		name := strings.TrimSpace(nameInput.GetText())
		desc := strings.TrimSpace(descInput.GetText())
		cat := strings.TrimSpace(catInput.GetText())
		if name == "" {
			a.GetErrorHandler().ShowWarning(a.ctx, "Name cannot be empty")
			return
		}
		if cat == "" {
			cat = "custom"
		}

		// Close the dialog by restoring focus to the prompt area.
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = state.container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.SetFocus(state.promptArea)

		go func() {
			id, err := promptService.CreatePrompt(a.ctx, name, desc, current, cat)
			if err != nil {
				a.GetErrorHandler().ShowError(a.ctx, fmt.Sprintf("Failed to save prompt: %v", err))
				return
			}
			a.GetErrorHandler().ShowSuccess(a.ctx, fmt.Sprintf("Saved prompt '%s' (id=%d)", name, id))
		}()
	}

	doCancel := func() {
		if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
			if a.labelsView != nil {
				split.RemoveItem(a.labelsView)
			}
			a.labelsView = state.container
			split.AddItem(a.labelsView, 0, 1, true)
			split.ResizeItem(a.labelsView, 0, 1)
		}
		a.SetFocus(state.promptArea)
	}

	nameInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			doCancel()
			return
		}
		if key == tcell.KeyEnter || key == tcell.KeyTab {
			a.SetFocus(descInput)
		}
	})
	descInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			doCancel()
			return
		}
		if key == tcell.KeyEnter || key == tcell.KeyTab {
			a.SetFocus(catInput)
		}
	})
	catInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			doCancel()
			return
		}
		if key == tcell.KeyEnter {
			doSave()
		}
	})

	// Replace the configurator panel with the save dialog temporarily.
	if split, ok := a.views["contentSplit"].(*tview.Flex); ok {
		if a.labelsView != nil {
			split.RemoveItem(a.labelsView)
		}
		a.labelsView = form
		split.AddItem(a.labelsView, 0, 1, true)
		split.ResizeItem(a.labelsView, 0, 1)
	}
	a.SetFocus(nameInput)
}
```

Confirm the `strings` import is present at the top of the file.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/prompt_configurator.go
git commit -m "feat(tui): implement save dialog for configurator prompts"
```

---

## Task 15 — Wire ESC handling for configurator picker

**Files:**
- Modify: `internal/tui/keys.go`

- [ ] **Step 1: Locate the global ESC dispatcher**

Run: `grep -n "isLabelsPickerActive\|setActivePicker(PickerNone)" internal/tui/keys.go | head -10`
Expected: shows the ESC handler section where picker dispatch occurs.

- [ ] **Step 2: Add a branch for the configurator**

In the ESC handler block, alongside existing picker checks (e.g., `isLabelsPickerActive()`, etc.), add:

```go
		if a.isPromptConfiguratorActive() {
			a.closePromptConfigurator()
			return nil
		}
```

Place this branch with the other picker-close branches. The exact location depends on the existing structure of the ESC dispatcher.

- [ ] **Step 3: Verify build and that no existing keybindings regressed**

Run: `make build`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/keys.go
git commit -m "feat(tui): handle Esc to close prompt configurator"
```

---

## Task 16 — Enrich single prompt picker with "Create new with AI"

**Files:**
- Modify: `internal/tui/prompts.go`

- [ ] **Step 1: Insert the special item as the first entry in the picker**

In `internal/tui/prompts.go`, locate `openPromptPicker`. Find the `reload` closure (around line 76). Modify the `reload` function so that the first item is always "Create new with AI", regardless of filter. Replace the body of `reload` with:

```go
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]

		// Always include "Create new with AI" as the first option.
		list.AddItem("✨ Create new with AI...", "Enter: open configurator", 0, func() {
			pctx := promptConfiguratorContext{
				mode:      "single",
				messageID: messageID,
			}
			a.closePromptPicker()
			a.openPromptConfigurator(pctx)
		})

		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			var icon string
			switch item.category {
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			default:
				icon = "📝"
			}

			display := fmt.Sprintf("%s %s", icon, item.name)
			promptID := item.id
			promptName := item.name

			list.AddItem(display, "Enter: apply", 0, func() {
				if a.logger != nil {
					a.logger.Printf("prompt picker: selected promptID=%d name=%s", promptID, promptName)
				}
				go a.applyPromptToMessage(messageID, promptID, promptName, message)
			})
		}
	}
```

- [ ] **Step 2: Relax the category filter**

In the same function, find the block at line 132-144 that filters out `bulk_analysis` prompts:

```go
		all = make([]promptItem, 0, len(prompts))
		for _, p := range prompts {
			// Skip bulk_analysis prompts for single message picker
			if p.Category == "bulk_analysis" {
				continue
			}
			all = append(all, promptItem{ ... })
		}
```

Remove the `if p.Category == "bulk_analysis" { continue }` check. The single picker will now show all prompts. (Both `{{body}}` and `{{messages}}` placeholders are handled at apply time.)

Result:

```go
		all = make([]promptItem, 0, len(prompts))
		for _, p := range prompts {
			all = append(all, promptItem{
				id:          p.ID,
				name:        p.Name,
				description: p.Description,
				category:    p.Category,
			})
		}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/prompts.go
git commit -m "feat(tui): enrich single prompt picker with 'Create new with AI'"
```

---

## Task 17 — Enrich bulk prompt picker with "Create new with AI"

**Files:**
- Modify: `internal/tui/bulk_prompts.go`

- [ ] **Step 1: Insert the special item at the top of the bulk picker**

In `internal/tui/bulk_prompts.go`, locate `openBulkPromptPicker` and the `reload` closure (around line 74). Modify the same way as the single picker — replace the `reload` body with:

```go
	reload := func(filter string) {
		list.Clear()
		visible = visible[:0]

		// "Create new with AI" as the first entry in bulk mode.
		messageIDs := make([]string, 0, len(a.selected))
		for id := range a.selected {
			messageIDs = append(messageIDs, id)
		}
		list.AddItem("✨ Create new with AI...", "Enter: open configurator", 0, func() {
			pctx := promptConfiguratorContext{
				mode:       "bulk",
				messageIDs: messageIDs,
			}
			a.closeBulkPromptPicker()
			a.openPromptConfigurator(pctx)
		})

		for _, item := range all {
			if filter != "" && !strings.Contains(strings.ToLower(item.name), strings.ToLower(filter)) &&
				!strings.Contains(strings.ToLower(item.description), strings.ToLower(filter)) {
				continue
			}
			visible = append(visible, item)

			var icon string
			switch item.category {
			case "bulk_analysis":
				icon = "🚀"
			case "summary":
				icon = "📄"
			case "analysis":
				icon = "📊"
			case "reply":
				icon = "💬"
			default:
				icon = "📝"
			}

			display := fmt.Sprintf("%s %s", icon, item.name)
			promptID := item.id
			promptName := item.name

			list.AddItem(display, "Enter: apply bulk prompt", 0, func() {
				if a.logger != nil {
					a.logger.Printf("bulk prompt picker: selected promptID=%d name=%s for %d messages", promptID, promptName, messageCount)
				}
				go a.applyBulkPrompt(promptID, promptName)
			})
		}
	}
```

- [ ] **Step 2: Relax the bulk_analysis category filter**

In the same function, find the block at lines 137-149 that only includes `bulk_analysis` prompts. Replace with:

```go
			all = make([]promptItem, 0, len(prompts))
			for _, p := range prompts {
				all = append(all, promptItem{
					id:          p.ID,
					name:        p.Name,
					description: p.Description,
					category:    p.Category,
				})
			}
```

The bulk picker will now show all prompts. Users can pick anything; the service swaps placeholders correctly.

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/bulk_prompts.go
git commit -m "feat(tui): enrich bulk prompt picker with 'Create new with AI'"
```

---

## Task 18 — Add commands `:prompt-new`, `:prompt-refine`, `:prompt-save`

**Files:**
- Modify: `internal/tui/commands.go`

- [ ] **Step 1: Add cases to `executeCommand`**

Find the `executeCommand` switch block in `internal/tui/commands.go` (around line 680). Insert these cases after the existing `prompt` case (line 775):

```go
	case "prompt-new", "pn":
		a.executePromptNewCommand(args)
	case "prompt-refine", "pr":
		a.executePromptRefineCommand(args)
	case "prompt-save", "ps":
		a.executePromptSaveCommand(args)
```

- [ ] **Step 2: Implement the three command handlers**

Append at the end of `internal/tui/commands.go`:

```go
// executePromptNewCommand opens the configurator with the current context.
// If a single message is selected, opens in single mode. If bulk mode is active, opens in bulk mode.
// Otherwise opens in draft mode.
func (a *App) executePromptNewCommand(args []string) {
	pctx := promptConfiguratorContext{}

	if a.bulkMode && len(a.selected) > 0 {
		ids := make([]string, 0, len(a.selected))
		for id := range a.selected {
			ids = append(ids, id)
		}
		pctx.mode = "bulk"
		pctx.messageIDs = ids
	} else if msgID := a.GetCurrentMessageID(); msgID != "" {
		pctx.mode = "single"
		pctx.messageID = msgID
	} else {
		pctx.mode = "draft"
	}

	a.openPromptConfigurator(pctx)
}

// executePromptRefineCommand applies a refinement to the currently active configurator prompt.
// Usage: :prompt-refine make output JSON
func (a *App) executePromptRefineCommand(args []string) {
	if a.promptConfiguratorState == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Open the configurator first (:prompt-new)")
		return
	}
	if len(args) == 0 {
		a.GetErrorHandler().ShowWarning(a.ctx, "Usage: :prompt-refine <refinement instruction>")
		return
	}
	refinement := strings.Join(args, " ")
	current := a.promptConfiguratorState.promptArea.GetText()
	if current == "" {
		a.GetErrorHandler().ShowWarning(a.ctx, "Generate a prompt first before refining")
		return
	}
	go a.refineConfiguratorPrompt(current, refinement)
}

// executePromptSaveCommand triggers the save dialog for the active configurator prompt.
func (a *App) executePromptSaveCommand(args []string) {
	if a.promptConfiguratorState == nil {
		a.GetErrorHandler().ShowWarning(a.ctx, "Open the configurator first (:prompt-new)")
		return
	}
	go a.savePromptFromConfigurator()
}
```

- [ ] **Step 3: Update `generateCommandSuggestion`**

Find `generateCommandSuggestion` (line 233 of `internal/tui/commands.go`). Find where existing prompt-related suggestions appear (search for the literal `"prompt"`). Add the new suggestions in the same idiomatic spot:

```go
	case strings.HasPrefix(buffer, "prompt-"), strings.HasPrefix(buffer, "pn"), strings.HasPrefix(buffer, "pr "), strings.HasPrefix(buffer, "ps"):
		switch {
		case strings.HasPrefix("prompt-new", buffer), strings.HasPrefix("pn", buffer):
			return "prompt-new"
		case strings.HasPrefix("prompt-refine", buffer), strings.HasPrefix("pr ", buffer):
			return "prompt-refine"
		case strings.HasPrefix("prompt-save", buffer), strings.HasPrefix("ps", buffer):
			return "prompt-save"
		}
```

If the existing structure differs significantly, follow the file's actual idiom. The intent is: as the user types `:pn` they see `:prompt-new` suggested.

- [ ] **Step 4: Verify build and tests**

Run: `go build ./... && go test ./internal/tui/ -run TestCommand -v`
Expected: build PASS. Existing tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat(tui): add :prompt-new, :prompt-refine, :prompt-save commands"
```

---

## Task 19 — Add unit tests for prompt_configurator.go

**Files:**
- Create: `internal/tui/prompt_configurator_test.go`

- [ ] **Step 1: Create the test file**

Create `internal/tui/prompt_configurator_test.go`:

```go
package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptConfiguratorTitle_SingleMode(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{mode: "single"})
	assert.Contains(t, got, "Prompt Configurator")
	assert.Contains(t, got, "1 msg")
}

func TestPromptConfiguratorTitle_BulkMode(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{
		mode:       "bulk",
		messageIDs: []string{"a", "b", "c"},
	})
	assert.Contains(t, got, "3 msgs scoped")
}

func TestPromptConfiguratorTitle_BulkWithCategory(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{
		mode:         "bulk",
		messageIDs:   []string{"a", "b"},
		categoryName: "Newsletters",
	})
	assert.Contains(t, got, "2 msgs from")
	assert.Contains(t, got, "Newsletters")
}

func TestPromptConfiguratorTitle_DraftMode(t *testing.T) {
	got := promptConfiguratorTitle(promptConfiguratorContext{mode: "draft"})
	assert.Contains(t, got, "draft only")
}
```

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/tui/ -run TestPromptConfiguratorTitle -v`
Expected: 4 PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/prompt_configurator_test.go
git commit -m "test(tui): add unit tests for prompt configurator title helpers"
```

---

## Task 20 — Update documentation

**Files:**
- Modify: `docs/KEYBOARD_SHORTCUTS.md`

- [ ] **Step 1: Add a new section for the configurator**

Run: `tail -30 docs/KEYBOARD_SHORTCUTS.md`
Expected: shows the end of the keyboard shortcuts doc.

Append a new section to `docs/KEYBOARD_SHORTCUTS.md`:

```markdown
## Prompt Configurator

The Prompt Configurator lets you generate, refine, and save AI prompts in natural language instead of writing templates by hand. It is accessed from the existing prompts picker (`p` key) — the first option in the list is now "✨ Create new with AI...".

### Inside the configurator panel

| Action | Default key | Config key |
|---|---|---|
| Apply current prompt to scoped context | `Ctrl+G` | `prompt_apply` |
| Refine prompt via LLM (after typing refinement) | `Ctrl+R` | `prompt_regenerate` |
| Save current prompt to library | `Ctrl+S` | `save_prompt` |
| Test prompt on 1 message (stretch — TBD) | `Ctrl+T` | `prompt_test` |
| Close configurator | `Esc` | (universal) |

### Commands

| Command | Alias | Description |
|---|---|---|
| `:prompt-new` | `:pn` | Open configurator with current context |
| `:prompt-refine <text>` | `:pr <text>` | Refine the active prompt with given instruction |
| `:prompt-save` | `:ps` | Open the save dialog for the active prompt |

### Notes

- All shortcuts above are configurable in `~/.config/giztui/config.json` under the `keys` block (e.g., `keys.save_prompt`).
- Saved prompts go to your shared prompt library — they appear in both single and bulk prompt pickers afterward.
- If you close the configurator without pressing `Ctrl+S`, the prompt is discarded.
```

- [ ] **Step 2: Verify the doc renders sensibly**

Run: `head -50 docs/KEYBOARD_SHORTCUTS.md && echo "---END HEAD---" && tail -50 docs/KEYBOARD_SHORTCUTS.md`
Expected: structure is consistent with existing format.

- [ ] **Step 3: Commit**

```bash
git add docs/KEYBOARD_SHORTCUTS.md
git commit -m "docs: document Prompt Configurator shortcuts and commands"
```

---

## Task 21 — Run full pre-commit check and fix any regressions

**Files:** N/A (verification task)

- [ ] **Step 1: Run pre-commit check**

Run: `make pre-commit-check`
Expected: PASS (fmt + vet + golangci-lint + essential tests).

- [ ] **Step 2: If anything fails, investigate**

If `golangci-lint` reports issues in the new code, fix them inline:
- Unused imports — remove
- Missing comments on exported types — add per Go conventions
- Cyclomatic complexity — extract helper functions

If existing tests regress, investigate the root cause. Do not skip or comment out tests.

- [ ] **Step 3: Verify the binary actually launches**

Run: `make build && ls bin/giztui`
Expected: binary exists.

- [ ] **Step 4: Manual smoke test (optional but recommended)**

Launch the app with a test account, then:
1. Open an email → press `p` → confirm "✨ Create new with AI..." appears.
2. Select it → confirm the configurator panel opens.
3. Type an intent → press Enter → confirm the LLM generates a prompt.
4. Type a refinement → press Enter on the refine line → confirm regeneration.
5. Press `Ctrl+S` → confirm save dialog appears.
6. Save → confirm status bar shows success.
7. Reopen the picker → confirm the saved prompt is present.

- [ ] **Step 5: Final commit (if any small fixes were applied)**

If any fixes were needed in step 2, commit them:

```bash
git add -u
git commit -m "fix: address pre-commit findings for prompt configurator"
```

Otherwise no commit needed.

---

## Self-review checklist

Run through this list with fresh eyes before considering the plan done:

- [ ] **Spec coverage**: every section of the spec maps to at least one task.
  - §6 (architecture) → Tasks 1–9 (service + wiring)
  - §7.1 (enriched picker) → Tasks 16, 17
  - §7.2 (configurator) → Tasks 10–15
  - §7.4 (deferred to Plan 2)
  - §8.5 (data flow: create new) → Tasks 11–14
  - §9 (error handling) → embedded throughout
  - §10 (configurable keys) → Task 7
  - §12 (commands) → Task 18
  - §13 (testing) → Tasks 2–6, 19, 21
- [ ] **No placeholders**: every step has either exact code or exact commands.
- [ ] **Type consistency**: `PromptGenerationOptions`, `GeneratedPrompt`, `promptConfiguratorContext`, `promptConfiguratorState` are used consistently across tasks.
- [ ] **CLAUDE.md compliance**:
  - All status messages go through `a.GetErrorHandler().Show*` — yes ✓
  - No `QueueUpdateDraw` inside streaming callbacks — yes ✓
  - No `QueueUpdateDraw` inside ESC/close paths — yes ✓
  - `ActivePicker` enum used (not booleans) — yes ✓
  - Theming via `GetComponentColors` — yes ✓
  - Command parity (every action has both shortcut and command) — yes ✓
- [ ] **Test coverage**: service layer has unit tests, configurator has unit tests for the title helper, integration test deferred to manual smoke test for now (full TUI integration test deferred to a follow-up).

Known follow-ups (not blockers for this plan):
1. Full TUI integration test with mock AIService driving the configurator flow end-to-end.
2. "Test on 1 message" feature (Ctrl+T) — stretch goal, deferred.
3. The `Detected mode` from `GeneratedPrompt` is captured but not yet used to filter pickers — will be wired in Plan 2 if needed.

---

**Plan complete and saved to `docs/superpowers/plans/2026-06-06-prompt-configurator.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints for review.

**Which approach?**
