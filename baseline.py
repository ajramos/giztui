#!/usr/bin/env python3
"""
baseline.py — deterministic, re-runnable SDLC baseline for this git repo.

Read-only: reads git history and extracts historical snapshots into a temp dir
(never touches the working tree). No network, no LLM, no randomness. Same
ref + complete history => same baseline.json.

Usage requires an explicit analysis ref and a complete (non-shallow) clone:

    python3 baseline.py --ref HEAD

The run fails closed on shallow clones, unresolvable refs, git/archive/extract
errors, and lizard analysis errors. Output records the resolved ref, commit
range, and history completeness so the baseline reproduces from an exact point.

See BASELINE.md "Methodology" and "Not reliably computable" for decisions/limits.
Encoded here:
  * Line-level metrics over NON-MERGE commits only.
  * Each line attributed to the commit that INTRODUCED it (blame), not the one
    that deletes it. Author email is recorded observationally only; AI-vs-human
    attribution is RETIRED as a performance measure (see BASELINE.md).
  * Death ages in three buckets, never summed: <1d intrasession iteration,
    1-7d rework, 7-30d debt. The buckets are interpretations of the recorded
    ages; the ages themselves are the observations.
  * "Rework" and "debt" are labelled interpretations. The output separates
    observations (births, deaths, ages, LOC, CCN) from these interpretations.
  * Complexity trajectory reports a FIXED COHORT (files present at first
    snapshot) to separate real accumulation from dilution by new files.
"""
import argparse
import json
import os
import shutil
import subprocess
import tempfile
from collections import defaultdict, Counter

WINDOW_DAYS = 183
DAY = 86400

EXCL_SUBSTR = ["/node_modules/", "/dist/", "/wailsjs/", "/mocks/",
               "/e2e/", "/test-results/", "/build/", "/graphify-out/"]
EXCL_SUFFIX = ["_test.go", ".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx",
               ".d.ts", ".pb.go"]
EXCL_BASENAME = {"go.sum", "package-lock.json", "package-lock.lock"}
BINARY_EXT = {".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf", ".woff",
              ".woff2", ".ttf", ".otf", ".dmg", ".zip", ".gz", ".tar",
              ".mp4", ".mov", ".webp", ".icns"}
