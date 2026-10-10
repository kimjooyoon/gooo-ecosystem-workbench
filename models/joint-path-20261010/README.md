# Small Gooo model trained on compatible path combinations

This experimental own model scores Gooo's declared alternatives from typed source
features and Korean/English intent. Gooo assembles and checks the selected body.
The architecture remains256→48→8 with12,728 parameters; QAT weights occupy2,759B,
resident tensors12,896B plus8B scales and worker scratch1,248B. There is no token
generation or pretrained external base. Original code/data/weights are MIT.

## What changed in training

The previous model independently learned direction labels. The new loss assigns
probability mass to every complete combination known to satisfy consumed cases.
For one three-choice Gooo function, the compatible combinations are masks1 and2.
Combining their per-site labels independently also admits two incorrect masks.

Starting from the published own FP32 checkpoint, two arms each used100 additional
FP32 epochs and100 QAT epochs at learning rate.002,600 updates per phase. The
control rehearsed48 original label examples; this model mixed that label loss
with the joint-target loss at equal weight. The nine target cases were consumed
in learning. The [frozen plan](../../experiments/path-chooser/JOINT_PLAN.md)
preceded the one original study; no result-driven hyperparameter retry occurred.

| Observation | Previous QAT | Label-only continuation | Joint continuation |
| --- | ---: | ---: | ---: |
| New native caller inputs, same function |6/11|6/11|11/11|
| Saved replay |6/11|6/11|11/11|
| Initial model calls |3|3|3|
| New replay calls |0|0|0|
| Existing24 development utterances |15/24|18/24|13/24|
| Development abstentions |5|1|3|

The joint model selected mask2 in actual public Gooo0.6.24. Native inputs were
disjoint from the target cases and previous finite evaluations, including exact
±9007199254740995. They exercise the same authored function family. Existing
utterance performance regressed, so this checkpoint is an explicit experimental
option; the default model is unchanged. Unseen-source behavior and calibrated
confidence remain unmeasured. Its forward site scores still factorize and can
favor one compatible implementation.

A post-hoc check also found a local hint mismatch in all three artifacts. The
source hint says "Compare whether input is negative", while their generated
condition is`0 < input`. For inputs-1,0,1 that condition matches the negative
predicate in1/3 probes. Other branch choices let the joint model satisfy the
caller outputs anyway. This hint was not an enforced assembly constraint;
finite caller success therefore does not establish every local intent. The
condition observation and authored diagnostic probes are retained separately.

## PC measurements

The complete two-arm training/export/assessment command took1.18s wall,
0.69s user+0.04s system CPU, and reported maximum RSS16,318,464B. GPU was unused;
host CPU utilization was not measured. SDK ranking for the three source sites
averaged30.27µs per sequence over200 repeats. Actual Gooo construction used three
predictions totaling39,501ns. Its whole command took0.86s and reported maximum
RSS88,014,848B, including compilation/execution/receipts. Cache-sensitive command
times are not a causal speedup comparison.

## Try it

From this repository with public Gooo0.6.24 and Go1.27.2, use a fresh output:

```sh
gooo body-context --activity Choose \
  --model "$PWD/models/joint-path-20261010/qat_ternary/model.json" \
  experiments/path-outcomes/testdata/source.gooo.fixture

GOWORK=off GOTOOLCHAIN=go1.27.2 gooo body-construct \
  --source experiments/path-outcomes/testdata/source.gooo.fixture --entry Main \
  --construction-cases models/joint-path-20261010/construction-cases.json \
  --cases experiments/path-chooser/joint-evaluation-cases.json --attempts 8 \
  --model "$PWD/models/joint-path-20261010/qat_ternary/model.json" \
  --out /tmp/fresh-joint-path-example
```

Omitting `--model` uses Gooo's deterministic declared route. The original source,
model, cases and construction evidence remain bound in saved receipts.

[Original losses, both arms, native outputs and independent exact-int64 recount](../../publication/joint-path-learning-20261010).
Producer:`e2e259a966473be21250a01b99644f646992c6c4`.
Weights SHA256:`dcd8e44591626d421d4961bfef82ec197e947cb1d5d2cd92868d908bc7de4aed`.
