package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJointSearchSnapshotRecountsMixedLocalCases(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		attempts, passed, local int64
		replay                  bool
	}{
		{"search", 2, 3, 1, false}, {"mixed-search", 9, 4, 2, false}, {"mixed-search-replay", 9, 4, 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ReadSnapshot(jointFixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			if s.Passed != tc.passed || s.Total != tc.passed || s.Joint == nil || s.Joint.ProgramAttempts != tc.attempts || s.Joint.LocalPassed != tc.local || s.Joint.LocalTotal != tc.local || s.Joint.Replayed != tc.replay || len(s.Joint.CandidateKinds) != int(tc.local) || len(s.Joint.Initial) != int(tc.local) {
				t.Fatal(s)
			}
		})
	}
}

func TestJointSearchSnapshotRejectsInconsistentKindsAndValues(t *testing.T) {
	for name, change := range map[string]func(map[string]any, map[string]any){
		"v1 search":          func(c, a map[string]any) { c["schema"] = "gooo/joint-construction/v1" },
		"missing kind":       func(c, a map[string]any) { delete(c, "candidate_kinds") },
		"unknown kind":       func(c, a map[string]any) { c["candidate_kinds"] = []any{"other", "record_mask"} },
		"count mismatch":     func(c, a map[string]any) { a["search_candidates"] = []any{} },
		"missing index":      func(c, a map[string]any) { a["masks"] = []any{0} },
		"out of bound index": func(c, a map[string]any) { a["masks"] = []any{16, 7} },
		"unscored": func(c, a map[string]any) {
			a["search_candidates"].([]any)[0].(map[string]any)["attempt"].(map[string]any)["scoring_completed"] = false
		},
		"changed count": func(c, a map[string]any) {
			a["search_candidates"].([]any)[0].(map[string]any)["attempt"].(map[string]any)["test_cases_passed"] = 0
		},
		"changed actual": func(c, a map[string]any) {
			a["search_candidates"].([]any)[0].(map[string]any)["attempt"].(map[string]any)["case_results"].([]any)[0].(map[string]any)["actual"] = json.Number("9007199254740993")
		},
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := decodeValue(jointFixture(t, "mixed-search"))
			if err != nil {
				t.Fatal(err)
			}
			value := decoded.(map[string]any)
			c := value["construction"].(map[string]any)
			a := c["attempts"].([]any)[0].(map[string]any)
			change(c, a)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("changed search observations accepted")
			}
		})
	}
}

func TestNativeJointSearchDiagnostic(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo diagnostics")
	}
	s, err := ReadSnapshot(jointFixture(t, "mixed-search"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Out: filepath.Join(t.TempDir(), "diagnosis")}, s)
	if err != nil {
		t.Fatal(err)
	}
	var plan struct{ Action string }
	if err = json.Unmarshal(raw, &plan); err != nil || plan.Action != "observe-new-inputs" {
		t.Fatal(string(raw), err)
	}
}

func TestJointSearchSnapshotKeepsExactLocalInteger(t *testing.T) {
	var candidate jointSearchCandidate
	err := json.Unmarshal([]byte(`{"schema":"gooo/search-candidate/v1","attempt":{"typecheck_passed":true,"scoring_completed":true,"test_cases_passed":1,"test_cases_total":1,"case_results":[{"actual":9007199254740992,"expected":9007199254740993,"passed":true}]}}`), &candidate)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = recountJointSearch(candidate); err == nil {
		t.Fatal("distinct large integers were counted as equal")
	}
}
