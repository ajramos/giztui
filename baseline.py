#!/usr/bin/env python3
"""
baseline.py — deterministic, re-runnable SDLC baseline for this git repo.

Read-only: reads git history and extracts historical snapshots into a temp dir
(never touches the working tree). No network, no LLM, no randomness. Same commit
range => same baseline.json (the optional --github block is the only exception
and is off by default).

Outputs baseline.json (raw + aggregate + per-file) and renders BASELINE.md.

See BASELINE.md "Methodology" and "Not reliably computable" for the decisions
and their limits. Key ones, encoded below:
  * Line-level metrics run over NON-MERGE commits only (corr #2).
  * Each line is segmented by the author of the commit that INTRODUCED it
    (blame), not the one that deleted it (corr #2).
  * Death ages are reported in three buckets, never summed: <1d intrasession
    iteration, 1-7d real rework, 7-30d debt (corr #3).
  * Birth-author x death-author 2x2 matrix; the headline cell is
    born-AI / died-by-human within 7d (corr #4).
  * AI-vs-human segment comparison is only defensible WITHIN July and only at
    @7/@14; @30 is descriptive of June (human) code and never a comparator
    (corr #1).
"""
import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from collections import defaultdict, Counter

# ---- config -----------------------------------------------------------------
AI_EMAILS = {"noreply@anthropic.com"}
WINDOW_DAYS = 183  # "last 6 months"; entire repo history is younger than this
DAY = 86400

# Exclusions for line-level rework and for the lizard snapshots. Declared
# verbatim into baseline.json meta.
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
    """True if path is generated/vendored/lockfile/binary — dropped entirely."""
    p = "/" + path
    if any(s in p for s in EXCL_SUBSTR):
        return True
    if any(path.endswith(s) for s in EXCL_SUFFIX):
        return True
    if os.path.basename(path) in EXCL_BASENAME:
        return True
    _, ext = os.path.splitext(path)
    if ext.lower() in BINARY_EXT:
        return True
    return False


def is_code(path):
    _, ext = os.path.splitext(path)
    return ext.lower() in CODE_EXT


def seg(email):
    return "ai" if email in AI_EMAILS else "human"


# ---- commit enumeration -----------------------------------------------------
def head_epoch():
    return int(sh(["git", "log", "-1", "--format=%at"]).strip())


def commits_in_window(head_at):
    """Non-merge commits with author-time within the window, oldest first."""
    since = head_at - WINDOW_DAYS * DAY
    out = sh(["git", "log", "--no-merges", "--reverse",
              "--format=%H|%at|%ae|%s"])
    rows = []
    for line in out.splitlines():
        h, at, ae, subj = line.split("|", 3)
        at = int(at)
        if at >= since:
            rows.append({"sha": h, "at": at, "email": ae, "subj": subj})
    return rows


def merges_in_window(head_at):
    since = head_at - WINDOW_DAYS * DAY
    out = sh(["git", "log", "--merges", "--format=%H|%at|%ae|%s"])
    rows = []
    for line in out.splitlines():
        h, at, ae, subj = line.split("|", 3)
        if int(at) >= since:
            rows.append({"sha": h, "at": int(at), "email": ae, "subj": subj})
    return rows


# ---- blame cache ------------------------------------------------------------
_BLAME_CACHE = {}          # (parent_sha, path) -> {lineno: (sha, email, at)}
_CACHE_FILE = None


