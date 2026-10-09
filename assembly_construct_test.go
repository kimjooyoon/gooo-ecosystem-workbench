package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const assemblySeed = `{"composition":{"plan":{"schema":"gooo/body-composition-plan/v1","entry_activity":"A","activities":[{"name":"A","id":"example/a","inputs":[{"port":"input0","from":-1}]}]},"steps":[{"generation":{"report":{"activity_id":"example/a","record_assembly":{"cases":[{"inputs":[9007199254740993],"expected":{"x":9007199254740993}}]}}}}]},"runtime":{"traces":[{"case_index":0,"deliveries":[{"activity_id":"example/a","actual":0,"expected":9007199254740993}]}]}}`

func TestAssemblySourceCasesPreserveExactExpectations(t *testing.T) {
	raw, entry, err := sourceAssemblyCases([]byte(assemblySeed), "example/a")
	if err != nil || entry != "A" || !strings.Contains(string(raw), "9007199254740993") {
		t.Fatal(string(raw), entry, err)
	}
	var cases jointCases
	if err = json.Unmarshal(raw, &cases); err != nil || len(cases.Cases) != 1 {
		t.Fatal(cases, err)
	}
	var row struct {
		Inputs   map[string]json.RawMessage `json:"inputs"`
		Expected map[string]json.RawMessage `json:"expected"`
	}
	if err = json.Unmarshal(cases.Cases[0], &row); err != nil || string(row.Inputs["A.input0"]) != "9007199254740993" ||
		string(row.Expected["A"]) != `{"x":9007199254740993}` {
		t.Fatal(row, err)
	}
}

func TestAssemblySourceCasesRejectUnsupportedOrIncompleteShapes(t *testing.T) {
	for _, changed := range []string{
		strings.Replace(assemblySeed, `"entry_activity":"A"`, `"entry_activity":"B"`, 1),
		strings.Replace(assemblySeed, `"from":-1`, `"from":0`, 1),
		strings.Replace(assemblySeed, `"input0","from":-1`, `"","from":-1`, 1),
		strings.Replace(assemblySeed, `"inputs":[9007199254740993]`, `"inputs":[]`, 1),
		strings.Replace(assemblySeed, `"expected":{"x":9007199254740993}`, `"actual":{"x":9007199254740993}`, 1),
		strings.Replace(assemblySeed, `"steps":[`, `"preparations":[{}],"steps":[`, 1),
	} {
		if _, _, err := sourceAssemblyCases([]byte(changed), "example/a"); err == nil {
			t.Fatal("accepted incomplete source cases", changed)
		}
	}
}

