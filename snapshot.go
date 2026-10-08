package workbench

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
)

// ReadSnapshot recounts native composition, joint construction or package execution.
// Classification and the proposed next action remain in the Gooo diagnostics recipe.
func ReadSnapshot(raw []byte) (Snapshot, error) {
	inputSHA := fmt.Sprintf("%x", sha256.Sum256(raw))
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		return Snapshot{}, err
	}
	var schema string
	if json.Unmarshal(keys["schema"], &schema) == nil && schema == packageJointSchema {
		return readPackageJointSnapshot(raw, inputSHA)
	}
	if keys["evaluation"] != nil {
		return readJointSnapshot(raw, inputSHA)
	}
	if keys["result"] != nil || keys["schema"] != nil && keys["runtime"] == nil {
		var envelope struct {
			Schema   string          `json:"schema"`
			Decision string          `json:"decision"`
			Error    json.RawMessage `json:"error"`
			Result   json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return Snapshot{}, err
		}
		if envelope.Schema != "gooo/workspace-body-execution-receipt/v1" ||
			!slices.Contains([]string{"PASS", "PROGRESS", "OBSERVED"}, envelope.Decision) || present(envelope.Error) {
			return Snapshot{}, fmt.Errorf("provide a successful package execution receipt")
		}
		keys = nil
		if err := json.Unmarshal(envelope.Result, &keys); err != nil {
			return Snapshot{}, err
		}
		var schema string
		if err := json.Unmarshal(keys["schema"], &schema); err != nil || schema != "gooo/workspace-body-execution/v1" || !present(keys["runtime"]) {
			return Snapshot{}, fmt.Errorf("package execution result requires its v1 schema and native runtime observations")
		}
		raw = envelope.Result
	}
	if keys["runtime"] == nil {
		if keys["joint_construction"] != nil {
			return Snapshot{}, fmt.Errorf("joint diagnosis requires the original body-construct output")
		}
		if !present(keys["passed"]) || !present(keys["total"]) {
			return Snapshot{}, fmt.Errorf("provide a body-compose result, package execution receipt, or passed/total counters")
		}
		var s Snapshot
		err := json.Unmarshal(raw, &s)
		s.Unit, s.InputSHA = "provided_counts", inputSHA
		return s, err
	}
	var runtime map[string]json.RawMessage
	if err := json.Unmarshal(keys["runtime"], &runtime); err != nil {
		return Snapshot{}, err
	}
	for _, field := range []string{"finite_passed", "finite_total", "model_calls", "traces"} {
		if !present(runtime[field]) {
			return Snapshot{}, fmt.Errorf("native runtime observation requires %s", field)
		}
	}
	if observed, handled, err := readNativeOutcomes(keys["runtime"], inputSHA); err != nil || handled {
		if err == nil {
			var original result
			if err = json.Unmarshal(raw, &original); err == nil {
				observed.Construction = constructionObservations(original)
				counts, countErr := summarize(raw, "observation", "captured")
				err = countErr
				observed.Rejected = int64(counts.Rejected)
			}
		}
		return observed, err
	}
	counts, err := summarize(raw, "observation", "captured")
	if err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{Unit: "activity_outputs", Passed: int64(counts.NamedPassed), Total: int64(counts.NamedTotal), Rejected: int64(counts.Rejected), InputSHA: inputSHA}
	var r result
	if err = json.Unmarshal(raw, &r); err != nil {
		return s, err
	}
	s.Construction = constructionObservations(r)
	if counts.FieldsTotal > 0 && recordFieldUnit(r) {
		s.Unit, s.Passed, s.Total = "record_fields", int64(counts.FieldsPassed), int64(counts.FieldsTotal)
	}
	var gaps []map[string]any
	var rejected []string
	for _, trace := range r.Runtime.Traces {
		for _, value := range trace.Deliveries {
			if len(value.Expected) == 0 {
				continue
			}
			actual, _ := decodeValue(value.Actual) // summarize has checked both values.
			expected, _ := decodeValue(value.Expected)
			if s.Unit == "record_fields" {
				fields := expected.(map[string]any)
				a := actual.(map[string]any)
				names := make([]string, 0, len(fields))
				for field := range fields {
					names = append(names, field)
				}
				slices.Sort(names)
				for _, field := range names {
					want := fields[field]
					got, exists := a[field]
					if !exists || !reflect.DeepEqual(got, want) {
						gaps = append(gaps, map[string]any{"activity": value.ID, "field": field, "actual": got, "expected": want})
					}
				}
			} else if !reflect.DeepEqual(actual, expected) {
				gaps = append(gaps, map[string]any{"activity": value.ID, "actual": actual, "expected": expected})
			}
		}
	}
	for _, steps := range [][]constructionStep{r.Composition.Preparations, r.Composition.Steps} {
		for _, step := range steps {
			if search := step.Generation.Report.Search; search != nil {
				for _, attempt := range search.Attempts {
					if !attempt.TypecheckPassed {
						rejected = append(rejected, attempt.Error)
					}
				}
			}
			if a := step.Generation.Report.Assembly; a != nil {
				for _, trial := range a.Attempts {
					if trial.Status == "TYPECHECK_FAILED" {
						rejected = append(rejected, trial.Reason)
					}
				}
			}
		}
	}
	detail, err := json.Marshal(map[string]any{"unit": s.Unit, "gaps": gaps, "type_rejections": rejected})
	if len(detail) > 1024 {
		detail, err = json.Marshal(map[string]any{"unit": s.Unit, "detail_limited": true, "gaps": len(gaps), "type_rejections": len(rejected), "input_sha256": s.InputSHA})
	}
	s.Detail = string(detail)
	return s, err
}

func present(raw json.RawMessage) bool {
	return len(raw) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// Field counts apply only when every expected output is a nonempty record and
// no extra actual field would disappear from that denominator. Otherwise use
// whole activity outputs, keeping scalar, empty-record and shape failures visible.
func recordFieldUnit(r result) bool {
	for _, trace := range r.Runtime.Traces {
		for _, value := range trace.Deliveries {
			if len(value.Expected) == 0 {
				continue
			}
			expected, _ := decodeValue(value.Expected)
			actual, _ := decodeValue(value.Actual)
			fields, expectedRecord := expected.(map[string]any)
			a, actualRecord := actual.(map[string]any)
			if !expectedRecord || len(fields) == 0 || !actualRecord {
				return false
			}
			for name := range a {
				if _, exists := fields[name]; !exists {
					return false
				}
			}
		}
	}
	return true
}
