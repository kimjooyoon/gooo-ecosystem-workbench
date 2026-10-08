package workbench

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func nativeOutcomeFixture(t *testing.T, name string) []byte {
	t.Helper()
	f, err := os.Open("examples/caller-native-failure/" + name + ".json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestNativeFaultConstructionPreservesActualCounts(t *testing.T) {
	for _, test := range []struct {
		name                               string
		attempts, rejected, native, passed int64
	}{
		{"budget-5", 5, 2, 3, 1}, {"budget-6", 6, 2, 4, 4}, {"budget-replay", 6, 2, 4, 4},
		{"mixed-fixed", 48, 16, 32, 4}, {"mixed-model", 6, 2, 4, 4}, {"mixed-replay", 6, 2, 4, 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, err := ReadSnapshot(nativeOutcomeFixture(t, test.name))
			if err != nil {
				t.Fatal(err)
			}
			if s.Joint == nil || s.Joint.ProgramAttempts != test.attempts || s.Joint.RejectedAttempts != test.rejected ||
				s.Joint.NativeProgramAttempts != test.native || s.Passed != test.passed || s.Total != 4 {
				t.Fatal("actual source-bound counts changed", s)
			}
			faultAttempts := int64(1)
			if test.name == "mixed-fixed" {
				faultAttempts = 8
			}
			if s.Joint.NativeFaultAttempts != faultAttempts || s.Joint.NativeFaults != 0 || s.Joint.BlockedActivities != 0 {
				t.Fatal("historical faults mixed with selected outcomes", s.Joint)
			}
		})
	}
}

func TestNativeOutcomesRetainFailedAndBlockedActivities(t *testing.T) {
	for _, name := range []string{"graph", "graph-partial-expectations", "all-faults"} {
		t.Run(name, func(t *testing.T) {
			s, err := ReadSnapshot(nativeOutcomeFixture(t, name))
			if err != nil {
				t.Fatal(err)
			}
			want := NativeOutcomes{Matched: 6, Faulted: 1, Blocked: 1, FaultedActivities: 1, BlockedActivities: 1}
			if name == "graph-partial-expectations" {
				want.Faulted, want.Blocked = 0, 0
			}
			if name == "all-faults" {
				want = NativeOutcomes{Faulted: 1, FaultedActivities: 1}
			}
			if s.NativeOutcomes == nil || *s.NativeOutcomes != want || s.Passed != want.Matched || s.Total != want.Matched+want.Faulted+want.Blocked || s.Unit != "activity_outputs" {
				t.Fatal("native outcomes or denominator changed", s)
			}
			if name == "all-faults" && (s.Joint.NativeFaultAttempts != 2 || s.Joint.NativeFaults != 1 || s.Joint.Decision != "PARTIAL_FINITE" || s.Joint.StopReason != "DECLARED_SPACE_EXHAUSTED") {
				t.Fatal("exhausted faulty space became complete", s.Joint)
			}
		})
	}
}

func TestNativeOutcomesRejectInconsistentObservations(t *testing.T) {
	for name, change := range map[string]func(map[string]any, map[string]any, map[string]any){
		"schema":                  func(r, d, b map[string]any) { r["schema"] = "gooo/body-composition-runtime/v2" },
		"missing outcomes":        func(r, d, b map[string]any) { delete(r, "outcomes") },
		"wrong count":             func(r, d, b map[string]any) { r["outcomes"].(map[string]any)["faulted"] = 0 },
		"missing sites":           func(r, d, b map[string]any) { delete(r, "fault_sites") },
		"site changed":            func(r, d, b map[string]any) { d["fault"].(map[string]any)["site"].(map[string]any)["operator"] = "%" },
		"nonzero divisor":         func(r, d, b map[string]any) { d["fault"].(map[string]any)["right"] = 1 },
		"missing operand":         func(r, d, b map[string]any) { delete(d["fault"].(map[string]any), "left") },
		"invented output":         func(r, d, b map[string]any) { d["actual"] = 0 },
		"invented blocked output": func(r, d, b map[string]any) { b["actual"] = 1 },
		"missing dependency":      func(r, d, b map[string]any) { b["blocked_by"] = nil },
		"wrong dependency":        func(r, d, b map[string]any) { b["blocked_by"] = []string{"faults://activity/first"} },
		"invented producer":       func(r, d, b map[string]any) { b["producer_id"] = "absent" },
		"invented input":          func(r, d, b map[string]any) { b["input"] = 0 },
		"passing fault":           func(r, d, b map[string]any) { d["passed"] = true },
		"missing pass flag":       func(r, d, b map[string]any) { delete(d, "passed") },
		"one execution":           func(r, d, b map[string]any) { r["runs"] = r["runs"].([]any)[:1] },
		"failed execution":        func(r, d, b map[string]any) { r["runs"].([]any)[0].(map[string]any)["exit_code"] = 2 },
		"timed out":               func(r, d, b map[string]any) { r["runs"].([]any)[0].(map[string]any)["timed_out"] = true },
		"different replay": func(r, d, b map[string]any) {
			r["runs"].([]any)[0].(map[string]any)["stdout_sha256"] = r["generated_sha256"]
		},
		"large integer rounded": func(r, d, b map[string]any) {
			for _, value := range r["traces"].([]any)[0].(map[string]any)["deliveries"].([]any) {
				row := value.(map[string]any)
				if row["activity_id"] == "faults://activity/first" {
					row["actual"] = json.Number("9007199254740992")
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			value, err := decodeValue(nativeOutcomeFixture(t, "graph"))
			if err != nil {
				t.Fatal(err)
			}
			r := value.(map[string]any)["runtime"].(map[string]any)
			rows := r["traces"].([]any)[0].(map[string]any)["deliveries"].([]any)
			var faulted, blocked map[string]any
			for _, v := range rows {
				d := v.(map[string]any)
				if d["fault"] != nil {
					faulted = d
				}
				if d["blocked_by"] != nil {
					blocked = d
				}
			}
			if faulted == nil || blocked == nil {
				t.Fatal("fixture lost fault or dependency")
			}
			change(r, faulted, blocked)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("inconsistent observations accepted")
			}
		})
	}
}

