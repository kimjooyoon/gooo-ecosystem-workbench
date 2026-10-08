# Native retry-policy construction, 2026-10-08

[The reusable Gooo source and commands](../../examples/retry-policy/README.md)
define a capped retry decision. This observation used clean workbench source
`ec937155` (the full identity is in `summary.json`) and compiler candidate
`99473378b011dd61ab323daa96f94a5cfc2624a0`, built with Go 1.27.1 on macOS arm64.
The candidate identifies itself as 0.6.7-dev; it was observed before binary publication.

| Mode | Construction cases | Separate native plans | Candidates tried | New model calls |
| --- | --- | --- | --- | --- |
| Fixed order | 5/5 | 12/12 | 8 of 8 | 0 |
| Own compact model | 5/5 | 12/12 | 1 of 8 | 1 |
| Saved model-program execution | retained 5/5 | 12/12 | retained 1 | 0 |

Each plan has three fields. Construction matched 15/15 fields; the twelve native
plans have 36 expected field values. Source conditions decide completion,
permanent failure, attempt exhaustion and invalid input. The model ranks the
three permitted result-field choices. Its first proposed mask was 7 and passed
all construction cases in this run.

The model's single prediction took 14,875 ns. Its recorded setup was 0.412 ms and
resident tensors occupied 2,096 bytes. These are individual constructor and
prediction observations; the record does not measure a general latency or CPU
utilization improvement. The frozen model weights match
`049081cc6dc62f3c4230f72e601c1fd421c514fce9cfdf7bc1eb412ca4c5514b`.

`retry_policy_test.go` separately evaluates 200 tuples per mode against an
arbitrary-precision Go oracle, including zero caps, odd caps and int64 extremes.
The native regression observed 200/200 plans and 600/600 fields in each mode.
Those tuples are split into four bounded files and run using the same saved
program. This test can be repeated with:

```sh
GOOO_COMPILER=/path/to/gooo go test -race -run 'Test(NativeRetry|RetryOracle)' -count=1 -v .
```

The twelve publication tuples and 200 regression tuples differ from the five
source construction tuples. Model-training overlap is unknown. Existing model
weights were used without additional training. The saved program returns a
decision; waiting, attempt accounting and performing the operation belong to
its caller.

Raw results, original/realized Gooo, generated Go and saved compositions are
retained here. `sha256.json` lists every retained file except itself. Historical
model-call fields remain in replay records; `runtime.model_calls` and
`generated_now` identify what the replay invocation actually did.
