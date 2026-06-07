# HTML → Markdown Email Rendering — Brainstorm & PoC Plan

- **Date:** 2026-06-07
- **Status:** Design APPROVED (2026-06-07). Direction B chosen. Ready for implementation plan.
- **Branch:** to be created off `main` (e.g. `feat/markdown-email-rendering`).

## Problem / Motivation

HTML-heavy emails (newsletters, marketing, notifications) render **messy** in the
TUI today: leftover layout junk, broken tables, and duplicated sections survive
the current cleanup heuristics. The user wants cleaner, more readable rendering
of these emails. markitdown (Microsoft's converter) was suggested because the
user has heard good things about its output and found the pure-Go libraries they
previously tried to be poor quality.

## Current State (verified 2026-06-07)

The codebase **already converts HTML to readable text** — this is a
replace/augment task, not a greenfield one:

- `internal/tui/markdown.go` → `renderMessageContent()` calls
  `render.FormatEmailForTerminal()`.
- `internal/render/format.go` → `FormatEmailForTerminal()`:
  - If `msg.HTML != ""`, calls `renderHTMLToText()` — a **hand-rolled ~280-line
    DOM walker** over `golang.org/x/net/html`.
  - Falls back to `msg.PlainText`.
  - Runs a long pipeline of cleanup heuristics:
    `dedupeNearDuplicateParagraphs`, `collapsePipeNavRuns`,
    `dedupeRepeatedLineBlocks`, glyph sanitization, `WrapTextPreserving`.
  - Emits **plain text with section markers** `[BODY]/[ATTACHMENTS]/[IMAGES]/[LINKS]`
    — **not** Markdown.
- The `M` key (`toggleMarkdown()`) is a **false friend**: it no longer toggles
  markdown — it's been repurposed to toggle **LLM whitespace touch-up**. The
  function name is legacy. Any new toggle must account for this.

## Key Findings from Brainstorm

1. **markitdown's HTML path ≈ `markdownify`.** markitdown subclasses
   `markdownify.MarkdownConverter` for HTML. Its unique value (PDF/docx/pptx/
   images) does not apply to email HTML bodies. So "porting markitdown" really
   means "porting `markdownify`'s HTML rules to Go" (~1k lines, bounded).
2. **markitdown is Python-only.** Using it from Go means shelling out to a CLI;
   every user needs Python + the package installed, plus a subprocess per email
   (install/latency/offline cost for a terminal app).
3. **Raw vs rendered Markdown.** markitdown outputs Markdown *source*
   (`# heading`, `**bold**`, `| tables |`). In a terminal `TextView` that shows
   literally unless rendered (e.g. via `charmbracelet/glamour`). Switching to
   Markdown implicitly bundles a second decision: show raw source or render it.

## Goal & Non-Goals

**Goal:** cleaner, more readable rendering of messy HTML-heavy emails in the TUI.
markitdown is a candidate *means*, not the end.

**Non-goals (for now):** binary attachment conversion (PDF/docx), changing the
plain-text email path, RSVP/calendar handling.

## Decision: PoC First

Rather than guess the converter, run the user's **real messy newsletters**
through each candidate and judge the output. The PoC result picks the path:
shell out to markitdown, port `markdownify` to Go, adopt a pure-Go lib, or
improve the current renderer.

## PoC Plan (approved 2026-06-07)

- **Location:** throwaway `poc/htmlmd/` (part of main module so it can import
  `internal/render`). Deleted + `go mod tidy` before the real feature branch.
- **Inputs:** `poc/htmlmd/samples/*.eml` — user exports 5–10 real messy
  newsletters via Gmail "Show original → Download original". PoC extracts the
  `text/html` MIME part (Go `net/mail` + `mime/multipart`).
- **Converters compared, per sample:**
  1. **Baseline** — current `render.FormatEmailForTerminal` (reused as-is).
  2. **markitdown** — shell out to the Python CLI (installed in a local venv).
  3. **Pure-Go** — `JohannesKaufmann/html-to-markdown` v2 (leading Go lib);
     `lunny/html2md` as fallback.
  4. *(optional)* **markdownify** directly — isolates whether quality comes from
     markitdown or markdownify (sizes a potential Go port).
- **Outputs:** `poc/htmlmd/out/<sample>/{baseline,markitdown,htmltomd}.md`, plus
  a **glamour-rendered** terminal preview of each (judge what it'd actually look
  like in giztui, not raw `**bold**`).
- **Decision gate — the PoC must answer:**
  - Does markitdown clearly beat pure-Go *on these emails*? (justifies a port)
  - Is the gap from `markdownify` or markitdown extras? (sizes the port)
  - Raw Markdown vs glamour-rendered — which reads better in a terminal?

## Open Questions Deferred to Post-PoC (real feature spec) — ALL RESOLVED

These were the open questions during the spike; all are now answered in
**Approved Design — Direction B** below.

- Toggle vs. default renderer; how to coexist with / rename the repurposed `M` key.
- Config keys (per CLAUDE.md conventions), command parity (`:command` for any shortcut).
- Conversion latency & caching of converted output.
- Graceful fallback when conversion fails or email is plain-text only.
- Service-first placement (business logic in `internal/services/`).
- If a Go port is chosen: scope, library to fork/port, maintenance plan.

## PoC Results (2026-06-07) — premise overturned

Ran the bake-off on **8 real promotional newsletters** fetched via giztui's own
Gmail client (`poc/htmlmd/fetch`, `category:promotions`): VICIO/Glovo,
Kinépolis, Movistar Plus (213 KB), etc.

**markitdown LOST — eliminated.** Systematic failures across all 8 samples:
- **Empty layout tables in every sample** (2–10 per email): newsletters wrap
  everything in `<table>` for layout; markitdown faithfully renders them as empty
  Markdown tables (`|  |  |` + `| --- |`), burying content.
- **Charset mojibake** on zero-width-heavy preheaders (`═Å ŌĆī ┬Ā`), NOT fixed by
  prepending `<meta charset="utf-8">` — markitdown's detection genuinely
  mishandles these. The file round-trip (Go string → temp file → CLI) is itself a
  liability the in-process Go path avoids.
- Plus the Python runtime dependency. markitdown's good reputation is for
  **documents** (PDF/docx/pptx), not marketing HTML.

**Pure-Go `html-to-markdown` v2 (+table plugin):** keeps real content and Markdown
structure, runs in-process (no charset bug), BUT noisy — 8–41 tracking
image-links per email (`[![](pixel)](giant-tracking-url)`), zero-width junk,
duplicated CTAs, enormous tracking URLs inline.

**Baseline (giztui's current renderer): most readable of the three today.** Its
cleanup heuristics already strip zero-width chars, drop tracking-pixel images,
collapse/reference links (`[n]` + `[LINKS]`), and dedup repeated blocks — exactly
what both Markdown converters lack. Content (`Alerta de descuentos`, `VICIO ENVÍO
GRATIS`, `McDonald's 2X1`, `Goiko -30%`, footer) reads cleanly. Remaining warts:
flat plain-text (no heading/bold/list structure), and some long tracking URLs
still leak inline as link labels when the anchor had no text.

### Decisive conclusion

**The bottleneck is junk-removal, not conversion.** giztui already owns the best
junk-removal layer. A Markdown converter alone (any of them) is a downgrade
without that cleanup. So the promising directions are:

- **(A) Improve the existing renderer** — keep plain-text output, fix the leaked
  tracking-URL labels and tighten cleanup. Lowest effort, no new deps, no Markdown
  structure.
- **(B) Pure-Go converter + giztui-style cleanup + glamour render** — adopt
  `html-to-markdown` (in-process) for Markdown *structure*, port giztui's cleanup
  onto it (strip zero-width, drop tracking images, reference long URLs, collapse
  layout tables, dedup), render with `charmbracelet/glamour` in the TUI. Gives
  true Markdown look, no Python. Medium effort.
- **(C) Port markdownify to Go** — REJECTED. markitdown/markdownify aren't better
  on this content; the win is cleanup, not conversion rules.

## Approved Design — Direction B (2026-06-07)

**Pure-Go converter + giztui-style cleanup + glamour rendering.** markitdown and
the Go-port (markdownify) paths are CLOSED by the PoC evidence.

### Resolved decisions

| Question | Decision |
| --- | --- |
| Converter | `html-to-markdown` v2 (base + commonmark + table plugins), **in-process** |
| Display model | **Markdown by default** for HTML emails; key toggles to raw |
| `M` key | Reclaimed for **Markdown ↔ raw toggle** (its original meaning) |
| LLM touch-up | Relocated off `M` to a `:touch-up` command (niche; keeps the feature) |
| Go port of markdownify | REJECTED — win is cleanup, not conversion rules |
| New deps | `html-to-markdown` + `glamour` graduate from PoC to real deps |

### Architecture (service-first)

**`internal/render/markdown_render.go`** (new; pure, unit-testable — formatting
already lives in `internal/render/`):

- `RenderEmailMarkdown(msg *gmailwrap.Message, opts MarkdownOptions) (string, error)`
  1. **Convert** `msg.HTML` → Markdown via `html-to-markdown` v2 in-process. The
     in-process path is deliberate: it avoids the temp-file charset round-trip that
     produced markitdown's mojibake.
  2. **Cleanup pipeline** (the high-value 80% — ports `format.go` smarts onto
     Markdown + newsletter-specific rules):
     - Strip zero-width / spacer glyphs (reuse `sanitizeForTerminal`).
     - Drop tracking-pixel image-links (`[![](pixel)](url)` → removed); keep images
       that carry real alt text as `[alt]`.
     - Collapse / remove **empty & layout-only** Markdown tables (all-empty or
       single-cell rows); preserve genuine multi-column data tables.
     - Reference long tracking URLs as `[n]` + a `## Links` section (reuse existing
       link-referencing concept).
     - Dedup repeated CTAs / blocks (reuse `dedupeNearDuplicateParagraphs`).
- `MarkdownToTerminal(markdown string, width int) (string, error)`: glamour render
  → ANSI → `tview.TranslateANSI` (confirmed available in derailed/tview v0.8.5) →
  tview markup for the `text` TextView.

**`DisplayService`** (`internal/services/interfaces.go`, 12th service in
`GetServices()`) extended with render-mode state, mirroring header-visibility:
`ToggleMarkdownRendering() bool`, `SetMarkdownRendering(bool)`,
`IsMarkdownRendering() bool`. Initialized from config default.

**TUI integration** (`internal/tui/markdown.go`):
- `renderMessageContent()` branches: if `IsMarkdownRendering()` && `msg.HTML != ""`
  → markdown pipeline; else existing `FormatEmailForTerminal` (= "raw" view).
- On conversion error or empty output → **fall back** to `FormatEmailForTerminal`
  (never blank). Plain-text-only emails (no HTML) → existing path unchanged.
- Rename `toggleMarkdown()` to actually toggle render mode (it currently toggles
  LLM touch-up). `M` binding at `keys.go:326` / `:1090` — honor the existing
  `isKeyConfigured('M')` guard.

### UX & command parity
- `M` → toggle Markdown ↔ raw (per message). ErrorHandler status: "📄 Markdown
  view" / "📃 Raw view" (async, never inside `QueueUpdateDraw`).
- `:markdown` / `:md` command toggles (CLAUDE.md command-parity mandate); add to
  `executeCommand()` and `generateCommandSuggestion()`.
- `:touch-up` command takes over the old LLM-touch-up toggle.

### Config (new `rendering` block, `config.go:64` pattern — like `LLM`/`Layout`/`Theme`)
```json
"rendering": {
  "markdown_default": true,
  "glamour_theme": "dark",
  "drop_tracking_images": true
}
```

### Caching & performance
- Cache rendered output keyed by `(messageID, mode, width)` — the 213 KB Movistar
  email must not re-convert on every keypress. Invalidate on width change. Reuse
  the existing message-cache pattern where practical.

### Testing
- Unit tests per cleanup rule, using the **PoC `samples/*.html` as fixtures**
  (zero-width strip, tracking-image drop, empty-table collapse, URL referencing).
  Copy the chosen fixtures into the test tree before deleting `poc/`.
- **Real-binary E2E** (hard-won lesson, twice): drive giztui via tmux + a real
  account, open a real newsletter, confirm `M` toggles and output is clean — not
  just unit-green.

### Non-goals
- Binary attachment conversion (PDF/docx/pptx), changing the plain-text path,
  RSVP/calendar handling.

### PoC cleanup
- Delete `poc/` directory and the markitdown venv. KEEP the two Go deps
  (`html-to-markdown`, `glamour`) — they are now real deps. Run `go mod tidy`.

## Next Steps

1. ~~Build PoC harness~~ ✅
2. ~~Fetch real samples via giztui Gmail client~~ ✅ (`poc/htmlmd/fetch`)
3. ~~Run bake-off; analyze~~ ✅ — see PoC Results above.
4. ~~Decide direction~~ ✅ — **B chosen**.
5. ~~Write approved design~~ ✅ — this section.
6. User reviews this spec → then `writing-plans` → `subagent-driven-development`
   on a `feat/` branch off `main`.
