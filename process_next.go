package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
)

type policyAdvice struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Action  string `json:"action"`
}

func diagnoseProcesses(ctx context.Context, o Options, root string, s Snapshot) (json.RawMessage, error) {
	if s.Unit != "native_process_runs" || s.Passed != 0 || s.Total != 0 || s.Joint != nil || s.NativeOutcomes != nil {
		return nil, fmt.Errorf("process diagnosis requires unscored process observations")
	}
	for _, record := range s.Processes {
		input, err := readProcessInput(record.Raw)
		if err != nil || input != record.Input {
			return nil, fmt.Errorf("process policy input differs from its original record: %v", err)
		}
	}
	if len(s.Processes) > 128 {
		return nil, fmt.Errorf("process diagnosis supports at most 128 reported runs")
	}
	model, err := prepareModel(o.Model, root)
	if err != nil {
		return nil, err
	}
	if err = save(filepath.Join(root, "observation.json"), s); err != nil {
		return nil, err
	}
	rows := make([]any, len(s.Processes))
	for i, p := range s.Processes {
		rows[i] = map[string]any{"processnext:Next": p.Input}
	}
	if err = save(filepath.Join(root, "inputs.json"), map[string]any{"schema": "gooo/body-composition-inputs/v1", "inputs": rows}); err != nil {
		return nil, err
	}
	source, err := assets.ReadFile("recipes/process-next.gooo")
	if err != nil {
		return nil, err
	}
	if err = write(filepath.Join(root, "process-next.gooo"), source); err != nil {
		return nil, err
	}
	if err = save(filepath.Join(root, "gooo.workspace.json"), map[string]any{
		"schema": "gooo/package-workspace-manifest/v1", "entry": map[string]string{"package_path": "processnext", "activity": "Next"},
		"packages": []any{map[string]any{"path": "processnext", "sources": []string{"process-next.gooo"}}},
	}); err != nil {
		return nil, err
	}
	var original []policyAdvice
	var observations Summary
	var selected string
	for _, replay := range []bool{false, true} {
		raw, r, program, err := executeInputPolicy(ctx, o, root, model, replay)
		if err != nil {
			return nil, err
		}
		advice, err := policyAdviceValues(r, len(rows))
		if err != nil {
			return nil, err
		}
		if !replay {
			original, selected = advice, program
			observations, err = summarize(raw, "process-next", map[bool]string{true: "model", false: "fixed"}[model != ""])
			if err != nil {
				return nil, err
			}
		} else if !reflect.DeepEqual(original, advice) || selected != program {
			return nil, fmt.Errorf("saved process policy changed the selected program or advice")
		}
	}
	observations.ReplayVerified = true
	items := make([]any, len(original))
	for i, advice := range original {
		items[i] = map[string]any{"observation": s.Processes[i], "advice": advice}
	}
	report := map[string]any{"schema": "gooo/ecosystem-process-diagnosis/v1", "decision": "OBSERVED",
		"input_sha256": s.InputSHA, "processes": items, "observation": observations,
		"next_context": "next-context.json",
		"scope":        "Gooo advice from reported failed native runtimes; original program correctness and reported source/executable identities are not independently verified; process record appearances are not independent execution counts; no proposed action is executed"}
	raw, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	if err = write(filepath.Join(root, "diagnostic.json"), append(raw, '\n')); err != nil {
		return nil, err
	}
	return raw, saveProcessContext(root, s, original, observations, selected)
}

func executeInputPolicy(ctx context.Context, o Options, root, model string, replay bool) ([]byte, result, string, error) {
	var r result
	args := []string{"package", "execute", "--json", "--inputs", filepath.Join(root, "inputs.json")}
	filename := "execution.json"
	if replay {
		args[1] = "replay"
		args = append(args, "--receipt", filepath.Join(root, "execution.json"))
		filename = "replay.json"
	} else if model != "" {
		args = append(args, "--assembly-model", model)
	}
	raw, err := command(ctx, o.Compiler, append(args, filepath.Join(root, "gooo.workspace.json"))...)
	if len(raw) > 0 {
		if e := write(filepath.Join(root, filename), raw); e != nil {
			return nil, r, "", e
		}
	}
	if err != nil {
		return nil, r, "", err
	}
	var envelope struct {
		Schema, Decision, Error string
		Result                  json.RawMessage
		From                    string `json:"replayed_from_sha256"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, r, "", err
	}
	if envelope.Schema != "gooo/workspace-body-execution-receipt/v1" || envelope.Decision != "OBSERVED" || envelope.Error != "" {
		return nil, r, "", fmt.Errorf("input policy requires input-only Gooo execution")
	}
	if err = json.Unmarshal(envelope.Result, &r); err != nil {
		return nil, r, "", err
	}
	var identity struct {
		Composition struct {
			SHA string `json:"generated_sha256"`
		}
		Runtime struct {
			Calls      *int `json:"model_calls"`
			Passed     *int `json:"finite_passed"`
			Total      *int `json:"finite_total"`
			Stage      string
			Projection bool `json:"projection_replayed"`
			Replay     bool `json:"runtime_replayed"`
		}
		Replay *struct {
			Calls *int `json:"model_calls"`
		}
	}
	if err = json.Unmarshal(envelope.Result, &identity); err != nil {
		return nil, r, "", err
	}
	x := identity.Runtime
	if x.Calls == nil || *x.Calls != 0 || x.Passed == nil || *x.Passed != 0 || x.Total == nil || *x.Total != 0 ||
		x.Stage != "COMPLETE" || !x.Projection || !x.Replay || !nativeDigest(identity.Composition.SHA) ||
		replay && (r.Generated || envelope.From == "" || identity.Replay == nil || identity.Replay.Calls == nil || *identity.Replay.Calls != 0) {
		return nil, r, "", fmt.Errorf("input policy acquired a score, inference during execution or unbound replay")
	}
	return envelope.Result, r, identity.Composition.SHA, nil
}

func policyAdviceValues(r result, count int) ([]policyAdvice, error) {
	if len(r.Composition.Steps) != 1 || len(r.Runtime.Traces) != count {
		return nil, fmt.Errorf("input policy output count differs")
	}
	id := r.Composition.Steps[0].Generation.Report.ActivityID
	values := make([]policyAdvice, count)
	for i, trace := range r.Runtime.Traces {
		if trace.CaseIndex != i || len(trace.Deliveries) != 1 || id == "" || trace.Deliveries[0].ID != id || len(trace.Deliveries[0].Expected) != 0 {
			return nil, fmt.Errorf("input policy output identity, order or observation scope differs")
		}
		if err := json.Unmarshal(trace.Deliveries[0].Actual, &values[i]); err != nil {
			return nil, err
		}
		if values[i].Code == "" || values[i].Message == "" || values[i].Action == "" {
			return nil, fmt.Errorf("input policy advice is incomplete")
		}
	}
	return values, nil
}
