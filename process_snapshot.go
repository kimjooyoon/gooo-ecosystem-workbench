package workbench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

// ProcessObservation retains a reported process, without scoring its program.
type ProcessObservation struct {
	RuntimeSchema  string          `json:"runtime_schema"`
	RuntimeStage   string          `json:"runtime_stage"`
	RuntimeFailure string          `json:"runtime_failure"`
	Location       string          `json:"location"`
	Source         string          `json:"reported_source_sha256"`
	Generated      string          `json:"reported_generated_sha256"`
	Executable     string          `json:"reported_executable_sha256"`
	Raw            json.RawMessage `json:"process"`
	Input          processInput    `json:"policy_input"`
}

type processInput struct {
	Started      bool   `json:"started"`
	Completed    bool   `json:"completed"`
	Canceled     bool   `json:"canceled"`
	TimedOut     bool   `json:"timed_out"`
	Truncated    bool   `json:"truncated"`
	HasExit      bool   `json:"has_exit"`
	Exit         int64  `json:"exit_code"`
	HasTiming    bool   `json:"has_timing"`
	StartContext string `json:"context_at_start_return"`
	HasWaitLimit bool   `json:"has_wait_limit"`
}

func readFailedProcesses(raw []byte, prefix string) ([]ProcessObservation, error) {
	var r struct {
		Schema, Decision string
		Result, Runtime  json.RawMessage
		Construction     json.RawMessage
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	if prefix == "" && (r.Schema == packageJointSchema || r.Schema == "gooo/workspace-body-execution-receipt/v1") {
		if r.Decision != "FAIL_CLOSED" || !present(r.Result) {
			return nil, nil
		}
		return readFailedProcesses(r.Result, "/result")
	}
	if r.Schema == "gooo/body-composition-runtime/v1" || r.Schema == "gooo/body-composition-runtime/v2" {
		return readFailedRuntime(raw, prefix)
	}
	if r.Schema != "" && r.Schema != "gooo/workspace-body-execution/v1" && r.Schema != "gooo/workspace-caller-construction/v1" {
		return nil, nil
	}
	if present(r.Runtime) {
		return readFailedRuntime(r.Runtime, prefix+"/runtime")
	}
	if !present(r.Construction) || !bytes.HasPrefix(bytes.TrimSpace(r.Construction), []byte("{")) {
		return nil, nil
	}
	var construction struct {
		Schema, Failure string
		Attempts        []struct{ Runtime json.RawMessage }
	}
	if err := json.Unmarshal(r.Construction, &construction); err != nil {
		return nil, err
	}
	if construction.Failure == "" || !slices.Contains([]string{"gooo/joint-construction/v1", "gooo/joint-construction/v2", "gooo/joint-construction/v3", "gooo/joint-construction/v4", "gooo/joint-construction/v5", "gooo/joint-construction/v6", "gooo/joint-construction/v7"}, construction.Schema) {
		return nil, nil
	}
	var records []ProcessObservation
	for i, attempt := range construction.Attempts {
		if !present(attempt.Runtime) {
			continue
		}
		found, err := readFailedRuntime(attempt.Runtime, fmt.Sprintf("%s/construction/attempts/%d/runtime", prefix, i))
		if err != nil {
			return nil, err
		}
		records = append(records, found...)
		if len(records) > 128 {
			return nil, fmt.Errorf("process diagnosis supports at most 128 reported runs")
		}
	}
	return records, nil
}

func readFailedRuntime(raw []byte, location string) ([]ProcessObservation, error) {
	var r struct {
		Schema, Stage, Failure string
		Source                 string `json:"original_source_sha256"`
		Generated              string `json:"generated_sha256"`
		Executable             string `json:"executable_sha256"`
		Runs                   []json.RawMessage
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	if !slices.Contains([]string{"gooo/body-composition-runtime/v1", "gooo/body-composition-runtime/v2"}, r.Schema) || r.Failure == "" || len(r.Runs) == 0 {
		return nil, nil
	}
	if r.Stage != "EXECUTE_1" && r.Stage != "EXECUTE_2" ||
		r.Stage == "EXECUTE_1" && len(r.Runs) != 1 || r.Stage == "EXECUTE_2" && len(r.Runs) != 2 {
		return nil, fmt.Errorf("failed native runtime stage and process count disagree")
	}
	if len(r.Runs) > 128 || !nativeDigest(r.Source) || !nativeDigest(r.Generated) || !nativeDigest(r.Executable) {
		return nil, fmt.Errorf("failed native runs require reported source, program and executable identities")
	}
	result := make([]ProcessObservation, len(r.Runs))
	for i, raw := range r.Runs {
		input, err := readProcessInput(raw)
		if err != nil {
			return nil, fmt.Errorf("%s/runs/%d: %w", location, i, err)
		}
		result[i] = ProcessObservation{RuntimeSchema: r.Schema, RuntimeStage: r.Stage, RuntimeFailure: r.Failure,
			Location: fmt.Sprintf("%s/runs/%d", location, i),
			Source:   r.Source, Generated: r.Generated, Executable: r.Executable, Raw: bytes.Clone(raw), Input: input}
	}
	return result, nil
}

func readProcessInput(raw []byte) (processInput, error) {
	var r struct {
		Started, Completed, Canceled *bool
		TimedOut                     *bool  `json:"timed_out"`
		Truncated                    *bool  `json:"output_truncated"`
		Exit                         *int64 `json:"exit_code"`
		Wall                         *int64 `json:"wall_ns"`
		Timing                       *struct {
			Start   *int64 `json:"start_ns"`
			Wait    *int64 `json:"wait_ns"`
			Context string `json:"context_at_start_return"`
			Limit   *int64 `json:"wait_limit_ns"`
		}
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return processInput{}, err
	}
	if r.Started == nil || r.Completed == nil || r.Canceled == nil || r.TimedOut == nil || r.Truncated == nil || r.Wall == nil || *r.Wall < 0 {
		return processInput{}, fmt.Errorf("reported process flags and nonnegative wall time are required")
	}
	p := processInput{Started: *r.Started, Completed: *r.Completed, Canceled: *r.Canceled, TimedOut: *r.TimedOut,
		Truncated: *r.Truncated, HasExit: r.Exit != nil, HasTiming: r.Timing != nil}
	if r.Exit != nil {
		p.Exit = *r.Exit
	}
	if p.Canceled && p.TimedOut || p.Completed && (!p.Started || p.Canceled || p.TimedOut || !p.HasExit || p.Exit != 0) {
		return processInput{}, fmt.Errorf("reported process completion and termination disagree")
	}
	if t := r.Timing; t != nil {
		if t.Start == nil || t.Wait == nil || *t.Start < 0 || *t.Wait < 0 || *t.Start > *r.Wall ||
			*t.Wait != *r.Wall-*t.Start || !slices.Contains([]string{"ACTIVE", "CANCELED", "DEADLINE_EXCEEDED"}, t.Context) || t.Limit != nil && *t.Limit < 0 {
			return processInput{}, fmt.Errorf("reported process timing is incomplete or inconsistent")
		}
		p.StartContext, p.HasWaitLimit = t.Context, t.Limit != nil
	}
	return p, nil
}
