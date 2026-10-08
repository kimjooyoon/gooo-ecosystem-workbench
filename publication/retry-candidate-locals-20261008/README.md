# Retry assembly from prepared local values

The retry source prepares its decision, capped delay and reason in local
variables. Its baseline result is `false`, `0`, `pending`. Three declared choices
can connect the prepared values to the corresponding result fields.

This observation uses clean workbench source `312b1106fc595d703af5e93398fa6b6654bf3fb6`
and compiler source `24e5e96f4cec97a97a22b8ab8cebd8ab04a0800f`, with public
decision-runtime v0.2.25-experimental. That compiler's version field still reads
0.6.7-dev. The published 0.6.7-dev binary uses the earlier source recorded in
`previous-compiler-build.json`; the new baseline reproduces its unused-variable
error in `previous-compiler-regression.log`.

## Complete and partial results

| Run | Candidates attempted | Construction cases | Construction fields | Native cases | New model calls |
| --- | ---: | ---: | ---: | ---: | ---: |
| Fixed order | 8 | 5/5 | 15/15 | 12/12 | 0 |
| Own compact model | 1 | 5/5 | 15/15 | 12/12 | 1 |
| Saved model result | 0 new | retained 5/5 | retained 15/15 | 12/12 | 0 |
| One fixed-order attempt | 1 | 0/5 | 8/15 | 0/12 | 0 |
| Saved one-attempt result | 0 new | retained 0/5 | retained 8/15 | 0/12 | 0 |

All attempted candidates passed type checking. The one-attempt run kept the
unconnected baseline as an executable partial result. Its zero matching cases
and eight matching fields are separate measurements. Saved replay retained the
same generated program in both the complete and partial runs.

The existing independent Go oracle also checked 200 unique input tuples and
600 output fields per mode. Fixed-order and model construction both passed.
These tuples differ from the five source selection tuples; the twelve published
native tuples also differ from those five. Model-training overlap is unknown.
The exact regression output is in `regression.log`.

## Model and resource observations

The existing shared model proposed mask 7 first, preserving all three prepared
values. Its single prediction took 46,042 ns, setup took 1.049583 ms, and retained
tensors used 2,096 bytes. Its weights digest is
`049081cc6dc62f3c4230f72e601c1fd421c514fce9cfdf7bc1eb412ca4c5514b`.
No training or model download was performed.

The model command, including native compilation and execution, took 0.36 seconds
with maximum RSS of 86,016,000 bytes. The raw command observations are retained
in the `.time` files. These are single sequential measurements with uncontrolled
caches and host load. CPU utilization was unmeasured; a general speed or memory
advantage remains unestablished.

## Reproduce

From the workbench repository root, use the pinned compiler or a compatible
later source with unread-local support:

```sh
go install github.com/kimjooyoon/meta-ontology-go/cmd/gooo@24e5e96f4cec97a97a22b8ab8cebd8ab04a0800f
gooo body-compose --source publication/retry-candidate-locals-20261008/source.gooo \
  --cases publication/retry-candidate-locals-20261008/cases.json \
  --composition publication/retry-candidate-locals-20261008/model-composition.json
gooo body-compose --source publication/retry-candidate-locals-20261008/partial.gooo \
  --cases publication/retry-candidate-locals-20261008/cases.json \
  --composition publication/retry-candidate-locals-20261008/partial-composition.json
GOOO_COMPILER="$(command -v gooo)" go test -race -run 'Test(NativeRetry|RetryOracle)' -count=1 -v .
```

The full workbench race suite passed in 88.666 seconds. The clean-source focused
regression passed in 5.438 seconds. Native test support requires Go 1.27.1.
`summary.json` retains source identities, separate denominators and measurement
limits; `SHA256SUMS` covers the retained files. Earlier retry observations remain
in their original publication directory.
