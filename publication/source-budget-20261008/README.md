# Source budgets through Gooo construction and diagnosis

This October 8 observation uses clean compiler
`cdb60e90f911d53d7c02790579e5f72941a2b71a` and clean experiment driver
`a46704d8d26c268ecacb175abb881751bdf6717a`, both built with Go 1.27.1.
[Summary](summary.json), [driver](../../experiments/source-budget/main.go),
[source and reproduction](../../examples/source-budget/README.md).

The same record program has eight candidate combinations. Its source attempt
budget is varied independently, and no separate assembly policy is supplied.
Each result contains `attempt_budget`; the Go adapter transports that observation
to `recipes/next-steps.gooo`, which selects the diagnostic action.

| Ordering | Source budget | Attempted / available | Construction cases | Native outputs | Gooo next action |
| --- | ---: | --- | --- | --- | --- |
| Deterministic | 1 | 1 / 8 | 2/5 | 6/14 | Expand declared choices or budget |
| Deterministic | 3 | 3 / 8 | 2/5 | 6/14 | Expand declared choices or budget |
| Deterministic | 8 | 8 / 8 | 5/5 | 14/14 | Observe new inputs |
| Deterministic | 16 | 8 / 8 | 5/5 | 14/14 | Observe new inputs |
| Own compact model | 1 | 1 / 8 | 5/5 | 14/14 | Observe new inputs |
| Own compact model | 3 | 1 / 8 | 5/5 | 14/14 | Observe new inputs |
| Own compact model | 8 | 1 / 8 | 5/5 | 14/14 | Observe new inputs |
| Own compact model | 16 | 1 / 8 | 5/5 | 14/14 | Observe new inputs |

The model ranks mask 7 first in these four runs. Its unchanged weights are the
public [own shared record model](https://huggingface.co/asketeddy/gooo-record-shared-field-tiny-v1),
also retained under `models/shared-qat` in this repository. Each model construction
makes one prediction with 2,096 bytes of resident tensors. The exact weight,
metadata and source digests and prediction timings remain in each raw result.
Diagnostic programs make zero model calls in both arms.

Seven runtime inputs produce fourteen named outputs. Five inputs also occur in
construction cases; two are additional inputs. Model-training exposure is unknown.
These observations establish this program's behavior and budget transport.
No general accuracy, timing advantage or host CPU utilization is inferred.

## Saved receipt compatibility

The previously published compiler receipts from
[`workspace-continuation-20261008`](https://github.com/kimjooyoon/meta-ontology-go/tree/322381429d275ffb36660a16ce63fd3621a972da/docs/research/workspace-continuation-20261008)
omit `attempt_budget`. The new compiler [replays the completed model receipt](legacy-package-replay.json)
at 4/4 native expectations with zero new predictions. It also
[continues the saved model checkpoint](legacy-package-resume.json) to 4/4,
records the source budget of eight in the successor, and makes zero new predictions.
The original published predecessor remains unchanged.

Retained artifacts include each source, full native result, replayable composition,
diagnostic output and next-step Gooo program with its cases and native result.
Redundant generated Go, duplicate copied sources and temporary module files are
omitted from publication. The driver recreates them in a fresh directory.

## 한국어

후보가 여덟 개여도 이번에 하나만 시도하도록 정할 수 있고, 시도 한도를 열여섯으로
정해도 실제 후보는 여덟 개뿐일 수 있습니다. 이 차이를 기록에 남겨 Gooo 프로그램이
읽도록 연결했습니다. 예산을 알아내기 위해 별도 정책을 붙일 필요가 줄었습니다.
다음 행동은 Gooo로 실행했고, 코드 수정이나 예산 확대는 이 관측 실험에서 수행하지
않았습니다. 이어가기와 소스 수정은 각각의 명시된 실행 경로에서 처리합니다.
