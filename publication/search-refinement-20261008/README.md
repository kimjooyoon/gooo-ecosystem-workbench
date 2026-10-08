# Expression search, a small record model, and visible grammar limits

Clean compiler `661ec03c5145a252564df2919a109f4a5d8834d8`, clean experiment
`0c7426e2efbe5e7d22629831d3141a96338d3ca7`, Go 1.27.1, Darwin arm64.
[Summary](summary.json) · [file hashes](sha256.json) ·
[source and reproduction](../../examples/search-refinement/README.md).

Gooo's source-owned integer grammar now enters the workbench refinement loop.
The initial `Add` declaration has one case and one attempt. Gooo observes the
native result, incorporates an explicit failed direct input, and raises the
selected activity's budget under the supplied Gooo policy. A separate activity
can assemble a record from that generated integer.

| Run | Feedback before → after | Final evaluation | Refinement rounds | Model calls: initial + refinement | Last Gooo action |
| --- | --- | --- | --- | --- | --- |
| Integer search | 0/2 → 2/2 | 2/2 | 4 | 0 + 0 | Observe new inputs |
| Integer → record, fixed ordering | 0/4 → 4/4 | 4/4 | 4 | 0 + 0 | Observe new inputs |
| Integer → record, own small model | 0/4 → 4/4 | 4/4 | 4 | 1 + 4 | Observe new inputs |
| Inner expression missing from grammar | 0/2 → 0/2 | 0/2 | 4 | 0 + 0 | Expand declared choices/grammar |
| Inner expression, two-candidate cap | 0/2 → 0/2 | 0/2 | 2 | 0 + 0 | Expand search space |

The first three runs use budgets `1 → 2 → 4 → 8`. Their selected integer
expressions change from `input` to `3`, then finally `input + 1`. An attempt
budget bounds each round; earlier attempts remain in the recorded history.
The initial diagnosis is a separate construction and is counted separately.

All runs have two adaptive feedback inputs and two separate final evaluation
inputs. The mixed runs check two activity outputs for each input. Evaluation is
performed after retaining a source revision and is not passed to either Gooo
decision program. Model training exposure is unknown. These small examples
measure supplied expectations; they do not establish general language accuracy.

## What the model contributed here

The unchanged [own shared record model](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1)
ranked the three Integer/Boolean/String record choices. In each of the four
refinement rounds, fixed ordering checked eight record combinations; the model
selected the passing combination first. Integer grammar selection remained
deterministic, so both mixed runs still took four refinement rounds.

Those four prediction measurements were 12,333, 20,375, 19,792 and 22,250 ns.
They cover prediction only. Loading, compilation, native execution, diagnostics
and source revision have separate costs. Resident model tensors were 2,096 bytes.
No model training occurred and this run does not establish end-to-end speedup.
Every final saved-program replay used zero new model predictions.

## What stayed unfinished

For `input + hole`, the selected offset grammar derives candidates from final
expected outputs and misses the needed inner constant `1` in these cases. Trying
all six retained expressions still produces 0/2. Gooo reports that a source
choice or grammar needs attention. The compiler also supports a contextual
residual grammar; selecting that grammar is a separate source change.

With a two-candidate cap, four generated expressions are omitted. The workbench
preserves this distinction, and Gooo proposes expanding the search space.
Neither case is reported as completed. The command currently automates bounded
budget/case revisions; grammar and candidate-cap changes remain proposed work.

## Retained evidence

Each run keeps the source export, copied input cases and policy, initial/final
Gooo decisions, native execution records, every refinement round and the selected
source/composition for replay. `refinement-result.json` embeds the full round
history. Duplicate `refinement.json`, generated Go/module files, realized-source
copies and per-directory runtime copies were removed to limit publication size.
The complete native result files retain those observations. The reproduction
command rebuilds the generated files in a fresh directory.

## 한국어

이번에는 Gooo가 숫자식의 빈칸을 채우고, 그 값을 다음 활동에 넘겨 작은 모델이
레코드를 조립하도록 연결했습니다. 모델을 사용한 레코드 조립은 매 라운드 첫 후보가
사례를 만족했습니다. 숫자식을 찾아가는 수정 과정은 고정 순서와 똑같이 네 번이었습니다.

안 되는 경우도 두 가지를 함께 남겼습니다. 필요한 식이 현재 문법에 없는 경우와,
후보 수 제한 때문에 일부 식을 확인하지 못한 경우입니다. 두 상황에서 Gooo가 서로
다른 다음 행동을 제안합니다. 조립 결과와 함께 무엇을 더 바꿔야 하는지 전달하는
메타프로그래밍 경로를 이어 붙인 실험입니다.
