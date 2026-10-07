# Source budget without an assembly policy

This example keeps eight possible record bodies and varies the source's
`attempts` declaration through 1, 3, 8 and 16. Gooo records the declared budget,
attempted candidates and available candidates separately. The workbench reads
the observation and executes `recipes/next-steps.gooo` to choose the next action.

The source and cases come from the compiler's `record-field-updates` example at
`cdb60e90f911d53d7c02790579e5f72941a2b71a`. Five inputs are also construction
cases; two additional runtime inputs check a quoted multiline title and a
retained record. They are finite examples, with unknown model-training exposure.

Build the compiler revision pinned in CI, then from the workbench repository:

```sh
go build -o /tmp/gooo-source-budget ./experiments/source-budget
/tmp/gooo-source-budget --compiler /path/to/gooo \
  --model models/shared-qat/model.json --out out/source-budget
```

Omit `--model` for the deterministic arm only. Each run keeps its Gooo source,
native result, construction receipt and Gooo diagnostic outputs. Diagnostics
make zero new model calls; a model arm applies only to construction ordering.
The runner requires a fresh output directory and bounds each command to two
minutes. The test suite also executes all four deterministic budgets in CI.
