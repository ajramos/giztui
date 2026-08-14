# SDLC Shock Plan

**Status:** Approved
**Date:** 2026-08-14
**Horizon:** 30-day implementation and stabilization window
**Operating model:** Single maintainer, automated enforcement, no artificial
human-approval requirement

## Purpose

Turn GizTUI's documented engineering practices into mandatory, reproducible
controls. The plan prioritizes release integrity and product-wide quality gates
before governance polish or further metrics work.

The 30-day horizon is a rollout and soak window, not an effort estimate. Work
advances by dependency and exit gate. A phase does not start until the preceding
gate is verified.

## Constraints

- Stable releases pause until the release-candidate rehearsal in Phase 3.
- Large features and unrelated refactors stay out of Phases 0-3.
- Pull requests and required checks replace direct integration to `main`.
- Required human approval remains zero while the repository has one maintainer.
- Emergency bypass remains possible but must be followed by a documented review.
- Commercial macOS and Windows signing is out of scope for this window.
- SBOM, provenance, checksum signing, and future signing integration points are
  in scope.
- Local, pull-request, mainline, and release validation share one fail-closed
  contract.

## Phase 0: Containment and Baseline

### Outcomes

- Capture repository, GitHub, workflow, toolchain, and dependency state.
- Reconcile documented checks with checks that actually execute.
- Patch known High/Critical vulnerabilities or record an explicit risk
  acceptance.
- Align the Go toolchain used by modules, CI, and release workflows.
- Define stable names for future required checks.

### Exit gate

- No known High/Critical finding remains without remediation or documented
  acceptance.
- The future required-check contract is explicit.
- No stable release uses the pre-remediation pipeline.

## Phase 1: Repository Control Plane

### Outcomes

- Enable the `main` ruleset.
- Require pull requests while keeping required approvals at zero.
- Block force-push and deletion of `main`.
- Protect creation, update, and deletion of release tags.
- Restrict release inputs to strict SemVer tags.
- Add a `release` environment restricted to version tags.
- Enable dependency alerts and security updates.
- Enable private vulnerability reporting, secret scanning, and push protection.
- Restrict allowed GitHub Actions and prepare full-SHA enforcement.

Remote settings receive one exact preview before application.

### Exit gate

- Changes cannot reach `main` without a pull request and required checks.
- Release tags cannot be moved or deleted accidentally.
- Preventive GitHub security controls are active.
- Emergency bypass is explicit and auditable.

## Phase 2: Canonical Product-Wide Quality Gate

### Outcomes

- Add one fail-closed local command, exposed as `make ci`.
- Fail when a required tool is absent; never report success after a skip.
- Pin linter, mock generator, scanner, and test tool versions.
- Validate formatting, vet, lint, root Go tests, targeted race tests, nested
  desktop Go tests, TypeScript, Vitest, Playwright, frontend build,
  architecture, complexity, vulnerabilities, version consistency, and
  machine-checkable documentation.
- Remove mandatory-path `continue-on-error`, `|| echo`, and equivalent failure
  suppression.
- Add workflow timeouts, concurrency, and stale-run cancellation.
- Upload Playwright traces, screenshots, and logs on failure.
- Establish coverage ratchets from measured baselines rather than arbitrary
  percentages.
- Make architecture checks honest: unsupported checks cannot claim success and
  reliable interface violations must fail.
- Enforce the per-file complexity ratchet in CI.

Playwright becomes required after two consecutive green CI executions.

### Exit gate

- A clean checkout reproduces required CI through one command.
- Root Go, TUI, desktop Go, and frontend behavior are covered.
- Every required check is active on `main`.
- A failing product surface blocks integration.

## Phase 3: Release Engineering and Supply Chain

### Outcomes

- Replace concurrent CLI and desktop publication with one orchestrator and one
  final publisher.
- Keep reusable build jobs per platform with read-only permissions.
- Create a draft release and publish only after all mandatory artifacts pass.
- Remove `if: always()`, checksum suppression, unmatched-file tolerance, and
  `npm install` fallback from mandatory publication paths.
- Add concurrency keyed by tag.
- Check out and validate the exact release tag.
- Require strict agreement among tag, `VERSION`, changelog, compiled version,
  and protected `main` ancestry.
