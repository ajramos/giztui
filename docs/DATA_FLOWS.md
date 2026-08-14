# Data Flows

How GizTUI moves data between your machine, Gmail, and the optional local AI
and integration services. Everything is local-first: there is no GizTUI backend
and no telemetry leaves your machine.

## Gmail

- OAuth2 credentials live in `~/.config/giztui/credentials.json`; access/refresh
  tokens in the same directory. Tokens are refreshed in place.
- The Gmail API is called only through `internal/gmail` (wrapped by
  `internal/services`). Reads: messages, threads, labels, drafts, filters,
  attachments (via `messages.attachments.get`). Writes: archive, trash, star,
  labels, drafts, send, and filter/rules changes.
- HTML email is rendered locally. Remote images are **not** fetched by the
  desktop client; the TUI renders text/markdown. Embedded attachment data stays
  on disk until you export or forward it.
- Search queries are translated to Gmail operators and executed by the API;
  results are cached in the local database.

## Ollama (default LLM)

- `internal/llm` posts prompts to `http://localhost:11434/api/generate`
  (configurable via `llm.endpoint`). The request and response never leave your
  machine; Ollama runs locally.
- Streaming responses are read incrementally and rendered in the AI summary and
  prompt panels. No prompt or message content is logged or persisted beyond the
  local prompt-usage counters shown by `:prompt stats`.

## Amazon Bedrock

- When `llm.provider = "bedrock"`, prompts go to the configured Bedrock model
  through the AWS SDK. Credentials come from the standard AWS credential chain
  (env, `~/.aws/`), never from GizTUI config.

## Slack

- `internal/services/slack` posts forwarded messages (subject + body) to the
  selected Slack channel via the Slack Web API. The token is configured in
  `~/.config/giztui/config.json` (`slack.token`). Message content is only sent
  when you explicitly run the forward action.

## Obsidian

- The Obsidian integration writes ingested email content into your vault path
  (`obsidian.vault`) as Markdown files. The vault lives on your machine; nothing
  is uploaded.

## Remote Images

- The desktop client proxies remote images through the Go backend
  (`FetchImage`) and returns them as data URIs so the WKWebView can render them
  without fetching from arbitrary hosts. The TUI does not load remote images.
- Other HTML subresources are not fetched.

## Telemetry (local, opt-in)

- `telemetry.enabled` defaults to `false`. When enabled, only bounded local
  events are recorded — command names, shortcut keys, and action outcomes with
  timing (no message content, no arguments, no email text).
- Events are stored in the local database under `~/.config/giztui/cache/`,
  pruned by `telemetry.retention_days` (default 90). View them with `:stats`;
  `:stats reset` wipes them.
- Nothing is ever uploaded; there is no remote telemetry endpoint.

## Privacy Summary

| Data | Leaves the machine? | Where it goes |
|------|--------------------|---------------|
| Email content (triage, labels, search) | No | Gmail API (your account) |
| AI summaries/prompts | Only if you configure Bedrock/remote LLM | Ollama localhost, or your Bedrock region |
| Slack forwards | Only when you forward | Your Slack workspace |
| Obsidian ingest | No | Your local vault |
| Telemetry | Never | Local database only |

See `docs/CONFIGURATION.md` for the corresponding settings and
`docs/COMMANDS.md` for the generated command reference.
