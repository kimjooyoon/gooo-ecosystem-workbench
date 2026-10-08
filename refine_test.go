package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func refinementOptions(t *testing.T) RefineOptions {
	t.Helper()
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native source refinement")
	}
	return RefineOptions{Options: Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "refine")},
		Source: "examples/source-refinement/source.gooo", Activity: "Select", Cases: "examples/source-refinement/feedback-cases.json",
		Policy: "examples/source-refinement/policy.gooo", EvaluationCases: "examples/source-refinement/evaluation-cases.json", MaxAttempts: 8, MaxRounds: 4}
}

func TestNativeRefinementDispatchesGoooBudgetAndRetainsSource(t *testing.T) {
	for _, model := range []string{"", "builtin"} {
		t.Run("model="+model, func(t *testing.T) {
			o := refinementOptions(t)
			o.Model = model
			before, err := os.ReadFile(o.Source)
			if err != nil {
				t.Fatal(err)
			}
			r, err := Refine(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if !r.Dispatched || r.InitialPlan.Action != "raise-attempt-budget" || r.Status != "PASS" || r.Final.Passed != 14 || r.Final.Total != 14 || r.EvaluationStatus != "PASS" || r.Evaluation.Passed != 6 || r.FinalPlan.Action != "observe-new-inputs" {
				t.Fatal("Gooo did not refine its own declared budget", r)
			}
			after, _ := os.ReadFile(o.Source)
			selected, err := os.ReadFile(filepath.Join(o.Out, r.SelectedSource))
			if err != nil || string(before) != string(after) || strings.Count(string(selected), "value_case") <= strings.Count(string(before), "value_case") {
				t.Fatal("original source changed or explicit feedback was not incorporated", err)
			}
			if model == "" && (r.InitialModelCalls != 0 || r.RefinementModelCalls != 0 || r.Rounds != 3) || model != "" && (r.InitialModelCalls != 1 || r.RefinementModelCalls != r.Rounds) {
				t.Fatal("model work was hidden or repeated rounds lost", r)
			}
			final, _ := os.ReadFile(filepath.Join(o.Out, "final-result.json"))
			var replay result
			if err = json.Unmarshal(final, &replay); err != nil || replay.Runtime.Calls != 0 || replay.Generated {
				t.Fatal("final program was not replayed without inference", err)
			}
		})
	}
}

func TestNativeRefinementPreservesBoundedPartialOutcome(t *testing.T) {
	o := refinementOptions(t)
	o.MaxAttempts = 2
	o.MaxRounds = 2
	r, err := Refine(context.Background(), o)
	if err != nil || !r.Dispatched || r.Status != "PROGRESS" || r.Final.Passed >= r.Final.Total || r.Rounds > 2 {
		t.Fatal("bounded partial outcome was lost", r, err)
	}
	for _, observation := range r.Final.Construction {
		if observation.Budget > 2 {
			t.Fatal("source budget escaped requested limit", observation)
		}
	}
}

func TestNativeRefinementDispatchesMissingConstructionCases(t *testing.T) {
	o := refinementOptions(t)
	source, err := os.ReadFile(o.Source)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(string(source), "\n") {
		if strings.Contains(line, "value_case") && !strings.Contains(line, "kept") && !strings.Contains(line, "already") {
			continue
		}
		lines = append(lines, line)
	}
	o.Source = filepath.Join(t.TempDir(), "underconstrained.gooo")
	if err = os.WriteFile(o.Source, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Refine(context.Background(), o)
	if err != nil || !r.Dispatched || r.InitialPlan.Action != "add-runtime-cases-to-construction" || r.Final.Passed != 14 || r.Status != "PASS" {
		t.Fatal("explicit counterexample route failed", r, err)
	}
}

func TestNativeRefinementWithholdsFinalEvaluationAndSkipsUnneededEdits(t *testing.T) {
	o := refinementOptions(t)
	o.Source = "examples/source-budget/assembly.gooo"
	o.Cases = "examples/source-budget/cases.json"
	raw, err := os.ReadFile(o.EvaluationCases)
	if err != nil {
		t.Fatal(err)
	}
	var evaluation struct {
		Schema string `json:"schema"`
		Cases  []struct {
			Inputs   map[string]any `json:"inputs"`
			Expected map[string]any `json:"expected"`
		} `json:"cases"`
	}
	if err = json.Unmarshal(raw, &evaluation); err != nil {
		t.Fatal(err)
	}
	evaluation.Cases[0].Expected["Label"] = "different final obligation"
	o.EvaluationCases = filepath.Join(t.TempDir(), "evaluation.json")
	if err = save(o.EvaluationCases, evaluation); err != nil {
		t.Fatal(err)
	}
	r, err := Refine(context.Background(), o)
	if err != nil || r.Dispatched || r.RefinementStatus != "NOT_RUN" || r.Final.Passed != 14 || r.Status != "PROGRESS" || r.EvaluationStatus != "PROGRESS" || r.Evaluation.Passed != 5 || r.FinalPlan.Action != "observe-new-inputs" {
		t.Fatal("evaluation affected source revision or was hidden", r, err)
	}
}

func TestRefinementBoundsAndCancellation(t *testing.T) {
	o := refinementOptions(t)
	o.MaxRounds = 9
	if _, err := Refine(context.Background(), o); err == nil {
		t.Fatal("invalid round limit accepted")
	}
	if _, err := os.Stat(o.Out); !os.IsNotExist(err) {
		t.Fatal("invalid bounds started work", err)
	}
	o.MaxRounds = 4
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := Refine(ctx, o)
	if err == nil || r.Status != "ERROR" {
		t.Fatal("canceled refinement succeeded", r, err)
	}
	if _, err = os.Stat(filepath.Join(o.Out, "refinement-dispatch.json")); err != nil {
		t.Fatal("cancellation observation missing", err)
	}
}

func TestRefinementReadsSourceBoundBeforeLoadingModel(t *testing.T) {
	o := refinementOptions(t)
	o.MaxAttempts = 1
	o.Model = "/missing/model.json"
	r, err := Refine(context.Background(), o)
	if err == nil || !strings.Contains(err.Error(), "source attempt budget 2") || strings.Contains(err.Error(), "model.json") || r.Dispatched {
		t.Fatal("initial source exceeded the caller's bound or loaded a model", r, err)
	}
	if _, err = os.Stat(filepath.Join(o.Out, "initial-result.json")); !os.IsNotExist(err) {
		t.Fatal("out-of-bound source constructed candidates", err)
	}
}
