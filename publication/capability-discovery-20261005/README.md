# JEV capability discovery assessed by Gooo

This publication binds one Korean natural-language query to a supplied Gooo
declaration, runs the provider-neutral JEV capability catalog, and passes its
status, intent, declaration binding, and digests through the Gooo-authored
`capability-assessment.gooo` program.

- Query: `코드 생성은 어떻게 해?`
- Declaration: `examples/catalog/operations.gooo`
- JEV: `github.com/kimjooyoon/gooo-jev` at
  `v0.0.0-20261005004607-ae1a015c4b33` (commit `ae1a015c4b336944d6aa2f84de99da0703e82073`)
- Compiler: `meta-ontology-go` at
  `f144dddb8261b9b525181dee510afc65f1603153`
- Result: catalog `AVAILABLE`, declaration bound, `real_use_case_coverage=PROGRESS`,
  first unresolved stage `generation`.
- Native execution: two connected Gooo activity outputs passed (2/2), all record
  fields passed (22/22), and saved composition replayed. Model/provider calls: 0.

`AVAILABLE` describes a catalog match. It does not prove that this declaration
can generate code. Generation and reverse observation remain unmeasured. The
assessment explicitly records `execution_attempted=false`; the native execution
above only checks the assessment program itself.

This directory keeps the exact query and declaration, JEV trail and guide,
Gooo source, generated Go, composition, expected and actual values, and saved
replay. `checksums.sha256` binds every retained file except itself.

Reproduce from the workbench root:

```sh
go run ./cmd/workbench discover --compiler /path/to/gooo \
  --query '코드 생성은 어떻게 해?' \
  --declaration examples/catalog/operations.gooo \
  --out out/capability-discovery
```
