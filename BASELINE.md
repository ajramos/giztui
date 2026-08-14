# SDLC Baseline — giztui

Quantitative "before" snapshot of the development process, to compare against a
future quality harness. Every number here comes from `baseline.json`, produced
by `baseline.py` (deterministic, read-only, no network). Reproduce with:

```sh
python3 baseline.py --ref HEAD     # full run -> baseline.json
```

The command requires a complete (non-shallow) clone and an explicit ref. It
fails closed on shallow clones, unresolvable refs, git/archive/extract errors,
and lizard errors. Determinism over the same ref is enforced in CI by
`scripts/verify-baseline-determinism.sh`.

- **Generated from ref:** `HEAD` = `3acefef` · HEAD author date **2026-08-14 (UTC)**
- **History:** first commit **2025-08-07** · repo age **~372 days** · complete
  (not shallow). The 183-day window samples the most recent history; the first
  month inside the window is partial.

> ⚠️ **AI-vs-human attribution is RETIRED** (shock-plan Phase 5). Sections 1 and
> 2 below are historical and describe the old methodology; `baseline.py` no
> longer segments lines or commits by AI/human and no longer reports a
> born-AI/died-human matrix or per-month AI/human volume. Author email is
> recorded observationally only. Complexity, rework ages, and LOC/CCN metrics
> below remain valid.

---

## ⚠️ Read this first — three findings that flip the naïve reading

1. **Complexity is NOT accumulating in existing files.** The headline "+48% total
   CCN" is **dilution by new code, not debt**. Tracking the *fixed cohort* of the
   117 files that existed on day one, total CCN is **flat (9806 → 9960, +1.6%)**
   and mean CCN **falls (4.50 → 4.29)**. The growth is entirely new files (the
   desktop client landed ~2026-07-29, file count 117 → 270). Valid result, stated
   plainly: your original worry ("files grow uncontrollably with generated code")
   is **not** visible in the files you already had.

2. **A line "death" is not the same as "bad code."** The verified AI-born /
   human-deleted example (Appendix B) is a *refactor extraction to a module*, not a
   bug fix. The born-AI/died-human quadrant therefore **conflates "human corrected
   AI code" with "human relocated/refactored AI code."** Treat it as an upper bound
   on real correction, not a defect count.

3. **The "human" segment is contaminated after June.** Author = who committed, not
   who wrote. Your July–August human commits almost certainly contain AI-generated
   code you committed yourself. **June (132 human / 0 AI commits) is the only clean
   human baseline.** This biases every AI-vs-human gap *downward* (the real gap is
   likely larger, not smaller).

---

## 1. Volume & segmentation

Segmentation is by **author email of the commit that introduced each line**
(from `git blame`), not the deleting commit. `ai` = `noreply@anthropic.com`;
everything else = `human`.

| Month | human commits | AI commits |
|---|--:|--:|
| 2026-06 | 132 | 0 |
| 2026-07 | 74 | 208 |
| 2026-08 (to 10th) | 7 | 42 |

- Non-merge commits: **462** · merge commits: **50**.
- "human" spans two emails, same person: `tatto2k@gmail.com` (199) +
  `ajramos@users.noreply.github.com` (14, GitHub-web/squash merges).
- The June→July jump *is* the AI-adoption inflection. It is measured, not inferred
  from diff size.

---

## 2. Rework (the primary metric)

Over **non-merge commits only**. Denominator universe = **42,768** added code
lines; **16,012** line deaths observed. Ages are bucketed and **never summed**:

| Bucket | Deaths | Meaning |
|---|--:|---|
| **< 1 day** | 3,208 | intra-session iteration (not rework) |
| **1–7 days** | 3,574 | real rework |
| **7–30 days** | 8,488 | debt |
| > 30 days | 742 | (censored — repo is young) |

### 2.1 Birth-author × death-author matrix (all ages, n = 16,012)

|            | dies by AI | dies by human |
|---|--:|--:|
| **born AI**    | 10,109 (63.1%) | **1,435 (9.0%)** ← headline cell |
| **born human** | 992 (6.2%) | 3,476 (21.7%) |

AI overwhelmingly rewrites its *own* code (63%). The headline cell — AI-written,
human-deleted — is 9.0%, split by age: **<1d 13 · 1–7d 283 · 7–30d 1,139**. Per
finding #2 above, much of the 7–30d part is refactor-relocation, not correction.

