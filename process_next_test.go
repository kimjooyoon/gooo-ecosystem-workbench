package workbench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNativeFailedProcessPolicyAndSavedModelReplay(t *testing.T) {
	compiler := os.Getenv("GOOO_COMPILER")
	if compiler == "" {
		t.Skip("set GOOO_COMPILER for native Gooo process advice")
	}
	tests := []struct {
		code   string
		change func(map[string]any)
	}{
		{"deadline-during-start", func(p map[string]any) {}},
		{"deadline-during-start", func(p map[string]any) { p["timing"].(map[string]any)["wait_limit_ns"] = json.Number("2000000000") }},
		{"process-wait-timeout", func(p map[string]any) {
			p["timing"].(map[string]any)["context_at_start_return"] = "ACTIVE"
			p["timing"].(map[string]any)["wait_limit_ns"] = json.Number("2000000000")
		}},
		{"process-timeout", func(p map[string]any) { p["timing"].(map[string]any)["context_at_start_return"] = "ACTIVE" }},
		{"process-timeout", func(p map[string]any) { delete(p, "timing") }},
		{"process-canceled", func(p map[string]any) { p["timed_out"], p["canceled"] = false, true; delete(p, "timing") }},
		{"process-start-failed", func(p map[string]any) {
			p["started"], p["timed_out"], p["exit_code"] = false, false, nil
			delete(p, "timing")
		}},
		{"process-output-truncated", func(p map[string]any) { p["timed_out"], p["output_truncated"] = false, true; delete(p, "timing") }},
		{"process-completed", func(p map[string]any) {
			p["timed_out"], p["completed"], p["exit_code"] = false, true, json.Number("0")
			delete(p, "timing")
		}},
		{"process-exit-failed", func(p map[string]any) { p["timed_out"] = false; delete(p, "timing") }},
		{"process-incomplete", func(p map[string]any) { p["timed_out"], p["exit_code"] = false, nil; delete(p, "timing") }},
	}
	s := Snapshot{Unit: "native_process_runs", InputSHA: "synthetic-policy-cases"}
	for _, test := range tests {
		value, err := decodeValue(failedProcessFixture(t))
		if err != nil {
			t.Fatal(err)
		}
		r := value.(map[string]any)
		p := r["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)["runs"].([]any)[0].(map[string]any)
		test.change(p)
		raw, _ := json.Marshal(r)
		one, err := ReadSnapshot(raw)
		if err != nil {
			t.Fatal(err)
		}
		s.Processes = append(s.Processes, one.Processes...)
	}
	var first []processAdvice
	for _, model := range []string{"", "models/graph-chooser-20261008/all-data-demonstration/qat_ternary/model.json"} {
		out := filepath.Join(t.TempDir(), "diagnosis")
		raw, err := Diagnose(context.Background(), Options{Compiler: compiler, Model: model, Out: out}, s)
		if err != nil {
			t.Fatal(err)
		}
		var report struct {
			Processes   []struct{ Advice processAdvice }
			Observation Summary
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatal(err)
		}
		if len(report.Processes) != len(tests) || report.Observation.NamedTotal != 0 || report.Observation.NamedPassed != 0 ||
			!report.Observation.ReplayVerified || report.Observation.SelectionPassed != 6 || report.Observation.SelectionTotal != 6 {
			t.Fatal("process policy gained an evaluation score or lost assembly evidence", string(raw))
		}
		wantCalls := 0
		if model != "" {
			wantCalls = 1
		}
		if report.Observation.ModelCalls != wantCalls {
			t.Fatal("model call count differs", report.Observation)
		}
		values := make([]processAdvice, len(tests))
		for i, row := range report.Processes {
			if row.Advice.Code != tests[i].code {
				t.Fatal(i, row.Advice, tests[i].code)
			}
			values[i] = row.Advice
		}
		contextRaw, err := os.ReadFile(filepath.Join(out, "next-context.json"))
		if err != nil {
			t.Fatal(err)
		}
		var next processContext
		if err := json.Unmarshal(contextRaw, &next); err != nil {
			t.Fatal(err)
		}
		if len(next.Processes) != len(values) || next.Observation != report.Observation || !nativeDigest(next.GeneratedSHA) {
			t.Fatal("next context differs from Gooo execution", string(contextRaw))
		}
		for i, item := range next.Processes {
			if item.Advice != values[i] || item.Input != s.Processes[i].Input {
				t.Fatal("next context changed Gooo advice or original typed state", i)
			}
		}
		if first == nil {
			first = values
		} else if !reflect.DeepEqual(first, values) {
			t.Fatal("fixed/model advice differs")
		}
	}
}
