# Can the model input distinguish the supplied choices?

`feature-audit` compares the actual float32 feature arrays of the existing
shared-field model. Go owns projection and exact grouping; the next action is
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

## Output and implementation

`feature-audit.json` contains the groups, exact-array hashes, accepted-mask counts,
maximum compatible rows, optional model predictions and Gooo's assessment.
`input.json` preserves the supplied dataset. `assessment/` retains the Gooo source,
input cases, generated code and native composition for replay. No weights are
updated. The original model hashes are recorded when inference is requested.

The feature contract is `triple_record_field_context_v1_shared_v1`, from runtime
`v0.2.25-experimental`. Other versions are rejected. Each row has a unique `id`,
nonempty `family`, exactly three source choices and one or more distinct
`accepted_masks` in 0..7. The file includes `schema`, `feature_version` and a
`label_source` reference. `filename-order.json` is a complete example.

The Gooo policy returns one of:

- `representation-collision`: preserve the missing source distinctions.
- `input-consistent`: measure a chooser on these inputs.
- `chooser-gap`: inspect candidate balance and source-family training splits.
- `observed-fit`: evaluate separate programs.
- `invalid-counts`: recount the observations.

These are recorded next actions; the command does not launch training.
