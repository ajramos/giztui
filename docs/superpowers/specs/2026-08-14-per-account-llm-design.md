# Per-Account LLM Configuration (+ subscription-auth providers) — Design Spec

**Date:** 2026-08-14
**Status:** DRAFT — for user review (co-authored; open questions marked ❓)
**Related:** replaces the ephemeral plan `lexical-meandering-lake`; builds on the
existing single-global LLM wiring.

## Summary

Give **each Gmail account its own LLM engine**. The professional account can run a
local **Ollama** model; the personal account can reuse a **ChatGPT Plus/Pro
subscription** (OAuth, no per-token billing) — or any other engine. Two accounts
may point at the same engine, but the user declares and configures it **per
account** in `config.json`. The active account's effective engine is **shown**
(read-only) in the TUI `:config` output and the desktop ConfigModal; there is no
in-app engine editor in this iteration.

Engines selectable per account (the "palette"):

| Engine | Status | Auth model |
|---|---|---|
| Ollama (local) | exists | none (local endpoint) |
| Bedrock | exists | AWS creds/region |
| OpenAI API-key | **new** | `api_key` (metered `api.openai.com`) |
| ChatGPT subscription | **new** | **OAuth → Codex backend** (reuse Plus/Pro) |
| Vertex / Gemini | **new** | Google service-account / ADC |

A **generic "subscription-auth provider" abstraction** is designed first (OAuth
login + token store/refresh + backend endpoint) so a Claude Pro/Max subscription
can plug in later with the same shape.

## Goals / Non-goals

**Goals**
- Per-account engine selection with a clean "account override else global" model.
- A provider abstraction that covers stateless (key/local) *and* subscription-auth engines.
- Reuse a ChatGPT subscription for GizTUI's AI features (summaries, replies, labels, prompts, chat).
- Zero disruption for existing users (automatic config backfill; global LLM unchanged when no per-account block).

**Non-goals (this iteration)**
- In-app editor for LLM settings (config is hand-edited, consistent with all LLM config today).
- Multi-account rotation / load-balancing across subscriptions.
- Claude subscription provider (design for it; implement later).

## Requirements

1. `config.json` account entries may carry an `llm` block; absent → inherit the global `llm`.
2. Switching account re-resolves and rebuilds the provider so AI immediately uses the new engine.
3. New engines: OpenAI API-key, ChatGPT-subscription, Vertex — each selectable per account.
4. Secrets (API keys, OAuth tokens) never live in `config.json` logs and never print.
5. `:config` (TUI) and ConfigModal (desktop) show the *active account's* effective engine.
6. Subscription login is an explicit, opt-in action (`:llm login <provider>`).

## Data model

Add to `AccountConfig` (`internal/config/config.go:127`):

```go
LLM *LLMConfig `json:"llm,omitempty"`  // nil → inherit global config.LLM
```

Pointer (not value) so "absent" is distinguishable from "zero" (`Enabled:false`,
`Provider:""`). Follows the existing keyed-override precedent `TTSConfig.Voices/Models`.

**Resolver** — single source of truth in `internal/config/config.go`:

```go
// EffectiveLLM returns the LLM for accountID: the account's override (core fields
// verbatim; cosmetic fields — templates/timeout/stream/cache — backfilled from
// DefaultLLMConfig) if present, else the global LLM. "" / unknown ID → global.
func (c *Config) EffectiveLLM(accountID string) LLMConfig
```

Key by account **ID** (stable switch key), not email. Core fields
(Provider/Model/Endpoint/Region/APIKey/Enabled) come verbatim so a personal `openai`
block never inherits the global ollama `Endpoint`.

**Migration:** none needed. `deepMergeMissing` (`internal/config/migrate.go:47`)
skips nil `omitempty` fields, so existing `config.json` files are untouched and
`:config migrate` reports nothing (per the config-self-migration DoD).

**New LLM fields** likely needed on `LLMConfig` (backfilled by default to preserve
back-compat): none required for API-key (`APIKey`/`Endpoint` exist); ChatGPT-sub and
Vertex may need small additions (e.g. `Project` for Vertex). ❓ Confirm we keep one
flat `LLMConfig` for all providers vs. provider-specific sub-structs. *Recommendation:*
keep flat + document which fields each provider reads.

## Provider abstraction

Today `internal/llm` has `Provider{Name(); Generate()}` + optional `ParamProvider`,
`StreamProvider`, built by `NewProviderFromConfig(provider, endpoint, model, timeout, apiKey)`
(`factory.go`). Extend, don't rewrite:

