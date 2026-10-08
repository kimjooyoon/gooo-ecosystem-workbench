// Recount retained source-fill runs against their frozen original expectations.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func exact(raw []byte) any {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value any
	check(d.Decode(&value))
	return value
}
func check(err error) {
	if err != nil {
		panic(err)
	}
}

func observe(name string) (map[string]any, string) {
	f, err := os.Open("examples/caller-fill-rejection/" + name + ".json.gz")
	check(err)
	defer f.Close()
	z, err := gzip.NewReader(f)
	check(err)
	defer z.Close()
	raw, err := io.ReadAll(z)
	check(err)
	s, err := workbench.ReadSnapshot(raw)
	check(err)
	var r struct {
		Generated    bool `json:"generated_now"`
		Construction struct {
			Source   string `json:"selected_source"`
			Original string `json:"original_source_sha256"`
			Initial  struct {
				Preparations []struct {
					Generation struct {
						Report struct {
							Record *struct {
								Calls     int   `json:"model_calls"`
								PredictNS int64 `json:"predict_ns"`
							} `json:"record_assembly"`
							Fill *struct {
								Calls int `json:"local_model_predictions"`
							} `json:"body_fill"`
						} `json:"report"`
					} `json:"generation"`
				} `json:"preparations"`
			} `json:"initial"`
		} `json:"construction"`
		Evaluation struct {
			Runtime struct {
				Traces []struct {
					Index      int                                          `json:"case_index"`
					Deliveries []struct{ Actual, Expected json.RawMessage } `json:"deliveries"`
				} `json:"traces"`
			} `json:"runtime"`
		} `json:"evaluation"`
	}
	check(json.Unmarshal(raw, &r))
	file := "holdout-cases.json"
	if len(name) >= 5 && name[:5] == "mixed" {
		file = "mixed-evaluation-cases.json"
	}
	original, err := os.ReadFile("examples/caller-fill-rejection/" + file)
	check(err)
	var cases struct {
		Cases []struct {
			Expected map[string]json.RawMessage `json:"expected"`
		} `json:"cases"`
	}
	check(json.Unmarshal(original, &cases))
	if len(cases.Cases) != len(r.Evaluation.Runtime.Traces) {
		panic("original row count differs")
	}
	seen := make([]bool, len(cases.Cases))
	passed := 0
	for _, row := range r.Evaluation.Runtime.Traces {
		if row.Index < 0 || row.Index >= len(seen) || seen[row.Index] || len(row.Deliveries) != 1 {
			panic("evaluation row identity differs")
		}
		seen[row.Index] = true
		want := exact(cases.Cases[row.Index].Expected["Main"])
		if !reflect.DeepEqual(want, exact(row.Deliveries[0].Expected)) {
			panic("original expectation changed")
		}
		if reflect.DeepEqual(want, exact(row.Deliveries[0].Actual)) {
			passed++
		}
	}
	if int64(passed) != s.Passed {
		panic("snapshot recount differs")
	}
	historical, fills, predict := 0, 0, int64(0)
	for _, step := range r.Construction.Initial.Preparations {
		if a := step.Generation.Report.Record; a != nil {
			historical += a.Calls
			predict += a.PredictNS
		}
		if a := step.Generation.Report.Fill; a != nil {
			fills += a.Calls
		}
	}
	fresh := 0
	if r.Generated {
		fresh = historical + fills
	}
	j := s.Joint
	for _, initial := range j.Initial {
		if !initial.Consistent {
			panic("inconsistent initial selection")
		}
	}
	return map[string]any{"run": name, "original_source_sha256": r.Construction.Original, "generated_now": r.Generated,
		"initial": j.Initial, "rejected_attempts": j.RejectedAttempts, "candidate_space": j.CandidateSpace,
		"program_attempts": j.ProgramAttempts, "native_program_attempts": j.NativeProgramAttempts, "local_training_passed": j.LocalPassed,
		"local_training_total": j.LocalTotal, "local_fill_holdout_passed": j.FillHoldoutPassed, "local_fill_holdout_total": j.FillHoldoutTotal,
		"final_passed": passed, "final_total": len(seen), "fresh_model_predictions": fresh, "historical_record_predictions": historical,
		"historical_fill_predictions": fills, "historical_predict_ns": predict, "evaluation_new_model_calls": j.NewModelCalls, "replayed": j.Replayed}, r.Construction.Source
}

func main() {
	names := []string{"budget-3", "budget-5", "budget-replay", "mixed-fixed", "mixed-model", "mixed-replay"}
	rows := make([]map[string]any, 0, len(names))
	sources := map[string]string{}
	for _, name := range names {
		r, source := observe(name)
		rows = append(rows, r)
		sources[name] = source
	}
	if sources["budget-5"] != sources["budget-replay"] || sources["mixed-fixed"] != sources["mixed-model"] || sources["mixed-model"] != sources["mixed-replay"] {
		panic("selected sources differ")
	}
	r := map[string]any{"schema": "gooo/caller-fill-rejection-observation/v1", "programs": 2, "fresh_constructions": 4, "saved_replays": 2, "observations": rows,
		"scope": "two related finite programs; unchanged graph model orders record choices; fills use source order; original final expectations recounted with exact JSON numbers; no training or general correctness claim"}
	raw, err := json.MarshalIndent(r, "", "  ")
	check(err)
	fmt.Println(string(raw))
}
