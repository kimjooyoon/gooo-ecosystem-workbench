# Let Gooo explain declaration changes

The `api-diff` command reads two Gooo workspaces through the compiler's
`package interface` command. Go matches stable package, declaration and field
identities. The [Gooo recipe](../../recipes/api-changes.gooo) classifies the
differences and returns the next operation to try.

This requires compiler source `8951c5f8f22526e9cc1225a4dd47e63a6d420c4a`
or a later source containing `package interface`; the existing public
0.6.18 release does not yet include that command. CI pins this exact candidate.

```sh
go run ./cmd/workbench api-diff --compiler /path/to/gooo-with-package-interface \
  --before examples/api-evolution/before/gooo.workspace.json \
  --after examples/api-evolution/after/gooo.workspace.json --out out/api-diff

go run ./cmd/workbench api-diff --compiler /path/to/gooo-with-package-interface \
  --before examples/api-evolution/before/gooo.workspace.json \
  --after examples/api-evolution/after/gooo.workspace.json \
  --model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json \
  --out out/api-diff-model
```

The example keeps the same record and field IDs while changing three details:

| Change | Gooo classification | Next operation |
| --- | --- | --- |
| Request count becomes required | `input-tightened` | Supply the required input in caller examples |
| Response count becomes optional | `output-weakened` | Exercise callers when that output is absent |
| Optional response note is added | `field-added` | Check record construction |

The recipe also handles additions/removals, type/cardinality changes, names,
field order, ordered function arguments, imports, and the workspace entry.
A record used in both directions gets a shared-use assessment. Usage comes
from every declared activity signature in either workspace snapshot. Unknown
external consumers remain outside this observation.

## Source and model roles

The compiler resolves names and fields. Go retains both original compiler
receipts and computes structural facts. Gooo owns the conditional rules,
classification, next operation, and priority (0–2). It contains 17 authored
selection examples and three two-way code assembly choices for the output
fields. A model can order these choices; the eight-program budget also permits
deterministic selection. There is no training during this command.

The selected program runs on the actual changes without expected labels.
`api-diff.json` reports `OBSERVED`, the number of structural changes, and how
many received a known classification. That count measures rule coverage over
these changes; it is not a correctness percentage or compatibility guarantee.
The 51 selection field checks belong to the 17 authored policy cases.

The first execution is replayed and compared. Additional changes are handled
in batches of at most 128 by the same saved program with zero new inference.
No changes are dropped at that boundary. Raw compiler receipts, structural
facts, generated program, execution, and every replay remain in the output
directory. The input workspaces are read without being edited.

## Reading an unchanged result

An empty declaration difference still runs the Gooo `none` rule and proposes
observing runtime behavior. Source digests can change while declarations stay
the same: comments, activity bodies, and runtime bindings can have changed.
The report keeps `source_identity_changed` separate. It compares the exported
surface, imports and entry; it does not inspect body behavior or prove that
existing callers will execute correctly.