- Keep `Provider`/`ParamProvider`/`StreamProvider` as-is (all engines implement Generate;
  streaming is a runtime type-assert in `ai_service.go:93-171`, so new engines get it free
  by implementing `GenerateStream`).
- Add a **subscription-auth capability** for engines that need a login + refreshing token:

```go
// AuthProvider is implemented by engines backed by an OAuth subscription rather
// than a static key. The App calls Login on ":llm login", and the client calls
// EnsureValidToken() before each request (refreshing when near expiry).
type AuthProvider interface {
    Provider
    Login(ctx context.Context) error       // interactive: opens browser / prints URL, waits for callback
    Logout() error
    AuthStatus() (loggedIn bool, detail string)
}
```

- **Token store** — new `internal/llm/authstore` (or `pkg/auth` sibling): persists tokens to
  `~/.config/giztui/llm-auth.json`, file mode `0600`, **keyed by provider, machine-global**
  (DECIDED 2026-08-14). One ChatGPT subscription is reused across every Gmail account that
  selects `chatgpt`: the per-account config only selects *which* engine, the credential is
  shared machine-wide, so there is a single `:llm login chatgpt` per machine.
- **Factory** (`factory.go`): add `case "openai"`, `case "chatgpt"`, `case "vertex"`, and
  **change the `default` from silent Ollama fallback to an error** so a per-account typo
  disables AI with a clear log rather than silently running the wrong engine. Both callers
  (`cmd/giztui/main.go:414`, `pkg/desktop/session.go:435`) already log + degrade (nil
  provider → AI off), so this is safe.

## Engines

