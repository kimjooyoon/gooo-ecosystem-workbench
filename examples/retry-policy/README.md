# A retry decision written in Gooo

`PlanRetry` returns whether another attempt is allowed, its delay in milliseconds,
and a reason. The conditions, local variables and bounded arithmetic live in
[source.gooo](source.gooo). The three result fields have two permitted expressions
each. The optional compact model ranks those eight combinations; Gooo checks the
five declared construction cases before saving the selected program.

## Inputs and behavior

| Input | Meaning |
| --- | --- |
| `input0` | The preceding operation succeeded |
| `input1` | The caller classifies the failure as transient |
| `input2` | Number of attempts already used |
| `input3` | Maximum total attempts |
| `input4` | Previous delay, in milliseconds |
| `input5` | Maximum delay, in milliseconds |

Negative counters or delays return `invalid-input`. Valid completed work returns
`completed`; other terminal reasons are `permanent-failure` and `attempt-limit`.
Otherwise the next delay doubles, capped at the supplied maximum. A previous
delay of zero starts at one millisecond when the cap permits it. A zero cap
explicitly permits immediate retry. Multiplication is guarded so an int64 value
cannot overflow while calculating the capped delay.

The caller owns the attempt count, failure classification, actual waiting and
execution. Reusing this function does not schedule a task by itself. There is no
random jitter in this policy.

## Construct, then reuse

Use the compiler revision pinned in this repository's CI or a later compatible
revision. From the repository root:

```sh
gooo body-compose --source examples/retry-policy/source.gooo \
  --cases examples/retry-policy/cases.json --out out/retry-fixed

gooo body-compose --source examples/retry-policy/source.gooo \
  --cases examples/retry-policy/cases.json \
  --model models/shared-qat/model.json --out out/retry-model

gooo body-compose --source out/retry-model/original.gooo \
  --composition out/retry-model/composition.json \
  --cases examples/retry-policy/cases.json
```

The first two commands assemble and compile the program. The last executes the
saved selection with no new model call. To use new input tuples, provide another
cases file with the same six input names. Model-free construction uses a fixed
candidate order. The bundled model is the existing independently trained Gooo
model; this example performs no download or training.

## What is checked

The twelve supplied examples cover each terminal reason, precedence of invalid
inputs, zero delays, odd caps and int64 limits. Their input tuples differ from the
five construction tuples. The native regression additionally evaluates 200
distinct tuples per mode against a Go oracle using arbitrary-precision arithmetic.
It splits those evaluations into four files within the compiler's 32 KiB bound,
and reuses one saved program in each mode. Field matches, whole-plan matches,
construction model calls and new execution calls are checked separately.

These finite checks describe this declared policy and its result assembly. The
caller still needs to decide which operations and failures permit retries.

[Published native observations](../../publication/retry-policy-20261008/README.md)
retain the selected sources, candidate counts, model call and saved execution.
