package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJointRejectionSnapshotSeparatesNativeAndRejectedAttempts(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		attempts, rejected, passed int64
		replay                     bool
	}{
		{"search-rejection", 3, 1, 4, false}, {"search-rejection-partial", 2, 1, 1, false},
		{"mixed-rejection", 17, 8, 4, false}, {"mixed-rejection-replay", 17, 8, 4, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ReadSnapshot(jointFixture(t, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			j := s.Joint
			if j == nil || j.ProgramAttempts != tc.attempts || j.RejectedAttempts != tc.rejected ||
				j.NativeProgramAttempts != tc.attempts-tc.rejected || j.Replayed != tc.replay || s.Passed != tc.passed || s.Total != 4 {
				t.Fatal(s)
			}
			for _, a := range j.History {
				if a.Rejection != nil && (a.CallerTotal != 0 || a.CallerPassed != 0) {
					t.Fatal("unexecuted candidate was scored", a)
				}
			}
		})
	}
}

func TestJointRejectionSnapshotRejectsFalseObservations(t *testing.T) {
	for name, change := range map[string]func(map[string]any, map[string]any, map[string]any){
		"old version":        func(c, a, rejection map[string]any) { c["schema"] = "gooo/joint-construction/v2" },
		"missing slot":       func(c, a, rejection map[string]any) { delete(rejection, "slot") },
		"wrong slot":         func(c, a, rejection map[string]any) { rejection["slot"] = 1 },
		"missing rejection":  func(c, a, rejection map[string]any) { delete(a, "rejection") },
		"wrong reason":       func(c, a, rejection map[string]any) { rejection["reason"] = "invented" },
		"selected rejection": func(c, a, rejection map[string]any) { c["selected_attempt"] = 1 },
		"fake native score":  func(c, a, rejection map[string]any) { a["runtime"].(map[string]any)["finite_passed"] = 1 },
		"fake native stage":  func(c, a, rejection map[string]any) { a["runtime"].(map[string]any)["stage"] = "COMPLETE" },
		"fake zero accuracy": func(c, a, rejection map[string]any) {
			a["search_candidates"].([]any)[0].(map[string]any)["attempt"].(map[string]any)["accuracy_percent"] = 0
		},
		"scored rejection": func(c, a, rejection map[string]any) {
			a["search_candidates"].([]any)[0].(map[string]any)["attempt"].(map[string]any)["scoring_completed"] = true
		},
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := decodeValue(jointFixture(t, "search-rejection"))
			if err != nil {
				t.Fatal(err)
			}
			value := decoded.(map[string]any)
			c := value["construction"].(map[string]any)
			a := c["attempts"].([]any)[1].(map[string]any)
			rejection := a["rejection"].(map[string]any)
			change(c, a, rejection)
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("changed rejection accepted")
			}
		})
	}
}

func TestNativeGoooDiagnosesRejectionReceipt(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER to execute Gooo diagnostics")
	}
	s, err := ReadSnapshot(jointFixture(t, "search-rejection"))
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

func TestJointRejectionPrefixShape(t *testing.T) {
	var r jointReceipt
	if err := json.Unmarshal(jointFixture(t, "mixed-rejection"), &r); err != nil {
		t.Fatal(err)
	}
	a := r.Construction.Attempts[8]
	slot := 1
	a.Rejection.Slot = &slot
	if err := validateJointRejection(r.Construction.Schema, []string{"record_mask", "source_search_index"},
		[]int{0, 1}, a.Rejection, 1, a.SearchCandidates, 0, a.Runtime); err != nil {
		t.Fatal("already scored record prefix was rejected", err)
	}
	if err := validateJointRejection(r.Construction.Schema, []string{"record_mask", "source_search_index"},
		[]int{0, 1}, a.Rejection, 0, a.SearchCandidates, 0, a.Runtime); err == nil {
		t.Fatal("missing scored prefix accepted")
	}
	var noRejections jointReceipt
	if err := json.Unmarshal(jointFixture(t, "search"), &noRejections); err != nil {
		t.Fatal(err)
	}
	noRejections.Construction.Schema = "gooo/joint-construction/v3"
	raw, err := json.Marshal(noRejections)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSnapshot(raw); err == nil {
		t.Fatal("v3 without a rejection accepted")
	}
}
