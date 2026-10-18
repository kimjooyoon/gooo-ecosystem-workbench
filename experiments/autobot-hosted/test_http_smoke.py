"""Local loopback mock tests only; these are not an AutoBot integration result."""

import contextlib
import copy
import http.server
import io
import json
import threading
import time
import unittest
import uuid

import http_smoke as smoke


PASSWORD = "FAKE_PASSWORD_SENTINEL_NOT_A_CREDENTIAL"
TOKEN = "FAKE_TOKEN_SENTINEL_NOT_A_CREDENTIAL"
SESSION = "FAKE_SESSION_SENTINEL_NOT_A_CREDENTIAL"
ENV = {"AUTOBOT_ADMIN_PASSWORD": PASSWORD, "AUTOBOT_ADMIN_USERNAME": "admin"}


class MockServer(http.server.ThreadingHTTPServer):
    daemon_threads = True

    def handle_error(self, request, client_address):
        # Broken pipes on deliberately bounded reads/deadlines are expected.
        pass


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        pass

    def do_GET(self):
        self.handle_call()

    def do_POST(self):
        self.handle_call()

    def handle_call(self):
        scenario = self.server.scenario
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        body = json.loads(raw) if raw else None
        scenario.calls.append((self.command, self.path, body, self.headers.get("Authorization")))
        index = len(scenario.calls)
        if scenario.mutation == "redirect" and index == 1:
            self.send_response(302)
            self.send_header("Location", "https://example.invalid/" + TOKEN)
            self.end_headers()
            return
        if scenario.mutation == "login401" and index == 1:
            self.reply(401, {"detail": PASSWORD + TOKEN})
            return
        if scenario.mutation == "decision403" and index == 3:
            self.reply(403, {"detail": PASSWORD + TOKEN})
            return
        if scenario.mutation == "oversized" and index == 1:
            self.reply(200, {"secret": TOKEN, "large": "x" * smoke.BODY_LIMIT})
            return
        if scenario.mutation == "oversized_no_length" and index == 1:
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(b"x" * (smoke.BODY_LIMIT + 1))
            return
        if scenario.mutation == "slow_headers" and index == 1:
            time.sleep(0.35)
            self.reply(200, {"secret": TOKEN})
            return
        if scenario.mutation == "slow_body" and index == 1:
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", "100")
            self.end_headers()
            for _ in range(100):
                self.wfile.write(b" ")
                self.wfile.flush()
                time.sleep(0.02)
            return
        if scenario.mutation == "duplicate_json" and index == 1:
            self.raw_reply(b'{"success":true,"success":false,"secret":"' + TOKEN.encode() + b'"}')
            return
        if scenario.mutation == "invalid_json" and index == 1:
            self.raw_reply(TOKEN.encode())
            return
        if scenario.mutation == "encoded_response" and index == 1:
            self.raw_reply(b"{}", encoding="gzip")
            return
        if self.path == "/api/auth/login":
            if body != {"username": "admin", "password": PASSWORD}:
                self.reply(401, {})
                return
            value = {
                "success": True,
                "user": {"username": "admin", "user_id": str(uuid.uuid4())},
                "token": TOKEN,
                "access_token": TOKEN,
                "session_id": SESSION,
            }
            if scenario.mutation == "missing_session":
                del value["session_id"]
            self.reply(200, value)
            return
        if self.headers.get("Authorization") != "Bearer " + TOKEN:
            self.reply(401, {})
            return
        if self.path == "/api/approval-gates":
            identifier = str(uuid.uuid4())
            if scenario.mutation == "duplicate_gate" and scenario.records:
                identifier = next(iter(scenario.records))
            value = dict(body)
            value.update(
                id=identifier,
                status="pending",
                requested_by_agent=None,
                workflow_id=None,
                workflow_step=None,
                source=None,
                resume_ref=None,
                decided_by_user=None,
                decided_at=None,
                comments=[],
                task_links=[],
            )
            if scenario.mutation == "action_injected":
                value["context"]["action"] = "unexpected"
            if scenario.mutation == "mistyped_context":
                value["context"]["synthetic"] = 1
            scenario.records[identifier] = value
            self.reply(201, value)
            return
        pieces = self.path.split("/")
        value = scenario.records[pieces[3]]
        if self.command == "POST":
            action = pieces[4]
            value.update(
                status="approved" if action == "approve" else "rejected",
                decided_by_user="admin",
                decided_at="2026-10-18T00:00:00+00:00",
                comments=[{
                    "id": str(uuid.uuid4()),
                    "approval_id": value["id"],
                    "author": "admin",
                    "author_type": "human",
                    "body": body["comment"],
                }],
            )
        value = copy.deepcopy(value)
        if self.command == "GET":
            if scenario.mutation == "stale_readback":
                value["status"] = "pending"
            if scenario.mutation == "wrong_decider":
                value["decided_by_user"] = TOKEN
            if scenario.mutation == "wrong_comment":
                value["comments"][0]["body"] = PASSWORD
        self.reply(200, value)

    def reply(self, status, value):
        raw = json.dumps(value).encode()
        self.raw_reply(raw, status=status)

    def raw_reply(self, raw, status=200, encoding=None):
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        if encoding:
            self.send_header("Content-Encoding", encoding)
        self.end_headers()
        self.wfile.write(raw)