func TestNativeAssemblyConstructionReusesSourceAndCallerOracles(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native assembly feedback")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "origin")
	_, err := Assemble(context.Background(), Options{Compiler: compiler, Out: origin}, AssemblyRequest{
		Source: "examples/assembly-feedback/source.gooo", Entry: "Describe", Cases: "examples/assembly-feedback/adaptive-cases.json"})
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json"} {
		name := "fixed"
		if model != "" {
			name = "model"
		}
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(root, name)
			r, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Model: model, Out: out},
				AssemblyConstructionRequest{Assembly: origin, HoldoutCases: "examples/assembly-feedback/holdout-cases.json", MaxProgramBudget: 8, MaxRounds: 4})
			if err != nil || !r.ReplayVerified || r.Feedback == nil || !r.Feedback.Prepared || !r.Feedback.Consumed ||
				len(r.Feedback.AddedIndices) != 2 || r.Loop == nil || !r.Loop.InitialCasesConsumed ||
				r.Loop.FinalEvaluation == nil || r.Loop.FinalEvaluation.Passed != 6 || r.Loop.FinalEvaluation.Total != 6 ||
				r.Loop.FinalEvaluation.Joint.NewModelCalls != 0 || *r.Loop.FinalEvaluation.Joint.Inputs.Other != 2 {
				t.Fatal("missing bounded source/caller feedback or independent final inputs", r, err)
			}
			if r.Origin.FieldsPassed != 2 || r.Origin.FieldsTotal != 6 || r.ModelRequested != (model != "") {
				t.Fatal("origin or optional model was relabeled", r)
			}
			for _, name := range assemblyOriginNames {
				want, err := os.ReadFile(filepath.Join(origin, name))
				got, readErr := os.ReadFile(filepath.Join(out, "origin", name))
				if err != nil || readErr != nil || string(want) != string(got) {
					t.Fatal("origin changed", name, err, readErr)
				}
			}
			cases, err := os.ReadFile(filepath.Join(out, r.Feedback.NextCasesFile))
			if err != nil || !strings.Contains(string(cases), "9007199254740993") {
				t.Fatal("lost exact integer", err)
			}
			for _, round := range r.Loop.Rounds {
				raw, err := os.ReadFile(filepath.Join(out, "construction", round.Result))
				if err != nil {
					t.Fatal(err)
				}
				var doc struct {
					Construction struct{ Initial json.RawMessage }
				}
				if err = json.Unmarshal(raw, &doc); err != nil {
					t.Fatal(err)
				}
				var preparation result
				if err = json.Unmarshal(append(append([]byte(`{"composition":`), doc.Construction.Initial...), '}'), &preparation); err != nil {
					t.Fatal(err)
				}
				want := 0
				if model != "" {
					want = 1
				}
				if preparation.Composition.Steps[0].Generation.Report.Assembly.Calls != want {
					t.Fatal("fresh model call differs", round)
				}
			}
		})
	}
	t.Run("prepared but not consumed", func(t *testing.T) {
		wrapper := filepath.Join(root, "fail-construct")
		quoted := "'" + strings.ReplaceAll(compiler, "'", "'\"'\"'") + "'"
		if err := os.WriteFile(wrapper, []byte("#!/bin/sh\nif [ \"$1\" = body-construct ]; then exit 7; fi\nexec "+quoted+" \"$@\"\n"), 0755); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(root, "failed")
		r, err := ConstructFromAssembly(context.Background(), Options{Compiler: wrapper, Out: out},
			AssemblyConstructionRequest{Assembly: origin, MaxProgramBudget: 8, MaxRounds: 4})
		if err == nil || r.Failure == "" || r.StopReason != "construction-failed" || r.Feedback == nil || !r.Feedback.Prepared || r.Feedback.Consumed ||
			r.Loop == nil || r.Loop.InitialCasesConsumed {
			t.Fatal("unexecuted feedback reported as consumed", r, err)
		}
		if _, err := os.ReadFile(filepath.Join(out, "assembly-construction.json")); err != nil {
			t.Fatal("partial progress not saved", err)
		}
	})
	t.Run("changed origin", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(origin, "source.gooo"), []byte("changed"), 0644); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(root, "changed")
		_, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: out},
			AssemblyConstructionRequest{Assembly: origin, MaxProgramBudget: 8, MaxRounds: 4})
		if err == nil {
			t.Fatal("changed origin accepted")
		}
		if _, err = os.Stat(out); !os.IsNotExist(err) {
			t.Fatal("invalid origin created output", err)
		}
	})
}

func TestNativeAssemblyConstructionCompleteRoute(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native retained assembly")
	}
	{
		root := t.TempDir()
		cases := "examples/model-assembly/cases.json"
		origin := filepath.Join(root, "origin")
		_, err := Assemble(context.Background(), Options{Compiler: compiler, Out: origin},
			AssemblyRequest{Source: "examples/model-assembly/source.gooo", Entry: "Describe", Cases: cases})
		if err != nil {
			t.Fatal(err)
		}
		r, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "next")},
			AssemblyConstructionRequest{Assembly: origin, HoldoutCases: "examples/assembly-feedback/holdout-cases.json", MaxProgramBudget: 8, MaxRounds: 4})
		want := "observe-new-inputs"
		if err != nil || r.StopReason != want || r.Loop != nil || r.Feedback != nil || r.Holdout == nil ||
			r.Holdout.NewModelCalls != 0 || r.Holdout.Observation.FieldsPassed != 6 || r.Holdout.Observation.FieldsTotal != 6 {
			t.Fatal("complete observation started fresh construction or lost holdout", r, err)
		}
	}
}

func TestAssemblyFeedbackUnscoredHasNoNewOracle(t *testing.T) {
	evaluation := `{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"A":3}}]}`
	_, rows, err := collectJointFeedback([]byte(feedbackCurrent), []byte(evaluation), []byte(assemblySeed))
	if err != nil || len(rows) != 1 || rows[0].Total != 0 {
		t.Fatal(rows, err)
	}
}

func TestAssemblyFeedbackReadsOriginalComposition(t *testing.T) {
	eval := `{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"A":9007199254740993},"expected":{"A":9007199254740993}}]}`
	_, rows, err := collectJointFeedback([]byte(feedbackCurrent), []byte(eval), []byte(assemblySeed))
	if err != nil || len(rows) != 1 || rows[0].Matched != 0 || rows[0].Total != 1 {
		t.Fatal(rows, err)
	}
}
