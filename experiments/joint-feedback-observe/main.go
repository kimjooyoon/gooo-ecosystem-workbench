// Recount saved joint loops without converting exact JSON integers to float64.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

type run struct {
	Name            string  `json:"run"`
	Rounds          int     `json:"rounds"`
	Attempts        int64   `json:"program_attempts"`
	ConstructionNS  int64   `json:"construction_command_ns"`
	ModelCalls      int     `json:"initial_model_calls"`
	PredictNS       []int64 `json:"predict_ns"`
	AddedRows       int     `json:"added_rows"`
	HoldoutPassed   int64   `json:"holdout_passed"`
	HoldoutTotal    int64   `json:"holdout_total"`
	OtherInputs     int64   `json:"holdout_other_inputs"`
	HoldoutNewCalls int64   `json:"holdout_new_model_calls"`
	SelectedSHA     string  `json:"selected_source_sha256"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func read(name string) []byte  { b, e := os.ReadFile(name); must(e); return b }
func digest(raw []byte) string { return fmt.Sprintf("%x", sha256.Sum256(raw)) }
func value(raw []byte) any {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	must(d.Decode(&v))
	return v
}
func require(ok bool, message string) {
	if !ok {
		panic(message)
	}
}
func rows(raw []byte) []json.RawMessage {
	var doc struct {
		Cases []json.RawMessage `json:"cases"`
	}
	must(json.Unmarshal(raw, &doc))
	return doc.Cases
}

func observe(root, name string) run {
	dir := filepath.Join(root, name)
	var loop workbench.JointLoop
	must(json.Unmarshal(read(filepath.Join(dir, "joint-loop.json")), &loop))
	require(loop.Failure == "" && len(loop.Rounds) > 0, "unfinished loop")
	out := run{Name: name, Rounds: len(loop.Rounds), PredictNS: []int64{}}
	adaptive := read(filepath.Join(dir, "evaluation-cases.json"))
	adaptiveRows := rows(adaptive)
	type assembly struct {
		Calls     int   `json:"model_calls"`
		PredictNS int64 `json:"predict_ns"`
	}
	type step struct {
		Generation struct {
			Report struct {
				Assembly *assembly `json:"record_assembly"`
			} `json:"report"`
		} `json:"generation"`
	}
	for i, round := range loop.Rounds {
		raw := read(filepath.Join(dir, round.Result))
		snapshot, err := workbench.ReadSnapshot(raw)
		must(err)
		require(snapshot.Joint != nil && snapshot.Joint.ProgramAttempts == round.Attempts && snapshot.Passed == round.EvaluationPassed && snapshot.Total == round.EvaluationTotal, "round recount differed")
		out.Attempts += round.Attempts
		out.ConstructionNS += round.ConstructionElapsedNS
		var report struct {
			Construction struct {
				Initial struct{ Preparations, Steps []step } `json:"initial"`
			} `json:"construction"`
		}
		must(json.Unmarshal(raw, &report))
		for _, s := range append(report.Construction.Initial.Preparations, report.Construction.Initial.Steps...) {
			if a := s.Generation.Report.Assembly; a != nil {
				out.ModelCalls += a.Calls
				if a.Calls > 0 {
					out.PredictNS = append(out.PredictNS, a.PredictNS)
				}
			}
		}
		if f := round.Feedback; f != nil {
			current := read(filepath.Join(dir, round.ConstructionFile))
			require(f.ResultSHA256 == digest(raw) && f.EvaluationSHA256 == digest(adaptive) && f.PreviousSHA256 == digest(current), "feedback input digest differed")
			if f.Prepared {
				next := read(filepath.Join(dir, f.NextCasesFile))
				require(f.NextSHA256 == digest(next), "feedback output digest differed")
				before, after := rows(current), rows(next)
				require(len(after) == len(before)+len(f.AddedIndices), "feedback row count differed")
				for j, row := range before {
					require(reflect.DeepEqual(value(row), value(after[j])), "original expectation changed")
				}
				for j, index := range f.AddedIndices {
					require(index >= 0 && index < len(adaptiveRows) && reflect.DeepEqual(value(adaptiveRows[index]), value(after[len(before)+j])), "counterexample changed")
				}
				require(f.Consumed == (i+1 < len(loop.Rounds) && loop.Rounds[i+1].ConstructionFile == f.NextCasesFile), "prepared/consumed distinction differed")
				out.AddedRows += len(f.AddedIndices)
			}
		}
	}
	final, err := workbench.ReadSnapshot(read(filepath.Join(dir, "holdout-result.json")))
	must(err)
	require(loop.FinalEvaluation != nil && reflect.DeepEqual(final, *loop.FinalEvaluation), "final holdout recount differed")
	require(final.Joint != nil && final.Joint.Replayed && final.Joint.NewModelCalls == 0, "holdout performed inference")
	out.HoldoutPassed, out.HoldoutTotal = final.Passed, final.Total
	out.OtherInputs = *final.Joint.Inputs.Other
	out.HoldoutNewCalls = final.Joint.NewModelCalls
	out.SelectedSHA = digest(read(filepath.Join(dir, loop.FinalDirectory, "selected.gooo")))
	return out
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: go run ./experiments/joint-feedback-observe <saved-study-directory>")
	}
	var runs []run
	for _, family := range []string{"arithmetic", "charge"} {
		fixed, model := observe(os.Args[1], family+"-fixed"), observe(os.Args[1], family+"-model")
		require(fixed.SelectedSHA == model.SelectedSHA, "selected sources differed between ordering modes")
		require(fixed.ModelCalls == 0 && model.ModelCalls == model.Rounds, "model calls differed from actual requested construction rounds")
		runs = append(runs, fixed, model)
	}
	out := map[string]any{"schema": "gooo/joint-feedback-study/v1", "program_families": 2, "runs": runs, "scope": "two frozen programs, four runs; no weight changes; finite holdout counts and exact counterexample preservation; repeated rounds are repeated work; timings are local observations while the test suite was also running"}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	must(encoder.Encode(out))
}
