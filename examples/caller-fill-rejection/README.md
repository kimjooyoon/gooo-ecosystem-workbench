# 실패한 후보를 기록하고 다음 조립으로 넘어가기

Gooo 소스에는 예산을 계산하는 함수의 조건식과 상한값 후보가 들어 있습니다.
후보 다섯 개 중 하나는 숫자 자리에 Boolean을 넣고, 하나는 0으로 나눕니다.
나머지 세 후보와 기존 입력·기대값은 그대로 유지했습니다.

컴파일러는 실패한 두 후보의 이유를 보관하고 나머지를 조립합니다. 이 도구는
그 기록을 읽어 **시도 수, 탈락 수, 실제 실행 수**를 구분합니다. 실패한 후보가
정답률 분모에 섞이지 않고, 다시 시도할 후보로 잘못 안내되지 않도록 Gooo로
작성한 `next-steps` 판단에도 탈락 수를 전달합니다.

```sh
go run ./cmd/workbench construct --compiler /path/to/new/gooo \
  --source examples/caller-fill-rejection/source.gooo --entry Main \
  --construction-cases examples/caller-fill-rejection/initial-cases.json \
  --evaluation-cases examples/caller-fill-rejection/evaluation-cases.json \
  --holdout-cases examples/caller-fill-rejection/holdout-cases.json \
  --max-program-budget 8 --max-rounds 5 --out out/fill-rejection
```

새 출력 폴더를 사용하세요. 이 기능은 joint construction v5를 출력하는 새
컴파일러 소스가 필요합니다. 기존 공개 바이너리 0.6.14는 첫 타입 오류에서
멈춥니다. CI에는 지원하는 컴파일러 커밋이 고정되어 있습니다.

실제 반복은 5회차, 총 13번의 시도로 끝났고 별도 평가 4개를 만족했습니다.
첫 실패 사례를 그대로 다음 조립 조건으로 옮기는 판단은 Gooo가 수행합니다.
매 회차는 처음부터 조립하므로 이전에 했던 시도도 총합에 포함합니다.

혼합 예제에는 레코드 선택지 세 개가 더 있어 조합이 40개입니다. 기존 자체
소형 모델로 그 선택지 순서를 정하면 이번 실행에서는 5번의 시도 중 3개가
실행됐습니다. 기본 순서는 40번의 시도 중 24개가 실행됐고, 결과는 모두 4/4였습니다.
모델은 레코드 순서에 한 번 사용됐고, 빈칸을 채우는 후보 선택은 결정론적으로
진행됐습니다. 두 관련 예제의 결과라서 일반 정답률이나 속도 우위로 볼 수는 없습니다.

`go run ./experiments/fill-rejection-observe`로 보관한 여섯 기록의 숫자를 다시
계산할 수 있습니다. 정수는 JSON Number로 읽고 원래 기대값과 비교합니다.
원본 반복 기록과 빌드 정보는 `publication/caller-fill-rejection-20261009`에 있습니다.
