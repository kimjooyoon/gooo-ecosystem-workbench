# Gooo dispatches a source revision and observes the next program

Clean compiler `cdb60e90f911d53d7c02790579e5f72941a2b71a`, clean workbench
`be791d9612d39de9d7cd36290fe989629482e9a8`, Go 1.27.1, Darwin arm64.
[Summary](summary.json), [raw-file hashes](sha256.json),
[source, policy and commands](../../examples/source-refinement/README.md).

The first program satisfies 6/14 named feedback outputs. Its Gooo diagnostic
returns `raise-attempt-budget`. The Go adapter then invokes `body-refine` with
the supplied Gooo policy. That policy incorporates one explicit counterexample,
changes the source attempt budget and retains an observed program. The adapter
replays the retained source and runs the Gooo diagnostic again.

| Run | Feedback before → after | Refinement budgets | Final evaluation | Model calls: initial + refinement | Final action |
| --- | --- | --- | --- | --- | --- |
| Deterministic | 6/14 → 14/14 | 2 → 4 → 8 | 6/6 | 0 + 0 | Observe new inputs |
| Own compact model | 6/14 → 14/14 | 2 → 4 | 6/6 | 1 + 2 | Observe new inputs |
| Bound to two attempts | 6/14 → 6/14 | 2 → 2 | 4/6 | 0 + 0 | Raise attempt budget; retained as PROGRESS |

The successful runs revise only the selected assembly contract, promoting one
feedback case and changing its budget. The original input file stays unchanged.
The bounded run also promotes that case but cannot complete the declared
behavior under its cap. Its next-action proposal remains visible beside the
policy stop; stopping is recorded separately from satisfying expectations.

Seven feedback inputs are adaptive: they influence policy decisions. Three
separate final inputs produce six evaluation outputs only after source selection.
Neither Gooo decision program receives those final evaluation results. Model
training exposure is unknown, and this single program does not establish broader
accuracy or a timing advantage.

The unchanged [own compact model](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1)
is copied from this repository's `models/shared-qat` bundle for the model run.
Raw construction receipts retain its weight/metadata digests and 2,096-byte
resident tensor count. The two refinement predictions took 17,625 and 20,791 ns.
The initial diagnosis separately constructs the baseline and makes another
prediction; its cost and attempts are not hidden inside the refinement count.
Both final feedback replay and final evaluation use zero new predictions.
Policy and diagnostic execution also use zero predictions. No training occurred.

## Retained evidence

Each directory keeps copied inputs, the source-derived plan, initial native
result, initial/final Gooo diagnostic programs and outputs, complete refinement
rounds, replayable initial/selected compositions, and final execution results.
`refinement-result.json` includes every revised source and policy input/output.
`refinement-dispatch.json` identifies the retained source relative to its directory.
Duplicate generated Go and temporary module files are omitted; the commands
recreate them in a fresh output directory.

Run the command in the linked example once without a model, once with
`--model builtin`, and once with `--max-attempts 2 --max-rounds 2`, using a
different new output directory for each run. These are the three runs above.

## 한국어

이번에는 다음 행동을 표시하는 데서 실제 소스 수정으로 연결했습니다. Gooo가
“아직 확인하지 않은 후보가 있다”고 판단하면, 또 다른 Gooo 정책이 시도 한도를
늘리거나 실패한 직접 입력을 조립 사례에 추가합니다. 수정한 소스로 다시 실행한
결과도 Gooo가 읽습니다. 한 번 명령을 시작한 뒤 중간에 사람의 추가 입력은 없었습니다.

같은 예제에서 고정 순서는 세 차례, 자체 모델은 두 차례의 수정 루프를 거쳐
피드백 14/14와 최종 평가 6/6에 도달했습니다. 한도를 작게 둔 실행은 6/14와 4/6에
머물렀고 부분 결과를 그대로 보관했습니다. 무엇을 맞혔고 무엇이 남았는지 함께
전달하는 메타프로그래밍 경로를 조금 더 연결한 실험입니다.
