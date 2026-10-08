package workbench

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func searchRefinementOptions(t *testing.T, mixed bool) RefineOptions {
	t.Helper()
	o := refinementOptions(t)
	root := "examples/search-refinement/"
	o.Source, o.Activity = root+"source.gooo", "Add"
	o.Cases, o.EvaluationCases = root+"feedback-cases.json", root+"evaluation-cases.json"
	if mixed {
		o.Source, o.Cases, o.EvaluationCases = root+"mixed.gooo", root+"mixed-feedback-cases.json", root+"mixed-evaluation-cases.json"
	}
	return o
}

func TestNativeSearchRefinesSourceAndComposesWithOwnRecordModel(t *testing.T) {
	for _, mode := range []string{"search", "mixed", "mixed-model"} {
		t.Run(mode, func(t *testing.T) {
			o := searchRefinementOptions(t, mode != "search")
			if mode == "mixed-model" {
				o.Model = "builtin"
			}
			r, err := Refine(context.Background(), o)
			if err != nil || !r.Dispatched || r.Status != "PASS" || r.InitialPlan.Action != "raise-attempt-budget" || r.Final.Passed != r.Final.Total || r.EvaluationStatus != "PASS" || r.Rounds < 2 {
				t.Fatal("source search did not enter refinement", err, r)
			}
			if r.InitialPlan.Observation.Kind != "integer_search" || r.InitialPlan.Observation.Budget != 1 || !r.FinalPlan.Observation.SpaceKnown {
				t.Fatal("search observations missing", r)
			}
			if mode == "mixed-model" && (r.InitialModelCalls != 1 || r.RefinementModelCalls != r.Rounds) || mode != "mixed-model" && (r.InitialModelCalls != 0 || r.RefinementModelCalls != 0) {
				t.Fatal("record model work lost or search called a model", r)
			}
			if mode != "search" && len(r.Final.Construction) != 2 {
				t.Fatal("mixed source lost a construction activity", r.Final.Construction)
			}
		})
	}
}

func TestNativeSearchKeepsUnresolvedGrammarAndCandidateCapVisible(t *testing.T) {
	for _, mode := range []string{"grammar", "cap"} {
		t.Run(mode, func(t *testing.T) {
			o := searchRefinementOptions(t, false)
			raw, err := os.ReadFile(o.Source)
			if err != nil {
				t.Fatal(err)
			}
			source := strings.Replace(string(raw), "return __GOOO", "return input + __GOOO", 1)
			want := "expand-declared-choices"
			if mode == "cap" {
				source = strings.Replace(source, `max_candidates "16"`, `max_candidates "2"`, 1)
				want = "expand-search-space"
			}
			o.Source = filepath.Join(t.TempDir(), "nested.gooo")
			if err = os.WriteFile(o.Source, []byte(source), 0644); err != nil {
				t.Fatal(err)
			}
			r, err := Refine(context.Background(), o)
			if err != nil || !r.Dispatched || r.Status != "PROGRESS" || r.Final.Passed != 0 || r.FinalPlan.Action != want {
				t.Fatal("remaining grammar or omitted candidates hidden", err, r)
			}
		})
	}
}
