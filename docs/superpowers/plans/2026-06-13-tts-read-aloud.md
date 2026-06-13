# Text-to-Speech Read-Aloud Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A context-aware key reads the focused panel's text aloud via local Piper TTS (play through the OS), toggling to stop; opt-in via config with setup docs.

**Architecture:** A decoupled `internal/tts` engine (`Synthesizer` = Piper external process → WAV; `Player` = OS audio command). A `SpeechService` composes synth+play with start/stop. Config-gated; a `keys.speak` handler reads the focused TextView.

**Tech Stack:** Go, Piper (external local binary), OS audio (`afplay`/`paplay`/`aplay`), tview.

Spec: `docs/superpowers/specs/2026-06-13-tts-read-aloud-design.md`

---

### Task 1: Config — TTSConfig + speak key

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/config/config_test.go`:

```go
func TestDefaultConfig_TTS(t *testing.T) {
	c := DefaultConfig()
	if c.TTS.Enabled {
		t.Error("tts.enabled should default to false (opt-in)")
	}
	if c.Keys.Speak != "" {
		t.Errorf("keys.speak should default to unbound, got %q", c.Keys.Speak)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestDefaultConfig_TTS -v`
Expected: FAIL — `c.TTS` / `c.Keys.Speak` undefined.

- [ ] **Step 3: Add the config types/fields**

In `internal/config/config.go`, add the struct (near other sub-structs):

```go
// TTSConfig controls opt-in local text-to-speech (Piper) for reading content aloud.
type TTSConfig struct {
	Enabled   bool   `json:"enabled"`
	PiperPath string `json:"piper_path"` // path to the piper binary
	ModelPath string `json:"model_path"` // path to the .onnx voice model
}
```

Add to `Config`:
```go
	// Text-to-speech (opt-in, local Piper)
	TTS TTSConfig `json:"tts"`
```

Add to the `KeyBindings` struct (after `AutoRefresh`, line ~271):
```go
	Speak                  string `json:"speak"` // Read the focused panel aloud (TTS); unbound by default
```

In `DefaultConfig()`'s returned `&Config{...}`, add:
```go
		TTS: TTSConfig{Enabled: false},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -run TestDefaultConfig_TTS -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add tts config block + speak key (opt-in)"
```

---

### Task 2: Engine — `internal/tts` package

**Files:**
- Create: `internal/tts/types.go`, `internal/tts/errors.go`, `internal/tts/synthesizer.go`, `internal/tts/external_piper.go`, `internal/tts/player.go`
- Test: `internal/tts/external_piper_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tts/external_piper_test.go`:

```go
package tts

import (
	"context"
	"errors"
	"testing"
)

func TestExternalPiper_Validation(t *testing.T) {
	s := &ExternalPiperSynthesizer{PiperPath: "/nonexistent/piper"}

	if _, err := s.Synthesize(context.Background(), "  ", SynthesizeOptions{ModelPath: "/nonexistent/model.onnx"}); !errors.Is(err, ErrEmptyText) {
		t.Fatalf("empty text should return ErrEmptyText, got %v", err)
	}
	if _, err := s.Synthesize(context.Background(), "hola", SynthesizeOptions{ModelPath: "/nonexistent/model.onnx"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("missing piper/model should return ErrNotConfigured, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tts/ -run TestExternalPiper_Validation -v`
Expected: FAIL — package/types undefined.

- [ ] **Step 3: Create the package files**

`internal/tts/types.go`:
```go
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
```

`internal/tts/errors.go`:
```go
package tts

import "errors"

var (
	// ErrEmptyText is returned when there is nothing to synthesize.
	ErrEmptyText = errors.New("tts: empty text")
	// ErrNotConfigured is returned when the engine binary or model is missing.
	ErrNotConfigured = errors.New("tts: engine not configured (piper binary or model missing)")
)
```

`internal/tts/synthesizer.go`:
```go
package tts

import "context"

// Synthesizer turns text into an audio file. Decoupled so the engine can be swapped.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesisResult, error)
}
```

`internal/tts/external_piper.go`:
```go
package tts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ExternalPiperSynthesizer runs the local Piper binary: text on stdin → a temp WAV file.
type ExternalPiperSynthesizer struct {
	PiperPath string
}

func (s *ExternalPiperSynthesizer) Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesisResult, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyText
	}
	if !fileExists(s.PiperPath) || !fileExists(opts.ModelPath) {
		return nil, ErrNotConfigured
	}
	tmp, err := os.CreateTemp("", "giztui-tts-*.wav")
	if err != nil {
		return nil, fmt.Errorf("tts: temp file: %w", err)
	}
	tmp.Close()
	wav := tmp.Name()

	cmd := exec.CommandContext(ctx, s.PiperPath, "--model", opts.ModelPath, "--output_file", wav) // #nosec G204 -- piper path is operator-configured
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(wav)
		return nil, fmt.Errorf("tts: piper failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return &SynthesisResult{AudioPath: wav, Engine: "piper", Model: opts.ModelPath}, nil
}

func fileExists(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
```

`internal/tts/player.go`:
```go
package tts

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// Player plays an audio file through the OS. Killable via the context.
type Player interface {
	Play(ctx context.Context, audioPath string) error
}

// OSPlayer uses the platform's audio CLI.
type OSPlayer struct{}

func (OSPlayer) Play(ctx context.Context, audioPath string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "afplay", audioPath).Run() // #nosec G204 -- audioPath is a temp file we created
	case "linux":
		if _, err := exec.LookPath("paplay"); err == nil {
			return exec.CommandContext(ctx, "paplay", audioPath).Run() // #nosec G204
		}
		return exec.CommandContext(ctx, "aplay", audioPath).Run() // #nosec G204
	default:
		return fmt.Errorf("tts: audio playback unsupported on %s", runtime.GOOS)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/tts/ -run TestExternalPiper_Validation -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tts/
git commit -m "feat(tts): internal/tts engine — Piper synthesizer + OS player (decoupled)"
```

---

### Task 3: `SpeechService`

**Files:**
- Modify: `internal/services/interfaces.go` (add `SpeechService`)
- Create: `internal/services/speech_service.go`
- Test: `internal/services/speech_service_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/services/speech_service_test.go`:

```go
package services

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ajramos/giztui/internal/tts"
)

type stubSynth struct{ called bool }

func (s *stubSynth) Synthesize(ctx context.Context, text string, opts tts.SynthesizeOptions) (*tts.SynthesisResult, error) {
	s.called = true
	return &tts.SynthesisResult{AudioPath: "/tmp/x.wav", Engine: "stub"}, nil
}

type stubPlayer struct {
	mu     sync.Mutex
	played bool
}

func (p *stubPlayer) Play(ctx context.Context, audioPath string) error {
	p.mu.Lock()
	p.played = true
	p.mu.Unlock()
	return nil
}

func TestSpeechService_IsConfigured(t *testing.T) {
	dir := t.TempDir()
	piper := filepath.Join(dir, "piper")
	model := filepath.Join(dir, "m.onnx")
	_ = os.WriteFile(piper, []byte("x"), 0600)
	_ = os.WriteFile(model, []byte("x"), 0600)

	s := NewSpeechService(&stubSynth{}, &stubPlayer{}, piper, model)
	if !s.IsConfigured() {
		t.Fatal("should be configured when both paths exist")
	}
	s2 := NewSpeechService(&stubSynth{}, &stubPlayer{}, piper, filepath.Join(dir, "missing.onnx"))
	if s2.IsConfigured() {
		t.Fatal("should NOT be configured when the model is missing")
	}
}

func TestSpeechService_SpeakStop(t *testing.T) {
	dir := t.TempDir()
	piper := filepath.Join(dir, "piper")
	model := filepath.Join(dir, "m.onnx")
	_ = os.WriteFile(piper, []byte("x"), 0600)
	_ = os.WriteFile(model, []byte("x"), 0600)
	syn := &stubSynth{}
	pl := &stubPlayer{}
	s := NewSpeechService(syn, pl, piper, model)

	if err := s.Speak(context.Background(), "hola"); err != nil {
		t.Fatalf("Speak error: %v", err)
	}
	if !syn.called || !pl.played {
		t.Fatal("Speak should synthesize then play")
	}
	s.Stop() // must not panic when idle/after completion
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/services/ -run 'TestSpeechService' -v`
Expected: FAIL — `NewSpeechService` undefined.

- [ ] **Step 3: Add the interface + implementation**

In `internal/services/interfaces.go`, add (near the other service interfaces; ensure `context` imported — it is):
```go
// SpeechService reads text aloud via a local TTS engine.
type SpeechService interface {
	Speak(ctx context.Context, text string) error
	Stop()
	IsConfigured() bool
	IsSpeaking() bool
}
```

Create `internal/services/speech_service.go`:
```go
package services

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/ajramos/giztui/internal/tts"
)

// SpeechServiceImpl composes a Synthesizer + Player and manages start/stop.
type SpeechServiceImpl struct {
	synth     tts.Synthesizer
	player    tts.Player
	piperPath string
	modelPath string

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewSpeechService(synth tts.Synthesizer, player tts.Player, piperPath, modelPath string) *SpeechServiceImpl {
	return &SpeechServiceImpl{synth: synth, player: player, piperPath: piperPath, modelPath: modelPath}
}

func (s *SpeechServiceImpl) IsConfigured() bool {
	return fileExists(s.piperPath) && fileExists(s.modelPath)
}

func (s *SpeechServiceImpl) IsSpeaking() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

func (s *SpeechServiceImpl) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *SpeechServiceImpl) Speak(ctx context.Context, text string) error {
	if strings.TrimSpace(text) == "" {
		return tts.ErrEmptyText
	}
	if !s.IsConfigured() {
		return tts.ErrNotConfigured
	}
	s.Stop() // cancel any in-flight speech
	cctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.cancel != nil {
			s.cancel = nil
		}
		s.mu.Unlock()
		cancel()
	}()

	res, err := s.synth.Synthesize(cctx, text, tts.SynthesizeOptions{ModelPath: s.modelPath})
	if err != nil {
		return err
	}
	defer func() {
		if res.AudioPath != "" {
			_ = os.Remove(res.AudioPath)
		}
	}()
	return s.player.Play(cctx, res.AudioPath)
}

// fileExists mirrors the tts helper for path checks.
func fileExists(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
```

(Note: `fileExists` is unexported and package-local to `services`; the `tts` package has its own. No conflict.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/services/ -run 'TestSpeechService' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/services/interfaces.go internal/services/speech_service.go internal/services/speech_service_test.go
git commit -m "feat(services): SpeechService (synth + play + start/stop)"
```

---

### Task 4: App wiring + TUI key

**Files:**
- Modify: `internal/tui/app.go` (field, accessor, initServices)
- Create: `internal/tui/speech.go`
- Modify: `internal/tui/keys.go` (speak key case)
- Test: `internal/tui/speech_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/speech_test.go`:

```go
package tui

import (
	"testing"

	"github.com/derailed/tview"
)

func TestFocusedSpeakText(t *testing.T) {
	a := &App{Application: tview.NewApplication()}
	tv := tview.NewTextView().SetText("read me aloud")
	a.SetFocus(tv)
	if got := a.focusedSpeakText(); got != "read me aloud" {
		t.Fatalf("focused TextView text = %q, want 'read me aloud'", got)
	}

	// Non-text focus → empty.
	a.SetFocus(tview.NewList())
	if got := a.focusedSpeakText(); got != "" {
		t.Fatalf("non-text focus should yield empty, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestFocusedSpeakText -v`
Expected: FAIL — `focusedSpeakText` undefined.

- [ ] **Step 3: Add the App field + accessor + init**

In `internal/tui/app.go`, add to the service-field block (near `autoRefreshService`):
```go
	speechService           services.SpeechService
```

Add an accessor (near `GetAnalyzerRulesService`):
```go
// GetSpeechService returns the text-to-speech service (may be unconfigured).
func (a *App) GetSpeechService() services.SpeechService { return a.speechService }
```

In `initServices()` (after the auto-refresh block), construct it:
```go
	a.speechService = services.NewSpeechService(
		&tts.ExternalPiperSynthesizer{PiperPath: a.Config.TTS.PiperPath},
		tts.OSPlayer{},
		a.Config.TTS.PiperPath,
		a.Config.TTS.ModelPath,
	)
```
Add the import `"github.com/ajramos/giztui/internal/tts"` to `app.go`.

- [ ] **Step 4: Create `speech.go`**

Create `internal/tui/speech.go`:
```go
package tui

import (
	"strings"

	"github.com/ajramos/giztui/internal/tts"
	"github.com/derailed/tview"
)

// focusedSpeakText returns the text of the focused panel when it is a TextView (reader, AI
// summary, action-plan digest), tags stripped; otherwise "".
func (a *App) focusedSpeakText() string {
	if tv, ok := a.GetFocus().(*tview.TextView); ok {
		return strings.TrimSpace(tv.GetText(true))
	}
	return ""
}

// toggleSpeak reads the focused panel aloud, or stops if already speaking.
func (a *App) toggleSpeak() {
	svc := a.GetSpeechService()
	if svc == nil {
		return
	}
	if svc.IsSpeaking() {
		svc.Stop()
		go a.GetErrorHandler().ShowInfo(a.ctx, "🔇 Stopped reading")
		return
	}
	if !svc.IsConfigured() {
		go a.GetErrorHandler().ShowWarning(a.ctx, "TTS not configured — set tts.piper_path and tts.model_path (see docs/TTS.md)")
		return
	}
	text := a.focusedSpeakText()
	if text == "" {
		go a.GetErrorHandler().ShowInfo(a.ctx, "Nothing to read here")
		return
	}
	go func() {
		if err := svc.Speak(a.ctx, text); err != nil && err != tts.ErrEmptyText {
			a.GetErrorHandler().ShowError(a.ctx, "TTS failed: "+err.Error())
		}
	}()
	go a.GetErrorHandler().ShowInfo(a.ctx, "🔊 Reading aloud…")
}
```

- [ ] **Step 5: Wire the key in `keys.go`**

In `internal/tui/keys.go`, after the `case a.Keys.AutoRefresh:` block, add:
```go
	case a.Keys.Speak:
		if a.logger != nil {
			a.logger.Printf("Configurable shortcut: '%s' -> speak", key)
		}
		a.toggleSpeak()
		return true
```
(Like `AutoRefresh`, `Keys.Speak` defaults to `""` so the case can't match until bound.)

- [ ] **Step 6: Run test + build**

Run: `go test ./internal/tui/ -run TestFocusedSpeakText -v && go build ./...`
Expected: PASS, build success.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go internal/tui/speech.go internal/tui/keys.go internal/tui/speech_test.go
git commit -m "feat(tui): contextual speak key reads the focused panel aloud (TTS)"
```

---

### Task 5: Docs — `docs/TTS.md` + CONFIGURATION.md + `:help`

**Files:**
- Create: `docs/TTS.md`
- Modify: `docs/CONFIGURATION.md`, `internal/tui/app.go` (`generateHelpText`)

- [ ] **Step 1: Write `docs/TTS.md`**

Create `docs/TTS.md` with: what it does (read the focused panel aloud), prerequisites and install
steps per OS:
- **Piper**: download the release binary (github.com/rhasspy/piper) or `brew install piper`
  (macOS); place it somewhere and note the path.
- **Voice model**: download a Spanish voice (e.g. `es_ES-carlfm-x_low.onnx` + the matching
  `.onnx.json`) from the Piper voices repository; put both in e.g. `~/.config/giztui/piper/`.
- **Audio playback**: macOS has `afplay` built in; Linux needs `paplay` (PulseAudio) or `aplay`
  (`sudo apt install pulseaudio-utils` or `alsa-utils`).
- **Config** (`~/.config/giztui/config.json`):
  ```json
  "tts": {
    "enabled": true,
    "piper_path": "~/.config/giztui/piper/piper",
    "model_path": "~/.config/giztui/piper/es_ES-carlfm-x_low.onnx"
  },
  "keys": { "speak": "R" }
  ```
  (choose any free key for `speak`).
- Note: run `:config migrate` (or `giztui --migrate-config`) to add the `tts` keys to an existing config.
- Usage: focus the reader / AI summary / digest panel and press your speak key; press again to stop.

- [ ] **Step 2: Link from CONFIGURATION.md**

In `docs/CONFIGURATION.md`, add a short "Text-to-speech" subsection pointing to `docs/TTS.md` and
listing the `tts.*` keys (mirroring the table style used for other config blocks).

- [ ] **Step 3: Add to `:help`**

In `internal/tui/app.go` `generateHelpText`, add (near the other configurable-key notes):
```go
	fmt.Fprintf(&help, "    %-8s  🔊  Read the focused panel aloud (TTS; set keys.speak — see docs/TTS.md)\n", a.Keys.Speak)
```
(Place it where the configurable shortcuts are listed; if `a.Keys.Speak` is empty it still renders
a harmless blank key cell — acceptable, or guard with `if a.Keys.Speak != ""`.)

- [ ] **Step 4: Commit**

```bash
git add docs/TTS.md docs/CONFIGURATION.md internal/tui/app.go
git commit -m "docs: TTS setup guide + config + help entry"
```

---

### Task 6: Full verification

**Files:** none (verification only)

- [ ] **Step 1: Pre-commit gate**

Run: `make pre-commit-check`
Expected: fmt + vet + lint + essential tests pass. (Piper isn't installed in CI — the engine test
only exercises validation paths, which need no binary.)

- [ ] **Step 2: Targeted + leak check**

Run: `go test ./internal/tts/ ./internal/services/ ./internal/config/ ./internal/tui/ ./test/helpers/ 2>&1 | tail -6`
Expected: all `ok`.

- [ ] **Step 3: Build**

Run: `make build`
Expected: success.

(Live E2E — install Piper + a model, set `tts.*`, bind `keys.speak`, focus the reader and press it;
the email is read aloud; press again to stop — deferred to the user's E2E sweep.)

---

## Self-review notes

- **Spec coverage:** config + speak key (Task 1), `internal/tts` engine Synthesizer/Piper/Player/errors
  (Task 2), `SpeechService` synth+play+start/stop+IsConfigured/IsSpeaking (Task 3), App field +
  accessor (not in the 12-tuple) + initServices + `focusedSpeakText`/`toggleSpeak` + key (Task 4),
  `docs/TTS.md` + config + `:help` (Task 5), verification (Task 6). All spec sections mapped.
  Added `IsSpeaking()` to the interface (needed by the toggle) — a small, consistent extension.
- **Type consistency:** `Synthesizer.Synthesize(ctx, text, SynthesizeOptions) (*SynthesisResult, error)`,
  `Player.Play(ctx, audioPath)`, `NewSpeechService(synth, player, piperPath, modelPath)`,
  `SpeechService{Speak, Stop, IsConfigured, IsSpeaking}`, `focusedSpeakText`/`toggleSpeak`,
  `Config.TTS`/`Keys.Speak` — consistent across tasks.
- **No placeholders:** every code step shows full code; commands have expected output.
- **gosec:** the `exec.CommandContext` calls carry `#nosec G204` with rationale (operator-configured
  piper path; temp WAV we created) — matches the existing link/attachment service pattern.