def load_cache(path):
    global _BLAME_CACHE, _CACHE_FILE
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
    """Line number (in `parent`) -> (introducing_sha, author_email, author_at).
    Deterministic: no -C copy detection. Cached per (parent, path)."""
    key = parent + ":" + path
    if key in _BLAME_CACHE:
        return _BLAME_CACHE[key]
    out = sh(["git", "blame", "-p", "--", path] if False else
             ["git", "blame", "-p", parent, "--", path])
    meta = {}          # sha -> (email, at)
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
            # header: <sha> <orig> <final> [num]
            try:
                cur_final = int(parts[2])
            except (IndexError, ValueError):
                cur_final = None
        elif line.startswith("author-mail "):
            mail = line[len("author-mail "):].strip().lstrip("<").rstrip(">")
            meta.setdefault(cur, ["", 0])
            m = list(meta.get(cur, ["", 0]))
            m[0] = mail
            meta[cur] = tuple(m)
        elif line.startswith("author-time "):
            t = int(line[len("author-time "):].strip())
            m = list(meta.get(cur, ["", 0]))
            m[1] = t
            meta[cur] = tuple(m)
    _BLAME_CACHE[key] = result
    return result


# ---- diff parsing -----------------------------------------------------------
def parse_commit_diff(parent, sha):
    """Return per-file {old_path, new_path, added:int, deleted_lines:[oldno]}.
    unified=0, rename detection (-M), no copy detection."""
    out = sh(["git", "diff", "--unified=0", "-M", "--no-color", parent, sha])
    files = []
    cur = None
    old_no = None
    for line in out.split("\n"):
        if line.startswith("diff --git "):
            if cur:
                files.append(cur)
            cur = {"old_path": None, "new_path": None, "added": 0,
                   "deleted_lines": []}
            old_no = None
        elif line.startswith("--- "):
            p = line[4:]
            cur["old_path"] = None if p == "/dev/null" else p[2:] if p[:2] in ("a/", "b/") else p
        elif line.startswith("+++ "):
            p = line[4:]
            cur["new_path"] = None if p == "/dev/null" else p[2:] if p[:2] in ("a/", "b/") else p
        elif line.startswith("@@"):
            # @@ -a,b +c,d @@
            seg_ = line.split("@@")[1].strip()
            oldpart = seg_.split(" ")[0]  # -a,b
            a = oldpart[1:]
            if "," in a:
                start, cnt = a.split(",")
                old_no = int(start)
                cur["_cnt"] = int(cnt)
            else:
                old_no = int(a)
                cur["_cnt"] = 1
        elif line.startswith("-") and not line.startswith("---"):
            if old_no is not None:
                cur["deleted_lines"].append(old_no)
                old_no += 1
        elif line.startswith("+") and not line.startswith("+++"):
            cur["added"] += 1
    if cur:
        files.append(cur)
    return files


# ---- rework analysis --------------------------------------------------------
def analyze_rework(commits, head_at, sample_events=8):
    births = []          # (at, seg) for every added CODE line in window
    events = []          # death events for CODE lines
    per_file = defaultdict(lambda: {"births": 0, "deaths": 0,
                                     "d_lt1": 0, "d_1_7": 0, "d_7_30": 0})
    verification = []
    for c in commits:
        parent = c["sha"] + "^"
        death_email = c["email"]
        death_seg = seg(death_email)
        for fd in parse_commit_diff(parent, c["sha"]):
            path = fd["new_path"] or fd["old_path"]
            if not path or is_excluded(path) or not is_code(path):
                continue
            # births: added lines are introduced by THIS commit
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
                b_seg = seg(b_email)                 # AUTHOR AT BIRTH (corr #2)
                age = (c["at"] - b_at) / DAY
                if age < 0:
                    age = 0.0
                ev = {"file": path, "birth_sha": b_sha[:12],
                      "birth_seg": b_seg, "birth_at": b_at,
                      "death_sha": c["sha"][:12], "death_seg": death_seg,
                      "death_at": c["at"], "age_days": round(age, 3)}
                events.append(ev)
                per_file[path]["deaths"] += 1
                if age < 1:
                    per_file[path]["d_lt1"] += 1
                elif age < 7:
                    per_file[path]["d_1_7"] += 1
                elif age < 30:
                    per_file[path]["d_7_30"] += 1
                if len(verification) < sample_events:
                    vv = dict(ev)
                    vv["birth_line_text"] = deleted_line_text(parent, blame_path, ln)
                    verification.append(vv)
    return births, events, per_file, verification