class Scenario:
    def __init__(self, mutation=None):
        self.mutation = mutation
        self.calls = []
        self.records = {}


@contextlib.contextmanager
def server(scenario):
    httpd = MockServer(("127.0.0.1", 0), Handler)
    httpd.scenario = scenario
    thread = threading.Thread(target=httpd.serve_forever, kwargs={"poll_interval": 0.01}, daemon=True)
    thread.start()
    try:
        yield httpd.server_address[1]
    finally:
        httpd.shutdown()
        httpd.server_close()
        thread.join(timeout=1)


class SmokeTests(unittest.TestCase):
    def exercise(self, mutation=None, seconds=1):
        scenario = Scenario(mutation)
        output = io.StringIO()
        errors = io.StringIO()
        with server(scenario) as port:
            client = smoke.Client(port=port, request_seconds=seconds)
            with contextlib.redirect_stdout(output), contextlib.redirect_stderr(errors):
                result = smoke.run(client, ENV)
        summary = json.dumps(result)
        for secret in (PASSWORD, TOKEN, SESSION):
            self.assertNotIn(secret, summary + output.getvalue() + errors.getvalue())
        self.assertIsNone(client.token)
        return result, scenario

    def assert_failure(self, mutation, code, count, seconds=1):
        result, scenario = self.exercise(mutation, seconds)
        self.assertEqual(result["status"], "FAIL")
        self.assertEqual(result["code"], code)
        self.assertEqual(len(scenario.calls), count)
        return result

    def test_two_real_http_mock_lifecycles(self):
        result, scenario = self.exercise()
        self.assertEqual(result["status"], "PASS")
        self.assertEqual(result["completed_lifecycles"], 2)
        self.assertEqual(len(scenario.calls), 7)
        self.assertEqual(len(scenario.records), 2)
        self.assertEqual({r["status"] for r in scenario.records.values()}, {"approved", "rejected"})
        for method, path, body, auth in scenario.calls[1:]:
            self.assertTrue(auth == "Bearer " + TOKEN)
            if path == "/api/approval-gates":
                self.assertEqual(set(body), {"title", "approval_type", "description", "context"})
                self.assertEqual(body["context"], {"synthetic": True})
            elif method == "POST":
                self.assertEqual(body, {"comment": smoke.COMMENT})

    def test_redirect_not_followed(self):
        self.assert_failure("redirect", "REDIRECT_REFUSED", 1)

    def test_first_login401_never_retried(self):
        result = self.assert_failure("login401", "LOGIN_REFUSED_READINESS_UNESTABLISHED", 1)
        self.assertEqual(result["http_status"], 401)

    def test_human_guard403_stops_without_second_gate(self):
        result = self.assert_failure("decision403", "HTTP_STATUS_MISMATCH", 3)
        self.assertEqual(result["http_status"], 403)

    def test_declared_oversized_body_rejected(self):
        self.assert_failure("oversized", "RESPONSE_TOO_LARGE", 1)

    def test_undeclared_oversized_body_rejected(self):
        self.assert_failure("oversized_no_length", "RESPONSE_TOO_LARGE", 1)

    def test_header_deadline(self):
        self.assert_failure("slow_headers", "HTTP_DEADLINE_EXCEEDED", 1, seconds=0.15)

    def test_dripping_body_absolute_deadline(self):
        started = time.monotonic()
        self.assert_failure("slow_body", "HTTP_DEADLINE_EXCEEDED", 1, seconds=0.15)
        self.assertLess(time.monotonic() - started, 0.8)

    def test_duplicate_json_rejected(self):
        self.assert_failure("duplicate_json", "INVALID_JSON", 1)

    def test_invalid_json_contains_no_response_leak(self):
        self.assert_failure("invalid_json", "INVALID_JSON", 1)

    def test_encoded_response_rejected(self):
        self.assert_failure("encoded_response", "ENCODED_RESPONSE_REFUSED", 1)

    def test_missing_session_prevents_create(self):
        self.assert_failure("missing_session", "LOGIN_SESSION_MISSING", 1)

    def test_unexpected_action_prevents_decision(self):
        self.assert_failure("action_injected", "NO_ACTION_CONTEXT_CHANGED", 2)

    def test_context_boolean_is_strict(self):
        self.assert_failure("mistyped_context", "NO_ACTION_CONTEXT_CHANGED", 2)

    def test_stale_db_readback_is_not_success(self):
        self.assert_failure("stale_readback", "GATE_STATUS_MISMATCH", 4)

    def test_wrong_decider_is_not_success(self):
        self.assert_failure("wrong_decider", "DECIDER_MISMATCH", 4)

    def test_wrong_stored_comment_is_not_success(self):
        self.assert_failure("wrong_comment", "COMMENT_MISMATCH", 4)

    def test_distinct_records_required(self):
        self.assert_failure("duplicate_gate", "DISTINCT_GATE_REQUIRED", 5)

    def test_missing_password_no_http(self):
        client = smoke.Client()
        result = smoke.run(client, {})
        self.assertEqual(result["code"], "ADMIN_PASSWORD_MISSING_OR_INVALID")
        self.assertEqual(result["http_events"], [])


if __name__ == "__main__":
    unittest.main()
