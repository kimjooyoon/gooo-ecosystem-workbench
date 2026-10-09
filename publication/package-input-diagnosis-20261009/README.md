# 실제 입력 결과를 Gooo로 진단하기

2026-10-09 KST. 작업장 소스 `fa4ea8e43285f3c7310ade5635484950498bdc3d`의
깨끗한 Go 1.27.2 빌드로 진단 명령을 실행했습니다. Gooo 규칙은 공개 설치본
0.6.17, 컴파일러 소스 `ae71176b0c180b1bf3a1d8244a10f5f947b03455`로 실행했습니다.

입력은 [원본 패키지 관측 네 개](../../testdata/package-joint/observed-fixtures.md)입니다.
그중 fresh/replay 기록은 기존 자체 소형 모델로 고른 41회 조립 결과를 담습니다.
진단은 그 이력을 읽고 Gooo의 조건문으로 다음 작업을 제안합니다.

| 관측 | 현재 평가 점수 | Gooo의 분류 | 제안 작업 |
| --- | --- | --- | --- |
| 정답 없이 새 입력 실행 | 0/0 | `evaluation-unobserved` | 비교할 평가 기준 추가 |
| 저장 기록으로 새 입력 실행 | 0/0 | `evaluation-unobserved` | 비교할 평가 기준 추가 |
| 조립 한도 5회, 일부 조건 미충족 | 0/0 | `program-budget-exhausted` | 남은 조합의 시도 한도 확대 |
| 정답 없는 입력에서 0으로 나눔 | 0/0 | `unscored-execution-fault` | 실패 입력과 원하는 동작 기록 |

0/0은 현재 입력에 정답이 제공되지 않았다는 뜻입니다. 이전 조립의 기대값과 점수,
실패 이력은 별도로 남습니다. 새 입력을 맞혔다는 비율은 계산하지 않습니다.
네 진단에서 추가 모델 호출은 모두 0회였고, 규칙 실행 결과는 각 원본
`*-policy-result.json.gz`에 있습니다. `*-diagnostic.json.gz`는 진단 명령의 원본 출력입니다.
입력 파일 원문은 testdata에, 실행한 규칙은 `joint-next.gooo`에 보존했습니다.

다시 실행하려면 저장소 루트에서 다음과 같이 합니다. 출력 폴더는 새 이름을 사용합니다.

```sh
gzip -dc testdata/package-joint/observed-fault.json.gz > /tmp/gooo-fault-observation.json
go run ./cmd/workbench diagnose --compiler gooo \
  --input /tmp/gooo-fault-observation.json --out out/input-fault-diagnosis
```

전체 단위 테스트, 해당 경로의 네이티브 실행·경쟁 상태 검사와 정적 검사를 통과했습니다.
과거 패키지 조립 반복의 공개 집계를 다시 계산해 원본과 같음을 확인했습니다.
진단이 제안한 작업은 결과로 전달되며, 이 명령에서 원래 프로그램을 수정하거나
다시 조립하지 않습니다. 원본 프로그램 재검사는 컴파일러의 저장 재실행 경로로 합니다.
