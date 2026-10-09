package main

import (
	"context"
	"math"
	"path/filepath"
	"testing"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

func TestPathTrainingGradientAndSDKRoundTrip(t *testing.T) {
	trainRows, testRows, err := corpus(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(trainRows) != 48 || len(testRows) != 24 {
		t.Fatal(len(trainRows), len(testRows))
	}
	n := initialize(7)
	e := trainRows[0]
	for i := range hidden {
		n[w1Count+i] = .4
	}
	var g [parameterCount]float64
	gradient(&n, &e, &g)
	for _, at := range []int{e.Active[0], w1Count, w2Start, b2Start} {
		plus, minus := n, n
		plus[at] += .001
		minus[at] -= .001
		var scratch [parameterCount]float64
		want := (gradient(&plus, &e, &scratch) - gradient(&minus, &e, &scratch)) / .002
		if math.Abs(want-g[at]) > .002 {
			t.Fatal("gradient mismatch", at, want, g[at])
		}
	}
	n, _, err = train(context.Background(), n, trainRows, 5, .01, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		path, err := export(filepath.Join(t.TempDir(), variant), n, variant)
		if err != nil {
			t.Fatal(err)
		}
		model, err := decision.LoadPath(path)
		if err != nil {
			t.Fatal(err)
		}
		var work decision.Workspace
		var p decision.Prediction
		if err := model.PredictInto(e.Text, &work, &p); err != nil {
			t.Fatal(err)
		}
		forward := projected(n, variant != "fp32")
		_, z := logits(&forward, &e)
		for i := range labels {
			if math.Abs(float64(z[i]-p.Logits[i])) > 2e-5 {
				t.Fatal("SDK logits differ", variant, i, z[i], p.Logits[i])
			}
		}
		if variant == "fp32" && model.PackedFileBytes() != 50912 {
			t.Fatal("FP32 bytes differ")
		}
		if variant != "fp32" && (model.PackedFileBytes() != 2759 || model.ResidentTensorBytes() != 12896) {
			t.Fatal("ternary bytes differ")
		}
		allocs := testing.AllocsPerRun(100, func() {
			if err := model.PredictInto(e.Text, &work, &p); err != nil {
				panic(err)
			}
		})
		if allocs != 0 {
			t.Fatal("inference allocated", allocs)
		}
	}
}
