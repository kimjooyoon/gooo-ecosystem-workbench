# Diagnose saved Gooo package executions

Date: 2026-10-08. Workbench source: `fcb1839c8e26196542b5e3eed3c8d651490b6341`.
The diagnostic compiler was built from `6fc331af1c4e2504e2d6e0cfe7222213ad45735b`.

Six [published package observations](https://github.com/kimjooyoon/meta-ontology-go/tree/6fc331af1c4e2504e2d6e0cfe7222213ad45735b/docs/research/workspace-continuation-20261008)
were each passed through the Gooo diagnostic program twice: deterministic
construction and construction ranked by the bundled tiny model. Those original
package executions used compiler `e411aafeaf674e61d6998197f6d80b94d155d0f1`.
This study diagnoses their saved values; it does not repeat their package generation.

| Original package stage, in both original routes | Native outputs | Diagnostic in both diagnostic modes | Proposed action |
| --- | --- | --- | --- |
| Checkpoint | 2/4 | `partial` | `expand-examples` |
| Continued | 4/4 | `complete` | `accept` |
| Saved replay | 4/4 | `complete` | `accept` |

All 12 invocations returned the expected classification and action for these six
inputs. The original JSON receipt hash is preserved in each `observation.json`;
input filenames are `<input_route>-<input_stage>.json` at the source link above.
Scalar text outputs use `activity_outputs`, so a record-field denominator cannot
hide their failures. The diagnostic source owns the classification and action.

The tiny model was used to construct the diagnostic program itself. Each model
invocation made one new ranking prediction, taking 13.084–17.208 microseconds.
The five declared construction cases matched 15/15 diagnostic fields in all runs.
Deterministic construction made zero predictions and returned the same diagnoses.
The recorded model calls belong to this diagnostic generation, separately from
any historical calls inside the input package receipt.

Whole-command wall time was 0.31–0.81 seconds. On this Darwin arm64 machine,
maximum resident memory was 81.00–83.95 MiB. The command includes compiler/build
and native execution work; these numbers do not isolate model memory. Raw
`command.time` files retain user/system CPU time. Host CPU utilization was not
sampled. Each route ran once with uncontrolled caches, so no speedup is inferred.

`summary.json` contains every row and output digest. Each row directory retains
the recounted observation, actual diagnostic, complete diagnostic construction
and runtime result, and process timing. `diagnostics.gooo` is the actual source
used for all runs. No model weights were trained or changed.

## Reproduce a diagnosis

Build the compiler and workbench at the source revisions above, then use a saved
input from the linked observation directory:

```sh
workbench diagnose --compiler /path/to/gooo \
  --input /path/to/model-checkpoint.json --model builtin \
  --out out/checkpoint-diagnosis
```

Omit `--model builtin` for deterministic construction. Every output directory
must be new. The diagnostic proposes an action; it does not issue a resume or
modify the original program. The current action rules use observed counts and
type rejections. Choosing resume from remaining candidate history is further
language work. Finite matching observations describe only the supplied cases.
