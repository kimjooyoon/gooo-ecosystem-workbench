package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type graphIdentity struct{ Name, ID string }
type graphInspection struct {
	Schema     string
	SourceSHA  string `json:"source_sha256"`
	Plan       json.RawMessage
	Assemblies []struct {
		Name, ID, Phase, Kind string
		ContractSHA           string `json:"contract_sha256"`
	} `json:"assemblies"`
	ModelCalls *int `json:"model_calls"`
	Tests      *int `json:"candidate_tests"`
	Executions *int `json:"native_executions"`
}

type graphPlan struct {
	Schema       string
	Entry        string `json:"entry_activity"`
	TypedSHA     string `json:"typed_plan_sha256"`
	Semantic     string `json:"semantic_fingerprint"`
	Activities   []graphIdentity
	Preparations []graphIdentity
}

type GraphAssemblyBody struct {
	ActivityID      string `json:"activity_id"`
	Phase           string `json:"phase"`
	Kind            string `json:"kind"`
	Checkpoint      string `json:"input_source_sha256"`
	SourcePassed    int    `json:"source_cases_passed"`
	SourceTotal     int    `json:"source_cases_total"`
	HoldoutPassed   int    `json:"source_holdout_passed"`
	HoldoutTotal    int    `json:"source_holdout_total"`
	ModelCalls      int    `json:"model_calls"`
	ModelContextSHA string `json:"model_context_sha256,omitempty"`
}

func readGraphInspection(raw []byte, sourceSHA, entry string) (p graphInspection, err error) {
	if err = json.Unmarshal(raw, &p); err != nil {
		return p, err
	}
	var plan graphPlan
	if err = json.Unmarshal(p.Plan, &plan); err != nil {
		return p, err
	}
	if p.Schema != "gooo/body-composition-inspection/v1" || p.SourceSHA != sourceSHA || !nativeDigest(sourceSHA) ||
		!zeroGraphCounter(p.ModelCalls) || !zeroGraphCounter(p.Tests) || !zeroGraphCounter(p.Executions) ||
		plan.Schema != "gooo/body-composition-plan/v1" || plan.Entry != entry || entry == "" ||
		!nativeDigest(plan.TypedSHA) || !nativeDigest("sha256:"+plan.Semantic) ||
		len(plan.Activities) < 1 || len(plan.Activities) > 16 || len(plan.Preparations) > 32 || len(p.Assemblies) > 48 {
		return p, fmt.Errorf("graph inspection requires the original source, entry, bounded native plan and explicit zero counters")
	}
	for _, identities := range [][]graphIdentity{plan.Activities, plan.Preparations} {
		names, ids := map[string]bool{}, map[string]bool{}
		for _, id := range identities {
			if id.ID == "" || id.Name == "" || names[id.Name] || ids[id.ID] {
				return p, fmt.Errorf("native graph plan has ambiguous activity identity")
			}
			names[id.Name], ids[id.ID] = true, true
		}
	}
	seen := map[string]bool{}
	for _, assembly := range p.Assemblies {
		if assembly.ID == "" || assembly.Name == "" || seen[assembly.ID] || !nativeDigest(assembly.ContractSHA) ||
			(assembly.Phase != "activity" && assembly.Phase != "called_body") ||
			(assembly.Kind != "record_choices" && assembly.Kind != "typed_paths" && assembly.Kind != "source_search" && assembly.Kind != "source_fill") {
			return p, fmt.Errorf("graph inspection has an ambiguous assembly inventory")
		}
		seen[assembly.ID] = true
	}
	return p, nil
}

func zeroGraphCounter(n *int) bool { return n != nil && *n == 0 }
func graphCounts(passed, total *int) bool {
	return passed != nil && total != nil && *passed >= 0 && *total >= *passed
}

type graphReceipt struct {
	Schema            string
	OriginalSHA       string `json:"original_source_sha256"`
	OriginalDigest    string `json:"original_source_digest"`
	ContractSHA       string `json:"contract_sha256"`
	Calls             *int   `json:"model_calls"`
	Passed            *int   `json:"passed"`
	Total             *int   `json:"total"`
	FieldsPassed      *int   `json:"fields_passed"`
	FieldsTotal       *int   `json:"fields_total"`
	TrainingPassed    *int   `json:"training_passed"`
	TrainingTotal     *int   `json:"training_total"`
	HoldoutPassed     *int   `json:"holdout_passed"`
	HoldoutTotal      *int   `json:"holdout_total"`
	FillPassed        *int   `json:"test_cases_passed"`
	FillTotal         *int   `json:"test_cases_total"`
	FillHoldoutPassed *int   `json:"holdout_cases_passed"`
	FillHoldoutTotal  *int   `json:"holdout_cases_total"`
	Predictions       *int   `json:"local_model_predictions"`
	External          *int   `json:"external_provider_calls"`
	ExternalKnown     *bool  `json:"external_provider_calls_known"`
	Operations        *int   `json:"provider_operations"`
	Model             *assemblyModel
	Retention         *assemblyModel  `json:"model_retention"`
	Context           json.RawMessage `json:"model_context"`
	Search            struct {
		Passed    *int `json:"selected_training_passed"`
		Total     *int `json:"training_cases"`
		Selection struct {
			Predictions *int `json:"local_model_predictions"`
		} `json:"selection"`
	} `json:"search"`
}

