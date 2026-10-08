# Gooo ecosystem workbench

Gooo로 작성한 작은 언어 생태계 도구 모음입니다. 표준 함수, 실행 결과를 읽는
진단 규칙, 새 Gooo 파일을 만드는 프로그램을 실제로 실행합니다.

작업장의 공구 상자를 만드는 단계입니다. 동작과 조립 규칙은 `.gooo`에 두고,
작은 자체 모델이 선언된 후보의 순서를 제안합니다. 컴파일러가 사례를 확인하고
Go 프로그램을 만들어 실행합니다. 파일 저장과 명령 연결은 Go로 구성했습니다.

## 구현한 요소

| 구성 | Gooo가 하는 일 | 사용할 명령 |
| --- | --- | --- |
| 표준 함수 13개 | 정수 범위·최솟값·최댓값, 논리 연산, 텍스트 선택 | `verify` |
| 진단 프로그램 | 부분 충족·관측 부족·잘못된 수·완료를 분기하고 다음 작업 구성 | `diagnose` |
| 전체 조립 반복 | 부품·호출부·평가 결과를 구분하고 Gooo가 다음 시도 한도를 계산 | `construct` |
| 모델 입력 검사 | 입력에서 사라진 구분과 선택기의 남은 오차를 나눠 다음 작업 구성 | `feature-audit` |
| 소스 수정 이어가기 | Gooo의 다음 행동에 따라 사례·시도 한도를 수정하고 다시 조립 | `refine` |
| 소스 조각 바꾸기 | 원문 확인 → 조건부 교체 → 바뀐 길이를 순서대로 구성 | [`splice`](examples/source-splice/README.md) |
| 보정값 보고서 예제 | 숫자식의 빈칸을 채우고 값·임계값 표시·문구를 레코드로 조립 | [네 조건의 실행](examples/calibrated-report/README.md) |
| 재시도 판단 모듈 | 성공·실패·횟수에 따라 재시도 여부와 상한이 있는 대기 시간 구성 | [조립하고 재사용하기](examples/retry-policy/README.md) |
| 시작 도구 | scalar/record/library 요청에 맞는 Gooo 소스와 다음 명령 생성 | `scaffold` |
| API 참조 생성 | 컴파일러가 해석한 공개 이름·시그니처·안정 타입 ID를 문서화 | `reference` |
| 완전성 영수증 | 선언·생성·역관찰·사례·경계·출처를 근거와 함께 단계별 기록 | `receipt` |
| 인보이스 승인 예제 | 레코드 상태와 검토자 조건으로 승인 결과 구성 | `verify`의 3개 고정 사례·저장 재실행 |
| CI 계획 예제 | 변경 파일 묶음을 등록된 확인 항목에 결정론적으로 대응 | `verify`의 12개 고정 사례·24개 활동 출력 |
| 기능 탐색 연결 | JEV의 한·영 기능 카탈로그 결과를 Gooo 평가·재실행과 연결 | `discover`의 출처 바인딩·미해결 단계 기록 |
| VS Code 편집기 | `.gooo` 문법 색상, LSP 진단·완성·이동·이름 변경, 결정론적 포맷, 본문 생성, 앱/라이브러리 프로젝트 시작 | [`editors/vscode`](editors/vscode) |

[표준 함수](recipes/stdlib.gooo)는 `Min`, `Max`, `Clamp`, `AbsSaturating`,
`Sign`, `IsZero`, `InRange`, `And`, `Or`, `Not`, `CoalesceText`, `ChooseText`,
`NonNegative`입니다. 뒤집힌 범위는 두 경계를 정렬합니다. int64 최솟값의
절댓값은 최댓값으로 포화시킵니다. 빈 문자열·한글·줄바꿈·정수 양끝을
포함한 [기대값](recipes/stdlib-cases.json)을 공개합니다.

표준 함수는 한 Gooo 선언 묶음으로 배포하고, `bind`로 연결합니다. 진단과
시작 프로그램의 세 필드에는 한영 의도와 두 후보식이 있습니다. `--model builtin`은
공개 자체 QAT 모델을 연결하며, 옵션을 생략하면 결정론 순서로 탐색합니다.
기본 모델은 이전 가중치를 사용합니다. 별도로 학습한 소스 그래프 모델은 아래 연구에서
명시적인 파일 경로로 연결합니다.