CODE_EXT = {".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".sh"}
CORRECTIVE_KW = ["fix", "bug", "hotfix", "regression", "broken", "revert",
                 "correct", "typo", "wrong", "crash", "fixup"]


def sh(args):
    return subprocess.run(args, cwd=REPO, capture_output=True, text=True,
                          errors="replace").stdout


def sh_strict(args):
    """Run git and fail closed on non-zero exit; used for correctness-critical reads."""
    r = subprocess.run(args, cwd=REPO, capture_output=True, text=True,
                       errors="replace")
    if r.returncode != 0:
        raise RuntimeError(
            f"git command failed ({r.returncode}): {' '.join(args)}\n"
            f"{r.stderr.strip()}")
    return r.stdout


def sh_bytes(args):
    return subprocess.run(args, cwd=REPO, capture_output=True).stdout


def guard_repo(ref):
    """Resolve `ref` to a commit SHA, rejecting shallow clones and bad refs."""
    if sh_strict(["git", "rev-parse", "--is-shallow-repository"]).strip() == "true":
        raise SystemExit(
            "refusing to analyze a shallow repository: baseline requires a "
            "complete clone (see BASELINE.md)")
    sha = sh_strict(["git", "rev-parse", f"{ref}^{{commit}}"]).strip()
    if not sha:
        raise SystemExit(f"cannot resolve analysis ref: {ref}")
    return sha


def is_excluded(path):
    p = "/" + path
    if any(s in p for s in EXCL_SUBSTR):
        return True
    if any(path.endswith(s) for s in EXCL_SUFFIX):
        return True
    if os.path.basename(path) in EXCL_BASENAME:
        return True
    _, ext = os.path.splitext(path)
    return ext.lower() in BINARY_EXT


def is_code(path):
    _, ext = os.path.splitext(path)
    return ext.lower() in CODE_EXT


def month_utc(epoch):
    import time
    return time.strftime("%Y-%m", time.gmtime(epoch))


def _date(epoch):
    import time
    return time.strftime("%Y-%m-%d", time.gmtime(epoch))


def head_epoch(ref):
    return int(sh_strict(["git", "log", "-1", f"--format=%at", ref]).strip())


def first_commit_epoch(ref):
    out = sh_strict(["git", "log", "--reverse", f"--format=%at", ref])
    lines = [ln for ln in out.splitlines() if ln.strip()]
    return int(lines[0]) if lines else None


def commits_in_window(head_at, ref):
    since = head_at - WINDOW_DAYS * DAY
    out = sh_strict(["git", "log", "--no-merges", "--reverse",
                     f"--format=%H|%at|%ae|%s", ref])
    rows = []
    for line in out.splitlines():
        h, at, ae, subj = line.split("|", 3)
        if int(at) >= since:
            rows.append({"sha": h, "at": int(at), "email": ae, "subj": subj})
    return rows


def merges_in_window(head_at, ref):
    since = head_at - WINDOW_DAYS * DAY
    out = sh_strict(["git", "log", "--merges", f"--format=%H|%at|%ae|%s", ref])
    return [dict(zip(("sha", "at", "email", "subj"),
                     (p[0], int(p[1]), p[2], p[3])))
            for p in (l.split("|", 3) for l in out.splitlines())
            if int(p[1]) >= since]


# ---- blame cache ------------------------------------------------------------
_BLAME_CACHE = {}
_CACHE_FILE = None


def load_cache(path):
    global _CACHE_FILE
    _CACHE_FILE = path
    if os.path.exists(path):
        with open(path) as f:
            raw = json.load(f)
        for k, v in raw.items():
            _BLAME_CACHE[k] = {int(ln): tuple(t) for ln, t in v.items()}


def save_cache():
    if not _CACHE_FILE:
        return
    ser = {k: {str(ln): list(t) for ln, t in v.items()}
           for k, v in _BLAME_CACHE.items()}
    tmp = _CACHE_FILE + ".tmp"
    with open(tmp, "w") as f:
        json.dump(ser, f)
    os.replace(tmp, _CACHE_FILE)


def blame_map(parent, path):
    """Line number in `parent` -> (sha, author_email, author_at). No -C."""
    key = parent + ":" + path
    if key in _BLAME_CACHE:
        return _BLAME_CACHE[key]
    out = sh(["git", "blame", "-p", parent, "--", path])
    meta = {}
    result = {}
    cur = None
    cur_final = None
    for line in out.split("\n"):
        if not line:
            continue
        if line[0] == "\t":
            if cur is not None and cur_final is not None:
                e, a = meta.get(cur, ("", 0))
                result[cur_final] = (cur, e, a)
            continue
        parts = line.split(" ")
        if len(parts[0]) == 40 and all(c in "0123456789abcdef" for c in parts[0]):
            cur = parts[0]
            try:
                cur_final = int(parts[2])
            except (IndexError, ValueError):
                cur_final = None
        elif line.startswith("author-mail "):
            mail = line[len("author-mail "):].strip().lstrip("<").rstrip(">")
            m = list(meta.get(cur, ["", 0]))
            m[0] = mail
            meta[cur] = tuple(m)
        elif line.startswith("author-time "):
            m = list(meta.get(cur, ["", 0]))
            m[1] = int(line[len("author-time "):].strip())
            meta[cur] = tuple(m)
    _BLAME_CACHE[key] = result
    return result


def deleted_line_text(parent, path, lineno):
    out = sh(["git", "blame", "-p", "-L", f"{lineno},{lineno}", parent, "--", path])
    for line in out.split("\n"):
        if line.startswith("\t"):
            return line[1:]
    return ""


# ---- diff parsing -----------------------------------------------------------
def parse_commit_diff(parent, sha):
    out = sh(["git", "diff", "--unified=0", "-M", "--no-color", parent, sha])
    files = []
    cur = None
    old_no = None
    for line in out.split("\n"):
        if line.startswith("diff --git "):
            if cur:
                files.append(cur)
            cur = {"old_path": None, "new_path": None, "added": 0, "deleted_lines": []}
            old_no = None
        elif line.startswith("--- "):
            p = line[4:]
            cur["old_path"] = None if p == "/dev/null" else (p[2:] if p[:2] in ("a/", "b/") else p)
        elif line.startswith("+++ "):
            p = line[4:]
            cur["new_path"] = None if p == "/dev/null" else (p[2:] if p[:2] in ("a/", "b/") else p)
        elif line.startswith("@@"):
            oldpart = line.split("@@")[1].strip().split(" ")[0]  # -a,b
            a = oldpart[1:]
            old_no = int(a.split(",")[0]) if "," in a else int(a)
        elif line.startswith("-") and not line.startswith("---"):
            if old_no is not None:
                cur["deleted_lines"].append(old_no)
                old_no += 1
        elif line.startswith("+") and not line.startswith("+++"):
            cur["added"] += 1
    if cur:
        files.append(cur)
    return files


# ---- rework -----------------------------------------------------------------
def analyze_rework(commits, head_at, sample_events=8):
    births = []
    events = []
    per_file = defaultdict(lambda: {"births": 0, "deaths": 0,
                                    "d_lt1": 0, "d_1_7": 0, "d_7_30": 0})
    verification = []
    for c in commits:
        parent = c["sha"] + "^"
        for fd in parse_commit_diff(parent, c["sha"]):
            path = fd["new_path"] or fd["old_path"]
            if not path or is_excluded(path) or not is_code(path):
                continue
            births.append(c["at"])
            per_file[path]["births"] += fd["added"]
            if not fd["deleted_lines"]:
                continue
            blame_path = fd["old_path"] or path
            bm = blame_map(parent, blame_path)
            for ln in fd["deleted_lines"]:
                info = bm.get(ln)
                if not info:
                    continue
                b_sha, b_email, b_at = info
                age = max(0.0, (c["at"] - b_at) / DAY)
                ev = {"file": path, "birth_sha": b_sha[:12],
                      "birth_email": b_email, "birth_at": b_at,
                      "death_sha": c["sha"][:12],
                      "death_email": c["email"], "death_at": c["at"],
                      "age_days": round(age, 3)}
                events.append(ev)
                pf = per_file[path]
                pf["deaths"] += 1
                if age < 1:
                    pf["d_lt1"] += 1
                elif age < 7:
                    pf["d_1_7"] += 1
                elif age < 30:
                    pf["d_7_30"] += 1
                if len(verification) < sample_events:
                    v = dict(ev); v["line_text"] = deleted_line_text(parent, blame_path, ln)
                    verification.append(v)
    return births, events, per_file, verification


def pct(num, denom):
    return {"num": num, "denom": denom,
            "pct": round(100.0 * num / denom, 2) if denom else None}


def summarize_rework(births, events, head_at):
    """Rework buckets are INTERPRETATIONS of the recorded death ages."""
    buckets = {"lt1_intrasession": 0, "d1_7_rework": 0, "d7_30_debt": 0, "gt30": 0}
    for e in events:
        a = e["age_days"]
        buckets["lt1_intrasession" if a < 1 else "d1_7_rework" if a < 7
                else "d7_30_debt" if a < 30 else "gt30"] += 1

    def rate_block(bfilter):
        b = [x for x in births if bfilter(x)]
        ev = [e for e in events if bfilter(e["birth_at"])]
        out = {"births": len(b), "naive": {}, "censored": {}}
        for N in (7, 14, 30):
            out["naive"][str(N)] = pct(sum(1 for e in ev if e["age_days"] <= N), len(b))
            denom = sum(1 for at in b if (head_at - at) >= N * DAY)
            num = sum(1 for e in ev if e["age_days"] <= N and (head_at - e["birth_at"]) >= N * DAY)
            out["censored"][str(N)] = pct(num, denom)
        return out

    return {"buckets": buckets, "rates": {"all": rate_block(lambda x: True)},
            "total_births": len(births), "total_deaths": len(events)}


# ---- reverts & corrective ---------------------------------------------------
def files_of(sha):
    out = sh(["git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha])
    return [p for p in out.splitlines() if p and not is_excluded(p)]


def deletion_count(sha):
    """Lines deleted by `sha`, summed over all files (observational)."""
    out = sh_strict(["git", "diff", "--numstat", "--unified=0", f"{sha}^", sha])
    total = 0
    for line in out.splitlines():
        parts = line.split("\t")
        if len(parts) >= 3 and parts[1].isdigit():
            total += int(parts[1])
    return total


def analyze_reverts(commits, head_at, sample_events=12):
    reverts = [c for c in commits
               if c["subj"].startswith("Revert") or "git revert" in c["subj"].lower()]
    last_touch = {}
    corrective = []
    kw = 0
    semantic = []
    for c in commits:
        sl = c["subj"].lower()
        has_kw = any(k in sl for k in CORRECTIVE_KW)
        touched = files_of(c["sha"])
        recent = any((c["at"] - last_touch.get(p, -1e18)) <= 7 * DAY and p in last_touch
                     for p in touched)
        if has_kw:
            kw += 1
        if has_kw and recent:
            corrective.append({"sha": c["sha"][:12], "subj": c["subj"]})
        # Semantic revert: a large deletion (>= 50 lines) landing on files the
        # same commit touched within 7 days, with no explicit Revert/corrective
        # keyword. An interpretation, sampled and bounded.
        if not has_kw and recent and deletion_count(c["sha"]) >= 50:
            semantic.append({"sha": c["sha"][:12], "subj": c["subj"],
                             "deleted": deletion_count(c["sha"])})
        for p in touched:
            last_touch[p] = c["at"]
    return {"explicit_reverts": [{"sha": c["sha"][:12], "subj": c["subj"]} for c in reverts],
            "explicit_revert_count": len(reverts),
            "corrective_keyword_commits": kw,
            "corrective_kw_and_recent": len(corrective),
            "corrective_sample": corrective[:30],
            "semantic_revert_candidates": semantic[:sample_events],
            "semantic_revert_candidate_count": len(semantic)}


# ---- weekly snapshots -------------------------------------------------------
def pctile(sv, p):
    if not sv:
        return 0
    k = (len(sv) - 1) * p / 100.0
    lo = int(k); hi = min(lo + 1, len(sv) - 1); frac = k - lo
    return round(sv[lo] * (1 - frac) + sv[hi] * frac, 1)


def weekly_anchors(head_at, weeks, ref):
    out = sh_strict(["git", "log", "--first-parent", f"--format=%H|%at", ref])
    hist = [(h, int(at)) for h, at in (l.split("|") for l in out.splitlines())]
    anchors, seen = [], set()
    for w in range(weeks):
        boundary = head_at - w * 7 * DAY
        pick = next((h for h, at in hist if at <= boundary), None)
        if pick and pick not in seen:
            seen.add(pick)
            anchors.append((boundary, pick))
    return list(reversed(anchors))


def snapshot_files(sha, tmproot):
    """rel_path -> {nloc, ccn, funcs, over10} for code files at `sha`. Fails closed."""
    import lizard
    d = os.path.join(tmproot, sha[:12])
    os.makedirs(d, exist_ok=True)
    archive = sh_strict(["git", "archive", "--format=tar", sha])
    p = subprocess.run(["tar", "-x", "-C", d], input=archive.encode(), capture_output=True)
    if p.returncode != 0:
        raise RuntimeError(
            f"snapshot extraction failed for {sha}: "
            f"{p.stderr.decode(errors='replace').strip()}")
    per_file = {}
    for root, _dirs, fnames in os.walk(d):
        for fn in sorted(fnames):
            full = os.path.join(root, fn)
            rel = os.path.relpath(full, d)
            if is_excluded(rel) or not is_code(rel):
                continue
            try:
                info = lizard.analyze_file(full)
            except Exception as exc:
                raise RuntimeError(f"lizard failed on {rel} at {sha}: {exc}")
            per_file[rel] = {
                "nloc": info.nloc,
                "ccn": sum(f.cyclomatic_complexity for f in info.function_list),
                "funcs": len(info.function_list),
                "over10": sum(1 for f in info.function_list if f.cyclomatic_complexity > 10),
            }
    shutil.rmtree(d, ignore_errors=True)
    return per_file


def agg_files(per_file, cohort=None):
    items = [(k, v) for k, v in per_file.items() if cohort is None or k in cohort]
    nlocs = sorted(v["nloc"] for _, v in items)
    tccn = sum(v["ccn"] for _, v in items)
    funcs = sum(v["funcs"] for _, v in items)
    over10 = sum(v["over10"] for _, v in items)
    return {"file_count": len(items),
            "loc_p50": pctile(nlocs, 50), "loc_p90": pctile(nlocs, 90),
            "loc_max": max(nlocs) if nlocs else 0,
            "total_ccn": tccn, "mean_ccn": round(tccn / funcs, 3) if funcs else 0,
            "func_count": funcs, "funcs_over_ccn10": over10,
            "over10_ratio": round(over10 / funcs, 4) if funcs else 0}


def trace_peak_file(path, snaps):
    """LOC trajectory of `path` across snapshots + git delete/split/rename events."""
    traj = []
    for boundary, sha, pf in snaps:
        traj.append({"date": _date(boundary),
                     "nloc": pf.get(path, {}).get("nloc"),
                     "present": path in pf})
    out = sh(["git", "log", "--follow", "-M", "--numstat",
              "--format=@@|%H|%at|%ae|%s", "--", path])
    events = []
    cur = None
    for line in out.splitlines():
        if line.startswith("@@|"):
            _, h, at, ae, subj = line.split("|", 4)
            cur = {"sha": h[:12], "date": _date(int(at)), "email": ae,
                   "subj": subj, "added": 0, "deleted": 0, "renamed": False}
            events.append(cur)
        elif cur is not None and "\t" in line:
            parts = line.split("\t")
            if len(parts) >= 3:
                a, dl = parts[0], parts[1]
                if "=>" in parts[2] or " => " in line:
                    cur["renamed"] = True
                cur["added"] += int(a) if a.isdigit() else 0
                cur["deleted"] += int(dl) if dl.isdigit() else 0
    # keep only structurally interesting events: big deletion, rename, or delete
    notable = [e for e in events if e["deleted"] >= 100 or e["renamed"]]
    return {"trajectory": traj, "events": notable[:15]}


def weekly_analysis(head_at, weeks, ref, tmproot):
    anchors = weekly_anchors(head_at, weeks, ref)
    snaps = [(b, s, snapshot_files(s, tmproot)) for b, s in anchors]
    if not snaps:
        return None
    cohort = set(snaps[0][2].keys())
    global_series, fixed_series = [], []
    for b, s, pf in snaps:
        g = agg_files(pf)
        g.update(boundary_date=_date(b), sha=s[:12])
        if pf:
            g["loc_max_file"] = max(pf, key=lambda k: (pf[k]["nloc"], k))
        global_series.append(g)
        fc = agg_files(pf, cohort)
        fc.update(boundary_date=_date(b), sha=s[:12])
        fixed_series.append(fc)
    first_pf, last_pf = snaps[0][2], snaps[-1][2]
    grew = sorted(({"file": f, "start": first_pf[f]["nloc"], "end": last_pf[f]["nloc"],
                    "growth": last_pf[f]["nloc"] - first_pf[f]["nloc"]}
                   for f in set(first_pf) & set(last_pf)),
                  key=lambda x: x["growth"], reverse=True)
    new = sorted(({"file": f, "end": last_pf[f]["nloc"]} for f in set(last_pf) - set(first_pf)),
                 key=lambda x: x["end"], reverse=True)
    peak = max(global_series, key=lambda s: s["loc_max"])
    trace = trace_peak_file(peak["loc_max_file"], snaps)
    return {"global_series": global_series, "fixed_cohort_series": fixed_series,
            "fixed_cohort_size": len(cohort),
            "top_grew_existing_files": grew[:10], "top_new_files": new[:10],
            "peak_loc_file": peak["loc_max_file"], "peak_loc_value": peak["loc_max"],
            "peak_loc_snapshot": peak["boundary_date"], "peak_file_trace": trace}


# ---- main -------------------------------------------------------------------
def main():
    global REPO
    ap = argparse.ArgumentParser()
    ap.add_argument("--repo", default=".")
    ap.add_argument("--ref", required=True,
                    help="explicit analysis ref (commit SHA, branch, or tag)")
    ap.add_argument("--sample", type=int, default=0)
    ap.add_argument("--weeks", type=int, default=8)
    ap.add_argument("--out", default="baseline.json")
    ap.add_argument("--cache", default=".baseline_blamecache.json")
    ap.add_argument("--no-weekly", action="store_true")
    args = ap.parse_args()
    REPO = os.path.abspath(args.repo)

    ref_sha = guard_repo(args.ref)
    load_cache(os.path.join(REPO, args.cache))

    head_at = head_epoch(ref_sha)
    all_commits = commits_in_window(head_at, ref_sha)
    merges = merges_in_window(head_at, ref_sha)
    rework_commits = all_commits[-args.sample:] if args.sample else all_commits

    births, events, per_file, verification = analyze_rework(
        rework_commits, head_at)
    save_cache()
    rework = summarize_rework(births, events, head_at)

    pf = []
    for path, d in per_file.items():
        pf.append({"file": path, **d,
                   "rework_rate_1_7": round(100.0 * d["d_1_7"] / d["births"], 2) if d["births"] else None})
    pf.sort(key=lambda r: (r["d_1_7"], r["deaths"]), reverse=True)

    reverts = analyze_reverts(all_commits, head_at)

    timing = {
        "note": "Solo-dev merging own branches in a personal repo. Branch "
                "cycle-time measures time-to-click-merge, not CI latency. "
                "Descriptive count only; not a quality signal here.",
        "merge_commits": len(merges), "non_merge_commits": len(all_commits)}

    weekly = None
    if not args.no_weekly:
        tmproot = tempfile.mkdtemp(prefix="baseline_snap_")
        try:
            weekly = weekly_analysis(head_at, args.weeks, ref_sha, tmproot)
        finally:
            shutil.rmtree(tmproot, ignore_errors=True)

    monthly = Counter(month_utc(c["at"]) for c in all_commits)

    first_at = first_commit_epoch(ref_sha)
    window_note = (
        "History is older than the analysis window; the window samples the "
        "most recent 183 days and early months inside the window are partial."
        if first_at is not None and (head_at - first_at) > WINDOW_DAYS * DAY
        else "Repo history (window) is entirely inside the 183-day window; "
             "the first month inside the window is partial.")

    out = {
        "meta": {
            "ref": args.ref,
            "analysis_ref_sha": ref_sha,
            "generated_from_head": ref_sha,
            "head_author_date_utc": _date(head_at),
            "first_commit_date_utc": _date(first_at) if first_at else None,
            "repo_age_days": round((head_at - first_at) / DAY, 1) if first_at else None,
            "history_complete": True,
            "shallow_repository": False,
            "range": {"head": ref_sha, "window_days": WINDOW_DAYS,
                      "commits_sampled": len(rework_commits),
                      "total_non_merge_commits": len(all_commits),
                      "total_merge_commits": len(merges)},
            "window_note": window_note,
            "sample_rework_commits": args.sample or len(all_commits),
            "exclusions": {"substrings": EXCL_SUBSTR, "suffixes": EXCL_SUFFIX,
                           "basenames": sorted(EXCL_BASENAME),
                           "binary_ext": sorted(BINARY_EXT), "code_ext": sorted(CODE_EXT)},
            "merge_handling": "Line metrics over non-merge commits only.",
            "rename_handling": "diff -M; pure renames add no births/deaths. No -C.",
            "attribution": "Retired. Author email is recorded observationally; "
                           "AI-vs-human line attribution is not reported as a "
                           "performance measure.",
        },
        "analysis_model": {
            "observations": ["line births", "line deaths", "death ages",
                             "LOC/CCN snapshots", "commit volumes"],
            "interpretations": ["rework buckets (<1d, 1-7d, 7-30d)",
                                "rework rates", "corrective and semantic-revert "
                                "classification"],
            "note": "Buckets and rates interpret the recorded ages; they are "
                    "never summed across buckets.",
        },
        "monthly_commit_volume": dict(sorted(monthly.items())),
        "rework": rework,
        "rework_per_file": pf,
        "verification_samples": verification,
        "reverts_and_corrections": reverts,
        "time_to_green": timing,
        "file_size_and_complexity_weekly": weekly,
    }
    with open(os.path.join(REPO, args.out), "w") as f:
        json.dump(out, f, indent=2)
        f.write("\n")
    print(f"wrote {args.out}: births={len(births)} deaths={len(events)} "
          f"commits={len(rework_commits)} ref={ref_sha[:12]}")


if __name__ == "__main__":
    main()
