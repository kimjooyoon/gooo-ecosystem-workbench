# First run: decide whether to retry an API call

[한국어](first-run.ko.md)

Run an existing Gooo policy, inspect the Go it generates, and execute its saved
selection again. For a transient failure after two attempts, with a previous
delay of 7 ms and a 100 ms cap, the example returns `retry: true`, `delay_ms: 14`,
and `reason: "retry"`. It calculates the decision without per-action human approval.

## 1. Get the example and select Go

Use Linux x86-64, macOS Apple Silicon, or macOS Intel with Git, curl, tar, and
either `sha256sum` or `shasum`. Native execution requires **Go 1.27.2**. Install
it, or use a Go installation with toolchain selection support (Go 1.21+); the
first command below may download that toolchain when it is not already cached.
Run the shell blocks in the same terminal, stopping if any command fails.

```sh
set -eu
export GOTOOLCHAIN=go1.27.2
go version
git clone --no-checkout --depth 1 https://github.com/kimjooyoon/gooo-ecosystem-workbench.git gooo-workbench
cd gooo-workbench
git fetch --depth 1 origin b94b3adb4845782743a1c1a45071a600824f81e6
git checkout --detach b94b3adb4845782743a1c1a45071a600824f81e6
```

Check that `go version` reports `go1.27.2`. The checkout pins the example to
workbench commit `b94b3adb4845782743a1c1a45071a600824f81e6`.
[source.gooo](../examples/retry-policy/source.gooo) contains the retry conditions,
bounded delay arithmetic, and three result-field choices; its five construction
cases select among eight combinations. [cases.json](../examples/retry-policy/cases.json)
contains twelve separate native input tuples and their expected decisions.

## 2. Install one pinned public compiler

