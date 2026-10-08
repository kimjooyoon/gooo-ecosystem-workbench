# Source distinctions lost before model inference

These observations use workbench source
`f7c0642d5c527e60e2df45c3fe787b1d2c171d1b`, the unchanged built-in shared-field
model, and the installed Gooo 0.6.8-dev compiler. The compiler identity is retained
in `compiler.json`; its source is `86b182da4543efa24ea5bcb504e7b13ea1449c39`.
Both commands completed on 2026-10-08 with Go 1.27.1 on macOS arm64.

```sh
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-order.json --model builtin --out out/audit-model
go run ./cmd/workbench feature-audit --compiler /path/to/gooo \
  --input examples/feature-audit/filename-order.json --out out/audit-no-model
```

| Observation | With model | Without model |
| --- | ---: | ---: |
| Source families | 1 | 1 |
| Candidate-order rows | 8 | 8 |
| Distinct complete feature arrays | 2 | 2 |
| Conflicting groups | 2 | 2 |
| Maximum compatible rows for a deterministic feature-only chooser | 2/8 | 2/8 |
| Model predictions | 8 | 0 |
| Accepted first predictions | 1/8 | Unobserved |
| New model calls in the Gooo assessment | 0 | 0 |

The arrays are grouped by all float32 bits, not a rounded value or a partial
digest. Each group contains four rows that require different masks. The largest
accepted-label count is one in each group, giving a row-weighted upper bound of
two for this supplied dataset. Several acceptable masks per row are supported.

The input projection merges `&&` with `||` and merges `input` with `stem`.
Those distinctions are lost before the model executes. Training on this same
feature contract cannot make a deterministic feature-only chooser recover them.
The next representation should preserve operator identity and relevant value
origins, then repeat the audit before training or making accuracy claims.

Both runs execute `recipes/feature-audit.gooo` and return
`representation-collision` with the action
`preserve-distinguishing-source-facts`. The assessment's native projection and
saved execution replay. The outputs retain the supplied input, exact feature
groups, model predictions when requested, policy source and generated Go.

The labels refer to the selected bodies in the
[filename-order study](https://github.com/kimjooyoon/meta-ontology-go/wiki/Text-Operations-and-Choice-Order).
They are assumptions supplied to this audit, not labels independently proved by
the audit command. The eight rows rearrange one program's candidates; they are
not eight independent programs. The earlier native suite contains twelve unique
filenames and is not rerun by this command. No model weights were updated.
Timing and host CPU utilization were not measured here.

`SHA256SUMS` covers the retained files except the checksum manifest itself.
Use `shasum -a 256 -c SHA256SUMS` from this directory to check the snapshot.
