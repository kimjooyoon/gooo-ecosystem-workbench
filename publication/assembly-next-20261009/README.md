# 조립 결과에서 다음 작업까지

같은 네 입력으로 명령을 세 번 실행했습니다. 고정 순서와 기존 자체 모델 조립은
호출 4/4·필드 12/12였고, 기대 문자열 하나를 바꾼 모델 호출 관측은 3/4·11/12였습니다.
소스의 조립용 기대값은 그대로라 세 실행 모두 조립용 필드는 12/12입니다.
세 번의 실행을 12개 독립 평가 사례로 세지 않습니다.

| 실행 | 대상 조립 추론 | 실제 호출 | Gooo가 제안한 다음 작업 |
| --- | --- | --- | --- |
| complete-fixed | 0회 | 4/4 | 다른 입력 관측 |
| complete-model | 1회 | 4/4 | 다른 입력 관측 |
| partial-model | 1회 | 3/4 | 원래 입력·기대값을 확인하고 반례를 조립 조건에 추가 |

경로 선택과 후속 제안은 실제 입력의 기대값을 붙이지 않은 Gooo 실행입니다.
각각 정답 개수는 0/0, 모델 호출은 0회이며 저장한 제안의 재생을 확인했습니다.
대상 코드의 저장 재생도 새 추론 없이 같은 출력과 선택 프로그램을 유지했습니다.
후속 제안이 실제 코드 변경이나 다음 모델 호출을 실행한 성과로 표시하지 않습니다.

컴파일러는 `e5be98712f6b435917b297447200ef50e7adb493`, Go 1.27.2,
decision runtime v0.2.26-experimental입니다. workbench 구현은 clean
`e4aa727`에서 빌드했습니다. `build.txt`와 `compiler-build.json`에 전체 소스가 있습니다.
모델은 기존 `models/graph-chooser-20261008/all-data-demonstration/qat_ternary`를
사용했고 추가 학습은 없습니다. 학습 노출이 확인된 독립 평가 자료로 분류하지 않았습니다.

## 처음 실행하기

저장소 루트에서 사전 확인을 지원하는 `gooo-dev`를 사용합니다.

```sh
go run ./cmd/workbench assemble --compiler gooo-dev \
  --source examples/model-assembly/source.gooo --entry Describe \
  --cases examples/model-assembly/cases.json \
  --model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json \
  --out out/assembly-next
```

`next-context.json`은 약 4KB이며 긴 입력 그래프를 포함하지 않습니다. 원본 파일의
경로·SHA256·크기와 최대 8개 불일치 위치를 보여줍니다. 실제 partial 기록의 위치는
`assembly.json`의 `/runtime/traces/0/deliveries/0`입니다. 당시 기대 문자열은
`해보자?`, 출력은 `해보자!`입니다. 숫자 `9007199254740993`은 그대로 보존됐습니다.
이 합성 변경은 기존 모델의 새 오류를 발견한 것으로 분류하지 않습니다.

## 원본 확인하기

각 tar.gz는 실행 폴더의 원본 파일을 그대로 압축했습니다. JSON을 재직렬화하지
않았습니다. 생성 코드, 소스·호출 사례, 사전 확인, 경로 및 후속 규칙, 조립과 재생이
포함됩니다. 옆의 report와 next-context는 같은 원본 바이트의 복사본입니다.

```sh
mkdir -p out/partial-observation
tar -xzf publication/assembly-next-20261009/partial-model.tar.gz \
  -C out/partial-observation
cat out/partial-observation/next-context.json
```

`FILES.sha256`은 공개 파일의 지문입니다. 압축을 푼 파일의 지문과 바이트 수는 각
context의 참조와 비교할 수 있습니다. 입력·기대값을 유지하고 실제 선택을 읽는
사용 흐름의 확인이며, 언어 전체의 완전성이나 모델 일반화 점수로 확대하지 않습니다.
