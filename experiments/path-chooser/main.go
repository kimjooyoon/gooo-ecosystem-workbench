package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

func main() {
	out := flag.String("out", "", "new external observation directory")
	flag.Parse()
	if *out == "" {
		panic("new external output required")
	}
	source, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	must(err)
	if len(dirty) != 0 {
		panic("clean committed producer required")
	}
	training, evaluation, err := corpus("experiments/path-chooser")
	must(err)
	must(os.Mkdir(*out, 0755))
	must(save(filepath.Join(*out, "training-rows.json"), training))
	must(save(filepath.Join(*out, "evaluation-rows.json"), evaluation))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	started := time.Now()
	base, trace, err := train(ctx, initialize(20261010), training, 100, .01, false)
	must(err)
	must(save(filepath.Join(*out, "fp32-loss.json"), trace))
	qat, trace, err := train(ctx, base, training, 100, .002, true)
	must(err)
	must(save(filepath.Join(*out, "qat-loss.json"), trace))
	trainingNS := time.Since(started).Nanoseconds()
	results := []map[string]any{}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		n := base
		if variant == "qat_ternary" {
			n = qat
		}
		path, err := export(filepath.Join(*out, variant), n, variant)
		must(err)
		model, err := decision.LoadPath(path)
		must(err)
		results = append(results, map[string]any{"variant": variant, "metadata_sha256": model.MetadataSHA256(),
			"weights_sha256": model.WeightsSHA256(), "packed_weights_bytes": model.PackedFileBytes(),
			"resident_tensor_bytes": model.ResidentTensorBytes(), "scratch_bytes": decision.WorkspaceBytes(),
			"training": assess(model, training), "unseen_utterances": assess(model, evaluation), "arity_benchmark": benchmark(model, evaluation)})
	}
	must(save(filepath.Join(*out, "study.json"), map[string]any{
		"schema": "gooo/per-choice-decision-study/v1", "producer": strings.TrimSpace(string(source)),
		"device": "CPU_Go", "go_version": runtime.Version(), "parameters": parameterCount, "train_rows": len(training),
		"evaluation_rows": len(evaluation), "training_phase_ns": trainingNS, "source_context_producer": "426caecb711e47da26fe659237a117f301d112f4",
		"source_context_sha256": fileDigest("experiments/path-chooser/source-context.json"), "results": results,
		"scope":                "Six source-derived site snapshots; unseen authored utterances share source shapes and direction wording. No unseen-program, calibrated confidence or native functional claim from classification.",
		"host_cpu_utilization": "UNMEASURED", "gpu_utilization": "NOT_USED", "token_generation": false}))
	fmt.Println("own per-choice decision study completed; original arrays and models retained")
}

func assess(model *decision.Model, data []example) map[string]any {
	var work decision.Workspace
	var p decision.Prediction
	passed, abstained := 0, 0
	rows := []map[string]any{}
	for _, e := range data {
		must(model.PredictInto(e.Text, &work, &p))
		if p.TopIndex == e.Label {
			passed++
		}
		if p.Abstained {
			abstained++
		}
		rows = append(rows, map[string]any{"id": e.ID, "expected_label": e.Label, "actual_label": p.TopIndex,
			"confidence": p.Confidence, "abstained": p.Abstained, "passed": p.TopIndex == e.Label})
	}
	return map[string]any{"passed": passed, "total": len(data), "abstained": abstained, "observations": rows}
}

func benchmark(model *decision.Model, data []example) []map[string]any {
	var workspace decision.Workspace
	var p decision.Prediction
	rows := []map[string]any{}
	for _, arity := range []int{1, 2, 3, 6, 16} {
		started := time.Now()
		for range 200 {
			for i := range arity {
				must(model.PredictInto(data[i%len(data)].Text, &workspace, &p))
			}
		}
		ns := time.Since(started).Nanoseconds()
		rows = append(rows, map[string]any{"choices": arity, "repeats": 200, "model_calls": 200 * arity,
			"mean_decision_ns": float64(ns) / float64(200*arity), "mean_sequence_ns": float64(ns) / 200,
			"scope": "serial independent site ranking with caller-owned workspace; no compilation or candidate execution"})
	}
	return rows
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func fileDigest(path string) string { raw, err := os.ReadFile(path); must(err); return digest(raw) }
