# Actual-input construction observations

The `observed-*.json.gz` files preserve complete, unmodified compiler receipts.
They were emitted by a clean Go 1.27.2 build of compiler source
`92c1b7f860bd7d72fe6a06c7bd5d6f727f43242b` ([PR 1406](https://github.com/kimjooyoon/meta-ontology-go/pull/1406)).

- `observed-fresh`: the three-package example, caller-labelled construction,
  48-attempt budget and the unchanged 2,096-byte tensor graph model. Selection
  completed after 41 attempts; the four subsequent actual inputs have no oracle.
- `observed-replay`: that same construction re-executed on actual inputs with
  zero new model calls. Construction history remains identical to `observed-fresh`.
- `observed-partial`: the original two-package example with a five-attempt
  budget. The selected construction remains partial; four actual inputs are unscored.
- `observed-fault`: a caller returns `plan.next / (input.limit - input.used)`.
  Construction uses `{used:0,limit:8} -> 0`. The actual input `{used:8,limit:8}`
  has no expected answer and produces a native `ZERO_DIVISOR` observation.

The original sources, package image, construction cases, historical attempts,
large integers and native outcomes are retained inside each receipt. The model
metadata digest is `3c68205a660695103712e2115ce90accd298b406dced8ad5ad5c9b84bab0a202`.
This is an existing all-data demonstration model on a related example.

`ReadSnapshot` recounts supplied evidence. It does not independently execute the
original program or verify the external input file. `Diagnose` runs the Gooo
`joint-next` recipe on those counts, preserving 0/0 for unlabelled evaluation and
the distinct scores from earlier construction. The native policy tests exercise
normal, partial and faulted observations using the public 0.6.17 compiler.
