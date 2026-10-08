package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeJointDiagnosticPolicy(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	base, err := ReadSnapshot(jointFixture(t, "complete"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, code, action string
		changes            map[string]any
	}{
		{"complete", "observed-complete", "observe-new-inputs", nil},
		{"missing expectations", "evaluation-unobserved", "add-evaluation-expectations", map[string]any{"evaluation_total": 0, "evaluation_passed": 0}},
		{"counterexample", "evaluation-gap", "add-counterexamples-to-construction", map[string]any{"evaluation_passed": 3}},
		{"consumed roots", "consumed-inputs-only", "add-different-evaluation-inputs", map[string]any{"other_inputs": 0}},
		{"remaining combinations", "program-budget-exhausted", "rerun-with-larger-program-budget", map[string]any{"caller_passed": 0, "more_candidates": true}},
		{"global limit", "program-limit-reached", "refine-declared-choices", map[string]any{"caller_passed": 0, "more_candidates": true, "program_attempts": 64, "program_budget": 64}},
		{"exhausted", "caller-space-exhausted", "expand-declared-choices", map[string]any{"caller_passed": 0, "more_candidates": false}},
		{"local obligations despite passing evaluation", "local-obligations-unmet", "review-local-and-caller-expectations", map[string]any{"local_passed": 0, "more_candidates": false}},
		{"invalid counts", "inconsistent-observation", "replay-construction", map[string]any{"caller_passed": 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := jointInput(base)
			for k, v := range tc.changes {
				input[k] = v
			}
			cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{map[string]any{"inputs": map[string]any{"ObservationEcho": input}, "expected": map[string]any{"ObservationEcho": input}}}})
			if err != nil {
				t.Fatal(err)
			}
			_, r, err := runRecipe(context.Background(), Options{Compiler: compiler}, t.TempDir(), "joint-next", "policy", "", cases)
			if err != nil {
				t.Fatal(err)
			}
			var actual struct{ Code, Action, Message string }
			if err = actualFor(r, "jointnext://activity/next", &actual); err != nil {
				t.Fatal(err)
			}
			if actual.Code != tc.code || actual.Action != tc.action || actual.Message == "" || r.Runtime.Calls != 0 || r.Runtime.Passed != 1 || r.Runtime.Total != 1 {
				t.Fatal("Gooo policy differed", actual, r.Runtime)
			}
		})
	}
}

func TestNativeJointSnapshotDiagnosticIntegration(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo execution")
	}
	for _, tc := range []struct{ name, code string }{
		{"complete", "observed-complete"}, {"partial", "program-budget-exhausted"}, {"replay", "observed-complete"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ReadSnapshot(jointFixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "diagnosis")}, s)
			if err != nil {
				t.Fatal(err)
			}
			var diagnosis struct {
				Code  string
				Joint *JointObservation `json:"joint_construction"`
				Calls int               `json:"new_model_calls"`
			}
			if err = json.Unmarshal(raw, &diagnosis); err != nil || diagnosis.Code != tc.code || diagnosis.Joint == nil || diagnosis.Calls != 0 {
				t.Fatal(string(raw), err)
			}
		})
	}
}
