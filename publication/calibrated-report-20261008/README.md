# Numeric construction followed by compact-model report assembly

Clean compiler `14717427b49ddd7562503e25ada41245e0577262`, clean experiment
`739b6edc9b812bf756e41fc07a2ddec4032f8de0`, Go 1.27.1, Darwin arm64.
[Summary](summary.json) · [model observations](model-observations.json) ·
[file hashes](sha256.json) · [source and commands](../../examples/calibrated-report/README.md).

The synthetic sensor graph computes `(2 * input + 1)^2` and reports the result,
whether it is at least 400, and a label. Gooo owns the numeric body, alternative
grammars, record choices, cases and refinement policy. Go connects the commands.
The existing own compact model ranks the three record-field choices.

## Four native runs of one graph

| Mode | Feedback before → after | Separate evaluation | Refinement rounds | Model calls: initial + refinement |
| --- | --- | --- | --- | --- |
| Fixed candidate order | 2/20 → 20/20 | 6/6 | 2 | 0 + 0 |
| Own compact model | 2/20 → 20/20 | 6/6 | 2 | 1 + 2 |
| Model with a one-round limit | 2/20 → 2/20 | 0/6 | 1 | 1 + 1 |
| No alternative grammar | 2/20 → 2/20 | 0/6 | 1 | 0 + 0 |

Ten root inputs each have two expected activity outputs. Three separate root
inputs also have two outputs each. All evaluation inputs are withheld until
source selection. For successful runs the native input-separation report marks
all three disjoint, including observations at the downstream activity. The two
bounded runs retain `PROGRESS` and their unresolved outputs.

In the successful runs the Gooo policy advances from quadratic/v1 to the declared
quadratic/v2 grammar and then stops. Both enumerate 89 numeric expressions and
retain 16 (17.98%). The first round checks eight numeric candidates and retains
`-3`, matching one of ten numeric cases. The next round checks one candidate and
retains `input * -2 + -1`, matching ten of ten. Its square gives the expected
energy. The grammar remains truncated in the receipt after functional cases pass.

The sign of the internal calibration is not determined by these expectations:
the positive and negative factors have the same square. Signed output would
need its own observable contract. This example checks the squared reading.

## Model contribution

Fixed order checks eight record combinations at initial construction and in
each refinement round. The first combination matches 1/6 declared record fields.
The unchanged model selects the 6/6 combination first in all five model calls
across the model and one-round runs. These are repeated calls for this fixture.
The numeric grammar and the Gooo policy remain deterministic; the model receives
field names, paired expressions and their Korean/English intent.

Observed prediction times are 13,709, 14,334, 12,792, 17,875 and 13,292 ns.
Resident model tensors occupy 2,096 bytes. The bundled weights occupy 446 bytes,
with SHA256 `049081cc6dc62f3c4230f72e601c1fd421c514fce9cfdf7bc1eb412ca4c5514b`.
Prediction time excludes loading, construction, compilation and execution.
CPU utilization and an end-to-end speedup were not measured. No training or
model downloads occurred; exposure of model training data to these cases is unknown.

The model is the existing
[shared field model](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1)
bundled under `models/shared-qat`. The full predictions and first-candidate
scores are retained in native results and the smaller model-observations file.

## Reproduction and checks

Each mode retains source, policy, cases, source plan, initial construction,
the complete refinement history, final execution and separate evaluation.
The selected original-source/composition pair supports replay. Reproducible
generated files and duplicate per-round receipts were omitted to limit disk use.
After this copy, all four selected programs replayed from the publication with
zero model calls and the same final evaluation counts. See `post-copy-replay.json`.

Native race checks passed for all workbench packages. New checks cover the two
successful modes, bounded controls, inference-free saved execution and a real
compiler JSON failure. The latter reproduced an empty error explanation in the
transport, then passed after the transport included the compiler's structured
diagnostic. Focused vet and formatting checks also passed.

This is one finite graph under four controls. Its functional results, grammar
coverage, field selection and separate evaluation are reported independently.

## 한국어

센서 원시값을 보정해 제곱값을 만들고, 그 값과 임계값 표시를 기록으로 묶는 예제입니다.
숫자식은 주변 계산과 사례를 보고 구성하고, 작은 자체 모델은 기록의 세 필드에서
먼저 시도할 조합을 골랐습니다. 다음 문법으로 넘어갈지는 Gooo 정책이 판단했습니다.

고정 순서와 모델 사용 모두 활동 출력 2/20에서 20/20으로 이어졌고, 별도 평가는 6/6이었습니다.
모델을 쓴 다섯 호출에서는 첫 필드 조합이 맞았습니다. 한 라운드만 허용하거나 대안을
빼면 2/20에 머물렀습니다. 성공한 결과와 멈춘 결과를 함께 보관합니다.

실행 중 잘못된 선언을 넣어보면서 오류 설명이 사라지는 문제도 확인했습니다. 이제
컴파일러가 JSON으로 보낸 원인을 도구의 오류 메시지에서도 읽을 수 있습니다.
