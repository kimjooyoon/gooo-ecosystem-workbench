package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const packagePartialSnapshot = `{"schema":"gooo/workspace-body-execution-receipt/v1","decision":"PROGRESS","result":{
"schema":"gooo/workspace-body-execution/v1",
"composition":{"preparations":[{"generation":{"report":{"record_assembly":{"model_calls":1,"attempts":[{"status":"TYPECHECK_FAILED","reason":"helper alternative has wrong type"}]}}}}]},
"runtime":{"schema":"gooo/body-composition-runtime/v1","model_calls":0,"finite_passed":1,"finite_total":2,
"traces":[{"deliveries":[{"activity_id":"example/main","actual":"accept","expected":"continue"}]},
{"deliveries":[{"activity_id":"example/main","actual":"done","expected":"done"}]}]}}}`

func TestSnapshotReadsPackageExecutionAndKeepsOuterIdentity(t *testing.T) {
	s, err := ReadSnapshot([]byte(packagePartialSnapshot))
	if err != nil || s.Unit != "activity_outputs" || s.Passed != 1 || s.Total != 2 || s.Rejected != 1 ||
		s.InputSHA != fmt.Sprintf("%x", sha256.Sum256([]byte(packagePartialSnapshot))) ||
		!strings.Contains(s.Detail, "helper alternative has wrong type") || !strings.Contains(s.Detail, "continue") {
		t.Fatal("package observation lost values, helper rejections, or receipt identity", s, err)
	}
}

func TestSnapshotMixedOutputsKeepsFailedScalar(t *testing.T) {
	raw := []byte(`{"runtime":{"model_calls":0,"finite_passed":1,"finite_total":2,"traces":[{"deliveries":[
{"activity_id":"record","actual":{"value":"ok"},"expected":{"value":"ok"}},
{"activity_id":"scalar","actual":1,"expected":2}]}]}}`)
	s, err := ReadSnapshot(raw)
	if err != nil || s.Unit != "activity_outputs" || s.Passed != 1 || s.Total != 2 || !strings.Contains(s.Detail, `"activity":"scalar"`) {
		t.Fatal("a matching record hid a failed scalar output", s, err)
	}
}

func TestSnapshotPackageInputOnlyRemainsUnobserved(t *testing.T) {
	raw := []byte(`{"schema":"gooo/workspace-body-execution-receipt/v1","decision":"OBSERVED","result":{
"schema":"gooo/workspace-body-execution/v1","runtime":{"model_calls":0,"finite_passed":0,"finite_total":0,
"traces":[{"deliveries":[{"activity_id":"example/main","actual":"a value"}]}]}}}`)
	s, err := ReadSnapshot(raw)
	if err != nil || s.Passed != 0 || s.Total != 0 {
		t.Fatal("input-only execution invented a correctness score", s, err)
	}
}

func TestSnapshotPackageRequiresSuccessfulExecutionEnvelope(t *testing.T) {
	for name, raw := range map[string]string{
		"error":               strings.Replace(packagePartialSnapshot, `"decision":"PROGRESS"`, `"decision":"FAIL_CLOSED","error":"build failed"`, 1),
		"unrecognized result": strings.Replace(packagePartialSnapshot, `"schema":"gooo/workspace-body-execution/v1"`, `"schema":"other"`, 1),
		"missing runtime":     `{"schema":"gooo/workspace-body-execution-receipt/v1","decision":"PASS","result":{"schema":"gooo/workspace-body-execution/v1"}}`,
		"nested envelope":     `{"schema":"gooo/workspace-body-execution-receipt/v1","decision":"PASS","result":` + packagePartialSnapshot + `}`,
		"wrong counts":        strings.Replace(packagePartialSnapshot, `"finite_passed":1`, `"finite_passed":2`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadSnapshot([]byte(raw)); err == nil {
				t.Fatal("invalid package observation accepted")
			}
		})
	}
}

