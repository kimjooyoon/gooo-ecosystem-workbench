# Three program families for graph-choice experiments

The sources implement filename classification, signed integer division/remainder
and bounded retry decisions in Gooo. They were copied from compiler source
`e0d046503939883330eb37f998a2e0e8fc5154e5`; the generator keeps finite cases and
budgets unchanged while swapping the first/second choices and rendering Korean,
English or mixed intent. Original semantic IDs remain intact.

This produces **3 program families × 8 arrangements × 3 intent forms = 72 rows**.
They are repeated forms of three programs. The words are manually authored
equivalent intentions; changing their language does not create a new behavior.

## Export with the actual compiler

Use Go 1.27.1 and a clean compiler build supporting
`triple_record_value_graph_v3_shared_v1`. Its source SHA must match the argument.

```sh
go run ./experiments/graph-choice-corpus --compiler /path/to/gooo \
  --expected-compiler e0d046503939883330eb37f998a2e0e8fc5154e5 --out out/graph-corpus
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input out/graph-corpus/audit-input.json --out out/graph-corpus-audit
```

For each row, source-only graph export happens before finite candidate execution.
Both raw outputs and the exact Gooo source are saved. The collector binds their
source and assembly-contract digests. Context export must perform zero model
calls and zero candidate evaluations. The finite run must independently reach
all declared cases/fields with zero model calls. Every unswapped source checks
all eight candidates and establishes a unique complete finite choice. Its other
arrangements must select the correspondingly swapped combination.

The finite labels describe 5 filename cases, 4 division cases and 5 retry cases.
They do not include a separate native holdout. No model is trained here.

## Current observation and next learning split

The [recorded input](../feature-audit/three-family-graph.json) produces 72 distinct
full float32 arrays with SDK v0.2.26-experimental. Gooo's existing assessment
returns `input-consistent` / `evaluate-chooser`. This checks whether the supplied
rows remain distinguishable before measuring a learned chooser.

For a later learning experiment, hold out one **whole family** at a time: 48
training rows and 24 evaluation rows, with all arrangements and languages of a
family staying together. Report all three folds separately. Choosing hyperparameters
after viewing those folds would turn them into development observations and
require new program families for an untouched evaluation.

Intent sensitivity needs an additional experiment in which the same source and
alternatives receive different valid intentions and different expected behavior.
These equivalent Korean/English wordings alone cannot establish that a model
follows intention. New graph weights, native execution on unseen cases, finite
completion under a fixed attempt budget and prediction cost remain subsequent
measurements.
