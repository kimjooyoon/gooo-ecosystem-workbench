# 내 소스에 모델을 쓸 수 있을까?

`assemble`은 소스와 모델을 먼저 확인하고, Gooo 규칙으로 조립 경로를 고릅니다.
조립한 Go 프로그램을 실제로 실행하고 저장한 프로그램도 다시 실행합니다.
터미널에는 선택 이유와 사례 결과가 나오고, 원본 기록은 출력 폴더에 남습니다.

이 예제는 숫자·사용 여부·문장을 받아 세 필드를 돌려줍니다. 사용 가능한 양수라면
문장에 `!`를 붙이고, 입력 정수는 그대로 보존합니다. 본문에는 조건식, 지역 변수,
대입식이 있고 마지막 레코드의 필드 세 개에 대안이 있습니다. 조립용 사례는 소스에,
추가 호출 사례 네 개는 `cases.json`에 선언했습니다.

## 실행

[컴파일러 준비](../../docs/usage.ko.md)의 고정 소스를 빌드한 다음 저장소 루트에서:

```sh
go run ./cmd/workbench assemble --compiler ./.gooo \
  --source examples/model-assembly/source.gooo --entry Describe \
  --cases examples/model-assembly/cases.json \
  --model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json \
  --out out/my-assembly
```

`--model`을 생략하면 소스에 선언한 순서로 조립합니다. `english.gooo`는 같은 안정 ID와
동작을 영어 타입 이름으로 표현합니다. `one-choice.gooo`는 선택 하나만 남겼습니다.
이 모델은 세 선택을 요구하므로 사전 확인에서 이유를 기록하고 고정 순서로 진행합니다.
없는 파일이나 손상된 모델은 원래 오류를 남기고 종료합니다.

## 결과 읽기

| 파일 | 내용 |
| --- | --- |
| `report.json` | 선택 이유, 소스·모델 지문, 조립 사례와 호출 사례의 결과, 원본 위치 |
| `preflight.json` | 모델이 읽을 원본 입력과 표현 가능 여부; 추론·후보 검사는 각각 0회 |
| `routing/execution.json`, `routing/replay.json` | [Gooo 경로 규칙](../../recipes/assembly-route.gooo)의 입력 실행과 저장 재생 |
| `assembly.json`, `replay.json` | 실제 조립·호출과 저장 프로그램의 호출 결과 |
| `composition/generated.go` | 선택한 Go 코드 |
| `composition/composition.json` | 다시 실행할 선택 기록 |
| `follow-up/execution.json`, `follow-up/replay.json` | [Gooo 후속 규칙](../../recipes/assembly-next.gooo)이 읽은 실제 개수와 제안의 저장 재생 |
| `next-context.json` | 다음 도구에 전달할 짧은 관측, 제안, 최대 8개 불일치 위치와 원본 지문 |

자동화에서는 `--json`을 붙여 같은 짧은 보고서를 표준 출력으로 받을 수 있습니다.
긴 입력 그래프는 `preflight.json`에 보관합니다. 보고서에는 지문과 원본 위치를 남깁니다.
Gooo 경로 규칙은 새 추론을 호출하지 않으며 실제 입력의 정답 점수는 0/0입니다.
대상 본문을 조립할 때의 모델 호출과 별도로 셉니다.

호출 기대값이 일부만 맞아도 결과를 보존합니다. 후속 규칙은 호출 결과가 다르면
원래 입력·기대값을 확인하고 반례를 조립 조건에 추가하도록 제안합니다. 조립용 사례의
필드가 다르면 소스의 후보를 확인하고, 모든 호출 기대값을 만족하면 다른 입력을
관측하도록 제안합니다. 비교할 기대값이 없으면 관측을 추가하라는 결과가 나옵니다.
터미널에 다음 작업과 `next-context.json` 경로가 표시됩니다.

`next-context.json`은 입력 그래프나 전체 실행 흔적을 복사하지 않습니다. 실패 위치는
`assembly.json` 안의 JSON 경로이며, 원본 파일의 경로·지문·크기를 함께 남깁니다.
같은 기대값을 다시 실행한 기록은 새 평가 사례로 세지 않습니다. 후속 제안은 정해진
개수를 읽는 결정론적 Gooo 프로그램이며, 제안된 작업의 실행과 다음 모델 호출은
이 명령에 포함되지 않습니다. 원본 기대값은 그대로 유지됩니다.

[후속 규칙의 실제 기록](../../publication/assembly-next-20261009/README.md)에는
4/4·3/4 결과, 작은 context와 압축한 원본 실행 폴더가 있습니다.

소스에 적은 조립용 사례는 후보 선택에 쓰입니다. `cases.json`의 기대값은 생성된 코드의
실제 호출과 비교합니다. 이 예제의 4/4와 필드 12/12는 공개한 네 입력에 대한 결과입니다.
모델의 학습에 노출됐는지 확인한 독립 평가 자료로 분류하지 않았습니다.

[실제 실행 기록](../../publication/source-model-assembly-20261009/README.md)에는 네 경로의
원본 출력, 입력·모델 지문과 전체 명령의 시간·메모리 관측이 있습니다.

현재 범위는 한 활동의 소스 소유 레코드 필드 조립입니다. 패키지 전체의 반복 조립에는
[`construct --workspace`](../package-caller-construction/README.md)를 사용합니다.
출력 폴더는 매번 새 경로여야 합니다. 이 명령의 사전 확인에는 `body-context --model`이
있는 개발 컴파일러가 필요합니다. 공개 0.6.21 실행 파일은 이전 소스이며 이 옵션이 없습니다.
