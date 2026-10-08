package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	workbench "github.com/kimjooyoon/gooo-ecosystem-workbench"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func require(ok bool, detail string) {
	if !ok {
		panic(detail)
	}
}
func read(path string) []byte { b, e := os.ReadFile(path); must(e); return b }
func exact(raw []byte) any {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	must(d.Decode(&v))
	return v
}
func field(v any, names ...string) any {
	for _, name := range names {
		v = v.(map[string]any)[name]
	}
	return v
}
func unzip(path string) []byte {
	f, e := os.Open(path)
	must(e)
	defer f.Close()
	r, e := gzip.NewReader(f)
	must(e)
	defer r.Close()
	b, e := io.ReadAll(r)
	must(e)
	return b
}

type observation struct {
	Name        string `json:"name"`
	Rounds      int    `json:"rounds"`
	Attempts    int64  `json:"attempts_including_restarts"`
	Models      int64  `json:"initial_model_calls"`
	Rejections  int64  `json:"rejected_attempts_including_restarts"`
	Faults      int64  `json:"native_fault_attempts_including_restarts"`
	Passed      int64  `json:"holdout_passed"`
	Total       int64  `json:"holdout_total"`
	ReplayCalls int64  `json:"holdout_new_model_calls"`
}

func observe(root, name string) observation {
	dir := filepath.Join(root, name)
	var loop workbench.JointLoop
	must(json.Unmarshal(read(filepath.Join(dir, "joint-loop.json")), &loop))
	expectedRounds, expectedAttempts := 5, int64(14)
	if name == "model-loop" {
		expectedRounds, expectedAttempts = 8, 105
	}
	require(loop.Schema == "gooo/package-construction-loop/v1" && len(loop.Rounds) == expectedRounds, "package loop boundary")
	result := observation{Name: name, Rounds: len(loop.Rounds)}
	var last any
	for _, round := range loop.Rounds {
		raw := unzip(filepath.Join(dir, round.Result+".gz"))
		s, e := workbench.ReadSnapshot(raw)
		must(e)
		require(s.Package != nil && s.Joint != nil && s.Joint.ProgramAttempts == round.Attempts && s.Passed == round.EvaluationPassed && s.Total == round.EvaluationTotal, "round recount")
		decoded := exact(raw)
		last = field(decoded, "result", "construction")
		require(reflect.DeepEqual(field(decoded, "result", "construction_cases"), exact(read(filepath.Join(dir, round.ConstructionFile)))), "original package caller cases")
		result.Attempts += s.Joint.ProgramAttempts
		result.Rejections += s.Joint.RejectedAttempts
		result.Faults += s.Joint.NativeFaultAttempts
		for _, step := range field(last, "initial", "preparations").([]any) {
			report := field(step, "generation", "report").(map[string]any)
			if r, ok := report["record_assembly"].(map[string]any); ok {
				n, e := r["model_calls"].(json.Number).Int64()
				must(e)
				result.Models += n
			}
		}
	}
	require(result.Attempts == expectedAttempts, "restarted attempts disappeared")
	wantedModels := int64(0)
	if name == "model-loop" {
		wantedModels = 8
	}
	require(result.Models == wantedModels, "model use differs")
	// The appended counterexample retains the entire original package-keyed row.
	next := field(exact(read(filepath.Join(dir, "round-0-feedback-cases.json"))), "cases").([]any)
	require(len(next) == 2 && loop.Rounds[0].Feedback != nil && loop.Rounds[0].Feedback.Consumed, "feedback not consumed")
	for i, file := range []string{"construction-cases.json", "evaluation-cases.json"} {
		want := field(exact(read(filepath.Join(dir, file))), "cases").([]any)[0]
		require(reflect.DeepEqual(next[i], want), "original adaptive row changed")
	}
	raw := unzip(filepath.Join(dir, "holdout-result.json.gz"))
	s, e := workbench.ReadSnapshot(raw)
	must(e)
	require(s.Package != nil && s.Joint.Replayed && s.Joint.NewModelCalls == 0, "holdout inference or package mapping")
	require(reflect.DeepEqual(last, field(exact(raw), "result", "construction")), "saved construction changed during holdout")
	require(s.Passed == 4 && s.Total == 4 && *s.Joint.Inputs.Other == 4, "holdout results")
	require(bytes.Contains(raw, []byte("9007199254740993")), "exact large input disappeared")
	result.Passed, result.Total, result.ReplayCalls = s.Passed, s.Total, s.Joint.NewModelCalls
	return result
}

func main() {
	root := "publication/package-construction-20261009"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	result := []observation{observe(root, "loop"), observe(root, "model-loop")}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	must(e.Encode(result))
	fmt.Fprintln(os.Stderr, "PASS original package-keyed feedback, complete saved history, recounted attempts, model calls and exact-number holdout")
}
