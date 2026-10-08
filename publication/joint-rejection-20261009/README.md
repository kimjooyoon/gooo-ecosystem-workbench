# Reading rejected expressions and continuing Gooo construction

2026-10-09 KST. The compiler candidate is clean source
`ae3890c241575d6bbbe2a9d730fd015d36b97ccd`, using Go 1.27.1 and decision
runtime v0.2.26-experimental. It reports development version 0.6.12 but contains
changes newer than that public release archive. The reader was a working-tree
implementation during this local run; its later commit is not a historical build
identity for the observation.

`joint-construction/v3` adds explicit rejected combinations. A local integer
expression may fail typechecking or pure evaluation before a caller can execute.
The reader keeps its rejection separate, verifies the unscored shape and counted
prefix, and reports both `rejected_attempts` and `native_program_attempts`.
Compiler replay is responsible for rederiving source-bound errors; this reader
does not independently establish source provenance.

## Actual automatic loop

The unchanged own graph QAT model ranks the record choices in the compiler's
frozen `examples/caller-search-rejection/mixed-model.gooo.fixture`. Integer
expression order remains deterministic. Starting with only caller `0 -> 0`, the
Gooo feedback rule copies two failing rows and their original expectations into
the next construction. No weights or expectations were changed.

- Seven rounds with budgets 1, 1, 2, 4, 8, 16 and 32.
- 49 total combination attempts, including repeated work across rounds.
- The last round tries 17 combinations: eight locally rejected, nine natively
  executed programs. Each native program attempt retains two runtime executions.
- Final separate evaluation: 4/4, including a large exact-integer input.
- Final saved replay: zero new model calls. Its four roots do not overlap the
  consumed caller roots. Model-training independence is not established.

The original `joint-loop.json` and full `loop.tar.gz` retain each round, Gooo
policies, copied source/cases, feedback selection and final replay. This is one
program run through an automatic loop, not seven different experiments.
Compiler-side direct fixed/model observations and the original stopping failure
are in the [compiler research directory](https://github.com/kimjooyoon/meta-ontology-go/tree/agent/joint-search-rejection-20261009/docs/research/caller-search-rejection-20261009).

## Validation and CI

The local full workbench race suite passed with public Gooo 0.6.12, including
native Gooo diagnostics on the retained v3 fixtures. Prefix/version regressions
added afterward were checked separately. CI pins the newer compiler source above
and runs a fresh model-driven rejection loop as well as all existing examples.
Local success is distinct from the final CI result.

The original four fixture files under `examples/joint-diagnostics` cover scalar
completion, its partial result, mixed model completion and saved replay. Their
unexecuted callers have no accuracy score. Native execution/toolchain failures,
cancellation and source reconstruction failures still terminate construction.
CPU utilization and controlled latency comparisons were not measured.
