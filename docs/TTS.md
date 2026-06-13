# Text-to-Speech (Read Aloud)

GizTUI can read the focused panel aloud — the message reader, an AI summary, or an Action Plan
digest — using a **local** neural TTS engine ([Piper](https://github.com/rhasspy/piper)). It is
**opt-in**: nothing is bundled, and you point GizTUI at a Piper binary + a voice model.

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