type graphExecutionStep struct {
	Checkpoint string `json:"input_source_sha256"`
	Generation struct {
		Report struct {
			Activity, Decision string
			ActivityID         string          `json:"activity_id"`
			Assembly           json.RawMessage `json:"record_assembly"`
			BodyPaths          json.RawMessage `json:"body_paths"`
			BodySearch         json.RawMessage `json:"body_search"`
			BodyFill           json.RawMessage `json:"body_fill"`
		}
	}
}

func readGraphExecution(raw []byte, p graphInspection, profiles graphAssemblyProfiles, generated bool) (r result, bodies []GraphAssemblyBody, err error) {
	if err = json.Unmarshal(raw, &r); err != nil {
		return r, nil, err
	}
	var e struct {
		Composition struct {
			Schema, Stage       string
			Plan                json.RawMessage
			Steps, Preparations []graphExecutionStep
		}
		Runtime struct {
			Stage      string
			Calls      *int `json:"model_calls"`
			Passed     *int `json:"finite_passed"`
			Total      *int `json:"finite_total"`
			Projection bool `json:"projection_replayed"`
			Replay     bool `json:"runtime_replayed"`
		}
	}
	if err = json.Unmarshal(raw, &e); err != nil {
		return r, nil, err
	}
	a, err := jointCanonical(p.Plan)
	if err != nil {
		return r, nil, err
	}
	b, err := jointCanonical(e.Composition.Plan)
	if err != nil {
		return r, nil, err
	}
	x := e.Runtime
	if profiles.Schema != "gooo/graph-assembly-profiles/v1" || (profiles.RecordRequested && !nativeDigest("sha256:"+profiles.MetadataSHA)) ||
		r.Generated != generated || e.Composition.Schema != "gooo/body-composition/v1" || e.Composition.Stage != "COMPLETE" ||
		r.Composition.OriginalSourceSHA != p.SourceSHA || !nativeDigest(r.Composition.GeneratedSHA) || !reflect.DeepEqual(a, b) ||
		x.Stage != "COMPLETE" || !zeroGraphCounter(x.Calls) || !graphCounts(x.Passed, x.Total) || !x.Projection || !x.Replay {
		return r, nil, fmt.Errorf("graph execution requires the inspected native plan and complete source-bound replay without runtime inference")
	}
	var plan graphPlan
	if err = json.Unmarshal(p.Plan, &plan); err != nil {
		return r, nil, err
	}
	if len(plan.Activities) != len(e.Composition.Steps) || len(plan.Preparations) != len(e.Composition.Preparations) {
		return r, nil, fmt.Errorf("graph step inventory differs")
	}
	bodies = []GraphAssemblyBody{}
	position := 0
	for phaseIndex, steps := range [][]graphExecutionStep{e.Composition.Preparations, e.Composition.Steps} {
		identities, phase := plan.Preparations, "called_body"
		if phaseIndex == 1 {
			identities, phase = plan.Activities, "activity"
		}
		for i, step := range steps {
			report := step.Generation.Report
			if report.ActivityID != identities[i].ID || report.Activity != identities[i].Name || report.Decision != "PASS" || !nativeDigest(step.Checkpoint) {
				return r, nil, fmt.Errorf("graph activity or checkpoint differs from native order")
			}
			receipts := []json.RawMessage{report.Assembly, report.BodyPaths, report.BodySearch, report.BodyFill}
			count, selected := 0, -1
			for j, receipt := range receipts {
				if present(receipt) {
					count++
					selected = j
				}
			}
			if count == 0 {
				continue
			}
			if count != 1 || position >= len(p.Assemblies) {
				return r, nil, fmt.Errorf("unexpected graph assembly receipt")
			}
			item := p.Assemblies[position]
			kind := []string{"record_choices", "typed_paths", "source_search", "source_fill"}[selected]
			if item.ID != report.ActivityID || item.Name != report.Activity || item.Phase != phase || item.Kind != kind {
				return r, nil, fmt.Errorf("graph assembly differs from inspected inventory")
			}
			body, e := graphReceiptObservation(receipts[selected], profiles, GraphAssemblyBody{ActivityID: item.ID, Phase: phase, Kind: kind, Checkpoint: step.Checkpoint}, item.ContractSHA)
			if e != nil {
				return r, nil, e
			}
			bodies = append(bodies, body)
			position++
		}
	}
	if position != len(p.Assemblies) {
		return r, nil, fmt.Errorf("graph omitted an inspected assembly")
	}
	return r, bodies, nil
}

