package workbench

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGraphInspectionAndExecutionRequireExplicitEvidence(t *testing.T) {
	sha := "sha256:" + strings.Repeat("a", 64)
	plan := `{"schema":"gooo/body-composition-plan/v1","entry_activity":"A","typed_plan_sha256":"` + sha + `","semantic_fingerprint":"` + strings.Repeat("b", 64) + `","activities":[{"name":"A","id":"a"}]}`
	inspection := `{"schema":"gooo/body-composition-inspection/v1","source_sha256":"` + sha + `","plan":` + plan + `,"model_calls":0,"candidate_tests":0,"native_executions":0,"assemblies":[{"name":"A","id":"a","phase":"activity","kind":"record_choices","contract_sha256":"` + sha + `"}]}`
	p, err := readGraphInspection([]byte(inspection), sha, "A")
	if err != nil {
		t.Fatal(err)
	}
	for _, changed := range []string{
		strings.Replace(inspection, `"model_calls":0`, `"unused":0`, 1),
		strings.Replace(inspection, `"candidate_tests":0`, `"candidate_tests":1`, 1),
		strings.Replace(inspection, `"record_choices"`, `"unknown"`, 1),
		strings.Replace(inspection, `"entry_activity":"A"`, `"entry_activity":"B"`, 1),
		strings.Replace(inspection, `"activities":[`, `"activities":[{"name":"A","id":"a"},`, 1),
	} {
		if _, err := readGraphInspection([]byte(changed), sha, "A"); err == nil {
			t.Fatal("incomplete inspection accepted", changed)
		}
	}
	receipt := `{"schema":"gooo/record-field-assembly/v1","original_source_sha256":"` + sha + `","contract_sha256":"` + sha + `","model_calls":0,"passed":0,"total":1,"fields_passed":0,"fields_total":3}`
	raw := `{"generated_now":true,"composition":{"schema":"gooo/body-composition/v1","stage":"COMPLETE","original_source_sha256":"` + sha + `","generated_sha256":"` + sha + `","plan":` + plan + `,"steps":[{"input_source_sha256":"` + sha + `","generation":{"report":{"activity":"A","activity_id":"a","decision":"PASS","record_assembly":` + receipt + `}}}]},"runtime":{"stage":"COMPLETE","model_calls":0,"finite_passed":0,"finite_total":1,"projection_replayed":true,"runtime_replayed":true}}`
	profile := graphAssemblyProfiles{Schema: "gooo/graph-assembly-profiles/v1"}
	_, bodies, err := readGraphExecution([]byte(raw), p, profile, true)
	if err != nil || len(bodies) != 1 || bodies[0].SourceTotal != 1 {
		t.Fatal(bodies, err)
	}
	for _, changed := range []string{
		strings.Replace(raw, `"model_calls":0`, `"model_calls":1`, 1),
		strings.Replace(raw, `"fields_total":3`, `"unused":3`, 1),
		strings.Replace(raw, `"activity_id":"a"`, `"activity_id":"b"`, 1),
		strings.Replace(raw, `"runtime_replayed":true`, `"runtime_replayed":false`, 1),
		strings.Replace(raw, `"decision":"PASS"`, `"decision":"FAIL"`, 1),
		strings.Replace(raw, `"record_assembly":`, `"body_fill":{},"record_assembly":`, 1),
	} {
		if _, _, err := readGraphExecution([]byte(changed), p, profile, true); err == nil {
			t.Fatal("incomplete native evidence accepted", changed)
		}
	}
}

func TestGraphReceiptUsesCheckpointSpecificModelContext(t *testing.T) {
	sha := "sha256:" + strings.Repeat("a", 64)
	text := "different checkpoint model input"
	receipt := map[string]any{"schema": "gooo/record-field-assembly/v1", "original_source_sha256": sha, "contract_sha256": sha,
		"model_calls": 1, "passed": 1, "total": 1, "fields_passed": 3, "fields_total": 3,
		"model":         map[string]any{"schema": "gooo/retained-path-model/v1", "loaded": true, "metadata_sha256": strings.Repeat("b", 64), "weights_sha256": strings.Repeat("c", 64)},
		"model_context": map[string]any{"text": text, "sha256": "sha256:" + jointDigest([]byte(text))}}
	profile := graphAssemblyProfiles{Schema: "gooo/graph-assembly-profiles/v1", RecordRequested: true, MetadataSHA: strings.Repeat("b", 64)}
	raw, _ := json.Marshal(receipt)
	body, err := graphReceiptObservation(raw, profile, GraphAssemblyBody{ActivityID: "a", Kind: "record_choices", Checkpoint: sha}, sha)
	if err != nil || body.ModelCalls != 1 || body.ModelContextSHA == "" {
		t.Fatal(body, err)
	}
	receipt["model_context"] = map[string]any{"text": text, "sha256": sha}
	raw, _ = json.Marshal(receipt)
	if _, err := graphReceiptObservation(raw, profile, GraphAssemblyBody{ActivityID: "a", Kind: "record_choices", Checkpoint: sha}, sha); err == nil {
		t.Fatal("unbound context accepted")
	}
}
