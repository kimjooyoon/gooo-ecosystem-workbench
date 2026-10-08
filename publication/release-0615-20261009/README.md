# Public Gooo 0.6.15 workbench integration

The compiler pin is the published source
`dc75f59fbee16e776dca13288efe1036d1bb5fab` (Go 1.27.1,
decision runtime v0.2.26-experimental). The actual downloaded macOS arm64
executable was verified and installed before these observations.

Local `go vet ./...` and `go test -race ./...` passed using that installed binary.
The unchanged caller-fill-rejection example completed five rounds and 13 program
attempts. The final separate holdout matched 4/4; replay made zero new model calls.
Original failed cases and expectations are retained in `joint-loop.json.gz`.

The release source is also pinned in CI. Its result must be read separately from
these local observations. Earlier example records remain at their original source
revisions. See the [public release observations](https://raw.githubusercontent.com/wiki/kimjooyoon/meta-ontology-go/observations/release-0615-published-20261009/README.md)
for downloaded asset checks, six direct compiler construction/replay outputs,
exact-number recount and the remaining caller execution failure.