def deleted_line_text(parent, path, lineno):
    out = sh(["git", "blame", "-p", "-L", f"{lineno},{lineno}", parent,
              "--", path])
    for line in out.split("\n"):
        if line.startswith("\t"):
            return line[1:]
    return ""


def month_utc(epoch):
    import time
    return time.strftime("%Y-%m", time.gmtime(epoch))


def summarize_rework(births, events, head_at):
    # buckets (corr #3) — never summed
    buckets = {"lt1_intrasession": 0, "d1_7_rework": 0, "d7_30_debt": 0,
               "gt30": 0}
    for e in events:
        a = e["age_days"]
        if a < 1:
            buckets["lt1_intrasession"] += 1
        elif a < 7:
            buckets["d1_7_rework"] += 1
        elif a < 30:
            buckets["d7_30_debt"] += 1
        else:
            buckets["gt30"] += 1

    # birth x death 2x2 matrix, split by bucket (corr #4)
    def blank():
        return {"ai_ai": 0, "ai_human": 0, "human_ai": 0, "human_human": 0}
    matrix = {"lt1": blank(), "d1_7": blank(), "d7_30": blank(), "all": blank()}
    for e in events:
        cell = f"{e['birth_seg']}_{e['death_seg']}"
        matrix["all"][cell] += 1
        a = e["age_days"]
        if a < 1:
            matrix["lt1"][cell] += 1
        elif a < 7:
            matrix["d1_7"][cell] += 1
        elif a < 30:
            matrix["d7_30"][cell] += 1

    # censored + naive rates, by birth segment and by birth month
    def rate_block(birth_filter):
        b = [x for x in births if birth_filter(x)]
        total = len(b)
        ev = [e for e in events if birth_filter((e["birth_at"], e["birth_seg"]))]
        out = {"births": total, "naive": {}, "censored": {}}
        for N in (7, 14, 30):
            num_naive = sum(1 for e in ev if e["age_days"] <= N)
            out["naive"][str(N)] = pct(num_naive, total)
            # censored: only births with >=N days of follow-up
            denom = sum(1 for (at, _s) in b if (head_at - at) >= N * DAY)
            num = sum(1 for e in ev
                      if e["age_days"] <= N and (head_at - e["birth_at"]) >= N * DAY)
            out["censored"][str(N)] = pct(num, denom)
        return out

    rates = {
        "all": rate_block(lambda x: True),
        "birth_ai": rate_block(lambda x: x[1] == "ai"),
        "birth_human": rate_block(lambda x: x[1] == "human"),
        # corr #1: the ONLY defensible AI-vs-human comparison — within July, @7/@14
        "july_ai": rate_block(lambda x: x[1] == "ai" and month_utc(x[0]) == "2026-07"),
        "july_human": rate_block(lambda x: x[1] == "human" and month_utc(x[0]) == "2026-07"),
        # descriptive only (corr #1): June is ~100% human; @30 belongs here
        "june_all": rate_block(lambda x: month_utc(x[0]) == "2026-06"),
    }
    return {"buckets": buckets, "matrix": matrix, "rates": rates,
            "total_births": len(births), "total_deaths": len(events)}


def pct(num, denom):
    return {"num": num, "denom": denom,
            "pct": round(100.0 * num / denom, 2) if denom else None}


