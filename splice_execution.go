package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
)

func executeSpliceRequest(ctx context.Context, o Options, root, model string, request SpliceRequest, replay bool) ([]byte, result, error) {
	var observed result
	manifest, inputs := filepath.Join(root, "gooo.workspace.json"), filepath.Join(root, "inputs.json")
	if !replay {
		source, err := assets.ReadFile("recipes/splice.gooo")
		if err != nil {
			return nil, observed, err
		}
		if err = write(filepath.Join(root, "splice.gooo"), source); err != nil {
			return nil, observed, err
		}
		workspace := map[string]any{"schema": "gooo/package-workspace-manifest/v1", "entry": map[string]string{"package_path": "sourceedit", "activity": "Splice"}, "packages": []any{map[string]any{"path": "sourceedit", "name": "sourceedit", "imports": []string{}, "sources": []string{"splice.gooo"}}}}
		if err = save(manifest, workspace); err != nil {
			return nil, observed, err
		}
		values := map[string]any{"sourceedit:Splice.input0": request.Source, "sourceedit:Splice.input1": request.Before, "sourceedit:Splice.input2": request.Expected, "sourceedit:Splice.input3": request.After, "sourceedit:Splice.input4": request.Replacement}
		if err = save(inputs, map[string]any{"schema": "gooo/body-composition-inputs/v1", "inputs": []any{values}}); err != nil {
			return nil, observed, err
		}
	}
	args := []string{"package", "execute", "--json", "--inputs", inputs}
	name := "execution.json"
	if replay {
		args[1] = "replay"
		args = append(args, "--receipt", filepath.Join(root, "execution.json"))
		name = "replay.json"
	} else if model != "" {
		args = append(args, "--assembly-model", model)
	}
	args = append(args, manifest)
	raw, err := command(ctx, o.Compiler, args...)
	if len(raw) > 0 {
		if e := write(filepath.Join(root, name), raw); e != nil {
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
		return nil, observed, fmt.Errorf("input-only Gooo execution required")
	}
	if err = json.Unmarshal(envelope.Result, &observed); err != nil {
		return nil, observed, err
	}
	if observed.Runtime.Calls != 0 || observed.Runtime.Total != 0 || observed.Runtime.Passed != 0 {
		return nil, observed, fmt.Errorf("input-only execution acquired a score or inference")
	}
	if replay {
		var saved struct {
			Replay *struct {
				Calls *int `json:"model_calls"`
			}
		}
		if err = json.Unmarshal(envelope.Result, &saved); err != nil {
			return nil, observed, err
		}
		if envelope.ReplayedFrom == "" || saved.Replay == nil || saved.Replay.Calls == nil || *saved.Replay.Calls != 0 {
			return nil, observed, fmt.Errorf("zero-inference package replay required")
		}
	}
	return envelope.Result, observed, nil
}
