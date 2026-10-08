# Can the model input distinguish the supplied choices?

`feature-audit` compares the actual float32 feature arrays of the expression,
source-origin and ordered-value-graph shared-field contracts. Go owns projection and exact grouping; the next action is
computed by [a Gooo activity](../../recipes/feature-audit.gooo).

```sh
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-order.json --model builtin --out out/input-audit
```

Gooo 0.6.8-dev can execute the assessment. The compiler source used to construct
the filename observations is recorded in the linked original study. Omit
`--model` to inspect feature distinguishability without loading weights.

## The included observation

The dataset has eight front/back arrangements of three choices from **one**
filename classifier. Field names, expressions and Korean/English intent are
copied from the actual construction context. The accepted masks refer to the
selected bodies in [the published order study](https://github.com/kimjooyoon/meta-ontology-go/wiki/Text-Operations-and-Choice-Order).
The source and twelve native inputs are unchanged across arrangements.

| Quantity | Observed |
| --- | ---: |
| Source families | 1 |
| Candidate arrangements | 8 |
| Different complete feature arrays | 2 |
| Groups requiring incompatible selections | 2 |
| Maximum compatible rows for a deterministic feature-only chooser | 2/8 |
| Existing model's accepted first choices | 1/8 |

The projection represents `suffix && visible` and `suffix || visible` identically.
It also represents `input` and `stem` identically. Swapping those alternatives
changes the desired output while leaving the full model feature array unchanged.
Changing the byte-length field's `0`/`bytes` order creates the two remaining arrays.
Training the same input projection cannot recover distinctions removed before
inference. This observation concerns this versioned feature contract and dataset.

## How the bound is counted

Rows with bit-identical arrays form one group. For each of the eight possible
masks, count how many rows accept that mask. The largest count is the maximum
number of rows one deterministic prediction can satisfy in the group. Add those
maxima over all groups. The input can list several accepted masks for a row when
several candidates are valid; overlapping accepted sets need not be a conflict.

The supplied labels are explicit assumptions of this calculation. The tool
does not establish their semantic correctness, all-input behavior or train/test
independence. Repeated source families are counted and should stay together when
constructing later training splits. This is a row-weighted empirical bound, not
the general accuracy ceiling of Gooo or language models.

## Follow values through source helpers

The origin fixture uses the same eight arrangements, with the compiler's actual
ancestor counts through `HasSuffix`, `HasPrefix`, `StripSuffix` and `ByteLength`:

```sh
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-origin-order.json --out out/origin-audit
```

Its source contexts were exported from compiler `246fdf5b0d40a1d1e148560f8389cdfb364da998`.
The complete exports, original Gooo variants and reproduction tool are retained
in [the helper-flow observation](https://github.com/kimjooyoon/meta-ontology-go/wiki/Helper-Value-Flow).
Each `origin_choices` item is the decoded corresponding `context.parts` item.
`accepted_masks` remains the prior study's label assumption `7 - order`; this
audit does not execute those candidates or establish new labels.

| Feature contract | Source families | Arrangements | Different arrays | Maximum compatible rows |
| --- | ---: | ---: | ---: | ---: |
| Expression v1 | 1 | 8 | 2 | 2/8 |
| Source origin v2 | 1 | 8 | 4 | 4/8 |

Origin v2 separates the original name from the computed stem. AND/OR still
collide, leaving four groups of two incompatible rows. The Gooo assessment
therefore requests `preserve-distinguishing-source-facts` for both inputs.
These are representation measurements. No v2 weights were trained or measured.
The bundled model is v1 and is rejected when requested with v2 input.

## Preserve operators and ordered values

The graph fixture retains the compiler's ordered source graph, including helper
values, canonical literals, input positions/types and conditional joins:

```sh
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-graph-order.json --out out/graph-audit
```

All eight arrangements produce different complete arrays; the empirical label
bound is 8/8 for these rows. The unchanged Gooo assessment returns
`input-consistent` and `evaluate-chooser`: no conflict was found in this supplied
dataset, and the actual chooser still needs measurement. Model calls are zero.
At that input-only stage no v3 weights were trained. The subsequent
[Go training study](../../publication/graph-chooser-20261008/README.md) publishes
trained v3 artifacts and their measured limitations. The bundled v1 model is rejected for v3.
This one-family input observation does not establish trained quality or rule
out hashed feature collisions on other sources.

The fixture is imported from compiler `dea641f709b37bfd3f8655a242bebaa6cdc0d18b`.
Its source-only graph exports and separately executed finite labels are in
[the published observation](https://github.com/kimjooyoon/meta-ontology-go/wiki/Source-Graph-Model-Input).
The importer checks matching source/contract digests, export call counts and
complete finite selection status. The original observation separately checked
all raw expected/actual fields. Recreate the fixture from that observation
directory using a new destination:

```sh
go run ./experiments/graph-feature-fixture \
  --observation /path/to/source-graph-context-20261008 --out /path/to/new-fixture.json
```

Gooo 0.6.8-dev can still execute the assessment recipe: graph projection runs in
this Go tool through SDK v0.2.26-experimental. Exporting new v3 graphs from Gooo
requires the newer compiler source described above.

## Output and implementation

`feature-audit.json` contains the groups, exact-array hashes, accepted-mask counts,
maximum compatible rows, optional model predictions and Gooo's assessment.
`input.json` preserves the supplied dataset. `assessment/` retains the Gooo source,
input cases, generated code and native composition for replay. No weights are
updated. The original model hashes are recorded when inference is requested.

The supported contracts are `triple_record_field_context_v1_shared_v1`,
`triple_record_field_flow_v2_shared_v1` and `triple_record_value_graph_v3_shared_v1`,
from runtime `v0.2.26-experimental`.
Other versions are rejected. Each row has a unique `id`,
nonempty `family`, exactly three source choices and one or more distinct
`accepted_masks` in 0..7. The file includes `schema`, `feature_version` and a
`label_source` reference. Expression v1 rows use `choices`; origin v2 rows use
`origin_choices` with the SDK's two 16-slot `origins` arrays per choice. Graph v3
rows use `graph`, the SDK's `RecordGraphInput` with three choices and 1..512 nodes.
Mixed or
mismatched rows are rejected. An optional model must declare the same contract,
even though all versions use 768 floats. Changing accepted-mask labels never
changes the model input arrays. All three filename JSON files are complete examples.

The Gooo policy returns one of:

- `representation-collision`: preserve the missing source distinctions.
- `input-consistent`: measure a chooser on these inputs.
- `chooser-gap`: inspect candidate balance and source-family training splits.
- `observed-fit`: evaluate separate programs.
- `invalid-counts`: recount the observations.

These are recorded next actions; the command does not launch training.
