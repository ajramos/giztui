# Contributing to GizTUI

Thanks for considering a contribution. GizTUI is a single-maintainer project
that relies on automated enforcement to stay safe to contribute to. This page
explains how changes reach `main` and what every change must satisfy.

The live backlog lives in GitHub Issues. Before starting work, check for an
existing issue or open one so the change is tracked.

## Development Setup

Follow `docs/DEVELOPMENT_SETUP.md` to install the pinned toolchain. The
canonical local gate is:

```bash
make ci-tools
make ci
```

`make pre-commit-check` is an alias for `make ci`. The optional pre-commit
hooks give fast feedback but do not replace the full gate.

## How Changes Reach main

`main` is protected. Nothing is pushed directly to it.

1. Create a feature branch from `main`.
2. Make the change. Keep it focused; do not mix unrelated refactors into a
   fix.
3. Run `make ci` until it passes.
4. Open a pull request to `main` and wait for the `required` check.
5. A maintainer merges the PR. The `required` check must be green.

Definition of done (see `docs/superpowers/plans/2026-08-14-sdlc-shock-plan.md`):

- A change cannot reach `main` without every required product check.
- A desktop regression cannot hide behind green root Go CI.
- A release cannot publish partially.
- Principal documentation matches actual behavior and configuration.

## Architecture Rules

The non-negotiable patterns are documented in `CLAUDE.md`. The short version:

- **Service-first.** Business logic lives in `internal/services/`; UI
  components only present and accept input. Never put Gmail/LLM calls in UI.
- **ErrorHandler only.** User feedback goes through `app.GetErrorHandler()`.
  No `fmt.Printf`/`log.Printf` for user messages.
- **Thread-safe state.** Use accessor methods (`GetCurrentView()`,
  `SetCurrentMessageID()`, `GetServices()`, ...). Never touch `App` fields
  directly.
- **Picker state.** Use the `ActivePicker` enum, never shared booleans.
- **Theming.** Use `app.GetComponentColors(component)` for all UI colors.
- **Command parity.** Every keyboard shortcut must have an equivalent
  `:command` with a short alias, wired into `executeCommand()` and the
  command suggestions.

## AI-Assisted Contributions

GizTUI's SDLC gates (`make ci`, the `required` check, the release workflow)
are the acceptance criterion for every change, whether written by a human or
with AI assistance. An AI-assisted contribution is welcome when:

- The author understands and can explain the change.
- The change passes `make ci` with no suppression of failures.
- Commit messages are clean and do not add tooling or model signatures
  (`Co-authored-by` lines, "Generated with ..." trailers). This repository's
  git history stays tool-agnostic.

Contributions that cannot pass the gate are not accepted regardless of how
they were produced.

## Reporting Bugs and Vulnerabilities

- For suspected security vulnerabilities, use the private route in
  `SECURITY.md` — never open a public issue.
- For everything else, use the bug report issue form so the environment,
  reproduction steps, and expected behavior are captured.

## Maintainer and Governance

- Maintainer: Ángel J. Ramos (`@ajramos`).
- There is currently one maintainer, so required human approval on PRs is
  zero by design. Automated gates substitute for the review capacity of a
  larger team.
- The protected `main` branch and `refs/tags/v*` rulesets can only be
  bypassed by the maintainer, and every bypass must be followed by a
  documented review in the tracking issue.
- Product direction is driven by the open backlog and the issue tracker, not
  by unrecorded decisions. Significant design choices are captured under
  `docs/decisions/`.

## Getting Help

Open an issue with the `question` label for usage or contribution questions.
There is no separate discussion forum; GitHub Issues is the canonical
backlog and support channel.
