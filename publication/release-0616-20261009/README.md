# Public Gooo 0.6.16 workbench integration

The compiler pin is published source
`146a5085427972f3a50c38b33384e3911c9019eb` (Go 1.27.1,
decision runtime v0.2.26-experimental). The actual downloaded macOS arm64
executable was verified and installed before these observations.

Local `go test -race ./...` passed (root package 172.088 seconds) using that
installed binary; `go vet ./...` also passed. Two Gooo feedback examples were
executed again with their original cases:

- The six-candidate budget example used five rounds and 14 total attempts,
  retaining final 4/4. The final history includes two preflight rejections and
  one native-fault combination. Saved evaluation makes zero new model calls.
- The adaptive fault example added the original evaluation-only faulting input
  and expected value to construction. Three rounds and four attempts retained
  final 3/3 and zero new model calls during saved evaluation.

All eight rounds were independently compared with the earlier feature study
using exact JSON numbers. Original and selected Gooo source, construction cases,
candidate selection, all native traces, fault operands, expectations and large
integers are unchanged. `compare.go.txt` contains the comparison and its output
is retained. `observations.tar.gz` contains both loop reports, all eight raw
rounds and each round's original/selected source and construction cases.

The first adaptive invocation used a nonexistent case filename and stopped before
construction. The corrected command uses the existing `adaptive-initial.json`,
`adaptive-evaluation.json` and `adaptive-holdout.json` fixtures without changing
them. That command-path error does not enter the language outcome counts.

The release source is pinned in CI; its result is a separate observation from
these local runs. The earlier feature records retain their original revisions.
See the [public release observations](https://raw.githubusercontent.com/wiki/kimjooyoon/meta-ontology-go/observations/release-0616-published-20261009/README.md)
for exact downloaded assets, the six direct compiler construction/replay results,
two graph observations, two source reuses, unchanged own-model inference and
independent recount. No new model was trained for this integration.
