package main

import (
	"context"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func TestGradientMatchesFiniteDifference(t *testing.T) {
	n := initialize(17)
	for j := range hidden {
		n[w1Count+j] = .4
	}
	e := fieldExample{Active: []uint16{0, 192}, Label: 1}
	e.X[0], e.X[192] = .5, .7
	f := forwardWeights(n, false)
	var gradientValue [parameterCount]float64
	gradient(&f, &e, .5, &gradientValue)
	for _, index := range []int{0, 192, w1Count, w2Start, w2Start + hidden} {
		plus, minus := n, n
		plus[index] += .001
		minus[index] -= .001
		p, m := forwardWeights(plus, false), forwardWeights(minus, false)
		var scratch [parameterCount]float64
		want := (gradient(&p, &e, .5, &scratch) - gradient(&m, &e, .5, &scratch)) / .002
		if math.Abs(want-gradientValue[index]) > .002 {
			t.Fatal("gradient differs", index, want, gradientValue[index])
		}
	}
}

func TestTrainingIsRepeatableAndLearnsSyntheticBit(t *testing.T) {
	var data []fieldExample
	for i := range 16 {
		e := fieldExample{Label: i % 2, Active: []uint16{uint16(i % 2)}}
		e.X[i%2] = 1
		data = append(data, e)
	}
	initial := initialize(91)
	first, trace, err := train(context.Background(), initial, data, 80, 8, .01, .5, 91, false)
	if err != nil {
		t.Fatal(err)
	}
	second, again, err := train(context.Background(), initial, data, 80, 8, .01, .5, 91, false)
	if err != nil || first != second || !reflect.DeepEqual(trace, again) || first == initial {
		t.Fatal("training did not deterministically update weights", err)
	}
	f := forwardWeights(first, false)
	for i := range data {
		_, z := f.forward(&data[i])
		got := 0
		if z[1] > z[0] {
			got = 1
		}
		if got != data[i].Label {
			t.Fatal("synthetic learnability regression")
		}
	}
	if trace[len(trace)-1].Mean >= trace[0].Mean {
		t.Fatal("synthetic loss did not decrease")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := train(ctx, initial, data, 1, 8, .01, .5, 91, false); err == nil {
		t.Fatal("canceled training continued")
	}
}

func TestExportMatchesSDKArithmeticAndCompactStorage(t *testing.T) {
	n := initialize(29)
	for j := range hidden {
		n[w1Count+j] = .1
	}
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		path, err := exportModel(filepath.Join(t.TempDir(), variant), n, variant, .5)
		if err != nil {
			t.Fatal(err)
		}
		m, err := jointdecision.LoadRecordGraphSharedThree(path)
		if err != nil {
			t.Fatal(err)
		}
		f := forwardWeights(n, variant != "fp32")
		var x [jointdecision.ThreeFeatureDim]float32
		var activations [3][hidden]float32
		for part := range 3 {
			e := fieldExample{Active: []uint16{0, 8, 192}}
			e.X[0], e.X[8], e.X[192] = .2, float32(part)*.1, .3
			copy(x[part*width:(part+1)*width], e.X[:])
			activations[part], _ = f.forward(&e)
		}
		var workspace jointdecision.ThreeWorkspace
		var output jointdecision.ThreePrediction
		if err = m.PredictRecordGraphSharedFeaturesInto(&x, &workspace, &output); err != nil {
			t.Fatal(err)
		}
		for mask := range 8 {
			var want float32
			for part := range 3 {
				label := (mask >> part) & 1
				for j, value := range activations[part] {
					weight := f.Values[w2Start+label*hidden+j]
					if variant != "fp32" {
						weight = float32(f.Codes2[label*hidden+j])
					}
					want = float32(want + float32(value*weight))
				}
			}
			if variant != "fp32" {
				want = float32(want * f.Scale2)
			}
			want = float32(want + float32(0))
			if math.Float32bits(want) != math.Float32bits(output.Logits[mask]) {
				t.Fatal("SDK and training/export arithmetic differ", variant, mask, want, output.Logits[mask])
			}
		}
		if variant == "fp32" {
			if m.PackedFileBytes() != 8288 || m.ResidentTensorBytes() != 8288 {
				t.Fatal("FP32 size differs")
			}
		} else if m.PackedFileBytes() != 446 || m.ResidentTensorBytes() != 2096 || m.MatrixScaleBytes() != 8 {
			t.Fatal("ternary size differs")
		}
	}
}
