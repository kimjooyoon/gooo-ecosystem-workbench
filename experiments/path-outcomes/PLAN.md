# Finite compatible paths as decision-model targets

The previous three-choice native program passed two of three separate cases.
Before training another model, enumerate its declared paths and keep every path
that satisfies the supplied examples. A single arbitrary direction label can
discard another functioning implementation. Independent per-site labels can
also admit combinations that fail when assembled together.

This Go experiment reads a fresh compiler `body-context --include-plan` export,
checks the source, normalized plan and individual input hashes, and evaluates
up to 64 complete typed combinations. Larger palettes stop without producing
training targets. It neither changes source choices nor extends production
construction budgets. This is an offline full-palette observation.

The frozen first corpus uses the original three-choice `Choose` activity from
compiler366. These explicitly authored **consumed target cases** request absolute
value: -7, -5, -1, 0, 1, 3, 7 and ±9007199254740993. Earlier Main observations
motivate these labels; they are not a held-out test. MinInt64 is excluded from
this requested absolute-value profile because its positive value is outside
int64. No claim covers the rest of int64 or the caller Main.

`testdata/original-context.json` is the unchanged historical366 model preflight
from the [three-choice native observation](https://github.com/kimjooyoon/meta-ontology-go/wiki/Per-Choice-Decision-Model).
Unit tests use it as a binding fixture. Each real CLI observation obtains a fresh
export without loading that historical model.

For each typed candidate, retain its exact Gooo body, unchanged SDK Go source,
choices, all int64 outcomes and finite compatibility. Execute those Go projections
in a separate native process and compare every result to the SDK evaluator.
Native panics and evaluator failures remain observations, never valid targets.
Keep joint compatible masks, per-site marginal counts and the count of invalid
combinations allowed by those marginals. Output-equivalence groups apply only
to the observed input set.

No weights are trained in this corpus-construction experiment. The existing
2/3 native result and subsequent 5/5 feedback result keep their original scope.
The corpus is intended for a future joint/set-valued objective and source-family
split. It does not establish unseen-source accuracy or calibrated probabilities.

Run from a clean committed checkout with Go1.27.2 and a new external directory:

```sh
GOWORK=off GOTOOLCHAIN=go1.27.2 go run ./experiments/path-outcomes \
  --compiler /absolute/path/to/gooo \
  --source experiments/path-outcomes/testdata/source.gooo.fixture \
  --activity Choose --cases experiments/path-outcomes/testdata/targets.json \
  --out /tmp/gooo-compatible-paths-observation
```
