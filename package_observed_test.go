package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageObservedConstructionPreservesSeparateEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, decision         string
		attempts, callerPassed int64
	}{
		{"observed-fresh", "COMPLETE_FINITE", 41, 1}, {"observed-replay", "COMPLETE_FINITE", 41, 1},
		{"observed-partial", "PARTIAL_FINITE", 5, 0}, {"observed-fault", "COMPLETE_FINITE", 1, 1},
	} {
		raw := packageJointFixture(t, tc.name)
		s, err := ReadSnapshot(raw)
		if err != nil || s.Package == nil || s.Joint == nil || s.Passed != 0 || s.Total != 0 ||
			s.Joint.ProgramAttempts != tc.attempts || s.Joint.CallerPassed != tc.callerPassed || s.Joint.CallerTotal != 1 ||
			s.Joint.Decision != tc.decision || s.Joint.NewModelCalls != 0 ||
			s.Joint.Replayed != strings.HasSuffix(tc.name, "replay") || s.InputSHA != jointDigest(raw) {
			t.Fatal("unscored execution lost construction evidence or invented correctness", tc.name, s, err)
		}
		if tc.name == "observed-fault" && nativeActivityCount(s, false) != 1 {
			t.Fatal("unscored native failure disappeared", s.NativeOutcomes)
		}
		var envelope packageJointEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatal(err)
		}
		if s.Package.EvaluationMode != "inputs" || s.Package.InputsSHA256 != envelope.InputsSHA {
			t.Fatal("actual input identity is missing", s.Package)
		}
	}
}

func TestPackageObservedConstructionRejectsScoredOrUnboundInputs(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"missing input binding": func(v map[string]any) { delete(v, "inputs_digest") },
		"ambiguous input mode":  func(v map[string]any) { v["cases_digest"] = v["inputs_digest"] },
		"fabricated completion": func(v map[string]any) { v["decision"] = "COMPLETE_FINITE" },
	} {
		t.Run(name, func(t *testing.T) {
			v, err := decodeValue(packageJointFixture(t, "observed-fresh"))
			if err != nil {
				t.Fatal(err)
			}
			change(v.(map[string]any))
			raw, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ReadSnapshot(raw); err == nil {
				t.Fatal("contradictory input-only evidence accepted")
			}
		})
	}
	// A scored record cannot become an unscored record by changing its envelope.
	v, _ := decodeValue(packageJointFixture(t, "complete"))
	r := v.(map[string]any)
	r["decision"], r["inputs_digest"] = "OBSERVED", r["cases_digest"]
	delete(r, "cases_digest")
	raw, _ := json.Marshal(r)
	if _, err := ReadSnapshot(raw); err == nil {
		t.Fatal("scored rows were relabelled as actual-only observations")
	}
}

func TestNativeObservedPackageDiagnosticUsesGooo(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo diagnostic execution")
	}
	for _, tc := range []struct{ name, code, action string }{
		{"observed-replay", "evaluation-unobserved", "add-evaluation-expectations"},
		{"observed-partial", "program-budget-exhausted", "rerun-with-larger-program-budget"},
		{"observed-fault", "unscored-execution-fault", "add-fault-expectations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ReadSnapshot(packageJointFixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "diagnosis")}, s)
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Code, Action string
				Evaluation   struct{ Passed, Total int }
				Calls        int `json:"new_model_calls"`
			}
			if err := json.Unmarshal(raw, &result); err != nil || result.Code != tc.code || result.Action != tc.action ||
				result.Evaluation.Passed != 0 || result.Evaluation.Total != 0 || result.Calls != 0 {
				t.Fatal("Gooo diagnosis invented or lost an observation", string(raw), err)
			}
		})
	}
}
