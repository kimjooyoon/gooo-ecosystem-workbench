package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInitialGraphFeedbackKeepsWholeCallerRows(t *testing.T) {
	evaluation := []byte(`{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"A":9007199254740993,"B":"한글"},"expected":{"A":9007199254740993}},{"inputs":{"A":9007199254740993,"B":"한글"},"expected":{"A":0}}]}`)
	raw := strings.Replace(assemblySeed, `"actual":0,"expected":9007199254740993`, `"actual":0,"expected":9007199254740993`, 1)
	raw = strings.Replace(raw, `}]}]}}`, `}]},{"case_index":1,"deliveries":[{"activity_id":"example/a","actual":0,"expected":0}]}]}}`, 1)
	doc, rows, err := collectInitialGraphFeedback(evaluation, []byte(raw))
	if err != nil || len(doc.Cases) != 0 || len(rows) != 2 || rows[0].Known || rows[1].Known || rows[0].Matched != 0 || rows[1].Matched != 1 {
		t.Fatal(doc, rows, err)
	}
	if !strings.Contains(string(rows[0].raw), "9007199254740993") || !strings.Contains(string(rows[0].raw), `"B":"한글"`) {
		t.Fatal("caller roots or exact number lost", string(rows[0].raw))
	}
	if _, _, err = collectJointFeedback([]byte(`{"schema":"gooo/body-composition-cases/v1","cases":[]}`), evaluation, []byte(raw)); err == nil {
		t.Fatal("normal construction accepted empty history")
	}
}

func TestNativeFullGraphAssemblyFeedback(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for full graph assembly")
	}
	for _, fixture := range []string{"chain", "called-fill", "mixed"} {
		t.Run(fixture, func(t *testing.T) {
			base := "examples/full-graph-assembly/" + fixture
			root := t.TempDir()
			origin := filepath.Join(root, "origin")
			report, err := AssembleGraph(context.Background(), Options{Compiler: compiler, Out: origin}, AssemblyRequest{
				Source: base + ".gooo", Entry: "Main", Cases: base + "-cases.json"})
			if err != nil || !report.Observation.ReplayVerified || report.Observation.NamedPassed == report.Observation.NamedTotal || len(report.Bodies) < 1 {
				t.Fatal("missing partial native graph", report, err)
			}
			r, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "next")}, AssemblyConstructionRequest{
				Assembly: origin, HoldoutCases: base + "-holdout.json", MaxProgramBudget: 64, MaxRounds: 8})
			if err != nil || !r.ReplayVerified || r.Feedback == nil || !r.Feedback.Prepared || !r.Feedback.Consumed || r.Loop == nil || r.Loop.FinalEvaluation == nil ||
				r.Loop.FinalEvaluation.Passed != r.Loop.FinalEvaluation.Total || r.Loop.FinalEvaluation.Joint.NewModelCalls != 0 {
				t.Fatal("whole graph feedback was not consumed", r, err)
			}
			next, err := os.ReadFile(filepath.Join(root, "next", r.Feedback.NextCasesFile))
			if err != nil {
				t.Fatal(err)
			}
			retained, err := readJointCases(next, false)
			if err != nil {
				t.Fatal(err)
			}
			original, err := os.ReadFile(base + "-cases.json")
			if err != nil {
				t.Fatal(err)
			}
			caller, err := readJointCases(original, false)
			if err != nil {
				t.Fatal(err)
			}
			for i, index := range r.Feedback.AddedIndices {
				want, _ := jointCanonical(caller.Cases[index])
				got, _ := jointCanonical(retained.Cases[i])
				if !reflect.DeepEqual(want, got) {
					t.Fatal("invented joint oracle", string(got), string(want))
				}
			}
			seed, err := os.ReadFile(filepath.Join(root, "next", "caller-history.json"))
			if err != nil || string(seed) != string(emptyGraphHistory) {
				t.Fatal("source examples were inverted into caller roots", string(seed), err)
			}
			var c assemblyContext
			contextRaw, err := os.ReadFile(filepath.Join(origin, "next-context.json"))
			if err != nil || json.Unmarshal(contextRaw, &c) != nil || c.Schema != graphContextSchema {
				t.Fatal(c, err)
			}
			for _, item := range c.Artifacts {
				want, _ := os.ReadFile(filepath.Join(origin, item.Path))
				got, _ := os.ReadFile(filepath.Join(root, "next/origin", item.Path))
				if string(want) != string(got) {
					t.Fatal("origin changed", item.Path)
				}
			}
		})
	}
}

