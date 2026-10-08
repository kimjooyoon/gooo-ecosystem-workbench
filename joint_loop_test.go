package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeJointLoopUsesGoooBudgetsAndOwnModel(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	for _, tc := range []struct {
		name, model, stop  string
		budget             int64
		rounds, wantRounds int
		attempts           int64
	}{
		{"fixed", "", "observe-new-inputs", 8, 4, 4, 15},
		{"graph-model", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json", "observe-new-inputs", 8, 4, 1, 1},
		{"budget limit", "", "program-budget-limit", 2, 4, 2, 3},
		{"round limit", "", "round-limit", 8, 1, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "loop")
			r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Model: tc.model, Out: out}, JointRequest{
				Source: "examples/joint-diagnostics/source.gooo", ConstructionCases: "examples/joint-diagnostics/construction-cases.json",
				EvaluationCases: "examples/joint-diagnostics/evaluation-cases.json", Entry: "Main", MaxProgramBudget: tc.budget, MaxRounds: tc.rounds})
			if err != nil {
				t.Fatal(err)
			}
			if r.StopReason != tc.stop || len(r.Rounds) != tc.wantRounds {
				t.Fatalf("unexpected loop: %+v", r)
			}
			var attempts int64
			for i, round := range r.Rounds {
				attempts += round.Attempts
				if round.Budget != int64(1<<i) {
					t.Fatal("Gooo budget progression changed", r)
				}
				raw, err := os.ReadFile(filepath.Join(out, round.Result))
				if err != nil {
					t.Fatal(err)
				}
				var observed struct {
					Construction struct {
						Initial json.RawMessage `json:"initial"`
					} `json:"construction"`
				}
				if err = json.Unmarshal(raw, &observed); err != nil {
					t.Fatal(err)
				}
				var initial result
				if err = json.Unmarshal(append(append([]byte(`{"composition":`), observed.Construction.Initial...), '}'), &initial); err != nil {
					t.Fatal(err)
				}
				calls := 0
				for _, steps := range [][]constructionStep{initial.Composition.Preparations, initial.Composition.Steps} {
					for _, step := range steps {
						if a := step.Generation.Report.Assembly; a != nil {
							calls += a.Calls
						}
					}
				}
				wantCalls := 0
				if tc.model != "" {
					wantCalls = 1
				}
				if calls != wantCalls {
					t.Fatal("own model usage differs", calls)
				}
			}
			if attempts != tc.attempts {
				t.Fatal("restarted attempts must remain counted", attempts)
			}
			if tc.stop == "observe-new-inputs" {
				last := r.Rounds[len(r.Rounds)-1]
				if last.EvaluationPassed != 7 || last.EvaluationTotal != 7 {
					t.Fatal(last)
				}
				replay, err := command(context.Background(), compiler, "body-construct", "--source", filepath.Join(out, "source.gooo"), "--construction", filepath.Join(out, r.FinalDirectory, "construction.json"), "--cases", filepath.Join(out, "evaluation-cases.json"))
				if err != nil {
					t.Fatal(err)
				}
				s, err := ReadSnapshot(replay)
				if err != nil || s.Passed != 7 || s.Joint == nil || !s.Joint.Replayed || s.Joint.NewModelCalls != 0 {
					t.Fatal("saved replay differed", s, err)
				}
			}
		})
	}
}

func TestJointLoopBoundsAndCancellation(t *testing.T) {
	for _, r := range []JointRequest{{MaxRounds: 0, MaxProgramBudget: 8}, {MaxRounds: 17, MaxProgramBudget: 8}, {MaxRounds: 4, MaxProgramBudget: 65}} {
		if _, err := ConstructJoint(context.Background(), Options{}, r); err == nil {
			t.Fatal("missing bound")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := filepath.Join(t.TempDir(), "canceled")
	_, err := ConstructJoint(ctx, Options{Compiler: "unused-compiler", Out: out}, JointRequest{Source: "examples/joint-diagnostics/source.gooo",
		ConstructionCases: "examples/joint-diagnostics/construction-cases.json", EvaluationCases: "examples/joint-diagnostics/evaluation-cases.json", MaxProgramBudget: 8, MaxRounds: 4})
	if err != context.Canceled {
		t.Fatal(err)
	}
}