## 시작하기

Go 1.27.1과 [Gooo 0.6.11 개발판](https://github.com/kimjooyoon/meta-ontology-go/releases/tag/v0.6.11-dev)을
사용합니다. `splice`에 필요한 패키지 문자열 연산 수정이 배포 파일에 포함됐습니다.
CI는 배포 소스 `27594824ad628ef0e8362bbc1e0e034aee88e07c`를 고정하고 빌드한 버전과 출처를 확인합니다.
소스의 연산자·값 관계와 의도를 작은 모델에 전달하는 경로도 포함합니다.
`gooo version --build --json`으로 설치 버전과 소스를 확인할 수 있습니다.
설치 후 이 저장소에서:

```sh
go run ./cmd/workbench verify --model builtin --out out/verified
go run ./cmd/workbench scaffold --profile record --model builtin --out out/my-record-project
go run ./cmd/workbench scaffold --profile library --model builtin --out out/my-library
go run ./cmd/workbench diagnose --input examples/partial-composition.json \
  --model builtin --out out/next-work
go run ./cmd/workbench receipt --input out/verified --out out/completeness
go run ./cmd/workbench discover --query '코드 생성은 어떻게 해?' \
  --declaration examples/catalog/operations.gooo --out out/capability-discovery
```

`library`는 `Clamp(Integer) -> Integer` 공개 계약과 Gooo 본문을 포함한
단일 패키지 출발점입니다. 결과의 `activity-generation.json`은 선택된 activity의
본문 생성 기록을 담습니다. `--compiler /path/to/gooo`로 다른 설치 위치를
지정합니다. 모든 출력 폴더는 새 경로입니다. 결정론 실행은 같은 명령에서
`--model builtin`을 빼면 됩니다.

컴파일러를 소스로 준비하는 방법은 [실행 안내](docs/usage.ko.md)에 있습니다.

## 모델에 필요한 구분이 남아 있는지 확인하기

```sh
go run ./cmd/workbench feature-audit \
  --input examples/feature-audit/filename-order.json --model builtin --out out/feature-audit
```

파일 이름 분류의 후보 앞뒤를 여덟 방식으로 바꾼 자료입니다. 현재 모델 입력은
`&&`와 `||`, `input`과 `stem`을 구분하지 않아 전체 특징 배열이 두 종류만 남습니다.
동일한 배열에 서로 다른 선택이 필요하므로, 이 입력 형식만 사용하는 결정론적 선택기는
제공된 여덟 배치 중 최대 2개를 맞힐 수 있습니다. 포함된 모델의 첫 선택은 1/8입니다.

Go는 실제 float32 배열을 비트 단위로 묶고, Gooo의 [평가 규칙](recipes/feature-audit.gooo)이
`preserve-distinguishing-source-facts`를 다음 작업으로 반환합니다. 입력·집계·Gooo 소스·
실행 기록을 함께 저장합니다. [입력 형식과 측정 범위](examples/feature-audit/README.md).

함수 안의 값 출처를 포함하는 v2 입력은 `filename-origin-order.json`으로 검사합니다.
같은 여덟 배치에서 배열이 네 종류로 늘고, 제공된 정답 기준 최대치는 4/8입니다.
`&&`와 `||`는 여전히 합쳐집니다. 이 비교는 입력이 얼마나 구분되는지 측정하며,
v2 모델을 학습하거나 정답률을 측정한 결과는 아닙니다. v2 검사는 모델 없이 실행합니다.

연산자와 값의 순서를 보존한 v3 입력은 `filename-graph-order.json`으로 검사합니다.
같은 여덟 배치가 여덟 배열로 구분됐고, Gooo 평가 함수는 `evaluate-chooser`를 반환합니다.
제공된 입력에서 구분 충돌을 찾지 못했으므로 실제 모델 판단을 측정하라는 뜻입니다.
[v3 전용 학습 가중치](models/graph-chooser-20261008/README.md)는 별도 경로로 제공합니다.

[세 프로그램의 후속 입력 실험](examples/graph-choice-corpus/README.md)은 파일명 처리,
정수 나눗셈, 재시도 판단을 한글·영어·혼합 의도와 여덟 후보 배치로 표현합니다.
총 72행은 세 프로그램의 변형이며, 전체 특징 배열도 72종으로 구분됐습니다.
Gooo가 입력 충돌 여부를 평가하고 실제 선택기 측정을 다음 작업으로 제안합니다.
이 자료에서 출발해 아래 연구에서 프로그램 전체를 분리한 학습과 평가를 진행했습니다.

[요구를 바꾼 조립 실험](publication/intent-contrasts-20261008/README.md)에서는 같은 부품에
여덟 가지 요구를 줬습니다. 예를 들어 파일 이름의 접미사를 남길지 제거할지 바꿉니다.
세 프로그램의 576행이 서로 구분됐으며, 요구 문장을 지우면 24종류로 겹칩니다.
그 정보만 쓰는 결정론적 선택기는 이 자료에서 최대 72/576을 맞힐 수 있습니다.
24개 요구·프로그램 조합을 실제로 생성해 별도 입력 128건과 저장 재실행을 확인했습니다.
이 자료에 이어 [첫 자체 그래프 모델](publication/graph-chooser-20261008/README.md)을 Go로 학습했습니다.
프로그램 하나를 학습에서 뺀 QAT 모델의 첫 선택은 10~28/192행에서 맞았습니다.
24개 조립 과제의 전체 시도 수는 결정론 순서 108회, 학습 자료 내 모델 63회,
해당 프로그램 제외 모델 120회였습니다. 8회 한도에서 세 방식 모두 선언 사례를
충족했고 별도 실행 128건과 저장 재실행을 확인했습니다. 적은 시도 한도에서의
부분 충족, 낮은 전이 성능, 학습 손실과 공개 가중치를 함께 제공합니다.

이 가중치를 바꾸지 않고 [새 소스 수정 도구](publication/dependent-splice-20261008/README.md)에
연결했습니다. 모델은 네 후보, 고정 순서는 여덟 후보를 시도해 조립했고,
두 완성된 결과 모두 별도 입력 542개와 저장 재실행을 통과했습니다.
모델의 첫 제안은 틀렸으며 두 번만 시도하면 맞는 필드 수는 고정 순서보다 적었습니다.
새 프로그램 하나의 관측입니다. 반환된 소스로 Gooo 함수도 수정해 실제로 실행했습니다.

## Gooo가 Gooo 파일 만들기

[시작 프로그램](recipes/starter.gooo)은 템플릿을 Gooo 지역 변수에 보관하고,
모델에 짧은 후보식과 의도를 전달합니다. 선택된 본문을 네이티브 프로그램으로
실행하면 `filename`, `source`, `next`를 가진 프로젝트 계획이 나옵니다.
Go 연결 코드가 `main.gooo`를 저장하고, 컴파일러로 검사와 본문 생성을 실행합니다.

생성 결과에는 `main.gooo`, 프로젝트 계획, 생성 기록과 `Identity` 본문의
Go 생성 결과가 남습니다. 만든 파일을 수정해 다음 프로그램의 시작점으로 씁니다.
템플릿의 Gooo 본문 구분자는 문자열 안에서 `\x60`으로 표현합니다.

## 실행 결과를 다음 작업으로 연결하기

`diagnose`는 `body-compose` 결과와 `gooo package execute` / `resume` / `replay`
실행 영수증을 읽어 실제 값과 기대값을 다시 비교합니다. 보조 함수와 루트 활동의
타입 탈락 기록도 함께 전달합니다. 부분 충족 여부와 다음 작업은
[진단 Gooo 소스](recipes/diagnostics.gooo)가 계산합니다.

`observation.json`의 `unit`은 집계 단위를 표시합니다.

- `activity_outputs`: 숫자·문자·레코드가 섞인 경우, 기대값이 있는 활동 출력 전체를 비교합니다.
- `record_fields`: 모든 기대 출력이 비어 있지 않은 레코드이고 실제 값에 추가 필드가 없으면 필드별로 비교합니다.
- `provided_counts`: 사용자가 직접 제공한 `passed` / `total`을 사용합니다.

공개 부분 결과는 **활동 출력 6/14·타입 탈락 1개**입니다. 이전 도구는 레코드의
17/21필드만 집계해, 함께 실행한 문자열 출력 7개를 분모에서 빠뜨렸습니다.
이제 두 종류를 모두 포함하며, 진단 결과는 `partial`과 `repair-and-replay`입니다.
기대값이 없는 실행은 0/0으로 남고 `unobserved`로 분류됩니다.

```sh
go run ./cmd/workbench diagnose --input /path/to/package-execution.json \
  --model builtin --out out/package-diagnosis
```

원래 입력은 `captured-input.json`에, 그 전체의 SHA256은 `observation.json`에
보관합니다. 상세가 1,024바이트를 넘으면 집계 단위·불일치 수·타입 탈락 수·원본
해시를 전달합니다. 개별 값은 원본 기록에서 확인할 수 있습니다. 모델을 생략해도
같은 Gooo 진단 규칙으로 실행됩니다. 다음 행동은 제안이며 후속 코드 수정은 별도 실행입니다.

### 조립 기록에서 다음 행동 고르기

`diagnose`는 레코드 조립 기록이 있으면
[다음 행동 프로그램](recipes/next-steps.gooo)도 실행합니다.
기본 진단은 실행 결과를 요약하고, `construction_next_steps`는 각 활동의
조립 사례·시도 수·후보 수·기록된 예산을 보고 구체적인 다음 행동을 제안합니다.

| 관측 | Gooo가 제안하는 행동 |
| --- | --- |
| 조립 사례가 남았고 후보와 예산도 남음 | `resume-candidates` — 이어가기 정책을 정해 계속 조립 |
| 후보는 남았고 예산을 다 씀 | `raise-attempt-budget` — 허용된 범위에서 소스의 시도 한도 확대 |
| 후보를 모두 확인함 | `expand-declared-choices` — 남은 사례를 표현할 선택지 보완 |
| 조립 사례는 맞았지만 실행 사례가 남음 | `add-runtime-cases-to-construction` — 남은 실행 사례를 조립 조건에 반영 |
| 제공된 실행 기대값을 모두 만족 | `observe-new-inputs` — 새로운 입력 관측 |
| 시도 예산이 기록되지 않음 | `inspect-attempt-budget` — 선언한 예산 확인 |

후보 순서의 길이와 시도 예산은 별개입니다. 새 컴파일러의 `attempt_budget`에서
소스에 선언한 예산을 읽습니다. 이전 기록은 저장된 정책 입력을 사용하고, 두 정보가
모두 없으면 확인 필요 상태로 남깁니다. `budget_source`는 각각 `source_contract`,
`policy_observation`, `unavailable`을 기록합니다. 소스 예산과 정책 관측이 다르면
불일치로 진단하며 소스 예산을 보존합니다. 타입 검사에서 탈락한
후보도 이미 사용한 시도로 셉니다. 선택된 활동 ID와 입력 원본 해시를 함께 남깁니다.

[시도 한도 관측 예제](examples/source-budget/README.md)는 같은 여덟 후보에 한도
1·3·8·16을 적용합니다. [실행 기록](publication/source-budget-20261008/README.md)에
고정 순서와 자체 소형 모델의 결과, Gooo가 고른 다음 행동을 함께 공개했습니다.

`construction-next-steps.json`에 각 제안이, `construction-next/`에 Gooo 소스와
실행 결과가 저장됩니다. 이 프로그램의 분기는 Gooo 코드로 실행하며 새 모델 호출은
없습니다. 기본 진단 프로그램을 조립할 때는 기존의 선택적 모델을 사용할 수 있습니다.
각 활동의 제안은 전체 실행에서 실패를 일으킨 활동을 확정하는 인과 분석까지 포함하지
않습니다. 실제 이어가기는 컴파일러가 원래 소스와 기록을 다시 확인한 뒤 수행합니다.

### 부품을 맞춰도 전체가 틀리는 경우

`body-construct`가 있는 컴파일러에서는 다음 명령으로 전체 조립을 반복할 수 있습니다.
[Gooo 규칙](recipes/joint-next.gooo)이 결과를 읽고 다음 시도 한도를 계산하며,
Go 실행부가 정해둔 상한 안에서 그 제안을 실행합니다.

```sh
go run ./cmd/workbench construct --compiler /path/to/gooo \
  --source examples/joint-diagnostics/source.gooo --entry Main \
  --construction-cases examples/joint-diagnostics/construction-cases.json \
  --evaluation-cases examples/joint-diagnostics/evaluation-cases.json \
  --max-program-budget 8 --max-rounds 4 --out out/joint-fixed
```

기본 순서는 결정론적입니다. 자체 그래프 모델을 쓰려면
`--model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json`을
추가합니다. 모델은 각 조립 실행의 초기 후보 순서를 제안합니다. 진단 규칙과
다음 한도 계산은 모델 없이 Gooo 코드로 실행됩니다.

각 회차는 처음부터 다시 조립합니다. 예를 들어 1·2·4·8회 한도를 순서대로
사용하면 최대 15번의 프로그램 시도가 생깁니다. `joint-loop.json`은 회차별
실제 시도 수와 다음 행동을, `round-N.json`은 원래 결과를 남깁니다.
마지막 `round-N/`에는 선택한 Gooo 소스와 다시 실행할 조립 기록이 있습니다.

기존 결과만 살펴보려면 `diagnose --input /path/to/body-construct-result.json`을
사용합니다. 이 경로는 다음 관측을 분리합니다.

- 처음 각 부품을 준비할 때의 사례와 시도 수
- 각 전체 프로그램 조합의 부품 검사와 호출부 검사
- 선택한 프로그램의 별도 평가, 조립에 소비된 입력과 그 밖의 입력 수

별도 평가가 모두 맞아도 부품이나 호출부의 조건이 남으면 완료로 안내하지 않습니다.
평가에서 새로 실패한 사례는 다음 조립 조건으로 옮기도록 안내하며, 새 평가 입력도
필요하다고 표시합니다. 후보 확장과 기대값 수정은 현재 자동 반복의 범위에 포함되지
않습니다. 저장 기록의 실제 값을 다시 세는 작업과 원래 프로그램을 재실행하는 작업은
별개입니다. [고정 관측과 검증 계획](examples/joint-diagnostics/PLAN.md)을 함께 확인할 수 있습니다.

### 진단에서 소스 수정까지 이어가기

`refine`는 조립할 활동과 피드백 사례, Gooo 정책을 받아 첫 프로그램을 실행합니다.
Gooo가 한도 확대나 사례 추가를 제안하면 컴파일러의 `body-refine`를 실행합니다.
그 안의 Gooo 정책이 다음 한도와 유지할 결과를 정하며, 선택된 프로그램을 다시
실행한 뒤 다음 행동도 갱신합니다. [실행 예제](examples/source-refinement/README.md).

정수 표현식의 빈칸을 문법으로 채우는 [탐색 예제](examples/search-refinement/README.md)도
같은 수정 루프를 사용합니다. 숫자식 결과를 자체 소형 모델이 조립한 레코드에 연결할
수 있습니다. 현재 목록을 다 시도한 경우와 후보 수 제한으로 표현식이 제외된 경우를
Gooo가 구분해 다음 행동을 제안합니다.

소스에 `search_alternative`를 선언하면 `refine --search-policy`로 그 제안을
실제 탐색 설정 변경에 연결할 수 있습니다. Gooo 정책이 후보 제한을 넓히거나
허용된 다른 문법으로 전환하며, 각 수정본과 실행 결과를 남깁니다.
[선언과 실행 예제](examples/search-policy/README.md).

[보정값 보고서](examples/calibrated-report/README.md)는 그 흐름을 두 활동으로 연결한
작은 센서 예제입니다. 숫자식은 관측한 값으로 후보를 만들고, 자체 모델은 결과 레코드의
세 필드 후보를 추천합니다. Gooo 정책이 다음 문법으로 옮길지 고릅니다. 고정 순서,
모델 사용, 한 라운드 제한, 대안 문법 제거를 같은 입력으로 비교할 수 있습니다.

```sh
go run ./cmd/workbench refine --compiler ./.gooo \
  --source examples/source-refinement/source.gooo --activity Select \
  --cases examples/source-refinement/feedback-cases.json \
  --evaluation-cases examples/source-refinement/evaluation-cases.json \
  --policy examples/source-refinement/policy.gooo \
  --max-attempts 8 --max-rounds 4 --out out/refined
```

`--model builtin`을 더하면 자체 소형 모델이 조립 후보 순서를 정합니다. 모델을
생략하면 고정 순서로 실행합니다. 원본 파일은 보존하고 수정본을 출력 폴더에 남깁니다.
처음 실행과 수정 루프의 모델 호출 수를 따로 기록하며, 수정 루프는 첫 조립을 다시
실행합니다. 최종 평가는 선택이 끝난 뒤 실행하므로 정책의 입력에 포함되지 않습니다.
한도 안에서 해결되지 않은 사례는 `PROGRESS`로 남습니다.

## 작은 도메인 사례: 인보이스 승인

[인보이스 승인 프로그램](examples/invoice-approval/approval.gooo)은 인보이스가
제출됐는지와 검토자가 활성 상태인지 확인해 승인 결과와 이유를 만듭니다.
고정 사례는 승인·미제출·비활성 검토자 세 경로를 실행하고, `ApproveInvoice`와
결과 전달 활동 `Echo`의 실제 값을 검사한 뒤 저장된 프로그램을 재실행합니다.
이는 작동하는 예제 도메인으로, 실제 회계 정책이나 결제 시스템에 대한 주장은
아닙니다. 전체 검증과 완전성 영수증에서 함께 실행됩니다.

## 변경 파일에서 CI 계획 만들기

[CI 계획 프로그램](recipes/ci-plan.gooo)은 등록된 소수의 파일 묶음에 `go`, `docs`,
`yaml` 확인 항목을 연결합니다. 등록되지 않은 파일 묶음은 `UNKNOWN`으로 남고,
잘못된 입력은 `FAIL_CLOSED`로 표시됩니다. 고정 사례는 12개이며 `PlanCI`와 결과 전달
`Echo`의 출력 24개와 레코드 필드 96개를 비교하고 저장 재실행합니다. 이는 기존
컴파일러 예제를 바탕으로 한 제한된 Gooo 프로그램입니다. 확인 명령을 실행하거나
일반 저장소의 CI 계획을 자동 추론하지 않습니다.

```sh
go run ./cmd/workbench verify --compiler ./.gooo --out out/verified
go run ./cmd/workbench receipt --compiler ./.gooo --input out/verified --out out/completeness
```

직접 관측 수를 전달할 때는 `passed`, `total`, `rejected`, `detail` JSON을 사용합니다.
분류·작업 선택 규칙은 [Gooo 진단 본문](recipes/diagnostics.gooo)에 있습니다.
이 진단 요청의 네이티브 사례는 전달한 상세 문장을 그대로 되돌리는 관측을
기대값으로 둡니다. 새 요청의 진단 자체에 추가 정답을 붙였다고 세지 않습니다.

## 선언에서 API 참조 만들기

`reference`는 컴파일러의 `operation-interface`를 읽어 공개 활동명,
입출력 시그니처, 안정 타입 ID가 있는 문서를 만듭니다. 문서 템플릿은
[Gooo 본문](recipes/reference.gooo)이 생성합니다. 컴파일러 JSON이 선언된
이름과 타입을 제공하고 Go가 이를 Gooo에 전달합니다. 실제 값, 기대값 비교와
저장 재실행을 함께 보관합니다.

```sh
go run ./cmd/workbench reference --compiler ./.gooo \
  --package examples/catalog --entry ApproveInvoice --out out/api-reference
```

`reference.md`는 선언된 인터페이스만 설명합니다. 구현 동작이나 자연어 설명을
추론해 채우지 않습니다. 문서는 컴파일러가 낸 선언 지문과 인터페이스 지문을
함께 표시하며, 원본 JSON과 Gooo 실행 기록도 결과에 저장합니다.
고정 예제와 실제 출력을 [API 참조 관측](publication/api-reference-20261005)에서 확인할 수 있습니다.

## Gooo 기능 탐색과 사용 사례 경계

`discover`는 [Gooo-jev](https://github.com/kimjooyoon/gooo-jev)의 자연어 질의 결과와
선언 출처 관측을 읽고, [Gooo 평가 프로그램](recipes/capability-assessment.gooo)이
카탈로그 상태·선언 바인딩·다음 미해결 단계를 구성합니다. 한국어·영어 질문을
지원하며 `AVAILABLE`, `DEFERRED`, `UNKNOWN`을 그대로 보존합니다.

```sh
go run ./cmd/workbench discover --query '코드 생성은 어떻게 해?' \
  --declaration examples/catalog/operations.gooo --out out/capability-discovery
```

출력 폴더에는 질의, 선언, JEV 탐색 trail·guide, Gooo 본문 조립과 저장 재실행,
평가 JSON이 남습니다. 선언이 연결된 카탈로그 결과는 `real_use_case_coverage=PROGRESS`,
첫 미해결 단계는 `generation`으로 표시합니다. 이는 코드 생성 능력을 증명하지
않으며, 실제 생성과 역관찰이 완료되기 전까지 `PASS`로 승격하지 않습니다.
JEV 카탈로그 자체는 결정론적이며 외부 모델/provider 호출은 없습니다.
원 입력·trail·guide·Gooo 소스·실제 출력·재실행은
[공개 능력 탐색 관측](publication/capability-discovery-20261005)에 있습니다.

## 완전성 영수증

검증 실행이 끝난 뒤 `receipt` 명령은 그 결과의 Gooo 선언, 생성 파일,
저장 재실행, 유한 사례, 컴파일러 출처를 읽어 근거 해시와 함께 영수증을
만듭니다. 데이터 구조와 `first_unresolved_stage` 규칙은
[Gooo 계약](recipes/completeness.gooo)이 소유합니다. 해당 엔티티의 안정 ID는
`gooo://completeness/domain-completeness-receipt/v1`입니다.

```sh
go run ./cmd/workbench verify --compiler ./.gooo --model builtin --out out/verified
go run ./cmd/workbench receipt --compiler ./.gooo --input out/verified --out out/completeness
```

현재 workbench 예제에서는 선언·생성·역관찰·출처가 확인돼 `PASS`, 작성된
유한 예제만 확인돼 `use_case=PROGRESS`, 별도로 계측하지 않은 외부 효과 경계는
`UNKNOWN`으로 남습니다. 영수증은 단일 완성도 점수를 만들지 않습니다. 모델은
이 결정론적 측정에 개입하지 않으며, 모델이 선택한 조립도 별도 출처 정보에
그대로 남습니다. 검증 실행 시점의 저장소 HEAD를 기록하고, 다른 체크아웃에서
영수증을 만들면 출처를 `UNKNOWN`으로 낮춥니다. 이 기능은 공개 이슈 [#1023](https://github.com/kimjooyoon/meta-ontology-go/issues/1023)의
첫 적용 사례이고, 자연어 탐색과 여러 저장소 간 사용 사례 연결은 아직 범위 밖입니다.

## 현재 확인 범위

기대값과 실제 값을 별도로 비교하며, 정수는 JSON 숫자의 원래 자릿수를 유지합니다.
표준 함수, 두 생태계 프로그램, 두 제한된 예제의 유한 사례를 결정론·모델 경로에서 실행하고,
저장한 조립을 추가 모델 연결 없이 재실행합니다. 원본과 분모는
[공개 관측](publication/initial-20261005)에 있습니다.

이번 도구는 정해진 함수와 제한된 파일 묶음에서 동작합니다. 편집기는 컴파일러
LSP를 통해 진단과 의미 완성을 제공합니다. 다중 파일 공개 API 참조와 실제
컴파일러 명령은 별도로 제공합니다.

VS Code 지원은 별도 parser를 복제하지 않습니다. 편집기는 설치된 컴파일러의
`gooo lsp`를 표준 입출력으로 실행합니다. 진단·자동완성·hover·정의 이동·참조·
이름 변경·문서 기호·semantic token은 컴파일러가 분석한 결과를 사용합니다.
본문 생성은 명시적 편집기 명령으로 남겨 두어 입력 중 소스를 바꾸지 않습니다.
설치한 `gooo`가 `lsp` 명령을 제공해야 합니다.

## 연결된 연구와 코드

- [Gooo 컴파일러](https://github.com/kimjooyoon/meta-ontology-go): 선언·IR·타입 검사·코드 생성·실제 실행.
- [Go 판단 SDK](https://github.com/kimjooyoon/gooo-decision-runtime): 작은 모델의 고정된 입력 계약.
- [포함한 자체 모델과 원 출처](models/shared-qat/README.md): 2,072개 매개변수·삼진 QAT, 공개 MIT 가중치.
- [SKETCH](https://people.csail.mit.edu/asolar/papers/thesis.pdf)의 제한된 합성 공간과
  [PROV-O](https://www.w3.org/TR/prov-o/)의 생성·사용 관계에서 배웁니다. 이 저장소에서는
  소스·선택·실제 값을 연결해 다음 작업에 전달하는 부분을 구체적으로 실험합니다.

MIT license. 자체 모델의 학습 코드·계획·가중치·분할별 결과를 함께 제공합니다.
