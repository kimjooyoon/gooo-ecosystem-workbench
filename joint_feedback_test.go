package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const feedbackCurrent = `{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"A":0},"expected":{"A":0}}]}`
const feedbackEvaluation = `{"schema":"gooo/body-composition-cases/v1","cases":[
 {"inputs":{"A":9007199254740993},"expected":{"A":9007199254740993}},
 {"expected":{"A":9007199254740993},"inputs":{"A":9007199254740993}},
 {"inputs":{"A":3},"expected":{"A":{"x":3},"B":6}}
]}`
const feedbackResult = `{"construction":{"selected":{"plan":{"activities":[{"name":"A","id":"example/a"},{"name":"B","id":"example/b"}]}}},
"evaluation":{"runtime":{"traces":[
 {"case_index":2,"deliveries":[{"activity_id":"example/a","actual":{"x":3},"expected":{"x":3}},{"activity_id":"example/b","actual":3,"expected":6}]},
 {"case_index":0,"deliveries":[{"activity_id":"example/a","actual":9007199254740992,"expected":9007199254740993}]},
 {"case_index":1,"deliveries":[{"activity_id":"example/a","actual":9007199254740992,"expected":9007199254740993}]}
]}}}`

func TestJointFeedbackRowsPreserveExactValuesAndIdentity(t *testing.T) {
	_, rows, err := collectJointFeedback([]byte(feedbackCurrent), []byte(feedbackEvaluation), []byte(feedbackResult))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Matched != 0 || rows[0].Total != 1 || rows[0].Known || !rows[1].Known || rows[2].Matched != 1 || rows[2].Total != 2 {
		t.Fatal(rows)
	}
	if rows[0].RowSHA256 == rows[1].RowSHA256 || rows[0].CanonicalSHA256 != rows[1].CanonicalSHA256 {
		t.Fatal("raw provenance and semantic duplicate identity were mixed")
	}
}

func TestNativeJointFeedbackAndFinalHoldout(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	out := filepath.Join(t.TempDir(), "loop")
	r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{
		Source: "examples/joint-diagnostics/source.gooo", ConstructionCases: "examples/joint-feedback/initial-cases.json",
		EvaluationCases: "examples/joint-feedback/adaptive-cases.json", HoldoutCases: "examples/joint-feedback/holdout-cases.json",
		Entry: "Main", MaxProgramBudget: 8, MaxRounds: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rounds) != 5 || r.Rounds[0].Feedback == nil || !r.Rounds[0].Feedback.Prepared || len(r.Rounds[0].Feedback.AddedIndices) != 2 ||
		r.Rounds[0].Feedback.AddedIndices[0] != 0 || r.Rounds[0].Feedback.AddedIndices[1] != 1 {
		t.Fatalf("missing automatic distinct counterexamples: %+v", r)
	}
	if r.FinalEvaluation == nil || r.FinalEvaluation.Passed != 4 || r.FinalEvaluation.Total != 4 || r.FinalEvaluation.Joint == nil ||
		!r.FinalEvaluation.Joint.Replayed || r.FinalEvaluation.Joint.NewModelCalls != 0 || *r.FinalEvaluation.Joint.Inputs.Other != 4 {
		t.Fatal("final holdout lost separation", r.FinalEvaluation)
	}
	for _, name := range []string{"construction-cases.json", "evaluation-cases.json", "holdout-cases.json"} {
		original := "initial-cases.json"
		if name == "evaluation-cases.json" {
			original = "adaptive-cases.json"
		}
		if name == "holdout-cases.json" {
			original = "holdout-cases.json"
		}
		want, err := os.ReadFile(filepath.Join("examples/joint-feedback", original))
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil || string(got) != string(want) {
			t.Fatal("original expectation file changed", name, err)
		}
	}
	var next struct {
		Cases []json.RawMessage `json:"cases"`
	}
	raw, err := os.ReadFile(filepath.Join(out, r.Rounds[0].Feedback.NextCasesFile))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &next); err != nil || len(next.Cases) != 3 {
		t.Fatal("construction did not retain original and two distinct failures", string(raw), err)
	}
}

func TestJointFeedbackRejectsChangedEvidence(t *testing.T) {
	for _, raw := range []string{
		strings.Replace(feedbackResult, `"case_index":1`, `"case_index":0`, 1),
		strings.Replace(feedbackResult, `"activity_id":"example/b"`, `"activity_id":"example/a"`, 1),
		strings.Replace(feedbackResult, `"actual":3,"expected":6`, `"actual":3,"expected":7`, 1),
		strings.Replace(feedbackResult, `"name":"B"`, `"name":"A"`, 1),
	} {
		if _, _, err := collectJointFeedback([]byte(feedbackCurrent), []byte(feedbackEvaluation), []byte(raw)); err == nil {
			t.Fatal("changed evidence accepted", raw)
		}
	}
}

func TestJointFeedbackInputOnlyHasNoOracle(t *testing.T) {
	evaluation := `{"schema":"gooo/body-composition-inputs/v1","inputs":[{"A":3}]}`
	raw := `{"construction":{"selected":{"plan":{"activities":[{"name":"A","id":"example/a"}]}}},"evaluation":{"runtime":{"traces":[{"case_index":0,"deliveries":[{"activity_id":"example/a","actual":3}]}]}}}`
	_, rows, err := collectJointFeedback([]byte(feedbackCurrent), []byte(evaluation), []byte(raw))
	if err != nil || len(rows) != 1 || rows[0].Total != 0 {
		t.Fatal(rows, err)
	}
}

