Title: Issue‑First Workflow (Gh Issues) for Gmail TUI

Summary
- Purpose: Ensure every new request (feature/bug/chore) becomes a well‑formed GitHub issue before any implementation begins.
- Source of truth: GitHub Issues; work only starts after the issue is created and accepted.
- Language: Issue titles/bodies in English (aligns with project rules). Chat and coordination may be in Spanish.

Philosophy
- Propose first, then execute: the assistant drafts the issue and seeks approval for larger/critical work before coding.
- Small, low‑risk items can be created immediately and announced, but still must exist as issues before work.

Scope
- Applies to all new tasks: features, bugs, enhancements, docs, tests, infra, chores, and epics.
- Covers taxonomy, classification, acceptance criteria, milestones/epics, and automation via `gh`.

Taxonomy (Labels)
- type: `type:feature`, `type:bug`, `type:enhancement`, `type:docs`, `type:test`, `type:infra`, `type:chore`, `type:epic`
- area: `area:email`, `area:compose`, `area:attachments`, `area:labels`, `area:calendar`, `area:ai`, `area:ux`, `area:theme`, `area:help`, `area:testing`, `area:infra`, `area:config`, `area:build`, `area:docs`
- priority: `P0` (critical), `P1` (high), `P2` (medium, default), `P3` (low)
- size: `size:S` (≤2h), `size:M` (≤1d), `size:L` (≤3d), `size:XL` (>3d)
- Optional status: `status:ready`, `status:in-progress`, `status:blocked` (used to reflect state; not required to create)

Roadmap (Milestones)
- v0.1 Core MCP Parity (Email/Labels)
- v0.2 UX & Commands
- v0.3 AI & Calendar
- v0.4 Theme & Help
- v0.5 Testing & Infra
- v0.6 Release & Docs

Definitions
- Definition of Ready (DoR):
  - Clear title (Area: Concise goal), English body
  - Summary and rationale
  - ≥3 Acceptance Criteria (observable/verifyable)
  - Type, Area, Priority, Size
  - Milestone and Epic linkage (create epic if missing)
  - Dependencies/risks noted
- Definition of Done (DoD):
  - Code merged and linked (Fixes/Closes #N)
  - Tests updated/added, CI green
  - UX verified (short QA notes)
  - Docs/config updated (README/Help as relevant)
  - Issue closed

Workflow
1) Intake & Clarify
   - If ambiguous, ask up to 3 concise questions; otherwise proceed.
   - Capture: Title, Summary, Acceptance Criteria, Non‑goals, Risks.
2) Shape & Classify
   - Assign Type/Area; set Priority (P2 default; P1 for enablers or user‑visible blockers; P0 for outages); estimate Size.
3) Linkage
   - Choose Milestone (per roadmap). Create/link Epic (`type:epic`) when grouping is needed.
4) Draft Issue (English)
   - Use the shared template below. Include Implementation Hints and QA.
5) Review Gate
   - If Size ∈ {L, XL} or Priority ∈ {P0, P1} → present for approval before coding.
   - If Size = S and Priority ∈ {P2, P3} → create immediately, then announce.
6) Create via GitHub CLI (non‑interactive)
   - Create missing labels/milestones/epics idempotently.
   - Create the issue; record the number; add to epic checklist.
7) Start Work
   - Only after issue exists and passes DoR (label `status:ready` or explicit approval).
8) PR & Close
   - Link PR to issue ("Fixes #N"); satisfy DoD; close issue on merge.

Issue Template (Body)
```
## Summary
One‑paragraph explanation of the goal and rationale.

## Acceptance Criteria
- [ ] Criterion 1 (observable)
- [ ] Criterion 2 (observable)
- [ ] Criterion 3 (observable)

## Notes
Context, risks, decisions, dependencies.

## Implementation Hints
Files/modules likely impacted, patterns to follow, experiments.

## QA
Manual steps and expected results. Edge cases.
```

Bug Template Add‑On
```
## Bug Details
- Expected:
- Actual:
- Steps to Reproduce:
- Environment (OS, terminal, config):
- Severity (P0–P3):
```

Automation (CLI Snippets)
- Labels (idempotent):
```
gh label create "type:feature" --color FFD700 --description "New feature" || true
gh label create "type:bug" --color D93F0B --description "Bug" || true
gh label create "type:epic" --color 5319E7 --description "Epic" || true
gh label create "area:email" --color D4C5F9 --description "Email management" || true
gh label create "P1" --color D93F0B --description "High" || true
gh label create "size:M" --color EDEDED --description "Medium" || true
```
- Milestones (idempotent):
```
gh api repos/:owner/:repo/milestones -f title='v0.1 Core MCP Parity (Email/Labels)' -f state='open' || true
```
- Create Epic:
```
gh issue create \
  --title "[EPIC] MCP Parity: Email Management" \
  --body "This epic tracks Email features and dependencies." \
  --label "type:epic,area:email,P1,size:XL" \
  --milestone "v0.1 Core MCP Parity (Email/Labels)"
```
- Create Issue (feature):
```
gh issue create \
  --title "Email: Query emails (Gmail search syntax)" \
  --body "$(cat <<'EOF'
## Summary
Search and query emails with Gmail search syntax.

## Acceptance Criteria
- [ ] Supports advanced Gmail operators
- [ ] Paginates results with count and preview
- [ ] Handles network errors and API limits gracefully

## QA
- [ ] Queries by sender, date and text return expected results
EOF
)" \
  --label "type:feature,area:email,P1,size:L" \
  --milestone "v0.1 Core MCP Parity (Email/Labels)"
```

Priority & Size Heuristics
- Promote to P1 when the task unlocks other work, addresses user‑visible correctness, or removes a major UX gap.
- Size S: one file or straightforward flow; L/XL: cross‑module, async, or requires new UI surfaces.

Operating Rules
- Do not start coding without an issue number.
- Keep issues in English; code, comments, and docs in English per project rules; commit messages reference the issue.
- All CLI invocations must be non‑interactive and idempotent where possible.

Out of Scope
- Managing code changes here; this rule only governs issue creation and workflow gates.


