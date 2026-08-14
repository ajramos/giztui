# Decision Index

Lightweight index of design decisions and implementation plans for GizTUI.
Each row names the source document and its status. Statuses:

- **Implemented** — shipped in a published release.
- **In progress** — tracked in the live backlog; see the linked issue.
- **Superseded** — replaced by a later plan.
- **Archived** — historical; no longer acted on.

Design specs live in `docs/superpowers/specs/`, implementation plans in
`docs/superpowers/plans/`. The tracking contract for the SDLC shock plan is
issue #87; the index below is maintained manually and is not generated.

## Plans

| Plan | Status | Notes |
|------|--------|-------|
| 2026-06-06-prompt-configurator | Implemented | Prompt configuration UI. |
| 2026-06-07-inbox-action-plan | Implemented | Inbox action-plan panel. |
| 2026-06-07-markdown-email-rendering | Implemented | HTML-to-markdown rendering. |
| 2026-06-08-auto-refresh-inbox | Implemented | Inbox auto-refresh. |
| 2026-06-08-prompt-preview | Implemented | Prompt preview. |
| 2026-06-09-action-plan-rework | Implemented | Action-plan rework. |
| 2026-06-09-inplace-panels-and-action-plan-fixes | Implemented | In-place panels. |
| 2026-06-10-action-plan-bulk-category-move | Implemented | Bulk category move. |
| 2026-06-10-v1.6.1-prompt-and-render-fixes | Implemented | Prompt/render fixes. |
| 2026-06-11-analyzer-body-context | Implemented | Analyzer body context. |
| 2026-06-11-analyzer-prompt-viewer | Implemented | Analyzer prompt viewer. |
| 2026-06-11-analyzer-rules-inplace | Implemented | Analyzer rules UI. |
| 2026-06-12-bulk-operation-progress | Implemented | Bulk operation progress. |
| 2026-06-13-analyzer-existing-labels | Implemented | Analyzer existing-label reuse. |
| 2026-06-13-analyzer-interests | Implemented | Analyzer interests. |
| 2026-06-13-analyzer-summarize-action | Implemented | Summarize action. |
| 2026-06-13-autorefresh-slack-notify | Implemented | Auto-refresh Slack notify. |
| 2026-06-13-config-self-migration | Implemented | Config self-migration. |
| 2026-06-13-tts-read-aloud | Implemented | TTS read-aloud. |
| 2026-06-17-slack-autorefresh-summary | Implemented | Slack auto-refresh summary. |
| 2026-06-18-vim-state-extraction | Implemented | Vim state extraction refactor. |
| 2026-06-20-action-plan-enter-load | Implemented | Action-plan Enter-to-load. |
| 2026-06-20-analyzer-faithful-labels | Implemented | Faithful analyzer labels. |
| 2026-06-21-command-state-extraction | Implemented | Command state extraction. |
| 2026-06-21-overlay-backup-extraction | Implemented | Overlay/backup extraction. |
| 2026-06-21-search-state-extraction | Implemented | Search state extraction. |
| 2026-06-22-command-tab-completion | Implemented | Command tab-completion. |
| 2026-06-24-bulk-state-extraction | Implemented | Bulk state extraction. |
| 2026-06-28-command-typo-suggestion | Implemented | Command typo suggestion. |
| 2026-06-29-per-command-help | Implemented | Per-command help. |
| 2026-07-04-action-plan-confirm-all | Implemented | Action-plan confirm-all. |
| 2026-07-04-deterministic-rules | Implemented | Deterministic rules service. |
| 2026-08-14-sdlc-shock-plan | In progress | Tracking issue #87; Phases 0-3 complete, Phase 4 active. |

## Specs

| Spec | Status |
|------|--------|
| 2026-06-06-inbox-analysis-prompt-configurator-design | Implemented |
| 2026-06-07-html-to-markdown-rendering-design | Implemented |
| 2026-06-08-action-plan-rework-design | Implemented |
| 2026-06-08-auto-refresh-inbox-design | Implemented |
| 2026-06-08-prompt-preview-design | Implemented |
| 2026-06-09-inplace-panels-and-action-plan-fixes-design | Implemented |
| 2026-06-10-action-plan-bulk-category-move-design | Implemented |
| 2026-06-11-analyzer-body-context-design | Implemented |
| 2026-06-11-analyzer-prompt-viewer-design | Implemented |
| 2026-06-11-analyzer-rules-inplace-design | Implemented |
| 2026-06-12-bulk-operation-progress-design | Implemented |
| 2026-06-13-analyzer-existing-labels-design | Implemented |
| 2026-06-13-analyzer-interests-design | Implemented |
| 2026-06-13-analyzer-summarize-action-design | Implemented |
| 2026-06-13-autorefresh-slack-notify-design | Implemented |
| 2026-06-13-config-self-migration-design | Implemented |
| 2026-06-13-tts-read-aloud-design | Implemented |
| 2026-06-17-slack-autorefresh-summary-design | Implemented |
| 2026-06-18-vim-state-extraction-design | Implemented |
| 2026-06-20-action-plan-enter-load-design | Implemented |
| 2026-06-20-analyzer-faithful-labels-design | Implemented |
| 2026-06-21-command-state-extraction-design | Implemented |
| 2026-06-21-overlay-backup-extraction-design | Implemented |
| 2026-06-21-search-state-extraction-design | Implemented |
| 2026-06-22-command-tab-completion-design | Implemented |
| 2026-06-24-bulk-state-extraction-design | Implemented |
| 2026-06-28-command-typo-suggestion-design | Implemented |
| 2026-06-29-per-command-help-design | Implemented |
| 2026-07-04-action-plan-confirm-all-design | Implemented |
| 2026-07-04-deterministic-rules-design | Implemented |
| 2026-08-14-per-account-llm-design | Implemented | Phase 1 shipped; subscription-auth providers deferred to backlog. |
