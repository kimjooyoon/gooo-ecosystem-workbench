# 실행 안내

## Gooo 컴파일러 준비

Go1.27.1을 사용합니다. 이 저장소에서 변경 없는 고정 컴파일러를 빌드합니다.

```sh
git clone https://github.com/kimjooyoon/meta-ontology-go.git .compiler
git -C .compiler switch --detach f144dddb8261b9b525181dee510afc65f1603153
cd .compiler
go build -trimpath -o ../.gooo ./cmd/gooo
cd ..
go run ./cmd/workbench verify --compiler ./.gooo --model builtin --out out/verified
```

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
`observation.json`은 실제 값에서 다시 센17/21필드, 타입 탈락1개와 상세 내용을 담습니다.
외부 프로그램이 이 두 파일을 읽어 다음 작업을 만들 수 있습니다. 현재 도구는
작업을 분류하고 구성합니다. 본문을 자동으로 수정하는 후속 실행기는 별도 구현 과제입니다.

`captured-input.json`은 원래 입력 전체를 보관합니다. 전달할 상세 문장은
현재 Text 입력의1,024바이트 범위에 맞춥니다. 상세가 더 길면 남은 필드 수와
타입 탈락 수·원본 SHA256을 전달하고 `detail_limited`를 표시합니다.
같은 결과의 필드 차이는 필드명 순서로 기록합니다.

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
