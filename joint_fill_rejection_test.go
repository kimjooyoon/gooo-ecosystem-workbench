package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func rejectionFillFixture(t *testing.T, name string) []byte {
	t.Helper()
	return jointFixture(t, "../caller-fill-rejection/"+name)
}

func TestJointFillRejectionRecountsAttempts(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		attempts, rejected, passed int64
	}{
		{"budget-3", 3, 2, 1}, {"budget-5", 5, 2, 4}, {"budget-replay", 5, 2, 4},
		{"mixed-fixed", 40, 16, 4}, {"mixed-model", 5, 2, 4}, {"mixed-replay", 5, 2, 4},
	} {
		s, err := ReadSnapshot(rejectionFillFixture(t, tc.name))
		if err != nil {
			t.Fatal(tc.name, err)
		}
		j := s.Joint
		if j == nil || j.ProgramAttempts != tc.attempts || j.RejectedAttempts != tc.rejected || j.NativeProgramAttempts != tc.attempts-tc.rejected || s.Passed != tc.passed || s.Total != 4 {
			t.Fatal(tc.name, j)
		}
		found := false
		for _, o := range j.Initial {
			if o.Kind == "source_fill" {
				found = true
				if !o.Consistent || o.Scored != 3 || o.Rejected != 2 || o.Budget != 5 {
					t.Fatal(o)
				}
			}
		}
		if !found {
			t.Fatal("missing initial fill")
		}
	}
}

func TestJointFillRejectionRejectsInventedScores(t *testing.T) {
	for name, change := range map[string]func(map[string]any, map[string]any, map[string]any){
		"caller score":  func(c, a, f map[string]any) { a["runtime"].(map[string]any)["finite_total"] = 1 },
		"local score":   func(c, a, f map[string]any) { f["test_cases_total"] = 1 },
		"holdout score": func(c, a, f map[string]any) { f["holdout_cases_passed"] = 1 },
		"reason":        func(c, a, f map[string]any) { f["rejection"].(map[string]any)["reason"] = "invented" },
		"stage":         func(c, a, f map[string]any) { f["rejection"].(map[string]any)["stage"] = "invented" },
		"identity":      func(c, a, f map[string]any) { f["candidate_id"] = "invented" },
		"plan":          func(c, a, f map[string]any) { f["plan_sha256"] = "" },
		"fills":         func(c, a, f map[string]any) { f["hole_fills"] = nil },
		"schema":        func(c, a, f map[string]any) { c["schema"] = "gooo/joint-construction/v4" },
		"slot":          func(c, a, f map[string]any) { a["rejection"].(map[string]any)["slot"] = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			v, err := decodeValue(rejectionFillFixture(t, "budget-5"))
			if err != nil {
				t.Fatal(err)
			}
			c := v.(map[string]any)["construction"].(map[string]any)
			a := c["attempts"].([]any)[1].(map[string]any)
			f := a["fill_candidates"].([]any)[0].(map[string]any)
			change(c, a, f)
			raw, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("altered rejection accepted")
			}
		})
	}
}

func TestNativeJointFillRejectionFeedbackLoop(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo feedback")
	}
	base := "examples/caller-fill-rejection/"
	out := filepath.Join(t.TempDir(), "loop")
	r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{
		Source: base + "source.gooo", ConstructionCases: base + "initial-cases.json", EvaluationCases: base + "evaluation-cases.json",
		HoldoutCases: base + "holdout-cases.json", Entry: "Main", MaxProgramBudget: 8, MaxRounds: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rounds) != 5 || r.StopReason != "add-different-evaluation-inputs" || r.FinalEvaluation == nil || r.FinalEvaluation.Passed != 4 ||
		r.Rounds[0].Feedback == nil || !r.Rounds[0].Feedback.Consumed || len(r.Rounds[0].Feedback.AddedIndices) != 1 {
		t.Fatal(r)
	}
	var attempts int64
	for _, round := range r.Rounds {
		attempts += round.Attempts
	}
	if attempts != 13 || r.FinalEvaluation.Joint.RejectedAttempts != 2 || r.FinalEvaluation.Joint.NewModelCalls != 0 {
		t.Fatal(attempts, r)
	}
	next, err := os.ReadFile(filepath.Join(out, r.Rounds[0].Feedback.NextCasesFile))
	if err != nil {
		t.Fatal(err)
	}
	value, err := decodeValue(next)
	if err != nil {
		t.Fatal(err)
	}
	rows := value.(map[string]any)["cases"].([]any)
	if len(rows) != 2 {
		t.Fatal("counterexample count changed")
	}
	for i, name := range []string{"initial-cases.json", "evaluation-cases.json"} {
		original, err := os.ReadFile(base + name)
		if err != nil {
			t.Fatal(err)
		}
		v, err := decodeValue(original)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rows[i], v.(map[string]any)["cases"].([]any)[0]) {
			t.Fatal("original counterexample changed")
		}
	}
}

func TestNativeRejectedFillDoesNotProposeAlreadyCheckedCandidates(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER")
	}
	s := Snapshot{Passed: 0, Total: 1, Construction: []ConstructionObservation{{ActivityID: "fill://activity", Kind: "source_fill",
		Matched: 0, Total: 1, Scored: 1, Rejected: 2, Ranked: 3, Budget: 3, Observed: true, BudgetKnown: true, Consistent: true, SpaceKnown: true}}}
	plans, err := constructionNextSteps(context.Background(), Options{Compiler: compiler}, t.TempDir(), s)
	if err != nil || len(plans) != 1 || plans[0].Action != "expand-declared-choices" {
		t.Fatal(plans, err)
	}
}
