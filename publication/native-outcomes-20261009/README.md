# Native arithmetic outcomes — 2026-10-09

The workbench now reads compiler `joint-construction/v6` and arithmetic
`body-composition-runtime/v3` observations. Source-owned Gooo rules decide the
next action from recounted values. Transport and exact JSON-number handling use Go.

## Retained observations

- [Nine original compressed receipts](../../examples/caller-native-failure) include
  six caller-construction runs, a dependency graph, a separately authored subset
  of graph expectations, and an all-fault candidate space.
- [summary.json](summary.json) recounts the six original compiler runs. The
  [recount program](../../experiments/native-outcome-observe/main.go) independently
  compares source hashes, original construction cases, final expectations and
  native outputs; it also requires the workbench reader to produce the same counts.
- Those originals used clean compiler `717d876e481c233ae3587488077be87843175fe6`.
  The workbench CI pins `3b762198b42d9ab552421b19519d3d4834b34ab3`, which adds the
  compiler's retained evidence and documentation to that implementation.
- The local feedback run and model run use that clean CI revision. Public
  `v0.6.15-dev` assets remain source `dc75f59`; this feature has no new release tag yet.
- The `loop-*.json.gz` and `adaptive-*.json.gz` files preserve every local round,
  feedback update, original cases and final evaluation. The latter starts with a
  native fault on an evaluation-only input; that unchanged row is consumed in
  construction, producing three rounds, four attempts and final 3/3 observations.
- Local full workbench race tests passed in 163.164 seconds, with a subsequent
  race run for the added adaptive-fault regression and exact-integer check
  passing in 6.544 seconds. Vet passed. The original unsupported-v6 TDD failure
  and successful reader run are retained alongside those logs.

The six-candidate example records two local rejections, four native combinations,
and one native arithmetic fault. The selected program matches 4/4 supplied final
expectations. With five attempts the result stays partial at 1/4.

The mixed example records 48 attempts in fixed order and six with the unchanged
own graph model. Selected sources agree. The retained original model prediction
took 33,833 ns in one observation; the full original command took 1.50 seconds and
peaked at 87,113,728 bytes of RSS. These describe different scopes and a single
run. They do not establish a speedup or a machine-wide CPU utilization measurement.
No model training is part of these runs. Model metadata and weight digests are
`3c68205a660695103712e2115ce90accd298b406dced8ad5ad5c9b84bab0a202` and
`76f68845a03ed8c8bc261a57c919e96dcdd352d3c35bcdaeb3d9b51a88975c6f`.

The additional local run on the CI revision is retained as
`workbench-model.json.gz`, with zero-inference saved replay and the original
`workbench-model.time.txt`: 2.23 seconds wall time, 0.78 user + 0.53 system CPU
seconds, and 86,769,664 bytes maximum RSS. Tests were running concurrently, so
this is a recorded execution rather than a controlled performance comparison.

## Counting and continuation

`matched + mismatched + faulted + blocked + unobserved` describes supplied
expected outputs. A completed native observation has zero unobserved outputs.
Activity failure counts also include activities with no named expectation.
The graph subset scores 6/6 named outputs while still retaining one failed and
one blocked activity; Gooo therefore proposes adding those expectations.

Rejected combinations never acquire caller scores. Native fault combinations
count within native attempts and consume the original attempt budget. A selected
fault outside named expectations prevents a complete construction decision.
Two original successful native process observations with matching output digests
are required for arithmetic-outcome metadata. The workbench checks this recorded
consistency; independent source execution remains the compiler replay's job.

The automatic feedback example uses five rounds and 14 total attempts, including
repeated work across rounds. It preserves the original counterexample, consumes
it on the next round, and evaluates the final saved construction with no new
model inference. Adaptive evaluation influences selection; the final four-row
evaluation runs afterward and its input overlap remains visible in the receipt.

Current scope: finite examples, int64 division/remainder by zero, explicit
dependency blocking, exact integer values and deterministic Gooo advice.
Unknown native panic, process failure, timeout and cancellation remain failures.
