package main

import (
	"fmt"
	"math/bits"
	"path/filepath"
	"time"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

type prediction struct {
	ID        string `json:"id"`
	Expected  uint16 `json:"expected_mask"`
	Predicted uint16 `json:"predicted_mask"`
}
type score struct {
	Rows        int          `json:"rows"`
	Correct     int          `json:"correct_masks"`
	Fields      int          `json:"correct_fields"`
	FieldTotal  int          `json:"total_fields"`
	Pairs       int          `json:"intent_flip_pairs"`
	ExactFlips  int          `json:"exact_predicted_flips"`
	BothCorrect int          `json:"pairs_with_both_masks_correct"`
	Predictions []prediction `json:"predictions"`
}

func evaluate(rows []row, indices []int, m *jointdecision.ThreeModel, mode string) (score, error) {
	var result score
	byID := map[string]prediction{}
	var workspace jointdecision.ThreeWorkspace
	for _, i := range indices {
		r := &rows[i]
		features := &r.Full
		if mode == "ablated" {
			features = &r.Ablated
		} else if mode == "legacy" {
			features = &r.Legacy
		}
		var output jointdecision.ThreePrediction
		var err error
		if mode == "legacy" {
			err = m.PredictRecordSharedFeaturesInto(features, &workspace, &output)
		} else {
			err = m.PredictRecordGraphSharedFeaturesInto(features, &workspace, &output)
		}
		if err != nil {
			return result, err
		}
		p := prediction{r.ID, r.Label, output.Mask}
		byID[r.ID] = p
		result.Predictions = append(result.Predictions, p)
		result.Rows++
		result.FieldTotal += 3
		if output.Mask == r.Label {
			result.Correct++
		}
		result.Fields += 3 - bits.OnesCount16(output.Mask^r.Label)
	}
	for _, i := range indices {
		r := &rows[i]
		for bit := range 3 {
			if r.Request&(1<<bit) != 0 {
				continue
			}
			otherID := fmt.Sprintf("%s-%s-%d-request%d", r.Family, r.Language, r.Order, r.Request^(1<<bit))
			other, ok := byID[otherID]
			if !ok {
				continue
			}
			p := byID[r.ID]
			result.Pairs++
			if p.Predicted^other.Predicted == 1<<bit {
				result.ExactFlips++
			}
			if p.Predicted == p.Expected && other.Predicted == other.Expected {
				result.BothCorrect++
			}
		}
	}
	return result, nil
}

func assessArtifact(root, path string, rows []row, training, evaluation []int, repeats int) map[string]any {
	start := time.Now()
	m, err := jointdecision.LoadRecordGraphSharedThree(path)
	must(err)
	loadNS := time.Since(start).Nanoseconds()
	fit, err := evaluate(rows, training, m, "full")
	must(err)
	test, err := evaluate(rows, evaluation, m, "full")
	must(err)
	ablated, err := evaluate(rows, evaluation, m, "ablated")
	must(err)
	indices := evaluation
	if len(indices) == 0 {
		indices = training
		ablated, err = evaluate(rows, indices, m, "ablated")
		must(err)
	}
	var workspace jointdecision.ThreeWorkspace
	var output jointdecision.ThreePrediction
	for range 16 {
		must(m.PredictRecordGraphSharedFeaturesInto(&rows[indices[0]].Full, &workspace, &output))
	}
	start = time.Now()
	checksum := uint64(0)
	for range repeats {
		for _, i := range indices {
			must(m.PredictRecordGraphSharedFeaturesInto(&rows[i].Full, &workspace, &output))
			checksum += uint64(output.Mask)
		}
	}
	elapsed := time.Since(start).Nanoseconds()
	r := map[string]any{"model_metadata_sha256": m.MetadataSHA256(), "model_weights_sha256": m.WeightsSHA256(), "variant": m.Variant(),
		"train": fit, "evaluation": test, "intent_ablated": ablated, "evaluation_is_held_out": len(evaluation) > 0,
		"ablated_scope": map[bool]string{true: "held-out-family", false: "all-data-in-sample"}[len(evaluation) > 0],
		"load_ns":       loadNS, "prepared_prediction_calls": repeats * len(indices), "prepared_prediction_ns": elapsed,
		"prediction_checksum": checksum, "weights_file_bytes": m.PackedFileBytes(), "resident_tensor_bytes": m.ResidentTensorBytes(), "matrix_scale_bytes": m.MatrixScaleBytes()}
	save(filepath.Join(root, "assessment.json"), r)
	return r
}