func graphReceiptObservation(raw []byte, profiles graphAssemblyProfiles, body GraphAssemblyBody, contract string) (GraphAssemblyBody, error) {
	var q graphReceipt
	if err := json.Unmarshal(raw, &q); err != nil {
		return body, err
	}
	passed, total, calls := q.Passed, q.Total, q.Calls
	original := q.OriginalSHA
	switch body.Kind {
	case "record_choices":
		if q.Schema != "gooo/record-field-assembly/v1" || q.ContractSHA != contract || !graphCounts(q.FieldsPassed, q.FieldsTotal) {
			return body, fmt.Errorf("incomplete source-bound record selection")
		}
	case "typed_paths":
		passed, total, calls = q.Search.Passed, q.Search.Total, q.Search.Selection.Predictions
		q.Model = q.Retention
		if q.Schema != "gooo/body-codegen-typed-path-receipt/v1" {
			return body, fmt.Errorf("unknown typed path receipt")
		}
	case "source_search":
		passed, total, original = q.TrainingPassed, q.TrainingTotal, q.OriginalDigest
		calls = new(int)
		if q.Schema != "gooo/body-codegen-ir-search-plan/v1" || !zeroGraphCounter(q.Operations) {
			return body, fmt.Errorf("source search requires explicit zero provider operations")
		}
		if !graphCounts(q.HoldoutPassed, q.HoldoutTotal) {
			return body, fmt.Errorf("search omitted holdout counters")
		}
		body.HoldoutPassed, body.HoldoutTotal = *q.HoldoutPassed, *q.HoldoutTotal
	case "source_fill":
		passed, total, calls, original = q.FillPassed, q.FillTotal, q.Predictions, q.OriginalDigest
		knownSchema := q.Schema == "gooo/body-codegen-ir-fill-plan/v1" || q.Schema == "gooo/body-codegen-ir-fill-plan/v2" || q.Schema == "gooo/body-codegen-ir-fill-plan/v3-record"
		if !knownSchema || !zeroGraphCounter(calls) || !zeroGraphCounter(q.External) || q.ExternalKnown == nil || !*q.ExternalKnown || !graphCounts(q.FillHoldoutPassed, q.FillHoldoutTotal) {
			return body, fmt.Errorf("source fill requires deterministic calls and explicit selection/holdout counters")
		}
		body.HoldoutPassed, body.HoldoutTotal = *q.FillHoldoutPassed, *q.FillHoldoutTotal
	}
	if original != body.Checkpoint || !graphCounts(passed, total) || calls == nil || *calls < 0 || (!profiles.RecordRequested && *calls != 0) {
		return body, fmt.Errorf("graph body lost its checkpoint, source case counts or explicit model calls")
	}
	body.SourcePassed, body.SourceTotal, body.ModelCalls = *passed, *total, *calls
	if *calls > 0 {
		m := q.Model
		if m == nil || m.Loaded == nil || !*m.Loaded || m.Schema != "gooo/retained-path-model/v1" || m.MetadataSHA != profiles.MetadataSHA || !nativeDigest("sha256:"+m.WeightsSHA) {
			return body, fmt.Errorf("graph prediction has no bound requested model identity")
		}
		if body.Kind == "record_choices" {
			var context struct {
				Text string
				SHA  string `json:"sha256"`
			}
			if json.Unmarshal(q.Context, &context) != nil || context.Text == "" || context.SHA != "sha256:"+jointDigest([]byte(context.Text)) {
				return body, fmt.Errorf("graph prediction has no bound checkpoint-specific record context")
			}
			body.ModelContextSHA = context.SHA
		} else {
			var context struct {
				Status      string
				MetadataSHA string `json:"model_metadata_sha256"`
				ActivityID  string `json:"activity_id"`
			}
			if json.Unmarshal(q.Context, &context) != nil || context.Status != "ENCODED" || context.MetadataSHA != profiles.MetadataSHA || context.ActivityID != body.ActivityID {
				return body, fmt.Errorf("typed graph prediction has no source-bound model context")
			}
			canonical, err := jointCanonical(q.Context)
			if err != nil {
				return body, err
			}
			body.ModelContextSHA = "sha256:" + jointDigest(canonical)
		}
	}
	return body, nil
}