# ---- reverts & corrective commits ------------------------------------------
def analyze_reverts(commits):
    reverts = [c for c in commits
               if c["subj"].startswith("Revert") or "git revert" in c["subj"].lower()]
    # corrective: touches a file modified in prior 7 days AND keyword in subject
    file_last_touch = {}       # path -> last author-time seen (walking oldest->newest)
    corrective = []
    kw_hits = []
    for c in commits:
        subj_l = c["subj"].lower()
        has_kw = any(k in subj_l for k in CORRECTIVE_KW)
        touched = files_of(c["sha"])
        recent = False
        for p in touched:
            lt = file_last_touch.get(p)
            if lt is not None and (c["at"] - lt) <= 7 * DAY:
                recent = True
                break
        if has_kw:
            kw_hits.append(c["sha"])
        if has_kw and recent:
            corrective.append({"sha": c["sha"][:12], "subj": c["subj"]})
        for p in touched:
            file_last_touch[p] = c["at"]
    return {"explicit_reverts": [{"sha": c["sha"][:12], "subj": c["subj"]} for c in reverts],
            "explicit_revert_count": len(reverts),
            "corrective_keyword_commits": len(kw_hits),
            "corrective_kw_and_recent": len(corrective),
            "corrective_sample": corrective[:25]}


def files_of(sha):
    out = sh(["git", "diff-tree", "--no-commit-id", "--name-only", "-r", sha])
    return [p for p in out.splitlines() if p and not is_excluded(p)]


# ---- weekly snapshots: file size (A) + complexity (B) ----------------------
def weekly_anchors(head_at, weeks):
    """Last first-parent commit at or before each weekly boundary, newest set."""
    out = sh(["git", "log", "--first-parent", "--format=%H|%at"])
    hist = [(h, int(at)) for h, at in (l.split("|") for l in out.splitlines())]
    anchors = []
    seen = set()
    for w in range(weeks):
        boundary = head_at - w * 7 * DAY
        pick = next((h for h, at in hist if at <= boundary), None)
        if pick and pick not in seen:
            seen.add(pick)
            anchors.append((boundary, pick))
    return list(reversed(anchors))


def snapshot_metrics(sha, tmproot):
    import lizard
    d = os.path.join(tmproot, sha[:12])
    os.makedirs(d, exist_ok=True)
    tar = sh_bytes(["git", "archive", "--format=tar", sha])
    p = subprocess.Popen(["tar", "-x", "-C", d], stdin=subprocess.PIPE)
    p.communicate(tar)
    file_nloc = {}
    total_ccn = 0
    fn_count = 0
    over10 = 0
    ccn_list = []
    for root, _dirs, fnames in os.walk(d):
        for fn in fnames:
            full = os.path.join(root, fn)
            rel = os.path.relpath(full, d)
            if is_excluded(rel) or not is_code(rel):
                continue
            try:
                info = lizard.analyze_file(full)
            except Exception:
                continue
            file_nloc[rel] = info.nloc
            for f in info.function_list:
                fn_count += 1
                total_ccn += f.cyclomatic_complexity
                ccn_list.append(f.cyclomatic_complexity)
                if f.cyclomatic_complexity > 10:
                    over10 += 1
    shutil.rmtree(d, ignore_errors=True)
    nlocs = sorted(file_nloc.values())
    return {
        "file_count": len(file_nloc),
        "loc_p50": pctile(nlocs, 50),
        "loc_p90": pctile(nlocs, 90),
        "loc_max": max(nlocs) if nlocs else 0,
        "loc_max_file": max(file_nloc, key=file_nloc.get) if file_nloc else None,
        "total_ccn": total_ccn,
        "mean_ccn": round(total_ccn / fn_count, 3) if fn_count else 0,
        "func_count": fn_count,
        "funcs_over_ccn10": over10,
        "_file_nloc": file_nloc,
    }


def pctile(sorted_vals, p):
    if not sorted_vals:
        return 0
    k = (len(sorted_vals) - 1) * p / 100.0
    lo = int(k)
    hi = min(lo + 1, len(sorted_vals) - 1)
    frac = k - lo
    return round(sorted_vals[lo] * (1 - frac) + sorted_vals[hi] * frac, 1)


