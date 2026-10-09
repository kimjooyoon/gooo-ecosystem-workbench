# Bound consumer feedback, 2026-10-09

Local actual runs of the [predeclared experiment](../../examples/assembly-graph-feedback/PLAN.md),
using public Gooo 0.6.22-dev source d3b44fc6, Go 1.27.2 and decision runtime 0.2.26.
The graph has Describe → Present, plus fixed pure helper Decorate. Source-local
expectations name Describe; adaptive and final expectations name Present.

| Origin / new construction | Origin fields | Caller rows added | Fresh rounds | Repeated program attempts | New construction model calls | Final fields | New final calls |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Fixed / fixed | 2/6 | 0, 1 | 4 | 15 | 0 | 6/6 | 0 |
| Fixed / own model | 2/6 | 0, 1 | 4 | 13 | 4 | 6/6 | 0 |
| Own model / fixed | 4/6 | 0 | 4 | 15 | 0 | 6/6 | 0 |
| Own model / own model | 4/6 | 0 | 4 | 13 | 4 | 6/6 | 0 |

All four routes preserved the origin's eleven referenced artifact bytes. Final
evaluation used two root tuples outside the construction set, after selection.
Training exposure is unknown. Counts concern one small program and its supplied
cases. Program attempts include repeated fresh-round work; they are not unique
candidates or a latency measurement. Local runs overlapped the full race suite.

[observations.json](observations.json) is a public projection emitted directly by
the [Go observer](../../experiments/assembly-graph-observe/main.go). It retains
original construction-record hashes and byte counts, per-round observations,
original caller indices and inference counts. Original native output files remain
in the explicitly supplied local output directory. The projection excludes local
absolute paths; source and expectation files are in the example directory.

Reproduce from the repository root, with a new output directory:

```sh
mkdir -p out
go run ./experiments/assembly-graph-observe --compiler gooo --out out/graph-observation \
  > out/graph-observation-public.json
```

The observer creates graph-observation and refuses an existing directory.
Timing fields are single observations. Hashes and elapsed times vary per run.

[compiler-build.json](compiler-build.json) identifies the actual compiler.
[tdd-red.log](tdd-red.log) preserves the first missing-API compile failure.
[native-focused.log](native-focused.log) records the four native routes passing.
Unit mutation checks also reject extra roots, wrong identities, another assembly
body, search/fill bodies, prepared assembly helpers and absent finite counters.

[full-native-race.log](full-native-race.log) records the complete native workbench
race suite passing (root package 298.327s). The observer was added during that
run and checked separately; final `go vet ./...` passed and
[unit-race.log](unit-race.log) covers the final graph mutations and help tests.
The public PR CI also executes the new Go observer package and an explicit
two-command CLI graph route against the unchanged pinned compiler source.
