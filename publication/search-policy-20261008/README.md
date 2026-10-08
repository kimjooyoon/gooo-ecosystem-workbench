# Gooo chooses the next declared construction path

Clean compiler `765918d9b0da11359ebbbe88c517831e4cebeb1f`, clean experiment
`f428428a91b43e18d2c50a462f34c5d61bae8692`, Go 1.27.1, Darwin arm64.
[Summary](summary.json) · [file hashes](sha256.json) ·
[source and reproduction](../../examples/search-policy/README.md).

The source starts with `input + hole` and permits two later search settings:
a wider candidate list, then a grammar that uses the surrounding expression.
A Gooo program reads failed examples and search counts, selects the next setting,
and continues construction. The original function body and intent persist.
Each transition and revised source is retained.

## Six native runs

| Run | Feedback before → after | Separate final evaluation | Refinement rounds | Model calls: initial + refinement |
| --- | --- | --- | --- | --- |
| Legacy budget/case policy | 0/2 → 0/2 | 0/2 | 2 | 0 + 0 |
| Source-declared search alternatives | 0/2 → 2/2 | 2/2 | 4 | 0 + 0 |
| Integer → record, fixed ordering | 0/4 → 4/4 | 4/4 | 4 | 0 + 0 |
| Integer → record, own small model | 0/4 → 4/4 | 4/4 | 4 | 1 + 4 |
| Two-round limit | 0/2 → 0/2 | 0/2 | 2 | 0 + 0 |
| No declared alternatives | 0/2 → 0/2 | 0/2 | 2 | 0 + 0 |

Each run uses two adaptive feedback inputs and two separate evaluation inputs.
Mixed runs measure two activity outputs per input. Final evaluation occurs after
source selection and is withheld from both Gooo decision programs. The initial
diagnostic construction precedes the counted refinement rounds. Partial results
retain `PROGRESS` status and their unresolved expectations.

The native input-separation report marks both scalar evaluation inputs disjoint
from recorded construction inputs. For each mixed run, one root input has an
overlapping downstream record input and one is disjoint. The table counts both
activity outputs for both inputs; 4/4 therefore includes that overlap. Withholding
evaluation from the policy and checking input separation are distinct observations.

The successful runs follow these four decisions:

1. `INCORPORATE`: add a failed direct input to construction examples and try two
   candidates. The original declared expectation remains.
2. `ADVANCE_SEARCH wider`: the current cap keeps two expressions and omits four;
   select the source's wider setting, capped at 16 candidates and eight attempts.
3. `ADVANCE_SEARCH contextual`: all six retained offset expressions still miss
   the inner value needed by `input + hole`; select the declared residual grammar.
4. `STOP`: that grammar constructs the inner constant `1`, satisfying the two
   feedback inputs. Evaluate the retained source on the separate final inputs.

The legacy run has the same alternatives in its source but its policy never
selects them. The two-round control stops before a grammar transition. The
no-alternatives control has no permitted next setting. These controls show why
the new policy path matters for this fixture.

## Model contribution and replay

The unchanged [own shared record model](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1)
ranks the three Integer/Boolean/String field choices in the second activity.
Fixed ordering checks eight record combinations in every refinement round;
the model picks the passing combination first in each of the four rounds.
The integer grammar transitions follow the Gooo policy in both runs, so both
still use four rounds.

The four model prediction times are 15,833, 19,833, 19,958 and 18,375 ns.
Resident model tensors occupy 2,096 bytes. These are prediction and tensor
observations only; loading, compilation, execution and source revision have
additional costs. CPU utilization and end-to-end speedup were not measured here.
No training occurred, and model training exposure to these cases is unknown.

All six saved programs were replayed after publication cleanup. Each
`post-prune-replay.json` preserves its final evaluation result with zero new
model calls. The selected source and composition are sufficient to regenerate
the execution files.

## Evidence and scope

Sources, policies, input cases, exported search plans, Gooo decisions, full
native results, every source revision and selected compositions are retained.
Byte-identical duplicate refinement receipts and reproducible generated Go,
module, realized-source and runtime copies were removed to limit disk use.
`refinement-result.json` contains the complete round history. The source and
composition pairs remain usable for replay.

The compiler carries up to four declared alternatives through parsing,
formatting and semantic lowering. The runner visits each grammar/cap pair at
most once through these transitions, respects the caller's attempt bound, and
supports at most eight rounds. The policies in these runs stop earlier.
The experiment covers the supplied arithmetic and record cases. Broader
program structures, general natural-language accuracy and correctness for
unseen inputs require additional evidence.

## 한국어

앞선 실험에서는 Gooo가 “후보 수를 늘려보자”, “다른 식을 만드는 방법이 필요하다”까지
제안했습니다. 이번에는 소스에 다음 방법을 적어두고, Gooo 정책이 그 제안을 실행하게
연결했습니다. 후보 수를 늘려도 맞지 않으면 허용된 다음 문법으로 넘어갑니다.

모델 없이도 이 흐름은 진행됩니다. 작은 모델은 뒤에 이어지는 레코드 조립에서
어느 조합부터 확인할지 돕습니다. 시도할 대안이나 라운드가 부족하면 부분 결과를
남깁니다. 이번 결과는 이 작은 예제에서 조립을 계속하는 경로가 실제로 작동하는지
확인한 기록입니다.