### OpenAI API-key — `internal/llm/openai.go` (new)
Standard `POST {endpoint}/chat/completions`, `Authorization: Bearer {api_key}`, single
user message (matches how ollama's `Generate(prompt)` is fed). SSE streaming → `StreamProvider`;
temperature/max_tokens → `ParamProvider`. Default endpoint `https://api.openai.com/v1`.
Smallest new provider; ships first to exercise the palette end-to-end.

### ChatGPT subscription — `internal/llm/chatgpt.go` + auth store (new)
Reuses a Plus/Pro subscription via the **same OAuth method OpenAI's Codex CLI uses**
(as `opencode-openai-codex-auth` / `open-hax/codex` do). Shape:

1. **Login** (`:llm login chatgpt`): OAuth 2.0 **PKCE** against `auth.openai.com`
   (authorize → loopback `http://localhost:<port>/callback` → code → token exchange).
   Persists `access_token` + `refresh_token` (+ expiry, + the account/plan id from the
   `id_token` claims) to the auth store.
2. **Request**: `POST` to the ChatGPT **Codex backend** (Responses-style endpoint) with
   `Authorization: Bearer <access_token>` and the account-id / beta headers; parse the SSE
   stream into tokens → `StreamProvider`.
3. **Refresh**: `EnsureValidToken()` refreshes via the refresh_token before expiry.

> ❗ **To be pinned during implementation** (not guessed here): the exact authorize/token
> endpoints, the public `client_id`, the callback port, the Codex backend URL, the required
> headers (`chatgpt-account-id`, `OpenAI-Beta`, `originator`, …), the request/response body,
> and the available model ids. These will be lifted from the authoritative reference
> (OpenAI Codex CLI source and/or the opencode plugin's `lib/` auth code) and captured in
> this spec before the client is written. This is the single biggest unknown.

> ⚠️ **Experimental / ToS:** this is an unofficial reuse of the subscription; endpoints and
> auth may change or break, and it is intended for **personal** use. It is strictly opt-in
> (the user must run `:llm login chatgpt`) and clearly documented as best-effort.

### Vertex / Gemini — `internal/llm/vertex.go` (new)
Google auth via service-account JSON or ADC; `project` + `region` + gemini model; the
`generateContent` body differs from the single-prompt shape (adapt in the client). Heaviest
external auth after ChatGPT; ships last.

## Per-account resolution & wiring

**Concurrency:** `App.LLM` (`internal/tui/app.go:62`) is read from goroutines
(`ai.go:461`, `markdown.go:176/186`, `messages.go:979`) and will now be **reassigned on
switch**. Add `GetLLM()/SetLLM()` under the existing `a.mu` and route reads through them.

**TUI account switch** (`internal/tui/accounts.go:285` `switchToAccount`): after the client/DB
swap and **before** `reinitializeClientDependentServices()` (line 363), resolve
`EffectiveLLM(newAccount.ID)`, build the provider, `SetLLM(prov)` (nil on error → AI off).
The reinit rebuilds `aiService` from `GetLLM()` (app.go:1079-1080). **Fix:** null `a.aiService`
when the provider is nil so switching to a non-AI account disables AI instead of leaving a
stale service.

**TUI startup** (`cmd/giztui/main.go:388-417`): build from `EffectiveLLM(activeAccountID)`
(from `accountService`, already built before `NewApp`), not global `cfg.LLM`.

**Desktop** (`pkg/desktop/session.go`): thread the active account ID into
`buildAPI`→`buildAIService` (170/418) and resolve `cfg.EffectiveLLM(accountID)`; set
`currentAccountID` before `buildAPI` in `SwitchAccount` (410-411). The stack already rebuilds
per switch.

**Centralize** the `LLMConfig → provider` unpacking (region/endpoint mapping currently
duplicated in main.go and session.go) into one `config` helper reused by all sites.

## Show active engine (read-only)
- **TUI `:config`** (`internal/tui/commands.go:2327`, currently only `migrate`): the
  no-subcommand path shows `AI: <provider> · <model> (account: <id>)` or `AI: disabled`,
  plus login state for subscription engines (`logged in` / `run :llm login chatgpt`).
- **Desktop ConfigModal**: only the population changes — `desktop/app_mail.go:251-252` sets
  `ConfigInfo.LLMProvider/LLMModel` from `EffectiveLLM(currentAccountID)`. DTO/frontend/mock
  unchanged.

## Login UX
New `:llm` command family (TUI) + desktop equivalent:
- `:llm` — show active engine + auth status.
- `:llm login <provider>` — run the OAuth flow (prints/open the URL, waits for callback).
- `:llm logout <provider>` — drop stored tokens.
❓ Command name: `:llm` vs `:ai` vs `:model`. *Recommendation:* `:llm`.

## Security
- Tokens/keys in `~/.config/giztui/llm-auth.json`, `0600`, never in `config.json`.
- Never log secrets — log provider·model only (audit `BuildProvider`, error paths, `:config`).
- Optional env fallbacks (`OPENAI_API_KEY`, `AWS_REGION`) mirroring today's bedrock pattern.

## Risks / open questions
- ❗ ChatGPT-subscription OAuth specifics unverified — pin from reference before coding (above).
- ⚠️ Experimental/ToS; opt-in; document as best-effort.
- Bedrock has no streaming (falls back to `Generate`) — document.
- Removing the silent Ollama fallback is a behavior change — verified callers degrade gracefully.
- Legacy single-account config (no `accounts` array) → `EffectiveLLM("")` = global; unchanged.
- ❓ Token scope: per-provider machine-global (recommended) vs per-account.
- ❓ One flat `LLMConfig` for all engines vs per-provider sub-structs (recommend flat).

## Phased plan (order DECIDED 2026-08-14)
1. ✅ **Plumbing (DONE)** — `AccountConfig.LLM`, `EffectiveLLM`, `GetLLM/SetLLM`, switch/startup/
   desktop wiring, factory error-on-unknown, `:config`/ConfigModal display. Ships with existing
   engines (ollama+bedrock) already per-account-selectable. Fully testable alone.
2. ✅ **ChatGPT subscription (DONE)** (the real goal, first) — auth store (`internal/llm/authstore`,
   0600, machine-global) + OAuth 2.0 PKCE login + Codex Responses-API SSE client
   (`internal/llm/chatgpt.go`) + factory `case "chatgpt"` + `:llm login|logout|status` (TUI).
   Endpoints/client-id mirror the Codex CLI (public values), adjustable via overridable fields;
   httptest-backed unit tests cover PKCE, token exchange/refresh, `validToken`, and SSE parse.
   ⚠️ Live path is unverifiable without a real subscription — best-effort, opt-in.
3. **OpenAI API-key** provider — standard chat/completions + SSE.
4. **Vertex/Gemini** — Google auth + client.

## Verification
- **Unit:** `EffectiveLLM` (override/inherit/unknown/no-dilution); each provider client against
  `httptest` (chat + SSE + token refresh); factory errors on unknown provider; auth-store
  round-trip + 0600.
- **Manual:** two accounts with different `llm` blocks → `:config` reflects the active engine;
  switch account and confirm the model used changes; `:llm login chatgpt` completes OAuth and a
  summary runs off the subscription; typo provider → AI disabled with a clear log.
- **Gates:** `make pre-commit-check`; desktop `npx tsc --noEmit && npm run build && npm test`;
  `go build ./... && go test ./pkg/desktop/`.
