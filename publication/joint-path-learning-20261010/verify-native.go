package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func raw(path string) []byte {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		b, err = os.ReadFile(path + ".gz")
		must(err)
		z, err := gzip.NewReader(bytes.NewReader(b))
		must(err)
		defer z.Close()
		b, err = io.ReadAll(z)
	}
	must(err)
	return b
}
func read(path string) any {
	d := json.NewDecoder(bytes.NewReader(raw(path)))
	d.UseNumber()
	var v any
	must(d.Decode(&v))
	return v
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
func field(value any, names ...string) any {
	for _, name := range names {
		var ok bool
		value, ok = value.(map[string]any)[name]
		if !ok {
			panic("missing " + name)
		}
	}
	return value
}
func eq(a, b any, label string) {
	x, err := json.Marshal(a)
	must(err)
	y, err := json.Marshal(b)
	must(err)
	if !bytes.Equal(x, y) {
		panic("mismatch " + label)
	}
}
func num(v any) int64 { n, err := v.(json.Number).Int64(); must(err); return n }

func main() {
	if len(os.Args) != 3 {
		panic("native directory and training directory required")
	}
	root, training := os.Args[1], os.Args[2]
	cases := field(read(root+"/evaluation-cases.json"), "cases").([]any)
	eq(len(cases), 11, "frozen evaluation count")
	consumed := map[int64]bool{0: true, 3: true, 5: true, -7: true, 1: true, -1: true}
	for _, test := range read(root + "/consumed-training-cases.json").([]any) {
		consumed[num(field(test, "input"))] = true
	}
	seen := map[int64]bool{}
	for _, test := range cases {
		input := num(field(test, "inputs", "Main"))
		if consumed[input] || seen[input] {
			panic("evaluation input reused or duplicated")
		}
		seen[input] = true
	}
	study := read(training + "/study.json")
	sourceSHA := fmt.Sprintf("sha256:%x", sha256.Sum256(raw(root+"/source.gooo")))
	results := []map[string]any{}
	for _, arm := range []string{"previous", "control", "joint"} {
		name := map[string]string{"previous": "previous_published", "control": "label_only", "joint": "joint"}[arm]
		var trained any
		for _, r := range field(study, "results").([]any) {
			if field(r, "arm") == name && field(r, "variant") == "qat_ternary" {
				trained = r
			}
		}
		if trained == nil {
			panic("training arm absent")
		}
		pre := read(root + "/" + arm + "-preflight.json")
		original := read(root + "/" + arm + ".stdout.json")
		replay := read(root + "/" + arm + "-replay.stdout.json")
		eq(field(pre, "original_source_sha256"), sourceSHA, "bound source")
		eq(field(original, "construction", "initial", "original_source_sha256"), sourceSHA, "actual construction source")
		eq(field(pre, "model_predictions"), 0, "preflight inference")
		eq(field(pre, "candidate_tests"), 0, "preflight tests")
		eq(field(pre, "model_compatibility", "status"), "READY_FOR_RANKING", "preflight representation")
		body := field(field(original, "construction", "initial", "preparations").([]any)[0], "generation", "report", "body_paths")
		selection := field(body, "search", "selection")
		eq(field(selection, "local_model_predictions"), 3, "three actual initial calls")
		eq(field(selection, "external_provider_calls"), 0, "external calls")
		for _, key := range []string{"metadata_sha256", "weights_sha256"} {
			eq(field(trained, key), field(pre, "model_compatibility", "model", key), "training/preflight artifact")
			eq(field(trained, key), field(body, "model_retention", key), "training/native artifact")
		}
		inputs := field(pre, "inputs").([]any)
		nativeInputs := field(body, "model_context", "inputs").([]any)
		eq(len(inputs), 3, "source sites")
		eq(len(nativeInputs), len(inputs), "actual sites")
		for i, input := range inputs {
			eq(field(input, "input_sha256"), field(nativeInputs[i], "input_sha256"), "actual input fingerprint")
			eq(field(input, "decision_id"), field(nativeInputs[i], "decision_id"), "actual site identity")
		}
		mask := 0
		for i, choice := range field(pre, "expanded_plan", "decisions").([]any) {
			id := field(choice, "id").(string)
			selected := field(selection, "choices", id)
			options := field(choice, "options").([]any)
			if selected == field(options[1], "label") {
				mask |= 1 << i
			} else {
				eq(selected, field(options[0], "label"), "declared choice")
			}
		}
		eq(mask, field(trained, "consumed_joint_target", "top_mask"), "SDK/native initial mask")
		var predictNS int64
		for _, r := range field(selection, "receipts").([]any) {
			predictNS += num(field(r, "predict_ns"))
		}
		actualPassed := 0
		var deliveries []any
		for stage, v := range []any{original, replay} {
			run := field(v, "evaluation", "runtime")
			eq(field(run, "producer_source_sha"), "366ee6ef8f9d2b9b57331dfd26b6f908c0289851", "compiler source")
			eq(field(run, "finite_total"), len(cases), "finite denominator")
			eq(field(v, "evaluation", "new_model_calls"), 0, "evaluation/replay new calls")
			traces := field(run, "traces").([]any)
			eq(len(traces), len(cases), "trace count")
			passed := 0
			rows := []any{}
			for i, trace := range traces {
				d := field(trace, "deliveries").([]any)
				eq(len(d), 1, "Main delivery count")
				eq(field(trace, "case_index"), i, "case index")
				eq(field(d[0], "input"), field(cases[i], "inputs", "Main"), "exact int64 input")
				eq(field(d[0], "expected"), field(cases[i], "expected", "Main"), "exact expectation")
				ok := num(field(d[0], "actual")) == num(field(d[0], "expected"))
				eq(field(d[0], "passed"), ok, "actual comparison")
				if ok {
					passed++
				}
				rows = append(rows, d[0])
			}
			eq(field(run, "finite_passed"), passed, "independent count")
			if stage == 0 {
				actualPassed, deliveries = passed, rows
			} else {
				eq(rows, deliveries, "saved exact outputs")
			}
		}
		eq(field(replay, "generated_now"), false, "saved projection")
		eq(field(replay, "evaluation", "construction_replayed"), true, "saved construction")
		results = append(results, map[string]any{"arm": arm, "selected_mask": mask, "model_calls": 3, "model_predict_ns": predictNS,
			"weights_sha256": field(trained, "weights_sha256"), "program_attempts": len(field(original, "construction", "attempts").([]any)),
			"passed": actualPassed, "total": len(cases), "replay_passed": actualPassed, "new_replay_calls": 0, "deliveries": deliveries})
	}
	out := map[string]any{"schema": "gooo/joint-path-native-study/v1", "source_sha256": sourceSHA, "training_producer": field(study, "producer"),
		"new_input_tuples": 11, "results": results, "scope": "One consumed source family with new finite caller inputs. Exact original inputs/artifacts/counts and saved replay checked; not unseen-program accuracy or calibrated confidence."}
	b, err := json.MarshalIndent(out, "", "  ")
	must(err)
	fmt.Println(string(b))
}