func TestNativeGraphOwnModelAndUnscoredPaths(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for model and input-only graph routes")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "model")
	report, err := AssembleGraph(context.Background(), Options{Compiler: compiler, Model: "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json", Out: origin}, AssemblyRequest{
		Source: "examples/full-graph-assembly/chain.gooo", Entry: "Main", Cases: "examples/full-graph-assembly/chain-cases.json"})
	if err != nil || report.Observation.ModelCalls < 1 || len(report.Bodies) != 2 || report.Bodies[0].Checkpoint == report.Bodies[1].Checkpoint {
		t.Fatal("own model did not use sequential native graph contexts", report, err)
	}
	r, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "next")}, AssemblyConstructionRequest{Assembly: origin, MaxProgramBudget: 1, MaxRounds: 1})
	if err != nil || r.ModelRequested || r.Feedback == nil || !r.Feedback.Consumed || r.Loop == nil {
		t.Fatal(r, err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "next/construction/round-0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Construction struct {
			Initial struct{ Steps []constructionStep }
		}
	}
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for _, step := range doc.Construction.Initial.Steps {
		if a := step.Generation.Report.Assembly; a != nil && a.Calls != 0 {
			t.Fatal("origin model inherited", a.Calls)
		}
	}
	files, contextDoc, err := readAssemblyOrigin(origin)
	if err != nil {
		t.Fatal(err)
	}
	contextDoc.Artifacts[0].Path = "unknown.json"
	badContext, _ := json.Marshal(contextDoc)
	if err = os.WriteFile(filepath.Join(origin, "next-context.json"), badContext, 0644); err != nil {
		t.Fatal(err)
	}
	badOut := filepath.Join(root, "bad-origin")
	_, err = ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: badOut}, AssemblyConstructionRequest{Assembly: origin, MaxProgramBudget: 1, MaxRounds: 1})
	if err == nil {
		t.Fatal("unknown graph artifact accepted")
	}
	if _, err = os.Stat(badOut); !os.IsNotExist(err) {
		t.Fatal("invalid origin created output", err)
	}
	if err = os.WriteFile(filepath.Join(origin, "next-context.json"), files["next-context.json"], 0644); err != nil {
		t.Fatal(err)
	}
	inputOnly := []byte(`{"schema":"gooo/body-composition-inputs/v1","inputs":[{"Describe.input0":9007199254740993,"Describe.input1":true,"Describe.input2":"입력만","Extra":2}]}`)
	filename := filepath.Join(root, "inputs.json")
	if err = os.WriteFile(filename, inputOnly, 0644); err != nil {
		t.Fatal(err)
	}
	unscored := filepath.Join(root, "unscored")
	report, err = AssembleGraph(context.Background(), Options{Compiler: compiler, Out: unscored}, AssemblyRequest{Source: "examples/full-graph-assembly/chain.gooo", Entry: "Main", Cases: filename})
	if err != nil || report.Observation.NamedTotal != 0 || report.Observation.NamedPassed != 0 || report.Next.Action != "add-caller-expectations" {
		t.Fatal(report, err)
	}
	r, err = ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "unscored-next")}, AssemblyConstructionRequest{Assembly: unscored, MaxProgramBudget: 1, MaxRounds: 1})
	if err != nil || r.Feedback != nil || r.Loop != nil || r.StopReason != "add-caller-expectations" {
		t.Fatal(r, err)
	}
}

func TestGraphNextPolicyUsesSourceCasesAndAvailableBodies(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for graph next policy")
	}
	for _, tc := range []struct {
		passed, total, bodies, sourcePassed, sourceTotal int
		action                                           string
	}{
		{0, 1, 0, 0, 0, "inspect-source-choices"}, {0, 1, 1, 1, 1, "add-counterexamples-to-construction"},
		{1, 1, 1, 0, 1, "inspect-source-choices"}, {0, 0, 1, 1, 1, "add-caller-expectations"}, {1, 1, 1, 1, 1, "observe-new-inputs"},
	} {
		bodies := make([]GraphAssemblyBody, tc.bodies)
		if len(bodies) > 0 {
			bodies[0].SourcePassed, bodies[0].SourceTotal = tc.sourcePassed, tc.sourceTotal
		}
		a, _, err := runGraphAssemblyPolicy(context.Background(), Options{Compiler: compiler}, t.TempDir(), "policy", Summary{NamedPassed: tc.passed, NamedTotal: tc.total}, bodies)
		if err != nil || a.Action != tc.action {
			t.Fatal(tc, a, err)
		}
	}
}

func TestNativeTypedGraphReportsJointProfileLimit(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for typed graph scope")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "origin")
	report, err := AssembleGraph(context.Background(), Options{Compiler: compiler, Out: origin}, AssemblyRequest{
		Source: "examples/full-graph-assembly/typed.gooo", Entry: "Main", Cases: "examples/full-graph-assembly/typed-cases.json"})
	if err != nil || len(report.Bodies) != 1 || report.Bodies[0].Kind != "typed_paths" || report.Next.Action != "expand-joint-profile" {
		t.Fatal(report, err)
	}
	r, err := ConstructFromAssembly(context.Background(), Options{Compiler: compiler, Out: filepath.Join(root, "next")}, AssemblyConstructionRequest{Assembly: origin, MaxProgramBudget: 4, MaxRounds: 4})
	if err != nil || r.StopReason != "expand-joint-profile" || r.Feedback != nil || r.Loop != nil {
		t.Fatal("unsupported joint route started construction", r, err)
	}
}
