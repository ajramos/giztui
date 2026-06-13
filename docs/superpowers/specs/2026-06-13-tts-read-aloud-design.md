# Text-to-speech: read aloud (idea E)

**Date:** 2026-06-13
**Status:** Approved (design)

## Problem

The user wants to read emails and AI analyses/summaries **aloud** (accessibility, hands-free
triage). There is no audio output today.

## Goal

A configurable, context-aware key reads the focused panel's text aloud via a local neural TTS
engine (Piper), playing the audio through the OS; pressing it again stops. Opt-in (Piper must be
installed) with clear setup docs and a helpful message when unconfigured.

## Decisions (confirmed with user)

- **Engine:** Piper (local neural, CPU, Spanish-first), the openclaw pattern — a decoupled Go
  `Synthesizer` interface + external-process impl, swappable later.
- **Read-aloud only (v1):** Piper → temp WAV → **play** → delete. No ffmpeg/OGG, no saving a voice
  note (that's a future "idea H").
- **Opt-in via config** (`tts.piper_path` / `tts.model_path`); if absent, the speak key warns
  instead of failing. No bundling, no PATH autodetection.
- **Trigger:** a context-aware key (`keys.speak`, unbound by default) reads the focused panel;
  pressing it while speaking stops.
- **Docs are a first-class deliverable** (`docs/TTS.md`) — per the CLAUDE.md Definition of Done.

## Architecture

### A) Engine — new package `internal/tts/`

- `types.go`: `SynthesizeOptions{ ModelPath string }`, `SynthesisResult{ AudioPath, Engine, Model string }`.
- `synthesizer.go`: `Synthesizer interface { Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesisResult, error) }`.
- `external_piper.go`: `ExternalPiperSynthesizer{ PiperPath string }`. `Synthesize`:
  validates non-empty text and that `PiperPath`/`opts.ModelPath` exist; runs
  `exec.CommandContext(ctx, piperPath, "--model", modelPath, "--output_file", tmpWav)` with the
  text on **stdin**; captures stderr; returns `SynthesisResult{AudioPath: tmpWav, Engine:"piper", Model: modelPath}`.
  Distinguishes config error (missing binary/model) vs process error vs timeout.
- `player.go`: `Player interface { Play(ctx, audioPath) error }` + `OSPlayer` using a
  `switch runtime.GOOS` (`afplay` macOS, `paplay`→`aplay` fallback Linux) via `exec.CommandContext`
  (so a cancelled ctx kills playback).
- `errors.go`: `ErrNotConfigured`, `ErrEmptyText` (typed sentinels).

### B) Orchestration — `internal/services/speech_service.go`

`SpeechService interface { Speak(ctx context.Context, text string) error; Stop(); IsConfigured() bool }`.
`SpeechServiceImpl` holds a `tts.Synthesizer`, a `tts.Player`, the configured piper/model paths, a
`sync.Mutex` + a `context.CancelFunc` for the in-flight Speak.
- `IsConfigured()`: piper path and model path are both set and exist on disk.
- `Speak(ctx, text)`: if `!IsConfigured()` → `tts.ErrNotConfigured`; cancel any prior Speak;
  derive a cancellable child ctx (store its cancel); synthesize → play; delete the temp WAV after.
  Runs to completion under the child ctx; the TUI calls it in a goroutine.
- `Stop()`: invoke the stored cancel (kills synth/playback via ctx), clear it.

### C) Config — `internal/config/config.go`

`TTSConfig{ Enabled bool; PiperPath string; ModelPath string }`; `Config.TTS`; default
`{Enabled:false}`. New `KeyBindings.Speak string` (json `speak`), default `""` (unbound — TTS is
opt-in and needs setup; the user binds it). Surfaced to existing configs via `:config migrate`.

### D) TUI wiring

- App: `speechService services.SpeechService` field + `GetSpeechService()` accessor (NOT added to
  the 12-tuple `GetServices()` — separate accessor like `GetAnalyzerRulesService`, to avoid
  rippling every call site); init in `initServices()` (Piper synth + OS player + config paths).
- `internal/tui/speech.go` (new): `focusedSpeakText() string` — if the focused primitive is a
  `*tview.TextView`, return `GetText(true)` (tags stripped) trimmed, else `""` (covers reader, AI
  summary, action-plan digest uniformly). `toggleSpeak()`: if `speechService` is speaking → `Stop()`;
  else grab `focusedSpeakText()`; if empty → info "nothing to read"; if `!IsConfigured()` →
  warning with the config hint; else `go speechService.Speak(ctx, text)` + track speaking state.
- `keys.go`: `case a.Keys.Speak:` (mirrors the optional `auto_refresh` key — only matches when
  bound) → `a.toggleSpeak()`; `return true`.

### E) Docs — `docs/TTS.md` (first-class) + CONFIGURATION.md + `:help`

`docs/TTS.md`: install Piper per OS (release/brew/download), download a Spanish voice model
(`es_ES-carlfm-x_low.onnx` + its `.json`) and where to put it (e.g. `~/.config/giztui/piper/`),
the audio player dependency (`afplay` built-in on macOS; `paplay`/`aplay` + pulseaudio/alsa-utils
on Linux), the `tts.*` config keys + example, and a note that `:config migrate` adds the keys.
Link it from README and the CONFIGURATION.md TTS section. Add the speak key to `:help`.

## Error handling / threading

- Synthesis + playback run on a worker goroutine (the TUI calls `Speak` via `go`); `Stop()` cancels
  the ctx (kills the external processes). No `QueueUpdateDraw` in this path beyond the standard
  `go ...ShowError/ShowInfo` for user messages.
- `ErrNotConfigured` / empty text → user-facing message, no process spawned.
- Temp WAV created in the OS temp dir; deleted after playback (and on error).

## Testing

- `internal/tts`: `TestExternalPiper_Validation` — empty text → `ErrEmptyText`; missing
  piper/model path → `ErrNotConfigured` (no process executed). (Real synthesis is an E2E/manual
  step — needs Piper installed.)
- `internal/services`: `TestSpeechService_IsConfigured` (paths set+exist vs not, using temp files);
  `TestSpeechService_SpeakStop` with stub `Synthesizer`+`Player` (no real processes) — `Speak`
  invokes synth then play; `Stop` cancels.
- `internal/tui`: `TestFocusedSpeakText` — focused `*tview.TextView` returns its stripped text;
  non-text focus → "".

## Out of scope (future ideas)

- **Idea H:** ffmpeg→OGG/Opus + save/send the audio as a voice note (the openclaw file-gen use case).
- Streaming synthesis, voice/rate selection UI, PATH autodetection, bundling Piper.
- STT (the other openclaw prompt) — separate idea (transcribe voice notes).
