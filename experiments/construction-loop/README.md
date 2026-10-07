# Follow a Gooo continuation proposal

This Go experiment starts from a saved package execution. Workbench executes
`next-steps.gooo` to propose an action for each recorded construction. When Gooo
returns `resume-candidates`, the runner calls `gooo package resume` with the
explicit workspace, cases and policy supplied on the command line. It then
diagnoses the new receipt. No source file or model weight is changed.

```sh
go run ./experiments/construction-loop \
  --compiler /path/to/gooo --workbench /path/to/workbench \
  --workspace /path/to/meta-ontology-go/examples/called-body-construction/gooo.workspace.json \
  --cases /path/to/meta-ontology-go/examples/called-body-construction/cases.json \
  --policy /path/to/meta-ontology-go/examples/package-assembly-policy/gooo.workspace.json \
  --receipt /path/to/model-checkpoint.json \
  --model builtin --out out/construction-loop
```

The compiler must include `package resume` (compiler PR1353). Workbench must
include the construction-next-step diagnostic. The initial checkpoint example
and explicit policies are available in the compiler's called-body-construction
example. Omit `--model` to construct the basic diagnostic deterministically;
the next-step Gooo program itself uses no inference.

The default two diagnostic rounds allow at most one resume. `--rounds` accepts
1..16; the final round only observes. `loop.json` records the action sequence and
why dispatch stopped. `no-resume-proposal` can mean a completed finite observation,
exhausted choices, missing information or a case gap: inspect the actual actions.
Every diagnosis and consumed/produced receipt remains in the output directory.

The Go code dispatches the named operation. Gooo source owns the rules that
select it; the compiler revalidates the saved program before resuming. The driver
does not expand choices, rewrite cases, resolve unknown history, or train a model.
