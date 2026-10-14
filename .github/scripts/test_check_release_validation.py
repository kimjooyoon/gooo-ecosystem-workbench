"""Finite, offline validator tests; mutations do not prove CI/publication or an IDE.

The baseline projects twelve actual GitHub GETs for vscode-v0.2.4 plus four
timestamp refresh GETs. Every changed record below is a synthetic case. All
collector/CLI subprocesses and waits are mocked: this suite cannot publish.
"""
import contextlib
import copy
from datetime import datetime, timedelta, timezone
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit


HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("release_validation", HERE / "check-release-validation.py")
GATE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GATE)
RECORDED = json.loads((HERE / "testdata/release-validation-024.json").read_text())
REPOSITORY = "kimjooyoon/gooo-ecosystem-workbench"
REPOSITORY_ID = 1404918815
SHA = "e982ab9f70d0c8d754a67c825d39f45f17448e92"
TAG = "vscode-v0.2.4"
OTHER_SHA = "a" * 40
PREFIX = "https://api.github.com/repos/" + REPOSITORY
FILES = ("ci.yml", "vscode-extension.yml")
EXPECTED_REQUIRED = {
    "ci.yml": ("native-examples", (
        "Observe the pinned compiler identity",
        "Check Go transport code and execute Gooo programs")),
    "vscode-extension.yml": ("package", (
        "Validate extension manifests and package",)),
}


def snapshot():
    value = copy.deepcopy(RECORDED)
    value.pop("_record")
    return value


def live_snapshot(filename="ci.yml", status="in_progress"):
    """Synthetic same-job live observation, derived from the real identities."""
    value = snapshot()
    block = value["workflows"][filename]
    for run in (block["runs"]["workflow_runs"][0], block["run"]):
        run.update(status=status, conclusion=None)
    job = block["jobs"]["jobs"][0]
    job.update(status=status, conclusion=None)
    job["steps"] = [{"name": "Set up job", "status": status, "conclusion": None}]
    return value


