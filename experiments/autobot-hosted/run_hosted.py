#!/usr/bin/env python3
"""One bounded, disposable AutoBot Docker diagnostic; no real credentials."""
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time

REVISION = "5c224241326d909219a1f525cbe88a9d65eedff4"
SERVICES = {"autobot-backend", "autobot-redis", "autobot-postgres",
            "autobot-chromadb", "autobot-secrets-init"}
ROOT = Path(sys.argv[1]).resolve()
OUT = Path(sys.argv[2]).resolve()
OUT.mkdir(parents=True, exist_ok=True)
REPORT = {"schema": "gooo/autobot-disposable-diagnostic/v1",
          "upstream_revision": REVISION, "steps": [],
          "scope": "Synthetic real HTTP/DB lifecycle; no Gooo adoption, model journey or policy-history claim"}
ENV = {k: os.environ[k] for k in ("PATH", "HOME", "LANG", "TMPDIR") if k in os.environ}
ENV.update({"COMPOSE_ANSI": "never", "BUILDKIT_PROGRESS": "plain"})
SECRETS = []
COMPOSE = []
DEADLINE = time.monotonic() + 1200  # Reserve the remaining job time for cleanup/upload.


class Failure(Exception):
    pass


def save():
    (OUT / "result.json").write_text(json.dumps(REPORT, indent=2) + "\n")


def redact(data):
    text = data.decode("utf-8", errors="replace")
    for value in SECRETS:
        text = text.replace(value, "[REDACTED]")
    text = re.sub(r"eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+", "[JWT REDACTED]", text)
    # Runtime logs are untrusted. Suppress entire credential-related lines.
    return "\n".join("[credential-related line omitted]" if re.search(
        r"password|secret|bearer|authorization|access_token|session_id|api.key", line, re.I)
        else line for line in text.splitlines())


def command(stage, args, seconds, allow_failure=False, expose_tail=True):
    started = time.monotonic()
    if stage not in ("container_status", "service_logs", "cleanup", "source_status_after"):
        seconds = min(seconds, max(0, DEADLINE - started))
        if seconds <= 0:
            raise Failure("overall_deadline")
    with tempfile.TemporaryFile() as capture:
        proc = subprocess.Popen(args, cwd=ROOT, env=ENV, stdout=capture,
                                stderr=subprocess.STDOUT, start_new_session=True)
        reason = None
        while proc.poll() is None:
            if time.monotonic() - started > seconds:
                reason = "deadline"
            if os.fstat(capture.fileno()).st_size > 32 * 1024 * 1024:
                reason = "output_limit"
            if reason:
                os.killpg(proc.pid, signal.SIGKILL)
                break
            time.sleep(0.25)
        code = proc.wait()
        if os.fstat(capture.fileno()).st_size > 32 * 1024 * 1024:
            reason = "output_limit"
        capture.seek(0)
        data = capture.read(32 * 1024 * 1024 + 1)
    step = {"stage": stage, "exit_code": code, "stop_reason": reason,
            "seconds": round(time.monotonic() - started, 3),
            "captured_bytes": len(data)}
    REPORT["steps"].append(step)
    if expose_tail:
        # If capture stopped mid-stream, omit the unfinished final line before redaction.
        public_data = data[:data.rfind(b"\n") + 1] if reason else data
        (OUT / (stage + ".log")).write_text(redact(public_data)[-24000:] + "\n")
    save()
    print(json.dumps(step), flush=True)
    if (code != 0 or reason) and not allow_failure:
        raise Failure(stage)
    return data, code


def require(condition, label):
    if not condition:
        raise Failure(label)


