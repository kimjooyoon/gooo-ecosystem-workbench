# Gooo assesses source-origin input collisions

Observed on 2026-10-08 with workbench source
`f689cfe766d695d2d602a5e27e6c1e8a36ca7e6d`, runtime SDK
`v0.2.25-experimental`, Go 1.27.1, macOS arm64.
The installed Gooo 0.6.8-dev binary executes the assessment policy; its runtime
receipt binds compiler source `86b182da4543efa24ea5bcb504e7b13ea1449c39`.
The v2 input itself comes from the separately pinned helper-flow compiler
`246fdf5b0d40a1d1e148560f8389cdfb364da998`. These are different roles.

| Measurement | Expression v1 | Source origin v2 |
| --- | ---: | ---: |
| Source families | 1 | 1 |
| Candidate arrangements | 8 | 8 |
| Distinct complete feature arrays | 2 | 4 |
| Conflicting groups | 2 | 4 |
| Maximum compatible rows under the supplied labels | 2/8 | 4/8 |
| Actual model predictions | 8 | 0 |
| Accepted first predictions | 1/8 | unmeasured |

The two contracts have separate measurements on the same source family. Adding
their rows would not create sixteen independent programs. The origin projection
distinguishes `input` from the helper-computed `stem`; it still merges `&&` and
`||`. Grouping compares all 3,072 bytes of each 768-float array. Hashes identify
the stored groups; hashes alone do not determine equality.

The unchanged Gooo `Assess` activity returns `representation-collision` and
`preserve-distinguishing-source-facts` in both runs. Its saved native composition
replays with zero inference. The eight model calls above belong only to the v1
feature chooser. No v2 weights were trained, relabeled or applied, and neither
run evaluates new filename cases. Accepted masks remain the prior order study's
explicit assumptions. A changed label does not change the model features.

## Reproduce

From the pinned workbench checkout, choose new output directories:

```sh
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-order.json --model builtin --out out/expression-v1
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-origin-order.json --out out/origin-v2
```

Both directories retain the supplied input, exact groups, Gooo policy result,
policy source, generated code and saved native composition. `model_observed`
distinguishes unmeasured predictions from observed zero successes.
The v1 model metadata and weight hashes match the prior observation.

The source origin fixture is copied from the three decoded `context.parts`
of each [original helper-flow export](https://raw.githubusercontent.com/wiki/kimjooyoon/meta-ontology-go/observations/helper-value-flow-20261008/README.md).
Its four feature hashes match that earlier independent comparison script.
This is a reusable assessment route with unchanged feature definitions.

`SHA256SUMS` covers the retained files other than itself. Verify from this directory:

```sh
shasum -a 256 -c SHA256SUMS
```