func TestSnapshotUnitsPreserveEveryMismatch(t *testing.T) {
	for _, tc := range []struct {
		name, actual, expected, unit string
		passed, total                int64
	}{
		{"fields", `{"a":1,"b":0}`, `{"a":1,"b":2}`, "record_fields", 1, 2},
		{"missing field", `{"a":1}`, `{"a":1,"b":null}`, "record_fields", 1, 2},
		{"extra field", `{"a":1,"extra":0}`, `{"a":1}`, "activity_outputs", 0, 1},
		{"empty record", `{"extra":0}`, `{}`, "activity_outputs", 0, 1},
		{"record shape", `null`, `{"a":null}`, "activity_outputs", 0, 1},
		{"null expectation", `false`, `null`, "activity_outputs", 0, 1},
		{"exact integer", `9007199254740992`, `9007199254740993`, "activity_outputs", 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(fmt.Sprintf(`{"runtime":{"model_calls":0,"finite_passed":0,"finite_total":1,"traces":[{"deliveries":[{"activity_id":"check","actual":%s,"expected":%s}]}]}}`, tc.actual, tc.expected))
			s, err := ReadSnapshot(raw)
			if err != nil || s.Unit != tc.unit || s.Passed != tc.passed || s.Total != tc.total || !strings.Contains(s.Detail, `"activity":"check"`) {
				t.Fatal("mismatch disappeared or changed units", s, err)
			}
		})
	}
}

func TestSnapshotMissingNativeObservationIsNotZeroCoverage(t *testing.T) {
	for _, raw := range []string{`{"runtime":null}`, `{"runtime":{}}`, `{"passed":null,"total":0}`} {
		if _, err := ReadSnapshot([]byte(raw)); err == nil {
			t.Fatal("missing observations accepted", raw)
		}
	}
	s, err := ReadSnapshot([]byte(`{"passed":1,"total":2,"detail":"one remains"}`))
	if err != nil || s.Unit != "provided_counts" || s.InputSHA == "" {
		t.Fatal(s, err)
	}
}

func TestSnapshotSummarizesHelperAndRootConstruction(t *testing.T) {
	var envelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(packagePartialSnapshot), &envelope); err != nil {
		t.Fatal(err)
	}
	raw := strings.Replace(string(envelope.Result), `"preparations":[`, `"steps":[{"generation":{"report":{"record_assembly":{"model_calls":1,"fields_passed":2,"fields_total":3}}}}],"preparations":[`, 1)
	s, err := summarize([]byte(raw), "diagnostics", "model")
	if err != nil || s.ModelCalls != 2 || s.Rejected != 1 || s.SelectionPassed != 2 || s.SelectionTotal != 3 {
		t.Fatal("helper and root observations were not both counted", s, err)
	}
}

func TestNativePackageSnapshotDiagnostics(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for actual Gooo/native verification")
	}
	for _, model := range []string{"", "builtin"} {
		t.Run("model="+model, func(t *testing.T) {
			s, err := ReadSnapshot([]byte(packagePartialSnapshot))
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "diagnose")
			raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Model: model, Out: out}, s)
			if err != nil {
				t.Fatal(err)
			}
			var diagnosis struct{ Code, Message, Action string }
			if err = json.Unmarshal(raw, &diagnosis); err != nil || diagnosis.Code != "partial" || diagnosis.Action != "repair-and-replay" || diagnosis.Message != s.Detail {
				t.Fatal("Gooo did not diagnose the package observation", string(raw), err)
			}
			generated, err := os.ReadFile(filepath.Join(out, "diagnose", "result.json"))
			if err != nil {
				t.Fatal(err)
			}
			counts, err := summarize(generated, "diagnostics", model)
			wantCalls := 0
			if model != "" {
				wantCalls = 1
			}
			if err != nil || counts.ModelCalls != wantCalls || counts.SelectionPassed != 15 || counts.SelectionTotal != 15 {
				t.Fatal("diagnostic generation lost model or finite selection observations", counts, err)
			}
		})
	}
}