func TestNativeJointFeedbackPolicyAndLimits(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	o := Options{Compiler: compiler}
	conflict := `{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"A":9007199254740993},"expected":{"A":0}}]}`
	for _, tc := range []struct {
		name, current string
		limit         bool
	}{
		{"distinct counterexamples", feedbackCurrent, false},
		{"conflicting expectation retained", conflict, false},
		{"row limit", `{"schema":"gooo/body-composition-cases/v1","cases":[` + strings.Repeat(`{"inputs":{"A":0},"expected":{"A":0}},`, 127) + `{"inputs":{"A":0},"expected":{"A":0}}]}`, true},
		{"byte limit", `{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"A":"` + strings.Repeat("a", 32620) + `"},"expected":{"A":0}}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			update, err := prepareJointFeedback(context.Background(), o, root, "feedback", []byte(tc.current), []byte(feedbackEvaluation), []byte(feedbackResult))
			if err != nil {
				t.Fatal(err)
			}
			if update.Prepared == tc.limit || update.Consumed || len(update.SelectedIndices) != 2 || update.Rows[1].Include || !update.Rows[2].Include {
				t.Fatal(update)
			}
			if tc.limit {
				if update.StopReason != "construction-cases-limit" || update.NextCasesFile != "" || len(update.AddedIndices) != 0 {
					t.Fatal(update)
				}
				return
			}
			next, err := os.ReadFile(filepath.Join(root, update.NextCasesFile))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := readJointCases(next, false)
			if err != nil || len(doc.Cases) != 3 {
				t.Fatal(doc, err)
			}
			original, _ := readJointCases([]byte(tc.current), false)
			before, _ := jointCanonical(original.Cases[0])
			after, _ := jointCanonical(doc.Cases[0])
			if string(before) != string(after) || update.NextSHA256 != jointDigest(next) {
				t.Fatal("oracle or digest changed")
			}
		})
	}
	cases := []any{}
	for _, tc := range []struct {
		matched, total int
		known, include bool
		reason         string
	}{
		{0, 1, false, true, "counterexample"}, {1, 1, false, false, "matched"}, {0, 1, true, false, "already-retained-or-duplicate"},
		{0, 0, false, false, "no-expectations"}, {2, 1, false, false, "invalid-observation"},
	} {
		input := map[string]any{"matched": tc.matched, "total": tc.total, "known": tc.known}
		cases = append(cases, map[string]any{"inputs": map[string]any{"ObservationEcho": input}, "expected": map[string]any{"ObservationEcho": input, "Decide": map[string]any{"include": tc.include, "reason": tc.reason}}})
	}
	raw, err := json.Marshal(map[string]any{"schema": jointCasesSchema, "cases": cases})
	if err != nil {
		t.Fatal(err)
	}
	_, r, err := runRecipe(context.Background(), o, t.TempDir(), "joint-feedback", "policy", "", raw)
	if err != nil || r.Runtime.Calls != 0 || r.Runtime.Passed != 10 || r.Runtime.Total != 10 {
		t.Fatal(r.Runtime, err)
	}
}

func TestNativeJointFeedbackStopsAtBoundsAndContradictions(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	for _, tc := range []struct {
		name                      string
		rounds                    int
		current, evaluation, stop string
	}{
		{"pending feedback", 1, "examples/joint-feedback/initial-cases.json", "examples/joint-feedback/adaptive-cases.json", "round-limit"},
		{"contradiction", 6, "", "", "expand-declared-choices"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.current == "" {
				tc.current, tc.evaluation = filepath.Join(dir, "initial.json"), filepath.Join(dir, "adaptive.json")
				if err := write(tc.current, []byte(`{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"Main":3},"expected":{"Main":0}}]}`)); err != nil {
					t.Fatal(err)
				}
				if err := write(tc.evaluation, []byte(`{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"Main":3},"expected":{"Main":15}}]}`)); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(dir, "loop")
			loop, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{Source: "examples/joint-diagnostics/source.gooo", ConstructionCases: tc.current, EvaluationCases: tc.evaluation, Entry: "Main", MaxProgramBudget: 8, MaxRounds: tc.rounds})
			if err != nil {
				t.Fatal(err)
			}
			if loop.StopReason != tc.stop || loop.Rounds[0].Feedback == nil || !loop.Rounds[0].Feedback.Prepared || loop.Rounds[0].Feedback.Consumed != (tc.rounds > 1) {
				t.Fatal(loop)
			}
			if tc.rounds > 1 {
				raw, err := os.ReadFile(filepath.Join(out, loop.Rounds[len(loop.Rounds)-1].Result))
				if err != nil {
					t.Fatal(err)
				}
				s, err := ReadSnapshot(raw)
				if err != nil {
					t.Fatal(err)
				}
				if s.Joint.CallerPassed != 1 || s.Joint.CallerTotal != 2 {
					t.Fatal("contradictory requirements were weakened", s.Joint)
				}
			}
		})
	}
}
