# 실행 안내

## Gooo 컴파일러 준비

Go 1.27.1을 사용합니다. [Gooo 0.6.15 개발판](https://github.com/kimjooyoon/meta-ontology-go/releases/tag/v0.6.15-dev)의
운영체제별 실행 파일에 여러 빈칸의 호출 기반 조립과 잘못된 빈칸 후보 이후의 탐색이 포함되어 있습니다.
설치했다면 아래 명령의 `--compiler ./.gooo`에 설치한 실행 파일 경로를 지정합니다.
`gooo`가 명령 경로에 있다면 다음처럼 바로 시작할 수 있습니다.

```sh
gooo version --build --json
go run ./cmd/workbench construct --compiler gooo \
  --source examples/caller-source-fill/source.gooo --entry Main \
  --construction-cases examples/caller-source-fill/initial-cases.json \
  --evaluation-cases examples/caller-source-fill/evaluation-cases.json \
  --holdout-cases examples/caller-source-fill/holdout-cases.json \
  --max-program-budget 4 --max-rounds 4 --out out/source-fill-public
```

CI와 같은 소스를 직접 빌드할 수도 있습니다.
아래 명령은 작업장 저장소의 루트에서 실행하며 `.compiler`가 없는 상태를 기준으로 합니다.

```sh
git init .compiler
git -C .compiler remote add origin https://github.com/kimjooyoon/meta-ontology-go.git
git -C .compiler fetch --depth 1 origin dc75f59fbee16e776dca13288efe1036d1bb5fab
git -C .compiler switch --detach FETCH_HEAD
GOTOOLCHAIN=go1.27.1 go -C .compiler build -trimpath -o ../.gooo ./cmd/gooo
./.gooo version --build --json
go run ./cmd/workbench verify --compiler ./.gooo --model builtin --out out/verified
```

0.6.15는 조건식·대입식 묶음, 정수식·레코드 조립을 호출 결과로 고릅니다.
잘못된 정수 계산식과 빈칸 후보의 타입·학습용 계산 오류는 이유를 남기고 다음 후보를 시도합니다.
패키지 이름과 import를 Gooo 소스에서 읽는 설정과 `splice`용 문자열 연산도 포함합니다.

[다섯 후보 예제](../examples/caller-fill-rejection/README.md)는
탈락한 후보를 시도 한도에 포함하고, 사례를 실행해 얻은 점수와 따로 기록하는 과정을 보여줍니다.
실제 호출 프로그램의 실행 오류·시간 초과·취소는 요청을 종료합니다.
이 경로의 실패 기록은 [다음 개선 사례](https://github.com/kimjooyoon/meta-ontology-go/wiki/Current-Status)에 남겼습니다.
소스 그래프 입력, 제곱식 탐색·정수 나눗셈·미사용 지역 변수 처리도 사용할 수 있습니다.
`version --build --json`의 버전 문자열은 `0.6.15-dev`이며, 소스 리비전은
위 고정한 리비전과 같습니다. 공개 파일도 같은 소스에서 빌드했습니다.
[버전 사용 안내](https://github.com/kimjooyoon/meta-ontology-go/blob/v0.6.15-dev/docs/releases/0.6.15-dev.md)와
[배포·설치 상태](https://github.com/kimjooyoon/meta-ontology-go/wiki/Current-Status)에서
실제 관측과 지원 범위를 확인합니다.

## 여러 빈칸을 호출 결과로 조립하기

```sh
go run ./cmd/workbench construct --compiler ./.gooo \
  --source examples/caller-source-fill/source.gooo --entry Main \
  --construction-cases examples/caller-source-fill/initial-cases.json \
  --evaluation-cases examples/caller-source-fill/evaluation-cases.json \
  --holdout-cases examples/caller-source-fill/holdout-cases.json \
  --max-program-budget 4 --max-rounds 4 --out out/source-fill
```

처음에는 정상 입력 한 개로 조립합니다. 상한에 도달한 입력에서 틀리면
Gooo 피드백 규칙이 그 사례를 다음 조립 조건에 추가합니다. 각 회차의 시도 한도는
1 → 1 → 2 → 4이고 실제 시도 수는 1 + 1 + 2 + 3 = 7입니다.
마지막 별도 평가는 선택이 끝난 뒤 실행합니다.

`joint-loop.json`의 `local_passed/local_total`은 부품의 학습용 사례,
`caller_passed/caller_total`은 호출부 사례입니다. `fill_holdout_passed/total`은
소스에 적힌 별도 평가이며 선택 점수에 더하지 않습니다. 최종 입력의 결과는
`final_evaluation.passed/total`에서 봅니다. 조립에 사용된 입력 수와 다른 입력 수도
별도로 남습니다. 이 값들은 제공한 사례의 충족 수를 뜻합니다.

`--model`은 레코드 후보를 고르는 모델이고, `--fill-model /path/to/model.json`은
초기 빈칸 조립용 operation-classifier 모델입니다. 둘은 입력 형식이 다릅니다.
생략하면 소스 후보 순서로 진행합니다. 이번 관측은 기존 그래프 모델을 레코드
선택에 사용했고, 빈칸 후보는 결정론적으로 골랐습니다.

## 결과 파일 읽기

`summary.json`은 실제 기대값 충족 수, 선택 필드 수, 모델 호출 수와 저장 재실행
확인을 보여줍니다. 각 작업의 `result.json`·`replay.json`과 조립 폴더도 보관합니다.
여기에는 승인 가능·미제출·비활성 검토자 사례로 구성한 인보이스 승인 도메인
예제도 포함됩니다. 이는 세 경로를 실행하는 학습용 예제이며 실제 결제 정책은
정의하지 않습니다.
CI 계획 프로그램은 `go`, `docs`, `yaml` 확인 항목에 연결된 제한된 변경 파일 묶음을
분류합니다. 12개 고정 입력에서 알맞은 계획·미등록 입력·불완전 입력을 확인하며,
검사 명령 자체는 실행하지 않습니다.
모델 사용은 진단과 시작 프로그램에서 각각 한 번입니다. 표준 함수는 Gooo에
작성한 본문을 그대로 생성하며 모델 호출이 없습니다.

## 모델 입력의 구분 능력 확인

```sh
go run ./cmd/workbench feature-audit --compiler ./.gooo \
  --input examples/feature-audit/filename-order.json --model builtin --out out/feature-audit
```

후보의 앞뒤만 바꾼 여덟 입력을 실제 모델 특징으로 변환해 같은 배열끼리 묶습니다.
포함한 자료는 한 프로그램에서 나왔으며 두 종류의 배열만 남습니다. 입력이 같지만
필요한 선택이 다른 사례를 세고, Gooo가 다음 작업을 기록합니다. `--model`을 생략하면
가중치를 읽지 않고 입력의 구분 능력만 확인합니다. 자세한 분모와 입력 형식은
[예제 안내](../examples/feature-audit/README.md)에 있습니다.

## 프로젝트 만들기

```sh
go run ./cmd/workbench scaffold --compiler ./.gooo --profile scalar --out out/scalar
go run ./cmd/workbench scaffold --compiler ./.gooo --profile record --model builtin --out out/record
go run ./cmd/workbench scaffold --compiler ./.gooo --profile library --model builtin --out out/library
```

scalar는 Integer 입력을, record는 제목 필드를 가진 Item 입력을 그대로 반환하는
`Identity` 활동을 만듭니다. `main.gooo`의 선언과 계산 본문을 수정해 확장합니다.
파일은 Gooo 프로그램의 실제 출력에서 얻고, 새 파일을 검사해 Go 본문으로 생성합니다.
library 프로필은 `Clamp(Integer) -> Integer` 공개 계약으로 시작하며, 선택된
activity의 타입과 생성 결과는 `activity-generation.json`에서 확인할 수 있습니다.

## 결과를 다음 작업으로 넘기기

```sh
go run ./cmd/workbench diagnose --compiler ./.gooo \
  --input examples/partial-composition.json --model builtin --out out/repair
```

이 예제의 `diagnostic.json`에는 부분 충족과 `repair-and-replay`가 남습니다.
`observation.json`은 실제 값에서 다시 센 활동 출력 6/14, 타입 탈락 1개와 상세 내용을 담습니다.
이 입력에는 레코드와 문자열 출력이 함께 있으므로 `unit`은 `activity_outputs`입니다.
레코드만 있는 입력은 필드별로 집계하고 `record_fields`로 표시합니다.
`--input`에는 `gooo package execute`, `resume`, `replay`의 JSON 영수증도 넣을 수 있습니다.
외부 프로그램이 이 두 파일을 읽어 다음 작업을 만들 수 있습니다. 현재 도구는
작업을 분류하고 구성합니다. 조립 사례·시도 한도·선언한 탐색 방법을 바꾸는 후속 실행은
아래 `refine` 명령에 연결돼 있습니다.

`captured-input.json`은 원래 입력 전체를 보관합니다. 전달할 상세 문장은
현재 Text 입력의 1,024바이트 범위에 맞춥니다. 상세가 더 길면 집계 단위·불일치 수와
타입 탈락 수·원본 SHA256을 전달하고 `detail_limited`를 표시합니다.
같은 결과의 필드 차이는 필드명 순서로 기록합니다.

조립 기록이 있는 입력은 `construction-next-steps.json`도 만듭니다.
각 활동의 조립 사례, 시도한 후보 수, 전체 후보 수, 소스에 선언한 예산을 별도로
전달하고 Gooo가 다음 행동을 고릅니다. 예를 들어 `resume-candidates`는 후보와
예산이 남았음을 뜻합니다. 이 제안을 사용해 `gooo package resume`를 호출할 때는
원래 워크스페이스와 실행 기록, 새 이어가기 정책을 명시합니다.

`diagnose` 자체는 소스를 수정하거나 이어가기를 실행하지 않습니다. 기본 진단의
`action`은 실행 점수·타입 탈락을 요약하며, `construction_next_steps`는 활동별
조립 상황을 추가로 설명합니다. 예산은 `attempt_budget`에서 읽고, 이전 기록은 정책
입력에서 읽습니다. `budget_source`로 어느 기록을 사용했는지 알 수 있습니다.
두 기록이 다르면 불일치로 진단합니다. 예산이 없으면 후보 수로 대신 추정하지 않고
`inspect-attempt-budget`으로 남깁니다.

## Gooo의 제안을 소스 수정으로 이어가기

`refine`는 처음 조립한 프로그램을 Gooo로 진단하고, 한도 확대나 사례 추가가
제안되면 지정된 Gooo 정책으로 소스를 수정해 다시 실행합니다.

```sh
go run ./cmd/workbench refine --compiler ./.gooo \
  --source examples/source-refinement/source.gooo --activity Select \
  --cases examples/source-refinement/feedback-cases.json \
  --policy examples/source-refinement/policy.gooo \
  --evaluation-cases examples/source-refinement/evaluation-cases.json \
  --max-attempts 8 --max-rounds 4 --out out/refinement
```

자체 소형 모델을 사용하려면 `--model builtin`을 더합니다. 다음 행동과 수정 규칙은
Gooo 프로그램으로 실행됩니다. 한도가 남아 있는지와 원래 선언한 후보를 모두
확인했는지를 구분하며, 수정 정책의 한도 안에서 해결되지 않은 결과도 남깁니다.

출력의 `refinement-dispatch.json`에는 수정 전후 관측, 다음 행동, 모델 호출 수가
있습니다. `selected_source`는 유지한 소스를 가리킵니다. 원본 파일은 보존하고,
피드백과 별도로 제공한 최종 평가는 소스 선택 후에 실행합니다.
[정책과 기록 설명](../examples/source-refinement/README.md)을 함께 볼 수 있습니다.

`--search-policy`를 주면 Gooo 소스의 `search_alternative`도 선택할 수 있습니다.
[보정값 보고서](../examples/calibrated-report/README.md)에서는 숫자식과 레코드 조립을
연결하고, 모델 유무와 중단 조건을 비교합니다. 후보를 얼마나 담았는지와 실행 기대값을
몇 개 맞혔는지는 각각의 기록으로 읽습니다. 대안 ID는 `shared_fit`처럼 식별자로 적습니다.

명령이 JSON으로 오류를 반환하면 실행 도구가 그 `error` 또는 `failure` 내용을
실패 메시지에 포함합니다. 소스의 어느 선언이 잘못됐는지 원인을 함께 확인할 수 있습니다.

## 잘못된 계산식을 지나가며 전체 프로그램 조립하기

Gooo 0.6.13의 `body-construct`는 레코드 선택과 정수 계산식을 함께 고릅니다.
아래 예제는 지역 계산식의 타입 검사 실패를 기록하고 다음 조합을 시도합니다.
위에서 준비한 `.compiler` 소스와 실행 파일을 사용합니다.

```sh
go run ./cmd/workbench construct --compiler ./.gooo \
  --source .compiler/examples/caller-search-rejection/mixed-model.gooo.fixture --entry Main \
  --construction-cases examples/joint-feedback/initial-cases.json \
  --evaluation-cases .compiler/examples/caller-search-rejection/mixed-construction-cases.json \
  --holdout-cases .compiler/examples/caller-search-rejection/mixed-evaluation-cases.json \
  --model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json \
  --max-program-budget 40 --max-rounds 8 --out out/caller-search
```

처음 조건은 `0 → 0` 하나입니다. Gooo 피드백 규칙이 실패 사례의 원래 기대값을
다음 회차에 추가합니다. 모델은 레코드 순서를 제안하고 계산식은 정해진 순서로
살핍니다. `--model`을 빼면 전체 후보 순서가 결정론적입니다.

`joint-loop.json`에서 회차별 `program_attempts`와 마지막
`final_evaluation.joint_construction`의 `rejected_attempts`,
`native_program_attempts`를 구분해 읽습니다. 실행 전에 거절한 조합도 예산에
포함되며 호출부 점수는 없습니다. `final_evaluation.passed/total`은 선택이 끝난
뒤 주어진 평가 사례를 얼마나 맞혔는지 나타냅니다.

기존 모델 관측은 7회차·49번의 조합 시도 후 마지막 평가 4/4입니다. 마지막 회차의
17번 중 8번은 지역 검사에서 거절됐고 9개 프로그램을 실행했습니다.
가중치와 원래 기대값을 유지한 한 프로그램의 기록입니다.
[관측 범위와 원본](../publication/joint-rejection-20261009/README.md).

## 공개 API 참조 만들기

컴파일러가 공개 시그니처와 타입 ID를 읽고 Gooo 템플릿으로 참조 문서를 만듭니다.

```sh
go run ./cmd/workbench reference --compiler ./.gooo \
  --package examples/catalog --entry ApproveInvoice --out out/api-reference
```

`reference.md`와 함께 컴파일러 인터페이스 원본, Gooo 실행, 기대값 확인과
저장 재실행을 보관합니다. 선언된 타입 계약을 보여주며 구현 동작을 설명하지 않습니다.

## 자연어로 Gooo 기능 찾기

```sh
go run ./cmd/workbench discover --compiler ./.gooo \
  --query '코드 생성은 어떻게 해?' \
  --declaration examples/catalog/operations.gooo \
  --out out/capability-discovery
```

Gooo-jev가 한·영 질의를 기능 카탈로그에 연결하고, Gooo 평가 프로그램이 선언
바인딩과 첫 미해결 단계를 만듭니다. `AVAILABLE`은 카탈로그 결과이지 실행 증거가
아닙니다. 선언 연결만 확인된 경우 사용 사례 범위는 `PROGRESS`, 다음 단계는
`generation`입니다. 출력에는 원 질의·선언·탐색 digest·Gooo 조립과 재실행을 둡니다.
현재 JEV 탐색은 결정론적이며 provider를 호출하지 않습니다.

## 완전성 영수증 만들기

성공한 검증 폴더를 입력해 근거가 연결된 영수증을 만듭니다.

```sh
go run ./cmd/workbench receipt --compiler ./.gooo --input out/verified --out out/completeness
```

데이터 구조와 최초 미해결 단계 계산은
[`recipes/completeness.gooo`](../recipes/completeness.gooo)에 정의됩니다.
작성된 유한 사례만 확인한 축은 `PROGRESS`, 계측하지 않은 외부 효과 경계는
`UNKNOWN`으로 남기며 단일 완성도 점수로 합치지 않습니다. 측정은 결정론적이고
모델 호출을 하지 않습니다. `verify` 시점의 저장소 HEAD와 현재 입력 위치가
일치하지 않으면 출처도 `UNKNOWN`으로 남깁니다.

## 소스 작성에서 확인한 규칙

- 함께 실행할 프로그램은 Gooo의 명시적 `bind` 연결을 사용합니다.
- 레코드를 반환하는 조립 사례는 단일 입력도 `["record"]` 같은 위치 배열로 적습니다.
- Gooo 본문 안에 새 Gooo 코드를 문자열로 넣을 때는 본문 구분자를 `\x60`으로 표현합니다.
- 현재 공유 모델 입력은 후보 한 항목당512바이트입니다. 긴 템플릿은 지역 변수에
  보관하고, 후보 참조와 의도를 전달합니다. 원래 템플릿의 실제 내용은 Gooo가 소유합니다.
- 모든 기대값이 없는 실행 요청에서는 전달한 문장을 되돌리는 관측을 함께 둡니다.
  진단 결과의 충족률과 그 전달 관측의 충족률은 각각의 범위로 읽습니다.

단위 검사는 `go test ./...`, 실제 컴파일러까지 연결한 검사는 다음과 같습니다.

```sh
GOOO_COMPILER="$PWD/.gooo" go test -race ./...
```

CI는 같은 고정 소스의 컴파일러로 실제 생성·실행·프로젝트 만들기·진단을 확인합니다.
