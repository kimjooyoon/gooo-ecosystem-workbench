# AutoBot disposable integration diagnostic

This experiment follows an AutoBot maintainer's invitation to retry the main-app
Docker login after [PR #18738](https://github.com/mrveiss/AutoBot-AI/pull/18738).
It checks out merge commit `5c224241326d909219a1f525cbe88a9d65eedff4` and uses the
upstream Dockerfile, generated per-deployment credentials and normal login API.

The experiment runs on a fresh standard GitHub Ubuntu runner. It starts only the
backend, PostgreSQL, Redis, ChromaDB and secrets initializer. Published host ports
are removed by a Compose override. It supplies no account tokens or paid provider
keys, starts no Ollama service, and downloads no model weights. The unmodified
backend image still installs its full Python dependencies and agent CLIs; startup
may attempt unavailable local service connections. This is a bounded diagnostic,
not a claim that the full product works without a model.

After container health and the actual background admin-seeding completion logs,
the client makes seven real HTTP requests from inside the backend container:
login, then create/approve/read and create/reject/read. Both synthetic gates omit
`context.action`, workflow, task and resume bindings. The test checks persisted
status, comments and the authenticated decider. It does not run a terminal command
or ask the model to make a decision.

The first failure is preserved. Authentication is never bypassed, no replacement
mock service is used in the hosted trial, and readiness failure is distinguished
from a login regression. Commands have output/time limits, the overall experiment
has a 20-minute budget plus bounded cleanup, and diagnostics omit generated
credentials and HTTP response bodies. Cleanup removes the experiment's containers
and volumes. GitHub's ephemeral runner is discarded after the job.

## Why this belongs in Gooo ecosystem research

Successful installation is a prerequisite for measuring whether Gooo policy
comparison and replay are useful in a real agent system. This experiment only
tests the host's API/database lifecycle. It does **not** install Gooo into AutoBot,
establish adoption, reconstruct historical policy A after A→B→missing-A, demonstrate
autonomous safety, or show an independent person's approval. Those are separate
questions that require actual inputs and later evidence.

The workflow is restricted to the experiment branch. It does not change the
workbench's normal CI or release behavior. No normal CI checks are being skipped
for a proposed main-branch change.

## Local client checks

`python3 -B experiments/autobot-hosted/test_http_smoke.py` uses an owned loopback
mock server to check client failure handling. Those tests are not AutoBot
integration results. The hosted diagnostic itself requires a GitHub runner and
an exact, clean upstream checkout.
