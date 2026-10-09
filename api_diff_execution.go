package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
)

func executeAPIBatch(ctx context.Context, o Options, root, model string, changes []APIChange, index int, replay bool) ([]byte, result, error) {
	var observed result
	manifest := filepath.Join(root, "gooo.workspace.json")
	inputs := filepath.Join(root, fmt.Sprintf("inputs-%03d.json", index))
	if !replay {
		source, err := assets.ReadFile("recipes/api-changes.gooo")
		if err != nil {
			return nil, observed, err
		}
		if err = write(filepath.Join(root, "api-changes.gooo"), source); err != nil {
			return nil, observed, err
		}
		workspace := map[string]any{"schema": "gooo/package-workspace-manifest/v1", "entry": map[string]string{"package_path": "apichanges", "activity": "Assess"},
			"packages": []any{map[string]any{"path": "apichanges", "sources": []string{"api-changes.gooo"}}}}
		if err = save(manifest, workspace); err != nil {
			return nil, observed, err
		}
	}
	rows := make([]any, len(changes))
	for i, change := range changes {
		rows[i] = map[string]any{"apichanges:Assess": map[string]any{"kind": change.Kind,
			"input_used": change.InputUsed, "output_used": change.OutputUsed,
			"before_required": change.BeforeRequired, "after_required": change.AfterRequired}}
	}
	if err := save(inputs, map[string]any{"schema": "gooo/body-composition-inputs/v1", "inputs": rows}); err != nil {
		return nil, observed, err
	}
	args := []string{"package", "execute", "--json", "--inputs", inputs}
	filename := "execution.json"
	if replay {
		args[1] = "replay"
		args = append(args, "--receipt", filepath.Join(root, "execution.json"))
		filename = fmt.Sprintf("replay-%03d.json", index)
	} else if model != "" {
		args = append(args, "--assembly-model", model)
	}
	args = append(args, manifest)
	raw, err := command(ctx, o.Compiler, args...)
	if len(raw) > 0 {
		if e := write(filepath.Join(root, filename), raw); e != nil {
			return nil, observed, e
		}
	}
	if err != nil {
		return nil, observed, err
	}
	var envelope struct {
		Schema, Decision, Error string
		Result                  json.RawMessage
		ReplayedFrom            string `json:"replayed_from_sha256"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, observed, err
	}
	if envelope.Schema != "gooo/workspace-body-execution-receipt/v1" || envelope.Decision != "OBSERVED" || envelope.Error != "" {
		return nil, observed, fmt.Errorf("input-only Gooo API policy execution required")
	}
	if err = json.Unmarshal(envelope.Result, &observed); err != nil {
		return nil, observed, err
	}
	if observed.Runtime.Calls != 0 || observed.Runtime.Total != 0 || observed.Runtime.Passed != 0 {
		return nil, observed, fmt.Errorf("actual API changes acquired a score or runtime inference")
	}
	if replay {
		var receipt struct {
			Replay *struct {
				Calls *int `json:"model_calls"`
			}
		}
		if err = json.Unmarshal(envelope.Result, &receipt); err != nil {
			return nil, observed, err
		}
		if observed.Generated || envelope.ReplayedFrom == "" || receipt.Replay == nil || receipt.Replay.Calls == nil || *receipt.Replay.Calls != 0 {
			return nil, observed, fmt.Errorf("zero-inference saved API policy required")
		}
	}
	return envelope.Result, observed, nil
}

func apiAssessments(observed result, count int) ([]APIAssessment, error) {
	if len(observed.Composition.Steps) != 1 || len(observed.Runtime.Traces) != count {
		return nil, fmt.Errorf("API policy output count differs from input facts")
	}
	id := observed.Composition.Steps[0].Generation.Report.ActivityID
	values := make([]APIAssessment, count)
	for i, trace := range observed.Runtime.Traces {
		if trace.CaseIndex != i || len(trace.Deliveries) != 1 || id == "" || trace.Deliveries[0].ID != id || len(trace.Deliveries[0].Expected) != 0 {
			return nil, fmt.Errorf("API policy output identity/order differs or acquired an oracle")
		}
		if err := json.Unmarshal(trace.Deliveries[0].Actual, &values[i]); err != nil {
			return nil, err
		}
		if values[i].Code == "" || values[i].Action == "" || values[i].Priority < 0 || values[i].Priority > 2 {
			return nil, fmt.Errorf("incomplete API policy assessment")
		}
	}
	return values, nil
}
