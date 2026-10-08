package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
	"github.com/kimjooyoon/gooo-decision-runtime/jointdecision"
)

func exportModel(root string, n network, variant string, temperature float64) (string, error) {
	if variant != "fp32" && variant != "ptq_ternary" && variant != "qat_ternary" {
		return "", fmt.Errorf("explicit model encoding required")
	}
	if err := os.Mkdir(root, 0755); err != nil {
		return "", err
	}
	meta := jointdecision.Metadata{Schema: jointdecision.SharedThreeSchema, Feature: jointdecision.RecordGraphSharedFeatureVersion,
		Arithmetic: jointdecision.SeparateArithmeticVersion, Variant: variant, FeatureDim: jointdecision.ThreeFeatureDim,
		HiddenDim: hidden, MaxBytes: jointdecision.RecordGraphInputMaxBytes, Temperature: temperature, WeightsFile: "weights.bin"}
	for mask := range 8 {
		meta.Labels = append(meta.Labels, fmt.Sprintf("mask_%d", mask))
	}
	var weights []byte
	for i, values := range [][]float32{n[:w1Count], n[w1Count:w2Start], n[w2Start:]} {
		encoding, scale := "float32_le", float32(1)
		var block []byte
		if variant == "fp32" || i == 1 {
			block = make([]byte, len(values)*4)
			for j, v := range values {
				binary.LittleEndian.PutUint32(block[j*4:], math.Float32bits(v))
			}
		} else {
			encoding = "ternary_base3_5"
			var codes []int8
			codes, scale = quantize(values)
			block = pack(codes)
		}
		meta.Tensors = append(meta.Tensors, decision.TensorMetadata{Name: [3]string{"w1", "b1", "w2"}[i], Rows: [3]int{hidden, 1, 2}[i], Cols: [3]int{width, hidden, hidden}[i], Count: len(values), Encoding: encoding, Offset: int64(len(weights)), Bytes: int64(len(block)), Scale: float64(scale)})
		weights = append(weights, block...)
	}
	meta.WeightsSHA = digest(weights)
	if err := os.WriteFile(filepath.Join(root, "weights.bin"), weights, 0644); err != nil {
		return "", err
	}
	path := filepath.Join(root, "model.json")
	save(path, meta)
	return path, nil
}

func pack(codes []int8) []byte {
	raw := make([]byte, (len(codes)+4)/5)
	for byteIndex := range raw {
		power := byte(1)
		for digit := range 5 {
			value := int8(0)
			index := byteIndex*5 + digit
			if index < len(codes) {
				value = codes[index]
			}
			raw[byteIndex] += byte(value+1) * power
			power *= 3
		}
	}
	return raw
}