def main():
    global COMPOSE
    require(os.environ.get("GITHUB_ACTIONS") == "true", "hosted_runner_required")
    observed, _ = command("revision", ["git", "rev-parse", "HEAD"], 10)
    require(observed.decode().strip() == REVISION, "revision_mismatch")
    status, _ = command("source_status", ["git", "status", "--porcelain"], 10)
    require(not status.strip(), "source_not_clean")
    bind_source = ROOT / "docker/postgres/init-databases.sql"
    REPORT["postgres_init_sha256_before"] = hashlib.sha256(bind_source.read_bytes()).hexdigest()
    REPORT["resources_before"] = {"cpu_count": os.cpu_count(),
        "memory_bytes": os.sysconf("SC_PAGE_SIZE") * os.sysconf("SC_PHYS_PAGES"),
        "disk_free_bytes": shutil.disk_usage(ROOT).free}
    require(REPORT["resources_before"]["disk_free_bytes"] >= 10 * 1024**3, "disk_preflight_under_10GiB")
    command("docker_version", ["docker", "version"], 20)
    command("compose_version", ["docker", "compose", "version"], 20)
    command("generate", ["bash", "docker/generate-secrets.sh"], 60, expose_tail=False)
    secret_file = ROOT / "docker/.env.secrets"
    for line in secret_file.read_text().splitlines():
        if "=" in line and not line.startswith("#"):
            key, value = line.split("=", 1)
            if key != "AUTOBOT_DOCKER_MGMT_PREFIX" and value:
                SECRETS.append(value)
    require(len(SECRETS) >= 6, "generated_secrets_incomplete")
    # Restrict published ports only; application auth and original dependencies remain unchanged.
    override = OUT.parent / "compose-no-published-ports.yml"
    override.write_text("services:\n" + "".join(
        "  " + service + ":\n    ports: !reset []\n" for service in sorted(SERVICES)))
    COMPOSE = ["docker", "compose", "--project-name", "gooo-autobot-smoke",
               "--env-file", "docker/.env.docker", "--env-file", "docker/.env.secrets",
               "-f", "docker-compose.yml", "-f", str(override)]
    raw, _ = command("compose_config", COMPOSE + ["config", "--format", "json"], 30, expose_tail=False)
    config = json.loads(raw)
    pending, seen = ["autobot-backend"], set()
    while pending:
        name = pending.pop()
        if name in seen:
            continue
        seen.add(name)
        service = config["services"][name]
        pending.extend(service.get("depends_on", {}))
        require(not service.get("ports"), "unexpected_published_port")
        require(not service.get("privileged") and service.get("network_mode") != "host",
                "unexpected_host_privilege")
        require("docker.sock" not in json.dumps(service.get("volumes", [])), "unexpected_docker_socket")
    require(seen == SERVICES, "dependency_closure_changed")
    env = config["services"]["autobot-backend"]["environment"]
    require(env.get("AUTOBOT_USER_MODE") == "single_company", "unexpected_user_mode")
    require(env.get("AUTOBOT_LLM_PROVIDER") == "ollama", "unexpected_provider")
    require(not env.get("GH_TOKEN") and not env.get("ANTHROPIC_API_KEY"), "provider_credentials_present")
    require(bool(env.get("AUTOBOT_ADMIN_PASSWORD")), "admin_password_missing")
    REPORT["selected_services"] = sorted(seen)
    REPORT["published_ports"] = 0
    REPORT["compose_config_sha256"] = hashlib.sha256(raw).hexdigest()
    save()
    command("build", COMPOSE + ["build", "autobot-backend"], 720)
    command("start_health", COMPOSE + ["up", "--no-build", "-d", "--wait",
            "--wait-timeout", "300", "autobot-backend"], 330)
    # HTTP health precedes the background admin seed; do not mislabel that gap a login regression.
    ready_deadline = time.monotonic() + 180
    ready = False
    while time.monotonic() < ready_deadline:
        raw, _ = command("seed_readiness", COMPOSE + ["logs", "--no-color", "autobot-backend"],
                         10, expose_tail=False)
        text = raw.decode("utf-8", errors="replace")
        require("Default admin seeding failed (non-critical):" not in text, "admin_seed_failed")
        if "Admin seed: created default admin user 'admin'" in text and \
                "PHASE 2 COMPLETE: All background services initialized" in text:
            ready = True
            break
        time.sleep(10)
    require(ready, "READINESS_UNESTABLISHED")
    command("copy_client", ["docker", "cp", str(Path(__file__).with_name("http_smoke.py")),
            "autobot-backend:/tmp/gooo-http-smoke.py"], 20)
    raw, code = command("http_lifecycle", ["docker", "exec", "autobot-backend",
            "python3", "/tmp/gooo-http-smoke.py"], 100, allow_failure=True, expose_tail=False)
    # The client emits only an explicit non-secret result schema.
    result = json.loads(raw)
    allowed = {"schema", "upstream_revision", "scope", "model_execution", "terminal_execution",
               "historical_policy_reconstruction", "gooo_adoption_claim", "status", "stage", "code",
               "http_status", "completed_lifecycles", "http_events"}
    require(isinstance(result, dict) and set(result) <= allowed, "unexpected_client_result_fields")
    require(result.get("schema") == "gooo/autobot-no-action-http-smoke/v1" and
            result.get("upstream_revision") == REVISION, "client_result_identity")
    for field in ("model_execution", "terminal_execution", "historical_policy_reconstruction", "gooo_adoption_claim"):
        require(result.get(field) is False, "client_scope_changed")
    # Accept only literal diagnostic codes from the locally shipped client, never response text.
    codes = set(re.findall(r'"([A-Z][A-Z_]{2,})"', Path(__file__).with_name("http_smoke.py").read_text()))
    require(result.get("scope") == "synthetic_login_and_two_no_action_database_lifecycles",
            "client_scope_changed")
    require(result.get("status") in ("PASS", "FAIL") and result.get("code") in codes,
            "client_nonstatic_text")
    require(type(result.get("completed_lifecycles")) is int and
            0 <= result["completed_lifecycles"] <= 2, "client_count_type")
    require("http_status" not in result or (type(result["http_status"]) is int and
            100 <= result["http_status"] <= 599), "client_http_status_type")
    events = result.get("http_events")
    require(isinstance(events, list) and len(events) <= 7, "client_event_count")
    expected = [("login", 200), ("approve_create", 201), ("approve_decision", 200),
                ("approve_readback", 200), ("reject_create", 201), ("reject_decision", 200),
                ("reject_readback", 200)]
    require(result.get("stage") in {"configuration", "complete"} | {x[0] for x in expected},
            "client_stage_changed")
    require(events == [{"stage": name, "status": "HTTP_OK", "http_status": status}
                       for name, status in expected[:len(events)]], "client_events_changed")
    (OUT / "http-lifecycle.json").write_text(json.dumps(result, indent=2) + "\n")
    require(code == 0 and result.get("status") == "PASS" and
            result.get("completed_lifecycles") == 2 and len(events) == 7 and
            result.get("stage") == "complete" and result.get("code") == "TWO_STORED_LIFECYCLES_VERIFIED" and
            "http_status" not in result, "http_lifecycle_failed")
    REPORT["status"] = "HTTP_LIFECYCLE_COMPLETED"


