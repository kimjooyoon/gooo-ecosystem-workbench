# Rejected source-fill observations

Compiler source: clean `7c9d8d911b7b9f6c95b7f417fd3715fb92f9f051`, Go 1.27.1.
See `build.json`. Version string 0.6.14-dev describes the development line;
the published 0.6.14 binary predates this change.

`summary.json` is reproduced by `go run ./experiments/fill-rejection-observe`.
Six compressed CLI records in `examples/caller-fill-rejection` retain four fresh
constructions and two saved replays. Original expectation files remain intact.
The Go reader recounts exact JSON numbers, including values above 2^53.

The unchanged own graph chooser made one prediction in `mixed-model`; local
fill selection remained deterministic. The 40-to-5 attempt difference includes
16 versus 2 rejected combinations, leaving 24 versus 3 native combinations.
All complete runs match four original final expectations. The partial budget
record remains 1/4. No new training or general accuracy claim.

`loop-observations.tar.gz` retains the actual Gooo feedback loop. Budgets were
1, 1, 2, 4 and 8; attempts were 1, 1, 2, 4 and 5 (13 total, counting repeated
work). It added the original boundary counterexample and ended with a separate
4/4 evaluation. The archive includes the inputs, generated Gooo/Go, feedback
decisions and results. `workbench-loop.json` is its compact retained summary.

The reader validates internal counts and original case values. Compiler replay
reconstructs source and plan identity; a standalone reader does not replace it.
The all-data demonstration model is not a held-out evaluation model.
