#!/usr/bin/env python3
"""Bounded real HTTP diagnostic for AutoBot 5c224241; Python standard library.

Run inside the isolated backend container, which inherits the generated admin
credentials. There is no configurable remote host, proxy support, redirect
following, auth bypass, provider call, terminal command, or retry. HTTP bodies
and credentials remain in memory. stdout contains only an allowlisted summary.

This creates two actual disposable database records and exercises ordinary
no-action approval gates. It does not establish model-driven behavior, human
adoption, policy history reconstruction, Gooo use, or production safety.
"""

import contextlib
import http.client
import json
import os
import signal
import socket
import sys
import time
import uuid


REVISION = "5c224241326d909219a1f525cbe88a9d65eedff4"
BODY_LIMIT = 64 * 1024
REQUEST_SECONDS = 12
USERNAME = "admin"
CONTEXT = {"synthetic": True}
DESCRIPTION = "Disposable HTTP lifecycle diagnostic; no requested effect"
COMMENT = "Synthetic hosted lifecycle smoke"


class Failure(Exception):
    """Only locally defined codes may leave the process."""

    def __init__(self, code, http_status=None):
        super().__init__(code)
        self.code = code
        self.http_status = http_status


def require(condition, code="RESPONSE_SCHEMA_MISMATCH"):
    if not condition:
        raise Failure(code)


@contextlib.contextmanager
def absolute_deadline(seconds):
    """Bound even slow-drip headers/bodies, not just socket inactivity."""
    def expired(_signum, _frame):
        raise Failure("HTTP_DEADLINE_EXCEEDED")

    previous_handler = signal.getsignal(signal.SIGALRM)
    started = time.monotonic()
    signal.signal(signal.SIGALRM, expired)
    previous_timer = signal.setitimer(signal.ITIMER_REAL, seconds)
    try:
        yield
    finally:
        signal.setitimer(signal.ITIMER_REAL, 0)
        signal.signal(signal.SIGALRM, previous_handler)
        if previous_timer[0]:
            remaining = max(0.000001, previous_timer[0] - (time.monotonic() - started))
            signal.setitimer(signal.ITIMER_REAL, remaining, previous_timer[1])


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "INVALID_JSON")
        result[key] = value
    return result


def reject_constant(_value):
    raise Failure("INVALID_JSON")


class Client:
    # The non-default port is used only by local mock-server tests. Host remains
    # fixed loopback and the command-line entry point accepts no arguments.
    def __init__(self, port=8001, request_seconds=REQUEST_SECONDS):
        self.port = port
        self.request_seconds = request_seconds
        self.token = None
        self.stage = "configuration"
        self.events = []

    def request(self, stage, method, path, expected_status, body=None):
        self.stage = stage
        connection = http.client.HTTPConnection(
            "127.0.0.1", self.port, timeout=self.request_seconds
        )
        headers = {"Accept": "application/json", "Accept-Encoding": "identity"}
        if self.token is not None:
            headers["Authorization"] = "Bearer " + self.token
        payload = None
        if body is not None:
            payload = json.dumps(body, allow_nan=False).encode("utf-8")
            headers["Content-Type"] = "application/json"
        try:
            with absolute_deadline(self.request_seconds):
                connection.request(method, path, body=payload, headers=headers)
                response = connection.getresponse()
                if 300 <= response.status < 400:
                    raise Failure("REDIRECT_REFUSED", response.status)
                if response.status != expected_status:
                    if stage == "login" and response.status == 401:
                        raise Failure("LOGIN_REFUSED_READINESS_UNESTABLISHED", response.status)
                    raise Failure("HTTP_STATUS_MISMATCH", response.status)
                require(
                    response.getheader("Content-Type", "").split(";", 1)[0].lower()
                    == "application/json",
                    "UNEXPECTED_CONTENT_TYPE",
                )
                require(
                    response.getheader("Content-Encoding", "identity").lower()
                    == "identity",
                    "ENCODED_RESPONSE_REFUSED",
                )
                length = response.getheader("Content-Length")
                if length is not None:
                    require(length.isascii() and length.isdecimal(), "INVALID_CONTENT_LENGTH")
                    require(int(length) <= BODY_LIMIT, "RESPONSE_TOO_LARGE")
                raw = response.read(BODY_LIMIT + 1)
                require(len(raw) <= BODY_LIMIT, "RESPONSE_TOO_LARGE")
                if length is not None:
                    require(len(raw) == int(length), "TRUNCATED_RESPONSE")
                try:
                    value = json.loads(
                        raw.decode("utf-8"),
                        object_pairs_hook=unique_object,
                        parse_constant=reject_constant,
                    )
                except (UnicodeDecodeError, ValueError, RecursionError):
                    raise Failure("INVALID_JSON") from None
                require(isinstance(value, dict), "RESPONSE_SCHEMA_MISMATCH")
        except Failure:
            raise
        except (TimeoutError, socket.timeout):
            raise Failure("HTTP_DEADLINE_EXCEEDED") from None
        except (OSError, http.client.HTTPException):
            raise Failure("HTTP_TRANSPORT_ERROR") from None
        finally:
            connection.close()
        self.events.append({"stage": stage, "status": "HTTP_OK", "http_status": expected_status})
        return value


def nonempty_string(value, maximum=4096):
    return isinstance(value, str) and 0 < len(value) <= maximum


def canonical_uuid(value):
    require(isinstance(value, str) and len(value) == 36)
    try:
        parsed = str(uuid.UUID(value))
    except ValueError:
        raise Failure("RESPONSE_SCHEMA_MISMATCH") from None
    require(parsed == value)
    return parsed


