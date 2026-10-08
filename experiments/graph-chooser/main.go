package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type plan struct {
	Schema      string   `json:"schema"`
	Seed        uint64   `json:"seed"`
	Feature     string   `json:"feature_version"`
	Input       string   `json:"input"`
	FP32Epochs  int      `json:"fp32_epochs"`
	QATEpochs   int      `json:"qat_epochs"`
	Batch       int      `json:"batch_fields"`
	FP32Rate    float64  `json:"fp32_learning_rate"`
	QATRate     float64  `json:"qat_learning_rate"`
	Temperature float64  `json:"temperature"`
	Repeats     int      `json:"benchmark_repeats"`
	Folds       []string `json:"folds"`
}

func main() {
	config := flag.String("plan", "experiments/graph-chooser/plan.json", "frozen study configuration")
	out := flag.String("out", "", "new observation directory")
	legacy := flag.String("legacy-model", "models/shared-qat/model.json", "existing v1 model for explicit v1 projection baseline")
	flag.Parse()
	if *out == "" {
		panic("new output directory required")
	}
	raw, err := os.ReadFile(*config)
	must(err)
	var p plan
	must(json.Unmarshal(raw, &p))
	if p.Schema != "gooo/graph-chooser-study-plan/v1" || p.Feature != jointdecision.RecordGraphSharedFeatureVersion ||
		p.FP32Epochs < 1 || p.FP32Epochs > 1000 || p.QATEpochs < 1 || p.QATEpochs > 1000 || p.Batch < 1 || p.Batch > 1728 ||
		p.FP32Rate <= 0 || p.FP32Rate > .1 || p.QATRate <= 0 || p.QATRate > .1 || p.Temperature <= 0 || p.Repeats < 1 || p.Repeats > 100 ||
		strings.Join(p.Folds, ",") != "filenames,division,retry,all-data-demonstration" {
		panic("bounded predeclared study required")
	}
	source, err := exec.Command("git", "rev-parse", "HEAD").Output()
	must(err)
	must(exec.Command("git", "diff", "--exit-code", "HEAD", "--").Run())
	must(os.Mkdir(*out, 0755))
	start := time.Now()
	rows, inputSHA, err := loadRows(p.Input)
	must(err)
	preparationNS := time.Since(start).Nanoseconds()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	all, _ := split(rows, "all-data-demonstration")
	old, err := jointdecision.LoadRecordSharedThree(*legacy)
	must(err)
	baseline, err := evaluate(rows, all, old, "legacy")
	must(err)
	save(filepath.Join(*out, "legacy-v1-assessment.json"), map[string]any{"projection": "Explicit graph choice field/first/second/intent to v1 input; source nodes and roots are excluded.", "feature_version": old.FeatureVersion(), "model_metadata_sha256": old.MetadataSHA256(), "model_weights_sha256": old.WeightsSHA256(), "score": baseline})
	report := map[string]any{"schema": "gooo/graph-chooser-study/v1", "code_source_sha": strings.TrimSpace(string(source)), "plan_sha256": digest(raw), "input_sha256": inputSHA,
		"plan": p, "go_version": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH, "device": "CPU", "parameters": parameterCount, "input_preparation_ns": preparationNS,
		"scope": "Three-family authored corpus. Held-out folds and in-sample demonstration are explicit. No general natural-language-programming accuracy or native execution claim in these array predictions."}
	var results []map[string]any
	for _, fold := range p.Folds {
		training, evaluation := split(rows, fold)
		root := filepath.Join(*out, fold)
		must(os.Mkdir(root, 0755))
		save(filepath.Join(root, "split.json"), map[string]any{"held_out_family": fold, "training_ids": identities(rows, training), "evaluation_ids": identities(rows, evaluation)})
		data := examples(rows, training)
		start = time.Now()
		fp32, loss, err := train(ctx, initialize(p.Seed), data, p.FP32Epochs, p.Batch, p.FP32Rate, p.Temperature, p.Seed, false)
		must(err)
		fp32NS := time.Since(start).Nanoseconds()
		save(filepath.Join(root, "fp32-loss.json"), loss)
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			weights := fp32
			trainNS := fp32NS
			if variant == "qat_ternary" {
				start = time.Now()
				var loss []epochLoss
				weights, loss, err = train(ctx, fp32, data, p.QATEpochs, p.Batch, p.QATRate, p.Temperature, p.Seed, true)
				must(err)
				trainNS = time.Since(start).Nanoseconds()
				save(filepath.Join(root, "qat-loss.json"), loss)
			}
			modelRoot := filepath.Join(root, variant)
			path, err := exportModel(modelRoot, weights, variant, p.Temperature)
			must(err)
			assessment := assessArtifact(modelRoot, path, rows, training, evaluation, p.Repeats)
			results = append(results, map[string]any{"fold": fold, "variant": variant, "training_phase_ns": trainNS, "training_phase": map[bool]string{true: "additional-qat", false: "shared-fp32-phase"}[variant == "qat_ternary"], "assessment": assessment})
			report["results"] = results
			save(filepath.Join(*out, "study.json"), report)
			fmt.Printf("%s %s: %d training rows, %d evaluation rows; exported and reloaded\n", fold, variant, len(training), len(evaluation))
		}
	}
	save(filepath.Join(*out, "plan.json"), p)
}

func identities(rows []row, indices []int) []string {
	result := make([]string, 0, len(indices))
	for _, i := range indices {
		result = append(result, rows[i].ID)
	}
	return result
}
