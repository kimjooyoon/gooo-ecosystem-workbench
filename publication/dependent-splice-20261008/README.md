# Dependent source editing: first unseen-program observation

2026-10-08. One new Gooo program uses three dependent field decisions:
whether to edit, the replacement source under that decision, and the byte count
of the updated source. The unchanged graph QAT model was trained on filenames,
division and retry. Splice was absent from that training corpus.

The [plan](../../experiments/dependent-splice/PLAN.md) and source were committed
as `cd2d3d6`, then a newline-escaping correction as `7438d23`, before observing
the model on this task. The Go runner and independent oracle were committed as
`e2cb19a` before the first model run. Compiler `e6150a23` was built from a clean
checkout with Go 1.27.1. Exact compiler build settings and source/model hashes
are in [summary.json](summary.json).

## What happened

The fixed order required all eight candidates. The model ranked the complete
candidate fourth. Its first recommendation was wrong: it selected the improved
replacement and final-length expressions but left the edit decision false.
The program therefore never reached the useful replacement path.

| Attempt limit | Fixed: selection cases / fields | Model: selection cases / fields | Fixed: native cases / fields | Model: native cases / fields |
| --- | --- | --- | --- | --- |
| 1 | 2/8 · 9/24 | 2/8 · 9/24 | 321/542 · 963/1,626 | 321/542 · 963/1,626 |
| 2 | 2/8 · 15/24 | 2/8 · 9/24 | 321/542 · 1,184/1,626 | 321/542 · 963/1,626 |
| 4 | 5/8 · 21/24 | 8/8 · 24/24 | 321/542 · 1,405/1,626 | 542/542 · 1,626/1,626 |
| 8 | 8/8 · 24/24 | 8/8 · 24/24 | 542/542 · 1,626/1,626 | 542/542 · 1,626/1,626 |

Saved replay reproduced every actual output, including partial constructions.
The 542 unique native input tuples are disjoint from the eight source-selection
tuples. They contain 221 edits and 321 unchanged outcomes. Thus a program that
always preserves the source already satisfies 321/542 complete outputs. Report
that baseline and the field counts together; a high unchanged-case proportion
can hide missing editing behavior.

The model's order was `[6,4,2,7,0,5,3,1]`; fixed order was `[0,1,2,3,4,5,6,7]`.
In the eight-attempt model run only four candidates were needed. Each model
construction made one prediction. Native execution and saved replay made zero.
Four budgets are separate constructions of this same program, with the same
wording and candidate placement. These results do not establish broad transfer
or a calibrated probability of fulfilling an arbitrary request.

## Using the generated tool to edit Gooo

Both modes executed an input-only package request that changed a small Gooo
activity from `input + 1` to `input * 2`. The returned source was saved and
compiled, then checked on `-5`, `0`, and `9`, yielding `-10`, `0`, and `18`.
Each mode passed 3/3 checks. The edit request itself has no supplied expected
value and therefore carries observation counts, without a correctness score.
The independent study separately compares edits with its Go oracle.

The actual editing conditions and sequential updates live in
[recipes/splice.gooo](../../recipes/splice.gooo). Production Go transports the
request, checks output accounting, saves files and requests replay. The oracle
is only part of the experiment.

## Cost and limitations

Four local model constructions recorded prediction intervals of 62.416–97.416
microseconds and model setup of 0.214–0.537 milliseconds. Prediction includes
the model's context prediction call and local result handling. These are four
single observations across different attempt budgets, with no warmed-repeat
latency study. Resident tensors were 2,096 bytes, plus model metadata and other
runtime state; packed weights are 446 bytes. Whole-process RAM and host CPU
utilization changes were not measured in this study.

The end-to-end runs took about 5.7–6.0 seconds each, including compilation and
two passes over nine native input batches. They do not isolate model speed or
establish an end-to-end speed advantage. This study demonstrates fewer candidate
attempts on one new program and retains a case where the model is worse at the
two-attempt field budget.

The context graph preserves the earlier decision, conditional write and final
read. The current shared field judge scores field choices independently before
combining them. A next experiment can compare this with a chooser that receives
the earlier selected branch or scores the whole combination. Freeze new tasks
and evaluation cases before training on this now-observed program.

## Reproduce and inspect

See the [tool guide](../../examples/source-splice/README.md) for compiler and
command requirements. [evidence.tar.gz](evidence.tar.gz) contains the roster,
all source variants, all model predictions, attempts, generated code, native
outputs, saved replay and both edited-program executions. Partial outcomes are
retained. Digests are in `SHA256SUMS`. CI runs the same frozen experiment on Linux
and publishes its own full observations as a workflow artifact.

This integration exposed a compiler gap: package calls treated `len` as an
unresolved activity although body generation supported it. Compiler PR
[1378](https://github.com/kimjooyoon/meta-ontology-go/pull/1378) fixes that resolution
and tests nested calls, shadowing, field expressions and saved replay. Original
0.6.9-dev binaries require this source fix for the input-only Splice tool.