Use the [Gooo 0.6.23-dev experimental prerelease](https://github.com/kimjooyoon/meta-ontology-go/releases/tag/v0.6.23-dev),
source `2b17c4879d0bb2cd1e48ccc23b8743c0fb564267`. These commands download a
platform binary, verify the pinned archive and checksum file, and install it
inside the ignored `.compiler/bin` directory. They do not build the engine.
This is a selected reproducible version, not a stable/latest-release claim.

```sh
case "$(uname -s)/$(uname -m)" in
  Linux/x86_64)
    GOOO_ASSET=gooo-linux-amd64.tar.gz
    GOOO_ARCHIVE_SHA256=603619c0938cb2b73db2b1420cf74dc1a8d856c311267b781e9687f23e57bae9 ;;
  Darwin/arm64)
    GOOO_ASSET=gooo-darwin-arm64.tar.gz
    GOOO_ARCHIVE_SHA256=0658f1c5f1c710a52f6f6d4d0437b78ac24df77f9d2cab35cea3db20d62fdb04 ;;
  Darwin/x86_64)
    GOOO_ASSET=gooo-darwin-amd64.tar.gz
    GOOO_ARCHIVE_SHA256=54345ca51ca1bc36e96f6047366473d8190b0a739bf4089c1a0a0a0a9718c630 ;;
  *) echo "This guide supports Linux x86-64 and macOS arm64/x86-64." >&2; exit 1 ;;
esac
gooo_check_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum -c -
  else shasum -a 256 -c -; fi
}
GOOO_RELEASE_URL=https://github.com/kimjooyoon/meta-ontology-go/releases/download/v0.6.23-dev
mkdir -p .compiler/downloads .compiler/bin
curl --fail --location --show-error "$GOOO_RELEASE_URL/$GOOO_ASSET" --output ".compiler/downloads/$GOOO_ASSET"
curl --fail --location --show-error "$GOOO_RELEASE_URL/SHA256SUMS" --output .compiler/downloads/SHA256SUMS
(
  cd .compiler/downloads
  printf '%s  SHA256SUMS\n' acfdc5d95098398b69f5278e299fdb4b393f98f4a4d155461c5ab8665347d8cd | gooo_check_sha256
  printf '%s  %s\n' "$GOOO_ARCHIVE_SHA256" "$GOOO_ASSET" | gooo_check_sha256
  awk -v file="$GOOO_ASSET" '$2 == file { print }' SHA256SUMS | gooo_check_sha256
)
tar -xzf ".compiler/downloads/$GOOO_ASSET" -C .compiler/bin
export PATH="$PWD/.compiler/bin:$PATH"
command -v gooo
gooo version --build --json
```

The checksum command selects one row because the full checksum file also lists
assets you did not download. Stop on a checksum failure. The compiler identity
must report `version: "0.6.23-dev"`, `go_version: "go1.27.2"`,
`vcs_revision: "2b17c4879d0bb2cd1e48ccc23b8743c0fb564267"`,
`vcs_modified: "false"`, and `decision_runtime.version: "v0.2.26-experimental"`.
The absolute path printed by `command -v gooo` is the compiler for this guide.

## 3. Check, generate Go, and inspect the decision

From the same repository root:

```sh
GOOO_RUN=out/retry-first-use
mkdir -p out
mkdir "$GOOO_RUN"
gooo version --build --json > "$GOOO_RUN/compiler-build.json"
gooo check --json examples/retry-policy/source.gooo > "$GOOO_RUN/check.json"
gooo body-compose --source examples/retry-policy/source.gooo \
  --cases examples/retry-policy/cases.json \
  --out "$GOOO_RUN/composition" > "$GOOO_RUN/execution.json"
```

No model option is supplied. Gooo tests the declared candidates in fixed order,
generates ordinary Go, compiles it with Go 1.27.2, and runs the twelve inputs.
`check` checks the source; the construction cases and native expectations are
evaluated by `body-compose`.

Open `out/retry-first-use/execution.json`. In
`runtime.traces[0].deliveries[0].actual`, the first input tuple
`false, true, 2, 4, 7, 100` produces:

```json
{"retry": true, "delay_ms": 14, "reason": "retry"}
```

The six inputs mean: preceding operation succeeded, failure is transient,
attempts used, maximum attempts, previous delay, and maximum delay.
`runtime.finite_passed` and `runtime.finite_total` should both be `12`;
`runtime.model_calls` should be `0`.

| File under `out/retry-first-use/composition/` | What to inspect |
| --- | --- |
| `realized.gooo` | Selected Gooo body with the three computed fields connected |
| `generated.go` | Go generated from that selected body |
| `main.go`, `go.mod` | Native execution driver and its Go module |
| `composition.json` | Saved source-bound selection used by the next command |
| `original.gooo`, `cases.json`, `runtime.json` | Original source, supplied cases, and native result |

## 4. Execute the saved selection

```sh
gooo body-compose --source "$GOOO_RUN/composition/original.gooo" \
  --composition "$GOOO_RUN/composition/composition.json" \
  --cases examples/retry-policy/cases.json > "$GOOO_RUN/replay.json"
```

In `replay.json`, `generated_now` should be `false`, the twelve native cases
should still pass, and `runtime.model_calls` should be `0`. This reruns the
saved selection without choosing a new candidate or making a new model call;
native compilation/execution remains part of replay. It does not add new
independent test inputs. For another construction run, set `GOOO_RUN` to a new
path such as `out/retry-first-use-2`; existing output directories are refused.

## Optional: use the same compiler in VS Code

Install the [0.2.2 VSIX](https://github.com/kimjooyoon/gooo-ecosystem-workbench/releases/tag/vscode-v0.2.2)
using **Extensions: Install from VSIX...**. Open the `gooo-workbench` folder.
In VS Code settings, set `gooo.compilerPath` to the exact absolute executable
path printed above, ending in `.compiler/bin/gooo`. Open
`examples/retry-policy/source.gooo` and run **Gooo: Check Current File**.
Use the terminal commands above to run the example, then open
`composition/realized.gooo` and `composition/generated.go` under your output
folder. A new terminal needs the same `GOTOOLCHAIN` and absolute PATH exports.

The caller supplies the failure classification and attempt count and performs
the operation and actual waiting. This policy returns a decision; it does not
schedule work or apply API effects. The supplied finite cases establish the
shown behavior, not universal safety for arbitrary APIs. This direct path
requires no Node/npm, hosted model, or Laya service.
