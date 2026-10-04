# Gooo declaration to API reference

This observation exercises the published compiler `operation-interface` output
and a Gooo-authored Markdown template. The compiler supplies a selected public
activity, its declared input/output types, stable type IDs, a source digest, and
an interface digest. `recipes/reference.gooo` assembles the reference page.

The sample has two input types, `Invoice` and `Reviewer`, and output type
`Approval`. The generated page reproduces that signature and all three stable
type IDs. It labels its scope as a declaration reference and makes no claim
about runtime behavior. The original two Gooo declaration files and the exact
compiler interface JSON are retained here.

The native body-composition case compares the delivered document with a
separately assembled expected page. The result is 1/1 named output. A saved
composition replay reproduces it without a model call. This checks the template
on this declared fixture; it is not evidence for every Gooo feature or all
possible source declarations. An unknown activity fails without emitting a
reference.

`local-tests.txt` is the successful actual-compiler `go test -race ./...` run.
`checksums.sha256` binds every retained file.

Reproduce from the repository root:

```sh
go run ./cmd/workbench reference --compiler /path/to/gooo \
  --package examples/catalog --entry ApproveInvoice --out out/api-reference
```