class ReleaseValidationTests(unittest.TestCase):
    def check(self, value, **changes):
        args = dict(repository=REPOSITORY, repository_id=REPOSITORY_ID,
                    sha=SHA, tag=TAG, checkout_sha=SHA)
        args.update(changes)
        return GATE.check_snapshot(value, **args)

    def refuse(self, value, **changes):
        with self.assertRaises(GATE.GateError):
            self.check(value, **changes)

    def test_actual_recorded_projection_has_exact_selected_runs(self):
        self.assertIn("projection", RECORDED["_record"]["kind"])
        self.assertEqual(len(RECORDED["_record"]["reads"]), 12)
        self.assertEqual(len(RECORDED["_record"]["timestamp_projection_refresh"]["reads"]), 4)
        self.assertEqual(RECORDED["_record"]["source_sha256"], "873e1550b83bfa6b9e070df1aa8f902b8b4f555976014d4a2131848ef9a7a5a0")
        self.assertEqual(GATE.REQUIRED, EXPECTED_REQUIRED)
        self.assertEqual((GATE.LIMIT, GATE.TAG_DEPTH, GATE.WAIT_SECONDS), (100, 5, 300))
        self.assertEqual([len(RECORDED["workflows"][name]["jobs"]["jobs"][0]["steps"]) for name in FILES], [38, 10])
        result = self.check(snapshot())
        self.assertEqual(result["status"], "PASS")
        self.assertEqual(result["main_sha"], SHA)
        for name, run_id, job_id in (("ci.yml", 38579526665, 115905415857),
                                     ("vscode-extension.yml", 38579526678, 115905414116)):
            self.assertEqual(result["selected"][name], {
                "workflow_id": RECORDED["workflows"][name]["definition"]["id"],
                "run_id": run_id, "run_number": RECORDED["workflows"][name]["run"]["run_number"],
                "run_attempt": 1, "job_id": job_id,
                "job_name": "native-examples" if name == "ci.yml" else "package"})

    def test_expected_context_must_match_checkout_and_repository(self):
        for change in ({"repository": "wrong/repository"}, {"repository_id": REPOSITORY_ID + 1},
                       {"repository_id": True}, {"checkout_sha": OTHER_SHA},
                       {"sha": SHA.upper()}, {"tag": "vscode-v0.2.3"}, {"tag": "../tag"}):
            with self.subTest(change=change):
                self.refuse(snapshot(), **change)
        for field, value in (("id", True), ("full_name", "wrong/repository"), ("default_branch", "master")):
            current = snapshot()
            current["repo"][field] = value
            self.refuse(current)

    def test_exact_lightweight_tag_and_bounded_annotated_chain(self):
        for field, value in (("ref", "refs/tags/other"), ("url", PREFIX + "/git/refs/tags/other")):
            current = snapshot()
            current["tag_ref"][field] = value
            self.refuse(current)
        for field, value in (("sha", OTHER_SHA), ("type", "tree"), ("url", "https://other/commit")):
            current = snapshot()
            current["tag_ref"]["object"][field] = value
            self.refuse(current)
        annotated = snapshot()
        commit = copy.deepcopy(annotated["tag_ref"]["object"])
        tag_object = {"sha": OTHER_SHA, "type": "tag", "url": PREFIX + "/git/tags/" + OTHER_SHA}
        annotated["tag_ref"]["object"] = tag_object
        annotated["tag_objects"] = [{"sha": OTHER_SHA, "url": tag_object["url"], "object": commit}]
        self.assertEqual(self.check(annotated)["status"], "PASS")  # Synthetic annotated-tag support.
        for field, value in (("sha", "b" * 40), ("url", "https://other/tag")):
            current = copy.deepcopy(annotated)
            current["tag_objects"][0][field] = value
            self.refuse(current)
        annotated["tag_objects"] *= GATE.TAG_DEPTH + 1
        self.refuse(annotated)

    def test_main_ancestry_accepts_only_identical_or_forward_history(self):
        advanced = snapshot()
        advanced["main"]["commit"]["sha"] = OTHER_SHA
        advanced["compare"].update(status="ahead", ahead_by=1, url=PREFIX + "/compare/" + SHA + "..." + OTHER_SHA)
        self.assertEqual(self.check(advanced)["main_sha"], OTHER_SHA)  # Synthetic historic tag on main.
        for field, value in (("status", "diverged"), ("status", "behind"), ("behind_by", 1),
                             ("behind_by", False), ("ahead_by", True), ("ahead_by", 1),
                             ("url", PREFIX + "/compare/" + OTHER_SHA + "..." + SHA)):
            current = snapshot()
            current["compare"][field] = value
            self.refuse(current)
        for field in ("base_commit", "merge_base_commit"):
            current = snapshot()
            current["compare"][field]["sha"] = OTHER_SHA
            self.refuse(current)
        advanced["compare"]["ahead_by"] = 0
        self.refuse(advanced)
        current = snapshot()
        current["main"]["name"] = "other"
        self.refuse(current)

    def test_each_workflow_definition_and_run_binding_is_required(self):
        for name in FILES:
            for field, value in (("id", True), ("state", "disabled_manually"), ("path", ".github/workflows/other.yml")):
                with self.subTest(workflow=name, definition=field):
                    current = snapshot()
                    current["workflows"][name]["definition"][field] = value
                    self.refuse(current)
            for target in ("listed", "current"):
                for field, value in (("event", "pull_request"), ("event", "workflow_dispatch"),
                                     ("head_branch", TAG), ("head_sha", OTHER_SHA), ("workflow_id", 1),
                                     ("path", ".github/workflows/other.yml"), ("run_attempt", True),
                                     ("run_number", 0), ("id", 0)):
                    with self.subTest(workflow=name, target=target, field=field, value=value):
                        current = snapshot()
                        block = current["workflows"][name]
                        run = block["run"] if target == "current" else block["runs"]["workflow_runs"][0]
                        run[field] = value
                        self.refuse(current)
                for key in ("repository", "head_repository"):
                    for field, value in (("id", REPOSITORY_ID + 1), ("full_name", "other/fork")):
                        current = snapshot()
                        block = current["workflows"][name]
                        run = block["run"] if target == "current" else block["runs"]["workflow_runs"][0]
                        run[key][field] = value
                        self.refuse(current)

    def test_newest_failure_or_retry_cannot_reuse_old_success(self):
        for name in FILES:
            value = snapshot()
            block = value["workflows"][name]
            older = copy.deepcopy(block["run"])
            older["id"] -= 1
            older["run_number"] -= 1
            older.update(created_at="2026-10-14T14:10:00Z", run_started_at="2026-10-14T14:10:00Z", updated_at="2026-10-14T14:18:00Z")
            block["runs"]["workflow_runs"].insert(0, older)
            block["runs"]["total_count"] = 2
            self.assertEqual(self.check(value)["selected"][name]["run_id"], block["run"]["id"])
            block["run"]["conclusion"] = "failure"
            block["runs"]["workflow_runs"][1]["conclusion"] = "failure"
            self.refuse(value)
            value = snapshot()
            block = value["workflows"][name]
            newer = copy.deepcopy(block["run"])
            newer["id"] += 1
            newer["run_number"] += 1
            newer["updated_at"] = "2026-10-14T16:00:00Z"
            block["runs"]["workflow_runs"].append(newer)
            block["runs"]["total_count"] = 2
            self.refuse(value)  # The separate current GET must select the newest run.
            value = snapshot()
            value["workflows"][name]["run"]["run_attempt"] = 2
            self.refuse(value)  # A rerun started after list discovery.

    def test_older_run_new_attempt_cannot_hide_behind_newer_success(self):
        for name in FILES:
            value = snapshot()
            block = value["workflows"][name]
            older = copy.deepcopy(block["run"])
            older.update(id=older["id"] - 1, run_number=older["run_number"] - 1, run_attempt=2,
                         created_at="2026-10-14T14:10:00Z", run_started_at="2026-10-14T16:00:00Z",
                         updated_at="2026-10-14T16:00:00Z", status="in_progress", conclusion=None)
            block["runs"]["workflow_runs"].insert(0, older)
            block["runs"]["total_count"] = 2
            self.refuse(value)  # A GET of the newer-ID success cannot stand for latest activity.
            block["run"] = copy.deepcopy(older)
            job = block["jobs"]["jobs"][0]
            job.update(id=job["id"] + 100, run_id=older["id"], run_attempt=2,
                       status="in_progress", conclusion=None)
            job["steps"] = [{"name": "Set up job", "status": "in_progress", "conclusion": None}]
            selected = self.check(value)
            self.assertEqual(selected["status"], "WAIT")
            self.assertEqual(selected["selected"][name]["run_id"], older["id"])
            self.assertEqual(selected["selected"][name]["run_attempt"], 2)
            block["run"].update(status="completed", conclusion="failure")
            block["runs"]["workflow_runs"][0].update(status="completed", conclusion="failure")
            job.update(status="completed", conclusion="failure")
            self.refuse(value)

    def test_required_run_activity_dates_must_be_parseable(self):
        for name in FILES:
            for field in ("updated_at", "created_at"):
                for value in (None, "not-a-dateZ", "2026-10-14T15:00:00"):
                    current = snapshot()
                    current["workflows"][name]["runs"]["workflow_runs"][0][field] = value
                    self.refuse(current)
            current = snapshot()
            current["workflows"][name]["runs"]["workflow_runs"][0]["run_started_at"] = None
            self.assertEqual(self.check(current)["status"], "PASS")  # API start may be absent for queued work.

    def test_same_top_second_cannot_choose_success_over_unsafe_rerun(self):
        for name in FILES:
            current = snapshot()
            block = current["workflows"][name]
            older = copy.deepcopy(block["run"])
            older.update(id=older["id"] - 1, run_number=older["run_number"] - 1, run_attempt=2,
                         created_at="2026-10-14T14:10:00Z", run_started_at=older["updated_at"])
            block["runs"]["workflow_runs"].insert(0, older)
            block["runs"]["total_count"] = 2
            self.assertEqual(self.check(current)["status"], "PASS")  # Both successes have deterministic ordering.
            for status, conclusion in (("completed", "failure"), ("in_progress", None), ("queued", None)):
                value = copy.deepcopy(current)
                value["workflows"][name]["runs"]["workflow_runs"][0].update(status=status, conclusion=conclusion)
                self.refuse(value)  # Created-at/ID tie breakers cannot hide unsafe activity in this second.

    def test_exact_attempt_named_job_and_job_identity(self):
        for name in FILES:
            for field, value in (("run_id", 1), ("run_attempt", 2), ("run_attempt", True),
                                 ("head_sha", OTHER_SHA), ("head_branch", TAG), ("id", True),
                                 ("name", "unrelated")):
                current = snapshot()
                current["workflows"][name]["jobs"]["jobs"][0][field] = value
                self.refuse(current)
            for count in (0, 2):
                current = snapshot()
                jobs = current["workflows"][name]["jobs"]
                jobs["jobs"] *= count
                jobs["total_count"] = count
                self.refuse(current)

    def test_required_steps_cannot_be_missing_duplicated_or_skipped(self):
        for name, (_, required_names) in EXPECTED_REQUIRED.items():
            for required_name in required_names:
                for action in ("remove", "duplicate", "failure", "skipped", "cancelled", "neutral", "live"):
                    with self.subTest(workflow=name, step=required_name, action=action):
                        current = snapshot()
                        steps = current["workflows"][name]["jobs"]["jobs"][0]["steps"]
                        step = next(s for s in steps if s["name"] == required_name)
                        if action == "remove":
                            steps.remove(step)
                        elif action == "duplicate":
                            steps.append(copy.deepcopy(step))
                        elif action == "live":
                            step.update(status="in_progress", conclusion=None)
                        else:
                            step["conclusion"] = action
                        self.refuse(current)
            current = snapshot()
            current["workflows"][name]["jobs"]["jobs"][0]["steps"][-1]["conclusion"] = "failure"
            self.refuse(current)  # Cleanup/other job steps must also have succeeded.
            for target in ("run", "job"):
                for conclusion in ("failure", "cancelled", "timed_out", "skipped", None):
                    current = snapshot()
                    block = current["workflows"][name]
                    terminal = block["run"] if target == "run" else block["jobs"]["jobs"][0]
                    terminal["conclusion"] = conclusion
                    self.refuse(current)

    def test_lists_are_complete_capped_unique_and_not_empty(self):
        for name in FILES:
            for kind, field in (("runs", "workflow_runs"), ("jobs", "jobs")):
                for count in (True, -1, 0, 2, GATE.LIMIT + 1):
                    current = snapshot()
                    current["workflows"][name][kind]["total_count"] = count
                    self.refuse(current)
            current = snapshot()
            runs = current["workflows"][name]["runs"]
            runs["workflow_runs"] *= 2
            runs["total_count"] = 2
            self.refuse(current)
            current = snapshot()
            current["workflows"][name]["runs"] = {"total_count": 0, "workflow_runs": []}
            self.refuse(current)

    def test_complete_lists_at_limit_pass_and_above_limit_refuse(self):
        for kind in ("runs", "jobs"):
            for count in (100, 101):
                current = snapshot()
                block = current["workflows"]["ci.yml"]
                if kind == "runs":
                    original = block["run"]
                    runs = []
                    for index in range(count):
                        run = copy.deepcopy(original)
                        run.update(id=original["id"] + index, run_number=original["run_number"] + index)
                        activity = datetime(2026, 10, 14, 16, 0, tzinfo=timezone.utc) + timedelta(seconds=index)
                        run["updated_at"] = activity.isoformat().replace("+00:00", "Z")
                        runs.append(run)
                    block["runs"] = {"total_count": count, "workflow_runs": runs}
                    block["run"] = copy.deepcopy(runs[-1])
                    block["jobs"]["jobs"][0]["run_id"] = runs[-1]["id"]
                else:
                    original = block["jobs"]["jobs"][0]
                    jobs = [copy.deepcopy(original)]
                    for index in range(1, count):
                        job = copy.deepcopy(original)
                        job.update(id=original["id"] + index, name="unrelated-" + str(index))
                        jobs.append(job)
                    block["jobs"] = {"total_count": count, "jobs": jobs}
                if count == 100:
                    self.assertEqual(self.check(current)["status"], "PASS")
                else:
                    self.refuse(current)

    def test_wait_requires_the_same_actual_named_live_job(self):
        for name in FILES:
            for status in ("queued", "in_progress"):
                live = live_snapshot(name, status)
                self.assertEqual(self.check(live)["status"], "WAIT")
                for target in ("run", "job"):
                    current = copy.deepcopy(live)
                    block = current["workflows"][name]
                    part = block["run"] if target == "run" else block["jobs"]["jobs"][0]
                    part["conclusion"] = "success"
                    self.refuse(current)
                    current = copy.deepcopy(live)
                    block = current["workflows"][name]
                    part = block["run"] if target == "run" else block["jobs"]["jobs"][0]
                    part.update(status="completed", conclusion="success")
                    self.refuse(current)
                current = copy.deepcopy(live)
                current["workflows"][name]["jobs"] = {"total_count": 0, "jobs": []}
                self.refuse(current)
                current = copy.deepcopy(live)
                current["workflows"][name]["jobs"]["jobs"][0]["steps"][0].update(status="completed", conclusion="failure")
                self.refuse(current)

    def test_malformed_missing_nested_values_raise_gateerror(self):
        self.refuse(None)
        for key in ("repo", "tag_ref", "tag_objects", "main", "compare", "workflows"):
            current = snapshot()
            current[key] = None
            self.refuse(current)
        for key in ("definition", "runs", "run", "jobs"):
            current = snapshot()
            current["workflows"]["ci.yml"][key] = None
            self.refuse(current)
        for key in ("jobs", "steps"):
            current = snapshot()
            if key == "jobs":
                current["workflows"]["ci.yml"]["jobs"]["jobs"] = [None]
            else:
                current["workflows"]["ci.yml"]["jobs"]["jobs"][0]["steps"] = [None]
            self.refuse(current)

    def test_collector_uses_unfiltered_exact_head_lists_and_current_attempt_jobs(self):
        value = snapshot()
        replies = [value[k] for k in ("repo", "tag_ref", "main", "compare")]
        for name in FILES:
            replies.extend(value["workflows"][name][key] for key in ("definition", "runs", "run", "jobs"))
        responses = [SimpleNamespace(stdout=json.dumps(reply).encode()) for reply in replies]
        reads = []
        with patch.object(GATE.subprocess, "run", side_effect=responses) as run, patch.object(GATE.time, "monotonic", return_value=100):
            observed = GATE.collect(REPOSITORY, SHA, TAG, 400, reads)
        self.assertEqual(observed, value)
        self.assertEqual(run.call_count, 12)
        self.assertEqual(len(reads), 12)
        for call in run.call_args_list:
            args = call.args[0]
            self.assertEqual(args[:4], ["gh", "api", "--hostname", "github.com"])
            self.assertTrue(args[4].startswith("repos/" + REPOSITORY))
            self.assertNotIn("--method", args)
            self.assertTrue(call.kwargs["check"])
            self.assertLessEqual(call.kwargs["timeout"], 30)
        endpoints = [call.args[0][-1] for call in run.call_args_list]
        for name in FILES:
            endpoint = next(e for e in endpoints if "/workflows/" + name + "/runs?" in e)
            self.assertEqual(parse_qs(urlsplit(endpoint).query), {
                "head_sha": [SHA], "event": ["push"], "branch": ["main"], "per_page": ["100"]})
            self.assertNotIn("status", parse_qs(urlsplit(endpoint).query))
            self.assertIn("repos/" + REPOSITORY + "/actions/runs/" + str(value["workflows"][name]["run"]["id"]) + "/attempts/1/jobs?per_page=100", endpoints)

    def test_collector_errors_size_and_deadline_fail_without_polling(self):
        for error in (subprocess.CalledProcessError(1, "gh"), subprocess.TimeoutExpired("gh", 30)):
            with patch.object(GATE.subprocess, "run", side_effect=error) as run, patch.object(GATE.time, "monotonic", return_value=100):
                with self.assertRaises(GATE.GateError):
                    GATE.collect(REPOSITORY, SHA, TAG, 400, [])
                self.assertEqual(run.call_count, 1)
        for output in (b"not json", b" " * (2 * 1024 * 1024 + 1)):
            with patch.object(GATE.subprocess, "run", return_value=SimpleNamespace(stdout=output)) as run, patch.object(GATE.time, "monotonic", return_value=100):
                with self.assertRaises(GATE.GateError):
                    GATE.collect(REPOSITORY, SHA, TAG, 400, [])
                self.assertEqual(run.call_count, 1)
        with patch.object(GATE.subprocess, "run") as run, patch.object(GATE.time, "monotonic", return_value=400):
            with self.assertRaises(GATE.GateError):
                GATE.collect(REPOSITORY, SHA, TAG, 400, [])
            run.assert_not_called()

    def invoke_cli(self, observations, heads=(SHA, SHA), times=None):
        """Simulate read-only CLI observations; no GitHub/git commands or waits."""
        with tempfile.TemporaryDirectory() as temporary:
            out = Path(temporary) / "receipt.json"
            argv = ["check-release-validation.py", "--repository", REPOSITORY,
                    "--repository-id", str(REPOSITORY_ID), "--sha", SHA, "--tag", TAG, "--out", str(out)]
            with patch.object(GATE.sys, "argv", argv), patch.object(GATE, "collect", side_effect=observations) as collect, \
                    patch.object(GATE.subprocess, "check_output", side_effect=heads) as git, \
                    patch.object(GATE.time, "monotonic", side_effect=times, return_value=100), \
                    patch.object(GATE.time, "sleep") as sleep, contextlib.redirect_stdout(io.StringIO()):
                code = GATE.main()
            return code, json.loads(out.read_text()), collect.call_count, git.call_count, sleep.call_count

    def test_cli_requires_two_unchanged_reads_and_checkout(self):
        good = snapshot()
        code, report, reads, heads, sleeps = self.invoke_cli([good, copy.deepcopy(good)])
        self.assertEqual((code, report["status"], reads, heads, sleeps), (0, "PASS", 2, 2, 0))
        self.assertFalse(report["publication_atomicity"])
        changed_attempt = snapshot()
        block = changed_attempt["workflows"]["ci.yml"]
        block["run"]["run_attempt"] = block["runs"]["workflow_runs"][0]["run_attempt"] = block["jobs"]["jobs"][0]["run_attempt"] = 2
        self.assertEqual(self.check(changed_attempt)["status"], "PASS")
        changed_tag = snapshot()
        changed_tag["tag_ref"]["object"]["sha"] = OTHER_SHA
        changed_job = snapshot()
        changed_job["workflows"]["ci.yml"]["jobs"]["jobs"][0]["id"] += 1
        added_step = snapshot()
        added_step["workflows"]["ci.yml"]["jobs"]["jobs"][0]["steps"].append({
            "name": "New validation", "status": "completed", "conclusion": "success"})
        self.assertEqual(self.check(changed_job)["status"], "PASS")
        self.assertEqual(self.check(added_step)["status"], "PASS")
        for final in (changed_attempt, changed_tag, changed_job, added_step):
            code, report, reads, _, sleeps = self.invoke_cli([good, final])
            self.assertEqual((code, report["status"], reads, sleeps), (1, "REFUSED", 2, 0))
        code, report, _, heads, _ = self.invoke_cli([good, good], heads=(SHA, OTHER_SHA))
        self.assertEqual((code, report["status"], heads), (1, "REFUSED", 2))

    def test_cli_ignores_unconsumed_metadata_but_retains_raw_audit_hash(self):
        first = snapshot()
        final = snapshot()
        final["repo"].update(stargazers_count=123, owner={"login": "display-only"})
        final["tag_ref"]["node_id"] = "display-only"
        final["workflows"]["ci.yml"]["definition"]["name"] = "Display name"
        final["workflows"]["ci.yml"]["run"]["display_title"] = "Display title"
        final["workflows"]["ci.yml"]["jobs"]["jobs"][0]["runner_name"] = "Display runner"
        final["workflows"]["ci.yml"]["jobs"]["jobs"][0]["steps"][0]["started_at"] = "2026-10-14T14:20:08Z"
        self.assertNotEqual(GATE.fingerprint(first), GATE.fingerprint(final))
        code, report, reads, _, sleeps = self.invoke_cli([first, final])
        self.assertEqual((code, report["status"], reads, sleeps), (0, "PASS", 2, 0))
        self.assertNotEqual(report["observations"][0]["snapshot_sha256"], report["final_snapshot_sha256"])
        # Synthetic annotated tag metadata is likewise not an executable readiness input.
        annotated = snapshot()
        commit = copy.deepcopy(annotated["tag_ref"]["object"])
        annotated["tag_ref"]["object"] = {"sha": OTHER_SHA, "type": "tag", "url": PREFIX + "/git/tags/" + OTHER_SHA}
        annotated["tag_objects"] = [{"sha": OTHER_SHA, "url": PREFIX + "/git/tags/" + OTHER_SHA,
                                     "object": commit, "tagger": {"name": "Display name"}}]
        final = copy.deepcopy(annotated)
        final["tag_objects"][0]["tagger"]["name"] = "Updated display name"
        code, report, reads, _, _ = self.invoke_cli([annotated, final])
        self.assertEqual((code, report["status"], reads), (0, "PASS", 2))

    def test_cli_allows_only_valid_forward_main_movement_during_reread(self):
        first = snapshot()
        final = snapshot()
        final["main"]["commit"]["sha"] = OTHER_SHA
        final["compare"].update(status="ahead", ahead_by=1, url=PREFIX + "/compare/" + SHA + "..." + OTHER_SHA)
        code, report, reads, _, _ = self.invoke_cli([first, final])
        self.assertEqual((code, report["status"], reads, report["main_sha"]), (0, "PASS", 2, OTHER_SHA))
        final["compare"]["merge_base_commit"]["sha"] = OTHER_SHA
        code, report, reads, _, _ = self.invoke_cli([first, final])
        self.assertEqual((code, report["status"], reads), (1, "REFUSED", 2))

    def test_cli_final_success_after_deadline_is_refused(self):
        good = snapshot()
        code, report, reads, _, sleeps = self.invoke_cli([good, good], times=(100, 401))
        self.assertEqual((code, report["status"], reads, sleeps), (1, "REFUSED", 2, 0))
        self.assertIn("deadline", report["reason"])

    def test_cli_wait_is_bounded_and_missing_or_failed_proof_never_waits(self):
        good = snapshot()
        code, report, reads, _, sleeps = self.invoke_cli([live_snapshot(), good, good])
        self.assertEqual((code, report["status"], reads, sleeps), (0, "PASS", 3, 1))
        for current in (snapshot(), live_snapshot()):
            block = current["workflows"]["ci.yml"]
            block["jobs"] = {"total_count": 0, "jobs": []}
            code, report, reads, _, sleeps = self.invoke_cli([current])
            self.assertEqual((code, report["status"], reads, sleeps), (1, "REFUSED", 1, 0))
        failed = snapshot()
        failed["workflows"]["ci.yml"]["run"]["conclusion"] = "failure"
        code, report, reads, _, sleeps = self.invoke_cli([failed])
        self.assertEqual((code, report["status"], reads, sleeps), (1, "REFUSED", 1, 0))
        code, report, reads, _, sleeps = self.invoke_cli([live_snapshot()], times=(100, 391))
        self.assertEqual((code, report["status"], reads, sleeps), (1, "REFUSED", 1, 0))
        self.assertIn("five minutes", report["reason"])


if __name__ == "__main__":
    unittest.main()
