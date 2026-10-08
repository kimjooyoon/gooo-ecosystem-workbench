# Graph feature audit through Gooo

Observed on 2026-10-08 from clean workbench source
`ab54d00b14c97beec14ed9053309f3b8c5849f72`, built with Go 1.27.1 and public
decision runtime v0.2.26-experimental. `workbench-build.txt` retains embedded
source/module information with trailing whitespace removed; `workbench-sha256.txt`
identifies the temporary executable. The assessment uses the installed Gooo
0.6.8-dev compiler, source `86b182da4543efa24ea5bcb504e7b13ea1449c39`.
The model-input graph was previously exported from compiler dea641f7;
the installed compiler here executes the stable Gooo assessment recipe.

The input has one source family and eight arrangements. The actual 768-float
arrays form eight groups, with no contradictory supplied labels in a group.
The empirical maximum compatible count is 8/8 for these rows. No model is loaded:
model calls are zero, model performance remains unobserved.

The unchanged Gooo `Assess` activity returns:

```json
{"code":"input-consistent","action":"evaluate-chooser","message":"제공된 자료에서 충돌을 찾지 못했다. 후보 판단을 별도로 측정한다."}
```

`assessment/` retains the Gooo source, input observation, generated Go and saved
composition/runtime outputs. Native projection/runtime replay succeeded. The
eight feature hashes and row memberships match the independent compiler export
study. The importer reproduced `filename-graph-order.json` byte for byte and
checks matching source/contract digests between exports and finite labels.

The full workbench race suite passed in 90.942s, with GOOO_COMPILER set so native
assessments executed. The subsequent graph-focused race run (including the new
synthetic zero-weight dispatch test) passed in 2.158s; static checks passed.
Synthetic-model tests measure ABI/dispatch accounting, not trained quality.
The public observation here uses no model.

## Reproduce

```sh
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-graph-order.json --out /path/to/new-audit
```

The graph exports and finite labeling evidence are linked in
[the example guide](../../examples/feature-audit/README.md).
Source families must stay separate in later training/evaluation work. Eight
distinguishable arrangements do not establish generalization or remove hash
collisions on other programs. No weights were changed or uploaded.

`SHA256SUMS` binds all files in this directory except itself. Paths and timings
may differ on reproduction; feature arrays, group membership and policy results
are the reproducible comparison targets.
