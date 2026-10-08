package workbench

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func jointFixture(t *testing.T, name string) []byte {
	t.Helper()
	f, err := os.Open(filepath.Join("examples", "joint-diagnostics", name+".json.gz"))
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

func TestJointSnapshotRecountsSeparateStages(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		passed, total, attempts, caller int64
		replay                          bool
	}{
		{"complete", 4, 4, 2, 1, false}, {"partial", 1, 7, 1, 0, false}, {"replay", 7, 7, 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := jointFixture(t, tc.name)
			s, err := ReadSnapshot(raw)
			if err != nil {
				t.Fatal(err)
			}
			j := s.Joint
			if j == nil || s.Passed != tc.passed || s.Total != tc.total || j.ProgramAttempts != tc.attempts || j.CallerPassed != tc.caller || j.LocalPassed != 1 || j.LocalTotal != 1 || j.Replayed != tc.replay || len(s.Construction) != 0 || len(j.Initial) != 1 {
				t.Fatalf("stages mixed: %+v %+v", s, j)
			}
			if s.InputSHA != fmt.Sprintf("%x", sha256.Sum256(raw)) {
				t.Fatal("lost outer input identity")
			}
			if j.Initial[0].Matched != 1 || j.Initial[0].Total != 1 {
				t.Fatal("lost historical local preparation")
			}
		})
	}
}

func TestJointSnapshotRejectsInconsistentObservations(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"selected index", func(m map[string]any) { m["construction"].(map[string]any)["selected_attempt"] = 99 }},
		{"missing zero calls", func(m map[string]any) { delete(m["evaluation"].(map[string]any), "new_model_calls") }},
		{"new prediction", func(m map[string]any) { m["evaluation"].(map[string]any)["new_model_calls"] = 1 }},
		{"missing replay flag", func(m map[string]any) { delete(m["evaluation"].(map[string]any), "construction_replayed") }},
		{"historical caller score", func(m map[string]any) {
			m["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)["finite_passed"] = 1
		}},
		{"local score", func(m map[string]any) {
			m["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["local_passed"] = 0
		}},
		{"missing candidate case", func(m map[string]any) {
			m["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["candidates"].([]any)[0].(map[string]any)["cases"] = []any{}
		}},
		{"wrong decision", func(m map[string]any) { m["construction"].(map[string]any)["decision"] = "PARTIAL_FINITE" }},
		{"wrong stop", func(m map[string]any) { m["construction"].(map[string]any)["stop_reason"] = "DECLARED_SPACE_EXHAUSTED" }},
		{"overlap sum", func(m map[string]any) {
			m["evaluation"].(map[string]any)["input_separation"].(map[string]any)["other_inputs"] = 100
		}},
		{"exact integer", func(m map[string]any) {
			m["evaluation"].(map[string]any)["runtime"].(map[string]any)["traces"].([]any)[3].(map[string]any)["deliveries"].([]any)[0].(map[string]any)["actual"] = json.Number("18014398509481987")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := decodeValue(jointFixture(t, "complete"))
			if err != nil {
				t.Fatal(err)
			}
			m := value.(map[string]any)
			tc.mutate(m)
			raw, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = ReadSnapshot(raw); err == nil {
				t.Fatal("inconsistent observation accepted")
			}
		})
	}
}
