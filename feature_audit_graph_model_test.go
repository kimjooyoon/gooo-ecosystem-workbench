package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

// A zero-weight model checks only SDK dispatch and report accounting. It has
// no learned knowledge; stable score ties choose mask 0 for every input.
func TestFeatureAuditGraphSyntheticModelAccounting(t *testing.T) {
	meta := jointdecision.Metadata{Schema: jointdecision.SharedThreeSchema, Variant: "fp32",
		Feature: jointdecision.RecordGraphSharedFeatureVersion, Arithmetic: jointdecision.SeparateArithmeticVersion,
		FeatureDim: 768, HiddenDim: 8, MaxBytes: jointdecision.RecordGraphInputMaxBytes,
		Temperature: 1, WeightsFile: "weights.bin"}
	var weights []byte
	for i, name := range []string{"w1", "b1", "w2"} {
		rows, cols := [3]int{8, 1, 2}[i], [3]int{256, 8, 8}[i]
		meta.Tensors = append(meta.Tensors, decision.TensorMetadata{Name: name, Rows: rows, Cols: cols,
			Count: rows * cols, Encoding: "float32_le", Offset: int64(len(weights)), Bytes: int64(4 * rows * cols), Scale: 1})
		weights = append(weights, make([]byte, 4*rows*cols)...)
	}
	for i := range 8 {
		meta.Labels = append(meta.Labels, fmt.Sprintf("mask_%d", i))
	}
	meta.WeightsSHA = fmt.Sprintf("%x", sha256.Sum256(weights))
	root := t.TempDir()
	raw, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "weights.bin"), weights, 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "model.json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	model, err := loadAuditModel(path, meta.Feature)
	if err != nil {
		t.Fatal(err)
	}
	_, input := graphAuditFixture(t)
	r, err := auditFeatureRows(context.Background(), input, model)
	if err != nil || !r.ModelObserved || r.ModelCalls != 8 || r.ModelSatisfied != 1 || r.MaximumCompatible != 8 || len(r.Predictions) != 8 {
		t.Fatal("synthetic model dispatch/accounting differs", r, err)
	}
	for _, p := range r.Predictions {
		if p.Mask != 0 {
			t.Fatal("zero weights did not preserve tie order", p)
		}
	}
}
