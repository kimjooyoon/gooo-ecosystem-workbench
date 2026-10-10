# Original joint-path learning and native comparison

Clean producer`e2e259a966473be21250a01b99644f646992c6c4`, Go1.27.2, own Go CPU
trainer and decision SDK0.2.26. Two fixed arms continued the original own FP32
checkpoint with the same architecture and update schedule. Training sources,
label rows, consumed joint targets, losses, four exported artifacts and timing
are under`training/`. Source-family count for the new joint target is one.

The [model card](../../models/joint-path-20261010) reports the native improvement
and the development-label regression together. The fixed label-only control
also used the additional training schedule; it did not satisfy the new negative
caller cases. Original published weights are retained as a third reference.
No tuning/retry followed these results.

`native/` preserves the public0.6.24/source366 compiler identity, unchanged Gooo
source, original construction case,11 new caller cases, three preflights,
construction outputs and saved replays. All three runs made3 initial predictions
and one program attempt. Results: previous6/11, control6/11, joint11/11. Their
saved outputs match exactly, with zero new inference. Model/input/source hashes
match the original training and preflight, including exact large int64 values.

The11 evaluation tuples are disjoint from the nine consumed target inputs,
construction inputs and the prior five-case feedback evaluation. They are new
values for the same authored function, not a held-out source family. Ranking
probability mass over the accepted set is not calibrated correctness probability.

After that run, a separate AST audit found all three generated conditions to be
`0 < input`, whereas the comparison's source hint requests a negative predicate.
The authored truth-table probes-1,0,1 match in1/3 cases for every arm. Those
post-hoc probes are a local hint diagnostic, not new held-out function tests.
The compiler had not enforced that natural-language hint as a formal constraint.
`native/hint-audit.json.gz` retains the finding;`hint-audit/main.go` reads the
unchanged generated source without invoking the model or compiler. Functional
case compatibility and local intent agreement need separate measurements.

The original readonly recount produced`native-summary.json`. Its helper accepts
these compressed files directly; it does not compile or retrain anything:

```sh
GOWORK=off GOTOOLCHAIN=go1.27.2 go run \
  publication/joint-path-learning-20261010/verify-native.go \
  publication/joint-path-learning-20261010/native \
  publication/joint-path-learning-20261010/training
```

All gzip-n files were compared after decompression to the original bytes.
`FILES.sha256` covers the retained records. Native generated Go is unchanged
and archived with its original directory layout. The final checkpoint under
`models/` is byte-identical to this study's joint QAT export.

Focused race tests verify analytic gradients against finite differences,
equivalent candidate relabeling, whole-mask versus marginal loss, pinned corpus
integrity, deterministic updates and cancellation. The path-chooser package and
vet passed before the original study. Compiler inference ABI and model dimensions
are unchanged. HF publication has not occurred in this study.
