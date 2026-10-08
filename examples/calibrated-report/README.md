# Assemble a calibrated reading and its report

This small synthetic sensor example joins two Gooo activities. `Energy` fills a
numeric expression hole and squares its local value. `Report` constructs a
record containing that result, a threshold flag and a label. The source states
the intended transformation `(2 * input + 1)^2`, allowed grammars, field choices,
expectations and attempt limits.

The first quadratic grammar retains individual constants before shared fits.
The next source-declared grammar gives compatible shared expressions priority.
The existing Gooo search policy reads actual results and selects that alternative.
The optional own compact model ranks the three record-field choices after numeric
construction. The numeric search and policy run deterministically in both modes.

Use compiler commit `4421805b9db75e56bb97c571486afcf53f66089c` or a descendant.
The new grammar is available in the public compiler development branch.

```sh
go run ./cmd/workbench refine --compiler /path/to/gooo \
  --source examples/calibrated-report/source.gooo --activity Energy \
  --cases examples/calibrated-report/feedback-cases.json \
  --evaluation-cases examples/calibrated-report/evaluation-cases.json \
  --policy examples/search-policy/policy.gooo --search-policy --model builtin \
  --max-attempts 8 --max-rounds 4 --out out/calibrated-report
```

Omit `--model builtin` for fixed candidate ordering. The bundled model already
exists in this repository; the example downloads and trains no model.

## Four modes with saved observations

```sh
go build -trimpath -o /tmp/gooo-calibrated-study ./experiments/calibrated-report
/tmp/gooo-calibrated-study --compiler /path/to/gooo --out out/calibrated-study
```

The driver compares fixed ordering, own-model ordering, a model run limited to
one refinement round, and fixed ordering with the alternative grammar removed.
It stores original and revised sources, per-round model predictions and candidate
counts, compiled execution, replay and separate final evaluation.

Ten root inputs have two expected outputs each: the numeric activity and its
report. The three evaluation inputs are supplied only after source selection.
All reported counts describe this finite graph. The sign of the internal factor
is unobservable in the squared result: both `2 * input + 1` and its negation can
satisfy these expectations. A consumer that needs the signed calibration itself
would need an expectation for that value.

The two bounded controls retain incomplete results. Saved graph execution uses
no new inference. Candidate-space coverage and functional output counts remain
separate in the native records.