func TestNativeArithmeticFaultBecomesUnchangedCounterexample(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo feedback")
	}
	base := "examples/caller-native-failure/"
	out := filepath.Join(t.TempDir(), "loop")
	r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{
		Source: base + "adaptive-fault.gooo", ConstructionCases: base + "adaptive-initial.json", EvaluationCases: base + "adaptive-evaluation.json",
		HoldoutCases: base + "adaptive-holdout.json", Entry: "Main", MaxProgramBudget: 2, MaxRounds: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rounds) != 3 || r.Rounds[0].Action != "add-counterexamples-to-construction" || r.Rounds[0].Feedback == nil || !r.Rounds[0].Feedback.Consumed ||
		r.Rounds[1].Action != "rerun-with-larger-program-budget" || r.Rounds[2].Attempts != 2 || r.FinalEvaluation == nil || r.FinalEvaluation.Passed != 3 || r.FinalEvaluation.Total != 3 || r.FinalEvaluation.Joint.NewModelCalls != 0 {
		t.Fatal("native failure was not consumed as a counterexample", r)
	}
	first, err := os.ReadFile(filepath.Join(out, r.Rounds[0].Result))
	if err != nil {
		t.Fatal(err)
	}
	s, err := ReadSnapshot(first)
	if err != nil {
		t.Fatal(err)
	}
	if s.NativeOutcomes == nil || s.NativeOutcomes.Faulted != 1 || s.Joint.NativeFaultAttempts != 0 {
		t.Fatal("evaluation fault mixed with construction", s)
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
	original, err := os.ReadFile(base + "adaptive-evaluation.json")
	if err != nil {
		t.Fatal(err)
	}
	v, err := decodeValue(original)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows[1], v.(map[string]any)["cases"].([]any)[0]) {
		t.Fatal("native-fault counterexample changed")
	}
}

func TestNativeOutcomeVersionRetainsRecordSelectors(t *testing.T) {
	if err := validateJointKinds("gooo/joint-construction/v6", nil, []int{0}, 1, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := validateJointKinds("gooo/joint-construction/v6", nil, []int{0}, 0, 1, 0); err == nil {
		t.Fatal("implicit selector hid a search")
	}
	value, err := decodeValue(nativeOutcomeFixture(t, "budget-6"))
	if err != nil {
		t.Fatal(err)
	}
	value.(map[string]any)["construction"].(map[string]any)["schema"] = "gooo/joint-construction/v5"
	raw, _ := json.Marshal(value)
	if _, err = ReadSnapshot(raw); err == nil {
		t.Fatal("native fault history was downgraded")
	}
}

func TestNativeFaultDiagnosticsAndFeedback(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo feedback")
	}
	for _, test := range []struct{ name, code string }{{"graph", "native-fault"}, {"graph-partial-expectations", "unscored-native-fault"}, {"all-faults", "caller-space-exhausted"}} {
		t.Run(test.name, func(t *testing.T) {
			s, err := ReadSnapshot(nativeOutcomeFixture(t, test.name))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "diagnosis")}, s)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic struct {
				Code  string
				Calls int `json:"new_model_calls"`
			}
			if json.Unmarshal(raw, &diagnostic) != nil || diagnostic.Code != test.code || diagnostic.Calls != 0 {
				t.Fatal(string(raw))
			}
		})
	}
	base := "examples/caller-native-failure/"
	out := filepath.Join(t.TempDir(), "loop")
	r, err := ConstructJoint(context.Background(), Options{Compiler: compiler, Out: out}, JointRequest{
		Source: base + "source.gooo", ConstructionCases: base + "initial-cases.json", EvaluationCases: base + "evaluation-cases.json",
		HoldoutCases: base + "holdout-cases.json", Entry: "Main", MaxProgramBudget: 8, MaxRounds: 5})
	if err != nil {
		t.Fatal(err)
	}
	var attempts int64
	for _, round := range r.Rounds {
		attempts += round.Attempts
	}
	if len(r.Rounds) != 5 || attempts != 14 || r.FinalEvaluation == nil || r.FinalEvaluation.Passed != 4 || r.FinalEvaluation.Total != 4 ||
		r.FinalEvaluation.Joint.NativeFaultAttempts != 1 || r.FinalEvaluation.Joint.RejectedAttempts != 2 || r.FinalEvaluation.Joint.NewModelCalls != 0 ||
		r.Rounds[0].Feedback == nil || !r.Rounds[0].Feedback.Consumed {
		t.Fatal("native failure interrupted feedback", r, attempts)
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
			t.Fatal("original expectation changed")
		}
	}
}
