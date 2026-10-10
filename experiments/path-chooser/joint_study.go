package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

func jointStudy(out, producer string, training, evaluation []example) error {
	base, err := loadOriginalFP32(originalFP32Root)
	if err != nil {
		return err
	}
	target, err := loadJointCorpus(jointCorpusRoot)
	if err != nil {
		return err
	}
	if err := os.Mkdir(out, 0755); err != nil {
		return err
	}
	if err := save(filepath.Join(out, "consumed-joint-target.json"), target); err != nil {
		return err
	}
	if err := save(filepath.Join(out, "training-label-rows.json"), training); err != nil {
		return err
	}
	if err := save(filepath.Join(out, "development-label-rows.json"), evaluation); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	started := time.Now()
	results := []map[string]any{}
	for _, arm := range []string{"label_only", "joint"} {
		if err := os.Mkdir(filepath.Join(out, arm), 0755); err != nil {
			return err
		}
		n := base
		for _, variant := range []string{"fp32", "qat_ternary"} {
			qat := variant == "qat_ternary"
			var trace any
			if arm == "label_only" {
				n, trace, err = train(ctx, n, training, 100, .002, qat)
			} else {
				n, trace, err = trainJoint(ctx, n, training, target, 100, .002, qat)
			}
			if err != nil {
				return err
			}
			if err := save(filepath.Join(out, arm, variant+"-loss.json"), trace); err != nil {
				return err
			}
			path, err := export(filepath.Join(out, arm, variant), n, variant)
			if err != nil {
				return err
			}
			model, err := decision.LoadPath(path)
			if err != nil {
				return err
			}
			results = append(results, jointAssessment(model, target, training, evaluation, arm, variant))
		}
	}
	original, err := decision.LoadPath("models/per-choice-20261010/qat_ternary/model.json")
	if err != nil {
		return err
	}
	results = append(results, jointAssessment(original, target, training, evaluation, "previous_published", "qat_ternary"))
	return save(filepath.Join(out, "study.json"), map[string]any{
		"schema": "gooo/joint-path-learning-study/v1", "producer": producer, "go_version": runtime.Version(),
		"device": "CPU_Go", "parameters": parameterCount, "original_fp32_weights_sha256": originalFP32SHA,
		"epochs_per_phase": 100, "updates_per_phase": 100 * ((len(training) + 7) / 8), "learning_rate": .002,
		"joint_arm_loss_weights": map[string]float64{"rehearsal_mean": .5, "joint_target": .5},
		"target_archive_sha256":  jointTargetsSHA, "context_archive_sha256": jointContextSHA,
		"training_and_assessment_ns": time.Since(started).Nanoseconds(), "results": results,
		"scope":                "One consumed source family and its finite compatible mask set. Original24 utterances are a development regression set sharing prior source snapshots. No held-out-source, native functionality or calibrated probability claim from these scores.",
		"host_cpu_utilization": "UNMEASURED", "gpu_utilization": "NOT_USED", "token_generation": false,
	})
}

func jointAssessment(model *decision.Model, target jointExample, training, evaluation []example, arm, variant string) map[string]any {
	return map[string]any{"arm": arm, "variant": variant, "metadata_sha256": model.MetadataSHA256(),
		"weights_sha256": model.WeightsSHA256(), "packed_weights_bytes": model.PackedFileBytes(),
		"resident_tensor_bytes": model.ResidentTensorBytes(), "scratch_bytes": decision.WorkspaceBytes(),
		"rehearsal_labels": assess(model, training), "development_labels": assess(model, evaluation),
		"consumed_joint_target": assessJoint(model, target), "repeated_site_benchmark": benchmark(model, target.Sites)}
}

func assessJoint(model *decision.Model, e jointExample) map[string]any {
	var work decision.Workspace
	var p decision.Prediction
	pairs := make([][2]float64, len(e.Sites))
	mask := uint16(0)
	rows := []map[string]any{}
	for i, site := range e.Sites {
		must(model.PredictInto(site.Text, &work, &p))
		labels := e.Allowed[i]
		a, b := float64(p.Logits[labels[0]]/model.Temperature()), float64(p.Logits[labels[1]]/model.Temperature())
		maximum := max(a, b)
		x, y := math.Exp(a-maximum), math.Exp(b-maximum)
		pairs[i] = [2]float64{x / (x + y), y / (x + y)}
		if pairs[i][1] > pairs[i][0] {
			mask |= 1 << i
		}
		rows = append(rows, map[string]any{"id": site.ID, "input_sha256": digest([]byte(site.Text)), "global_label": model.PredictLabel(&p), "global_confidence": p.Confidence, "abstained": p.Abstained, "allowed_probabilities": pairs[i]})
	}
	accepted := 0.
	for _, m := range e.Masks {
		mass := 1.
		for i := range e.Sites {
			mass *= pairs[i][(m>>i)&1]
		}
		accepted += mass
	}
	if math.IsNaN(accepted) || math.IsInf(accepted, 0) {
		panic(fmt.Errorf("nonfinite joint ranking mass"))
	}
	return map[string]any{"compatible_masks": e.Masks, "top_mask": mask, "top_mask_compatible": slices.Contains(e.Masks, mask),
		"allowed_pair_compatible_mass": accepted, "sites": rows, "scope": "Factorized ranking mass over consumed finite targets; not a calibrated correctness probability or native execution."}
}
