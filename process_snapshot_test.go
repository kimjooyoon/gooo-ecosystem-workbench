package workbench

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func failedProcessFixture(t *testing.T) []byte {
	t.Helper()
	f, err := os.Open("examples/native-process-failure/windows-start.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	raw, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestFailedProcessSnapshotPreservesOriginalUnscoredFailure(t *testing.T) {
	raw := failedProcessFixture(t)
	s, err := ReadSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Unit != "native_process_runs" || s.Passed != 0 || s.Total != 0 || s.Joint != nil || len(s.Processes) != 1 {
		t.Fatalf("failed process gained evaluation evidence: %+v", s)
	}
	p := s.Processes[0]
	if p.Location != "/construction/attempts/0/runtime/runs/0" || p.Input.HasWaitLimit ||
		!p.Input.TimedOut || p.Input.StartContext != "DEADLINE_EXCEEDED" ||
		p.Source == "" || p.Generated == "" || p.Executable == "" {
		t.Fatalf("original process identity or timing lost: %+v", p)
	}
	var run struct {
		Wall   int64 `json:"wall_ns"`
		Timing struct {
			Start int64 `json:"start_ns"`
			Wait  int64 `json:"wait_ns"`
		}
	}
	if err := json.Unmarshal(p.Raw, &run); err != nil {
		t.Fatal(err)
	}
	if run.Wall != 2855231800 || run.Timing.Start != 2854721800 || run.Timing.Wait != 510000 {
		t.Fatal("original process integers changed", run)
	}
}

func TestFailedProcessSnapshotRejectsChangedProcessEvidence(t *testing.T) {
	for name, edit := range map[string]func(map[string]any){
		"phase sum":       func(p map[string]any) { p["wall_ns"] = json.Number("1") },
		"missing flag":    func(p map[string]any) { delete(p, "started") },
		"contradiction":   func(p map[string]any) { p["completed"] = true },
		"negative time":   func(p map[string]any) { p["timing"].(map[string]any)["start_ns"] = json.Number("-1") },
		"unknown context": func(p map[string]any) { p["timing"].(map[string]any)["context_at_start_return"] = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			value, err := decodeValue(failedProcessFixture(t))
			if err != nil {
				t.Fatal(err)
			}
			r := value.(map[string]any)
			p := r["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)["runs"].([]any)[0].(map[string]any)
			edit(p)
			raw, _ := json.Marshal(r)
			if _, err := ReadSnapshot(raw); err == nil {
				t.Fatal("changed process evidence accepted")
			}
		})
	}
}

func TestFailedProcessSnapshotKeepsMissingTimingUnknown(t *testing.T) {
	value, err := decodeValue(failedProcessFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	r := value.(map[string]any)
	runtime := r["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)
	p := runtime["runs"].([]any)[0].(map[string]any)
	delete(p, "timing")
	raw, _ := json.Marshal(map[string]any{"runtime": runtime})
	s, err := ReadSnapshot(raw)
	if err != nil || len(s.Processes) != 1 || s.Processes[0].Input.HasTiming || s.Processes[0].Input.HasWaitLimit {
		t.Fatal("missing timing became an observation", s, err)
	}
}

func TestFailedProcessReaderKeepsProvidedCountsAndEnvelopeScope(t *testing.T) {
	counts := []byte(`{"passed":1,"total":2,"construction":[]}`)
	if s, err := ReadSnapshot(counts); err != nil || s.Passed != 1 || s.Total != 2 || len(s.Processes) != 0 {
		t.Fatal("provided counts changed", s, err)
	}
	var original map[string]json.RawMessage
	if err := json.Unmarshal(failedProcessFixture(t), &original); err != nil {
		t.Fatal(err)
	}
	for _, schema := range []string{packageJointSchema, "gooo/workspace-body-execution-receipt/v1"} {
		raw, _ := json.Marshal(map[string]any{"schema": schema, "decision": "FAIL_CLOSED", "result": original})
		s, err := ReadSnapshot(raw)
		if err != nil || len(s.Processes) != 1 || s.Processes[0].Location != "/result/construction/attempts/0/runtime/runs/0" {
			t.Fatal("failed envelope lost original location", schema, s, err)
		}
	}
	input := []byte(`{"schema":"unknown","construction":{"schema":"gooo/joint-construction/v5","failure":"failed","attempts":[]}}`)
	if records, err := readFailedProcesses(input, ""); err != nil || len(records) != 0 {
		t.Fatal("unknown schema became a supported process input", records, err)
	}
}

func TestFailedProcessSnapshotPreservesLargeIntegerTiming(t *testing.T) {
	value, err := decodeValue(failedProcessFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	r := value.(map[string]any)
	p := r["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)["runs"].([]any)[0].(map[string]any)
	p["wall_ns"] = json.Number("9007199254740994")
	p["timing"].(map[string]any)["start_ns"] = json.Number("9007199254740993")
	p["timing"].(map[string]any)["wait_ns"] = json.Number("1")
	raw, _ := json.Marshal(r)
	s, err := ReadSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Timing struct {
			Start int64 `json:"start_ns"`
		}
	}
	if err := json.Unmarshal(s.Processes[0].Raw, &record); err != nil || record.Timing.Start != 9007199254740993 {
		t.Fatal("process timing was rounded", record, err)
	}
}

func TestFailedProcessSnapshotRejectsCompletedStageWithFailure(t *testing.T) {
	value, err := decodeValue(failedProcessFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	r := value.(map[string]any)
	r["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)["stage"] = "COMPLETE"
	raw, _ := json.Marshal(r)
	if _, err := ReadSnapshot(raw); err == nil {
		t.Fatal("completed runtime was treated as a failed process runtime")
	}
}

func TestFailedProcessSnapshotKeepsBothRunsAtSecondExecution(t *testing.T) {
	value, err := decodeValue(failedProcessFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	r := value.(map[string]any)
	runtime := r["construction"].(map[string]any)["attempts"].([]any)[0].(map[string]any)["runtime"].(map[string]any)
	second := runtime["runs"].([]any)[0]
	firstBytes, _ := json.Marshal(second)
	firstValue, _ := decodeValue(firstBytes)
	first := firstValue.(map[string]any)
	first["completed"], first["timed_out"], first["exit_code"] = true, false, json.Number("0")
	delete(first, "timing")
	runtime["stage"], runtime["runs"] = "EXECUTE_2", []any{first, second}
	raw, _ := json.Marshal(r)
	s, err := ReadSnapshot(raw)
	if err != nil || len(s.Processes) != 2 || !s.Processes[0].Input.Completed || !s.Processes[1].Input.TimedOut ||
		s.Processes[0].Input.HasTiming || !s.Processes[1].Input.HasTiming {
		t.Fatal("separate process observations changed", s, err)
	}
}
