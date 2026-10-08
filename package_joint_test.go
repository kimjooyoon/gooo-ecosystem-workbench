package workbench

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func packageJointFixture(t *testing.T, name string) []byte {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata/package-joint", name+".json.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestPackageJointSnapshotOriginalCountsAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name             string
		attempts, passed int64
		replay           bool
	}{
		{"partial", 5, 1, false}, {"complete", 41, 4, false}, {"replay", 41, 4, true},
	} {
		raw := packageJointFixture(t, tc.name)
		s, err := ReadSnapshot(raw)
		if err != nil || s.Package == nil || s.Joint == nil || s.Joint.ProgramAttempts != tc.attempts ||
			s.Passed != tc.passed || s.Total != 4 || s.Joint.Replayed != tc.replay || s.InputSHA != jointDigest(raw) {
			t.Fatal(tc.name, s, err)
		}
		if s.Package.Entry.Package != "app/retry" || s.Package.Entry.Activity != "Main" || s.Joint.NativeFaultAttempts == 0 {
			t.Fatal(s)
		}
	}
}

func TestPackageJointRejectsChangedBindingsAndCounts(t *testing.T) {
	original := packageJointFixture(t, "complete")
	for name, change := range map[string]func(map[string]any){
		"result schema": func(v map[string]any) { v["schema"] = "other" },
		"source":        func(v map[string]any) { v["program"].(map[string]any)["lowered_gooo_source"] = "changed" },
		"entry": func(v map[string]any) {
			v["program"].(map[string]any)["entry"].(map[string]any)["activity"] = "Elsewhere"
		},
		"package name": func(v map[string]any) {
			v["program"].(map[string]any)["activities"].([]any)[0].(map[string]any)["package_path"] = "other"
		},
		"caller cases": func(v map[string]any) {
			v["construction_cases"].(map[string]any)["cases"].([]any)[0].(map[string]any)["expected"].(map[string]any)["app/retry:Main"] = json.Number("1")
		},
		"native count": func(v map[string]any) {
			v["evaluation"].(map[string]any)["runtime"].(map[string]any)["finite_passed"] = json.Number("3")
		},
		"replay flag": func(v map[string]any) { v["evaluation"].(map[string]any)["construction_replayed"] = true },
	} {
		t.Run(name, func(t *testing.T) {
			v, err := decodeValue(original)
			if err != nil {
				t.Fatal(err)
			}
			change(v.(map[string]any)["result"].(map[string]any))
			raw, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ReadSnapshot(raw); err == nil {
				t.Fatal("changed observation accepted")
			}
		})
	}
}

func TestPackageWorkspaceSnapshotCopiesOnlyDeclaredSources(t *testing.T) {
	request := JointRequest{Workspace: "examples/package-caller-construction/gooo.workspace.json"}
	files, err := jointSourceInputs(request)
	if err != nil || len(files) != 3 {
		t.Fatal(err, len(files))
	}
	for name, raw := range files {
		original := filepath.Join("examples/package-caller-construction", strings.TrimPrefix(filepath.ToSlash(name), "workspace/"))
		want, err := os.ReadFile(original)
		if err != nil || string(raw) != string(want) {
			t.Fatal("source changed", name, err)
		}
	}
	for _, invalid := range []JointRequest{{}, {Source: "one", Workspace: "two"}, {Workspace: request.Workspace, Entry: "Main"}} {
		if _, err := jointSourceInputs(invalid); err == nil {
			t.Fatal("ambiguous source mode accepted")
		}
	}
}

