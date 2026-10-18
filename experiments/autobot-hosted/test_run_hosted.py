"""Owned-fixture/fake-Docker tests; no Docker daemon or external service."""

import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


REVISION = "5c224241326d909219a1f525cbe88a9d65eedff4"
ORCHESTRATOR_BYTES = Path(__file__).with_name("run_hosted.py").read_bytes()
CLIENT_BYTES = Path(__file__).with_name("http_smoke.py").read_bytes()
ORCHESTRATOR_SHA256 = hashlib.sha256(ORCHESTRATOR_BYTES).hexdigest()
CLIENT_SHA256 = hashlib.sha256(CLIENT_BYTES).hexdigest()
SECRET_VALUES = ["FAKEONLYSENTINELVALUE" + str(i) for i in range(6)]
EVENTS = [{"stage": stage, "status": "HTTP_OK", "http_status": status} for stage, status in (
    ("login", 200), ("approve_create", 201), ("approve_decision", 200),
    ("approve_readback", 200), ("reject_create", 201), ("reject_decision", 200),
    ("reject_readback", 200),
)]
VALID = {
    "schema": "gooo/autobot-no-action-http-smoke/v1",
    "upstream_revision": REVISION,
    "scope": "synthetic_login_and_two_no_action_database_lifecycles",
    "model_execution": False,
    "terminal_execution": False,
    "historical_policy_reconstruction": False,
    "gooo_adoption_claim": False,
    "status": "PASS",
    "stage": "complete",
    "code": "TWO_STORED_LIFECYCLES_VERIFIED",
    "completed_lifecycles": 2,
    "http_events": EVENTS,
}

FAKE_TOOL = r'''
import hashlib,json,sys
from pathlib import Path
root=Path.cwd()
scenario=json.loads((root/'scenario.json').read_text())
args=sys.argv[1:]
name=Path(sys.argv[0]).name
with (root/'calls.jsonl').open('a') as f:
 f.write(json.dumps([name]+args)+'\n')
if name=='git':
 if args==['rev-parse','HEAD']: print(scenario['revision'])
 sys.exit(0)
if name=='bash':
 (root/'docker/.env.secrets').write_text('\n'.join('KEY'+str(i)+'='+s for i,s in enumerate(scenario['secrets']))+'\n')
 print('password '+scenario['secrets'][0])
 sys.exit(0)
if args[:1]==['version'] or args==['compose','version']:
 print('fake docker version')
elif args[:1]==['cp']:
 print('obsolete copy delivery refused')
 sys.exit(45)
elif args[:1]==['exec']:
 received=sys.stdin.buffer.read(256*1024+1)
 if args!=['exec','-i','autobot-backend','python3','-'] or hashlib.sha256(received).hexdigest()!=scenario['expected_client_sha256'] or len(received)!=scenario['expected_client_bytes']:
  print('fake input delivery failure '+scenario['secrets'][0])
  sys.exit(2)
 if scenario.get('non_json_client'):
  print('fake interpreter failure '+scenario['secrets'][0])
  sys.exit(2)
 if scenario.get('overoutput_client'):
  sys.stdout.buffer.write(json.dumps(scenario['client']).encode()+b' '*(33*1024*1024))
  sys.stdout.flush()
  sys.exit(0)
 print(json.dumps(scenario['client']))
 sys.exit(scenario.get('client_exit',0))
elif 'config' in args:
 services={k:{} for k in ['autobot-backend','autobot-redis','autobot-postgres','autobot-chromadb','autobot-secrets-init']}
 services['autobot-backend']={'depends_on':{k:{} for k in services if k!='autobot-backend'},'environment':{'AUTOBOT_USER_MODE':'single_company','AUTOBOT_LLM_PROVIDER':'ollama','AUTOBOT_ADMIN_PASSWORD':scenario['secrets'][0]}}
 print(json.dumps({'services':services}))
elif 'build' in args:
 if scenario.get('overoutput_build'):
  sys.stdout.buffer.write(b'X'*(33*1024*1024)+b'FAKEONLYSENTINEL')
  sys.stdout.flush()
  sys.exit(0)
 for value in scenario['secrets']: print('opaque '+value)
 print('authorization: Bearer something')
 print('eyJabc.eyJdef.signature')
 if scenario.get('build_failure'): sys.exit(42)
elif 'up' in args:
 print('healthy')
elif 'logs' in args:
 print("Admin seed: created default admin user 'admin'")
 print('PHASE 2 COMPLETE: All background services initialized')
 print('password '+scenario['secrets'][1])
elif 'ps' in args:
 print('[]')
elif 'down' in args:
 print('cleanup')
 if scenario.get('cleanup_failure'): sys.exit(43)
else:
 sys.exit(44)
'''


