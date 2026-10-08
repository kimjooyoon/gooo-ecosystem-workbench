package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fillFixture(t *testing.T, name string) []byte {
	t.Helper()
	return jointFixture(t, "../caller-source-fill/"+name)
}

func TestJointFillSnapshotKeepsHoldoutsSeparate(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		attempts, passed, holdout, initial int64
		replay                             bool
	}{
		{"budget-2", 2, 1, 0, 1, false}, {"budget-3", 3, 4, 1, 1, false}, {"budget-replay", 3, 4, 1, 1, true},
		{"mixed-fixed", 24, 4, 1, 2, false}, {"mixed-model", 3, 4, 1, 2, false}, {"mixed-replay", 3, 4, 1, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ReadSnapshot(fillFixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			j := s.Joint
			if j == nil || j.ProgramAttempts != tc.attempts || j.NativeProgramAttempts != tc.attempts || j.FillHoldoutPassed != tc.holdout ||
				j.FillHoldoutTotal != 1 || j.LocalTotal != tc.initial || s.Passed != tc.passed || s.Total != 4 || j.Replayed != tc.replay || int64(len(j.Initial)) != tc.initial {
				t.Fatal(s)
			}
			for _, initial := range j.Initial {
				if !initial.Observed || !initial.Consistent {
					t.Fatal(initial)
				}
			}
		})
	}
}

func TestJointFillSnapshotRejectsMixedDenominatorsAndValues(t *testing.T) {
	for name, change := range map[string]func(map[string]any, map[string]any, map[string]any){
		"old schema":            func(c, a, f map[string]any) { c["schema"] = "gooo/joint-construction/v2" },
		"unknown kind":          func(c, a, f map[string]any) { c["candidate_kinds"] = []any{"record_mask"} },
		"missing fill":          func(c, a, f map[string]any) { delete(a, "fill_candidates") },
		"merged totals":         func(c, a, f map[string]any) { a["local_total"] = 2 },
		"missing holdout count": func(c, a, f map[string]any) { delete(f, "holdout_cases_total") },
		"false holdout pass":    func(c, a, f map[string]any) { f["holdout_cases_passed"] = 1 },
		"missing holdout":       func(c, a, f map[string]any) { f["value_holdout_results"] = []any{} },
		"both formats":          func(c, a, f map[string]any) { f["case_results"] = f["value_case_results"] },
		"exact integer": func(c, a, f map[string]any) {
			row := f["value_case_results"].([]any)[0].(map[string]any)
			row["actual"], row["expected"] = json.Number("9007199254740992"), json.Number("9007199254740993")
		},
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := decodeValue(fillFixture(t, "budget-3"))
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.(map[string]any)
			c := v["construction"].(map[string]any)
			a := c["attempts"].([]any)[0].(map[string]any)
			f := a["fill_candidates"].([]any)[0].(map[string]any)
			change(c, a, f)
			raw, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("inconsistent fill observations accepted")
			}
		})
	}
}

func TestJointFillRejectedSearchPrefix(t *testing.T) {
	var r jointReceipt
	if err := json.Unmarshal(jointFixture(t, "search-rejection"), &r); err != nil {
		t.Fatal(err)
	}
	a := r.Construction.Attempts[1]
	slot := 1
	a.Rejection.Slot = &slot
	for _, count := range []int{0, 1, 2} {
		err := validateJointRejection("gooo/joint-construction/v4", []string{"source_fill_index", "source_search_index", "record_mask"},
			[]int{0, 1, 0}, a.Rejection, 0, a.SearchCandidates, count, a.Runtime)
		if (err == nil) != (count == 1) {
			t.Fatal(count, err)
		}
	}
}

func TestNativeJointFillFeedbackLoop(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo feedback")
	}
	base := "examples/caller-source-fill/"
	request := JointRequest{Source: base + "source.gooo", ConstructionCases: base + "initial-cases.json", EvaluationCases: base + "evaluation-cases.json",
		HoldoutCases: base + "holdout-cases.json", Entry: "Main", MaxProgramBudget: 4, MaxRounds: 4}
	r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "loop")}, request)
	if err != nil {
		t.Fatal(err)
	}
	if r.StopReason != "add-different-evaluation-inputs" || len(r.Rounds) != 4 || r.Rounds[0].Feedback == nil ||
		!r.Rounds[0].Feedback.Consumed || len(r.Rounds[0].Feedback.AddedIndices) != 1 || r.FinalEvaluation.Passed != 4 ||
		r.FinalEvaluation.Joint.FillHoldoutPassed != 1 || r.FinalEvaluation.Joint.NewModelCalls != 0 {
		t.Fatal(r)
	}
	var attempts int64
	for _, round := range r.Rounds {
		attempts += round.Attempts
	}
	if attempts != 7 {
		t.Fatal("restarted combinations must remain counted", attempts)
	}
	request.FillModel = "/missing/fill-model.json"
	if _, err = ConstructJoint(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "missing")}, request); err == nil {
		t.Fatal("fill model flag ignored")
	}
}
