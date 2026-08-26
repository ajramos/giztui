# QUALITY-BASELINE

The **CURRENT** complexity mark of the repository. These values are
**ratchet thresholds** (_ratchet_): the mark to **not exceed**. They are not
objectives or ideal values — they describe the worst that already exists today
in the code, so that from here on it does not worsen.

- **Activation date:** 2026-08-14
- **Code base:** `bf0afe5` plus SDLC corrections from plan #87
- **Tool:** lizard 1.23.0 (`lizard . -s cyclomatic_complexity`)
- **Scope:** Go source code + TypeScript/React frontend.

## Ratchet thresholds (do not exceed)

| Metric | Current mark | Where it lives today |
|---|---:|---|
| **Maximum CCN (function)** | **265** | `runCommand` — `desktop/frontend/src/commandRunner.ts` (lines 9–512) |
| **Maximum NLOC (file)** | **3337** | `internal/tui/messages.go` |
| **Maximum length (function)** | **908** | `bindKeys` — `internal/tui/keys.go` (lines 557–1464) |

Interpretation: as long as a change does not generate a function with CCN > 265, nor a
file with NLOC > 3337, nor a function of more than 908 lines, the current mark is
not crossed. Crossing it is a measurable worsening of the worst case in the repo.

## Ratchet per file

`quality-baseline-per-file.csv` is the fine-grained ratchet: one row per
source file with `max_nloc,max_ccn,max_func_len,funcs_over_ccn10`, sorted
by path for stable diffs. It is more useful than the global maximum because a
file that worsens is not masked by another that already holds the record.
Rule: no cell can **increase** relative to the registered row; it can
decrease freely. `make ci-architecture` applies the rule and also rejects
new source files that have not yet been reviewed and accepted in the
baseline.

The baseline was updated when activating the gate to incorporate Gmail rules work after v1.27.0. From this point on, `make
quality-baseline-update` should only be run deliberately after reviewing
the metrics diff; it is not an automatic CI step.

Note: `internal/tui/keys.go` is registered with `max_ccn=228` and
`max_func_len=908`, both from `bindKeys` (the monolithic `SetInputCapture` of
contextual routing), which is a separate issue not addressed here. Freezing it
does not prevent refactoring it: a ratchet of maximums only forbids worsening.

## How to verify

```sh
make ci-tools
make ci-architecture
```

Exclusions applied: dependencies (`node_modules`), built artifacts
(`dist`, `build`), generated bindings (`wailsjs`), generated mocks
(`internal/services/mocks`, marked `DO NOT EDIT`), and all test code
(Go `*_test.go`, TS `*.test/*.spec`, `e2e/`, `test/`).

## Measurement context

- Source files under ratchet: **263**.
- The global maximums of the table above did not change when activating the gate.
- The ratchet freezes existing debt; reduction goals are managed
  separately and are not obtained by regenerating the baseline upward.
