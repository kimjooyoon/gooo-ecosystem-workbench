package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func TestConstructionSummarySeparatesOriginAdaptiveAndFinalInputs(t *testing.T) {
	r := workbench.AssemblyConstruction{
		Origin:         workbench.Summary{NamedPassed: 0, NamedTotal: 3, FieldsPassed: 3, FieldsTotal: 9, ModelCalls: 2},
		ReplayVerified: true,
		Loop: &workbench.JointLoop{StopReason: "add-different-evaluation-inputs", Rounds: []workbench.JointRound{
			{Attempts: 1}, {Attempts: 2}, {Attempts: 4, EvaluationUnit: "record_fields", EvaluationPassed: 9,
				EvaluationTotal: 9, Action: "add-different-evaluation-inputs", Message: "다른 입력에서도 결과를 관측하세요."},
		}, FinalEvaluation: &workbench.Snapshot{Unit: "record_fields", Passed: 6, Total: 6,
			Joint: &workbench.JointObservation{CallerPassed: 99, CallerTotal: 99, Replayed: true}}},
	}
	var out bytes.Buffer
	root := t.TempDir()
	if err := writeConstructionSummary(&out, r, root, true); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"원래 호출 기대값: 0/3 · 필드: 3/9", "원래 조립 추론: 2회", "결정론적 (모델 생략)",
		"3회차 · 프로그램 시도 7회", "조립 중 마지막 평가: 필드 9/9", "마지막 별도 입력: 필드 6/6",
		"다른 입력에서도 결과를 관측하세요.", "새 추론 0회", filepath.Join(root, "assembly-construction.json"), "--json"} {
		if !strings.Contains(text, want) {
			t.Errorf("summary missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "99/99") || strings.Contains(text, "지정 모델 연결") || strings.Count(text, "\n") > 15 {
		t.Fatalf("summary conflated historical callers/model or flooded output: %s", text)
	}
}

func TestConstructionSummaryKeepsUnknownAndLocalLimits(t *testing.T) {
	loop := workbench.JointLoop{StopReason: "program-budget-limit", Rounds: []workbench.JointRound{
		{Attempts: 2, EvaluationUnit: "activity_outputs", Action: "rerun-with-larger-program-budget", Message: "unused"},
	}}
	var out bytes.Buffer
	if err := writeConstructionSummary(&out, loop, t.TempDir(), true); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"지정 모델 연결", "활동 출력 0/0", "기대값: 미측정", "program-budget-limit", "joint-loop.json"} {
		if !strings.Contains(text, want) {
			t.Fatal(want, text)
		}
	}
	for _, invented := range []string{"마지막 별도 입력", "unused", "새 추론 0회", "100%"} {
		if strings.Contains(text, invented) {
			t.Fatal("invented observation", invented, text)
		}
	}
}

type brokenConstructionWriter struct{}

func (brokenConstructionWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestConstructionSummaryPropagatesOutputErrors(t *testing.T) {
	if err := writeConstructionSummary(brokenConstructionWriter{}, workbench.JointLoop{}, t.TempDir(), false); err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatal(err)
	}
	if err := writeConstructionSummary(&bytes.Buffer{}, "unknown", t.TempDir(), false); err == nil {
		t.Fatal("unsupported summary was accepted")
	}
}
