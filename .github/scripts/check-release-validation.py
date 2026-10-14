#!/usr/bin/env python3
"""Require exact-commit main validation before publishing a VS Code tag."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
from urllib.parse import urlencode

LIMIT, TAG_DEPTH, WAIT_SECONDS = 100, 5, 300
REQUIRED = {
    "ci.yml": ("native-examples", (
        "Observe the pinned compiler identity",
        "Check Go transport code and execute Gooo programs")),
    "vscode-extension.yml": ("package", (
        "Validate extension manifests and package",)),
}
LIVE = {"queued", "in_progress"}


class GateError(Exception):
    """Missing, contradictory, or unsuccessful validation evidence."""


def need(condition, reason):
    if not condition:
        raise GateError(reason)


def positive(value):
    return type(value) is int and value > 0


def identity(value, repository, repository_id):
    need(isinstance(value, dict) and value.get("id") == repository_id
         and type(value.get("id")) is int and value.get("full_name") == repository,
         "repository identity mismatch")


def complete_list(payload, field):
    need(isinstance(payload, dict), "missing list payload")
    items, count = payload.get(field), payload.get("total_count")
    need(type(count) is int and 0 <= count <= LIMIT and isinstance(items, list)
         and count == len(items), "incomplete or oversized " + field + " pagination")
    return items


def activity_order(run):
    """A rerun of an older run ID is newer activity, not an older proof."""
    def instant(key, optional=False):
        value = run.get(key)
        if optional and value is None:
            return datetime.min.replace(tzinfo=timezone.utc)
        need(isinstance(value, str) and value.endswith('Z'), 'invalid run activity timestamp')
        try:
            return datetime.fromisoformat(value[:-1] + '+00:00')
        except ValueError as exc:
            raise GateError('invalid run activity timestamp') from exc
    need(all(positive(run.get(k)) for k in ('id', 'run_number', 'run_attempt')), 'invalid run identity')
    return (max(instant('updated_at'), instant('run_started_at', True)),
            instant('created_at'), run['run_number'], run['id'])


def latest_run(runs):
    need(runs, 'no validation run')
    ranked = [(activity_order(run), run) for run in runs]
    latest_time = max(order[0] for order, run in ranked)
    tied = [run for order, run in ranked if order[0] == latest_time]
    if len(tied) > 1:
        need(all(run.get('status') == 'completed' and run.get('conclusion') == 'success'
                 for run in tied), 'ambiguous latest validation activity')
    return max(ranked, key=lambda item: item[0])[1]


def readiness_projection(snapshot, result):
    """Compare consumed readiness; raw response hashes remain audit evidence."""
    def fields(value, keys):
        return {key: value.get(key) for key in keys}
    def target(value):
        return fields(value, ('sha', 'type', 'url'))
    projected = {'selected': result['selected'],
                 'tag_ref': {**fields(snapshot['tag_ref'], ('ref', 'url')),
                             'object': target(snapshot['tag_ref']['object'])},
                 'tag_objects': [{**fields(obj, ('sha', 'url')), 'object': target(obj['object'])}
                                 for obj in snapshot['tag_objects']], 'workflows': {}}
    for filename in REQUIRED:
        block = snapshot['workflows'][filename]
        current = block['run']
        job = next(j for j in block['jobs']['jobs'] if j['name'] == REQUIRED[filename][0])
        projected['workflows'][filename] = {
            'definition': fields(block['definition'], ('id', 'path', 'state')),
            'run': fields(current, ('id', 'run_attempt', 'run_number', 'status', 'conclusion',
                                   'head_sha', 'head_branch', 'event', 'path', 'workflow_id')),
            'job': fields(job, ('id', 'run_id', 'run_attempt', 'name', 'status', 'conclusion',
                               'head_sha', 'head_branch')),
            'steps': [fields(step, ('name', 'status', 'conclusion')) for step in job['steps']],
        }
    return projected


def check_snapshot(snapshot, repository, repository_id, sha, tag, checkout_sha):
    try:
        return _check_snapshot(snapshot, repository, repository_id, sha, tag, checkout_sha)
    except (TypeError, ValueError, KeyError, AttributeError) as exc:
        raise GateError("malformed validation snapshot") from exc


def _check_snapshot(snapshot, repository, repository_id, sha, tag, checkout_sha):
    """Validate one complete read observation; WAIT never means missing proof."""
    need(isinstance(repository, str) and re.fullmatch(r"[\w.-]+/[\w.-]+", repository)
         and positive(repository_id), "invalid expected repository")
    need(isinstance(sha, str) and re.fullmatch(r"[0-9a-f]{40}", sha)
         and checkout_sha == sha, "checkout SHA mismatch")
    need(isinstance(tag, str) and re.fullmatch(r"vscode-v\d+\.\d+\.\d+(?:-[\w.-]+)?", tag),
         "invalid VS Code release tag")
    need(isinstance(snapshot, dict), "missing snapshot")
    identity(snapshot.get("repo"), repository, repository_id)
    need(snapshot["repo"].get("default_branch") == "main", "unexpected default branch")
    prefix = "https://api.github.com/repos/" + repository
    ref, objects = snapshot.get("tag_ref"), snapshot.get("tag_objects")
    need(isinstance(ref, dict) and ref.get("ref") == "refs/tags/" + tag
         and ref.get("url") == prefix + "/git/refs/tags/" + tag
         and isinstance(objects, list) and len(objects) <= TAG_DEPTH, "invalid tag reference")
    target = ref.get("object", {})
    for obj in objects:
        need(isinstance(target, dict) and target.get("type") == "tag"
             and isinstance(obj, dict) and obj.get("sha") == target.get("sha")
             and isinstance(obj.get("sha"), str) and re.fullmatch(r"[0-9a-f]{40}", obj["sha"])
             and obj.get("url") == prefix + "/git/tags/" + obj.get("sha", ""),
             "annotated tag chain mismatch")
        target = obj.get("object", {})
    need(isinstance(target, dict) and target.get("type") == "commit"
         and target.get("sha") == sha and target.get("url") == prefix + "/git/commits/" + sha,
         "tag does not resolve to release SHA")
    main, comparison = snapshot.get("main"), snapshot.get("compare")
    need(isinstance(main, dict) and main.get("name") == "main"
         and isinstance(main.get("commit"), dict), "missing main identity")
    main_sha = main["commit"].get("sha")
    need(isinstance(main_sha, str) and re.fullmatch(r"[0-9a-f]{40}", main_sha), "invalid main SHA")
    need(isinstance(comparison, dict)
         and comparison.get("status") == ("identical" if main_sha == sha else "ahead")
         and type(comparison.get("behind_by")) is int and comparison["behind_by"] == 0
         and type(comparison.get("ahead_by")) is int
         and (comparison["ahead_by"] == 0 if main_sha == sha else comparison["ahead_by"] > 0)
         and comparison.get("url") == prefix + "/compare/" + sha + "..." + main_sha
         and comparison.get("base_commit", {}).get("sha") == sha
         and comparison.get("merge_base_commit", {}).get("sha") == sha,
         "release SHA is not on current main ancestry")
    selected, waiting = {}, False
    workflows = snapshot.get("workflows", {})
    for filename, (job_name, required_steps) in REQUIRED.items():
        block = workflows.get(filename, {})
        definition = block.get("definition", {})
        workflow_id, path = definition.get("id"), ".github/workflows/" + filename
        need(positive(workflow_id) and definition.get("path") == path
             and definition.get("state") == "active", "invalid workflow " + filename)
        runs = complete_list(block.get("runs"), "workflow_runs")
        need(runs, "no exact-commit main push run for " + filename)
        for run in runs + [block.get("run", {})]:
            need(isinstance(run, dict), "missing current run")
            identity(run.get("repository"), repository, repository_id)
            identity(run.get("head_repository"), repository, repository_id)
            need(all(positive(run.get(k)) for k in ("id", "run_number", "run_attempt"))
                 and run.get("workflow_id") == workflow_id and run.get("path") == path
                 and run.get("event") == "push" and run.get("head_branch") == "main"
                 and run.get("head_sha") == sha, "run binding mismatch for " + filename)
        need(len({r["id"] for r in runs}) == len(runs), "duplicate run identifiers")
        latest, current = latest_run(runs), block["run"]
        need(all(latest[k] == current[k] for k in ("id", "run_number", "run_attempt")),
             "selected run or attempt changed")
        jobs = complete_list(block.get("jobs"), "jobs")
        matches = [j for j in jobs if j.get("name") == job_name]
        need(len(matches) == 1, "missing or ambiguous required job " + job_name)
        job = matches[0]
        need(all(positive(job.get(k)) for k in ("id", "run_id", "run_attempt"))
             and job.get("run_id") == current["id"]
             and job.get("run_attempt") == current["run_attempt"]
             and job.get("head_sha") == sha and job.get("head_branch") == "main",
             "job binding mismatch for " + filename)
        steps = job.get("steps")
        need(isinstance(steps, list), "missing job steps")
        need(not any(s.get("status") == "completed" and s.get("conclusion") != "success"
                     for s in steps), "unsuccessful validation step")
        if current.get("status") in LIVE and job.get("status") in LIVE:
            need(current.get("conclusion") is None and job.get("conclusion") is None,
                 "live run/job has terminal conclusion")
            waiting = True
        else:
            need(current.get("status") == "completed" and current.get("conclusion") == "success"
                 and job.get("status") == "completed" and job.get("conclusion") == "success",
                 "latest validation run/job was not successful")
            need(steps and all(s.get("status") == "completed" and s.get("conclusion") == "success"
                               for s in steps), "incomplete validation steps")
            need(all(sum(s.get("name") == name for s in steps) == 1 for name in required_steps),
                 "required validation step missing or duplicated")
        selected[filename] = {"workflow_id": workflow_id, "run_id": current["id"],
                              "run_number": current["run_number"], "run_attempt": current["run_attempt"],
                              "job_id": job["id"], "job_name": job_name}
    return {"status": "WAIT" if waiting else "PASS", "main_sha": main_sha, "selected": selected}


def fingerprint(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def collect(repository, sha, tag, deadline, reads):
    def get(endpoint):
        remaining = deadline - time.monotonic()
        need(remaining > 0, "validation observation deadline expired")
        try:
            result = subprocess.run(["gh", "api", "--hostname", "github.com", endpoint],
                                    capture_output=True, timeout=min(30, remaining), check=True)
            need(len(result.stdout) <= 2 * 1024 * 1024, "API response too large")
            value = json.loads(result.stdout)
        except (subprocess.SubprocessError, json.JSONDecodeError) as exc:
            raise GateError("GitHub GET failed: " + endpoint) from exc
        reads.append({"endpoint": endpoint, "sha256": fingerprint(value)})
        return value
    base = "repos/" + repository
    snap = {"repo": get(base), "tag_ref": get(base + "/git/ref/tags/" + tag), "tag_objects": []}
    target = snap["tag_ref"].get("object", {})
    while target.get("type") == "tag":
        need(len(snap["tag_objects"]) < TAG_DEPTH, "annotated tag depth exceeded")
        object_sha = target.get("sha", "")
        need(re.fullmatch(r"[0-9a-f]{40}", object_sha), "invalid annotated tag SHA")
        obj = get(base + "/git/tags/" + object_sha)
        snap["tag_objects"].append(obj)
        target = obj.get("object", {})
    snap["main"] = get(base + "/branches/main")
    main_sha = snap["main"].get("commit", {}).get("sha", "")
    need(re.fullmatch(r"[0-9a-f]{40}", main_sha), "invalid main SHA")
    snap["compare"] = get(base + "/compare/" + sha + "..." + main_sha)
    snap["workflows"] = {}
    query = urlencode({"head_sha": sha, "event": "push", "branch": "main", "per_page": LIMIT})
    for filename in REQUIRED:
        endpoint = base + "/actions/workflows/" + filename
        block = {"definition": get(endpoint), "runs": get(endpoint + "/runs?" + query)}
        runs = complete_list(block["runs"], "workflow_runs")
        need(runs and all(positive(r.get("id")) and positive(r.get("run_number")) for r in runs),
             "no valid matching workflow runs")
        latest = latest_run(runs)
        run_endpoint = base + "/actions/runs/" + str(latest["id"])
        block["run"] = get(run_endpoint)
        attempt = block["run"].get("run_attempt")
        need(positive(attempt), "invalid current run attempt")
        block["jobs"] = get(run_endpoint + "/attempts/" + str(attempt) + "/jobs?per_page=100")
        snap["workflows"][filename] = block
    return snap


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("repository", "repository-id", "sha", "tag", "out"):
        parser.add_argument("--" + name, required=True, type=int if name == "repository-id" else str)
    args = parser.parse_args()
    report = {"status": "REFUSED", "repository": args.repository, "repository_id": args.repository_id,
              "sha": args.sha, "tag": args.tag, "source_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              "reads": [], "observations": [], "publication_atomicity": False}
    deadline = time.monotonic() + WAIT_SECONDS
    try:
        need(re.fullmatch(r"[\w.-]+/[\w.-]+", args.repository) and positive(args.repository_id)
             and re.fullmatch(r"[0-9a-f]{40}", args.sha)
             and re.fullmatch(r"vscode-v\d+\.\d+\.\d+(?:-[\w.-]+)?", args.tag), "invalid expected context")
        checkout = subprocess.check_output(["git", "rev-parse", "HEAD"], timeout=10, text=True).strip()
        report["checkout_sha"] = checkout
        need(checkout == args.sha, "checkout SHA mismatch")
        while True:
            snapshot = collect(args.repository, args.sha, args.tag, deadline, report["reads"])
            result = check_snapshot(snapshot, args.repository, args.repository_id, args.sha, args.tag, checkout)
            report["observations"].append({**result, "snapshot_sha256": fingerprint(snapshot)})
            if result["status"] == "PASS":
                final = collect(args.repository, args.sha, args.tag, deadline, report["reads"])
                checked = check_snapshot(final, args.repository, args.repository_id, args.sha, args.tag, checkout)
                need(checked['status'] == 'PASS' and
                     readiness_projection(final, checked) == readiness_projection(snapshot, result),
                     "validation changed during final reread")
                need(subprocess.check_output(["git", "rev-parse", "HEAD"], timeout=10, text=True).strip() == args.sha,
                     "checkout changed during validation")
                need(time.monotonic() <= deadline, "validation observation deadline expired")
                report.update(checked, final_snapshot_sha256=fingerprint(final))
                break
            need(time.monotonic() + 10 < deadline, "live validation did not finish within five minutes")
            time.sleep(10)
    except (GateError, subprocess.SubprocessError, OSError, TypeError, ValueError, KeyError, AttributeError) as exc:
        report["reason"] = str(exc)
    destination = Path(args.out)
    destination.parent.mkdir(parents=True, exist_ok=True)
    with os.fdopen(os.open(destination, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600), "w") as output:
        json.dump(report, output, indent=2)
        output.write("\n")
    print(json.dumps({"status": report["status"], "reason": report.get("reason"), "out": str(destination)}))
    return 0 if report["status"] == "PASS" else 1


if __name__ == "__main__":
    sys.exit(main())
