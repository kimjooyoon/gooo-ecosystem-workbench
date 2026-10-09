package main

import (
	"fmt"
	"os"
)

const commandHelp = `Gooo 작업장: 소스를 조립하고 실제 결과를 이어 쓰기

처음 시작할 때
  scaffold       scalar, record, library 소스 예제 만들기
  assemble       소스·모델 확인 → 본문 조립 → 실제 호출 → 다음 작업 읽기
  construct      호출 결과를 다음 조립 조건에 반영하기

이미 있는 소스와 결과 읽기
  diagnose       저장된 실행 결과에서 다음 작업 찾기
  reference      패키지의 선언을 사용 안내로 만들기
  discover       한글·영문 질문으로 지원 기능 찾기
  api-diff       같은 ID의 선언 변경을 Gooo 규칙으로 분류하기

실험을 이어 갈 때
  verify         제공한 예제 조립과 저장 재생 확인하기
  receipt        verify 결과의 완전성 기록 만들기
  splice         요청한 소스 부분을 바꾸고 결과 확인하기
  refine         실패 사례와 Gooo 정책으로 소스 선택 반복하기
  feature-audit  모델 입력 표현이 후보를 구분하는지 살펴보기

명령별 옵션: gooo-workbench help assemble
첫 예제와 결과 읽기: examples/model-assembly/README.md
assemble에서 --model을 생략하면 소스의 고정 순서로 조립합니다.
assemble --graph는 여러 본문·호출된 보조 함수·추가 입력을 함께 유지합니다.
construct는 조립 중 평가와 마지막 별도 입력을 나눠 요약합니다. 전체 출력: --json
각 실행은 새 --out 폴더에 원본 입력과 결과를 남깁니다.
`

func knownCommand(name string) bool {
	switch name {
	case "verify", "scaffold", "splice", "assemble", "construct", "diagnose", "refine",
		"reference", "discover", "receipt", "feature-audit", "api-diff":
		return true
	}
	return false
}

func runHelp(args []string) error {
	if len(args) == 1 {
		_, err := fmt.Fprint(os.Stdout, commandHelp)
		return err
	}
	if args[0] != "help" || len(args) != 2 {
		return fmt.Errorf("use gooo-workbench help [command]")
	}
	if !knownCommand(args[1]) {
		return fmt.Errorf("unknown command %q; use gooo-workbench --help", args[1])
	}
	return run([]string{args[1], "--help"})
}