### 2.2 AI vs human — the ONLY defensible comparison (within July, @7/@14)

Comparing segments at @30 would compare June (human) against August (AI), not AI
vs human (see Limitations). Within **July**, both coexist, using censored rates:

| Segment (July births) | births | reworked ≤7d | reworked ≤14d |
|---|--:|--:|--:|
| AI    | 28,889 | **15.7%** | **40.2%** |
| human |  5,339 | 9.0% | 9.1% |

AI-authored July code was reworked ~**1.7× at 7 days and ~4.4× at 14 days** the
human rate — and this *understates* the gap (human segment is contaminated). Note
human rework saturates by day 7 (9.0%→9.1%) while AI keeps rising (15.7%→40.2%).

### 2.3 Descriptive-only cohorts (never a segment comparator)

- **June code (≈100% human), @30 censored: 90.9%.** Early scaffolding was almost
  entirely rewritten within a month — an early-project effect, reported as
  description of June, not as a "human" quality number.
- **All births, censored vs naïve:** @7 16.4% / 15.9% · @14 39.1% / 33.8% ·
  @30 52.9% / 35.7%. (@30 censored is computed on only the 6,539 lines old enough
  to have 30 days of follow-up — mostly June/early-July.)

### 2.4 Top files by real rework (1–7 day deaths)

| File | births | <1d | 1–7d | 7–30d | rate@1–7d |
|---|--:|--:|--:|--:|--:|
| `desktop/frontend/src/App.tsx` | 8092 | 1497 | 1721 | 4313 | 21.3% |
| `internal/config/manager.go` | 0 | 0 | 337 | 0 | n/a (renamed away) |
| `internal/cache/store.go` | 0 | 0 | 319 | 0 | n/a (renamed away) |
| `internal/tui/keys.go` | 603 | 95 | 109 | 190 | 18.1% |
| `desktop/frontend/src/api.ts` | 1236 | 47 | 107 | 958 | 8.7% |
| `desktop/frontend/src/Help.tsx` | 437 | 62 | 80 | 2 | 18.3% |
| `desktop/app.go` | 1230 | 32 | 76 | 824 | 6.2% |

`births=0` rows are files whose lines all died under their old path (rename/delete);
their per-file rate is undefined, not zero.

---

## 3. Complexity trajectory (metric B)

Weekly snapshots of `main`'s first-parent history; lizard over code files with the
same exclusions as §exclusions.

### 3.1 Fixed cohort (117 files present at first snapshot) — real accumulation

| Week ending | total CCN | mean CCN | funcs >10 | >10 / funcs |
|---|--:|--:|--:|--:|
| 2026-06-24 | 9806 | 4.50 | 185 | 0.085 |
| 2026-07-08 | 9841 | 4.48 | 187 | 0.085 |
| 2026-07-29 | 9783 | 4.46 | 185 | 0.084 |
| 2026-08-12 | 9960 | 4.29 | 187 | **0.081** |

**Flat-to-down.** No debt accumulation in the original files.

### 3.2 Global (all files) — for contrast

| Week ending | files | total CCN | funcs >10 | >10 / funcs |
|---|--:|--:|--:|--:|
| 2026-06-24 | 117 | 9806 | 185 | 0.085 |
| 2026-07-29 | 247 | 13570 | 215 | 0.058 |
| 2026-08-12 | 270 | 14646 | 226 | **0.054** |

Total CCN +49% but the **normalized** rate of complex functions **falls** (0.085 →
0.054): new code is on average *less* complex per function than the original core.

### 3.3 Largest file on main

`internal/tui/messages.go` is the LOC peak throughout, essentially **flat
(3299 → 3337)** — never decomposed. No structural split/rename events on
first-parent `main`.

> **Caveat (metric A/B):** weekly snapshots follow `main`'s first-parent history,
> so they miss transient *branch* peaks. `desktop/frontend/src/App.tsx` reached
> ~2,000+ lines on feature branches and was decomposed to <500 before merge (PR
> #73/#69), so `main` never records the peak. The rework analysis (all non-merge
> commits) *does* see it — hence App.tsx tops §2.4.

---

## 4. File-size trajectory (metric A)

LOC = lizard NLOC (non-comment). Per week, `main` first-parent:

