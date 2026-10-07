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
모델 가중치와 학습량은 유지했습니다.

## 시작하기

Go 1.27.1과 Gooo 컴파일러가 필요합니다. 현재 CI와 Gooo 본문 생성 연동은
[`884d4409`](https://github.com/kimjooyoon/meta-ontology-go/tree/884d44096455aa8a47ac63e378b7a05f10665bfe)을 기준으로 확인합니다.
이미 `gooo`가 설치됐다면 이 저장소에서:

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

MIT license. 새로운 가중치 학습은 이번 저장소의 작업에 포함하지 않았습니다.