class HostedTests(unittest.TestCase):
    def run_fixture(self, **changes):
        with tempfile.TemporaryDirectory(prefix="gooo-fake-hosted-") as temporary:
            parent = Path(temporary)
            root = parent / "upstream"
            out = parent / "out"
            bindir = parent / "bin"
            harness = parent / "harness"
            (root / "docker/postgres").mkdir(parents=True)
            bindir.mkdir()
            harness.mkdir()
            runner_bytes = ORCHESTRATOR_BYTES
            if changes.pop("drop_client_stdin", False):
                # Controlled regression in this owned fixture only: the fake
                # Docker process must fail instead of returning a canned PASS.
                self.assertEqual(runner_bytes.count(b"stdin=client_input"), 1)
                runner_bytes = runner_bytes.replace(b"stdin=client_input", b"stdin=None", 1)
            (harness / "run_hosted.py").write_bytes(runner_bytes)
            (harness / "http_smoke.py").write_bytes(CLIENT_BYTES)
            (root / "docker/postgres/init-databases.sql").write_text("-- immutable fixture\n")
            (root / "docker/.env.docker").write_text("# fixture\n")
            scenario = {"revision": REVISION, "secrets": SECRET_VALUES, "client": copy.deepcopy(VALID),
                        "expected_client_sha256": CLIENT_SHA256, "expected_client_bytes": len(CLIENT_BYTES)}
            scenario.update(changes)
            (root / "scenario.json").write_text(json.dumps(scenario))
            for tool in ("git", "docker", "bash"):
                path = bindir / tool
                path.write_text("#!" + sys.executable + "\n" + FAKE_TOOL)
                path.chmod(0o700)
            env = {"PATH": str(bindir) + os.pathsep + os.environ.get("PATH", ""),
                   "HOME": str(parent), "LANG": "C", "GITHUB_ACTIONS": "true"}
            completed = subprocess.run(
                [sys.executable, "-B", str(harness / "run_hosted.py"), str(root), str(out)],
                env=env, capture_output=True, timeout=15,
            )
            result = json.loads((out / "result.json").read_text())
            calls = [json.loads(line) for line in (root / "calls.jsonl").read_text().splitlines()]
            artifacts = {p.name: p.read_text() for p in out.iterdir() if p.is_file()}
            public = completed.stdout.decode() + completed.stderr.decode() + "".join(artifacts.values())
            leaked = any(secret in public for secret in SECRET_VALUES)
            return completed.returncode, result, calls, artifacts, leaked

    def test_good_lifecycle_and_redacted_logs_cleanup(self):
        code, result, calls, artifacts, leaked = self.run_fixture()
        self.assertEqual(code, 0)
        self.assertEqual(result["status"], "HTTP_LIFECYCLE_COMPLETED")
        self.assertTrue(result["cleanup_succeeded"])
        self.assertTrue(result["postgres_init_unchanged"])
        self.assertFalse(leaked, "Fake sentinel leaked into public output")
        self.assertTrue(any("down" in call for call in calls))
        self.assertIn(["docker", "exec", "-i", "autobot-backend", "python3", "-"], calls)
        self.assertFalse(any(call[:2] == ["docker", "cp"] for call in calls))
        self.assertEqual(result["http_client_input"], {"transport": "docker_exec_stdin",
            "bytes": len(CLIENT_BYTES), "sha256": CLIENT_SHA256})
        self.assertIn("[REDACTED]", artifacts["build.log"])
        self.assertNotIn("eyJabc", artifacts["build.log"])

    def test_build_failure_stops_before_start_and_still_cleans(self):
        code, result, calls, artifacts, leaked = self.run_fixture(build_failure=True)
        self.assertEqual(code, 1)
        self.assertEqual(result["first_failure"], "build")
        self.assertTrue(result["cleanup_succeeded"])
        self.assertFalse(any("up" in call or "exec" in call for call in calls))
        self.assertTrue(any("down" in call for call in calls))
        self.assertFalse(leaked, "Fake sentinel leaked into failure output")

    def test_cleanup_failure_makes_successful_lifecycle_nonzero(self):
        code, result, calls, artifacts, leaked = self.run_fixture(cleanup_failure=True)
        self.assertEqual(code, 1)
        self.assertFalse(result["cleanup_succeeded"])
        self.assertEqual(result["cleanup_status"], "CLEANUP_UNESTABLISHED")
        self.assertTrue(any("down" in call for call in calls))
        self.assertFalse(leaked)

    def test_malformed_client_untyped_http_status_cannot_leak_or_pass(self):
        client = copy.deepcopy(VALID)
        client["http_status"] = {"detail": SECRET_VALUES[0]}
        code, result, calls, artifacts, leaked = self.run_fixture(client=client)
        self.assertNotEqual(code, 0, "Malformed client result was accepted")
        self.assertFalse(leaked, "Unvalidated nested client value leaked")
        self.assertTrue(result["cleanup_succeeded"])

    def test_arbitrary_client_code_cannot_leak_or_pass(self):
        client = copy.deepcopy(VALID)
        client["code"] = SECRET_VALUES[0]
        code, result, calls, artifacts, leaked = self.run_fixture(client=client)
        self.assertNotEqual(code, 0, "Arbitrary regex-shaped code was accepted")
        self.assertFalse(leaked, "Arbitrary client code leaked")
        self.assertTrue(result["cleanup_succeeded"])

    def test_finished_overoutput_without_newline_is_empty_public_log(self):
        code, result, calls, artifacts, leaked = self.run_fixture(overoutput_build=True)
        self.assertEqual(code, 1)
        self.assertEqual(result["first_failure"], "build")
        build = next(step for step in result["steps"] if step["stage"] == "build")
        self.assertEqual(build["stop_reason"], "output_limit")
        self.assertEqual(artifacts["build.log"].strip(), "")
        self.assertFalse(any("up" in call or "exec" in call for call in calls))
        self.assertTrue(result["cleanup_succeeded"])
        self.assertFalse(leaked)

    def test_missing_actual_stdin_delivery_is_refused(self):
        code, result, calls, artifacts, leaked = self.run_fixture(drop_client_stdin=True)
        self.assertEqual(code, 1)
        self.assertEqual(result["first_failure"], "http_client_result_not_json")
        self.assertEqual(result["http_client_launch_failure"]["exit_code"], 2)
        self.assertEqual(result["http_client_launch_failure"]["classification"], "CLIENT_RESULT_NOT_JSON")
        self.assertFalse(result["http_client_launch_failure"]["raw_output_retained"])
        self.assertNotIn("http-lifecycle.json", artifacts)
        self.assertNotIn("http_lifecycle.log", artifacts)
        self.assertTrue(result["cleanup_succeeded"])
        self.assertFalse(leaked, "Fake failed delivery output leaked")

    def test_non_json_interpreter_failure_preserves_only_safe_metadata(self):
        code, result, calls, artifacts, leaked = self.run_fixture(non_json_client=True)
        self.assertEqual(code, 1)
        self.assertEqual(result["first_failure"], "http_client_result_not_json")
        failure = result["http_client_launch_failure"]
        self.assertEqual(failure["exit_code"], 2)
        self.assertEqual(failure["classification"], "CLIENT_RESULT_NOT_JSON")
        self.assertGreater(failure["captured_bytes"], 0)
        self.assertIsNone(failure["stop_reason"])
        self.assertFalse(failure["raw_output_retained"])
        self.assertNotIn("http-lifecycle.json", artifacts)
        self.assertTrue(result["cleanup_succeeded"])
        self.assertFalse(leaked, "Fake interpreter failure body leaked")

    def test_exit_zero_valid_json_prefix_with_overoutput_cannot_pass(self):
        code, result, calls, artifacts, leaked = self.run_fixture(overoutput_client=True)
        self.assertEqual(code, 1)
        self.assertEqual(result["first_failure"], "http_client_capture_incomplete")
        failure = result["http_client_launch_failure"]
        self.assertEqual(failure["exit_code"], 0)
        self.assertEqual(failure["classification"], "CLIENT_CAPTURE_INCOMPLETE")
        self.assertEqual(failure["stop_reason"], "output_limit")
        self.assertFalse(failure["raw_output_retained"])
        self.assertNotIn("http-lifecycle.json", artifacts)
        self.assertTrue(result["cleanup_succeeded"])
        self.assertFalse(leaked)


if __name__ == "__main__":
    unittest.main()
