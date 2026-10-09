# 실패한 프로세스 기록을 Gooo로 읽기

조립 중 네이티브 실행이 멈추면 최종 프로그램을 고르기 전에 실패할 수 있습니다.
`diagnose`는 실패한 런타임의 프로세스 기록을 읽고
[`process-next.gooo`](../../recipes/process-next.gooo)의 조건문으로 다음 작업을 구분합니다.
Go 코드는 원본을 읽고 명령을 연결하며, Gooo가 진단 코드·설명·작업을 만듭니다.

```sh
gzip -dc examples/native-process-failure/windows-start.json.gz > /tmp/windows-start.json
go run ./cmd/workbench diagnose --compiler /absolute/path/to/gooo \
  --input /tmp/windows-start.json --out out/windows-start
```

모델 경로를 생략하면 고정 순서로 조립합니다. 기존 자체 모델을 연결하려면
`--model models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json`을
추가합니다. 모델은 선언된 세 선택을 정렬하고, Gooo의 작성된 사례가 선택을 검사합니다.
선택한 프로그램은 저장 기록으로 한 번 더 실행하며 새 추론 없이 같은 진단을 확인합니다.

예제는 컴파일러 `aa79dc94`의 Windows 실행
[37883255912](https://github.com/kimjooyoon/meta-ontology-go/actions/runs/37883255912)에서
보존한 JSON입니다. 원본 출력 첫 줄과 작업 로그에서 각각 추출한 JSON이 같았습니다.
압축을 풀면 당시 바이트가 유지됩니다. 시작 2,854,721,800ns와 대기 510,000ns의
합계는 2,855,231,800ns입니다. 시작 호출이 돌아온 순간 제한이 이미 만료됐으므로
진단은 `deadline-during-start`, 다음 작업은 `inspect-start-and-parent-budget`입니다.

이 예제에서 정책 회귀 사례를 만들었습니다. 따라서 이 입력은 별도 일반화 평가에
해당하지 않습니다. 작성한 두 사례의 필드 검사 6개와 실제 입력의 정답 점수 0/0을
따로 보관합니다. 테스트에는 취소·시작 실패·출력 제한·일반 시간 초과·오류 종료 등
11개 상태 조합이 있습니다. 고정 순서와 모델 선택의 결과를 대조합니다.

## 읽는 범위

- 실패한 body-compose 런타임과 body-construct 시도, `FAIL_CLOSED` 패키지 실행/조립 기록
- 원본 JSON 위치와 보고된 소스·생성 프로그램·실행 파일 지문
- 시작·대기 시간의 정수 합계, 이전 기록의 없는 타이밍/대기 제한, 종료 상태
- 실행 완료와 프로그램 정답 여부, 작성한 선택 사례와 정답 없는 실제 입력의 구분

타입·지문 형식·상태·시간 합계가 맞지 않으면 읽기 오류를 반환합니다. 진단 근거는
제공된 기록입니다. 원래 프로그램의 실행·소스 일치를 확인하려면 해당 소스와 컴파일러
관측이 추가로 필요합니다.
실패 기록에 프로세스 실행 정보가 없으면 기존 진단 경로의 지원 범위를 따릅니다.
저장 이력의 같은 기록이 여러 번 등장할 수 있어 레코드 수를 독립 실행 수로 세지 않습니다.
진단은 다음 작업을 다른 도구가 받아 사용할 수 있는 데이터로 반환합니다.
