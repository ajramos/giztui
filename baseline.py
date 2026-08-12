#!/usr/bin/env python3
"""
baseline.py — deterministic, re-runnable SDLC baseline for this git repo.

Read-only: reads git history and extracts historical snapshots into a temp dir
(never touches the working tree). No network, no LLM, no randomness. Same commit
range => same baseline.json.

See BASELINE.md "Methodology" and "Not reliably computable" for decisions/limits.
Encoded here:
  * Line-level metrics over NON-MERGE commits only.
  * Each line segmented by the author of the commit that INTRODUCED it (blame),
    not the one that deleted it.
  * Death ages in three buckets, never summed: <1d intrasession iteration,
    1-7d real rework, 7-30d debt.
  * Birth-author x death-author 2x2 matrix; headline cell = born-AI/died-human.
  * AI-vs-human comparison only WITHIN July at @7/@14; @30 descriptive of June.
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

AI_EMAILS = {"noreply@anthropic.com"}
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


def sh_bytes(args):
    return subprocess.run(args, cwd=REPO, capture_output=True).stdout


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


def seg(email):
    return "ai" if email in AI_EMAILS else "human"


def month_utc(epoch):
    import time
    return time.strftime("%Y-%m", time.gmtime(epoch))


def _date(epoch):
    import time
    return time.strftime("%Y-%m-%d", time.gmtime(epoch))


def head_epoch():
    return int(sh(["git", "log", "-1", "--format=%at"]).strip())


def commits_in_window(head_at):
    since = head_at - WINDOW_DAYS * DAY
    out = sh(["git", "log", "--no-merges", "--reverse", "--format=%H|%at|%ae|%s"])
    rows = []
    for line in out.splitlines():
        h, at, ae, subj = line.split("|", 3)
        if int(at) >= since:
            rows.append({"sha": h, "at": int(at), "email": ae, "subj": subj})
    return rows


def merges_in_window(head_at):
    since = head_at - WINDOW_DAYS * DAY
    out = sh(["git", "log", "--merges", "--format=%H|%at|%ae|%s"])
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
    verify_ai_human = []
    for c in commits:
        parent = c["sha"] + "^"
        death_seg = seg(c["email"])
        for fd in parse_commit_diff(parent, c["sha"]):
            path = fd["new_path"] or fd["old_path"]
            if not path or is_excluded(path) or not is_code(path):
                continue
            births.extend([(c["at"], death_seg)] * fd["added"])
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
                b_seg = seg(b_email)
                age = max(0.0, (c["at"] - b_at) / DAY)
                ev = {"file": path, "birth_sha": b_sha[:12], "birth_seg": b_seg,
                      "birth_email": b_email, "birth_at": b_at,
                      "death_sha": c["sha"][:12], "death_seg": death_seg,
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
                if b_seg == "ai" and death_seg == "human" and len(verify_ai_human) < sample_events:
                    v = dict(ev); v["line_text"] = deleted_line_text(parent, blame_path, ln)
                    v["death_line_number"] = ln
                    verify_ai_human.append(v)
    return births, events, per_file, verification, verify_ai_human


def pct(num, denom):
    return {"num": num, "denom": denom,
            "pct": round(100.0 * num / denom, 2) if denom else None}


def summarize_rework(births, events, head_at):
    buckets = {"lt1_intrasession": 0, "d1_7_rework": 0, "d7_30_debt": 0, "gt30": 0}
    for e in events:
        a = e["age_days"]
        buckets["lt1_intrasession" if a < 1 else "d1_7_rework" if a < 7
                else "d7_30_debt" if a < 30 else "gt30"] += 1

    def blank():
        return {"ai_ai": 0, "ai_human": 0, "human_ai": 0, "human_human": 0}
    matrix = {"lt1": blank(), "d1_7": blank(), "d7_30": blank(), "all": blank()}
    for e in events:
        cell = f"{e['birth_seg']}_{e['death_seg']}"
        matrix["all"][cell] += 1
        a = e["age_days"]
        b = "lt1" if a < 1 else "d1_7" if a < 7 else "d7_30" if a < 30 else None
        if b:
            matrix[b][cell] += 1

    def rate_block(bfilter):
        b = [x for x in births if bfilter(x)]
        ev = [e for e in events if bfilter((e["birth_at"], e["birth_seg"]))]
        out = {"births": len(b), "naive": {}, "censored": {}}
        for N in (7, 14, 30):
            out["naive"][str(N)] = pct(sum(1 for e in ev if e["age_days"] <= N), len(b))
            denom = sum(1 for (at, _s) in b if (head_at - at) >= N * DAY)
            num = sum(1 for e in ev if e["age_days"] <= N and (head_at - e["birth_at"]) >= N * DAY)
            out["censored"][str(N)] = pct(num, denom)
        return out

    rates = {
        "all": rate_block(lambda x: True),
        "birth_ai": rate_block(lambda x: x[1] == "ai"),
        "birth_human": rate_block(lambda x: x[1] == "human"),
        "july_ai": rate_block(lambda x: x[1] == "ai" and month_utc(x[0]) == "2026-07"),
        "july_human": rate_block(lambda x: x[1] == "human" and month_utc(x[0]) == "2026-07"),
        "june_all": rate_block(lambda x: month_utc(x[0]) == "2026-06"),
    }
    return {"buckets": buckets, "matrix": matrix, "rates": rates,
            "total_births": len(births), "total_deaths": len(events)}


# ---- reverts & corrective ---------------------------------------------------
def files_of(sha):
    out = sh(["git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha])
    return [p for p in out.splitlines() if p and not is_excluded(p)]


def analyze_reverts(commits):
    reverts = [c for c in commits
               if c["subj"].startswith("Revert") or "git revert" in c["subj"].lower()]
    last_touch = {}
    corrective = []
    kw = 0
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
        for p in touched:
            last_touch[p] = c["at"]
    return {"explicit_reverts": [{"sha": c["sha"][:12], "subj": c["subj"]} for c in reverts],
            "explicit_revert_count": len(reverts),
            "corrective_keyword_commits": kw,
            "corrective_kw_and_recent": len(corrective),
            "corrective_sample": corrective[:30]}


# ---- weekly snapshots -------------------------------------------------------
def pctile(sv, p):
    if not sv:
        return 0
    k = (len(sv) - 1) * p / 100.0
    lo = int(k); hi = min(lo + 1, len(sv) - 1); frac = k - lo
    return round(sv[lo] * (1 - frac) + sv[hi] * frac, 1)


def weekly_anchors(head_at, weeks):
    out = sh(["git", "log", "--first-parent", "--format=%H|%at"])
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
    """rel_path -> {nloc, ccn, funcs, over10} for code files at `sha`."""
    import lizard
    d = os.path.join(tmproot, sha[:12])
    os.makedirs(d, exist_ok=True)
    p = subprocess.Popen(["tar", "-x", "-C", d], stdin=subprocess.PIPE)
    p.communicate(sh_bytes(["git", "archive", "--format=tar", sha]))
    per_file = {}
    for root, _dirs, fnames in os.walk(d):
        for fn in sorted(fnames):
            full = os.path.join(root, fn)
            rel = os.path.relpath(full, d)
            if is_excluded(rel) or not is_code(rel):
                continue
            try:
                info = lizard.analyze_file(full)
            except Exception:
                continue
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
            cur = {"sha": h[:12], "date": _date(int(at)), "seg": seg(ae),
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


def weekly_analysis(head_at, weeks, tmproot):
    anchors = weekly_anchors(head_at, weeks)
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
    ap.add_argument("--sample", type=int, default=0)
    ap.add_argument("--weeks", type=int, default=8)
    ap.add_argument("--out", default="baseline.json")
    ap.add_argument("--cache", default=".baseline_blamecache.json")
    ap.add_argument("--no-weekly", action="store_true")
    args = ap.parse_args()
    REPO = os.path.abspath(args.repo)
    load_cache(os.path.join(REPO, args.cache))

    head_at = head_epoch()
    all_commits = commits_in_window(head_at)
    merges = merges_in_window(head_at)
    rework_commits = all_commits[-args.sample:] if args.sample else all_commits

    births, events, per_file, verification, verify_ai_human = analyze_rework(
        rework_commits, head_at)
    save_cache()
    rework = summarize_rework(births, events, head_at)

    pf = []
    for path, d in per_file.items():
        pf.append({"file": path, **d,
                   "rework_rate_1_7": round(100.0 * d["d_1_7"] / d["births"], 2) if d["births"] else None})
    pf.sort(key=lambda r: (r["d_1_7"], r["deaths"]), reverse=True)

    reverts = analyze_reverts(all_commits)

    email_counts = Counter(c["email"] for c in all_commits)
    human_emails = {e: n for e, n in email_counts.items() if e not in AI_EMAILS}

    timing = {
        "note": "Solo-dev merging own branches in a personal repo. Branch "
                "cycle-time measures time-to-click-merge, not CI latency. "
                "Descriptive count only; not a quality signal here.",
        "merge_commits": len(merges), "non_merge_commits": len(all_commits)}

    weekly = None
    if not args.no_weekly:
        tmproot = tempfile.mkdtemp(prefix="baseline_snap_")
        try:
            weekly = weekly_analysis(head_at, args.weeks, tmproot)
        finally:
            shutil.rmtree(tmproot, ignore_errors=True)

    monthly = defaultdict(lambda: {"ai": 0, "human": 0})
    for c in all_commits:
        monthly[month_utc(c["at"])][seg(c["email"])] += 1

    out = {
        "meta": {
            "generated_from_head": sh(["git", "log", "-1", "--format=%H"]).strip(),
            "head_author_date_utc": _date(head_at),
            "window_days": WINDOW_DAYS,
            "window_note": "Repo history (~8 weeks) is entirely inside the 6-month "
                           "window; June/August are partial months.",
            "ai_emails": sorted(AI_EMAILS),
            "human_emails_note": "'human' = every non-AI author. Spans a local "
                                 "email and a GitHub-web/squash-merge email for the "
                                 "same person. See human_email_breakdown.",
            "human_email_breakdown": human_emails,
            "contamination_note": "Author = who committed, not who wrote. Human "
                                  "commits in Jul-Aug likely contain AI-generated "
                                  "code committed by the human; June is the only "
                                  "clean human baseline. Not corrected.",
            "sample_rework_commits": args.sample or len(all_commits),
            "total_non_merge_commits": len(all_commits),
            "total_merge_commits": len(merges),
            "exclusions": {"substrings": EXCL_SUBSTR, "suffixes": EXCL_SUFFIX,
                           "basenames": sorted(EXCL_BASENAME),
                           "binary_ext": sorted(BINARY_EXT), "code_ext": sorted(CODE_EXT)},
            "merge_handling": "Line metrics over non-merge commits only.",
            "rename_handling": "diff -M; pure renames add no births/deaths. No -C.",
            "segmentation": "By author email of the introducing commit (blame).",
        },
        "monthly_commit_volume": dict(monthly),
        "rework": rework,
        "rework_per_file": pf,
        "verification_samples": verification,
        "verification_ai_born_human_death": verify_ai_human,
        "reverts_and_corrections": reverts,
        "time_to_green": timing,
        "file_size_and_complexity_weekly": weekly,
    }
    with open(os.path.join(REPO, args.out), "w") as f:
        json.dump(out, f, indent=2)
    print(f"wrote {args.out}: births={len(births)} deaths={len(events)} "
          f"commits={len(rework_commits)} ai_human_samples={len(verify_ai_human)}")


if __name__ == "__main__":
    main()
