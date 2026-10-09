# 분기·변수 선택을 호출 반례로 다시 조립하기

부품 자체의 작은 사례를 통과한 분기라도, 전체 호출에서 기대한 값과 다를 수 있습니다.
작업장은 원래 실패 입력을 Gooo의 후속 조립 규칙에 넘깁니다. 컴파일러가 소스에 적힌
선택지를 한도 안에서 다시 조합하고, 선택한 코드를 저장해 별도 입력으로 재생합니다.

## 실행 파일 준비

네이티브 typed 경로의 전체 조립은 컴파일러 dev `26c4e315`부터 지원합니다.
공개 0.6.23 릴리스 소스 `2b17c487`에는 이 기능이 없습니다.
컴파일러의 현재 checkout에서 Go 1.27.2로 개발 실행 파일을 별도 이름으로 만듭니다.

```sh
GOWORK=off GOTOOLCHAIN=go1.27.2 go build -o /tmp/gooo-typed-dev ./cmd/gooo
/tmp/gooo-typed-dev version --build --json
```

이 문서의 `typed.gooo`는 `input - 1`의 피연산자 순서를 고르는 작은 예제입니다.
일반 `-input`을 typed 조립에 쓰는 개선은
[컴파일러 PR1432](https://github.com/kimjooyoon/meta-ontology-go/pull/1432)에 있습니다.
버전 문자열과 함께 실제 소스 리비전을 확인합니다.

작업장 루트에서:

```sh
go run ./cmd/workbench assemble --graph --compiler /tmp/gooo-typed-dev \
  --source examples/full-graph-assembly/typed.gooo --entry Main \
  --cases examples/full-graph-assembly/typed-cases.json --out out/typed-origin
go run ./cmd/workbench construct --compiler /tmp/gooo-typed-dev \
  --assembly out/typed-origin --max-program-budget 4 --max-rounds 4 --out out/typed-next
```

`--assembly`는 원래 소스·입력·기대값을 읽습니다. 새 `--evaluation-cases`를 함께 지정하지 않습니다.
별도 최종 입력은 `--holdout-cases`로 추가합니다. 모델을 생략한 조립은 결정론적으로 진행됩니다.
처음 사용한 모델도 다음 명령에 자동으로 상속되지 않습니다.

## 구분해서 읽을 결과

- 본문 자체 사례: 분기·변수 후보가 소스에 적힌 값과 맞았는지.
- 호출 사례: 전체 연결을 실행해 맞았는지. 재조립의 선택에 소비됩니다.
- 별도 입력: 선택이 끝난 코드를 저장 재생해 맞았는지.
- 타입 오류 후보: 이유와 시도 횟수만 남고 호출 점수는 생기지 않습니다.
- 실행 중 계산 실패: 네이티브 실패와 차단된 후속 활동을 남깁니다.

작업장이 읽는 초기 typed 영수증에는 원래 소스의 시도 상한이 없습니다.
그 값은 `budget_known: false`로 남기고 전체 후보 개수로 추정하지 않습니다.
각 프로그램 시도의 후보 집합에는 컴파일러가 내보낸 실제 한도·mask·선택 ID가 남습니다.
정확한 정수는 `9007199254740993`까지 반올림 없이 읽습니다.

`rejection.gooo`는 지역 변수의 참조·선언 순서를 함께 바꾸다가 타입 검사에서 거절되는
예제입니다. `fault.gooo`는 전체 호출의 0 나눗셈을 기록하는 예제입니다.
[원본 v7 관측](../joint-diagnostics/TYPED-PATH-OBSERVATIONS.md)은 작은 사례의 검사 범위를 공개합니다.
