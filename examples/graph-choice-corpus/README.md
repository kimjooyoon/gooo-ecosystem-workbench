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

## Change the requested behavior while keeping the code choices

`--intent-contrasts` adds eight output policies per source. Each field can request
either existing expression; changing the request changes its intent text and
finite expectations while preserving both alternatives and the program body.
Independent Go oracles calculate the expectations and must agree with every
original manual case before producing a contrast. The compiler then separately
observes each finite label. The arrangement with accepted mask seven exhausts
all eight candidates to establish a unique complete label for each policy.

```sh
go run ./experiments/graph-choice-corpus --compiler /path/to/gooo \
  --expected-compiler e0d046503939883330eb37f998a2e0e8fc5154e5 \
  --intent-contrasts --out out/intent-contrasts
go run ./experiments/graph-choice-corpus --compiler /path/to/gooo \
  --expected-compiler e0d046503939883330eb37f998a2e0e8fc5154e5 \
  --native-contrasts out/intent-contrasts --out out/intent-native
```

This design has **3 families × 8 requested policies × 8 arrangements × 3 wording
forms = 576 rows**. The native command checks every requested policy with mixed
wording and one arrangement: 24 constructions, 128 finite runtime cases, followed
by saved replay. The 16 distinct input tuples are disjoint from source selection
cases and reused across policies. It independently compares exact native inputs
and outputs with the Go oracles, including int64 boundaries and UTF-8 byte lengths.

Both modes publish deterministic `raw-evidence.tar.gz` archives. The contrast
collector also writes `intent-ablated-input.json`; only intent strings are replaced,
so source nodes, roots, choices and labels remain identical. Audit either input
with the `feature-audit` command above. CI separately rechecks the compact recorded
fixtures, full array collisions and the Gooo assessment. It does not regenerate
the corpus with its older pinned compiler.

[Actual observations](../../publication/intent-contrasts-20261008/README.md) report
576 full arrays and 24 ablated arrays, with an ablated maximum of 72/576 compatible
labels. This is a representation bound on the supplied rows. New graph weights,
learned intent sensitivity and prediction cost remain unmeasured.

A later three-fold learning study must keep each whole family together: **384
training and 192 evaluation rows** per fold. All wording, policies and arrangements
of the held-out family stay out of training. These authored templates do not measure
understanding arbitrary new Korean/English phrasing. Report first selection, finite
completion within the attempt budget, field accuracy and intent-flip consistency
separately; keep native runtime inputs out of candidate selection.
