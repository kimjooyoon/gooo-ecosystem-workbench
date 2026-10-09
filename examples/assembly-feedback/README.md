# 저장한 조립 결과로 다음 조립 진행하기

조립은 됐지만 새로운 호출에서 일부 필드가 틀릴 수 있습니다. 저장한 `assemble`
폴더를 다음 명령에 넘기면 Gooo가 실패한 입력을 고르고 기존 조립 반복을 진행합니다.
소스와 원래 기대값은 그대로 둡니다.

```sh
gooo-workbench assemble --source examples/assembly-feedback/source.gooo \
  --entry Describe --cases examples/assembly-feedback/adaptive-cases.json --out out/first
gooo-workbench construct --assembly out/first \
  --holdout-cases examples/assembly-feedback/holdout-cases.json \
  --max-program-budget 8 --max-rounds 4 --out out/next
```

이 예제의 첫 소스 사례는 모든 후보가 같은 답을 내는 입력입니다.
그래서 처음에는 조립용 사례 3/3이어도 실제 호출은 필드 2/6입니다.
후속 실행은 실패한 원본 행 2개를 추가하고, 최대 8개 후보·4회차 안에서 진행합니다.
[관측 전 계획](PLAN.md)에 입력과 평가 기준을 먼저 정했습니다.
[세 경로의 실제 관측과 원본](../../publication/assembly-feedback-20261009/README.md)을 공개합니다.

`out/next/assembly-construction.json`에서 읽을 항목:

터미널에는 원래 관측, 이번 조립 방식, 회차와 반복 시도, 평가 단계가 짧게 나옵니다.
표준 출력의 JSON을 읽던 도구는 `construct --json`을 사용합니다.
저장한 파일의 형식은 이어서 읽을 수 있고, 새 회차에는 Gooo가 낸 다음 작업의 문장도 남습니다.

| 항목 | 뜻 |
| --- | --- |
| origin_observation | 이전 호출·필드와 소스 사례의 관측 |
| origin_replay_verified | 원래 프로그램을 같은 입력으로 다시 실행한 결과 |
| feedback.prepared | 반례 파일을 준비했는지 |
| feedback.consumed_by_next_round | 다음 조립에서 실제 읽었는지 |
| construction_loop.rounds | 각 회차의 시도 한도·실제 시도·충족한 필드 |
| construction_loop.final_evaluation | 선택 뒤 다른 입력으로 재생한 결과 |

`origin/`은 이전 폴더의 원본을 보관합니다. `source-cases.json`은 소스 안의
입력 배열과 기대값을 호출 형식으로 옮긴 것입니다. `origin-feedback-cases.json`에
Gooo가 고른 실패 행을 추가합니다. 큰 정수도 원래 JSON 값으로 보존합니다.

자체 모델로 후속 조립을 진행하려면 위 `construct`에 다음 옵션을 추가합니다.

```sh
--model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json
```

이전 조립이 모델을 사용했어도 후속 명령의 옵션을 생략하면 결정론적으로 진행합니다.
각 회차가 새로 조립하므로 이전 순서 재개와 구분하고, 중복 시도와 새 모델 호출도 셉니다.
저장 재생과 마지막 평가에서는 새 추론이 없습니다.

이미 호출 기대값을 충족한 결과는 새로운 조립을 시작하지 않습니다.
별도 최종 입력을 주면 원래 선택된 프로그램으로만 실행하고 `origin_holdout`에 남깁니다.
그 안의 generation_model_calls는 원래 생성 기록이며 new_model_calls가 새 호출 수입니다.
실패하면 원본과 부분 진행 기록을 새 출력 폴더에서 확인할 수 있습니다.

루트 레코드 활동 뒤에 고정 활동을 연결하려면
[두 활동과 보조 함수 예제](../assembly-graph-feedback/README.md)를 사용합니다.
작업장의 assemble는 호출 기대값을 요구합니다. 입력만 주어 실행하는 경로는
이 연결의 실행 예제에 포함하지 않습니다. 누락된 기대값을 추측해 채우지 않습니다.
여러 패키지의 호출 피드백은 `construct --workspace`를 사용합니다.
모순된 기대값도 그대로 남겨 한도 안에서 멈춥니다. 최종 입력의 분리를 기록하며,
모델 학습 자료와의 독립성은 확인되지 않았습니다.
