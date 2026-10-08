# Let Gooo act on a search-space diagnosis

An activity can declare ordered alternatives for its expression grammar and
candidate cap. `workbench refine --search-policy` dispatches both candidate-space
and grammar-exhausted proposals to the extended Gooo policy. The policy decides
when to incorporate a failed example, increase attempts, advance to the next
declared setting, or retain a partial result.

```sh
go run ./cmd/workbench refine --compiler /path/to/gooo \
  --source examples/search-policy/source.gooo --activity Add \
  --cases examples/search-policy/feedback-cases.json \
  --evaluation-cases examples/search-policy/evaluation-cases.json \
  --policy examples/search-policy/policy.gooo --search-policy \
  --max-attempts 8 --max-rounds 6 --out out/search-policy
```

Use compiler `765918d9b0da11359ebbbe88c517831e4cebeb1f` or a descendant.
The original source contains `input + hole`, a two-candidate offset search, a
permitted wider offset search and a permitted contextual residual search. The
Gooo policy can select these declared transitions without further user input.
The body, intent and supplied expectations remain intact; failed direct inputs
may be added to the adaptive construction cases.

For the own-model arm, use `mixed.gooo`, `--model builtin`, and the two
`examples/search-refinement/mixed-*-cases.json` files. The model ranks the three
record choices after the integer activity has been constructed. Grammar
transitions are selected by the ordinary Gooo policy with no policy inference.

## Reproduce six runs

```sh
go build -trimpath -o /tmp/gooo-search-policy-study ./experiments/search-policy
/tmp/gooo-search-policy-study --compiler /path/to/gooo --out out/search-policy-study
```

The driver compares the legacy budget/case policy with source-declared
transitions, mixed deterministic and model construction, a two-round cap, and a
source with no permitted alternatives. The legacy policy uses the same scalar
source but never selects its alternatives. Each run saves native observations,
all source revisions, Gooo decisions and separate final evaluation.

[The six-run publication](../../publication/search-policy-20261008/README.md)
includes successful and partial runs, model observations and saved-program replay.

The initial budget is checked before constructing candidates or loading a model.
`--max-attempts` bounds the selected activity. Other graph activities keep their
own declared budgets. The search policy sees the current candidate-space counts
and next unvisited source setting. Its next attempt limit is bounded by the
caller and that setting's candidate cap. At most eight refinement rounds run;
repeated grammar/cap pairs are skipped. The selected program then replays
without new model predictions.

The supplied policy chooses the next declared setting after exhausting the
current list or attempt allowance. It may stop with unresolved expectations when
there is no permitted next setting or the round cap is reached. Existing policies
keep their input shape when `--search-policy` is omitted. The two adaptive feedback
inputs and two post-selection evaluation inputs are finite evidence. Model
training exposure is unknown, and no model training occurs in this experiment.
