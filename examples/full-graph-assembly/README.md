# 여러 본문을 함께 고르기

두 본문을 연결해 문장을 만들고, 별도 숫자를 더하는 예제입니다.
먼저 소스에 적힌 사례로 조립합니다. 그 뒤 실제 호출에서 틀린 행을 골라
같은 그래프를 다시 조립합니다. 빈칸을 채우는 보조 함수와 정수식 탐색을
함께 쓰는 `mixed.gooo`도 있습니다.

## 실행

[공개 Gooo 0.6.23](https://github.com/kimjooyoon/meta-ontology-go/releases/tag/v0.6.23-dev)을
[설치](../../docs/usage.ko.md)한 뒤 PATH의 `gooo`를 사용합니다.

저장소 루트에서:

```sh
go run ./cmd/workbench assemble --graph \
  --source examples/full-graph-assembly/chain.gooo --entry Main \
  --cases examples/full-graph-assembly/chain-cases.json --out out/chain
go run ./cmd/workbench construct --assembly out/chain \
  --holdout-cases examples/full-graph-assembly/chain-holdout.json \
  --max-program-budget 64 --max-rounds 8 --out out/chain-next
```

자체 소형 모델로 최초 조립 순서를 고르려면 첫 명령에 추가합니다:

```sh
--model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json
```

다음 `construct`에서 모델을 생략하면 새 조립은 결정론적으로 시작합니다.
최초 조립에서 사용한 모델을 자동으로 이어받지 않습니다.
`called-fill`과 `mixed`도 파일 이름만 바꿔 같은 명령으로 실행합니다.
이번 어댑터의 source-fill 경로는 소스 후보를 결정론적으로 고릅니다.

## 결과 읽기

| 파일 | 읽을 내용 |
| --- | --- |
| `plan.json` | 실제 컴파일러의 입력·연결·본문·보조 함수 목록. 조회 추론·사례 검사·실행은 모두 0 |
| `report.json` | 호출 기대값의 개수, 본문별 자체 사례, 실제 모델 호출, Gooo의 다음 작업 |
| `composition/generated.go` | 실제로 실행한 생성 코드 |
| `next-context.json` | 원본 파일의 크기·해시와 최대 8개 불일치 위치 |
| `origin-feedback-cases.json` | Gooo가 고른 원래 호출 행. 추가 루트, 여러 출력, 큰 정수도 그대로 유지 |
| `construction/joint-loop.json` | 반례 소비 여부, 라운드·후보 예산, 반복 시도, 마지막 별도 입력 평가 |

예제의 `9007199254740993`은 소수점으로 변환하지 않고 유지합니다.
기대값 없이 실행하려면 `--cases`에 `gooo/body-composition-inputs/v1` 문서를 넘깁니다.
이 경우 호출 점수는 0/0이고 Gooo가 기대값을 추가하도록 안내합니다.

본문 자체 사례는 원래 `.gooo`에 남습니다. 이를 거꾸로 계산해서 전체 그래프의
호출 입력이나 기대값으로 바꾸지 않습니다. `caller-history.json`은 첫 피드백 전의
빈 호출 이력입니다. 실제 네이티브 조립에는 Gooo가 선택한 1개 이상의 호출 행을 넘깁니다.

저장 재생은 이미 고른 코드와 본문별 소스 체크포인트를 컴파일러가 확인한 뒤 실행합니다.
각 본문에서 모델에 들어간 내용은 그 시점의 소스에 묶입니다.
`replay_new_model_calls`는 0이며 저장 기록에 남은 최초 추론 횟수와 구분합니다.
선택에 사용한 호출 사례와 최종 별도 사례는 따로 셉니다.
별도 파일이라는 사실만으로 학습 데이터와의 독립성을 주장하지 않습니다.

`typed.gooo`는 실행·재생까지 지원하는 정수식 순서 선택 예제입니다.
작업장은 이 본문의 호출 반례도 전체 조립으로 넘깁니다.
공개 0.6.23의 컴파일러 소스 `2b17c487`은 이 재조립을 지원하지 않아 실제 컴파일러의
지원 범위 오류를 남깁니다. dev `26c4e315`부터 네이티브 typed 경로 재조립을 지원합니다.
[개발 소스 준비와 예제](../caller-typed-paths/README.md)에서 사용할 실행 파일을 확인합니다.
