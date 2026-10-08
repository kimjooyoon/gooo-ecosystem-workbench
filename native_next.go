package workbench

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
)

func diagnoseNative(ctx context.Context, o Options, root string, s Snapshot) (json.RawMessage, error) {
	if err := save(filepath.Join(root, "observation.json"), s); err != nil {
		return nil, err
	}
	cases, err := json.Marshal(map[string]any{"schema": "gooo/body-composition-cases/v1", "cases": []any{
		map[string]any{"inputs": map[string]any{"ObservationEcho": s.NativeOutcomes}, "expected": map[string]any{"ObservationEcho": s.NativeOutcomes}},
	}})
	if err != nil {
		return nil, err
	}
	_, r, err := runRecipe(ctx, o, root, "native-next", "native-next", "", cases)
	if err != nil {
		return nil, err
	}
	if r.Runtime.Calls != 0 || r.Runtime.Passed != 1 || r.Runtime.Total != 1 {
		return nil, fmt.Errorf("native diagnostic lost its observation or predicted during execution")
	}
	var next map[string]json.RawMessage
	if err = actualFor(r, "nativenext://activity/next", &next); err != nil {
		return nil, err
	}
	for _, field := range []string{"code", "action", "message"} {
		var value string
		if json.Unmarshal(next[field], &value) != nil || value == "" {
			return nil, fmt.Errorf("native diagnostic omitted its typed recommendation")
		}
	}
	for key, value := range map[string]any{"native_outcomes": s.NativeOutcomes, "input_sha256": s.InputSHA,
		"evaluation":      map[string]any{"unit": s.Unit, "passed": s.Passed, "total": s.Total},
		"new_model_calls": 0, "model_requested": o.Model != "", "model_used": false,
		"scope": "deterministic Gooo advice from recounted receipt values; diagnose does not rerun the original program or independently verify its source; no proposed action is executed"} {
		next[key], err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	return raw, write(filepath.Join(root, "diagnostic.json"), append(raw, '\n'))
}
