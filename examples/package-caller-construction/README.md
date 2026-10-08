# Let Gooo continue construction across packages

The retry application imports its budget calculation from another Gooo package.
The initial example uses zero of eight attempts; several implementations satisfy
it. An adaptive caller case reaches the boundary at eight and reveals the wrong
choice. Gooo's `joint-feedback.gooo` preserves that entire original row for the
next construction; `joint-next.gooo` chooses the subsequent budgets.

This example needs the compiler's development `package construct` command from
[PR 1402](https://github.com/kimjooyoon/meta-ontology-go/pull/1402). The published
0.6.16 compiler supports the older single-source examples. Build the revision
pinned by this repository's CI for package construction.

```sh
go run ./cmd/workbench construct --compiler /path/to/development/gooo \
  --workspace examples/package-caller-construction/gooo.workspace.json \
  --construction-cases examples/package-caller-construction/initial-cases.json \
  --evaluation-cases examples/package-caller-construction/evaluation-cases.json \
  --holdout-cases examples/package-caller-construction/holdout-cases.json \
  --max-program-budget 8 --max-rounds 5 --out out/package-loop
```

The observed progression is budgets **1, 1, 2, 4, 8**, using **14 total attempts**
across five fresh rounds. The last round executes six candidates, retaining two
local rejections and one native arithmetic fault. Final holdout matches **4/4**
with zero new inference. Earlier repeated attempts remain in the total.

`--evaluation-cases` is adaptive feedback and can influence selection. Only
`--holdout-cases` waits until selection stops. `app/retry:Main` keys and integers
above 2^53 are retained exactly when an entire failing row is added.

## Keep the original package program

Before construction, the workbench copies the original manifest and only its
listed source files into `out/package-loop/workspace/`. All rounds and final
replay use this copy. The workspace owns its entry; `--workspace` excludes
`--source` and `--entry`. Output must be a new directory.

Each round retains the full package receipt and selected Gooo/Go exports under
`round-N/`. `round-N/package-construction.json` is the replay input. Gooo policy
sources, their inputs and native outputs are kept in the diagnosis and feedback
folders. A proposal is counted as consumed only after the next round succeeds.
A limit or process failure leaves earlier rounds and the current failure record.

Saved package construction rechecks every attempted program against the current
copied workspace. Reading a receipt with `diagnose` recounts its observations;
it does not independently rerun the original native program.

## Use the existing compact model

The mixed workspace imports both the budget library and the record-choice
library. It has 48 source combinations. Use the unchanged demonstration model:

```sh
go run ./cmd/workbench construct --compiler /path/to/development/gooo \
  --workspace examples/package-caller-construction/mixed.workspace.json \
  --construction-cases examples/package-caller-construction/mixed-initial-cases.json \
  --evaluation-cases examples/package-caller-construction/mixed-construction-cases.json \
  --holdout-cases examples/package-caller-construction/mixed-evaluation-cases.json \
  --model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json \
  --max-program-budget 64 --max-rounds 8 --out out/package-model-loop
```

The model orders record choices at the beginning of each fresh round. Gooo owns
which observed failures to add and which program budget to try next. Source
fills follow the declared order. Final replay needs no model files. These are
related finite examples; the model's transfer performance remains a separate
measurement.
