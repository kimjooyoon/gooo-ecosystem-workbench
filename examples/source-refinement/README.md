# Gooo selects and runs its next source revision

`workbench refine` connects the Gooo next-step program to the compiler's existing
`body-refine` operation. The caller supplies a constructing activity, expected
feedback outputs, a Gooo revision policy, and finite bounds. From there the
command constructs, diagnoses, dispatches a source revision when proposed,
replays the retained program, and diagnoses the result.

The source, policy and cases come from the compiler's `assembly-feedback` example
at `cdb60e90f911d53d7c02790579e5f72941a2b71a`.

```sh
go run ./cmd/workbench refine --compiler /path/to/gooo \
  --source examples/source-refinement/source.gooo --activity Select \
  --cases examples/source-refinement/feedback-cases.json \
  --evaluation-cases examples/source-refinement/evaluation-cases.json \
  --policy examples/source-refinement/policy.gooo \
  --max-attempts 8 --max-rounds 4 --out out/refined
```

Add `--model builtin` to use this repository's own compact record model for
construction ordering. The Gooo diagnostic and revision policies execute
deterministically. Policy expressions select budget changes, whether to promote
explicit direct-input counterexamples, and which observed round to retain.

| Initial Gooo action | Dispatch |
| --- | --- |
| `raise-attempt-budget` | Run the supplied Gooo source revision policy |
| `add-runtime-cases-to-construction` | Run the same policy; the compiler identifies promotable direct-input failures |
| `observe-new-inputs` | Replay the selected program and run optional final evaluation |
| Other actions | Preserve the proposal and observed program |

The last row includes fully exhausted choice sets and a request to resume a
saved candidate ranking. The separate `construction-loop` experiment handles
package continuation. New choices and grammars remain source declarations.

## Evidence and bounds

The compiler exports the named activity's plan before any model loading or
candidate evaluation; its initial budget must fit `--max-attempts`. Refinement
is bounded to 1..8 rounds and at most 64 attempts per round. The wrapper applies
a two-minute timeout to each compiler invocation and honors cancellation.

All input files are copied into the new output directory. The original source
stays unchanged. The compiler modifies the selected activity's assembly budget
or selection cases in new source artifacts. Semantic IDs, bodies and permitted
choices retain their declared meaning. The policy determines whether a round is
retained, including when it does not improve the supplied feedback score.

`refinement-dispatch.json` records initial/final observations, both Gooo proposals,
whether dispatch occurred, refinement rounds, model calls and optional evaluation.
`source-plan.json` binds the initial budget to the compiler's source export.
`refinement-result.json` retains every source revision and native policy result.
`final-result.json` is a fresh saved-composition replay with zero new inference.
The summary's `selected_source` points to the retained source relative to output.

The initial diagnosis is a separate construction. When dispatch occurs,
`body-refine` constructs the baseline again and can call the model once in each
round. Both stages' calls are counted. Feedback cases affect the policy and
are adaptive measurements. Optional final evaluation is run only after source
selection and is never passed to either decision program. A failed evaluation
or unresolved feedback leaves the aggregate status `PROGRESS`.

The example has seven feedback inputs and three separate final inputs. Model
training exposure is unknown. The tests cover deterministic and model ordering,
budget exhaustion, missing construction cases, an already satisfied program,
an unmet final expectation, cancellation and initial source-budget bounds.
