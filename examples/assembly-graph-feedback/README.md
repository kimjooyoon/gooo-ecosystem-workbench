# 연결한 활동의 실패로 다음 조립 진행하기

이 예제는 입력으로 문구·사용 여부·정수를 조립하고, 다음 활동에서 조건에 맞으면
문구를 괄호로 감쌉니다. 마지막 결과가 틀리면 Gooo가 원래 입력 행을 골라 다음
코드 생성에 반영합니다. 두 명령으로 저장된 결과를 이어 쓸 수 있습니다.

공개 Gooo 0.6.22와 Go 1.27.2를 사용합니다. 저장소 루트에서:

```sh
go run ./cmd/workbench assemble --source examples/assembly-graph-feedback/source.gooo \
  --entry Present --assembly-activity Describe \
  --cases examples/assembly-graph-feedback/adaptive-cases.json --out out/graph-first
go run ./cmd/workbench construct --assembly out/graph-first \
  --holdout-cases examples/assembly-graph-feedback/holdout-cases.json \
  --max-program-budget 8 --max-rounds 5 --out out/graph-next
```

`Describe`가 세 필드의 식을 조립합니다. `bind`가 그 레코드를 `Present`로
전달합니다. `Present`는 조건이 참이면 고정 보조 함수 `Decorate`를 호출합니다.
`--entry`는 실행할 마지막 활동이고, `--assembly-activity`는 모델 입력을 확인할
루트 레코드 활동입니다. 하나의 활동이면 기존처럼 `--entry`만 지정합니다.

인자가 하나인 활동은 입력 키가 활동 이름(`Describe`)이고, 여러 인자는
`Describe.input0`처럼 포트 이름을 붙입니다. 저장된 native 계획의 두 형식을
그대로 읽습니다. [단일 인자 소스](single-root.gooo)와
[호출 사례](single-adaptive-cases.json)도 같은 연결을 사용합니다.

소스의 첫 사례는 모든 후보가 같은 값을 내는 입력입니다. 다음 입력에서 드러나는
실패를 마지막 결과의 기대값으로 검사합니다. `source-cases.json`은 Describe의
소스 사례를 그대로 옮기고, 실패한 호출 행은 Present의 기대값을 그대로 추가합니다.
중간 레코드로부터 원래 입력을 역산하거나, 실제 출력을 기대값으로 바꾸지 않습니다.

자체 소형 모델을 쓰려면 assemble 또는 construct에 다음을 각각 지정합니다:

```sh
--model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json
```

construct에서 모델을 생략하면 이전 조립의 모델을 이어받지 않고 결정론적으로
진행합니다. 명시하면 각 새 회차의 조립에 모델이 호출됩니다. 저장 재생과 마지막
평가의 새 추론 수는 0입니다. [관측 전 계획](PLAN.md)에 확인 범위를 적었습니다.
[네 경로의 실제 관측](../../publication/assembly-graph-feedback-20261009/README.md)은
고정 시작 2/6, 모델 시작 4/6과 후속·최종 필드 6/6을 각각 기록합니다.

`assembly-construction.json`에서 원래 관측, 반례 준비·실제 사용 여부, 각 회차의
시도 수와 `final_evaluation`을 읽습니다. 소스 사례와 마지막 활동의 사례는
따로 셉니다. 마지막 두 입력은 선택이 끝난 뒤 실행하며 다음 선택에 사용하지 않습니다.
학습 노출 여부는 확인되지 않았습니다.

연결 범위는 루트 레코드 조립 하나와 고정된 후속 활동·순수 보조 함수입니다.
추가 루트 입력, 여러 조립 활동, 조립이 필요한 보조 함수에는 별도 원래 입력과
기대값 연결이 필요합니다. 현재 저장 연결은 이런 그래프를 오류와 함께 남깁니다.
기존 `construct --source`와 `construct --workspace`는 더 넓은 조립 그래프를 받습니다.
