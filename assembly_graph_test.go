package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assemblyGraphSeed() string {
	raw := strings.Replace(assemblySeed, `"entry_activity":"A"`, `"entry_activity":"B"`, 1)
	raw = strings.Replace(raw, `"from":-1}]}]`, `"from":-1}]},{"name":"B","id":"example/b","inputs":[{"port":"input","from":0}]}]`, 1)
	return strings.Replace(raw, `}]},"runtime"`, `},{"generation":{"report":{"activity_id":"example/b"}}}]},"runtime"`, 1)
}

func TestAssemblySourceCasesFollowBoundConsumerEntry(t *testing.T) {
	raw, entry, err := sourceAssemblyCases([]byte(assemblyGraphSeed()), "example/a")
	if err != nil || entry != "B" || !strings.Contains(string(raw), "9007199254740993") {
		t.Fatal(string(raw), entry, err)
	}
	if strings.Contains(string(raw), `"B"`) || !strings.Contains(string(raw), `"A.input0"`) {
		t.Fatal("source-local expectation became a consumer oracle", string(raw))
	}
	for _, changed := range []string{
		strings.Replace(assemblyGraphSeed(), `"entry_activity":"B"`, `"entry_activity":"missing"`, 1),
		strings.Replace(assemblyGraphSeed(), `"port":"input","from":0`, `"port":"input","from":-1`, 1),
		strings.Replace(assemblyGraphSeed(), `"activity_id":"example/b"`, `"activity_id":"wrong"`, 1),
		strings.Replace(assemblyGraphSeed(), `"name":"B"`, `"name":"A"`, 1),
	} {
		if _, _, err := sourceAssemblyCases([]byte(changed), "example/a"); err == nil {
			t.Fatal("accepted ambiguous graph", changed)
		}
	}
}

func TestAssemblyExecutionAllowsFixedConsumersOnly(t *testing.T) {
	source := "sha256:" + strings.Repeat("a", 64)
	p := assemblyPreflight{SourceSHA: source, ActivityID: "example/a"}
	raw := `{"generated_now":true,"composition":{"original_source_sha256":"` + source +
		`","generated_sha256":"sha256:` + strings.Repeat("b", 64) +
		`","steps":[{"generation":{"report":{"activity_id":"example/a","record_assembly":{"model_calls":0,"fields_passed":0,"fields_total":0}}}},` +
		`{"generation":{"report":{"activity_id":"example/b"}}}]},` +
		`"runtime":{"stage":"COMPLETE","model_calls":0,"finite_passed":0,"finite_total":0,"projection_replayed":true,"runtime_replayed":true}}`
	if _, err := readAssemblyExecution([]byte(raw), p, true, false); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []string{
		strings.Replace(raw, `"activity_id":"example/b"`, `"activity_id":"example/a"`, 1),
		strings.Replace(raw, `"activity_id":"example/b"`, `"activity_id":"example/b","record_assembly":{}`, 1),
		strings.Replace(raw, `"activity_id":"example/b"`, `"activity_id":"example/b","body_search":{}`, 1),
		strings.Replace(raw, `"activity_id":"example/b"`, `"activity_id":"example/b","body_fill":{}`, 1),
		strings.Replace(raw, `"steps":[`, `"preparations":[{}],"steps":[`, 1),
		strings.Replace(raw, `"fields_total":0`, `"unused":0`, 1),
	} {
		if _, err := readAssemblyExecution([]byte(changed), p, true, false); err == nil {
			t.Fatal("accepted unsupported graph", changed)
		}
	}
}

func TestNativeAssemblyGraphFeedsFinalCallerFailures(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for bound assembly feedback")
	}
	model := "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json"
	for _, originModel := range []string{"", model} {
		name := "fixed"
		if originModel != "" {
			name = "model"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			origin := filepath.Join(root, "origin")
			report, err := Assemble(context.Background(), Options{Compiler: compiler, Model: originModel, Out: origin},
				AssemblyRequest{Source: "examples/assembly-graph-feedback/source.gooo", Entry: "Present", AssemblyActivity: "Describe",
					Cases: "examples/assembly-graph-feedback/adaptive-cases.json"})
			if err != nil || !report.Observation.ReplayVerified || report.Observation.NamedTotal != 2 || report.Observation.FieldsTotal != 6 ||
				report.Observation.FieldsPassed == 6 {
				t.Fatal("missing partial consumer observation", report, err)
			}
			for _, nextModel := range []string{"", model} {
				out := filepath.Join(root, "next-fixed")
				if nextModel != "" {
					out = filepath.Join(root, "next-model")
				}
				r, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Model: nextModel, Out: out},
					AssemblyConstructionRequest{Assembly: origin, HoldoutCases: "examples/assembly-graph-feedback/holdout-cases.json", MaxProgramBudget: 8, MaxRounds: 5})
				if err != nil || !r.ReplayVerified || r.Feedback == nil || !r.Feedback.Consumed || r.Loop == nil ||
					r.Loop.FinalEvaluation == nil || r.Loop.FinalEvaluation.Passed != 6 || r.Loop.FinalEvaluation.Total != 6 ||
					r.Loop.FinalEvaluation.Joint.NewModelCalls != 0 || *r.Loop.FinalEvaluation.Joint.Inputs.Other != 2 {
					t.Fatal("caller feedback did not reach graph construction", r, err)
				}
				seed, err := os.ReadFile(filepath.Join(out, "source-cases.json"))
				if err != nil || strings.Contains(string(seed), `"Present"`) || !strings.Contains(string(seed), `"Describe.input0"`) {
					t.Fatal("source oracle or root mapping changed", string(seed), err)
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
					var initial result
					if err = json.Unmarshal(append(append([]byte(`{"composition":`), doc.Construction.Initial...), '}'), &initial); err != nil {
						t.Fatal(err)
					}
					calls := 0
					for _, step := range initial.Composition.Steps {
						if step.Generation.Report.Assembly != nil {
							calls += step.Generation.Report.Assembly.Calls
						}
					}
					want := 0
					if nextModel != "" {
						want = 1
					}
					if calls != want || len(initial.Composition.Steps) != 2 {
						t.Fatal("fresh graph/model route differs", calls, want)
					}
				}
				for _, file := range assemblyOriginNames {
					before, e1 := os.ReadFile(filepath.Join(origin, file))
					after, e2 := os.ReadFile(filepath.Join(out, "origin", file))
					if e1 != nil || e2 != nil || string(before) != string(after) {
						t.Fatal("original graph changed", file, e1, e2)
					}
				}
			}
		})
	}
}
