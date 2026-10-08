# Source-graph choices across three Gooo programs

2026-10-08. Reproduction code: `8cb81bf9ba262b0c083894c4edb1b241949d7f2e`.
Compiler: clean `e0d046503939883330eb37f998a2e0e8fc5154e5`, Go 1.27.1,
SDK v0.2.26-experimental. This compiler is the recorded release-preparation
candidate, not a claim that 0.6.9 was publicly released. `compiler.json` binds it.

| Program family | Arrangements | Intent forms | Rows | Source selection cases per row |
| --- | ---: | ---: | ---: | ---: |
| Filename classifier | 8 | Korean / English / mixed | 24 | 5 |
| Integer division/remainder | 8 | Korean / English / mixed | 24 | 4 |
| Retry decision | 8 | Korean / English / mixed | 24 | 5 |

Every row reached its source-declared finite cases and fields. Candidate checks
were deterministic, with zero model calls. Each original arrangement required
all eight candidates and had a single fully matching combination; the other
arrangements selected the corresponding swapped mask. Each family used 108
candidate attempts across its 24 rows. These counts repeat the same supplied
cases; they are not counts of independent programs or held-out input cases.

`rows/<id>/source.gooo`, `context.json` and `finite.json` retain the actual source,
source-only graph and separate label observation. `corpus-index.json` binds source,
contract and encoded-context digests. The collector rejects missing zero-call
fields, mismatched identities, partial labels and nonunique baseline labels.

The full float32 projection gave 72 distinct arrays and zero conflicting groups
for these rows. The Gooo program in `assessment/source.gooo` returned
`input-consistent` / `evaluate-chooser` and replayed its generated program.
`feature-audit.json` records the report, source identity and model-call count.
Its input is `examples/feature-audit/three-family-graph.json` in this repository.
Two independent collection directories produced byte-identical audit inputs
and corpus indices; timing and file-location fields in raw outputs may differ.

No model was trained, and these results do not measure Korean/English intention
following. All three wordings request equivalent behavior. The next learning
split holds out an entire family (48 train rows / 24 evaluation rows); intent
sensitivity also needs source alternatives with different requested behavior.
See [sources, collection command and scope](../../examples/graph-choice-corpus/README.md).

Local checks: full `go test -race ./...` passed, including native Gooo tests using
the installed 0.6.8 compiler for existing ecosystem operations (root suite 92.234s).
The new collector's unit regressions passed (1.683s), and `go vet ./...` passed.
Graph extraction itself used the exact newer compiler recorded above.
CI consumes the recorded graph corpus with SDK v0.2.26 and executes the Gooo
assessment using its existing pinned compiler; it does not claim to re-export
v3 source graphs with that older executable.

`sha256.json` covers every retained file except the checksum manifest itself.
