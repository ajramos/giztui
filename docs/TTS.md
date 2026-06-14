# Text-to-Speech (Read Aloud)

GizTUI can read the focused panel aloud — the message reader, an AI summary, or an Action Plan
digest. It is **opt-in** (bind a key to it) and has two engines, selected by `tts.engine`:

| `tts.engine` | Engine | Notes |
|--------------|--------|-------|
| `"auto"` (default) | macOS → `say`, others → `piper` | Auto-detects the OS |
| `"say"` | macOS built-in `say` | **Zero setup** — no binary, no model, no deps |
| `"piper"` | [Piper](https://github.com/rhasspy/piper) neural TTS | Cross-platform, better voices; needs a binary + voice model |

## macOS quick start (`say`, zero setup)

On macOS the default `"auto"` resolves to the built-in `say` command — **nothing to install**. Just
bind a key:

```json
"keys": { "speak": "ctrl+e" }
```

Optionally pick a voice (e.g. a Spanish one — see `say -v '?'` for the list) and pin the engine:

```json
"tts": { "enabled": true, "engine": "say", "voice": "Mónica" }
```

### Read each email in its own language

GizTUI auto-detects the language of the text and picks a matching voice, so an English email is
read by an English voice and a Spanish one by a Spanish voice. Map ISO 639-1 codes to voices
(`say -v '?'` lists voices with their language tag):

```json
"tts": {
  "enabled": true,
  "engine": "say",
  "voice": "Mónica",
  "voices": { "en": "Samantha", "es": "Mónica" }
}
```

Detection is **restricted to the languages you list**, which makes it accurate even on short
emails. Unrecognized languages fall back to `voice` (or the system default). For the Piper engine
use `models` instead (ISO 639-1 → `.onnx` path), e.g. `"models": {"en":"…/en.onnx","es":"…/es.onnx"}`.

That's it — focus a message and press your speak key. Everything below is only for the **Piper**
engine (better voices, or Linux).

## 1. Install Piper

- **macOS:** `brew install piper-tts`, or download a release binary from
  <https://github.com/rhasspy/piper/releases> and place it somewhere (note the path).
- **Linux:** download the release binary from the same page (or your distro's package), e.g. to
  `~/.config/giztui/piper/piper`.

## 2. Download a voice model

Grab a Spanish voice (model `.onnx` **and** its `.onnx.json`) from the Piper voices repository
<https://huggingface.co/rhasspy/piper-voices> — e.g. `es_ES-carlfm-x_low`. Put both files together,
for example:

```
~/.config/giztui/piper/es_ES-carlfm-x_low.onnx
~/.config/giztui/piper/es_ES-carlfm-x_low.onnx.json
```

## 3. Audio playback dependency

GizTUI plays the generated audio with the OS player:

- **macOS:** `afplay` (built in — nothing to install).
- **Linux:** `paplay` (PulseAudio) or `aplay` (ALSA). Install one:
  `sudo apt install pulseaudio-utils` or `sudo apt install alsa-utils`.

## 4. Configure GizTUI

Add to `~/.config/giztui/config.json` (run `:config migrate` / `giztui --migrate-config` to get the
`tts` block added automatically, then edit it):

```json
"tts": {
  "enabled": true,
  "engine": "piper",
  "piper_path": "~/.config/giztui/piper/piper",
  "model_path": "~/.config/giztui/piper/es_ES-carlfm-x_low.onnx"
},
"keys": {
  "speak": "R"
}
```

`keys.speak` is **unbound by default** — pick any free key for it.

## 5. Use it

Focus a text panel (message reader, AI summary, or Action Plan digest) and press your `speak` key.
GizTUI synthesizes the text with Piper and plays it. Press the key again to **stop**.

If TTS isn't configured (paths missing), pressing the key shows a hint instead of failing.

## Notes & limitations

- First version reads the **full text** (no streaming) and plays through the speakers; it does not
  save or send an audio file (see the roadmap for a future "voice note export").
- Spanish-first, but any Piper voice model works — point `model_path` at it.
- The engine is decoupled internally (`internal/tts`), so a different TTS backend can replace Piper
  later without changing the rest of the app.
