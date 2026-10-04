# Initial ecosystem observation

This directory retains actual outputs from the first workbench release. Compiler
source: `f144dddb8261b9b525181dee510afc65f1603153`, Go 1.27.1, darwin/arm64.
The shared QAT model is unchanged; its identity is recorded in
[the model card](../../models/shared-qat/README.md).

| Gooo program | Ordering | Named outputs matched | Record fields matched | Generation predictions |
| --- | --- | ---: | ---: | ---: |
| 13 standard functions | deterministic | 91/91 | — | 0 |
| Diagnostic rules | deterministic | 24/24 | 24/24 | 0 |
| Diagnostic rules | model | 24/24 | 24/24 | 1 |
| Starter planner | deterministic | 6/6 | 9/9 | 0 |
| Starter planner | model | 6/6 | 9/9 | 1 |

These are **151/151 named output checks and 66/66 record-field checks** across
authored finite examples, with the same diagnostic/starter examples repeated
under two ordering modes. They are observations of these examples. Each of the
five generated compositions was also replayed and its actual values recounted.
Native execution and saved replay made zero new model predictions.

The [summary](summary.json) includes candidate-selection counts separately.
`verification/` retains native generation results, saved replay results, selected
Gooo and generated Go. `record-starter/` retains a Gooo-authored record source,
its project plan, and the compiler's successful check/code-generation result.
`partial-diagnosis/` retains the actual observed 17/21 fields, one type rejection,
and the Gooo-produced `partial` / `repair-and-replay` action. Its complete original
input is [the captured public example](../../examples/partial-composition.json).
The input's SHA256 is retained in `observation.json`.

For that new partial-diagnosis request, the native expectation checks detail
transport through `ObservationEcho`. The diagnosis itself has no new independent
oracle attached to the request. The authored diagnostic golden cases above check
classification and actions separately. An automatic repair executor remains
future work.

`local-tests.txt` records the successful native integration and race tests.
`checksums.sha256` binds the retained files. Reproduce with:

```sh
go run ./cmd/workbench verify --model builtin --out out/my-observation
GOOO_COMPILER="$(command -v gooo)" go test -race ./...
go run ./cmd/workbench scaffold --profile record --model builtin --out out/my-project
go run ./cmd/workbench diagnose --input examples/partial-composition.json \
  --model builtin --out out/my-diagnosis
```

This release adds three kinds of ecosystem tools. Function counts, ordering
modes, output checks, and replay runs are recorded in their own units.