func TestNativePackageJointFeedbackPreservesOriginalRows(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native package construction")
	}
	root := "examples/package-caller-construction/"
	out := filepath.Join(t.TempDir(), "loop")
	r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{
		Workspace: root + "gooo.workspace.json", ConstructionCases: root + "initial-cases.json", EvaluationCases: root + "evaluation-cases.json",
		HoldoutCases: root + "holdout-cases.json", MaxProgramBudget: 8, MaxRounds: 5})
	if err != nil {
		t.Fatal(err)
	}
	var attempts int64
	for _, round := range r.Rounds {
		attempts += round.Attempts
	}
	if r.Schema != "gooo/package-construction-loop/v1" || len(r.Rounds) != 5 || attempts != 14 || r.Rounds[0].Feedback == nil || !r.Rounds[0].Feedback.Consumed ||
		r.FinalEvaluation == nil || r.FinalEvaluation.Passed != 4 || r.FinalEvaluation.Total != 4 || r.FinalEvaluation.Package == nil ||
		!r.FinalEvaluation.Joint.Replayed || r.FinalEvaluation.Joint.NewModelCalls != 0 {
		t.Fatal(r, attempts)
	}
	for _, name := range []string{"gooo.workspace.json", "app.gooo.fixture", "budget.gooo.fixture"} {
		want, _ := os.ReadFile(root + name)
		got, err := os.ReadFile(filepath.Join(out, "workspace", name))
		if err != nil || string(want) != string(got) {
			t.Fatal("package source changed", name, err)
		}
	}
	next, err := os.ReadFile(filepath.Join(out, r.Rounds[0].Feedback.NextCasesFile))
	if err != nil {
		t.Fatal(err)
	}
	added, err := readJointCases(next, false)
	if err != nil || len(added.Cases) != 2 {
		t.Fatal(err)
	}
	for i, name := range []string{"initial-cases.json", "evaluation-cases.json"} {
		original, _ := os.ReadFile(root + name)
		doc, err := readJointCases(original, false)
		if err != nil {
			t.Fatal(err)
		}
		before, _ := decodeValue(doc.Cases[0])
		after, _ := decodeValue(added.Cases[i])
		if !reflect.DeepEqual(before, after) {
			t.Fatal("original package-keyed row changed")
		}
	}
	driver, err := os.ReadFile(filepath.Join(out, r.FinalDirectory, "main.go"))
	if err != nil || len(driver) == 0 {
		t.Fatal("driver export missing", err)
	}
}

func TestPackageFeedbackMappingPreservesPortsAndLargeIntegers(t *testing.T) {
	raw := []byte(`{"schema":"gooo/body-composition-cases/v1","cases":[{"inputs":{"app:Main.input0":9007199254740993,"app:Main.input1":2},"expected":{"app:Main":9007199254740995}}]}`)
	mapped, err := translatePackageFeedback(raw, map[string]string{"app:Main": "Lowered"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(mapped)
	if err != nil || !strings.Contains(string(encoded), `"Lowered.input0":9007199254740993`) || !strings.Contains(string(encoded), `"Lowered":9007199254740995`) {
		t.Fatal(string(encoded), err)
	}
	if _, err := translatePackageFeedback(raw, map[string]string{"elsewhere:Main": "Lowered"}); err == nil {
		t.Fatal("unresolved package case accepted")
	}
}

func TestNativePackageJointLimitsPreservePartialRounds(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native package construction")
	}
	root := "examples/package-caller-construction/"
	for _, tc := range []struct {
		name, stop string
		budget     int64
		rounds     int
	}{
		{"budget", "program-budget-limit", 2, 5}, {"rounds", "round-limit", 8, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "partial")
			r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{Workspace: root + "gooo.workspace.json",
				ConstructionCases: root + "initial-cases.json", EvaluationCases: root + "evaluation-cases.json", MaxProgramBudget: tc.budget, MaxRounds: tc.rounds})
			if err != nil || r.StopReason != tc.stop || len(r.Rounds) == 0 {
				t.Fatal(err, r.StopReason)
			}
			if tc.rounds == 1 && (r.Rounds[0].Feedback == nil || r.Rounds[0].Feedback.Consumed) {
				t.Fatal("unexecuted feedback claimed consumption")
			}
			if _, err := os.ReadFile(filepath.Join(out, r.FinalDirectory, "package-construction.json")); err != nil {
				t.Fatal("partial receipt missing", err)
			}
		})
	}
}
