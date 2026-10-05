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
| 시작 도구 | scalar/record 요청에 맞는 Gooo 소스와 다음 명령 생성 | `scaffold` |
| API 참조 생성 | 컴파일러가 해석한 공개 이름·시그니처·안정 타입 ID를 문서화 | `reference` |
| 완전성 영수증 | 선언·생성·역관찰·사례·경계·출처를 근거와 함께 단계별 기록 | `receipt` |

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

Go 1.27.1과 Gooo 컴파일러가 필요합니다. 이번 관측의 컴파일러 소스는
[`f144dddb`](https://github.com/kimjooyoon/meta-ontology-go/tree/f144dddb8261b9b525181dee510afc65f1603153)입니다.
이미 `gooo`가 설치됐다면 이 저장소에서:

```sh
go run ./cmd/workbench verify --model builtin --out out/verified
go run ./cmd/workbench scaffold --profile record --model builtin --out out/my-record-project
go run ./cmd/workbench diagnose --input examples/partial-composition.json \
  --model builtin --out out/next-work
go run ./cmd/workbench receipt --input out/verified --out out/completeness
```

`--compiler /path/to/gooo`로 다른 설치 위치를 지정합니다. 모든 출력 폴더는
새 경로입니다. 결정론 실행은 같은 명령에서 `--model builtin`을 빼면 됩니다.

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

`diagnose`는 기존 `body-compose` 결과에서 실제 값과 기대값을 다시 비교합니다.
레코드 필드가 있으면 그 충족 수를 사용하고, 타입 탈락 이유와 남은 값도 전달합니다.
공개 부분 결과는 **17/21필드·타입 탈락1개**이며, Gooo 진단 프로그램이
`partial`과 `repair-and-replay`를 반환합니다. 다음 작업을 받는 프로그램은
`diagnostic.json`과 원래 조립 기록을 함께 사용할 수 있습니다.

원래 입력은 `captured-input.json`에 보관합니다. 상세 문장이 현재 컴파일러의
1,024바이트 입력 범위를 넘으면, 남은 필드 수·타입 탈락 수·원본 해시를 담은
요약을 전달합니다. 원본 기록으로 개별 값을 다시 확인할 수 있습니다.

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
그대로 남습니다. 이 기능은 공개 이슈 [#1023](https://github.com/kimjooyoon/meta-ontology-go/issues/1023)의
첫 적용 사례이고, 자연어 탐색과 여러 저장소 간 사용 사례 연결은 아직 범위 밖입니다.

## 현재 확인 범위

기대값과 실제 값을 별도로 비교하며, 정수는 JSON 숫자의 원래 자릿수를 유지합니다.
표준 함수와 두 생태계 프로그램의 유한 사례를 결정론·모델 경로에서 실행하고,
저장한 조립을 추가 모델 연결 없이 재실행합니다. 원본과 분모는
[공개 관측](publication/initial-20261005)에 있습니다.

이번 도구는 정해진 함수와 두 프로젝트 형태에서 동작합니다. 다음에는 다중 파일
공개 API 참조와 실제 컴파일러 진단·편집기 연결을 넓힙니다. 함수 수나 반복
호출 수를 서로 다른 실험 방식으로 세지 않습니다.

## 연결된 연구와 코드

- [Gooo 컴파일러](https://github.com/kimjooyoon/meta-ontology-go): 선언·IR·타입 검사·코드 생성·실제 실행.
- [Go 판단 SDK](https://github.com/kimjooyoon/gooo-decision-runtime): 작은 모델의 고정된 입력 계약.
- [포함한 자체 모델과 원 출처](models/shared-qat/README.md): 2,072개 매개변수·삼진 QAT, 공개 MIT 가중치.
- [SKETCH](https://people.csail.mit.edu/asolar/papers/thesis.pdf)의 제한된 합성 공간과
  [PROV-O](https://www.w3.org/TR/prov-o/)의 생성·사용 관계에서 배웁니다. 이 저장소에서는
  소스·선택·실제 값을 연결해 다음 작업에 전달하는 부분을 구체적으로 실험합니다.

MIT license. 새로운 가중치 학습은 이번 저장소의 작업에 포함하지 않았습니다.
