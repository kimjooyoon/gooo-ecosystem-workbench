# Gooo proposes the next construction step, Go dispatches it

Date: 2026-10-08. Workbench source `a5a7b9c81f95e7233fb6899dc538ed69e4c60df7`;
compiler source `6fc331af1c4e2504e2d6e0cfe7222213ad45735b`.

The [Go experiment runner](../../experiments/construction-loop) diagnosed two
saved checkpoints, followed the Gooo `resume-candidates` proposal, then diagnosed
the fresh native result. All commands ran without further user input. The explicit
workspace, cases and continuation policy were supplied at the start.

| Route | Initial native cases | First Gooo action | Fresh native cases | Next Gooo action | New model calls in resume |
| --- | --- | --- | --- | --- | ---: |
| Deterministic | 2/4 | `resume-candidates` | 4/4 | `observe-new-inputs` | 0 |
| Own tiny model | 2/4 | `resume-candidates` | 4/4 | `observe-new-inputs` | 0 |

The initial helper matched 3/5 construction cases after one attempt. Its recorded
budget was eight and its ranking contained eight candidates. Gooo used those
distinct counts to propose continuation. The resumed helper matched 5/5 cases;
the compiler added three attempts in the deterministic route and one in the model
route. Historical ordering and the original attempt budget were retained.

The model route made one fresh tiny-model prediction for each of its two basic
diagnostic programs. `next-steps.gooo` executed ordinary declared conditions with
zero inference; package resume also made zero new predictions. Historical model
calls in the initial receipt belong to the earlier generation. This distinction
is retained in `summary.json` and the full generated diagnostic results.

The separate one-round observation kept the `resume-candidates` proposal but
dispatched no resume, recorded `round-limit`, and retained the initial receipt.
The runner stops on proposals it does not implement; stopping alone does not mean
the program is complete.

## Evidence and reproduction

Each route directory contains the exact initial receipt, fresh resumed receipt,
both diagnostic results, both Gooo next-step results, and the dispatch sequence.
The initial receipts came from the compiler's
[workspace continuation study](https://github.com/kimjooyoon/meta-ontology-go/tree/6fc331af1c4e2504e2d6e0cfe7222213ad45735b/docs/research/workspace-continuation-20261008)
and were originally produced at `e411aafeaf674e61d6998197f6d80b94d155d0f1`.
`summary.json` includes their hashes, separate construction and native counts,
model timings and whole-loop wall/CPU/maximum-RSS observations.

Use the runner instructions with one of these `initial.json` receipts and the
compiler's `examples/called-body-construction` workspace/cases plus
`examples/package-assembly-policy/gooo.workspace.json` continuation policy.
Set the working directory to the compiler checkout when using those relative paths.

The independent finite test bank covers 15 next-step states, including exhausted
budget, exhausted candidates, a native/construction case gap, missing budget,
unobserved results and inconsistent counts. Its 15 Next outputs contain 45
expected fields; the 15 input echoes contribute another 150 fields. These
denominators remain separate from this study's four native application cases.

These are two local runs over one small workspace. Caches and host activity were
uncontrolled, and host utilization was not sampled. No speedup or generalization
is inferred. The experiment does not add choices, rewrite expectations, identify
which helper caused a native mismatch, or train weights. Gooo owns the next-step
rules; Go dispatches the explicit operation, and the compiler revalidates source
and history before executing it.