def weekly_analysis(head_at, weeks, tmproot):
    anchors = weekly_anchors(head_at, weeks)
    series = []
    first_nloc = None
    last_nloc = None
    for boundary, sha in anchors:
        m = snapshot_metrics(sha, tmproot)
        fn = m.pop("_file_nloc")
        if first_nloc is None:
            first_nloc = fn
        last_nloc = fn
        m["week_ending"] = month_utc(boundary)
        m["boundary_date"] = _date(boundary)
        m["sha"] = sha[:12]
        series.append(m)
    # top-10 absolute growth files (first vs last snapshot)
    growth = []
    if first_nloc is not None and last_nloc is not None:
        for f in set(list(first_nloc) + list(last_nloc)):
            g = last_nloc.get(f, 0) - first_nloc.get(f, 0)
            growth.append({"file": f, "start": first_nloc.get(f, 0),
                           "end": last_nloc.get(f, 0), "growth": g})
        growth.sort(key=lambda x: x["growth"], reverse=True)
    return {"series": series, "top_growth": growth[:10]}


def _date(epoch):
    import time
    return time.strftime("%Y-%m-%d", time.gmtime(epoch))


# ---- main -------------------------------------------------------------------
def main():
    global REPO
    ap = argparse.ArgumentParser()
    ap.add_argument("--repo", default=".")
    ap.add_argument("--sample", type=int, default=0,
                    help="limit rework to N most-recent non-merge commits (0=all)")
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

    rework_commits = all_commits
    if args.sample:
        rework_commits = all_commits[-args.sample:]

    births, events, per_file, verification = analyze_rework(rework_commits, head_at)
    save_cache()
    rework = summarize_rework(births, events, head_at)

    # per-file table (code files), sorted by real-rework (1-7d) deaths desc
    pf = []
    for path, d in per_file.items():
        row = {"file": path, **d,
               "rework_rate_1_7": round(100.0 * d["d_1_7"] / d["births"], 2) if d["births"] else None}
        pf.append(row)
    pf.sort(key=lambda r: (r["d_1_7"], r["deaths"]), reverse=True)

    reverts = analyze_reverts(all_commits)

    # time-to-green: DOWNGRADED to descriptive count only (corr #6)
    timing = {
        "note": "Solo-dev merging own branches in a personal repo. Branch "
                "cycle-time measures time-to-click-merge, not CI latency. "
                "Reported as a plain count; not a quality signal here.",
        "merge_commits": len(merges),
        "non_merge_commits": len(all_commits),
    }

    weekly = None
    if not args.no_weekly:
        tmproot = tempfile.mkdtemp(prefix="baseline_snap_")
        try:
            weekly = weekly_analysis(head_at, args.weeks, tmproot)
        finally:
            shutil.rmtree(tmproot, ignore_errors=True)

    # monthly volume + segment split
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
            "sample_rework_commits": args.sample or len(all_commits),
            "total_non_merge_commits": len(all_commits),
            "total_merge_commits": len(merges),
            "exclusions": {"substrings": EXCL_SUBSTR, "suffixes": EXCL_SUFFIX,
                           "basenames": sorted(EXCL_BASENAME),
                           "binary_ext": sorted(BINARY_EXT),
                           "code_ext": sorted(CODE_EXT)},
            "merge_handling": "Line metrics over non-merge commits only; merges "
                              "counted separately.",
            "rename_handling": "diff -M detects renames; pure renames add no "
                               "births/deaths. No -C copy detection (determinism).",
            "segmentation": "By author email of the INTRODUCING commit (blame).",
        },
        "monthly_commit_volume": dict(monthly),
        "rework": rework,
        "rework_per_file": pf,
        "verification_samples": verification,
        "reverts_and_corrections": reverts,
        "time_to_green": timing,
        "file_size_and_complexity_weekly": weekly,
    }
    with open(os.path.join(REPO, args.out), "w") as f:
        json.dump(out, f, indent=2, sort_keys=False)
    print(f"wrote {args.out}: births={len(births)} deaths={len(events)} "
          f"commits_analyzed={len(rework_commits)}")


if __name__ == "__main__":
    main()
