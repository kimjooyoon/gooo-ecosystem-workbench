package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func writeConstructionSummary(out io.Writer, value any, directory string, modelRequested bool) error {
	root, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	var text strings.Builder
	switch report := value.(type) {
	case workbench.AssemblyConstruction:
		fmt.Fprintf(&text, "후속 조립\n원래 호출 기대값: %d/%d · 필드: %d/%d\n원래 조립 추론: %d회 · 저장 재생: %t\n",
			report.Origin.NamedPassed, report.Origin.NamedTotal, report.Origin.FieldsPassed, report.Origin.FieldsTotal,
			report.Origin.ModelCalls, report.ReplayVerified)
		writeConstructionMode(&text, report.ModelRequested)
		if report.Feedback != nil {
			fmt.Fprintf(&text, "원래 실패 사례를 다음 조립에 반영: %t\n", report.Feedback.Consumed)
		}
		if report.Loop != nil {
			writeLoopSummary(&text, *report.Loop)
		} else {
			fmt.Fprintf(&text, "멈춘 이유: %s\n", report.StopReason)
			if report.Holdout != nil {
				h := report.Holdout
				fmt.Fprintf(&text, "마지막 별도 입력: 호출 기대값 %d/%d · 필드 %d/%d · 새 추론 %d회\n",
					h.Observation.NamedPassed, h.Observation.NamedTotal, h.Observation.FieldsPassed,
					h.Observation.FieldsTotal, h.NewModelCalls)
			}
		}
		fmt.Fprintf(&text, "결과: %s\n", filepath.Join(root, "assembly-construction.json"))
	case workbench.JointLoop:
		fmt.Fprintln(&text, "호출 결과로 조립하기")
		writeConstructionMode(&text, modelRequested)
		writeLoopSummary(&text, report)
		fmt.Fprintf(&text, "결과: %s\n", filepath.Join(root, "joint-loop.json"))
	default:
		return fmt.Errorf("unsupported construction summary %T", value)
	}
	fmt.Fprintln(&text, "점수는 제공한 기대값의 범위입니다. 전체 JSON 출력: --json")
	_, err = io.WriteString(out, text.String())
	return err
}

func writeConstructionMode(text *strings.Builder, modelRequested bool) {
	mode := "결정론적 (모델 생략)"
	if modelRequested {
		mode = "지정 모델 연결"
	}
	fmt.Fprintf(text, "이번 새 조립: %s\n", mode)
}

func writeLoopSummary(text *strings.Builder, loop workbench.JointLoop) {
	var attempts int64
	for _, round := range loop.Rounds {
		attempts += round.Attempts
	}
	fmt.Fprintf(text, "조립: %d회차 · 프로그램 시도 %d회 (회차 간 반복 포함)\n", len(loop.Rounds), attempts)
	if len(loop.Rounds) > 0 {
		last := loop.Rounds[len(loop.Rounds)-1]
		fmt.Fprintf(text, "조립 중 마지막 평가: %s %d/%d\n", constructionUnit(last.EvaluationUnit),
			last.EvaluationPassed, last.EvaluationTotal)
		if last.EvaluationTotal == 0 {
			fmt.Fprintln(text, "조립 중 평가의 기대값: 미측정")
		}
		if loop.StopReason == last.Action && last.Message != "" {
			fmt.Fprintf(text, "Gooo의 다음 작업: %s\n", last.Message)
		}
	}
	fmt.Fprintf(text, "멈춘 이유: %s\n", loop.StopReason)
	if final := loop.FinalEvaluation; final != nil {
		fmt.Fprintf(text, "마지막 별도 입력: %s %d/%d\n", constructionUnit(final.Unit), final.Passed, final.Total)
		if final.Total == 0 {
			fmt.Fprintln(text, "별도 입력의 기대값: 미측정")
		}
		if final.Joint != nil {
			fmt.Fprintf(text, "별도 입력의 저장 재생: %t · 새 추론 %d회\n", final.Joint.Replayed, final.Joint.NewModelCalls)
		}
	}
}

func constructionUnit(unit string) string {
	switch unit {
	case "record_fields":
		return "필드"
	case "activity_outputs":
		return "활동 출력"
	default:
		return unit
	}
}