def check_gate(value, title, status, gate_id=None):
    actual_id = canonical_uuid(value.get("id"))
    if gate_id is not None:
        require(actual_id == gate_id, "GATE_ID_MISMATCH")
    require(value.get("status") == status, "GATE_STATUS_MISMATCH")
    require(value.get("title") == title)
    require(value.get("description") == DESCRIPTION)
    require(value.get("approval_type") == "workflow_gate")
    context = value.get("context")
    require(
        isinstance(context, dict) and len(context) == 1 and context.get("synthetic") is True,
        "NO_ACTION_CONTEXT_CHANGED",
    )
    for key in ("requested_by_agent", "workflow_id", "workflow_step", "source", "resume_ref"):
        require(key in value and value[key] is None, "NO_ACTION_BINDING_CHANGED")
    require(value.get("task_links") == [], "NO_ACTION_BINDING_CHANGED")
    if status == "pending":
        require(value.get("decided_by_user") is None and value.get("decided_at") is None)
        require(value.get("comments") == [])
    else:
        require(value.get("decided_by_user") == USERNAME, "DECIDER_MISMATCH")
        require(nonempty_string(value.get("decided_at")), "DECISION_TIME_MISSING")
        comments = value.get("comments")
        require(isinstance(comments, list) and len(comments) == 1, "COMMENT_MISMATCH")
        comment = comments[0]
        require(isinstance(comment, dict), "COMMENT_MISMATCH")
        require(comment.get("approval_id") == actual_id, "COMMENT_MISMATCH")
        require(comment.get("author") == USERNAME, "COMMENT_MISMATCH")
        require(comment.get("author_type") == "human", "COMMENT_MISMATCH")
        require(comment.get("body") == COMMENT, "COMMENT_MISMATCH")
        canonical_uuid(comment.get("id"))
    return actual_id


def run(client, environ):
    """Return only allowlisted diagnostic fields; never return response content."""
    completed = 0
    result = {
        "schema": "gooo/autobot-no-action-http-smoke/v1",
        "upstream_revision": REVISION,
        "scope": "synthetic_login_and_two_no_action_database_lifecycles",
        "model_execution": False,
        "terminal_execution": False,
        "historical_policy_reconstruction": False,
        "gooo_adoption_claim": False,
    }
    try:
        password = environ.get("AUTOBOT_ADMIN_PASSWORD")
        require(nonempty_string(password, 128), "ADMIN_PASSWORD_MISSING_OR_INVALID")
        require(environ.get("AUTOBOT_ADMIN_USERNAME", USERNAME) == USERNAME, "ADMIN_USERNAME_MISMATCH")
        login = client.request(
            "login", "POST", "/api/auth/login", 200,
            {"username": USERNAME, "password": password},
        )
        password = None
        require(login.get("success") is True, "LOGIN_NOT_SUCCESSFUL")
        user = login.get("user")
        require(isinstance(user, dict) and user.get("username") == USERNAME, "LOGIN_USER_MISMATCH")
        canonical_uuid(user.get("user_id"))
        token = login.get("token")
        require(nonempty_string(token, 16384), "LOGIN_TOKEN_MISSING_OR_INVALID")
        require(all(33 <= ord(c) <= 126 for c in token), "LOGIN_TOKEN_MISSING_OR_INVALID")
        require(login.get("access_token") == token, "LOGIN_TOKEN_ALIAS_MISMATCH")
        require(nonempty_string(login.get("session_id")), "LOGIN_SESSION_MISSING")
        client.token = token
        login = None
        previous_id = None
        for action, expected_state in (("approve", "approved"), ("reject", "rejected")):
            title = "Synthetic hosted approval smoke: " + action
            created = client.request(
                action + "_create", "POST", "/api/approval-gates", 201,
                {
                    "title": title,
                    "approval_type": "workflow_gate",
                    "description": DESCRIPTION,
                    "context": dict(CONTEXT),
                },
            )
            gate_id = check_gate(created, title, "pending")
            require(gate_id != previous_id, "DISTINCT_GATE_REQUIRED")
            decided = client.request(
                action + "_decision", "POST", "/api/approval-gates/" + gate_id + "/" + action,
                200, {"comment": COMMENT},
            )
            check_gate(decided, title, expected_state, gate_id)
            stored = client.request(
                action + "_readback", "GET", "/api/approval-gates/" + gate_id, 200,
            )
            check_gate(stored, title, expected_state, gate_id)
            previous_id = gate_id
            completed += 1
        result.update(status="PASS", stage="complete", code="TWO_STORED_LIFECYCLES_VERIFIED")
    except Failure as failure:
        result.update(status="FAIL", stage=client.stage, code=failure.code)
        if failure.http_status is not None:
            result["http_status"] = failure.http_status
    except BaseException:
        # No exception text/traceback: third-party response data may be embedded.
        result.update(status="FAIL", stage=client.stage, code="UNEXPECTED_LOCAL_FAILURE")
    finally:
        client.token = None
    result["completed_lifecycles"] = completed
    result["http_events"] = client.events
    return result


def main():
    if len(sys.argv) != 1:
        print('{"status":"FAIL","stage":"configuration","code":"ARGUMENTS_REFUSED"}')
        return 1
    result = run(Client(), os.environ)
    print(json.dumps(result, sort_keys=True, allow_nan=False))
    return 0 if result["status"] == "PASS" else 1


if __name__ == "__main__":
    sys.exit(main())