try:
    main()
except Failure as exc:
    REPORT["status"] = "STOPPED_AT_FIRST_FAILURE"
    REPORT["first_failure"] = str(exc)
except Exception as exc:
    REPORT["status"] = "DIAGNOSTIC_HARNESS_FAILURE"
    REPORT["exception_type"] = type(exc).__name__
finally:
    if COMPOSE:
        for stage, args, seconds in (
                ("container_status", COMPOSE + ["ps", "--all", "--format", "json"], 20),
                ("service_logs", COMPOSE + ["logs", "--no-color", "--tail", "150"] + sorted(SERVICES), 25),
                ("cleanup", COMPOSE + ["down", "--volumes", "--remove-orphans", "--timeout", "10"], 60)):
            try:
                _, code = command(stage, args, seconds, True)
                if stage == "cleanup":
                    REPORT["cleanup_succeeded"] = code == 0 and REPORT["steps"][-1]["stop_reason"] is None
            except Exception as exc:
                REPORT[stage + "_exception_type"] = type(exc).__name__
    if COMPOSE and not REPORT.get("cleanup_succeeded"):
        REPORT["cleanup_status"] = "CLEANUP_UNESTABLISHED"
    bind_source = ROOT / "docker/postgres/init-databases.sql"
    if bind_source.exists():
        REPORT["postgres_init_sha256_after"] = hashlib.sha256(bind_source.read_bytes()).hexdigest()
        REPORT["postgres_init_unchanged"] = REPORT["postgres_init_sha256_after"] == REPORT.get("postgres_init_sha256_before")
    try:
        raw, code = command("source_status_after", ["git", "status", "--porcelain", "--untracked-files=no"], 10, True)
        REPORT["tracked_source_unchanged"] = code == 0 and not raw.strip() and REPORT.get("postgres_init_unchanged") is True
    except Exception as exc:
        REPORT["tracked_source_unchanged"] = False
        REPORT["source_status_exception_type"] = type(exc).__name__
    if REPORT.get("status") == "HTTP_LIFECYCLE_COMPLETED" and not REPORT.get("tracked_source_unchanged"):
        REPORT["status"] = "SOURCE_POSTCONDITION_FAILED"
    REPORT["disk_free_bytes_after"] = shutil.disk_usage(ROOT).free
    save()
    print(json.dumps({"status": REPORT["status"], "first_failure": REPORT.get("first_failure")}), flush=True)
sys.exit(0 if REPORT.get("status") == "HTTP_LIFECYCLE_COMPLETED" and REPORT.get("cleanup_succeeded")
         and REPORT.get("tracked_source_unchanged") else 1)
