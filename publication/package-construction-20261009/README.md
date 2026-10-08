# Package construction driven by Gooo feedback

These observations use compiler `e6a22ba3329c4511ce53296b65b42dc79dbad263`, built
cleanly with Go 1.27.1. The compiler version string remains 0.6.16-dev; this source
adds package construction after the public 0.6.16 release. No model was trained.

| Program | Fresh rounds | Total attempts including repeats | Local rejections | Native fault combinations | Initial model calls | Final holdout |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| imported budget | 5 | 14 | 5 | 1 | 0 | 4/4 |
| two imported libraries, own model | 8 | 105 | 40 | 8 | 8 | 4/4 |

Gooo's unchanged `joint-feedback` rule selects the original failing caller row
for the next round. Its `joint-next` rule proposes budgets. The first example
uses 1, 1, 2, 4, 8; the mixed example continues with 16, 32, 64. Final evaluation
rechecks saved construction with zero new model calls. Repeated attempts are
counted each time; a single mixed construction's 41 attempts and this complete
adaptive loop's 105 attempts measure different operations.

Each directory retains unchanged compressed compiler JSON for every round and
holdout, the loop record, original cases, the added feedback cases and the copied
workspace sources. The source-owning package and caller keys remain visible.
`compiler-build.json` binds the executable. `local-race.log` records the full
native workbench race run (191.796 seconds for the main package). The initial
unsupported-receipt error is also kept.

Run `go run ./experiments/package-construction-observe` to reproduce `summary.json`.
The program recounts each receipt, compares its original package-keyed cases,
retains the exact added row, verifies the full saved construction history during
holdout and checks separate model-call counts. Numbers use exact JSON decoding.
The large-int examples preserve `9007199254740993`.

The two-library run uses the existing all-data demonstration graph model. Its
metadata and weights identities are retained in every initial construction.
These related examples establish bounded operation and replay. Model transfer,
host-wide CPU utilization and comparative speed are separate questions; the
local full race suite ran concurrently with part of the mixed observation.

CI replays these retained observations and executes a fresh package loop and a
fresh model construction. CI and merge status are recorded after that run ends.
