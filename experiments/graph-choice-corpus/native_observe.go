package main

import (
	"encoding/json"
	"fmt"
)

type nativeExport struct {
	Generated   *bool `json:"generated_now"`
	Composition struct {
		Steps []struct{ Generation finiteExport }
	}
	Runtime struct {
		SourceSHA  string `json:"original_source_sha256"`
		Compiler   string `json:"producer_source_sha"`
		Passed     int    `json:"finite_passed"`
		Total      int    `json:"finite_total"`
		Calls      *int   `json:"model_calls"`
		Projection bool   `json:"projection_replayed"`
		Replay     bool   `json:"runtime_replayed"`
		Traces     []struct {
			Index      int `json:"case_index"`
			Deliveries []struct {
				Actual json.RawMessage
				Input  json.RawMessage
				Inputs []struct {
					Port  string
					Value json.RawMessage
				}
			}
		}
	}
}

func checkNative(raw, source []byte, family familySpec, requested uint16, compiler string, generated bool) error {
	var result nativeExport
	if err := json.Unmarshal(raw, &result); err != nil {
		return err
	}
	r := result.Runtime
	inputs := nativeInputs[family.name]
	if result.Generated == nil || *result.Generated != generated || r.SourceSHA != digest(source) || r.Compiler != compiler ||
		r.Calls == nil || *r.Calls != 0 || !r.Projection || !r.Replay || r.Passed != len(inputs) || r.Total != len(inputs) || len(r.Traces) != len(inputs) {
		return fmt.Errorf("native identities, counts or replay differ")
	}
	if len(result.Composition.Steps) != 1 {
		return fmt.Errorf("one constructed entry required")
	}
	a := result.Composition.Steps[0].Generation.Report.Assembly
	if a.Mask == nil || *a.Mask != requested || a.Calls == nil || *a.Calls != 0 || a.SourceSHA != digest(source) {
		return fmt.Errorf("native construction choice differs or used inference")
	}
	for i, trace := range r.Traces {
		if trace.Index != i || len(trace.Deliveries) != 1 {
			return fmt.Errorf("one observed entry per input required")
		}
		delivery := trace.Deliveries[0]
		values := make([]json.RawMessage, len(delivery.Inputs))
		for j, input := range delivery.Inputs {
			if input.Port != fmt.Sprintf("input%d", j) {
				return fmt.Errorf("native input port order differs")
			}
			values[j] = input.Value
		}
		if family.name == "filenames" {
			if len(delivery.Inputs) != 0 || len(delivery.Input) == 0 {
				return fmt.Errorf("one scalar filename required")
			}
			values = []json.RawMessage{delivery.Input}
		} else if len(delivery.Input) != 0 {
			return fmt.Errorf("unexpected scalar input")
		}
		actualInput, err := json.Marshal(values)
		if err != nil || !sameJSON(actualInput, []byte(inputs[i])) {
			return fmt.Errorf("actual native input differs")
		}
		want, err := oracle(family.name, actualInput, requested)
		if err != nil || !sameJSON(delivery.Actual, want) {
			return fmt.Errorf("actual native output differs from independent oracle")
		}
	}
	return nil
}
