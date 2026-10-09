package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

func export(root string, n network, variant string) (string, error) {
	if err := os.Mkdir(root, 0755); err != nil {
		return "", err
	}
	labelNames := decision.PathLabels()
	threshold := .5
	meta := decision.Metadata{Schema: decision.PathMetadataSchema, FeatureVersion: decision.SemanticContextIntentFeatureVersion,
		Variant: variant, FeatureDim: width, HiddenDim: hidden, MaxBytes: decision.InputMaxBytes, Labels: labelNames[:],
		Temperature: 1, WeightsFile: "weights.bin", ConfidenceThreshold: &threshold}
	var raw []byte
	for i, segment := range [][2]int{{0, w1Count}, {w1Count, w2Start}, {w2Start, b2Start}, {b2Start, parameterCount}} {
		values := n[segment[0]:segment[1]]
		encoding, scale := "float32_le", float32(1)
		var block []byte
		if variant == "fp32" || i%2 == 1 {
			block = make([]byte, 4*len(values))
			for j, v := range values {
				binary.LittleEndian.PutUint32(block[j*4:], math.Float32bits(v))
			}
		} else {
			encoding = "ternary_base3_5"
			var codes []int8
			codes, scale = ternary(values)
			block = pack(codes)
		}
		rows, cols := [4]int{hidden, 1, labels, 1}, [4]int{width, hidden, hidden, labels}
		meta.Tensors = append(meta.Tensors, decision.TensorMetadata{Name: [4]string{"w1", "b1", "w2", "b2"}[i],
			Count: len(values), Rows: rows[i], Cols: cols[i], Encoding: encoding, Offset: int64(len(raw)), Bytes: int64(len(block)), Scale: float64(scale)})
		raw = append(raw, block...)
	}
	meta.WeightsSHA256 = digest(raw)
	if err := os.WriteFile(filepath.Join(root, "weights.bin"), raw, 0644); err != nil {
		return "", err
	}
	path := filepath.Join(root, "model.json")
	if err := save(path, meta); err != nil {
		return "", err
	}
	if _, err := decision.LoadPath(path); err != nil {
		return "", fmt.Errorf("SDK rejected exported model: %w", err)
	}
	return path, nil
}

func pack(codes []int8) []byte {
	raw := make([]byte, (len(codes)+4)/5)
	for i := range raw {
		power := byte(1)
		for j := range 5 {
			value := int8(0)
			if at := i*5 + j; at < len(codes) {
				value = codes[at]
			}
			raw[i] += byte(value+1) * power
			power *= 3
		}
	}
	return raw
}

func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
