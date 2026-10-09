# Own source-aware path decision model

The model ranks existing Gooo path labels from typed source facts and complete
Korean/English intent. The same network can be reused at1..16 sites through the
existing per-choice compiler route. The compiler owns allowed alternatives,
finite tests, bounded reconstruction and saved replay.

Architecture:256 features →48 ReLU values →8 path-label scores,12,728
parameters. QAT weights occupy2,759 packed bytes,12,896 resident tensor bytes
plus8 scale bytes. Each worker has1,248 scratch bytes. Metadata, Go runtime and
caller objects add memory. Five trits per byte use1.6 stored bits; FP32 biases
make total storage exceed the1.585-bit entropy per trit.

## Current evidence

Clean trainer`c2cde2d1be126c758382c3769563eef1f45e86bb`, CPU/Go1.27.2.
Six source-derived snapshots,48 authored training utterances and24 other
utterances sharing those snapshots. FP32 matched16/24, post-training
ternary11/24, QAT15/24. QAT abstained on5/24; one wrong prediction had
confidence≥0.8. These are separate observations; confidence is uncalibrated.

QAT SDK inference averaged about10µs/site,0.160ms for16 serial sites in
one200-repeat local run. Focused testing observed0 heap allocations with a
worker-owned workspace. Training/export/assessment took0.86s wall with maximum
RSS14,401,536bytes; training phases took0.183416750s. Process user+system
time was0.41s. Host utilization was UNMEASURED and GPU was not used.

Actual installed compiler426 accepted a one-choice unary preflight and matched
generation's input digest. Model1 call/12,958ns selected the declared reverse
layout. Local1/1, caller1/1, separate evaluation3/3 and replay3/3 passed, with0
new replay calls. Exact input9007199254740993/output18014398509481986 were
checked as Go json.Number strings. This is one native program observation.

## Use

From a compiler checkout with typed model preflight, using a fresh output directory:

```sh
gooo body-context --activity Choose \
  --model /path/to/gooo-ecosystem-workbench/models/per-choice-20261010/qat_ternary/model.json \
  examples/caller-typed-paths/unary.gooo.fixture

gooo body-construct --source examples/caller-typed-paths/unary.gooo.fixture \
  --entry Main --construction-cases examples/caller-typed-paths/construction-cases.json \
  --cases examples/caller-typed-paths/evaluation-cases.json --attempts 2 \
  --model /path/to/gooo-ecosystem-workbench/models/per-choice-20261010/qat_ternary/model.json \
  --out /tmp/fresh-own-path-program
```

Without a model, Gooo follows its deterministic source-declared route. The
default builtin model remains unchanged. Original source/model identities and
actual finite results stay in the receipts. See the
[frozen plan](../../experiments/path-chooser/PLAN.md) and
[original study](../../publication/per-choice-decision-20261010/README.md).
Weights/authored data are MIT under this repository's license. No Laya or other
pretrained base is inherited. Next work should use held-out source shapes,
test choice interactions and measure abstention/calibration independently.