- p50 LOC/file: 191 → 113 · p90: 766 → 518 · max: 3299 → 3337.

p50/p90 fall as many small new files enter; the max is stable (messages.go). Growth
split into files present at both ends vs new files is in
`baseline.json → file_size_and_complexity_weekly.top_grew_existing_files` /
`top_new_files` (top existing-file growth is small; the volume is new files).

---

## 5. Reverts & corrective commits

- **Explicit reverts: 0.**
- **Corrective (keyword + touches a file changed in prior 7 days): 98** (of 106
  keyword-matching commits). **Reported with a caveat, not as a clean rate:**
  - Manual classification of 30 → estimated **false-positive ≈ 30–45%** (below the
    50% threshold, so the count is reported). FPs: docs commits ("docs: correct…",
    "docs: fix…"), a feature matched on the word "typo", and refactors that merely
    contain the substring "fix" (`refactor(...): … fixes race`).
  - The 7-day-recency filter barely discriminates (106 keyword → 98 keyword+recent):
    in a repo this small and active, almost everything is recent. The signal is
    effectively "has a fix-ish keyword." **Weak metric; use the count as a ceiling.**

---

## 6. Time to green — DEGRADED (not a quality signal here)

You are a solo dev merging your own branches in a personal repo, so branch
cycle-time measures *time-to-click-merge*, not CI latency. Reported as a plain
count only: **50 merge commits, 462 non-merge commits.** No timing computed. See
Limitations for what a real "time to green" would need.

---

## 7. Not reliably computable / declared limitations

Nothing here is silently approximated.

1. **Window ≠ 6 months.** Whole history is ~8 weeks; all "months" are partial. No
   6-month trend exists.
2. **Segment vs calendar.** June is human-only, August is AI-only, so any @30
   AI-vs-human comparison is really June-vs-August. Only **July @7/@14** is a valid
   segment comparison (§2.2). @30 is descriptive of June (§2.3).
3. **"Human" is contaminated** after June (§⚠️.3). Not corrected — biases gaps down.
4. **A line "death" ≠ a defect** (§⚠️.2). Deaths include refactor relocations, so
   the born-AI/died-human cell is an upper bound on correction, not a bug count.
5. **Corrective FP ≈ 30–45%** (§5); keyword+recency is a weak proxy for intent.
6. **Real CI time-to-green is not computable** without network (GitHub Actions API)
   and would still be meaningless for a solo self-merger. `baseline.py` stays
   deterministic/offline by design; there is no `--github` mode.
7. **Weekly snapshots follow first-parent `main`** and miss branch-transient peaks
   (§3 caveat).
8. **@30 censored rates** rest on a small old subset (6,539 lines with ≥30d
   follow-up); the young repo right-censors everything born after ~2026-07-11.
9. **`ai` is an email heuristic** (`noreply@anthropic.com`). AI code committed under
   a human email is counted human (same as #3).

---

## Appendix A — exclusions (from `baseline.json.meta.exclusions`)

Dropped entirely from line metrics and lizard: paths under `node_modules/`,
`dist/`, `wailsjs/`, `mocks/`, `e2e/`, `test-results/`, `build/`, `graphify-out/`;
suffixes `_test.go`, `.test.ts(x)`, `.spec.ts(x)`, `.d.ts`, `.pb.go`; basenames
`go.sum`, `package-lock.json`; binary/asset extensions (png, jpg, dmg, …). Code
files = `.go .ts .tsx .js .jsx .py .sh`. Renames via `git diff -M`; no `-C` copy
detection (determinism). Merge commits excluded from line metrics.

## Appendix B — manual attribution checks (verified against git)

Both traced by hand to confirm birth-author (blame) / death-author / age:

- **human → AI:** `App.tsx` line born `ce43626` (`ajramos`, human, 2026-08-01),
  died `4607ee1` (`anthropic`, AI, 2026-08-03), age **2.07d**. `git blame
  4607ee1^` at the old line → `ce43626`. ✓
- **AI → human** (the newly-exercised column): `App.tsx` line `type PlanNode =`
  born `71bb809` (`anthropic`, AI, 2026-07-20), died `e51fafb` (`ajramos`, human,
  2026-07-28), age **7.03d**. `git blame e51fafb^ -L 2007` → `71bb809` (anthropic).
  ✓ — but the death is a *refactor extraction* (PR #69), illustrating limitation #4.
