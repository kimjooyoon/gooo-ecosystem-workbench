# Gooo로 소스의 한 조각 바꾸기

`splice`는 작은 텍스트 수정 도구입니다. 원문, 바꿀 조각, 그 앞뒤 내용을
받습니다. Gooo가 원문이 맞는지 확인한 뒤 내용을 바꾸고 바뀐 길이를 기록합니다.
각 단계는 앞 단계의 값을 읽습니다. 모델은 이 세 단계의 후보식 조합을 추천하고,
컴파일러는 소스에 있는 여덟 예제를 실행해 조립 결과를 고릅니다.

현재는 UTF-8 기준 최대 1,024바이트의 텍스트를 다룹니다. 파일 전체를 읽고 쓰는
역할은 Go 연결 코드가 맡습니다. 원본 파일에 바로 덮어쓰지 않고 새 출력 폴더에
결과를 저장합니다. 원문이 달라졌거나 교체 내용이 같으면 그대로 반환하며,
결과가 길이 한도를 넘는 교체도 적용하지 않습니다.

## 실행

Go 1.27.1과 [Gooo 0.6.10 개발판](https://github.com/kimjooyoon/meta-ontology-go/releases/tag/v0.6.10-dev)을
사용합니다. 배포 소스는 `eb0dc4704705147f9ba944db3df2ba5e3225cbff`입니다.
패키지의 `len` 처리 수정은 [컴파일러 PR 1378](https://github.com/kimjooyoon/meta-ontology-go/pull/1378)에 있으며,
이번 배포 파일에 포함됐습니다. 아래의 동결 실험 원본은 최초 관측 당시의 소스를 유지합니다.

저장소 루트에서, 준비한 컴파일러 경로를 지정합니다.

```sh
go run ./cmd/workbench splice --compiler /path/to/gooo \
  --input examples/source-splice/request.json --out out/splice-fixed
go run ./cmd/workbench splice --compiler /path/to/gooo \
  --input examples/source-splice/request.json \
  --model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json \
  --out out/splice-model
```

모델 옵션이 없으면 고정 순서로 조립합니다. 그래프 모델은 파일명·나눗셈·재시도
프로그램으로 학습한 기존 가중치를 그대로 사용합니다. 새 수정 도구에 대한
학습은 하지 않았습니다.

요청 예제는 짧은 Gooo 함수의 `input + 1`을 `input * 2`로 바꿉니다.
`edited.txt`가 수정된 소스이고 `splice.json`에는 변경 여부, 길이, 조립 검사와
저장 재실행 결과가 남습니다. 원본 요청·Gooo 선언·패키지 실행 기록도 함께 남습니다.
실제 요청에는 기대 정답을 주지 않으므로 실행 관측과 조립 예제의 충족률을 구분합니다.

## 새 프로그램에 대한 실험

```sh
go run ./experiments/dependent-splice --compiler /path/to/gooo --out out/splice-study
```

미리 고정한 [실험 계획](../../experiments/dependent-splice/PLAN.md)에 따라
고정 순서와 모델을 1·2·4·8회 한도로 비교합니다. 소스의 조립 예제와 겹치지 않는
542개 입력을 별도 Go 기준 함수와 비교하고, 같은 입력을 저장한 코드로 다시 실행합니다.
새 프로그램 하나를 여러 조건에서 관측한 결과입니다. 요청 문장·후보 위치를
바꾼 별도 일반화 연구는 포함하지 않습니다.

마지막에는 실제로 반환된 Gooo 소스를 컴파일해 음수·0·양수 입력에서 두 배 값이
나오는지 확인합니다. 후보의 일부 필드만 맞는 경우에도 원래 결과와 점수를 남깁니다.
여덟 조립 예제가 전부 맞은 뒤에만 `splice` 명령이 완성된 도구로 결과를 반환합니다.

선택과 수정 동작은 [Gooo 소스](../../recipes/splice.gooo)에 있으며, Go의 독립 기준
함수는 실험에서만 사용합니다. 새로운 요청마다 모델을 부를 필요 없이 저장된
`execution.json`을 `gooo package replay --inputs ... --receipt ...`에 전달할 수 있습니다.