- Verify required checks for the release SHA.
- Disable persisted checkout credentials in build jobs.
- Grant write permissions only to the final publisher.
- Pin Actions to reviewed commit SHAs.
- Pin and verify downloaded packaging tools and scripts.
- Generate central checksums, SPDX or CycloneDX SBOMs, and GitHub artifact
  attestations/SLSA provenance.
- Normalize build time and archive metadata where practical.
- Correct desktop product version and bundle identity, then assert package
  metadata after build.
- Prepare integration points for future Developer ID and Authenticode signing.

### Rehearsal

Publish a prerelease candidate, install or inspect every platform artifact, and
verify checksums, SBOM, provenance, package metadata, and Homebrew distribution
before reopening stable releases.

### Exit gate

- Release publication happens once.
- Build jobs have no repository write credential.
- A failed platform prevents stable publication.
- Every artifact maps to an exact source SHA and workflow run.
- A complete release candidate has passed rehearsal.

## Phase 4: Governance and Documentation Truth

### Outcomes

- Add `SECURITY.md`, `CONTRIBUTING.md`, `CODEOWNERS`, a concise maintainer and
  governance statement, issue forms, `SUPPORT.md`, and an AI-assisted
  contribution policy.
- Add a Code of Conduct after higher-value security and CI controls.
- Correct Go prerequisites, config examples, shortcut references, support
  links, and test claims.
- Generate or verify command and config references from code.
- Add link checking and JSON/YAML example validation to CI.
- Document data flows for Gmail, Ollama, Bedrock, Slack, Obsidian, remote
  images, and telemetry.
- Mark plans as implemented, superseded, or archived and add a lightweight
  decision index.
- Close delivered issues, split partially delivered epics, introduce workflow
  status labels, and use a milestone for the next release.
- Declare GitHub Issues the canonical live backlog.

### Exit gate

- Principal documentation has no broken links or invalid examples.
- README claims match enabled channels and shipped behavior.
- Open issues represent pending work.
- Security reports have a private, documented route.

## Phase 5: Reproducible Metrics and Debt Reduction

### Outcomes

- Require an explicit analysis ref in `baseline.py`.
- Reject shallow repositories and record range, ref, and history completeness.
- Fail on Git, extraction, and Lizard errors.
- Remove hard-coded repository-age metadata.
- Separate measured observations from interpretations such as rework and debt.
- Improve semantic-revert detection.
- Prove deterministic output with repeated runs over the same ref.
- Regenerate the baseline from complete history.
- Retire human-versus-AI line attribution as a personal performance measure.
- Keep the per-file complexity ratchet active.
- Define bounded reduction targets for `messages.go`, `bindKeys`, and
  `runCommand` without launching a broad refactor.

### Operational metrics

- Pull requests green on first required-check run.
- Failures by root, TUI, desktop, security, and release surface.
- Open vulnerabilities and remediation time.
- Failed or partial releases.
- Flaky tests identified and removed.
- Escaped regressions per release.
- Documentation failures found automatically.

Commit count, line count, and generated-code volume are not productivity
metrics.

### Exit gate

- The SDLC baseline reproduces from an exact ref and complete history.
- Complexity regressions are blocked automatically.
- Technical debt has a small, ordered backlog with measurable targets.
- Every retained metric supports a concrete decision.

## Execution Batches

1. Urgent dependency and toolchain remediation.
2. Fail-closed `make ci` contract.
3. Complete desktop CI.
4. Architecture, complexity, and security gates.
5. Remote repository protection.
6. Single release orchestrator.
7. SBOM, provenance, and reproducibility.
8. Governance and security policy.
9. Automated documentation correction and backlog cleanup.
10. Baseline regeneration and debt roadmap.

Each batch is independently reviewable and verified before the next starts.
Infrastructure work does not absorb unrelated feature or refactor changes.

## Definition of Done

- A change cannot reach `main` without every required product check.
- A desktop regression cannot hide behind green root Go CI.
- A release cannot publish partially.
- Every release artifact has checksums, an SBOM, and provenance.
- No known High/Critical vulnerability lacks a documented decision.
- Principal documentation matches actual behavior and configuration.
- The live backlog represents pending work.
- SDLC metrics reproduce from explicit source history.
- The process works for one maintainer without pretending to provide
  independent human review.

## Tracking

This document is the stable implementation contract. Mutable progress, blockers,
evidence, and the next action belong in the linked GitHub tracking issue rather
than in this file.

Tracking issue: [#87](https://github.com/ajramos/giztui/issues/87)
