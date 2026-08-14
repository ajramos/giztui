# Testing Guide

This guide documents the tests and gates that exist in the current repository.
The Makefile is the executable source of truth.

## Canonical Local Gate

Install the pinned tools once, then run the complete local product gate:

```bash
make ci-tools
make ci
```

`make pre-commit-check` is an alias for `make ci`. The local gate runs:

- Go formatting, vet, golangci-lint, actionlint, and release-script fixtures.
- Root module verification, build, tests, aggregate coverage, and targeted race
  tests for services, TUI, and the desktop adapter.
- Root and desktop `govulncheck`, plus npm audit.
- Architecture, per-file complexity, and coverage ratchets.
- Locked frontend install, 81 Vitest tests, TypeScript/Vite production build,
  nested desktop Go verification/test/build/vet, and 67 Playwright scenarios.

GitHub CI adds the Linux/macOS root matrix, Trivy SARIF scanning, dependency
review, Codecov upload, and the protected `required` aggregator. Local `make ci`
is the canonical product gate, not a byte-for-byte reproduction of hosted CI.

## Focused Commands

```bash
make test                    # Root scoped test suite
make test-unit               # internal/services with race detector
make test-tui                # TUI and test helpers with race detector
make test-integration        # Tests under test/
make test-coverage           # Root coverage HTML report
make test-race               # Full root race run
make ci-desktop              # Frontend, desktop Go, and Playwright gate
make ci-security             # Reachable Go vulnerabilities and npm audit
make ci-architecture         # Architecture and complexity ratchets
```

Run a package or named test directly while developing:

```bash
go test ./internal/services
go test ./internal/tui -run TestName
go test ./pkg/desktop -run TestName
cd desktop && go test ./...
cd desktop/frontend && npm test
cd desktop/frontend && npx playwright test e2e/inbox.spec.ts
```

## Test Locations

- `internal/**/*_test.go`: package-level Go tests.
- `pkg/desktop/**/*_test.go`: front-end-independent desktop adapter tests.
- `test/helpers`: TUI harnesses, snapshots, and shared test utilities.
- `test`: integration-style tests using local mocks and fixtures.
- `desktop/**/*_test.go`: nested desktop module tests.
- `desktop/frontend/src/*.test.ts`: Vitest tests for pure frontend logic.
- `desktop/frontend/e2e/*.spec.ts`: Playwright browser tests against the mock
  backend.

No test suite contacts live Gmail, LLM, Slack, or Obsidian services in canonical
CI. External dependencies must be mocked or represented by deterministic
fixtures.

## Visual Snapshots

Snapshot mismatches and missing baselines fail by default. Review the rendered
change before accepting it, then run:

```bash
UPDATE_SNAPSHOTS=true go test ./test/helpers/...
# or
make test-snapshots-update
```

Commit the reviewed snapshot files with the code change. Never enable snapshot
updates in CI.

## Coverage Ratchet

`make ci-go` writes `coverage.out` for:

```text
./internal/... ./test/helpers ./test ./pkg/...
```

The current aggregate statement floor is stored in `coverage-baseline.txt`
(18.5%). The nested desktop module and frontend use their own behavioral gates;
they are not included in this root aggregate. Codecov is reporting only and is
not the blocking coverage policy.

Raise the baseline after reviewed coverage improvements. Never lower it merely
to make CI green.

## Architecture And Complexity

`scripts/check-architecture.sh` blocks new direct Gmail-client debt and new
service-interface exceptions relative to `architecture-baseline.csv`. Existing
legacy exceptions are frozen rather than claimed as resolved. Direct `App` field
access is not yet machine-enforced.

`scripts/check-quality-ratchet.py` compares each production Go/TS/JS source file
with `quality-baseline-per-file.csv`. It tracks NLOC, maximum CCN, maximum
function length, and functions above CCN 10. Lizard values are tool-defined
approximations, especially for large Go functions containing closures.

After an intentional reviewed increase, refresh and inspect the baseline:

```bash
make quality-baseline-update
git diff -- quality-baseline-per-file.csv
```

## Mocks

Generated mocks are checked in under `internal/services/mocks`. Do not regenerate
them unless a covered interface changed. `make test-mocks` requires mockery and
must fail before deleting existing mocks when the pinned generator is absent.
Always review generated diffs and rerun `make ci`.

## Pre-commit Hooks

The optional `.pre-commit-config.yaml` provides fast formatting, vet, lint, and
focused checks. It is intentionally smaller than the canonical gate. Passing a
hook does not replace `make ci` before a pull request or completion claim.

## Adding Tests

- Put business-logic tests beside the service implementation.
- Prefer pure frontend unit tests for transformations and state-independent
  behavior.
- Add Playwright coverage for user-visible desktop workflows and coupled modal
  behavior.
- Add TUI snapshots only when a stable textual layout is part of the contract.
- Keep tests deterministic, isolated from user configuration, and safe under
  `go test -race` where the canonical gate enables it.
